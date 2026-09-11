package goal

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/eirslett/frontend-maven-plugin/frontend"
)

// deadServer answers every request with 404, which is enough to drive an install
// goal all the way into its installer and back out through the error mapping.
func deadServer(t *testing.T) string {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(server.Close)
	return server.URL
}

// installFakeToolchain lays down a node that echoes its arguments plus the entry
// points the task goals reach for, so a goal can be run for real.
func installFakeToolchain(t *testing.T, directory string) {
	t.Helper()
	node := filepath.Join(directory, "node", "node")
	writeFile(t, node, "#!/bin/sh\necho \"ran $*\"\n")
	requireNoError(t, os.Chmod(node, 0o755))
	for _, entry := range []string{
		"node/node_modules/npm/bin/npm-cli.js",
		"node/node_modules/npm/bin/npx-cli.js",
		"node/node_modules/pnpm/bin/pnpm.js",
		"node/node_modules/corepack/dist/corepack.js",
		"node_modules/bower/bin/bower",
		"node_modules/ember-cli/bin/ember",
		"node_modules/grunt-cli/bin/grunt",
		"node_modules/gulp/bin/gulp.js",
		"node_modules/jspm/jspm.js",
		"node_modules/karma/bin/karma",
		"node_modules/webpack/bin/webpack.js",
	} {
		writeFile(t, filepath.Join(directory, filepath.FromSlash(entry)), "// entry\n")
	}
	for _, tool := range []struct{ path, body string }{
		{"node/yarn/dist/bin/yarn", "#!/bin/sh\necho \"ran yarn $*\"\n"},
		{"bun/bun", "#!/bin/sh\necho \"ran bun $*\"\n"},
	} {
		path := filepath.Join(directory, filepath.FromSlash(tool.path))
		writeFile(t, path, tool.body)
		requireNoError(t, os.Chmod(path, 0o755))
	}
}

func newBase(t *testing.T) (Base, string) {
	t.Helper()
	session, directory := newTestSession(t)
	return Base{WorkingDirectory: directory, Session: session}, directory
}

func TestExecuteSkipsWhenTheGoalsOwnFlagIsSet(t *testing.T) {
	base, _ := newBase(t)
	goal := &Npm{Base: base, Skip: true}

	output := captureLog(t, func() { requireNoError(t, Execute(goal)) })

	if !strings.Contains(output, "Skipping execution.") {
		t.Errorf("log does not report the skip:\n%s", output)
	}
}

func TestExecuteSkipsTestsInATestingPhase(t *testing.T) {
	base, _ := newBase(t)
	base.SkipTests = true
	base.Session.LifecyclePhase = "test"
	goal := &Karma{Base: base}

	output := captureLog(t, func() { requireNoError(t, Execute(goal)) })

	if !strings.Contains(output, "Skipping execution.") {
		t.Errorf("log does not report the skip:\n%s", output)
	}
}

func TestExecuteRunsWhenSkipTestsAppliesOutsideATestingPhase(t *testing.T) {
	base, directory := newBase(t)
	installFakeToolchain(t, directory)
	base.SkipTests = true
	base.Session.LifecyclePhase = "generate-resources"
	goal := &Jspm{Base: base, Arguments: "install"}

	output := captureLog(t, func() { requireNoError(t, Execute(goal)) })

	if strings.Contains(output, "Skipping execution.") {
		t.Errorf("the goal was skipped outside a testing phase:\n%s", output)
	}
}

func TestExecuteNotesAnIgnoredTestFailureFlagOutsideATestingPhase(t *testing.T) {
	base, directory := newBase(t)
	installFakeToolchain(t, directory)
	base.TestFailureIgnore = true
	goal := &Jspm{Base: base, Arguments: "install"}

	output := captureLog(t, func() { requireNoError(t, Execute(goal)) })

	if !strings.Contains(output, "testFailureIgnore property is ignored in non test phases") {
		t.Errorf("log does not carry the note:\n%s", output)
	}
}

