package frontend

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/eirslett/frontend-maven-plugin/internal/logging"
)

// installFakeNode writes a shell script where the node binary would be, so the
// task executors can be driven end to end without a real Node.js.
func installFakeNode(t *testing.T, directory, body string) {
	t.Helper()
	path := filepath.Join(directory, "node", "node")
	writeFixture(t, path, []byte(body))
	requireNoError(t, os.Chmod(path, 0o755))
}

func TestNodeTaskExecutorRunsTheTaskAndLogsIt(t *testing.T) {
	config, directory := newTestInstallConfig(t)
	installFakeNode(t, directory, "#!/bin/sh\necho \"ran $*\"\n")
	executorConfig := NewInstallNodeExecutorConfig(config)
	writeFixture(t, filepath.Join(directory, "tool.js"), []byte("// tool\n"))

	executor := NewNodeTaskExecutorWithArguments("Test", executorConfig, "tool.js", []string{"--extra"})
	output := captureLog(t, 0, func() {
		requireNoError(t, executor.Execute("build --flag", map[string]string{"TOOL_ENV": "1"}))
	})

	if !strings.Contains(output, "Running 'tool.js build --flag --extra' in "+directory) {
		t.Errorf("log does not carry the task line:\n%s", output)
	}
	if !strings.Contains(output, "ran ") {
		t.Errorf("the task's own output was not logged:\n%s", output)
	}
}

func TestNodeTaskExecutorReportsAFailingTask(t *testing.T) {
	config, directory := newTestInstallConfig(t)
	installFakeNode(t, directory, "#!/bin/sh\nexit 5\n")
	executorConfig := NewInstallNodeExecutorConfig(config)
	writeFixture(t, filepath.Join(directory, "tool.js"), []byte("// tool\n"))

	executor := NewNodeTaskExecutor("Test", executorConfig, "tool.js")
	err := executor.Execute("build", nil)

	var taskRunner *TaskRunnerError
	if !errors.As(err, &taskRunner) {
		t.Fatalf("error = %v, want a TaskRunnerError", err)
	}
	if taskRunner.Error() != "'tool.js build' failed." {
		t.Errorf("message = %q, want %q", taskRunner.Error(), "'tool.js build' failed.")
	}
	cause := errors.Unwrap(taskRunner)
	if cause == nil || !strings.Contains(cause.Error(), "Process exited with an error: 5") {
		t.Errorf("cause = %v, want the process failure", cause)
	}
}

func TestNodeTaskExecutorFallsBackToTheInstallDirectory(t *testing.T) {
	directory := t.TempDir()
	working := filepath.Join(directory, "work")
	requireNoError(t, os.MkdirAll(working, 0o777))
	config := NewInstallConfig(directory, working,
		NewDirectoryCacheResolver(childFile(directory, "cache")), testPlatform())
	installFakeNode(t, directory, "#!/bin/sh\necho \"$1\"\n")
	// The script exists only under the install directory, not the working one.
	writeFixture(t, filepath.Join(directory, "node_modules", "tool", "tool.js"), []byte("// tool\n"))

	executor := NewNodeTaskExecutor("Test", NewInstallNodeExecutorConfig(config), "node_modules/tool/tool.js")
	output := captureLog(t, 0, func() { requireNoError(t, executor.Execute("", nil)) })

	want := filepath.Join(directory, "node_modules", "tool", "tool.js")
	if !strings.Contains(output, want) {
		t.Errorf("the task was not resolved against the install directory:\n%s", output)
	}
}

func TestNodeTaskExecutorKeepsAnAbsoluteTaskLocation(t *testing.T) {
	config, directory := newTestInstallConfig(t)
	installFakeNode(t, directory, "#!/bin/sh\necho \"$1\"\n")
	absolute := filepath.Join(directory, "elsewhere", "tool.js")
	writeFixture(t, absolute, []byte("// tool\n"))

	executor := NewNodeTaskExecutor("Test", NewInstallNodeExecutorConfig(config), absolute)
	output := captureLog(t, 0, func() { requireNoError(t, executor.Execute("", nil)) })

	if !strings.Contains(output, absolute) {
		t.Errorf("the absolute location was rewritten:\n%s", output)
	}
}

func TestNodeTaskExecutorCanBeRepointed(t *testing.T) {
	config, directory := newTestInstallConfig(t)
	installFakeNode(t, directory, "#!/bin/sh\necho \"$1\"\n")
	first := filepath.Join(directory, "first.js")
	second := filepath.Join(directory, "second.js")
	writeFixture(t, first, []byte("// first\n"))
	writeFixture(t, second, []byte("// second\n"))

	executor := NewNodeTaskExecutor("Test", NewInstallNodeExecutorConfig(config), first)
	executor.SetTaskLocation(second)
	output := captureLog(t, 0, func() { requireNoError(t, executor.Execute("", nil)) })

	if !strings.Contains(output, second) {
		t.Errorf("the task location was not changed:\n%s", output)
	}
}

