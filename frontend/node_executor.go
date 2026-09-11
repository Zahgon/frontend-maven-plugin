package frontend

import "github.com/eirslett/frontend-maven-plugin/internal/logging"

// NodeExecutor runs the installed node binary, with its own directory on PATH
// so that a script which shells out to "node" finds the same one.
type NodeExecutor struct {
	executor *ProcessExecutor
}

// NewNodeExecutor prepares a node invocation with the given arguments.
func NewNodeExecutor(config NodeExecutorConfig, arguments []string, additionalEnvironment map[string]string) *NodeExecutor {
	node := absolutePath(config.NodePath())
	localPaths := []string{parentPath(config.NodePath())}
	return &NodeExecutor{
		executor: NewProcessExecutor(
			config.WorkingDirectory(),
			localPaths,
			Prepend(node, arguments),
			config.Platform(),
			additionalEnvironment),
	}
}

// ExecuteAndGetResult runs node and returns its trimmed standard output.
func (e *NodeExecutor) ExecuteAndGetResult(logger *logging.Logger) (string, error) {
	return e.executor.ExecuteAndGetResult(logger)
}

// ExecuteAndRedirectOutput runs node, logging its output, and returns its exit
// code.
func (e *NodeExecutor) ExecuteAndRedirectOutput(logger *logging.Logger) (int, error) {
	return e.executor.ExecuteAndRedirectOutput(logger)
}
