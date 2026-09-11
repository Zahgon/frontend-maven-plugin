package frontend

import (
	"strconv"

	"github.com/eirslett/frontend-maven-plugin/internal/logging"
)

// NodeTaskExecutor runs a JavaScript entry point under the installed node.
type NodeTaskExecutor struct {
	logger          *logging.Logger
	taskName        string
	taskLocation    string
	argumentsParser *ArgumentsParser
	config          NodeExecutorConfig
	proxy           map[string]string
}

// NewNodeTaskExecutor runs the script at taskLocation, naming the task after it.
func NewNodeTaskExecutor(loggerName string, config NodeExecutorConfig, taskLocation string) *NodeTaskExecutor {
	return NewNodeTaskExecutorWithArguments(loggerName, config, taskLocation, nil)
}

// NewNodeTaskExecutorWithArguments runs the script at taskLocation with extra
// arguments appended to whatever the caller configured.
func NewNodeTaskExecutorWithArguments(loggerName string, config NodeExecutorConfig, taskLocation string, additionalArguments []string) *NodeTaskExecutor {
	return NewNamedNodeTaskExecutor(loggerName, config, taskNameFor(taskLocation), taskLocation, additionalArguments, nil)
}

// NewNamedNodeTaskExecutor runs the script at taskLocation under an explicit
// task name, with extra arguments and extra environment variables.
func NewNamedNodeTaskExecutor(loggerName string, config NodeExecutorConfig, taskName, taskLocation string, additionalArguments []string, proxy map[string]string) *NodeTaskExecutor {
	return &NodeTaskExecutor{
		logger:          logging.GetLogger(loggerName),
		taskName:        taskName,
		taskLocation:    taskLocation,
		argumentsParser: NewArgumentsParserWith(additionalArguments),
		config:          config,
		proxy:           proxy,
	}
}

// SetTaskLocation repoints the executor at a different script, which the pnpm
// and corepack runners do once they know which entry point was installed.
func (e *NodeTaskExecutor) SetTaskLocation(taskLocation string) {
	e.taskLocation = taskLocation
}

// Execute parses args, runs the task, and turns a non-zero exit into a
// TaskRunnerError naming the task and its arguments.
func (e *NodeTaskExecutor) Execute(args string, environment map[string]string) error {
	absoluteTaskLocation := e.absoluteTaskLocation()
	arguments := e.argumentsParser.Parse(args)
	e.logger.Info("Running " + taskToString(e.taskName, arguments) + " in " + e.config.WorkingDirectory())

	internalEnvironment := make(map[string]string, len(environment)+len(e.proxy))
	for key, value := range environment {
		internalEnvironment[key] = value
	}
	for key, value := range e.proxy {
		internalEnvironment[key] = value
	}

	result, err := NewNodeExecutor(e.config, Prepend(absoluteTaskLocation, arguments), internalEnvironment).
		ExecuteAndRedirectOutput(e.logger)
	if err != nil {
		return newTaskRunnerError(taskToString(e.taskName, arguments)+" failed.", err)
	}
	if result != 0 {
		return newTaskRunnerError(taskToString(e.taskName, arguments)+" failed. (error code "+strconv.Itoa(result)+")", nil)
	}
	return nil
}

// absoluteTaskLocation resolves a relative script against the working directory,
// falling back to the install directory when it is not there.
func (e *NodeTaskExecutor) absoluteTaskLocation() string {
	location := Normalize(e.taskLocation)
	if IsRelative(e.taskLocation) {
		taskFile := childFile(e.config.WorkingDirectory(), location)
		if !exists(taskFile) {
			taskFile = childFile(e.config.InstallDirectory(), location)
		}
		location = absolutePath(taskFile)
	}
	return location
}
