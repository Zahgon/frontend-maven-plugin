package frontend

import (
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/eirslett/frontend-maven-plugin/internal/logging"
)

// YarnInstallPath is where a Yarn installation is placed, relative to the
// install directory.
const YarnInstallPath = "/node/yarn"

// DefaultYarnDownloadRoot is where Yarn releases are fetched from.
const DefaultYarnDownloadRoot = "https://github.com/yarnpkg/yarn/releases/download/"

// yarnRootDirectory is the directory a Yarn archive is expected to unpack into.
const yarnRootDirectory = "dist"

var yarnInstallLock sync.Mutex

// YarnInstaller downloads a Yarn release and unpacks it under the Node.js
// installation.
type YarnInstaller struct {
	yarnVersion      string
	yarnDownloadRoot string
	userName         string
	password         string
	httpHeaders      map[string]string
	isYarnBerry      bool

	logger           *logging.Logger
	config           InstallConfig
	archiveExtractor ArchiveExtractor
	fileDownloader   FileDownloader
}

// NewYarnInstaller builds an installer against one install configuration.
func NewYarnInstaller(config InstallConfig, archiveExtractor ArchiveExtractor, fileDownloader FileDownloader) *YarnInstaller {
	return &YarnInstaller{
		logger:           logging.GetLogger("YarnInstaller"),
		config:           config,
		archiveExtractor: archiveExtractor,
		fileDownloader:   fileDownloader,
	}
}

// SetYarnVersion selects the Yarn version to install; it has to start with "v".
func (i *YarnInstaller) SetYarnVersion(yarnVersion string) *YarnInstaller {
	i.yarnVersion = yarnVersion
	return i
}

// SetIsYarnBerry records whether the project is on Yarn 2 or later, which
// changes what counts as already installed.
func (i *YarnInstaller) SetIsYarnBerry(isYarnBerry bool) *YarnInstaller {
	i.isYarnBerry = isYarnBerry
	return i
}

// SetYarnDownloadRoot overrides where the Yarn archive is fetched from.
func (i *YarnInstaller) SetYarnDownloadRoot(yarnDownloadRoot string) *YarnInstaller {
	i.yarnDownloadRoot = yarnDownloadRoot
	return i
}

// SetUserName sets the download username.
func (i *YarnInstaller) SetUserName(userName string) *YarnInstaller {
	i.userName = userName
	return i
}

// SetPassword sets the download password.
func (i *YarnInstaller) SetPassword(password string) *YarnInstaller {
	i.password = password
	return i
}

// SetHTTPHeaders sets extra headers to send with the download.
func (i *YarnInstaller) SetHTTPHeaders(httpHeaders map[string]string) *YarnInstaller {
	i.httpHeaders = httpHeaders
	return i
}

// Install downloads and installs Yarn unless the requested version is already
// in place.
func (i *YarnInstaller) Install() error {
	yarnInstallLock.Lock()
	defer yarnInstallLock.Unlock()

	if i.yarnDownloadRoot == "" {
		i.yarnDownloadRoot = DefaultYarnDownloadRoot
	}
	if i.yarnIsAlreadyInstalled() {
		return nil
	}
	if !strings.HasPrefix(i.yarnVersion, "v") {
		return newInstallationError("Yarn version has to start with prefix 'v'.", nil)
	}
	return i.installYarn()
}

func (i *YarnInstaller) yarnIsAlreadyInstalled() bool {
	executorConfig := NewInstallYarnExecutorConfig(i.config, i.isYarnBerry)
	nodeFile := executorConfig.YarnPath()
	if !exists(nodeFile) {
		return false
	}
	result, err := NewYarnExecutor(executorConfig, []string{"--version"}, nil).ExecuteAndGetResult(i.logger)
	if err != nil {
		return false
	}
	version := strings.TrimSpace(result)
	if version == strings.TrimPrefix(i.yarnVersion, "v") {
		i.logger.Info("Yarn {} is already installed.", version)
		return true
	}
	if i.isYarnBerry {
		if major, convErr := strconv.Atoi(strings.Split(version, ".")[0]); convErr == nil && major > 1 {
			i.logger.Info("Yarn Berry {} is installed.", version)
			return true
		}
	}
	i.logger.Info("Yarn {} was installed, but we need version {}", version, i.yarnVersion)
	return false
}

