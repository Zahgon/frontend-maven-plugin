package frontend

import "github.com/eirslett/frontend-maven-plugin/internal/logging"

// BunExecutor runs the installed bun binary, with its own directory on PATH.
type BunExecutor struct {
	executor *ProcessExecutor
}

// NewBunExecutor prepares a bun invocation with the given arguments.
func NewBunExecutor(config BunExecutorConfig, arguments []string, additionalEnvironment map[string]string) *BunExecutor {
	bun := absolutePath(config.BunPath())
	localPaths := []string{parentPath(config.BunPath())}
	return &BunExecutor{
		executor: NewProcessExecutor(
			config.WorkingDirectory(),
			localPaths,
			Prepend(bun, arguments),
			config.Platform(),
			additionalEnvironment),
	}
}

// ExecuteAndGetResult runs bun and returns its trimmed standard output.
func (e *BunExecutor) ExecuteAndGetResult(logger *logging.Logger) (string, error) {
	return e.executor.ExecuteAndGetResult(logger)
}

// ExecuteAndRedirectOutput runs bun, logging its output, and returns its exit
// code.
func (e *BunExecutor) ExecuteAndRedirectOutput(logger *logging.Logger) (int, error) {
	return e.executor.ExecuteAndRedirectOutput(logger)
}
