package frontend

import (
	"sync"

	"github.com/eirslett/frontend-maven-plugin/internal/logging"
)

// DefaultPnpmDownloadRoot is where pnpm tarballs are fetched from.
const DefaultPnpmDownloadRoot = "https://registry.npmjs.org/pnpm/-/"

var pnpmInstallLock sync.Mutex

// PnpmInstaller downloads a pnpm release into the Node.js installation and links
// a launcher for it next to node.
type PnpmInstaller struct {
	installer *nodeModuleInstaller
}

// NewPnpmInstaller builds an installer against one install configuration.
func NewPnpmInstaller(config InstallConfig, archiveExtractor ArchiveExtractor, fileDownloader FileDownloader) *PnpmInstaller {
	return &PnpmInstaller{
		installer: &nodeModuleInstaller{
			tool: nodeModuleTool{
				toolName:            "pnpm",
				displayName:         "PNPM",
				defaultDownloadRoot: DefaultPnpmDownloadRoot,
				cliEnvVar:           "PNPM_CLI_JS",
				executablePath: func(config NodeExecutorConfig) string {
					return config.PnpmExecutablePath()
				},
			},
			logger:           logging.GetLogger("PnpmInstaller"),
			config:           config,
			archiveExtractor: archiveExtractor,
			fileDownloader:   fileDownloader,
		},
	}
}

// SetNodeVersion is accepted for symmetry with the other installers and has no
// effect: which Node.js is installed does not change how pnpm is fetched.
func (i *PnpmInstaller) SetNodeVersion(string) *PnpmInstaller {
	return i
}

// SetPnpmVersion selects the pnpm version, with or without a leading "v".
func (i *PnpmInstaller) SetPnpmVersion(pnpmVersion string) *PnpmInstaller {
	i.installer.version = pnpmVersion
	return i
}

// SetPnpmDownloadRoot overrides where the pnpm tarball is fetched from.
func (i *PnpmInstaller) SetPnpmDownloadRoot(pnpmDownloadRoot string) *PnpmInstaller {
	i.installer.downloadRoot = pnpmDownloadRoot
	return i
}

// SetUserName sets the download username.
func (i *PnpmInstaller) SetUserName(userName string) *PnpmInstaller {
	i.installer.userName = userName
	return i
}

// SetPassword sets the download password.
func (i *PnpmInstaller) SetPassword(password string) *PnpmInstaller {
	i.installer.password = password
	return i
}

// SetHTTPHeaders sets extra headers to send with the download.
func (i *PnpmInstaller) SetHTTPHeaders(httpHeaders map[string]string) *PnpmInstaller {
	i.installer.httpHeaders = httpHeaders
	return i
}

// Install downloads and installs pnpm unless the requested version is already in
// place, then links its launcher.
func (i *PnpmInstaller) Install() error {
	pnpmInstallLock.Lock()
	defer pnpmInstallLock.Unlock()
	return i.installer.install()
}
