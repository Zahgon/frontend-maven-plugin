package frontend

import "strings"

// PnpmRunner runs pnpm.
type PnpmRunner interface {
	NodeTaskRunner
}

const pnpmTaskName = "pnpm"

type defaultPnpmRunner struct {
	*NodeTaskExecutor
}

// NewDefaultPnpmRunner builds the pnpm runner, falling back to the standalone
// executable when the plain .js entry point was not installed.
func NewDefaultPnpmRunner(config NodeExecutorConfig, proxyConfig *ProxyConfig, npmRegistryURL string) PnpmRunner {
	executor := NewNamedNodeTaskExecutor(
		"DefaultPnpmRunner",
		config,
		pnpmTaskName,
		absolutePath(config.PnpmPath()),
		BuildPnpmArguments(proxyConfig, npmRegistryURL),
		nil)

	if !exists(config.PnpmPath()) && exists(config.PnpmExecutablePath()) {
		executor.SetTaskLocation(absolutePath(config.PnpmExecutablePath()))
	}

	return &defaultPnpmRunner{NodeTaskExecutor: executor}
}

// BuildPnpmArguments assembles pnpm's registry and proxy flags. Unlike npm,
// pnpm takes a single comma-separated --noproxy.
//
// Exported so the behaviour can be asserted directly; the original marks it
// package-visible for its own tests.
func BuildPnpmArguments(proxyConfig *ProxyConfig, npmRegistryURL string) []string {
	arguments := make([]string, 0, 4)

	if npmRegistryURL != "" {
		arguments = append(arguments, "--registry="+npmRegistryURL)
	}

	if !proxyConfig.IsEmpty() {
		proxy := selectProxy(proxyConfig, npmRegistryURL)

		arguments = append(arguments, "--https-proxy="+proxy.URI())
		arguments = append(arguments, "--proxy="+proxy.URI())

		nonProxyHosts := proxy.GetNonProxyHosts()
		if nonProxyHosts != "" {
			arguments = append(arguments, "--noproxy="+strings.ReplaceAll(nonProxyHosts, "|", ","))
		}
	}

	return arguments
}