func TestExecuteDowngradesATaskFailureInATestingPhase(t *testing.T) {
	base, directory := newBase(t)
	installFakeToolchain(t, directory)
	node := filepath.Join(directory, "node", "node")
	writeFile(t, node, "#!/bin/sh\nexit 1\n")
	requireNoError(t, os.Chmod(node, 0o755))
	base.TestFailureIgnore = true
	base.Session.LifecyclePhase = "test"
	goal := &Karma{Base: base, KarmaConfPath: "karma.conf.js"}

	output := captureLog(t, func() { requireNoError(t, Execute(goal)) })

	if !strings.Contains(output, "There are test failures.") {
		t.Errorf("log does not report the tolerated failure:\n%s", output)
	}
}

func TestExecuteReportsATaskFailureAsABuildFailure(t *testing.T) {
	base, directory := newBase(t)
	installFakeToolchain(t, directory)
	node := filepath.Join(directory, "node", "node")
	writeFile(t, node, "#!/bin/sh\nexit 1\n")
	requireNoError(t, os.Chmod(node, 0o755))
	goal := &Karma{Base: base, KarmaConfPath: "karma.conf.js"}

	err := Execute(goal)

	var failure *FailureError
	if !errors.As(err, &failure) {
		t.Fatalf("error = %v, want a FailureError", err)
	}
	if failure.Error() != "Failed to run task" {
		t.Errorf("message = %q, want %q", failure.Error(), "Failed to run task")
	}
	var taskRunner *frontend.TaskRunnerError
	if !errors.As(err, &taskRunner) {
		t.Errorf("the task failure is not reachable through the chain: %v", err)
	}
}

func TestExecuteReportsAnInstallationFailureWithItsCause(t *testing.T) {
	base, _ := newBase(t)
	goal := &InstallNodeAndNpm{
		Base:             base,
		NodeVersion:      "v18.20.4",
		NpmVersion:       "8.6.0",
		NodeDownloadRoot: deadServer(t) + "/node/",
		NpmDownloadRoot:  frontend.DefaultNpmDownloadRoot,
	}

	err := Execute(goal)

	want := "Could not download Node.js: Got error code 404 from the server."
	if err == nil || err.Error() != want {
		t.Fatalf("error = %v, want %q", err, want)
	}
}

func TestExecuteFallsBackToTheWorkingDirectoryForInstalls(t *testing.T) {
	base, directory := newBase(t)
	installFakeToolchain(t, directory)
	goal := &Jspm{Base: base, Arguments: "install"}

	requireNoError(t, Execute(goal))

	if goal.InstallDirectory != directory {
		t.Errorf("install directory = %q, want the working directory %q",
			goal.InstallDirectory, directory)
	}
}

func TestExecuteCachesIntoTheLocalRepositoryWhenOneIsKnown(t *testing.T) {
	base, _ := newBase(t)
	base.InstallDirectory = base.WorkingDirectory

	resolver := base.cacheResolver()

	resolved := resolver.Resolve(frontend.NewCacheDescriptor("node", "v18.0.0", "tar.gz"))
	if !strings.Contains(resolved, filepath.Join("com", "github", "eirslett")) {
		t.Errorf("cache path = %q, want it in the local repository", resolved)
	}

	base.Session.LocalRepository = ""
	resolved = base.cacheResolver().Resolve(frontend.NewCacheDescriptor("node", "v18.0.0", "tar.gz"))
	if !strings.HasSuffix(resolved, filepath.Join("cache", "node-v18.0.0.tar.gz")) {
		t.Errorf("cache path = %q, want the per-install cache", resolved)
	}
}

