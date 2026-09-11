package goal

import "github.com/eirslett/frontend-maven-plugin/frontend"

// The five install goals. Each downloads a Node.js distribution and, except for
// install-bun, one package manager alongside it, taking its download
// credentials from the server the serverId parameter names.

// InstallNodeAndNpm is the install-node-and-npm goal.
type InstallNodeAndNpm struct {
	Base
	// NodeDownloadRoot is where to download the Node.js binary from. Defaults
	// to https://nodejs.org/dist/
	NodeDownloadRoot string
	// NpmDownloadRoot is where to download the NPM binary from. Defaults to
	// https://registry.npmjs.org/npm/-/
	NpmDownloadRoot string
	// DownloadRoot is where to download Node.js and NPM binaries from.
	//
	// Deprecated: use NodeDownloadRoot and NpmDownloadRoot instead. This
	// configuration is used only when neither of those is specified.
	DownloadRoot string
	// NodeVersion is the version of Node.js to install. IMPORTANT! Most Node.js
	// version names start with 'v', for example 'v0.10.18'.
	NodeVersion string
	// NpmVersion is the version of NPM to install.
	NpmVersion string
	// ServerID names the server holding the download username and password.
	ServerID string
	// Skip skips execution of this goal.
	Skip bool
}

// Name is the goal name.
func (g *InstallNodeAndNpm) Name() string { return nameInstallNodeAndNpm }

// Params exposes the shared parameters.
func (g *InstallNodeAndNpm) Params() *Base { return &g.Base }

// SkipExecution reports whether skip.installnodenpm is set.
func (g *InstallNodeAndNpm) SkipExecution() bool { return g.Skip }

// Run installs Node.js and then npm.
func (g *InstallNodeAndNpm) Run(factory *frontend.FrontendPluginFactory) error {
	proxyConfig := ProxyConfigFor(g.Session)
	nodeDownloadRoot := g.nodeDownloadRoot()
	npmDownloadRoot := g.npmDownloadRoot()
	server := DecryptServer(g.ServerID, g.Session)

	nodeInstaller := factory.NodeInstaller(proxyConfig).
		SetNodeVersion(g.NodeVersion).
		SetNodeDownloadRoot(nodeDownloadRoot).
		SetNpmVersion(g.NpmVersion)
	npmInstaller := factory.NPMInstaller(proxyConfig).
		SetNodeVersion(g.NodeVersion).
		SetNpmVersion(g.NpmVersion).
		SetNpmDownloadRoot(npmDownloadRoot)

	if server != nil {
		httpHeaders := HTTPHeadersOf(server)
		nodeInstaller.SetUserName(server.Username).SetPassword(server.Password).SetHTTPHeaders(httpHeaders)
		npmInstaller.SetUserName(server.Username).SetPassword(server.Password).SetHTTPHeaders(httpHeaders)
	}

	if err := nodeInstaller.Install(); err != nil {
		return err
	}
	return npmInstaller.Install()
}

func (g *InstallNodeAndNpm) nodeDownloadRoot() string {
	if g.DownloadRoot != "" && g.NodeDownloadRoot == "" {
		return g.DownloadRoot
	}
	return g.NodeDownloadRoot
}

func (g *InstallNodeAndNpm) npmDownloadRoot() string {
	if g.DownloadRoot != "" && g.NpmDownloadRoot == frontend.DefaultNpmDownloadRoot {
		return g.DownloadRoot
	}
	return g.NpmDownloadRoot
}

// InstallNodeAndYarn is the install-node-and-yarn goal.
type InstallNodeAndYarn struct {
	Base
	// NodeDownloadRoot is where to download the Node.js binary from. Defaults
	// to https://nodejs.org/dist/
	NodeDownloadRoot string
	// YarnDownloadRoot is where to download the Yarn binary from. Defaults to
	// https://github.com/yarnpkg/yarn/releases/download/...
	YarnDownloadRoot string
	// NodeVersion is the version of Node.js to install. IMPORTANT! Most Node.js
	// version names start with 'v', for example 'v0.10.18'.
	NodeVersion string
	// YarnVersion is the version of Yarn to install. IMPORTANT! Most Yarn names
	// start with 'v', for example 'v0.15.0'.
	YarnVersion string
	// ServerID names the server holding the download username and password.
	ServerID string
	// Skip skips execution of this goal.
	Skip bool
}

// Name is the goal name.
func (g *InstallNodeAndYarn) Name() string { return nameInstallNodeAndYarn }

// Params exposes the shared parameters.
func (g *InstallNodeAndYarn) Params() *Base { return &g.Base }

