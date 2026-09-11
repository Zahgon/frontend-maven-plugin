package frontend

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// The runners are thin, but which flags each package manager gets and where each
// tool's entry point is looked for are both contract.

func testProxies(t *testing.T) (*Proxy, *Proxy) {
	t.Helper()
	insecure := "www.google.ca|*.google.de"
	secure := "a|b"
	return NewProxy("id", "http", "localhost", 8888, "u", "p", &insecure),
		NewProxy("s", "https", "sec", 443, "su", "sp", &secure)
}

func TestBuildYarnArgumentsAddsRegistryAndProxy(t *testing.T) {
	insecure, secure := testProxies(t)
	config, _ := newTestInstallConfig(t)
	classic := NewInstallYarnExecutorConfig(config, false)

	got := BuildYarnArguments(classic, NewProxyConfig([]*Proxy{insecure, secure}), "www.npm.org")

	want := []string{
		"--registry=www.npm.org",
		"--https-proxy=http://u:p@localhost:8888",
		"--proxy=http://u:p@localhost:8888",
	}
	if !reflect.DeepEqual(want, got) {
		t.Errorf("arguments = %v, want %v", got, want)
	}
}

func TestBuildYarnArgumentsIsEmptyForBerry(t *testing.T) {
	insecure, _ := testProxies(t)
	config, _ := newTestInstallConfig(t)
	berry := NewInstallYarnExecutorConfig(config, true)

	got := BuildYarnArguments(berry, NewProxyConfig([]*Proxy{insecure}), "www.npm.org")

	if len(got) != 0 {
		t.Errorf("arguments = %v, want none: Yarn Berry rejects them", got)
	}
}

func TestBuildBunArguments(t *testing.T) {
	insecure, _ := testProxies(t)

	if got := BuildBunArguments(NewProxyConfig(nil), ""); len(got) != 0 {
		t.Errorf("arguments = %v, want none", got)
	}

	got := BuildBunArguments(NewProxyConfig([]*Proxy{insecure}), "www.npm.org")
	want := []string{
		"--registry=www.npm.org",
		"--https-proxy=http://u:p@localhost:8888",
		"--proxy=http://u:p@localhost:8888",
	}
	if !reflect.DeepEqual(want, got) {
		t.Errorf("arguments = %v, want %v", got, want)
	}
}

func TestBuildBowerArgumentsNamespacesItsProxyFlags(t *testing.T) {
	insecure, secure := testProxies(t)

	if got := BuildBowerArguments(NewProxyConfig(nil)); len(got) != 0 {
		t.Errorf("arguments = %v, want none", got)
	}

	got := BuildBowerArguments(NewProxyConfig([]*Proxy{insecure, secure}))
	want := []string{
		"--config.https-proxy=http://su:sp@sec:443",
		"--config.proxy=http://u:p@localhost:8888",
	}
	if !reflect.DeepEqual(want, got) {
		t.Errorf("arguments = %v, want %v", got, want)
	}

	onlyInsecure := BuildBowerArguments(NewProxyConfig([]*Proxy{insecure}))
	if !reflect.DeepEqual([]string{"--config.proxy=http://u:p@localhost:8888"}, onlyInsecure) {
		t.Errorf("arguments = %v, want only the insecure flag", onlyInsecure)
	}
}

func TestBuildPnpmArgumentsUsesOneCommaSeparatedNoproxy(t *testing.T) {
	insecure, _ := testProxies(t)

	got := BuildPnpmArguments(NewProxyConfig([]*Proxy{insecure}), "")

	want := []string{
		"--https-proxy=http://u:p@localhost:8888",
		"--proxy=http://u:p@localhost:8888",
		"--noproxy=www.google.ca,*.google.de",
	}
	if !reflect.DeepEqual(want, got) {
		t.Errorf("arguments = %v, want %v", got, want)
	}
}