func TestPackageManagerGoalsRunTheirTool(t *testing.T) {
	for _, testCase := range []struct {
		name string
		goal func(Base) Goal
		want string
	}{
		{"npm", func(b Base) Goal { return &Npm{Base: b, Arguments: "install"} }, "npm-cli.js install"},
		{"npx", func(b Base) Goal { return &Npx{Base: b, Arguments: "run x"} }, "npx-cli.js run x"},
		{"pnpm", func(b Base) Goal { return &Pnpm{Base: b, Arguments: "install"} }, "pnpm.js install"},
		{"corepack", func(b Base) Goal { return &Corepack{Base: b, Arguments: "enable"} }, "corepack.js enable"},
		{"yarn", func(b Base) Goal { return &Yarn{Base: b, Arguments: "install"} }, "ran yarn install"},
		{"bun", func(b Base) Goal { return &Bun{Base: b, Arguments: "install"} }, "ran bun install"},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			base, directory := newBase(t)
			installFakeToolchain(t, directory)
			goal := testCase.goal(base)

			output := captureLog(t, func() { requireNoError(t, Execute(goal)) })

			if !strings.Contains(output, testCase.want) {
				t.Errorf("%s did not run its tool (%q):\n%s", testCase.name, testCase.want, output)
			}
			if goal.Name() != testCase.name {
				t.Errorf("goal name = %q, want %q", goal.Name(), testCase.name)
			}
			if goal.SkipExecution() {
				t.Error("the goal reported itself as skipped")
			}
			if goal.Params() == nil {
				t.Error("the goal exposes no shared parameters")
			}
		})
	}
}

func TestPackageManagerGoalsStandDownWhenPackageJsonIsUnchanged(t *testing.T) {
	for _, testCase := range []struct {
		goal func(Base) Goal
		want string
	}{
		{func(b Base) Goal { return &Npm{Base: b} }, "Skipping npm install as package.json unchanged"},
		{func(b Base) Goal { return &Npx{Base: b} }, "Skipping npm install as package.json unchanged"},
		{func(b Base) Goal { return &Pnpm{Base: b} }, "Skipping pnpm install as package.json unchanged"},
		{func(b Base) Goal { return &Yarn{Base: b} }, "Skipping yarn install as package.json unchanged"},
		{func(b Base) Goal { return &Bun{Base: b} }, "Skipping bun install as package.json unchanged"},
		{func(b Base) Goal { return &Corepack{Base: b} }, "Skipping corepack install as package.json unchanged"},
	} {
		base, _ := newBase(t)
		base.Session.BuildContext = &stubBuildContext{incremental: true, changed: map[string]bool{}}

		output := captureLog(t, func() { requireNoError(t, Execute(testCase.goal(base))) })

		if !strings.Contains(output, testCase.want) {
			t.Errorf("log does not carry %q:\n%s", testCase.want, output)
		}
	}
}

func TestPackageManagerGoalsCanRefuseTheBuildsProxies(t *testing.T) {
	for _, testCase := range []struct {
		goal func(Base) Goal
		want string
	}{
		{func(b Base) Goal { return &Npm{Base: b} }, "npm not inheriting proxy config from Maven"},
		{func(b Base) Goal { return &Pnpm{Base: b} }, "pnpm not inheriting proxy config from Maven"},
		{func(b Base) Goal { return &Yarn{Base: b} }, "yarn not inheriting proxy config from Maven"},
		{func(b Base) Goal { return &Bun{Base: b} }, "bun not inheriting proxy config from Maven"},
		{func(b Base) Goal { return &Bower{Base: b} }, "bower not inheriting proxy config from Maven"},
	} {
		base, directory := newBase(t)
		installFakeToolchain(t, directory)

		output := captureLog(t, func() { _ = Execute(testCase.goal(base)) })

		if !strings.Contains(output, testCase.want) {
			t.Errorf("log does not carry %q:\n%s", testCase.want, output)
		}
	}
}

func TestRegistryUrlCanBeOverriddenAtRunTime(t *testing.T) {
	base, directory := newBase(t)
	installFakeToolchain(t, directory)
	base.Session.SystemProperties["npmRegistryURL"] = "http://override"

	for _, run := range []func() string{
		func() string { return (&Npm{Base: base, NpmRegistryURL: "http://configured"}).registryURL() },
		func() string { return (&Npx{Base: base, NpmRegistryURL: "http://configured"}).registryURL() },
		func() string { return (&Pnpm{Base: base, PnpmRegistryURL: "http://configured"}).registryURL() },
		func() string { return (&Yarn{Base: base, NpmRegistryURL: "http://configured"}).registryURL() },
		func() string { return (&Bun{Base: base, NpmRegistryURL: "http://configured"}).registryURL() },
	} {
		if got := run(); got != "http://override" {
			t.Errorf("registry URL = %q, want the override", got)
		}
	}
}

