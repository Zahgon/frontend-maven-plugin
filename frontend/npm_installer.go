package frontend

import (
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/eirslett/frontend-maven-plugin/internal/logging"
)

// DefaultNpmDownloadRoot is where npm tarballs are fetched from.
const DefaultNpmDownloadRoot = "https://registry.npmjs.org/npm/-/"

var npmInstallLock sync.Mutex

// NPMInstaller downloads an npm release and unpacks it into the Node.js
// installation's node_modules.
type NPMInstaller struct {
	nodeVersion     string
	npmVersion      string
	npmDownloadRoot string
	userName        string
	password        string
	httpHeaders     map[string]string

	logger           *logging.Logger
	config           InstallConfig
	archiveExtractor ArchiveExtractor
	fileDownloader   FileDownloader
}

// NewNPMInstaller builds an installer against one install configuration.
func NewNPMInstaller(config InstallConfig, archiveExtractor ArchiveExtractor, fileDownloader FileDownloader) *NPMInstaller {
	return &NPMInstaller{
		logger:           logging.GetLogger("NPMInstaller"),
		config:           config,
		archiveExtractor: archiveExtractor,
		fileDownloader:   fileDownloader,
	}
}

// SetNodeVersion records the Node.js version, which decides whether "provided"
// npm is available at all.
func (i *NPMInstaller) SetNodeVersion(nodeVersion string) *NPMInstaller {
	i.nodeVersion = nodeVersion
	return i
}

// SetNpmVersion selects the npm version, or "provided" to keep the one bundled
// with Node.js.
func (i *NPMInstaller) SetNpmVersion(npmVersion string) *NPMInstaller {
	i.npmVersion = npmVersion
	return i
}

// SetNpmDownloadRoot overrides where the npm tarball is fetched from.
func (i *NPMInstaller) SetNpmDownloadRoot(npmDownloadRoot string) *NPMInstaller {
	i.npmDownloadRoot = npmDownloadRoot
	return i
}

// SetUserName sets the download username.
func (i *NPMInstaller) SetUserName(userName string) *NPMInstaller {
	i.userName = userName
	return i
}

// SetPassword sets the download password.
func (i *NPMInstaller) SetPassword(password string) *NPMInstaller {
	i.password = password
	return i
}

// SetHTTPHeaders sets extra headers to send with the download.
func (i *NPMInstaller) SetHTTPHeaders(httpHeaders map[string]string) *NPMInstaller {
	i.httpHeaders = httpHeaders
	return i
}

func (i *NPMInstaller) npmProvided() (bool, error) {
	if i.npmVersion != "provided" {
		return false, nil
	}
	major, err := strconv.Atoi(strings.Split(strings.ReplaceAll(i.nodeVersion, "v", ""), ".")[0])
	if err != nil || major < 4 {
		return false, newInstallationError(
			"NPM version is '"+i.npmVersion+"' but Node didn't include NPM prior to v4.0.0", nil)
	}
	return true, nil
}

// Install downloads and installs npm unless the distribution already provides
// it or the requested version is in place, then copies its launcher scripts next
// to node.
func (i *NPMInstaller) Install() error {
	npmInstallLock.Lock()
	defer npmInstallLock.Unlock()

	if i.npmDownloadRoot == "" {
		i.npmDownloadRoot = DefaultNpmDownloadRoot
	}
	provided, err := i.npmProvided()
	if err != nil {
		return err
	}
	if !provided {
		installed, err := i.npmIsAlreadyInstalled()
		if err != nil {
			return err
		}
		if !installed {
			if err := i.installNpm(); err != nil {
				return err
			}
		}
	}
	return i.copyNpmScripts()
}

func (i *NPMInstaller) npmIsAlreadyInstalled() (bool, error) {
	npmPackageJSON := newFile(i.config.InstallDirectory() + Normalize("/node/node_modules/npm/package.json"))
	if !exists(npmPackageJSON) {
		return false, nil
	}
	foundNpmVersion, present, err := readPackageVersion(npmPackageJSON)
	if err != nil {
		return false, newInstallationError("Could not read package.json", err)
	}
	if !present {
		i.logger.Info("Could not read NPM version from package.json")
		return false, nil
	}
	if foundNpmVersion == i.npmVersion {
		i.logger.Info("NPM {} is already installed.", foundNpmVersion)
		return true, nil
	}
	i.logger.Info("NPM {} was installed, but we need version {}", foundNpmVersion, i.npmVersion)
	return false, nil
}