func (i *YarnInstaller) installYarn() error {
	i.logger.Info("Installing Yarn version {}", i.yarnVersion)
	extension := "tar.gz"
	downloadURL := i.yarnDownloadRoot + i.yarnVersion + "/yarn-" + i.yarnVersion + "." + extension

	cacheDescriptor := NewCacheDescriptor("yarn", i.yarnVersion, extension)
	archive := i.config.CacheResolver().Resolve(cacheDescriptor)

	if err := i.downloadFileIfMissing(downloadURL, archive, i.userName, i.password, i.httpHeaders); err != nil {
		return newInstallationError("Could not download Yarn", err)
	}

	installDirectory := i.installDirectory()

	// The existing yarn directory goes first, both to clear out stale files and
	// so the extracted directory can be renamed into its place.
	if isDirectory(installDirectory) {
		if err := deleteDirectory(installDirectory); err != nil {
			i.logger.Warn("Failed to delete existing Yarn installation.")
		}
	}

	if err := i.extractFile(archive, installDirectory); err != nil {
		if isEOFCause(err) {
			// https://github.com/eirslett/frontend-maven-plugin/issues/794
			// The download was probably interrupted and the archive is
			// incomplete: delete it so the next build starts from scratch.
			i.logger.Error("The archive file {} is corrupted and will be deleted. Please try the build again.", archive)
			_ = os.Remove(archive)
			if exists(installDirectory) {
				_ = deleteDirectory(installDirectory)
			}
		}
		return newInstallationError("Could not extract the Yarn archive", err)
	}

	if err := i.ensureCorrectYarnRootDirectory(installDirectory, i.yarnVersion); err != nil {
		return newInstallationError("Could not extract the Yarn archive", err)
	}

	i.logger.Info("Installed Yarn locally.")
	return nil
}

func (i *YarnInstaller) installDirectory() string {
	installDirectory := childFile(i.config.InstallDirectory(), YarnInstallPath)
	if !exists(installDirectory) {
		i.logger.Debug("Creating install directory {}", installDirectory)
		_ = os.MkdirAll(installDirectory, 0o777)
	}
	return installDirectory
}

func (i *YarnInstaller) extractFile(archive, destinationDirectory string) error {
	i.logger.Info("Unpacking {} into {}", archive, destinationDirectory)
	return i.archiveExtractor.Extract(archive, destinationDirectory)
}

// ensureCorrectYarnRootDirectory renames a Yarn 1.x "yarn-<version>" root to
// "dist", which is what every executor path is built against.
func (i *YarnInstaller) ensureCorrectYarnRootDirectory(installDirectory, yarnVersion string) error {
	yarnRoot := childFile(installDirectory, yarnRootDirectory)
	if exists(yarnRoot) {
		return nil
	}
	i.logger.Debug("Yarn root directory not found, checking for yarn-{}", yarnVersion)
	yarnOneXDirectory := childFile(installDirectory, "yarn-"+yarnVersion)
	if !isDirectory(yarnOneXDirectory) {
		return newIOError("Could not find yarn distribution directory during extract")
	}
	if !renameTo(yarnOneXDirectory, yarnRoot) {
		return newIOError("Could not rename versioned yarn root directory to %s", yarnRootDirectory)
	}
	return nil
}

func (i *YarnInstaller) downloadFileIfMissing(downloadURL, destination, userName, password string, httpHeaders map[string]string) error {
	if exists(destination) {
		return nil
	}
	i.logger.Info("Downloading {} to {}", downloadURL, destination)
	return i.fileDownloader.Download(downloadURL, destination, userName, password, httpHeaders)
}
