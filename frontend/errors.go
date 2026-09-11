package frontend

import "fmt"

// The original's failure surface is a small exception hierarchy, and which
// branch of it a failure lands on is observable: the goal layer turns a
// TaskRunnerException into a tolerable test failure but any other
// FrontendException into a hard build failure, and the archive extractor
// deliberately throws a *runtime* exception for a malicious zip entry so that it
// cannot be mistaken for an ordinary extraction problem. The hierarchy is
// therefore reproduced with typed errors rather than flattened into fmt.Errorf.

// FrontendError is the base of every failure the front-end library reports,
// the equivalent of FrontendException.
type FrontendError struct {
	Message string
	Cause   error
}

// Error renders the message, which is the text the goal layer surfaces.
func (e *FrontendError) Error() string {
	return e.Message
}

// Unwrap exposes the cause, mirroring Throwable.getCause.
func (e *FrontendError) Unwrap() error {
	return e.Cause
}

// InstallationError is InstallationException: an installer could not put the
// tool in place.
type InstallationError struct {
	FrontendError
}

// TaskRunnerError is TaskRunnerException: a front-end task ran and failed.
type TaskRunnerError struct {
	FrontendError
}

// ArchiveExtractionError is ArchiveExtractionException.
type ArchiveExtractionError struct {
	Message string
	Cause   error
}

// Error renders the message.
func (e *ArchiveExtractionError) Error() string {
	return e.Message
}

// Unwrap exposes the cause.
func (e *ArchiveExtractionError) Unwrap() error {
	return e.Cause
}

// BadArchiveEntryError is the RuntimeException the zip branch throws for an
// entry that would escape the destination directory. It is deliberately *not* an
// ArchiveExtractionError: the original's test suite distinguishes the two, and
// callers that catch extraction failures do not catch this one.
type BadArchiveEntryError struct {
	Message string
}

// Error renders the message, verbatim "Bad zip entry".
func (e *BadArchiveEntryError) Error() string {
	return e.Message
}

// DownloadError is DownloadException.
type DownloadError struct {
	Message string
	Cause   error
}

// Error renders the message.
func (e *DownloadError) Error() string {
	return e.Message
}

// Unwrap exposes the cause.
func (e *DownloadError) Unwrap() error {
	return e.Cause
}

// ProcessExecutionError is ProcessExecutionException.
type ProcessExecutionError struct {
	Message string
	Cause   error
}

// Error renders the message; a cause-only failure renders the cause, which is
// what `new ProcessExecutionException(Throwable)` produces.
func (e *ProcessExecutionError) Error() string {
	if e.Message == "" && e.Cause != nil {
		return e.Cause.Error()
	}
	return e.Message
}

// Unwrap exposes the cause.
func (e *ProcessExecutionError) Unwrap() error {
	return e.Cause
}

func newInstallationError(message string, cause error) *InstallationError {
	return &InstallationError{FrontendError{Message: message, Cause: cause}}
}

func newTaskRunnerError(message string, cause error) *TaskRunnerError {
	return &TaskRunnerError{FrontendError{Message: message, Cause: cause}}
}

func newArchiveExtractionError(message string, cause error) *ArchiveExtractionError {
	return &ArchiveExtractionError{Message: message, Cause: cause}
}

func newDownloadError(message string, cause error) *DownloadError {
	return &DownloadError{Message: message, Cause: cause}
}

func newProcessExecutionError(message string, cause error) *ProcessExecutionError {
	return &ProcessExecutionError{Message: message, Cause: cause}
}

// ioError stands in for the plain java.io.IOException family — including
// FileNotFoundException — that the installers raise and then wrap. Which
// subclass it was is never observable; the message is.
type ioError struct {
	message string
}

func (e *ioError) Error() string {
	return e.message
}

func newIOError(format string, args ...any) error {
	return &ioError{message: fmt.Sprintf(format, args...)}
}
