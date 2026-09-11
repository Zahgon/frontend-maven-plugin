package frontend

// NpxRunner runs npx.
type NpxRunner interface {
	NodeTaskRunner
}

const npxTaskName = "npx"

type defaultNpxRunner struct {
	*NodeTaskExecutor
}

// NewDefaultNpxRunner builds the npx runner.
func NewDefaultNpxRunner(config NodeExecutorConfig, proxyConfig *ProxyConfig, npmRegistryURL string) NpxRunner {
	return &defaultNpxRunner{
		NodeTaskExecutor: NewNamedNodeTaskExecutor(
			"DefaultNpxRunner",
			config,
			npxTaskName,
			absolutePath(config.NpxPath()),
			BuildNpxNpmArguments(proxyConfig, npmRegistryURL),
			nil),
	}
}

// BuildNpxNpmArguments assembles the *npm* flags npx forwards, which have to be
// separated from npx's own arguments by a literal "--":
//
//	npx some-package -- --registry=http://myspecialregisty.com
//
// Exported so the behaviour can be asserted directly; the original marks it
// package-visible for its own tests.
func BuildNpxNpmArguments(proxyConfig *ProxyConfig, npmRegistryURL string) []string {
	arguments := make([]string, 0, 3)

	if npmRegistryURL != "" {
		arguments = append(arguments, "--registry="+npmRegistryURL)
	}

	if !proxyConfig.IsEmpty() {
		proxy := selectProxy(proxyConfig, npmRegistryURL)
		arguments = append(arguments, "--https-proxy="+proxy.URI())
		arguments = append(arguments, "--proxy="+proxy.URI())
	}

	if len(arguments) == 0 {
		return arguments
	}
	return append([]string{"--"}, arguments...)
}
