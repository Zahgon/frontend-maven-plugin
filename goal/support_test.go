package goal

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/eirslett/frontend-maven-plugin/internal/logging"
)

// captureLog collects everything the goal layer logs while fn runs.
func captureLog(t *testing.T, fn func()) string {
	t.Helper()
	var buffer bytes.Buffer
	previousOut := logging.SetOutput(&buffer)
	previousLevel := logging.SetLevel(logging.Debug)
	defer func() {
		logging.SetOutput(previousOut)
		logging.SetLevel(previousLevel)
	}()
	fn()
	return buffer.String()
}

func writeFile(t *testing.T, path string, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o777); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o666); err != nil {
		t.Fatal(err)
	}
}

func requireNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// newTestSession points a goal at a fresh working directory with no settings and
// no incremental build context.
func newTestSession(t *testing.T) (*Session, string) {
	t.Helper()
	directory := t.TempDir()
	session := NewSession(directory)
	session.LocalRepository = filepath.Join(directory, "repository")
	return session, directory
}
