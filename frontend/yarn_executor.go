package frontend

import "github.com/eirslett/frontend-maven-plugin/internal/logging"

// YarnExecutor runs the installed yarn launcher, with both its own directory and
// node's on PATH — yarn is a script that has to be able to find node.
type YarnExecutor struct {
	executor *ProcessExecutor
}

// NewYarnExecutor prepares a yarn invocation with the given arguments.
func NewYarnExecutor(config YarnExecutorConfig, arguments []string, additionalEnvironment map[string]string) *YarnExecutor {
	yarn := absolutePath(config.YarnPath())
	localPaths := []string{
		parentPath(config.YarnPath()),
		parentPath(config.NodePath()),
	}
	return &YarnExecutor{
		executor: NewProcessExecutor(
			config.WorkingDirectory(),
			localPaths,
			Prepend(yarn, arguments),
			config.Platform(),
			additionalEnvironment),
	}
}

// ExecuteAndGetResult runs yarn and returns its trimmed standard output.
func (e *YarnExecutor) ExecuteAndGetResult(logger *logging.Logger) (string, error) {
	return e.executor.ExecuteAndGetResult(logger)
}

// ExecuteAndRedirectOutput runs yarn, logging its output, and returns its exit
// code.
func (e *YarnExecutor) ExecuteAndRedirectOutput(logger *logging.Logger) (int, error) {
	return e.executor.ExecuteAndRedirectOutput(logger)
}