func TestNpmRunnerPassesTheProxyThroughTheEnvironment(t *testing.T) {
	insecure, _ := testProxies(t)

	got := buildNpmProxyEnvironment(NewProxyConfig([]*Proxy{insecure}), "")

	want := map[string]string{
		"https_proxy": "http://u:p@localhost:8888",
		"http_proxy":  "http://u:p@localhost:8888",
	}
	if !reflect.DeepEqual(want, got) {
		t.Errorf("environment = %v, want %v", got, want)
	}
	if empty := buildNpmProxyEnvironment(NewProxyConfig(nil), ""); len(empty) != 0 {
		t.Errorf("environment = %v, want none", empty)
	}
}

// runnerTaskLocation drives a runner against a fake node that prints the script
// it was asked to run, so the location each runner reaches for is observable.
func runnerTaskLocation(t *testing.T, directory string, runner NodeTaskRunner) string {
	t.Helper()
	return captureLog(t, 0, func() { _ = runner.Execute("", nil) })
}

func TestEveryRunnerLooksForItsOwnEntryPoint(t *testing.T) {
	config, directory := newTestInstallConfig(t)
	installFakeNode(t, directory, "#!/bin/sh\necho \"$1\"\n")
	executorConfig := NewInstallNodeExecutorConfig(config)
	factory := NewFrontendPluginFactory(directory, directory)
	proxy := NewProxyConfig(nil)

	for name, expected := range map[string]string{
		"bower":   "node_modules/bower/bin/bower",
		"ember":   "node_modules/ember-cli/bin/ember",
		"grunt":   "node_modules/grunt-cli/bin/grunt",
		"gulp":    "node_modules/gulp/bin/gulp.js",
		"jspm":    "node_modules/jspm/jspm.js",
		"karma":   "node_modules/karma/bin/karma",
		"webpack": "node_modules/webpack/bin/webpack.js",
	} {
		var runner NodeTaskRunner
		switch name {
		case "bower":
			runner = NewDefaultBowerRunner(executorConfig, proxy)
		case "ember":
			runner = NewDefaultEmberRunner(executorConfig)
		case "grunt":
			runner = NewDefaultGruntRunner(executorConfig)
		case "gulp":
			runner = NewDefaultGulpRunner(executorConfig)
		case "jspm":
			runner = NewDefaultJspmRunner(executorConfig)
		case "karma":
			runner = NewDefaultKarmaRunner(executorConfig)
		case "webpack":
			runner = NewDefaultWebpackRunner(executorConfig)
		}
		output := runnerTaskLocation(t, directory, runner)
		want := filepath.Join(directory, filepath.FromSlash(expected))
		if !strings.Contains(output, want) {
			t.Errorf("%s runner did not reach for %s:\n%s", name, want, output)
		}
	}

	// The package-manager runners point at the installation, not the project.
	for name, want := range map[string]string{
		"npm":      filepath.Join(directory, "node", "node_modules", "npm", "bin", "npm-cli.js"),
		"npx":      filepath.Join(directory, "node", "node_modules", "npm", "bin", "npx-cli.js"),
		"pnpm":     filepath.Join(directory, "node", "node_modules", "pnpm", "bin", "pnpm.js"),
		"corepack": filepath.Join(directory, "node", "node_modules", "corepack", "dist", "corepack.js"),
	} {
		var runner NodeTaskRunner
		switch name {
		case "npm":
			runner = factory.NpmRunner(proxy, "")
		case "npx":
			runner = factory.NpxRunner(proxy, "")
		case "pnpm":
			runner = factory.PnpmRunner(proxy, "")
		case "corepack":
			runner = factory.CorepackRunner()
		}
		output := runnerTaskLocation(t, directory, runner)
		if !strings.Contains(output, want) {
			t.Errorf("%s runner did not reach for %s:\n%s", name, want, output)
		}
	}
}

func TestPnpmRunnerFallsBackToTheStandaloneExecutable(t *testing.T) {
	config, directory := newTestInstallConfig(t)
	installFakeNode(t, directory, "#!/bin/sh\necho \"$1\"\n")
	// Only the CommonJS entry point is installed, so the runner has to use it.
	writeFixture(t, filepath.Join(directory, "node", "node_modules", "pnpm", "bin", "pnpm.cjs"),
		[]byte("// pnpm\n"))

	runner := NewDefaultPnpmRunner(NewInstallNodeExecutorConfig(config), NewProxyConfig(nil), "")
	output := runnerTaskLocation(t, directory, runner)

	want := filepath.Join(directory, "node", "node_modules", "pnpm", "bin", "pnpm.cjs")
	if !strings.Contains(output, want) {
		t.Errorf("pnpm runner did not fall back to %s:\n%s", want, output)
	}
}

