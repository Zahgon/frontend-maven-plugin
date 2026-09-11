package frontend

import (
	"errors"
	"io"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/eirslett/frontend-maven-plugin/internal/logging"
)

// NodeInstallPath is where a Node.js installation is placed, relative to the
// install directory. It is concatenated onto that directory, so the leading
// separator matters.
const NodeInstallPath = "/node"

// nodeInstallLock serialises installs so two goals in one build cannot unpack
// into the same directory at once.
var nodeInstallLock sync.Mutex

// NodeInstaller downloads a Node.js distribution and puts the binary — and, when
// asked for the bundled npm, its node_modules — under the install directory.
type NodeInstaller struct {
	npmVersion       string
	nodeVersion      string
	nodeDownloadRoot string
	userName         string
	password         string
	httpHeaders      map[string]string

	logger           *logging.Logger
	config           InstallConfig
	archiveExtractor ArchiveExtractor
	fileDownloader   FileDownloader
}

// NewNodeInstaller builds an installer against one install configuration.
func NewNodeInstaller(config InstallConfig, archiveExtractor ArchiveExtractor, fileDownloader FileDownloader) *NodeInstaller {
	return &NodeInstaller{
		logger:           logging.GetLogger("NodeInstaller"),
		config:           config,
		archiveExtractor: archiveExtractor,
		fileDownloader:   fileDownloader,
	}
}

// SetNodeVersion selects the Node.js version to install.
func (i *NodeInstaller) SetNodeVersion(nodeVersion string) *NodeInstaller {
	i.nodeVersion = nodeVersion
	return i
}

// SetNodeDownloadRoot overrides where the distribution is fetched from.
func (i *NodeInstaller) SetNodeDownloadRoot(nodeDownloadRoot string) *NodeInstaller {
	i.nodeDownloadRoot = nodeDownloadRoot
	return i
}

// SetNpmVersion records the npm version; only the value "provided" changes what
// this installer does, by making it keep the distribution's own npm.
func (i *NodeInstaller) SetNpmVersion(npmVersion string) *NodeInstaller {
	i.npmVersion = npmVersion
	return i
}

// SetUserName sets the download username.
func (i *NodeInstaller) SetUserName(userName string) *NodeInstaller {
	i.userName = userName
	return i
}

// SetPassword sets the download password.
func (i *NodeInstaller) SetPassword(password string) *NodeInstaller {
	i.password = password
	return i
}

// SetHTTPHeaders sets extra headers to send with the download.
func (i *NodeInstaller) SetHTTPHeaders(httpHeaders map[string]string) *NodeInstaller {
	i.httpHeaders = httpHeaders
	return i
}

