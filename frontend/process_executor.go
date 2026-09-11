package frontend

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/eirslett/frontend-maven-plugin/internal/logging"
)

const pathEnvVar = "PATH"

// ProcessExecutor runs one front-end tool: the command line, the environment it
// sees — PATH included — and the working directory it starts in.
type ProcessExecutor struct {
	environment      map[string]string
	command          []string
	workingDirectory string
	timeout          time.Duration
}

// NewProcessExecutor prepares a process that runs without a time limit.
func NewProcessExecutor(workingDirectory string, paths, command []string, platform *Platform, additionalEnvironment map[string]string) *ProcessExecutor {
	return NewProcessExecutorWithTimeout(workingDirectory, paths, command, platform, additionalEnvironment, 0)
}

// NewProcessExecutorWithTimeout prepares a process that is killed after
// timeoutInSeconds, or runs unbounded when that is zero or negative.
func NewProcessExecutorWithTimeout(workingDirectory string, paths, command []string, platform *Platform, additionalEnvironment map[string]string, timeoutInSeconds int64) *ProcessExecutor {
	timeout := time.Duration(0)
	if timeoutInSeconds > 0 {
		timeout = time.Duration(timeoutInSeconds) * time.Second
	}
	return &ProcessExecutor{
		environment:      createEnvironment(paths, platform, additionalEnvironment),
		command:          append([]string(nil), command...),
		workingDirectory: workingDirectory,
		timeout:          timeout,
	}
}

// CommandLine renders the command the way it is logged, which is how the
// original's debug line reads.
func (e *ProcessExecutor) CommandLine() string {
	return strings.Join(e.command, " ")
}

// Environment exposes the environment the process will be started with.
func (e *ProcessExecutor) Environment() map[string]string {
	return e.environment
}

// ExecuteAndGetResult runs the process and returns its trimmed standard output.
//
// A non-zero exit never reaches the caller as a value: execute reports it as a
// failure, exactly as the original's executor does, so the version probes that
// call this see an error rather than an empty string.
func (e *ProcessExecutor) ExecuteAndGetResult(logger *logging.Logger) (string, error) {
	var stdout, stderr bytes.Buffer

	exitValue, err := e.execute(logger, &stdout, &stderr)
	if err != nil {
		return "", err
	}
	if exitValue == 0 {
		return strings.TrimSpace(stdout.String()), nil
	}
	return "", newProcessExecutionError(stdout.String()+" "+stderr.String(), nil)
}

// ExecuteAndRedirectOutput runs the process, logging standard output at INFO and
// standard error at ERROR, one line at a time, and returns its exit code.
func (e *ProcessExecutor) ExecuteAndRedirectOutput(logger *logging.Logger) (int, error) {
	stdout := newLoggerWriter(logger.Info)
	stderr := newLoggerWriter(logger.Error)
	defer stdout.Close()
	defer stderr.Close()

	return e.execute(logger, stdout, stderr)
}

func (e *ProcessExecutor) execute(logger *logging.Logger, stdout, stderr io.Writer) (int, error) {
	logger.Debug("Executing command line {}", e.CommandLine())

	ctx := context.Background()
	cancel := context.CancelFunc(func() {})
	if e.timeout > 0 {
		ctx, cancel = context.WithTimeout(ctx, e.timeout)
	}
	defer cancel()

	cmd := exec.CommandContext(ctx, e.command[0], e.command[1:]...)
	cmd.Dir = e.workingDirectory
	cmd.Env = flattenEnvironment(e.environment)
	cmd.Stdout = stdout
	cmd.Stderr = stderr

	err := cmd.Run()
	if err == nil {
		logger.Debug("Exit value {}", 0)
		return 0, nil
	}

	var exitError *exec.ExitError
	if errors.As(err, &exitError) {
		if e.timeout > 0 && ctx.Err() != nil {
			return 0, newProcessExecutionError("Process killed after timeout", nil)
		}
		// A non-zero exit is a failure, not a value. The original's executor
		// raises here rather than returning the code, which is why a failing
		// task reports "'npm install' failed." and never "(error code N)".
		exitValue := exitError.ExitCode()
		return 0, newProcessExecutionError("", &processExitError{exitValue: exitValue})
	}
	return 0, newProcessExecutionError("", err)
}

// processExitError is the failure a non-zero exit produces. Its message is the
// original's, minus the Java exception class name its rendering carried.
type processExitError struct {
	exitValue int
}

func (e *processExitError) Error() string {
	return fmt.Sprintf("Process exited with an error: %d (Exit value: %d)", e.exitValue, e.exitValue)
}

// ExitValue reports the status the process exited with.
func (e *processExitError) ExitValue() int {
	return e.exitValue
}

// createEnvironment builds the child environment: the parent's, then the
// caller's additions, then a PATH with the tool directories in front.
func createEnvironment(paths []string, platform *Platform, additionalEnvironment map[string]string) map[string]string {
	environment := make(map[string]string)
	for _, entry := range os.Environ() {
		if key, value, ok := strings.Cut(entry, "="); ok {
			environment[key] = value
		}
	}
	for key, value := range additionalEnvironment {
		environment[key] = value
	}

	if platform.IsWindows() {
		// Windows environment names are case-insensitive, so the inherited PATH
		// can arrive under any spelling and has to be found by folded compare.
		for pathName, pathValue := range environment {
			if strings.EqualFold(pathEnvVar, pathName) {
				environment[pathName] = extendPathVariable(pathValue, paths)
			}
		}
	} else {
		environment[pathEnvVar] = extendPathVariable(environment[pathEnvVar], paths)
	}

	return environment
}

// extendPathVariable prepends the tool directories to an existing PATH. Every
// element, the inherited value included, is followed by a separator, so the
// result ends in one.
func extendPathVariable(existingValue string, paths []string) string {
	var b strings.Builder
	for _, path := range paths {
		b.WriteString(path)
		b.WriteString(string(os.PathListSeparator))
	}
	if existingValue != "" {
		b.WriteString(existingValue)
		b.WriteString(string(os.PathListSeparator))
	}
	return b.String()
}

func flattenEnvironment(environment map[string]string) []string {
	flattened := make([]string, 0, len(environment))
	for key, value := range environment {
		flattened = append(flattened, key+"="+value)
	}
	return flattened
}

// loggerWriter turns a byte stream into whole lines on a logging function, the
// way commons-exec's LogOutputStream does. A trailing fragment with no newline
// is flushed on Close, which is why the executor closes both writers.
type loggerWriter struct {
	mu     sync.Mutex
	log    func(string, ...any)
	buffer bytes.Buffer
}

func newLoggerWriter(log func(string, ...any)) *loggerWriter {
	return &loggerWriter{log: log}
}

func (w *loggerWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.buffer.Write(p)
	for {
		line, err := w.buffer.ReadString('\n')
		if err != nil {
			// Not a whole line yet: ReadString consumed the fragment, so put it
			// back and wait for the rest of it.
			w.buffer.WriteString(line)
			break
		}
		w.log("{}", strings.TrimRight(line, "\r\n"))
	}
	return len(p), nil
}

// Close flushes any line that never got its newline.
func (w *loggerWriter) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.buffer.Len() > 0 {
		scanner := bufio.NewScanner(&w.buffer)
		for scanner.Scan() {
			w.log("{}", scanner.Text())
		}
		w.buffer.Reset()
	}
	return nil
}
