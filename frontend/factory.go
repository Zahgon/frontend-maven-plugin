package frontend

// defaultCachePath is where downloads are cached when the caller supplies no
// resolver of its own.
const defaultCachePath = "cache"

// defaultPlatform is guessed once, as the original does in a static initialiser.
var defaultPlatform = GuessPlatform()

// FrontendPluginFactory hands out the installers and runners, all wired to one
// working directory, install directory and download cache.
type FrontendPluginFactory struct {
	workingDirectory string
	installDirectory string
	cacheResolver    CacheResolver
}

// NewFrontendPluginFactory caches downloads in a "cache" directory under the
// install directory.
func NewFrontendPluginFactory(workingDirectory, installDirectory string) *FrontendPluginFactory {
	return NewFrontendPluginFactoryWithCache(workingDirectory, installDirectory, defaultCacheResolver(installDirectory))
}

// NewFrontendPluginFactoryWithCache caches downloads wherever the supplied
// resolver decides.
func NewFrontendPluginFactoryWithCache(workingDirectory, installDirectory string, cacheResolver CacheResolver) *FrontendPluginFactory {
	return &FrontendPluginFactory{
		workingDirectory: workingDirectory,
		installDirectory: installDirectory,
		cacheResolver:    cacheResolver,
	}
}

// BunInstaller builds an installer for Bun.
func (f *FrontendPluginFactory) BunInstaller(proxy *ProxyConfig) *BunInstaller {
	return NewBunInstaller(f.installConfig(), NewDefaultArchiveExtractor(), NewDefaultFileDownloader(proxy))
}

// NodeInstaller builds an installer for Node.js.
func (f *FrontendPluginFactory) NodeInstaller(proxy *ProxyConfig) *NodeInstaller {
	return NewNodeInstaller(f.installConfig(), NewDefaultArchiveExtractor(), NewDefaultFileDownloader(proxy))
}

// NPMInstaller builds an installer for npm.
func (f *FrontendPluginFactory) NPMInstaller(proxy *ProxyConfig) *NPMInstaller {
	return NewNPMInstaller(f.installConfig(), NewDefaultArchiveExtractor(), NewDefaultFileDownloader(proxy))
}

// CorepackInstaller builds an installer for corepack.
func (f *FrontendPluginFactory) CorepackInstaller(proxy *ProxyConfig) *CorepackInstaller {
	return NewCorepackInstaller(f.installConfig(), NewDefaultArchiveExtractor(), NewDefaultFileDownloader(proxy))
}

// PnpmInstaller builds an installer for pnpm.
func (f *FrontendPluginFactory) PnpmInstaller(proxy *ProxyConfig) *PnpmInstaller {
	return NewPnpmInstaller(f.installConfig(), NewDefaultArchiveExtractor(), NewDefaultFileDownloader(proxy))
}

// YarnInstaller builds an installer for Yarn.
func (f *FrontendPluginFactory) YarnInstaller(proxy *ProxyConfig) *YarnInstaller {
	return NewYarnInstaller(f.installConfig(), NewDefaultArchiveExtractor(), NewDefaultFileDownloader(proxy))
}

// BowerRunner builds a runner for bower.
func (f *FrontendPluginFactory) BowerRunner(proxy *ProxyConfig) BowerRunner {
	return NewDefaultBowerRunner(f.executorConfig(), proxy)
}

// BunRunner builds a runner for bun.
func (f *FrontendPluginFactory) BunRunner(proxy *ProxyConfig, npmRegistryURL string) BunRunner {
	return NewDefaultBunRunner(NewInstallBunExecutorConfig(f.installConfig()), proxy, npmRegistryURL)
}

// JspmRunner builds a runner for jspm.
func (f *FrontendPluginFactory) JspmRunner() JspmRunner {
	return NewDefaultJspmRunner(f.executorConfig())
}

// NpmRunner builds a runner for npm.
func (f *FrontendPluginFactory) NpmRunner(proxy *ProxyConfig, npmRegistryURL string) NpmRunner {
	return NewDefaultNpmRunner(f.executorConfig(), proxy, npmRegistryURL)
}

// CorepackRunner builds a runner for corepack.
func (f *FrontendPluginFactory) CorepackRunner() CorepackRunner {
	return NewDefaultCorepackRunner(f.executorConfig())
}

// PnpmRunner builds a runner for pnpm.
func (f *FrontendPluginFactory) PnpmRunner(proxyConfig *ProxyConfig, npmRegistryURL string) PnpmRunner {
	return NewDefaultPnpmRunner(f.executorConfig(), proxyConfig, npmRegistryURL)
}

// NpxRunner builds a runner for npx.
func (f *FrontendPluginFactory) NpxRunner(proxy *ProxyConfig, npmRegistryURL string) NpxRunner {
	return NewDefaultNpxRunner(f.executorConfig(), proxy, npmRegistryURL)
}

// YarnRunner builds a runner for yarn.
func (f *FrontendPluginFactory) YarnRunner(proxy *ProxyConfig, npmRegistryURL string, isYarnBerry bool) YarnRunner {
	return NewDefaultYarnRunner(NewInstallYarnExecutorConfig(f.installConfig(), isYarnBerry), proxy, npmRegistryURL)
}

// GruntRunner builds a runner for grunt.
func (f *FrontendPluginFactory) GruntRunner() GruntRunner {
	return NewDefaultGruntRunner(f.executorConfig())
}

// EmberRunner builds a runner for ember.
func (f *FrontendPluginFactory) EmberRunner() EmberRunner {
	return NewDefaultEmberRunner(f.executorConfig())
}

// KarmaRunner builds a runner for karma.
func (f *FrontendPluginFactory) KarmaRunner() KarmaRunner {
	return NewDefaultKarmaRunner(f.executorConfig())
}

// GulpRunner builds a runner for gulp.
func (f *FrontendPluginFactory) GulpRunner() GulpRunner {
	return NewDefaultGulpRunner(f.executorConfig())
}

// WebpackRunner builds a runner for webpack.
func (f *FrontendPluginFactory) WebpackRunner() WebpackRunner {
	return NewDefaultWebpackRunner(f.executorConfig())
}

func (f *FrontendPluginFactory) executorConfig() NodeExecutorConfig {
	return NewInstallNodeExecutorConfig(f.installConfig())
}

func (f *FrontendPluginFactory) installConfig() InstallConfig {
	return NewInstallConfig(f.installDirectory, f.workingDirectory, f.cacheResolver, defaultPlatform)
}

func defaultCacheResolver(root string) CacheResolver {
	return NewDirectoryCacheResolver(childFile(root, defaultCachePath))
}
