package goal

import (
	"path/filepath"

	"github.com/eirslett/frontend-maven-plugin/frontend"
)

// The package-manager goals. Each runs its tool in the working directory, and
// each skips itself when an incremental build reports that package.json has not
// changed.

// npmRegistryURLProperty is the system property that overrides a configured
// registry URL at run time.
const npmRegistryURLProperty = "npmRegistryURL"

// packageJSONUnchanged reports whether an incremental build has decided this
// goal has nothing to do, logging the skip message under the given tool name.
func (b *Base) packageJSONUnchanged(tool string) bool {
	buildContext := b.buildContext()
	packageJSON := filepath.Join(b.WorkingDirectory, "package.json")
	if buildContext.HasDelta(packageJSON) || !buildContext.IsIncremental() {
		return false
	}
	log.Info("Skipping " + tool + " install as package.json unchanged")
	return true
}

// proxyConfigFor returns the build's proxies, or none at all when the goal was
// told not to inherit them.
func (b *Base) proxyConfigFor(inherits bool, tool string) *frontend.ProxyConfig {
	if inherits {
		return ProxyConfigFor(b.Session)
	}
	log.Info(tool + " not inheriting proxy config from Maven")
	return frontend.NewProxyConfig(nil)
}

// Npm is the npm goal.
type Npm struct {
	Base
	// Arguments are the npm arguments. Default is "install".
	Arguments string
	// NpmInheritsProxyConfigFromMaven passes the build's proxies on to npm.
	NpmInheritsProxyConfigFromMaven bool
	// NpmRegistryURL is a registry override, passed as the registry option
	// during npm install if set.
	NpmRegistryURL string
	// Skip skips execution of this goal.
	Skip bool
}

// Name is the goal name.
func (g *Npm) Name() string { return nameNpm }

// Params exposes the shared parameters.
func (g *Npm) Params() *Base { return &g.Base }

// SkipExecution reports whether skip.npm is set.
func (g *Npm) SkipExecution() bool { return g.Skip }

// Run runs npm with the configured arguments.
func (g *Npm) Run(factory *frontend.FrontendPluginFactory) error {
	if g.packageJSONUnchanged("npm") {
		return nil
	}
	proxyConfig := g.proxyConfigFor(g.NpmInheritsProxyConfigFromMaven, "npm")
	return factory.NpmRunner(proxyConfig, g.registryURL()).Execute(g.Arguments, g.EnvironmentVariables)
}

func (g *Npm) registryURL() string {
	// Check whether it was overridden via `-D`, otherwise fall back to the
	// configured value.
	return g.Session.SystemProperty(npmRegistryURLProperty, g.NpmRegistryURL)
}

// Npx is the npx goal.
type Npx struct {
	Base
	// Arguments are the npx arguments. Default is "install".
	Arguments string
	// NpmInheritsProxyConfigFromMaven passes the build's proxies on to npx.
	NpmInheritsProxyConfigFromMaven bool
	// NpmRegistryURL is a registry override, passed as the registry option
	// during npm install if set.
	NpmRegistryURL string
	// Skip skips execution of this goal.
	Skip bool
}

// Name is the goal name.
func (g *Npx) Name() string { return nameNpx }

// Params exposes the shared parameters.
func (g *Npx) Params() *Base { return &g.Base }

// SkipExecution reports whether skip.npx is set.
func (g *Npx) SkipExecution() bool { return g.Skip }

// Run runs npx with the configured arguments.
func (g *Npx) Run(factory *frontend.FrontendPluginFactory) error {
	// The original's npx goal reports the skip under npm's name, and that
	// message is contract.
	if g.packageJSONUnchanged("npm") {
		return nil
	}
	proxyConfig := g.proxyConfigFor(g.NpmInheritsProxyConfigFromMaven, "npm")
	return factory.NpxRunner(proxyConfig, g.registryURL()).Execute(g.Arguments, g.EnvironmentVariables)
}

func (g *Npx) registryURL() string {
	return g.Session.SystemProperty(npmRegistryURLProperty, g.NpmRegistryURL)
}

// Pnpm is the pnpm goal.
type Pnpm struct {
	Base
	// Arguments are the pnpm arguments. Default is "install".
	Arguments string
	// PnpmInheritsProxyConfigFromMaven passes the build's proxies on to pnpm.
	PnpmInheritsProxyConfigFromMaven bool
	// PnpmRegistryURL is a registry override, passed as the registry option
	// during pnpm install if set.
	PnpmRegistryURL string
	// Skip skips execution of this goal.
	Skip bool
}

