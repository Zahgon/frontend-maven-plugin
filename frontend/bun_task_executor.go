package frontend

import (
	"strconv"

	"github.com/eirslett/frontend-maven-plugin/internal/logging"
)

// BunTaskExecutor runs a task through the installed bun binary.
type BunTaskExecutor struct {
	logger          *logging.Logger
	taskName        string
	argumentsParser *ArgumentsParser
	config          BunExecutorConfig
}

// NewBunTaskExecutor runs a task under an explicit name, with extra arguments
// appended to whatever the caller configured.
func NewBunTaskExecutor(loggerName string, config BunExecutorConfig, taskName string, additionalArguments []string) *BunTaskExecutor {
	return &BunTaskExecutor{
		logger:          logging.GetLogger(loggerName),
		taskName:        taskName,
		argumentsParser: NewArgumentsParserWith(additionalArguments),
		config:          config,
	}
}

// Execute parses args, runs bun, and turns a non-zero exit into a
// TaskRunnerError naming the task and its arguments.
func (e *BunTaskExecutor) Execute(args string, environment map[string]string) error {
	arguments := e.argumentsParser.Parse(args)
	e.logger.Info("Running " + taskToString(e.taskName, arguments) + " in " + e.config.WorkingDirectory())

	result, err := NewBunExecutor(e.config, arguments, environment).ExecuteAndRedirectOutput(e.logger)
	if err != nil {
		return newTaskRunnerError(taskToString(e.taskName, arguments)+" failed.", err)
	}
	if result != 0 {
		return newTaskRunnerError(taskToString(e.taskName, arguments)+" failed. (error code "+strconv.Itoa(result)+")", nil)
	}
	return nil
}