func TestWatchedGoalsRunAndRefreshTheirOutput(t *testing.T) {
	for _, testCase := range []struct {
		name    string
		goal    func(Base, watched) Goal
		trigger string
	}{
		{"grunt", func(b Base, w watched) Goal { return &Grunt{Base: b, watched: w} }, "Gruntfile.js"},
		{"gulp", func(b Base, w watched) Goal { return &Gulp{Base: b, watched: w} }, "gulpfile.js"},
		{"ember", func(b Base, w watched) Goal { return &Ember{Base: b, watched: w} }, "Gruntfile.js"},
		{"webpack", func(b Base, w watched) Goal { return &Webpack{Base: b, watched: w} }, "webpack.config.js"},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			base, directory := newBase(t)
			installFakeToolchain(t, directory)
			outputdir := filepath.Join(directory, "dist")
			context := &stubBuildContext{incremental: false}
			base.Session.BuildContext = context
			goal := testCase.goal(base, watched{Arguments: "build", Outputdir: outputdir})

			output := captureLog(t, func() { requireNoError(t, Execute(goal)) })

			if !strings.Contains(output, "Refreshing files after "+testCase.name+": "+outputdir) {
				t.Errorf("log does not report the refresh:\n%s", output)
			}
			if len(context.refreshed) != 1 || context.refreshed[0] != outputdir {
				t.Errorf("refreshed = %v, want the output directory", context.refreshed)
			}
			if goal.Name() != testCase.name {
				t.Errorf("goal name = %q, want %q", goal.Name(), testCase.name)
			}
			// The default trigger file is the tool's own configuration file.
			if got := goal.Params(); got == nil {
				t.Error("the goal exposes no shared parameters")
			}
		})
	}
}

func TestWatchedGoalsDefaultTheirTriggerFile(t *testing.T) {
	for _, testCase := range []struct {
		goal    func(Base) (Goal, *watched)
		trigger string
	}{
		{func(b Base) (Goal, *watched) { g := &Grunt{Base: b}; return g, &g.watched }, "Gruntfile.js"},
		{func(b Base) (Goal, *watched) { g := &Gulp{Base: b}; return g, &g.watched }, "gulpfile.js"},
		{func(b Base) (Goal, *watched) { g := &Ember{Base: b}; return g, &g.watched }, "Gruntfile.js"},
		{func(b Base) (Goal, *watched) { g := &Webpack{Base: b}; return g, &g.watched }, "webpack.config.js"},
	} {
		base, directory := newBase(t)
		installFakeToolchain(t, directory)
		goal, w := testCase.goal(base)

		requireNoError(t, Execute(goal))

		want := filepath.Join(directory, testCase.trigger)
		if len(w.Triggerfiles) != 1 || w.Triggerfiles[0] != want {
			t.Errorf("trigger files = %v, want [%s]", w.Triggerfiles, want)
		}
	}
}

func TestWatchedGoalsStandDownWhenNothingChanged(t *testing.T) {
	base, directory := newBase(t)
	installFakeToolchain(t, directory)
	srcdir := filepath.Join(directory, "src")
	base.Session.BuildContext = &stubBuildContext{incremental: true, changed: map[string]bool{}}
	goal := &Gulp{Base: base, watched: watched{Srcdir: srcdir}}

	output := captureLog(t, func() { requireNoError(t, Execute(goal)) })

	if !strings.Contains(output, "Skipping gulp as no modified files in "+srcdir) {
		t.Errorf("log does not report the skip:\n%s", output)
	}
}

