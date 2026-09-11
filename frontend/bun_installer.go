package frontend

import (
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/eirslett/frontend-maven-plugin/internal/logging"
)

// BunInstallPath is where a Bun installation is placed, relative to the install
// directory.
const BunInstallPath = "/bun"

// DefaultBunDownloadRoot is where Bun releases are fetched from.
const DefaultBunDownloadRoot = "https://github.com/oven-sh/bun/releases/download/"

var bunInstallLock sync.Mutex

// BunInstaller downloads a Bun release and puts its binary under the install
// directory.
type BunInstaller struct {
	bunVersion      string
	bunDownloadRoot string
	userName        string
	password        string
	httpHeaders     map[string]string

	logger           *logging.Logger
	config           InstallConfig
	archiveExtractor ArchiveExtractor
	fileDownloader   FileDownloader
}

// NewBunInstaller builds an installer against one install configuration.
func NewBunInstaller(config InstallConfig, archiveExtractor ArchiveExtractor, fileDownloader FileDownloader) *BunInstaller {
	return &BunInstaller{
		logger:           logging.GetLogger("BunInstaller"),
		config:           config,
		archiveExtractor: archiveExtractor,
		fileDownloader:   fileDownloader,
	}
}

// SetBunVersion selects the Bun version to install.
func (i *BunInstaller) SetBunVersion(bunVersion string) *BunInstaller {
	i.bunVersion = bunVersion
	return i
}

// SetBunDownloadRoot overrides where the Bun archive is fetched from.
func (i *BunInstaller) SetBunDownloadRoot(bunDownloadRoot string) *BunInstaller {
	i.bunDownloadRoot = bunDownloadRoot
	return i
}

// SetUserName sets the download username.
func (i *BunInstaller) SetUserName(userName string) *BunInstaller {
	i.userName = userName
	return i
}

// SetPassword sets the download password.
func (i *BunInstaller) SetPassword(password string) *BunInstaller {
	i.password = password
	return i
}

// SetHTTPHeaders sets extra headers to send with the download.
func (i *BunInstaller) SetHTTPHeaders(httpHeaders map[string]string) *BunInstaller {
	i.httpHeaders = httpHeaders
	return i
}

// Install downloads and installs Bun unless the requested version is already in
// place.
func (i *BunInstaller) Install() error {
	bunInstallLock.Lock()
	defer bunInstallLock.Unlock()

	if i.bunIsAlreadyInstalled() {
		return nil
	}
	if !strings.HasPrefix(i.bunVersion, "v") {
		i.logger.Warn("Bun version does not start with naming convention 'v'.")
	}
	return i.installBunDefault()
}

func (i *BunInstaller) bunIsAlreadyInstalled() bool {
	executorConfig := NewInstallBunExecutorConfig(i.config)
	bunFile := executorConfig.BunPath()
	if !exists(bunFile) {
		return false
	}
	version, err := NewBunExecutor(executorConfig, []string{"--version"}, nil).ExecuteAndGetResult(i.logger)
	if err != nil {
		i.logger.Warn("Unable to determine current bun version: {}", err.Error())
		return false
	}
	if version == strings.TrimPrefix(i.bunVersion, "v") {
		i.logger.Info("Bun {} is already installed.", version)
		return true
	}
	i.logger.Info("Bun {} was installed, but we need version {}", version, i.bunVersion)
	return false
}