// Name is the goal name.
func (g *Pnpm) Name() string { return namePnpm }

// Params exposes the shared parameters.
func (g *Pnpm) Params() *Base { return &g.Base }

// SkipExecution reports whether skip.pnpm is set.
func (g *Pnpm) SkipExecution() bool { return g.Skip }

// Run runs pnpm with the configured arguments.
func (g *Pnpm) Run(factory *frontend.FrontendPluginFactory) error {
	if g.packageJSONUnchanged("pnpm") {
		return nil
	}
	proxyConfig := g.proxyConfigFor(g.PnpmInheritsProxyConfigFromMaven, "pnpm")
	return factory.PnpmRunner(proxyConfig, g.registryURL()).Execute(g.Arguments, g.EnvironmentVariables)
}

func (g *Pnpm) registryURL() string {
	return g.Session.SystemProperty(npmRegistryURLProperty, g.PnpmRegistryURL)
}

// Yarn is the yarn goal.
type Yarn struct {
	Base
	// Arguments are the yarn arguments. Default is empty.
	Arguments string
	// YarnInheritsProxyConfigFromMaven passes the build's proxies on to yarn.
	YarnInheritsProxyConfigFromMaven bool
	// NpmRegistryURL is a registry override, passed as the registry option
	// during npm install if set.
	NpmRegistryURL string
	// Skip skips execution of this goal.
	Skip bool
}

// Name is the goal name.
func (g *Yarn) Name() string { return nameYarn }

// Params exposes the shared parameters.
func (g *Yarn) Params() *Base { return &g.Base }

// SkipExecution reports whether skip.yarn is set.
func (g *Yarn) SkipExecution() bool { return g.Skip }

// Run runs yarn with the configured arguments.
func (g *Yarn) Run(factory *frontend.FrontendPluginFactory) error {
	if g.packageJSONUnchanged("yarn") {
		return nil
	}
	proxyConfig := g.proxyConfigFor(g.YarnInheritsProxyConfigFromMaven, "yarn")
	isYarnBerry := IsYarnrcYamlFilePresent(g.Session, g.WorkingDirectory)
	return factory.YarnRunner(proxyConfig, g.registryURL(), isYarnBerry).
		Execute(g.Arguments, g.EnvironmentVariables)
}

func (g *Yarn) registryURL() string {
	return g.Session.SystemProperty(npmRegistryURLProperty, g.NpmRegistryURL)
}

// Bun is the bun goal.
type Bun struct {
	Base
	// Arguments are the bun arguments. Default is empty.
	Arguments string
	// BunInheritsProxyConfigFromMaven passes the build's proxies on to bun.
	BunInheritsProxyConfigFromMaven bool
	// NpmRegistryURL is a registry override, passed as the registry option
	// during npm install if set.
	NpmRegistryURL string
	// Skip skips execution of this goal.
	Skip bool
}

// Name is the goal name.
func (g *Bun) Name() string { return nameBun }

// Params exposes the shared parameters.
func (g *Bun) Params() *Base { return &g.Base }

// SkipExecution reports whether skip.bun is set.
func (g *Bun) SkipExecution() bool { return g.Skip }

// Run runs bun with the configured arguments.
func (g *Bun) Run(factory *frontend.FrontendPluginFactory) error {
	if g.packageJSONUnchanged("bun") {
		return nil
	}
	proxyConfig := g.proxyConfigFor(g.BunInheritsProxyConfigFromMaven, "bun")
	return factory.BunRunner(proxyConfig, g.registryURL()).Execute(g.Arguments, g.EnvironmentVariables)
}

func (g *Bun) registryURL() string {
	return g.Session.SystemProperty(npmRegistryURLProperty, g.NpmRegistryURL)
}

// Corepack is the corepack goal.
type Corepack struct {
	Base
	// Arguments are the corepack arguments. Default is "enable".
	Arguments string
	// Skip skips execution of this goal.
	Skip bool
}

// Name is the goal name.
func (g *Corepack) Name() string { return nameCorepack }

// Params exposes the shared parameters.
func (g *Corepack) Params() *Base { return &g.Base }

// SkipExecution reports whether skip.corepack is set.
func (g *Corepack) SkipExecution() bool { return g.Skip }

// Run runs corepack with the configured arguments.
func (g *Corepack) Run(factory *frontend.FrontendPluginFactory) error {
	if g.packageJSONUnchanged("corepack") {
		return nil
	}
	return factory.CorepackRunner().Execute(g.Arguments, g.EnvironmentVariables)
}
