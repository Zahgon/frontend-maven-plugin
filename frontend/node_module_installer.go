package frontend

import (
	"fmt"
	"os"
	"strings"

	"github.com/eirslett/frontend-maven-plugin/internal/logging"
)

// pnpm and corepack install the same way — fetch a tarball from an npm
// registry, unpack it into the Node.js installation's node_modules, and put a
// launcher next to node — and the original spells that procedure out twice. The
// procedure lives here once, parameterised by the handful of things that really
// do differ: the names in the log, the download root, which entry point the
// launcher points at, and corepack's extra "provided" case.

// nodeModuleTool is everything the shared installer needs to know about one of
// the two tools.
type nodeModuleTool struct {
	// toolName is the lower-case name used in most messages ("pnpm").
	toolName string
	// displayName is the name used in the version-comparison messages, which
	// the original capitalises differently ("PNPM").
	displayName string
	// defaultDownloadRoot is where the tarball comes from.
	defaultDownloadRoot string
	// cliEnvVar is the variable the Windows proxy script assigns the entry
	// point to ("PNPM_CLI_JS").
	cliEnvVar string
	// executablePath picks the entry point the launcher should invoke.
	executablePath func(NodeExecutorConfig) string
	// versionIsProvided reports whether this version means "whatever the Node
	// distribution bundled", which only corepack accepts.
	versionIsProvided func(version string) bool
}

// nodeModuleInstaller installs one npm-registry-hosted tool into the Node.js
// installation and links a launcher for it.
type nodeModuleInstaller struct {
	tool nodeModuleTool

	version      string
	downloadRoot string
	userName     string
	password     string
	httpHeaders  map[string]string

	logger           *logging.Logger
	config           InstallConfig
	archiveExtractor ArchiveExtractor
	fileDownloader   FileDownloader
}

func (i *nodeModuleInstaller) install() error {
	if i.downloadRoot == "" {
		i.downloadRoot = i.tool.defaultDownloadRoot
	}
	installed, err := i.isAlreadyInstalled()
	if err != nil {
		return err
	}
	if !installed {
		if err := i.installModule(); err != nil {
			return err
		}
	}

	if i.config.Platform().IsWindows() {
		return i.linkExecutableWindows()
	}
	return i.linkExecutable()
}

func (i *nodeModuleInstaller) isAlreadyInstalled() (bool, error) {
	packageJSON := newFile(i.config.InstallDirectory() +
		Normalize("/node/node_modules/"+i.tool.toolName+"/package.json"))
	if !exists(packageJSON) {
		return false, nil
	}
	if i.tool.versionIsProvided != nil && i.tool.versionIsProvided(i.version) {
		// Which version it should be is unknowable, so the packaged one is
		// assumed to have been set up correctly.
		return true, nil
	}
	foundVersion, present, err := readPackageVersion(packageJSON)
	if err != nil {
		return false, newInstallationError("Could not read package.json", err)
	}
	if !present {
		i.logger.Info("Could not read " + i.tool.displayName + " version from package.json")
		return false, nil
	}
	if foundVersion == strings.TrimPrefix(i.version, "v") {
		i.logger.Info(i.tool.displayName+" {} is already installed.", foundVersion)
		return true, nil
	}
	i.logger.Info(i.tool.displayName+" {} was installed, but we need version {}", foundVersion, i.version)
	return false, nil
}

func (i *nodeModuleInstaller) installModule() error {
	i.logger.Info("Installing "+i.tool.toolName+" version {}", i.version)
	versionClean := trimVersionPrefix(i.version)
	downloadURL := i.downloadRoot + i.tool.toolName + "-" + versionClean + ".tgz"

	cacheDescriptor := NewCacheDescriptor(i.tool.toolName, versionClean, "tar.gz")
	archive := i.config.CacheResolver().Resolve(cacheDescriptor)

	if err := i.downloadFileIfMissing(downloadURL, archive); err != nil {
		return newInstallationError("Could not download "+i.tool.toolName, err)
	}

	installDirectory := i.nodeInstallDirectory()
	nodeModulesDirectory := childFile(installDirectory, "node_modules")

	// The existing directory goes first, both to clear out stale files and so
	// the extracted "package" directory can be renamed into its place.
	legacyDirectory := childFile(installDirectory, i.tool.toolName)
	moduleDirectory := childFile(nodeModulesDirectory, i.tool.toolName)
	if err := clearExistingInstall(legacyDirectory, moduleDirectory); err != nil {
		i.logger.Warn("Failed to delete existing " + i.tool.displayName + " installation.")
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
		return newInstallationError("Could not extract the "+i.tool.toolName+" archive", err)
	}

	// Handles the difference between the old and the new download root
	// (nodejs.org/dist/npm and registry.npmjs.org); see
	// https://github.com/eirslett/frontend-maven-plugin/issues/65#issuecomment-52024254
	if exists(packageDirectory) && !exists(moduleDirectory) {
		if !renameTo(packageDirectory, moduleDirectory) {
			i.logger.Warn("Cannot rename " + i.tool.displayName + " directory, making a copy.")
			if err := copyDirectory(packageDirectory, moduleDirectory); err != nil {
				return newInstallationError("Could not copy "+i.tool.toolName, err)
			}
		}
	}

	i.logger.Info("Installed " + i.tool.toolName + " locally.")
	return nil
}

