package frontend

import (
	"os"
	"strings"
	"testing"

	"github.com/eirslett/frontend-maven-plugin/internal/logging"
)

// The process layer replaces commons-exec, whose observable behaviour — how PATH
// is assembled, what a non-zero exit does, and what a timeout says — the plugin
// depends on.

func TestProcessExecutorPrependsToolDirectoriesToPath(t *testing.T) {
	executor := NewProcessExecutor(t.TempDir(), []string{"/first/bin", "/second/bin"},
		[]string{"/bin/sh", "-c", `printf %s "$PATH"`}, testPlatform(), nil)

	result, err := executor.ExecuteAndGetResult(logging.GetLogger("Test"))

	requireNoError(t, err)
	want := "/first/bin:/second/bin:" + os.Getenv("PATH") + ":"
	if result != strings.TrimSpace(want) {
		t.Errorf("PATH = %q, want %q", result, strings.TrimSpace(want))
	}
}

func TestProcessExecutorKeepsATrailingSeparatorOnPath(t *testing.T) {
	// Every element, the inherited value included, is followed by a separator.
	environment := NewProcessExecutor(t.TempDir(), []string{"/only/bin"},
		[]string{"/bin/sh"}, testPlatform(), nil).Environment()

	if !strings.HasSuffix(environment["PATH"], ":") {
		t.Errorf("PATH = %q, want it to end with a separator", environment["PATH"])
	}
	if !strings.HasPrefix(environment["PATH"], "/only/bin:") {
		t.Errorf("PATH = %q, want it to start with the tool directory", environment["PATH"])
	}
}

func TestProcessExecutorFindsPathCaseInsensitivelyOnWindows(t *testing.T) {
	windows := GuessPlatformWith(OSWindows, ArchX64, func() bool { return false })

	environment := createEnvironment([]string{"C:\\tools"}, windows, map[string]string{"Path": "C:\\existing"})

	if !strings.HasPrefix(environment["Path"], "C:\\tools") {
		t.Errorf("Path = %q, want the tool directory in front", environment["Path"])
	}
}

func TestProcessExecutorAddsTheCallersEnvironment(t *testing.T) {
	executor := NewProcessExecutor(t.TempDir(), nil,
		[]string{"/bin/sh", "-c", `printf %s "$PROBE"`}, testPlatform(),
		map[string]string{"PROBE": "value"})

	result, err := executor.ExecuteAndGetResult(logging.GetLogger("Test"))

	requireNoError(t, err)
	if result != "value" {
		t.Errorf("output = %q, want %q", result, "value")
	}
}

func TestProcessExecutorTrimsCapturedOutput(t *testing.T) {
	executor := NewProcessExecutor(t.TempDir(), nil,
		[]string{"/bin/sh", "-c", `printf ' spaced \n'`}, testPlatform(), nil)

	result, err := executor.ExecuteAndGetResult(logging.GetLogger("Test"))

	requireNoError(t, err)
	if result != "spaced" {
		t.Errorf("output = %q, want %q", result, "spaced")
	}
}

func TestProcessExecutorTreatsANonZeroExitAsAFailure(t *testing.T) {
	executor := NewProcessExecutor(t.TempDir(), nil,
		[]string{"/bin/sh", "-c", "printf out; printf err >&2; exit 3"}, testPlatform(), nil)

	_, err := executor.ExecuteAndGetResult(logging.GetLogger("Test"))

	if err == nil {
		t.Fatal("a non-zero exit must be reported as a failure")
	}
	if err.Error() != "Process exited with an error: 3 (Exit value: 3)" {
		t.Errorf("message = %q, want the exit-value message", err.Error())
	}
}

func TestProcessExecutorRedirectsOutputToTheLog(t *testing.T) {
	executor := NewProcessExecutor(t.TempDir(), nil,
		[]string{"/bin/sh", "-c", "echo to-stdout; echo to-stderr >&2; printf no-newline"},
		testPlatform(), nil)

	var code int
	output := captureLog(t, 0, func() {
		var err error
		code, err = executor.ExecuteAndRedirectOutput(logging.GetLogger("Test"))
		requireNoError(t, err)
	})

	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}
	if !strings.Contains(output, "[INFO] to-stdout") {
		t.Errorf("stdout was not logged at INFO:\n%s", output)
	}
	if !strings.Contains(output, "[ERROR] to-stderr") {
		t.Errorf("stderr was not logged at ERROR:\n%s", output)
	}
	if !strings.Contains(output, "no-newline") {
		t.Errorf("a trailing fragment was not flushed:\n%s", output)
	}
}

func TestProcessExecutorReportsAMissingBinary(t *testing.T) {
	executor := NewProcessExecutor(t.TempDir(), nil,
		[]string{"/definitely/not/a/binary"}, testPlatform(), nil)

	_, err := executor.ExecuteAndGetResult(logging.GetLogger("Test"))

	if err == nil {
		t.Fatal("a missing binary must be reported as a failure")
	}
}

func TestProcessExecutorKillsAProcessThatOverrunsItsTimeout(t *testing.T) {
	executor := NewProcessExecutorWithTimeout(t.TempDir(), nil,
		[]string{"/bin/sh", "-c", "sleep 5"}, testPlatform(), nil, 1)

	_, err := executor.ExecuteAndGetResult(logging.GetLogger("Test"))

	if err == nil || err.Error() != "Process killed after timeout" {
		t.Fatalf("error = %v, want %q", err, "Process killed after timeout")
	}
}

func TestProcessExecutorRunsWithoutATimeoutWhenNoneIsSet(t *testing.T) {
	// A zero timeout must mean no watchdog at all, not a zero-length one: the
	// process has to outlive the interval a one-second limit would have killed
	// it at, and still deliver its output.
	executor := NewProcessExecutorWithTimeout(t.TempDir(), nil,
		[]string{"/bin/sh", "-c", "sleep 2; printf survived"}, testPlatform(), nil, 0)

	result, err := executor.ExecuteAndGetResult(logging.GetLogger("Test"))

	requireNoError(t, err)
	if result != "survived" {
		t.Errorf("output = %q, want %q", result, "survived")
	}
}

func TestProcessExecutorLogsTheCommandLine(t *testing.T) {
	executor := NewProcessExecutor(t.TempDir(), nil,
		[]string{"/bin/sh", "-c", "exit 0"}, testPlatform(), nil)

	if got := executor.CommandLine(); got != "/bin/sh -c exit 0" {
		t.Errorf("command line = %q, want %q", got, "/bin/sh -c exit 0")
	}
	output := captureLog(t, logging.Debug, func() {
		_, err := executor.ExecuteAndGetResult(logging.GetLogger("Test"))
		requireNoError(t, err)
	})
	if !strings.Contains(output, "Executing command line /bin/sh -c exit 0") {
		t.Errorf("the command line was not logged at DEBUG:\n%s", output)
	}
}

func TestProcessExitErrorCarriesItsStatus(t *testing.T) {
	err := &processExitError{exitValue: 9}

	if err.ExitValue() != 9 {
		t.Errorf("exit value = %d, want 9", err.ExitValue())
	}
	if err.Error() != "Process exited with an error: 9 (Exit value: 9)" {
		t.Errorf("message = %q, want the exit-value message", err.Error())
	}
}