func TestWatchedGoalsSkipFlags(t *testing.T) {
	base, _ := newBase(t)
	for _, goal := range []Goal{
		&Grunt{Base: base, watched: watched{Skip: true}},
		&Gulp{Base: base, watched: watched{Skip: true}},
		&Ember{Base: base, watched: watched{Skip: true}},
		&Webpack{Base: base, watched: watched{Skip: true}},
	} {
		if !goal.SkipExecution() {
			t.Errorf("%s did not report itself as skipped", goal.Name())
		}
	}
	for _, goal := range []Goal{
		&Bower{Base: base, Skip: true},
		&Jspm{Base: base, Skip: true},
		&Karma{Base: base, Skip: true},
		&Npm{Base: base, Skip: true},
		&Npx{Base: base, Skip: true},
		&Pnpm{Base: base, Skip: true},
		&Yarn{Base: base, Skip: true},
		&Bun{Base: base, Skip: true},
		&Corepack{Base: base, Skip: true},
		&InstallNodeAndNpm{Base: base, Skip: true},
		&InstallNodeAndYarn{Base: base, Skip: true},
		&InstallNodeAndPnpm{Base: base, Skip: true},
		&InstallNodeAndCorepack{Base: base, Skip: true},
		&InstallBun{Base: base, Skip: true},
	} {
		if !goal.SkipExecution() {
			t.Errorf("%s did not report itself as skipped", goal.Name())
		}
		if goal.Params() == nil {
			t.Errorf("%s exposes no shared parameters", goal.Name())
		}
	}
}

func TestTaskGoalsRunTheirTool(t *testing.T) {
	for _, testCase := range []struct {
		name string
		goal func(Base) Goal
		want string
	}{
		{"bower", func(b Base) Goal { return &Bower{Base: b, Arguments: "install"} }, "bin/bower install"},
		{"jspm", func(b Base) Goal { return &Jspm{Base: b, Arguments: "install"} }, "jspm.js install"},
		{"karma", func(b Base) Goal { return &Karma{Base: b, KarmaConfPath: "karma.conf.js"} },
			"bin/karma start karma.conf.js"},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			base, directory := newBase(t)
			installFakeToolchain(t, directory)
			goal := testCase.goal(base)

			output := captureLog(t, func() { requireNoError(t, Execute(goal)) })

			if !strings.Contains(output, testCase.want) {
				t.Errorf("%s did not run its tool (%q):\n%s", testCase.name, testCase.want, output)
			}
			if goal.Name() != testCase.name {
				t.Errorf("goal name = %q, want %q", goal.Name(), testCase.name)
			}
		})
	}
}

func TestInstallGoalsReachTheirInstallers(t *testing.T) {
	server := deadServer(t)
	for _, testCase := range []struct {
		name string
		goal func(Base) Goal
		want string
	}{
		{"install-node-and-npm", func(b Base) Goal {
			return &InstallNodeAndNpm{Base: b, NodeVersion: "v18.20.4", NpmVersion: "8.6.0",
				NodeDownloadRoot: server + "/node/", NpmDownloadRoot: frontend.DefaultNpmDownloadRoot}
		}, "Could not download Node.js"},
		{"install-node-and-yarn", func(b Base) Goal {
			return &InstallNodeAndYarn{Base: b, NodeVersion: "v18.20.4", YarnVersion: "v1.22.19",
				NodeDownloadRoot: server + "/node/", YarnDownloadRoot: server + "/yarn/"}
		}, "Could not download Node.js"},
		{"install-node-and-pnpm", func(b Base) Goal {
			return &InstallNodeAndPnpm{Base: b, NodeVersion: "v18.20.4", PnpmVersion: "7.9.0",
				NodeDownloadRoot: server + "/node/", PnpmDownloadRoot: frontend.DefaultPnpmDownloadRoot}
		}, "Could not download Node.js"},
		{"install-node-and-corepack", func(b Base) Goal {
			return &InstallNodeAndCorepack{Base: b, NodeVersion: "v18.20.4", CorepackVersion: "provided",
				NodeDownloadRoot: server + "/node/", CorepackDownloadRoot: frontend.DefaultCorepackDownloadRoot}
		}, "Could not download Node.js"},
		{"install-bun", func(b Base) Goal {
			return &InstallBun{Base: b, BunVersion: "v1.0.0", BunDownloadRoot: server + "/bun/"}
		}, "Could not download bun"},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			base, _ := newBase(t)
			goal := testCase.goal(base)

			err := Execute(goal)

			if err == nil || !strings.HasPrefix(err.Error(), testCase.want) {
				t.Fatalf("error = %v, want it to start with %q", err, testCase.want)
			}
			if goal.Name() != testCase.name {
				t.Errorf("goal name = %q, want %q", goal.Name(), testCase.name)
			}
		})
	}
}

