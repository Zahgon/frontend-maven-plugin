package frontend

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// Path handling is where the installation layout is decided, so the java.io.File
// semantics this package reproduces are asserted directly.

func TestNewFileCollapsesSeparatorsButKeepsDotDot(t *testing.T) {
	for path, want := range map[string]string{
		"/a//b":       "/a/b",
		"/a/b/":       "/a/b",
		"a/b":         "a/b",
		"/":           "/",
		"":            "",
		"/a/../b":     "/a/../b",
		"///a///b///": "/a/b",
	} {
		if got := newFile(path); got != want {
			t.Errorf("newFile(%q) = %q, want %q", path, got, want)
		}
	}
}

func TestChildFileJoinsOntoAParent(t *testing.T) {
	if got := childFile("/a", "b/c"); got != "/a/b/c" {
		t.Errorf("childFile = %q, want %q", got, "/a/b/c")
	}
	if got := childFile("/a/", "b"); got != "/a/b" {
		t.Errorf("childFile = %q, want %q", got, "/a/b")
	}
	if got := childFile("", "b"); got != "b" {
		t.Errorf("childFile = %q, want %q", got, "b")
	}
}

func TestParentPathDropsTheLastSegment(t *testing.T) {
	for path, want := range map[string]string{
		"/a/b/c": "/a/b",
		"/a":     "/",
		"a":      "",
		"a/b":    "a",
	} {
		if got := parentPath(path); got != want {
			t.Errorf("parentPath(%q) = %q, want %q", path, got, want)
		}
	}
}

func TestAbsolutePathResolvesAgainstTheWorkingDirectory(t *testing.T) {
	cwd, err := os.Getwd()
	requireNoError(t, err)

	if got := absolutePath("/already/absolute"); got != "/already/absolute" {
		t.Errorf("absolutePath = %q, want it unchanged", got)
	}
	if got := absolutePath("relative"); got != filepath.Join(cwd, "relative") {
		t.Errorf("absolutePath = %q, want it under %q", got, cwd)
	}
}

func TestCanonicalPathResolvesSymlinksAndMissingSuffixes(t *testing.T) {
	directory := t.TempDir()
	real := filepath.Join(directory, "real")
	requireNoError(t, os.Mkdir(real, 0o777))
	link := filepath.Join(directory, "link")
	if err := os.Symlink(real, link); err != nil {
		t.Skip("symlinks not supported")
	}

	resolved, err := canonicalPath(link)
	requireNoError(t, err)
	realResolved, err := filepath.EvalSymlinks(real)
	requireNoError(t, err)
	if resolved != realResolved {
		t.Errorf("canonicalPath(link) = %q, want %q", resolved, realResolved)
	}

	// A path that does not exist yet still resolves through its existing prefix.
	future, err := canonicalPath(filepath.Join(link, "not", "there"))
	requireNoError(t, err)
	if !strings.HasPrefix(future, realResolved) {
		t.Errorf("canonicalPath of a missing path = %q, want it under %q", future, realResolved)
	}
}

func TestExistsAndKindPredicates(t *testing.T) {
	directory := t.TempDir()
	file := filepath.Join(directory, "file")
	writeFixture(t, file, []byte("x"))

	if !exists(file) || !isFile(file) || isDirectory(file) {
		t.Error("a regular file must read as an existing file, not a directory")
	}
	if !exists(directory) || !isDirectory(directory) || isFile(directory) {
		t.Error("a directory must read as an existing directory, not a file")
	}
	missing := filepath.Join(directory, "missing")
	if exists(missing) || isFile(missing) || isDirectory(missing) {
		t.Error("a missing path must read as absent")
	}
	if !canWrite(directory) {
		t.Error("a fresh temporary directory must be writable")
	}
	if canWrite(missing) {
		t.Error("a missing directory must not read as writable")
	}
}