func (i *nodeModuleInstaller) linkExecutable() error {
	nodeInstallDirectory := i.nodeInstallDirectory()
	executable := childFile(nodeInstallDirectory, i.tool.toolName)

	if exists(executable) {
		i.logger.Info("Existing " + i.tool.toolName + " executable found, skipping linking.")
		return nil
	}

	executorConfig := NewInstallNodeExecutorConfig(i.config)
	jsExecutable := i.tool.executablePath(executorConfig)

	if !exists(jsExecutable) {
		return newInstallationError(
			"Could not link to "+i.tool.toolName+" executable, no "+i.tool.toolName+" installation found.", nil)
	}

	i.logger.Info("No "+i.tool.toolName+" executable found, creating symbolic link to {}.", jsExecutable)

	if err := os.Symlink(jsExecutable, executable); err != nil {
		return newInstallationError("Could not create symbolic link for "+i.tool.toolName+" executable.", err)
	}
	return nil
}

func (i *nodeModuleInstaller) linkExecutableWindows() error {
	nodeInstallDirectory := i.nodeInstallDirectory()
	executable := childFile(nodeInstallDirectory, i.tool.toolName+".cmd")

	if exists(executable) {
		i.logger.Info("Existing " + i.tool.toolName + " executable found, skipping linking.")
		return nil
	}

	executorConfig := NewInstallNodeExecutorConfig(i.config)
	jsExecutable := i.tool.executablePath(executorConfig)

	if !exists(jsExecutable) {
		return newInstallationError(
			"Could not link to "+i.tool.toolName+" executable, no "+i.tool.toolName+" installation found.", nil)
	}

	i.logger.Info("No "+i.tool.toolName+" executable found, creating proxy script to {}.", jsExecutable)

	relativeNodePath := relativise(nodeInstallDirectory, executorConfig.NodePath())
	relativeToolPath := relativise(nodeInstallDirectory, jsExecutable)

	// A script that proxies whatever it is passed on to the real executable.
	scriptContents := ":: Created by frontend-maven-plugin, please don't edit manually.\r\n" +
		"@ECHO OFF\r\n" +
		"\r\n" +
		"SETLOCAL\r\n" +
		"\r\n" +
		fmt.Sprintf("SET \"NODE_EXE=%%~dp0\\%s\"\r\n", relativeNodePath) +
		fmt.Sprintf("SET \"%s=%%~dp0\\%s\"\r\n", i.tool.cliEnvVar, relativeToolPath) +
		"\r\n" +
		"\"%NODE_EXE%\" \"%" + i.tool.cliEnvVar + "%\" %*"

	if err := os.WriteFile(executable, []byte(scriptContents), 0o666); err != nil {
		return newInstallationError("Could not create proxy script for "+i.tool.toolName+" executable.", err)
	}
	return nil
}

func (i *nodeModuleInstaller) nodeInstallDirectory() string {
	installDirectory := childFile(i.config.InstallDirectory(), NodeInstallPath)
	if !exists(installDirectory) {
		i.logger.Debug("Creating install directory {}", installDirectory)
		_ = os.MkdirAll(installDirectory, 0o777)
	}
	return installDirectory
}

func (i *nodeModuleInstaller) extractFile(archive, destinationDirectory string) error {
	i.logger.Info("Unpacking {} into {}", archive, destinationDirectory)
	return i.archiveExtractor.Extract(archive, destinationDirectory)
}

func (i *nodeModuleInstaller) downloadFileIfMissing(downloadURL, destination string) error {
	if exists(destination) {
		return nil
	}
	i.logger.Info("Downloading {} to {}", downloadURL, destination)
	return i.fileDownloader.Download(downloadURL, destination, i.userName, i.password, i.httpHeaders)
}

// trimVersionPrefix drops the "v" from a version like "v8.15.0" but leaves any
// other leading "v" alone — the original expresses that as the lookahead
// `^v(?=[0-9]+)`, which the tarball name depends on.
func trimVersionPrefix(version string) string {
	if len(version) < 2 || version[0] != 'v' || version[1] < '0' || version[1] > '9' {
		return version
	}
	return version[1:]
}
