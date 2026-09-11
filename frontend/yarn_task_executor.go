package frontend

import (
	"strconv"

	"github.com/eirslett/frontend-maven-plugin/internal/logging"
)

// YarnTaskExecutor runs a task through the installed yarn launcher.
type YarnTaskExecutor struct {
	logger          *logging.Logger
	taskName        string
	argumentsParser *ArgumentsParser
	config          YarnExecutorConfig
}

// NewYarnTaskExecutor runs a task under an explicit name, with extra arguments
// appended to whatever the caller configured.
func NewYarnTaskExecutor(loggerName string, config YarnExecutorConfig, taskName string, additionalArguments []string) *YarnTaskExecutor {
	return &YarnTaskExecutor{
		logger:          logging.GetLogger(loggerName),
		taskName:        taskName,
		argumentsParser: NewArgumentsParserWith(additionalArguments),
		config:          config,
	}
}

// Execute parses args, runs yarn, and turns a non-zero exit into a
// TaskRunnerError naming the task and its arguments.
func (e *YarnTaskExecutor) Execute(args string, environment map[string]string) error {
	arguments := e.argumentsParser.Parse(args)
	e.logger.Info("Running " + taskToString(e.taskName, arguments) + " in " + e.config.WorkingDirectory())

	result, err := NewYarnExecutor(e.config, arguments, environment).ExecuteAndRedirectOutput(e.logger)
	if err != nil {
		return newTaskRunnerError(taskToString(e.taskName, arguments)+" failed.", err)
	}
	if result != 0 {
		return newTaskRunnerError(taskToString(e.taskName, arguments)+" failed. (error code "+strconv.Itoa(result)+")", nil)
	}
	return nil
}