func TestYarnAndBunRunnersUseTheirOwnBinaries(t *testing.T) {
	config, directory := newTestInstallConfig(t)
	factory := NewFrontendPluginFactory(directory, directory)
	yarn := filepath.Join(directory, "node", "yarn", "dist", "bin", "yarn")
	writeFixture(t, yarn, []byte("#!/bin/sh\necho yarn-ran\n"))
	requireNoError(t, os.Chmod(yarn, 0o755))
	bun := filepath.Join(directory, "bun", "bun")
	writeFixture(t, bun, []byte("#!/bin/sh\necho bun-ran\n"))
	requireNoError(t, os.Chmod(bun, 0o755))
	_ = config

	yarnOutput := captureLog(t, 0, func() {
		requireNoError(t, factory.YarnRunner(NewProxyConfig(nil), "", false).Execute("", nil))
	})
	if !strings.Contains(yarnOutput, "yarn-ran") {
		t.Errorf("yarn runner did not run yarn:\n%s", yarnOutput)
	}

	bunOutput := captureLog(t, 0, func() {
		requireNoError(t, factory.BunRunner(NewProxyConfig(nil), "").Execute("", nil))
	})
	if !strings.Contains(bunOutput, "bun-ran") {
		t.Errorf("bun runner did not run bun:\n%s", bunOutput)
	}
}

func TestFactoryHandsOutEveryInstallerAndRunner(t *testing.T) {
	directory := t.TempDir()
	factory := NewFrontendPluginFactory(directory, directory)
	proxy := NewProxyConfig(nil)

	if factory.NodeInstaller(proxy) == nil ||
		factory.NPMInstaller(proxy) == nil ||
		factory.YarnInstaller(proxy) == nil ||
		factory.PnpmInstaller(proxy) == nil ||
		factory.CorepackInstaller(proxy) == nil ||
		factory.BunInstaller(proxy) == nil {
		t.Error("an installer was not handed out")
	}
	if factory.BowerRunner(proxy) == nil ||
		factory.BunRunner(proxy, "") == nil ||
		factory.JspmRunner() == nil ||
		factory.NpmRunner(proxy, "") == nil ||
		factory.CorepackRunner() == nil ||
		factory.PnpmRunner(proxy, "") == nil ||
		factory.NpxRunner(proxy, "") == nil ||
		factory.YarnRunner(proxy, "", false) == nil ||
		factory.GruntRunner() == nil ||
		factory.EmberRunner() == nil ||
		factory.KarmaRunner() == nil ||
		factory.GulpRunner() == nil ||
		factory.WebpackRunner() == nil {
		t.Error("a runner was not handed out")
	}
}

func TestFactoryCachesUnderTheInstallDirectoryByDefault(t *testing.T) {
	directory := t.TempDir()
	factory := NewFrontendPluginFactory(directory, directory)

	resolved := factory.installConfig().CacheResolver().
		Resolve(NewCacheDescriptor("node", "v18.0.0", "tar.gz"))

	want := filepath.Join(directory, "cache", "node-v18.0.0.tar.gz")
	if resolved != want {
		t.Errorf("cache path = %q, want %q", resolved, want)
	}
}

func TestFactoryAcceptsAnAlternativeCacheResolver(t *testing.T) {
	directory := t.TempDir()
	alternative := filepath.Join(directory, "elsewhere")
	factory := NewFrontendPluginFactoryWithCache(directory, directory,
		NewDirectoryCacheResolver(alternative))

	resolved := factory.installConfig().CacheResolver().
		Resolve(NewCacheDescriptor("npm", "8.6.0", "tar.gz"))

	want := filepath.Join(alternative, "npm-8.6.0.tar.gz")
	if resolved != want {
		t.Errorf("cache path = %q, want %q", resolved, want)
	}
}
