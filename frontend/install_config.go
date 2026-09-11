package frontend

// InstallConfig is where an installer puts what it downloads, and what platform
// it is downloading for.
type InstallConfig interface {
	InstallDirectory() string
	WorkingDirectory() string
	CacheResolver() CacheResolver
	Platform() *Platform
}

type defaultInstallConfig struct {
	installDirectory string
	workingDirectory string
	cacheResolver    CacheResolver
	platform         *Platform
}

// NewInstallConfig assembles an install configuration from its four parts.
//
// Both directories are normalised on the way in. The original holds them as
// java.io.File, whose constructor collapses duplicate separators and drops a
// trailing one; every executable path is then built by concatenating onto that
// value, so an un-normalised directory would put a stray separator into the
// middle of every path the plugin reports.
func NewInstallConfig(installDirectory, workingDirectory string, cacheResolver CacheResolver, platform *Platform) InstallConfig {
	return &defaultInstallConfig{
		installDirectory: newFile(installDirectory),
		workingDirectory: newFile(workingDirectory),
		cacheResolver:    cacheResolver,
		platform:         platform,
	}
}

func (c *defaultInstallConfig) InstallDirectory() string { return c.installDirectory }

func (c *defaultInstallConfig) WorkingDirectory() string { return c.workingDirectory }

func (c *defaultInstallConfig) CacheResolver() CacheResolver { return c.cacheResolver }

func (c *defaultInstallConfig) Platform() *Platform { return c.platform }
