package frontend

import (
	"sync"

	"github.com/eirslett/frontend-maven-plugin/internal/logging"
)

// DefaultCorepackDownloadRoot is where corepack tarballs are fetched from.
const DefaultCorepackDownloadRoot = "https://registry.npmjs.org/corepack/-/"

var corepackInstallLock sync.Mutex

// CorepackInstaller downloads a corepack release into the Node.js installation
// and links a launcher for it next to node.
type CorepackInstaller struct {
	installer *nodeModuleInstaller
}

// NewCorepackInstaller builds an installer against one install configuration.
func NewCorepackInstaller(config InstallConfig, archiveExtractor ArchiveExtractor, fileDownloader FileDownloader) *CorepackInstaller {
	return &CorepackInstaller{
		installer: &nodeModuleInstaller{
			tool: nodeModuleTool{
				toolName:            "corepack",
				displayName:         "corepack",
				defaultDownloadRoot: DefaultCorepackDownloadRoot,
				cliEnvVar:           "COREPACK_CLI_JS",
				executablePath: func(config NodeExecutorConfig) string {
					return config.CorepackPath()
				},
				// Unlike pnpm, corepack can be asked for the copy the Node.js
				// distribution bundles, in which case nothing is downloaded.
				versionIsProvided: func(version string) bool {
					return version == "provided"
				},
			},
			logger:           logging.GetLogger("CorepackInstaller"),
			config:           config,
			archiveExtractor: archiveExtractor,
			fileDownloader:   fileDownloader,
		},
	}
}

// SetNodeVersion is accepted for symmetry with the other installers and has no
// effect: which Node.js is installed does not change how corepack is fetched.
func (i *CorepackInstaller) SetNodeVersion(string) *CorepackInstaller {
	return i
}

// SetCorepackVersion selects the corepack version, with or without a leading
// "v", or "provided" to keep the one bundled with Node.js.
func (i *CorepackInstaller) SetCorepackVersion(corepackVersion string) *CorepackInstaller {
	i.installer.version = corepackVersion
	return i
}

// SetCorepackDownloadRoot overrides where the corepack tarball is fetched from.
func (i *CorepackInstaller) SetCorepackDownloadRoot(corepackDownloadRoot string) *CorepackInstaller {
	i.installer.downloadRoot = corepackDownloadRoot
	return i
}

// SetUserName sets the download username.
func (i *CorepackInstaller) SetUserName(userName string) *CorepackInstaller {
	i.installer.userName = userName
	return i
}

// SetPassword sets the download password.
func (i *CorepackInstaller) SetPassword(password string) *CorepackInstaller {
	i.installer.password = password
	return i
}

// SetHTTPHeaders sets extra headers to send with the download.
func (i *CorepackInstaller) SetHTTPHeaders(httpHeaders map[string]string) *CorepackInstaller {
	i.installer.httpHeaders = httpHeaders
	return i
}

// Install downloads and installs corepack unless the requested version is
// already in place, then links its launcher.
func (i *CorepackInstaller) Install() error {
	corepackInstallLock.Lock()
	defer corepackInstallLock.Unlock()
	return i.installer.install()
}