func (i *BunInstaller) installBunDefault() error {
	i.logger.Info("Installing Bun version {}", i.bunVersion)

	downloadURL := i.createDownloadURL()

	cacheDescriptor := NewCacheDescriptor("bun", i.bunVersion, "zip")
	archive := i.config.CacheResolver().Resolve(cacheDescriptor)

	if err := i.downloadFileIfMissing(downloadURL, archive, i.userName, i.password, i.httpHeaders); err != nil {
		return newInstallationError("Could not download bun", err)
	}

	installDirectory := i.installDirectory()

	// The existing bun directory goes first, both to clear out stale files and
	// so the extracted directory can be renamed into its place.
	bunExtractDirectory := childFile(installDirectory, createBunTargetArchitecturePath())
	if isDirectory(bunExtractDirectory) {
		if err := deleteDirectory(bunExtractDirectory); err != nil {
			i.logger.Warn("Failed to delete existing Bun installation.")
		}
	}

	if err := i.extractFile(archive, installDirectory); err != nil {
		if isEOFCause(err) {
			i.logger.Error("The archive file {} is corrupted and will be deleted. Please try the build again.", archive)
			_ = os.Remove(archive)
		}
		return newInstallationError("Could not extract the bun archive", err)
	}

	// Search for the bun binary.
	bunExecutable := "bun"
	if i.config.Platform().IsWindows() {
		bunExecutable = "bun.exe"
	}
	bunBinary := childFile(installDirectory, separator+createBunTargetArchitecturePath()+separator+bunExecutable)
	if !exists(bunBinary) {
		return newInstallationError("Could not install bun",
			newIOError("Could not find the downloaded bun binary in %s", bunBinary))
	}

	destinationDirectory := childFile(i.installDirectory(), BunInstallPath)
	if !exists(destinationDirectory) {
		i.logger.Info("Creating destination directory {}", destinationDirectory)
		_ = os.MkdirAll(destinationDirectory, 0o777)
	}

	destination := childFile(destinationDirectory, bunExecutable)
	i.logger.Info("Copying bun binary from {} to {}", bunBinary, destination)
	if exists(destination) && os.Remove(destination) != nil {
		return newInstallationError("Could not install Bun: Was not allowed to delete "+destination, nil)
	}
	if err := moveFile(bunBinary, destination); err != nil {
		return newInstallationError("Could not install Bun: Was not allowed to rename "+bunBinary+" to "+destination, nil)
	}
	if !setExecutable(destination, false) {
		return newInstallationError("Could not install Bun: Was not allowed to make "+destination+" executable.", nil)
	}
	if err := deleteDirectory(bunExtractDirectory); err != nil {
		return newInstallationError("Could not install bun", err)
	}

	i.logger.Info("Installed bun locally.")
	return nil
}

func (i *BunInstaller) createDownloadURL() string {
	downloadRoot := i.bunDownloadRoot
	if downloadRoot == "" {
		downloadRoot = DefaultBunDownloadRoot
	}
	downloadURL := fmt.Sprintf("%sbun-%s", downloadRoot, i.bunVersion)
	extension := "zip"
	fileending := fmt.Sprintf("%s.%s", createBunTargetArchitecturePath(), extension)

	return downloadURL + fileending
}

// createBunTargetArchitecturePath names the directory inside a Bun archive, and
// the tail of its download URL. It keeps the leading separator of
// BunInstallPath, which is what makes the URL read ".../bun-v1.0.0/bun-linux-x64.zip".
func createBunTargetArchitecturePath() string {
	currentOS := GuessOS()
	architecture := GuessArchitecture()
	destOs := ""
	switch currentOS {
	case OSLinux:
		destOs = "linux"
	case OSMac:
		destOs = "darwin"
	case OSWindows:
		destOs = "windows"
	}
	destArc := ""
	switch architecture {
	case ArchX64:
		destArc = "x64"
	case ArchARM64:
		destArc = "aarch64"
	}
	return fmt.Sprintf("%s-%s-%s", BunInstallPath, destOs, destArc)
}

func (i *BunInstaller) installDirectory() string {
	installDirectory := newFile(i.config.InstallDirectory() + "/")
	if !exists(installDirectory) {
		i.logger.Info("Creating install directory {}", installDirectory)
		_ = os.MkdirAll(installDirectory, 0o777)
	}
	return installDirectory
}

func (i *BunInstaller) extractFile(archive, destinationDirectory string) error {
	i.logger.Info("Unpacking {} into {}", archive, destinationDirectory)
	return i.archiveExtractor.Extract(archive, destinationDirectory)
}

func (i *BunInstaller) downloadFileIfMissing(downloadURL, destination, userName, password string, httpHeaders map[string]string) error {
	if exists(destination) {
		return nil
	}
	i.logger.Info("Downloading {} to {}", downloadURL, destination)
	return i.fileDownloader.Download(downloadURL, destination, userName, password, httpHeaders)
}
