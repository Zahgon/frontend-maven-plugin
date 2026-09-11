package goal

import (
	"errors"

	"github.com/eirslett/frontend-maven-plugin/frontend"
	"github.com/eirslett/frontend-maven-plugin/internal/logging"
)

var log = logging.GetLogger("frontend")

// Base holds the parameters every goal accepts.
type Base struct {
	// SkipTests skips this goal when it is bound to a testing phase.
	SkipTests bool
	// TestFailureIgnore downgrades a task failure in a testing phase to a
	// logged error. Its use is NOT RECOMMENDED, but quite convenient on
	// occasion.
	TestFailureIgnore bool
	// WorkingDirectory is the base directory for running all Node commands,
	// usually the directory that contains package.json.
	WorkingDirectory string
	// InstallDirectory is the base directory for installing node and npm. When
	// unset it becomes the working directory.
	InstallDirectory string
	// EnvironmentVariables are passed to the build in addition to the process
	// environment.
	EnvironmentVariables map[string]string
	// Session carries the settings, the project layout and the lifecycle phase.
	Session *Session
}

// Goal is one of the plugin's eighteen goals.
type Goal interface {
	// Name is the goal name, as it appears on the command line.
	Name() string
	// Params exposes the shared parameters.
	Params() *Base
	// SkipExecution reports whether this goal's own skip flag is set.
	SkipExecution() bool
	// Run does the work, against a factory wired to the resolved directories.
	Run(factory *frontend.FrontendPluginFactory) error
}

// FailureError is a build failure: the goal ran and could not finish. It is the
// equivalent of MojoFailureException.
type FailureError struct {
	Message string
	Cause   error
}

// Error renders the message.
func (e *FailureError) Error() string {
	return e.Message
}

// Unwrap exposes the cause.
func (e *FailureError) Unwrap() error {
	return e.Cause
}

// toFailure renders a front-end failure the way the original's
// MojoUtils.toMojoFailureException does: the message, then the cause's message
// after a colon when there is one.
func toFailure(err error) *FailureError {
	message := err.Error()
	if cause := errors.Unwrap(err); cause != nil {
		message += ": " + cause.Error()
	}
	return &FailureError{Message: message, Cause: err}
}

// isTestingPhase reports whether the goal is running in a phase where test
// settings apply.
func (b *Base) isTestingPhase() bool {
	phase := ""
	if b.Session != nil {
		phase = b.Session.LifecyclePhase
	}
	return phase == "test" || phase == "integration-test"
}

// skipTestPhase reports whether skipTests silences this execution.
func (b *Base) skipTestPhase() bool {
	return b.SkipTests && b.isTestingPhase()
}

// Execute runs a goal through the lifecycle every goal shares: the skip checks,
// the install-directory fallback, and the two ways a failure is reported.
func Execute(g Goal) error {
	base := g.Params()

	if base.TestFailureIgnore && !base.isTestingPhase() {
		log.Info("testFailureIgnore property is ignored in non test phases")
	}
	if base.skipTestPhase() || g.SkipExecution() {
		log.Info("Skipping execution.")
		return nil
	}

	if base.InstallDirectory == "" {
		base.InstallDirectory = base.WorkingDirectory
	}

	err := g.Run(frontend.NewFrontendPluginFactoryWithCache(
		base.WorkingDirectory,
		base.InstallDirectory,
		base.cacheResolver()))
	if err == nil {
		return nil
	}

	var taskRunnerError *frontend.TaskRunnerError
	if errors.As(err, &taskRunnerError) {
		if base.TestFailureIgnore && base.isTestingPhase() {
			log.Error("There are test failures.\nFailed to run task: " + taskRunnerError.Error())
			return nil
		}
		return &FailureError{Message: "Failed to run task", Cause: err}
	}
	return toFailure(err)
}

// cacheResolver keeps downloads in the local artifact repository, as the
// original's RepositoryCacheResolver does, falling back to the library's own
// per-install cache directory when no local repository is known.
func (b *Base) cacheResolver() frontend.CacheResolver {
	if b.Session != nil && b.Session.LocalRepository != "" {
		return NewRepositoryCacheResolver(b.Session.LocalRepository)
	}
	return frontend.NewDirectoryCacheResolver(b.InstallDirectory + "/cache")
}

// buildContext is the incremental-build context of this session, or the
// full-build one when the session names none.
func (b *Base) buildContext() BuildContext {
	if b.Session != nil && b.Session.BuildContext != nil {
		return b.Session.BuildContext
	}
	return NewFullBuildContext()
}
