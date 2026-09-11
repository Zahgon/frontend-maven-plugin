package frontend

import "strings"

// YarnExecutorConfig is where the yarn launcher and its node live, plus whether
// the project is on Yarn Berry — which takes none of the flags Yarn 1 accepts.
type YarnExecutorConfig interface {
	NodePath() string
	YarnPath() string
	WorkingDirectory() string
	Platform() *Platform
	IsYarnBerry() bool
}

var (
	yarnWindows = strings.ReplaceAll(YarnInstallPath+"/dist/bin/yarn.cmd", "/", `\`)
	yarnDefault = YarnInstallPath + "/dist/bin/yarn"
)

type installYarnExecutorConfig struct {
	nodePath      string
	installConfig InstallConfig
	isYarnBerry   bool
}

// NewInstallYarnExecutorConfig locates yarn inside an install directory this
// plugin populated itself.
func NewInstallYarnExecutorConfig(installConfig InstallConfig, isYarnBerry bool) YarnExecutorConfig {
	return &installYarnExecutorConfig{
		nodePath:      NewInstallNodeExecutorConfig(installConfig).NodePath(),
		installConfig: installConfig,
		isYarnBerry:   isYarnBerry,
	}
}

func (c *installYarnExecutorConfig) NodePath() string {
	return c.nodePath
}

func (c *installYarnExecutorConfig) YarnPath() string {
	yarnExecutable := yarnDefault
	if c.Platform().IsWindows() {
		yarnExecutable = yarnWindows
	}
	return newFile(c.installConfig.InstallDirectory() + yarnExecutable)
}

func (c *installYarnExecutorConfig) WorkingDirectory() string {
	return c.installConfig.WorkingDirectory()
}

func (c *installYarnExecutorConfig) Platform() *Platform {
	return c.installConfig.Platform()
}

func (c *installYarnExecutorConfig) IsYarnBerry() bool {
	return c.isYarnBerry
}
