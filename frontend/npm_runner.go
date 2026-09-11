package frontend

import "strings"

// NpmRunner runs npm.
type NpmRunner interface {
	NodeTaskRunner
}

const npmTaskName = "npm"

type defaultNpmRunner struct {
	*NodeTaskExecutor
}

// NewDefaultNpmRunner builds the npm runner, wiring the proxy configuration into
// both npm's flags and its environment.
func NewDefaultNpmRunner(config NodeExecutorConfig, proxyConfig *ProxyConfig, npmRegistryURL string) NpmRunner {
	return &defaultNpmRunner{
		NodeTaskExecutor: NewNamedNodeTaskExecutor(
			"DefaultNpmRunner",
			config,
			npmTaskName,
			absolutePath(config.NpmPath()),
			BuildNpmArguments(proxyConfig, npmRegistryURL),
			buildNpmProxyEnvironment(proxyConfig, npmRegistryURL)),
	}
}

// BuildNpmArguments assembles npm's registry and proxy flags.
//
// Exported so the behaviour can be asserted directly; the original marks it
// package-visible for its own tests.
func BuildNpmArguments(proxyConfig *ProxyConfig, npmRegistryURL string) []string {
	arguments := make([]string, 0, 6)

	if npmRegistryURL != "" {
		arguments = append(arguments, "--registry="+npmRegistryURL)
	}

	if !proxyConfig.IsEmpty() {
		proxy := selectProxy(proxyConfig, npmRegistryURL)

		arguments = append(arguments, "--https-proxy="+proxy.URI())
		arguments = append(arguments, "--proxy="+proxy.URI())

		nonProxyHosts := proxy.GetNonProxyHosts()
		if nonProxyHosts != "" {
			for _, nonProxyHost := range strings.Split(nonProxyHosts, "|") {
				arguments = append(arguments, "--noproxy="+strings.ReplaceAll(nonProxyHost, "*", ""))
			}
		}
	}

	return arguments
}

func buildNpmProxyEnvironment(proxyConfig *ProxyConfig, npmRegistryURL string) map[string]string {
	if proxyConfig.IsEmpty() {
		return map[string]string{}
	}
	proxy := selectProxy(proxyConfig, npmRegistryURL)
	return map[string]string{
		"https_proxy": proxy.URI(),
		"http_proxy":  proxy.URI(),
	}
}

// selectProxy picks the proxy a package manager should be pointed at: the one
// matching the registry, else the first secure one, else the first insecure one.
func selectProxy(proxyConfig *ProxyConfig, npmRegistryURL string) *Proxy {
	var proxy *Proxy
	if npmRegistryURL != "" {
		proxy = proxyConfig.ProxyForURL(npmRegistryURL)
	}
	if proxy == nil {
		proxy = proxyConfig.SecureProxy()
	}
	if proxy == nil {
		proxy = proxyConfig.InsecureProxy()
	}
	return proxy
}
