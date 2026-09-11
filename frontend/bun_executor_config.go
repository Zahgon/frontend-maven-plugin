package frontend

import "strings"

// BunExecutorConfig is where the bun binary and its node live once installed.
type BunExecutorConfig interface {
	NodePath() string
	BunPath() string
	WorkingDirectory() string
	Platform() *Platform
}

var (
	bunWindows = strings.ReplaceAll(BunInstallPath, "/", `\`) + `\bun.exe`
	bunDefault = BunInstallPath + "/bun"
)

type installBunExecutorConfig struct {
	nodePath      string
	installConfig InstallConfig
}

// NewInstallBunExecutorConfig locates bun inside an install directory this
// plugin populated itself.
func NewInstallBunExecutorConfig(installConfig InstallConfig) BunExecutorConfig {
	return &installBunExecutorConfig{
		nodePath:      NewInstallNodeExecutorConfig(installConfig).NodePath(),
		installConfig: installConfig,
	}
}

func (c *installBunExecutorConfig) NodePath() string {
	return c.nodePath
}

func (c *installBunExecutorConfig) BunPath() string {
	bunExecutable := bunDefault
	if c.Platform().IsWindows() {
		bunExecutable = bunWindows
	}
	return newFile(c.installConfig.InstallDirectory() + bunExecutable)
}

func (c *installBunExecutorConfig) WorkingDirectory() string {
	return c.installConfig.WorkingDirectory()
}

func (c *installBunExecutorConfig) Platform() *Platform {
	return c.installConfig.Platform()
}