func TestInstallGoalsSendServerCredentials(t *testing.T) {
	var gotUser string
	var gotHeader string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUser, _, _ = r.BasicAuth()
		gotHeader = r.Header.Get("X-Probe")
		w.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(server.Close)

	base, _ := newBase(t)
	base.Session.Settings = &Settings{Servers: []Server{{
		ID: "downloads", Username: "server-user", Password: "server-password",
		Configuration: Configuration{HTTPHeaders: &HTTPHeaders{
			Properties: []HeaderProperty{{Name: "X-Probe", Value: "yes"}},
		}},
	}}}

	_ = Execute(&InstallNodeAndNpm{
		Base: base, NodeVersion: "v18.20.4", NpmVersion: "8.6.0", ServerID: "downloads",
		NodeDownloadRoot: server.URL + "/node/", NpmDownloadRoot: frontend.DefaultNpmDownloadRoot,
	})

	if gotUser != "server-user" {
		t.Errorf("credentials = %q, want the server's", gotUser)
	}
	if gotHeader != "yes" {
		t.Errorf("header = %q, want the server's", gotHeader)
	}
}

func TestDeprecatedDownloadRootIsOnlyAFallback(t *testing.T) {
	npm := &InstallNodeAndNpm{DownloadRoot: "http://legacy/", NpmDownloadRoot: frontend.DefaultNpmDownloadRoot}
	if got := npm.nodeDownloadRoot(); got != "http://legacy/" {
		t.Errorf("node download root = %q, want the deprecated one", got)
	}
	if got := npm.npmDownloadRoot(); got != "http://legacy/" {
		t.Errorf("npm download root = %q, want the deprecated one", got)
	}
	npm.NodeDownloadRoot = "http://explicit/"
	npm.NpmDownloadRoot = "http://explicit-npm/"
	if got := npm.nodeDownloadRoot(); got != "http://explicit/" {
		t.Errorf("node download root = %q, want the explicit one", got)
	}
	if got := npm.npmDownloadRoot(); got != "http://explicit-npm/" {
		t.Errorf("npm download root = %q, want the explicit one", got)
	}

	pnpm := &InstallNodeAndPnpm{DownloadRoot: "http://legacy/", PnpmDownloadRoot: frontend.DefaultPnpmDownloadRoot}
	if got := pnpm.nodeDownloadRoot(); got != "http://legacy/" {
		t.Errorf("node download root = %q, want the deprecated one", got)
	}
	if got := pnpm.pnpmDownloadRoot(); got != "http://legacy/" {
		t.Errorf("pnpm download root = %q, want the deprecated one", got)
	}
	pnpm.NodeDownloadRoot = "http://explicit/"
	pnpm.PnpmDownloadRoot = "http://explicit-pnpm/"
	if got := pnpm.nodeDownloadRoot(); got != "http://explicit/" {
		t.Errorf("node download root = %q, want the explicit one", got)
	}
	if got := pnpm.pnpmDownloadRoot(); got != "http://explicit-pnpm/" {
		t.Errorf("pnpm download root = %q, want the explicit one", got)
	}
}

func TestFailureErrorCarriesItsCause(t *testing.T) {
	cause := errors.New("underlying")
	failure := &FailureError{Message: "outer", Cause: cause}

	if failure.Error() != "outer" {
		t.Errorf("message = %q, want %q", failure.Error(), "outer")
	}
	if !errors.Is(failure, cause) {
		t.Error("the cause is not reachable through the chain")
	}
}

func TestToFailureAppendsTheCauseMessage(t *testing.T) {
	wrapped := &frontend.InstallationError{
		FrontendError: frontend.FrontendError{Message: "outer", Cause: errors.New("inner")},
	}

	if got := toFailure(wrapped).Error(); got != "outer: inner" {
		t.Errorf("message = %q, want %q", got, "outer: inner")
	}
	if got := toFailure(errors.New("alone")).Error(); got != "alone" {
		t.Errorf("message = %q, want %q", got, "alone")
	}
}