func (i *NPMInstaller) installNpm() error {
	i.logger.Info("Installing npm version {}", i.npmVersion)
	downloadURL := i.npmDownloadRoot + "npm-" + i.npmVersion + ".tgz"

	cacheDescriptor := NewCacheDescriptor("npm", i.npmVersion, "tar.gz")
	archive := i.config.CacheResolver().Resolve(cacheDescriptor)

	if err := i.downloadFileIfMissing(downloadURL, archive, i.userName, i.password, i.httpHeaders); err != nil {
		return newInstallationError("Could not download npm", err)
	}

	installDirectory := i.nodeInstallDirectory()
	nodeModulesDirectory := childFile(installDirectory, "node_modules")

	// The existing npm directory goes first, both to clear out stale files and
	// so the extracted "package" directory can be renamed into its place.
	oldNpmDirectory := childFile(installDirectory, "npm")
	npmDirectory := childFile(nodeModulesDirectory, "npm")
	if err := clearExistingInstall(oldNpmDirectory, npmDirectory); err != nil {
		i.logger.Warn("Failed to delete existing NPM installation.")
	}

	packageDirectory := childFile(nodeModulesDirectory, "package")
	if err := i.extractFile(archive, nodeModulesDirectory); err != nil {
		if isEOFCause(err) {
			// https://github.com/eirslett/frontend-maven-plugin/issues/794
			// The download was probably interrupted and the archive is
			// incomplete: delete it so the next build starts from scratch.
			i.logger.Error("The archive file {} is corrupted and will be deleted. Please try the build again.", archive)
			_ = os.Remove(archive)
			if exists(packageDirectory) {
				_ = deleteDirectory(packageDirectory)
			}
		}
		return newInstallationError("Could not extract the npm archive", err)
	}

	// Handles the difference between the old and the new download root
	// (nodejs.org/dist/npm and registry.npmjs.org); see
	// https://github.com/eirslett/frontend-maven-plugin/issues/65#issuecomment-52024254
	if exists(packageDirectory) && !exists(npmDirectory) {
		if !renameTo(packageDirectory, npmDirectory) {
			i.logger.Warn("Cannot rename NPM directory, making a copy.")
			if err := copyDirectory(packageDirectory, npmDirectory); err != nil {
				return newInstallationError("Could not copy npm", err)
			}
		}
	}

	i.logger.Info("Installed npm locally.")
	return nil
}

func (i *NPMInstaller) copyNpmScripts() error {
	installDirectory := i.nodeInstallDirectory()
	nodeModulesDirectory := childFile(installDirectory, "node_modules")
	npmDirectory := childFile(nodeModulesDirectory, "npm")
	return copyNpmScripts(npmDirectory, installDirectory)
}

func (i *NPMInstaller) nodeInstallDirectory() string {
	installDirectory := childFile(i.config.InstallDirectory(), NodeInstallPath)
	if !exists(installDirectory) {
		i.logger.Debug("Creating install directory {}", installDirectory)
		_ = os.MkdirAll(installDirectory, 0o777)
	}
	return installDirectory
}

func (i *NPMInstaller) extractFile(archive, destinationDirectory string) error {
	i.logger.Info("Unpacking {} into {}", archive, destinationDirectory)
	return i.archiveExtractor.Extract(archive, destinationDirectory)
}

func (i *NPMInstaller) downloadFileIfMissing(downloadURL, destination, userName, password string, httpHeaders map[string]string) error {
	if exists(destination) {
		return nil
	}
	i.logger.Info("Downloading {} to {}", downloadURL, destination)
	return i.fileDownloader.Download(downloadURL, destination, userName, password, httpHeaders)
}

// clearExistingInstall removes a package-manager directory left by an older
// layout as well as the current one, reporting the first failure.
func clearExistingInstall(legacyDirectory, currentDirectory string) error {
	if isDirectory(legacyDirectory) {
		if err := deleteDirectory(legacyDirectory); err != nil {
			return err
		}
	}
	return deleteDirectory(currentDirectory)
}