func TestNodeTaskExecutorReportsAMissingNode(t *testing.T) {
	config, _ := newTestInstallConfig(t)
	executor := NewNodeTaskExecutor("Test", NewInstallNodeExecutorConfig(config), "tool.js")

	err := executor.Execute("", nil)

	var taskRunner *TaskRunnerError
	if !errors.As(err, &taskRunner) {
		t.Fatalf("error = %v, want a TaskRunnerError", err)
	}
	if taskRunner.Error() != "'tool.js ' failed." {
		t.Errorf("message = %q, want %q", taskRunner.Error(), "'tool.js ' failed.")
	}
}

func TestNodeTaskExecutorMergesProxyEnvironment(t *testing.T) {
	config, directory := newTestInstallConfig(t)
	installFakeNode(t, directory, "#!/bin/sh\necho \"$https_proxy|$OWN\"\n")
	writeFixture(t, filepath.Join(directory, "tool.js"), []byte("// tool\n"))

	executor := NewNamedNodeTaskExecutor("Test", NewInstallNodeExecutorConfig(config),
		"npm", "tool.js", nil, map[string]string{"https_proxy": "http://proxy:1"})
	output := captureLog(t, 0, func() {
		requireNoError(t, executor.Execute("", map[string]string{"OWN": "own"}))
	})

	if !strings.Contains(output, "http://proxy:1|own") {
		t.Errorf("the environments were not merged:\n%s", output)
	}
}

func TestYarnTaskExecutorRunsTheTask(t *testing.T) {
	config, directory := newTestInstallConfig(t)
	yarn := filepath.Join(directory, "node", "yarn", "dist", "bin", "yarn")
	writeFixture(t, yarn, []byte("#!/bin/sh\necho \"yarn $*\"\n"))
	requireNoError(t, os.Chmod(yarn, 0o755))

	executor := NewYarnTaskExecutor("Test", NewInstallYarnExecutorConfig(config, false), "yarn", nil)
	output := captureLog(t, 0, func() { requireNoError(t, executor.Execute("install", nil)) })

	if !strings.Contains(output, "Running 'yarn install' in "+directory) {
		t.Errorf("log does not carry the task line:\n%s", output)
	}
	if !strings.Contains(output, "yarn install") {
		t.Errorf("the task's own output was not logged:\n%s", output)
	}
}

func TestYarnTaskExecutorReportsAFailingTask(t *testing.T) {
	config, directory := newTestInstallConfig(t)
	yarn := filepath.Join(directory, "node", "yarn", "dist", "bin", "yarn")
	writeFixture(t, yarn, []byte("#!/bin/sh\nexit 2\n"))
	requireNoError(t, os.Chmod(yarn, 0o755))

	executor := NewYarnTaskExecutor("Test", NewInstallYarnExecutorConfig(config, false), "yarn", nil)
	err := executor.Execute("install", nil)

	if err == nil || err.Error() != "'yarn install' failed." {
		t.Fatalf("error = %v, want %q", err, "'yarn install' failed.")
	}
}

func TestBunTaskExecutorRunsTheTask(t *testing.T) {
	config, directory := newTestInstallConfig(t)
	bun := filepath.Join(directory, "bun", "bun")
	writeFixture(t, bun, []byte("#!/bin/sh\necho \"bun $*\"\n"))
	requireNoError(t, os.Chmod(bun, 0o755))

	executor := NewBunTaskExecutor("Test", NewInstallBunExecutorConfig(config), "bun", nil)
	output := captureLog(t, 0, func() { requireNoError(t, executor.Execute("install", nil)) })

	if !strings.Contains(output, "Running 'bun install' in "+directory) {
		t.Errorf("log does not carry the task line:\n%s", output)
	}
}

func TestBunTaskExecutorReportsAFailingTask(t *testing.T) {
	config, directory := newTestInstallConfig(t)
	bun := filepath.Join(directory, "bun", "bun")
	writeFixture(t, bun, []byte("#!/bin/sh\nexit 3\n"))
	requireNoError(t, os.Chmod(bun, 0o755))

	executor := NewBunTaskExecutor("Test", NewInstallBunExecutorConfig(config), "bun", nil)
	err := executor.Execute("install", nil)

	if err == nil || err.Error() != "'bun install' failed." {
		t.Fatalf("error = %v, want %q", err, "'bun install' failed.")
	}
}