func (i *NodeInstaller) npmProvided() (bool, error) {
	if i.npmVersion == "" {
		return false, nil
	}
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

// Install downloads and installs Node.js, unless the requested version is
// already in place.
func (i *NodeInstaller) Install() error {
	nodeInstallLock.Lock()
	defer nodeInstallLock.Unlock()

	if i.nodeDownloadRoot == "" {
		i.nodeDownloadRoot = i.config.Platform().NodeDownloadRoot()
	}
	if i.nodeIsAlreadyInstalled() {
		return nil
	}
	i.logger.Info("Installing node version {}", i.nodeVersion)
	if !strings.HasPrefix(i.nodeVersion, "v") {
		i.logger.Warn("Node version does not start with naming convention 'v'.")
	}
	if i.config.Platform().IsWindows() {
		provided, err := i.npmProvided()
		if err != nil {
			return err
		}
		if provided {
			return i.installNodeWithNpmForWindows()
		}
		return i.installNodeForWindows()
	}
	return i.installNodeDefault()
}

func (i *NodeInstaller) nodeIsAlreadyInstalled() bool {
	executorConfig := NewInstallNodeExecutorConfig(i.config)
	nodeFile := executorConfig.NodePath()
	if !exists(nodeFile) {
		return false
	}
	version, err := NewNodeExecutor(executorConfig, []string{"--version"}, nil).ExecuteAndGetResult(i.logger)
	if err != nil {
		i.logger.Warn("Unable to determine current node version: {}", err.Error())
		return false
	}
	if version == i.nodeVersion {
		i.logger.Info("Node {} is already installed.", version)
		return true
	}
	i.logger.Info("Node {} was installed, but we need version {}", version, i.nodeVersion)
	return false
}

func (i *NodeInstaller) installNodeDefault() error {
	platform := i.config.Platform()
	longNodeFilename := platform.LongNodeFilename(i.nodeVersion, false)
	downloadURL := i.nodeDownloadRoot + platform.NodeDownloadFilename(i.nodeVersion, false)
	classifier := platform.NodeClassifier(i.nodeVersion)

	tmpDirectory := i.tempDirectory()

	cacheDescriptor := NewClassifiedCacheDescriptor("node", i.nodeVersion, classifier, platform.ArchiveExtension())
	archive := i.config.CacheResolver().Resolve(cacheDescriptor)

	if err := i.downloadFileIfMissing(downloadURL, archive, i.userName, i.password, i.httpHeaders); err != nil {
		return newInstallationError("Could not download Node.js", err)
	}

	if err := i.extractFile(archive, tmpDirectory); err != nil {
		if isEOFCause(err) {
			// https://github.com/eirslett/frontend-maven-plugin/issues/794
			// The download was probably interrupted and the archive is
			// incomplete: delete it so the next build starts from scratch.
			i.logger.Error("The archive file {} is corrupted and will be deleted. Please try the build again.", archive)
			_ = os.Remove(archive)
			_ = deleteDirectory(tmpDirectory)
		}
		return newInstallationError("Could not extract the Node archive", err)
	}

	// Search for the node binary.
	nodeBinary := childFile(tmpDirectory, longNodeFilename+separator+"bin"+separator+"node")
	if !exists(nodeBinary) {
		return newInstallationError("Could not install Node",
			newIOError("Could not find the downloaded Node.js binary in %s", nodeBinary))
	}

	destinationDirectory := i.installDirectory()
	destination := childFile(destinationDirectory, "node")
	i.logger.Info("Copying node binary from {} to {}", nodeBinary, destination)
	if exists(destination) && os.Remove(destination) != nil {
		return newInstallationError("Could not install Node: Was not allowed to delete "+destination, nil)
	}
	if err := moveFile(nodeBinary, destination); err != nil {
		return newInstallationError("Could not install Node: Was not allowed to rename "+nodeBinary+" to "+destination, nil)
	}
	if !setExecutable(destination, false) {
		return newInstallationError("Could not install Node: Was not allowed to make "+destination+" executable.", nil)
	}

	provided, err := i.npmProvided()
	if err != nil {
		return err
	}
	if provided {
		tmpNodeModulesDir := childFile(tmpDirectory, longNodeFilename+separator+"lib"+separator+"node_modules")
		nodeModulesDirectory := childFile(destinationDirectory, "node_modules")
		npmDirectory := childFile(nodeModulesDirectory, "npm")
		if err := copyDirectory(tmpNodeModulesDir, nodeModulesDirectory); err != nil {
			return newInstallationError("Could not install Node", err)
		}
		i.logger.Info("Extracting NPM")
		if err := copyNpmScripts(npmDirectory, destinationDirectory); err != nil {
			return err
		}
	}

	if err := i.deleteTempDirectory(tmpDirectory); err != nil {
		return newInstallationError("Could not install Node", err)
	}

	i.logger.Info("Installed node locally.")
	return nil
}

func (i *NodeInstaller) installNodeWithNpmForWindows() error {
	platform := i.config.Platform()
	longNodeFilename := platform.LongNodeFilename(i.nodeVersion, true)
	downloadURL := i.nodeDownloadRoot + platform.NodeDownloadFilename(i.nodeVersion, true)
	classifier := platform.NodeClassifier(i.nodeVersion)

	tmpDirectory := i.tempDirectory()

	cacheDescriptor := NewClassifiedCacheDescriptor("node", i.nodeVersion, classifier, platform.ArchiveExtension())
	archive := i.config.CacheResolver().Resolve(cacheDescriptor)

	if err := i.downloadFileIfMissing(downloadURL, archive, i.userName, i.password, i.httpHeaders); err != nil {
		return newInstallationError("Could not download Node.js", err)
	}
	if err := i.extractFile(archive, tmpDirectory); err != nil {
		return newInstallationError("Could not extract the Node archive", err)
	}

	// Search for the node binary.
	nodeBinary := childFile(tmpDirectory, longNodeFilename+separator+WindowsNodeExecutable)
	if !exists(nodeBinary) {
		return newInstallationError("Could not install Node",
			newIOError("Could not find the downloaded Node.js binary in %s", nodeBinary))
	}

	destinationDirectory := i.installDirectory()
	destination := childFile(destinationDirectory, WindowsNodeExecutable)
	i.logger.Info("Copying node binary from {} to {}", nodeBinary, destination)
	if err := moveFile(nodeBinary, destination); err != nil {
		return newInstallationError("Could not install Node: Was not allowed to rename "+nodeBinary+" to "+destination, nil)
	}

	if i.npmVersion == "provided" {
		tmpNodeModulesDir := childFile(tmpDirectory, longNodeFilename+separator+"node_modules")
		nodeModulesDirectory := childFile(destinationDirectory, "node_modules")
		if err := copyDirectory(tmpNodeModulesDir, nodeModulesDirectory); err != nil {
			return newInstallationError("Could not install Node", err)
		}
	}
	if err := i.deleteTempDirectory(tmpDirectory); err != nil {
		return newInstallationError("Could not install Node", err)
	}

	i.logger.Info("Installed node locally.")
	return nil
}

func (i *NodeInstaller) installNodeForWindows() error {
	platform := i.config.Platform()
	downloadURL := i.nodeDownloadRoot + platform.NodeDownloadFilename(i.nodeVersion, false)

	destinationDirectory := i.installDirectory()
	destination := childFile(destinationDirectory, WindowsNodeExecutable)
	classifier := platform.NodeClassifier(i.nodeVersion)

	cacheDescriptor := NewClassifiedCacheDescriptor("node", i.nodeVersion, classifier, "exe")
	binary := i.config.CacheResolver().Resolve(cacheDescriptor)

	if err := i.downloadFileIfMissing(downloadURL, binary, i.userName, i.password, i.httpHeaders); err != nil {
		return newInstallationError("Could not download Node.js from: "+downloadURL, err)
	}

	i.logger.Info("Copying node binary from {} to {}", binary, destination)
	if err := copyFile(binary, destination); err != nil {
		return newInstallationError("Could not install Node.js", err)
	}

	i.logger.Info("Installed node locally.")
	return nil
}

func (i *NodeInstaller) tempDirectory() string {
	tmpDirectory := childFile(i.installDirectory(), "tmp")
	if !exists(tmpDirectory) {
		i.logger.Debug("Creating temporary directory {}", tmpDirectory)
		_ = os.MkdirAll(tmpDirectory, 0o777)
	}
	return tmpDirectory
}

func (i *NodeInstaller) installDirectory() string {
	installDirectory := childFile(i.config.InstallDirectory(), NodeInstallPath)
	if !exists(installDirectory) {
		i.logger.Debug("Creating install directory {}", installDirectory)
		_ = os.MkdirAll(installDirectory, 0o777)
	}
	return installDirectory
}

func (i *NodeInstaller) deleteTempDirectory(tmpDirectory string) error {
	if tmpDirectory != "" && exists(tmpDirectory) {
		i.logger.Debug("Deleting temporary directory {}", tmpDirectory)
		return deleteDirectory(tmpDirectory)
	}
	return nil
}

func (i *NodeInstaller) extractFile(archive, destinationDirectory string) error {
	i.logger.Info("Unpacking {} into {}", archive, destinationDirectory)
	return i.archiveExtractor.Extract(archive, destinationDirectory)
}

func (i *NodeInstaller) downloadFileIfMissing(downloadURL, destination, userName, password string, httpHeaders map[string]string) error {
	if exists(destination) {
		return nil
	}
	i.logger.Info("Downloading {} to {}", downloadURL, destination)
	return i.fileDownloader.Download(downloadURL, destination, userName, password, httpHeaders)
}

// copyNpmScripts puts a copy of the npm and npx launchers next to the node
// executable, so a shell on the extended PATH finds them. An existing copy is
// left alone.
func copyNpmScripts(npmDirectory, destinationDirectory string) error {
	for _, script := range []string{"npm", "npm.cmd", "npx", "npx.cmd"} {
		scriptFile := childFile(npmDirectory, "bin"+separator+script)
		if !exists(scriptFile) {
			continue
		}
		copyPath := childFile(destinationDirectory, script)
		if exists(copyPath) {
			continue
		}
		if err := copyFile(scriptFile, copyPath); err != nil {
			return newInstallationError("Could not copy npm", err)
		}
		setExecutable(copyPath, true)
	}
	return nil
}

// isEOFCause reports whether a failure was a truncated archive, which is the one
// extraction problem the installers respond to by deleting the download.
func isEOFCause(err error) bool {
	return errors.Is(err, io.ErrUnexpectedEOF) || errors.Is(err, io.EOF)
}