func TestExtensionIsTheTailOfTheLastSegment(t *testing.T) {
	for path, want := range map[string]string{
		"/a/b/archive.tar.gz": "gz",
		"/a/b/installer.msi":  "msi",
		"/a/b/noext":          "",
		"a.zip":               "zip",
		`C:\a\b.exe`:          "exe",
		"/a.b/c":              "",
	} {
		if got := extension(path); got != want {
			t.Errorf("extension(%q) = %q, want %q", path, got, want)
		}
	}
}

func TestRelativiseFindsTheRouteBetweenPaths(t *testing.T) {
	if got := relativise("/a/b", "/a/b/c/d"); got != filepath.Join("c", "d") {
		t.Errorf("relativise = %q, want %q", got, filepath.Join("c", "d"))
	}
	if got := relativise("/a/b", "/a/x"); got != filepath.Join("..", "x") {
		t.Errorf("relativise = %q, want %q", got, filepath.Join("..", "x"))
	}
	if runtime.GOOS != "windows" {
		// A path on another volume has no route; the target is used as it is.
		if got := relativise("relative", "/absolute"); got != "/absolute" {
			t.Errorf("relativise = %q, want %q", got, "/absolute")
		}
	}
}

func TestStartsWithPathComparesWholeSegments(t *testing.T) {
	if !startsWithPath("/tmp/ab/c", "/tmp/ab") {
		t.Error("a child path must read as inside its parent")
	}
	if !startsWithPath("/tmp/ab", "/tmp/ab") {
		t.Error("a path is inside itself")
	}
	if startsWithPath("/tmp/abc", "/tmp/ab") {
		t.Error("a sibling with a shared prefix must not read as inside")
	}
	if startsWithPath("/tmp/abc", "") {
		t.Error("no path is inside an empty parent")
	}
}

func TestAsErrorFindsATypedCause(t *testing.T) {
	wrapped := newArchiveExtractionError("outer", &BadArchiveEntryError{Message: "inner"})

	var badEntry *BadArchiveEntryError
	if !asError(error(wrapped), &badEntry) || badEntry.Error() != "inner" {
		t.Error("asError did not find the wrapped cause")
	}
	var download *DownloadError
	if asError(error(wrapped), &download) {
		t.Error("asError matched a type that is not in the chain")
	}
	if !errors.Is(wrapped, error(badEntry)) {
		t.Error("the cause is not reachable through the chain")
	}
}

func TestInstallConfigNormalisesItsDirectories(t *testing.T) {
	// The original holds both directories as java.io.File, whose constructor
	// collapses duplicate separators and drops a trailing one. Every executable
	// path is then built by concatenating onto that value, so leaving it
	// un-normalised puts a stray separator into the middle of every path the
	// plugin reports — the defect the differential against the original caught.
	for _, testCase := range []struct {
		installDirectory string
		wantInstallDir   string
		wantNodePath     string
	}{
		{"/tmp/install", "/tmp/install", "/tmp/install/node/node"},
		{"/tmp/install/", "/tmp/install", "/tmp/install/node/node"},
		{"/a//b", "/a/b", "/a/b/node/node"},
		{"/a///b///", "/a/b", "/a/b/node/node"},
	} {
		config := NewInstallConfig(testCase.installDirectory, testCase.installDirectory+"/work",
			NewDirectoryCacheResolver("/cache"), testPlatform())

		if got := config.InstallDirectory(); got != testCase.wantInstallDir {
			t.Errorf("InstallDirectory(%q) = %q, want %q",
				testCase.installDirectory, got, testCase.wantInstallDir)
		}
		if got := config.WorkingDirectory(); got != testCase.wantInstallDir+"/work" {
			t.Errorf("WorkingDirectory(%q) = %q, want %q",
				testCase.installDirectory, got, testCase.wantInstallDir+"/work")
		}
		if got := NewInstallNodeExecutorConfig(config).NodePath(); got != testCase.wantNodePath {
			t.Errorf("NodePath(%q) = %q, want %q",
				testCase.installDirectory, got, testCase.wantNodePath)
		}
	}
}
