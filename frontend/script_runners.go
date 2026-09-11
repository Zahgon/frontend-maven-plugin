package frontend

// The remaining runners all do the same thing: run one JavaScript entry point
// out of the project's node_modules under the installed node, with no flags of
// their own. The original spells each of them out as a two-line subclass; here
// they share one constructor and differ only in where their script lives.

// BowerRunner runs bower.
type BowerRunner interface {
	NodeTaskRunner
}

// CorepackRunner runs corepack.
type CorepackRunner interface {
	NodeTaskRunner
}

// EmberRunner runs ember-cli.
type EmberRunner interface {
	NodeTaskRunner
}

// GruntRunner runs grunt.
type GruntRunner interface {
	NodeTaskRunner
}

// GulpRunner runs gulp.
type GulpRunner interface {
	NodeTaskRunner
}

// JspmRunner runs jspm.
type JspmRunner interface {
	NodeTaskRunner
}

// KarmaRunner runs karma.
type KarmaRunner interface {
	NodeTaskRunner
}

// WebpackRunner runs webpack.
type WebpackRunner interface {
	NodeTaskRunner
}

// Where each tool's entry point lives inside the project's node_modules.
const (
	bowerTaskLocation   = "node_modules/bower/bin/bower"
	emberTaskLocation   = "node_modules/ember-cli/bin/ember"
	gruntTaskLocation   = "node_modules/grunt-cli/bin/grunt"
	gulpTaskLocation    = "node_modules/gulp/bin/gulp.js"
	jspmTaskLocation    = "node_modules/jspm/jspm.js"
	karmaTaskLocation   = "node_modules/karma/bin/karma"
	webpackTaskLocation = "node_modules/webpack/bin/webpack.js"
)

const corepackTaskName = "corepack"

type scriptRunner struct {
	*NodeTaskExecutor
}

// NewDefaultBowerRunner builds the bower runner, passing bower the build's proxy
// settings through its own --config flags.
func NewDefaultBowerRunner(config NodeExecutorConfig, proxyConfig *ProxyConfig) BowerRunner {
	return &scriptRunner{
		NodeTaskExecutor: NewNodeTaskExecutorWithArguments(
			"DefaultBowerRunner", config, bowerTaskLocation, BuildBowerArguments(proxyConfig)),
	}
}

// BuildBowerArguments assembles bower's proxy flags, which it namespaces under
// --config unlike every other tool here.
func BuildBowerArguments(proxyConfig *ProxyConfig) []string {
	arguments := make([]string, 0, 2)

	if !proxyConfig.IsEmpty() {
		if secureProxy := proxyConfig.SecureProxy(); secureProxy != nil {
			arguments = append(arguments, "--config.https-proxy="+secureProxy.URI())
		}
		if insecureProxy := proxyConfig.InsecureProxy(); insecureProxy != nil {
			arguments = append(arguments, "--config.proxy="+insecureProxy.URI())
		}
	}
	return arguments
}

// NewDefaultCorepackRunner builds the corepack runner, which runs the corepack
// installed alongside node rather than one from the project.
func NewDefaultCorepackRunner(config NodeExecutorConfig) CorepackRunner {
	executor := NewNamedNodeTaskExecutor(
		"DefaultCorepackRunner", config, corepackTaskName, absolutePath(config.CorepackPath()), nil, nil)
	if !exists(config.CorepackPath()) {
		executor.SetTaskLocation(absolutePath(config.CorepackPath()))
	}
	return &scriptRunner{NodeTaskExecutor: executor}
}

// NewDefaultEmberRunner builds the ember runner.
func NewDefaultEmberRunner(config NodeExecutorConfig) EmberRunner {
	return &scriptRunner{NodeTaskExecutor: NewNodeTaskExecutor("DefaultEmberRunner", config, emberTaskLocation)}
}

// NewDefaultGruntRunner builds the grunt runner.
func NewDefaultGruntRunner(config NodeExecutorConfig) GruntRunner {
	return &scriptRunner{NodeTaskExecutor: NewNodeTaskExecutor("DefaultGruntRunner", config, gruntTaskLocation)}
}

// NewDefaultGulpRunner builds the gulp runner.
func NewDefaultGulpRunner(config NodeExecutorConfig) GulpRunner {
	return &scriptRunner{NodeTaskExecutor: NewNodeTaskExecutor("DefaultGulpRunner", config, gulpTaskLocation)}
}

// NewDefaultJspmRunner builds the jspm runner.
func NewDefaultJspmRunner(config NodeExecutorConfig) JspmRunner {
	return &scriptRunner{NodeTaskExecutor: NewNodeTaskExecutor("DefaultJspmRunner", config, jspmTaskLocation)}
}

// NewDefaultKarmaRunner builds the karma runner.
func NewDefaultKarmaRunner(config NodeExecutorConfig) KarmaRunner {
	return &scriptRunner{NodeTaskExecutor: NewNodeTaskExecutor("DefaultKarmaRunner", config, karmaTaskLocation)}
}

// NewDefaultWebpackRunner builds the webpack runner.
func NewDefaultWebpackRunner(config NodeExecutorConfig) WebpackRunner {
	return &scriptRunner{NodeTaskExecutor: NewNodeTaskExecutor("DefaultWebpackRunner", config, webpackTaskLocation)}
}