// SkipExecution reports whether skip.installyarn is set.
func (g *InstallNodeAndYarn) SkipExecution() bool { return g.Skip }

// Run installs Node.js and then Yarn.
func (g *InstallNodeAndYarn) Run(factory *frontend.FrontendPluginFactory) error {
	proxyConfig := ProxyConfigFor(g.Session)
	server := DecryptServer(g.ServerID, g.Session)

	isYarnYamlFilePresent := IsYarnrcYamlFilePresent(g.Session, g.WorkingDirectory)

	nodeInstaller := factory.NodeInstaller(proxyConfig).
		SetNodeDownloadRoot(g.NodeDownloadRoot).
		SetNodeVersion(g.NodeVersion)
	yarnInstaller := factory.YarnInstaller(proxyConfig).
		SetYarnDownloadRoot(g.YarnDownloadRoot).
		SetYarnVersion(g.YarnVersion).
		SetIsYarnBerry(isYarnYamlFilePresent)

	if server != nil {
		httpHeaders := HTTPHeadersOf(server)
		nodeInstaller.SetUserName(server.Username).SetPassword(server.Password).SetHTTPHeaders(httpHeaders)
		yarnInstaller.SetUserName(server.Username).SetPassword(server.Password).SetHTTPHeaders(httpHeaders)
	}

	if err := nodeInstaller.Install(); err != nil {
		return err
	}
	return yarnInstaller.Install()
}

// InstallNodeAndPnpm is the install-node-and-pnpm goal.
type InstallNodeAndPnpm struct {
	Base
	// NodeDownloadRoot is where to download the Node.js binary from. Defaults
	// to https://nodejs.org/dist/
	NodeDownloadRoot string
	// PnpmDownloadRoot is where to download the pnpm binary from. Defaults to
	// https://registry.npmjs.org/pnpm/-/
	PnpmDownloadRoot string
	// DownloadRoot is where to download Node.js and pnpm binaries from.
	//
	// Deprecated: use NodeDownloadRoot and PnpmDownloadRoot instead. This
	// configuration is used only when neither of those is specified.
	DownloadRoot string
	// NodeVersion is the version of Node.js to install. IMPORTANT! Most Node.js
	// version names start with 'v', for example 'v0.10.18'.
	NodeVersion string
	// PnpmVersion is the version of pnpm to install. The version string can
	// optionally be prefixed with 'v' (both 'v1.2.3' and '1.2.3' are valid).
	PnpmVersion string
	// ServerID names the server holding the download username and password.
	ServerID string
	// Skip skips execution of this goal.
	Skip bool
}

// Name is the goal name.
func (g *InstallNodeAndPnpm) Name() string { return nameInstallNodeAndPnpm }

// Params exposes the shared parameters.
func (g *InstallNodeAndPnpm) Params() *Base { return &g.Base }

// SkipExecution reports whether skip.installnodepnpm is set.
func (g *InstallNodeAndPnpm) SkipExecution() bool { return g.Skip }

// Run installs Node.js and then pnpm.
func (g *InstallNodeAndPnpm) Run(factory *frontend.FrontendPluginFactory) error {
	proxyConfig := ProxyConfigFor(g.Session)
	// Named apart from the parameters they resolve, to keep the deprecated
	// downloadRoot fallback readable.
	resolvedNodeDownloadRoot := g.nodeDownloadRoot()
	resolvedPnpmDownloadRoot := g.pnpmDownloadRoot()
	server := DecryptServer(g.ServerID, g.Session)

	nodeInstaller := factory.NodeInstaller(proxyConfig).
		SetNodeVersion(g.NodeVersion).
		SetNodeDownloadRoot(resolvedNodeDownloadRoot)
	pnpmInstaller := factory.PnpmInstaller(proxyConfig).
		SetPnpmVersion(g.PnpmVersion).
		SetPnpmDownloadRoot(resolvedPnpmDownloadRoot)

	if server != nil {
		httpHeaders := HTTPHeadersOf(server)
		nodeInstaller.SetUserName(server.Username).SetPassword(server.Password).SetHTTPHeaders(httpHeaders)
		pnpmInstaller.SetUserName(server.Username).SetPassword(server.Password).SetHTTPHeaders(httpHeaders)
	}

	if err := nodeInstaller.Install(); err != nil {
		return err
	}
	return pnpmInstaller.Install()
}

func (g *InstallNodeAndPnpm) nodeDownloadRoot() string {
	if g.DownloadRoot != "" && g.NodeDownloadRoot == "" {
		return g.DownloadRoot
	}
	return g.NodeDownloadRoot
}

