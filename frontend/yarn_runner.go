package frontend

// YarnRunner runs yarn.
type YarnRunner interface {
	NodeTaskRunner
}

const yarnTaskName = "yarn"

type defaultYarnRunner struct {
	*YarnTaskExecutor
}

// NewDefaultYarnRunner builds the yarn runner.
func NewDefaultYarnRunner(config YarnExecutorConfig, proxyConfig *ProxyConfig, npmRegistryURL string) YarnRunner {
	return &defaultYarnRunner{
		YarnTaskExecutor: NewYarnTaskExecutor(
			"DefaultYarnRunner",
			config,
			yarnTaskName,
			BuildYarnArguments(config, proxyConfig, npmRegistryURL)),
	}
}

// BuildYarnArguments assembles yarn's registry and proxy flags.
//
// Yarn Berry takes none of them — passing them makes yarn fail outright — so
// for a Berry project the list is empty.
func BuildYarnArguments(config YarnExecutorConfig, proxyConfig *ProxyConfig, npmRegistryURL string) []string {
	arguments := make([]string, 0, 3)

	if config.IsYarnBerry() {
		return arguments
	}

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
