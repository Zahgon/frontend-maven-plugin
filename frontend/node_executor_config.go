package frontend

import "strings"

// NodeExecutorConfig is where the node binary and the JavaScript entry points of
// the package managers live, once installed.
type NodeExecutorConfig interface {
	NodePath() string
	NpmPath() string
	PnpmPath() string
	PnpmExecutablePath() string
	CorepackPath() string
	NpxPath() string
	InstallDirectory() string
	WorkingDirectory() string
	Platform() *Platform
}

// These are appended to the install directory by concatenation, so each keeps
// its leading separator. NodeInstallPath is "/node", which makes the Windows
// spelling "\node\node.exe".
var (
	nodeWindows = strings.ReplaceAll(NodeInstallPath, "/", `\`) + `\` + WindowsNodeExecutable
	nodeDefault = NodeInstallPath + "/node"
	npmPath     = NodeInstallPath + "/node_modules/npm/bin/npm-cli.js"
	pnpmPath    = NodeInstallPath + "/node_modules/pnpm/bin/pnpm.js"
	pnpmCJS     = NodeInstallPath + "/node_modules/pnpm/bin/pnpm.cjs"
	pnpmMJS     = NodeInstallPath + "/node_modules/pnpm/bin/pnpm.mjs"
	corepackJS  = NodeInstallPath + "/node_modules/corepack/dist/corepack.js"
	npxPath     = NodeInstallPath + "/node_modules/npm/bin/npx-cli.js"
)

type installNodeExecutorConfig struct {
	installConfig InstallConfig
}

// NewInstallNodeExecutorConfig locates the node toolchain inside an install
// directory this plugin populated itself.
func NewInstallNodeExecutorConfig(installConfig InstallConfig) NodeExecutorConfig {
	return &installNodeExecutorConfig{installConfig: installConfig}
}

func (c *installNodeExecutorConfig) NodePath() string {
	nodeExecutable := nodeDefault
	if c.Platform().IsWindows() {
		nodeExecutable = nodeWindows
	}
	return newFile(c.installConfig.InstallDirectory() + nodeExecutable)
}

func (c *installNodeExecutorConfig) NpmPath() string {
	return newFile(c.installConfig.InstallDirectory() + Normalize(npmPath))
}

func (c *installNodeExecutorConfig) PnpmPath() string {
	return newFile(c.installConfig.InstallDirectory() + Normalize(pnpmPath))
}

// PnpmExecutablePath prefers the ESM entry point pnpm ships from v8 and falls
// back to the CommonJS one, which is what older releases install.
func (c *installNodeExecutorConfig) PnpmExecutablePath() string {
	mjs := newFile(c.installConfig.InstallDirectory() + Normalize(pnpmMJS))
	if exists(mjs) {
		return mjs
	}
	return newFile(c.installConfig.InstallDirectory() + Normalize(pnpmCJS))
}

func (c *installNodeExecutorConfig) CorepackPath() string {
	return newFile(c.installConfig.InstallDirectory() + Normalize(corepackJS))
}

func (c *installNodeExecutorConfig) NpxPath() string {
	return newFile(c.installConfig.InstallDirectory() + Normalize(npxPath))
}

func (c *installNodeExecutorConfig) InstallDirectory() string {
	return c.installConfig.InstallDirectory()
}

func (c *installNodeExecutorConfig) WorkingDirectory() string {
	return c.installConfig.WorkingDirectory()
}

func (c *installNodeExecutorConfig) Platform() *Platform {
	return c.installConfig.Platform()
}