func (g *InstallNodeAndPnpm) pnpmDownloadRoot() string {
	if g.DownloadRoot != "" && g.PnpmDownloadRoot == frontend.DefaultPnpmDownloadRoot {
		return g.DownloadRoot
	}
	return g.PnpmDownloadRoot
}

// InstallNodeAndCorepack is the install-node-and-corepack goal.
type InstallNodeAndCorepack struct {
	Base
	// NodeDownloadRoot is where to download the Node.js binary from. Defaults
	// to https://nodejs.org/dist/
	NodeDownloadRoot string
	// CorepackDownloadRoot is where to download the corepack binary from.
	// Defaults to https://registry.npmjs.org/corepack/-/
	CorepackDownloadRoot string
	// NodeVersion is the version of Node.js to install. IMPORTANT! Most Node.js
	// version names start with 'v', for example 'v0.10.18'.
	NodeVersion string
	// CorepackVersion is the version of corepack to install, optionally
	// prefixed with 'v'. When not provided, the corepack version bundled with
	// Node is used.
	CorepackVersion string
	// ServerID names the server holding the download username and password.
	ServerID string
	// Skip skips execution of this goal.
	Skip bool
}

// Name is the goal name.
func (g *InstallNodeAndCorepack) Name() string { return nameInstallNodeAndCorepack }

// Params exposes the shared parameters.
func (g *InstallNodeAndCorepack) Params() *Base { return &g.Base }

// SkipExecution reports whether skip.installnodecorepack is set.
func (g *InstallNodeAndCorepack) SkipExecution() bool { return g.Skip }

// Run installs Node.js and then corepack.
func (g *InstallNodeAndCorepack) Run(factory *frontend.FrontendPluginFactory) error {
	proxyConfig := ProxyConfigFor(g.Session)
	resolvedNodeDownloadRoot := g.NodeDownloadRoot
	resolvedCorepackDownloadRoot := g.CorepackDownloadRoot

	// Set up the installers.
	nodeInstaller := factory.NodeInstaller(proxyConfig)
	nodeInstaller.SetNodeVersion(g.NodeVersion).SetNodeDownloadRoot(resolvedNodeDownloadRoot)
	if g.CorepackVersion == "provided" {
		// This makes the node installer copy over the whole node_modules
		// directory, corepack module included.
		nodeInstaller.SetNpmVersion("provided")
	}
	corepackInstaller := factory.CorepackInstaller(proxyConfig)
	corepackInstaller.SetCorepackVersion(g.CorepackVersion).SetCorepackDownloadRoot(resolvedCorepackDownloadRoot)

	// If applicable, configure authentication details.
	if server := DecryptServer(g.ServerID, g.Session); server != nil {
		httpHeaders := HTTPHeadersOf(server)
		nodeInstaller.SetUserName(server.Username).SetPassword(server.Password).SetHTTPHeaders(httpHeaders)
		corepackInstaller.SetUserName(server.Username).SetPassword(server.Password).SetHTTPHeaders(httpHeaders)
	}

	// Perform the installation.
	if err := nodeInstaller.Install(); err != nil {
		return err
	}
	return corepackInstaller.Install()
}

// InstallBun is the install-bun goal.
type InstallBun struct {
	Base
	// BunDownloadRoot is where to download the Bun binary from. Defaults to
	// https://github.com/oven-sh/bun/releases/download/
	BunDownloadRoot string
	// BunVersion is the version of Bun to install. IMPORTANT! Most Bun version
	// names start with 'v', for example 'v1.0.0'.
	BunVersion string
	// ServerID names the server holding the download username and password.
	ServerID string
	// Skip skips execution of this goal.
	Skip bool
}

// Name is the goal name.
func (g *InstallBun) Name() string { return nameInstallBun }

// Params exposes the shared parameters.
func (g *InstallBun) Params() *Base { return &g.Base }

// SkipExecution reports whether skip.installbun is set.
func (g *InstallBun) SkipExecution() bool { return g.Skip }

// Run installs Bun.
func (g *InstallBun) Run(factory *frontend.FrontendPluginFactory) error {
	proxyConfig := ProxyConfigFor(g.Session)
	server := DecryptServer(g.ServerID, g.Session)

	bunInstaller := factory.BunInstaller(proxyConfig).
		SetBunVersion(g.BunVersion).
		SetBunDownloadRoot(g.BunDownloadRoot)
	if server != nil {
		bunInstaller.SetUserName(server.Username).
			SetPassword(server.Password).
			SetHTTPHeaders(HTTPHeadersOf(server))
	}
	return bunInstaller.Install()
}