func TestExecutorsCaptureOutput(t *testing.T) {
	config, directory := newTestInstallConfig(t)
	installFakeNode(t, directory, "#!/bin/sh\necho captured\n")
	logger := logging.GetLogger("Test")

	node, err := NewNodeExecutor(NewInstallNodeExecutorConfig(config), []string{"--version"}, nil).
		ExecuteAndGetResult(logger)
	requireNoError(t, err)
	if node != "captured" {
		t.Errorf("node output = %q, want %q", node, "captured")
	}

	yarnPath := filepath.Join(directory, "node", "yarn", "dist", "bin", "yarn")
	writeFixture(t, yarnPath, []byte("#!/bin/sh\necho yarned\n"))
	requireNoError(t, os.Chmod(yarnPath, 0o755))
	yarn, err := NewYarnExecutor(NewInstallYarnExecutorConfig(config, true), []string{"--version"}, nil).
		ExecuteAndGetResult(logger)
	requireNoError(t, err)
	if yarn != "yarned" {
		t.Errorf("yarn output = %q, want %q", yarn, "yarned")
	}

	bunPath := filepath.Join(directory, "bun", "bun")
	writeFixture(t, bunPath, []byte("#!/bin/sh\necho bunned\n"))
	requireNoError(t, os.Chmod(bunPath, 0o755))
	bun, err := NewBunExecutor(NewInstallBunExecutorConfig(config), []string{"--version"}, nil).
		ExecuteAndGetResult(logger)
	requireNoError(t, err)
	if bun != "bunned" {
		t.Errorf("bun output = %q, want %q", bun, "bunned")
	}
}

func TestExecutorsRedirectOutput(t *testing.T) {
	config, directory := newTestInstallConfig(t)
	installFakeNode(t, directory, "#!/bin/sh\necho node-line\n")
	yarnPath := filepath.Join(directory, "node", "yarn", "dist", "bin", "yarn")
	writeFixture(t, yarnPath, []byte("#!/bin/sh\necho yarn-line\n"))
	requireNoError(t, os.Chmod(yarnPath, 0o755))
	bunPath := filepath.Join(directory, "bun", "bun")
	writeFixture(t, bunPath, []byte("#!/bin/sh\necho bun-line\n"))
	requireNoError(t, os.Chmod(bunPath, 0o755))
	logger := logging.GetLogger("Test")

	output := captureLog(t, 0, func() {
		for _, run := range []func() (int, error){
			func() (int, error) {
				return NewNodeExecutor(NewInstallNodeExecutorConfig(config), nil, nil).
					ExecuteAndRedirectOutput(logger)
			},
			func() (int, error) {
				return NewYarnExecutor(NewInstallYarnExecutorConfig(config, false), nil, nil).
					ExecuteAndRedirectOutput(logger)
			},
			func() (int, error) {
				return NewBunExecutor(NewInstallBunExecutorConfig(config), nil, nil).
					ExecuteAndRedirectOutput(logger)
			},
		} {
			code, err := run()
			requireNoError(t, err)
			if code != 0 {
				t.Errorf("exit code = %d, want 0", code)
			}
		}
	})

	for _, want := range []string{"node-line", "yarn-line", "bun-line"} {
		if !strings.Contains(output, want) {
			t.Errorf("output %q was not logged:\n%s", want, output)
		}
	}
}

func TestTaskNameIsDerivedFromItsLocation(t *testing.T) {
	// The greedy prefix leaves the ".js" in place, so a task keeps the file name
	// it was launched from.
	for location, want := range map[string]string{
		"node_modules/bower/bin/bower":        "bower",
		"node_modules/grunt-cli/bin/grunt":    "grunt",
		"node_modules/gulp/bin/gulp.js":       "gulp.js",
		"node_modules/webpack/bin/webpack.js": "webpack.js",
		"plain":                               "plain",
		"a/b.js":                              "b.js",
		"/abs/path/tool.js":                   "tool.js",
		"trailing/":                           "trailing/",
	} {
		if got := taskNameFor(location); got != want {
			t.Errorf("taskNameFor(%q) = %q, want %q", location, got, want)
		}
	}
}

func TestProxyPasswordsAreMasked(t *testing.T) {
	for input, want := range map[string]string{
		"--proxy=http://user:pass@host:8080": "--proxy=http://user:***@host:8080",
		"--proxy=http://user@host:8080":      "--proxy=http://user:***@host:8080",
		"--https-proxy=https://u:p@h:1/path": "--https-proxy=https://u:***@h:1/path",
		"--proxy=http://a:b@c@d:1":           "--proxy=http://a:***@d:1",
		"--proxy=noscheme":                   "--proxy=noscheme",
		"--proxy=http://h:1":                 "--proxy=http://h:1",
		"":                                   "",
		"   ":                                "   ",
		"--proxy=ftp://u:p@h":                "--proxy=ftp://u:p@h",
		"no-proxy-here":                      "no-proxy-here",
	} {
		if got := maskPassword(input); got != want {
			t.Errorf("maskPassword(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestTaskToStringMasksAndJoins(t *testing.T) {
	if got := taskToString("npm", nil); got != "'npm '" {
		t.Errorf("taskToString with no arguments = %q, want %q", got, "'npm '")
	}
	if got := taskToString("npm", []string{"install"}); got != "'npm install'" {
		t.Errorf("taskToString = %q, want %q", got, "'npm install'")
	}
	want := "'npm install --proxy=http://user:***@host:8080'"
	got := taskToString("npm", []string{"install", "--proxy=http://user:pass@host:8080"})
	if got != want {
		t.Errorf("taskToString = %q, want %q", got, want)
	}
}
