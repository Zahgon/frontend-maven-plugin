package frontend

// BunRunner runs bun.
type BunRunner interface {
	NodeTaskRunner
}

const bunTaskName = "bun"

type defaultBunRunner struct {
	*BunTaskExecutor
}

// NewDefaultBunRunner builds the bun runner.
func NewDefaultBunRunner(config BunExecutorConfig, proxyConfig *ProxyConfig, npmRegistryURL string) BunRunner {
	return &defaultBunRunner{
		BunTaskExecutor: NewBunTaskExecutor(
			"DefaultBunRunner",
			config,
			bunTaskName,
			BuildBunArguments(proxyConfig, npmRegistryURL)),
	}
}

// BuildBunArguments assembles bun's registry and proxy flags.
func BuildBunArguments(proxyConfig *ProxyConfig, npmRegistryURL string) []string {
	arguments := make([]string, 0, 3)

	if npmRegistryURL != "" {
		arguments = append(arguments, "--registry="+npmRegistryURL)
	}

	if !proxyConfig.IsEmpty() {
		proxy := selectProxy(proxyConfig, npmRegistryURL)
		arguments = append(arguments, "--https-proxy="+proxy.URI())
		arguments = append(arguments, "--proxy="+proxy.URI())
	}

	return arguments
}
