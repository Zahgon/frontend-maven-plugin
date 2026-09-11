package frontend

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

const (
	badTar = "testdata/bad.tgz"
	// A sample zip carrying the zip-slip vulnerability.
	badZip  = "testdata/bad.zip"
	goodTar = "testdata/good.tgz"
	goodZip = "testdata/good.zip"
)

func newExtractor(t *testing.T) (ArchiveExtractor, string) {
	t.Helper()
	return NewDefaultArchiveExtractor(), t.TempDir()
}

func TestExtractGoodTarFile(t *testing.T) {
	extractor, temp := newExtractor(t)
	if err := extractor.Extract(goodTar, temp); err != nil {
		t.Fatalf("Extract returned %v, want no error", err)
	}
}

func TestExtractGoodTarFileSymlink(t *testing.T) {
	extractor, temp := newExtractor(t)
	destination := filepath.Join(temp, "destination")
	if err := os.Mkdir(destination, 0o777); err != nil {
		t.Fatalf("could not create destination: %v", err)
	}
	link := createSymlinkOrSkipTest(t, filepath.Join(temp, "link"), destination)
	if err := extractor.Extract(goodTar, link); err != nil {
		t.Fatalf("Extract returned %v, want no error", err)
	}
}

func TestExtractBadTarFile(t *testing.T) {
	extractor, temp := newExtractor(t)
	err := extractor.Extract(badTar, temp)
	var extraction *ArchiveExtractionError
	if !errors.As(err, &extraction) {
		t.Fatalf("Extract returned %v, want an ArchiveExtractionError", err)
	}
}

func TestExtractGoodZipFile(t *testing.T) {
	_, temp := newExtractor(t)
	assertGoodZipExtractedTo(t, temp, temp)
}

func TestExtractGoodZipFileWithRelTarget(t *testing.T) {
	_, temp := newExtractor(t)
	assertGoodZipExtractedTo(t, createRelPath(t, temp), temp)
}

func TestExtractBadZipFile(t *testing.T) {
	_, temp := newExtractor(t)
	assertBadZipThrowsException(t, temp)
}

func TestExtractBadZipFileWithRelTarget(t *testing.T) {
	_, temp := newExtractor(t)
	assertBadZipThrowsException(t, createRelPath(t, temp))
}

func TestExtractBadTarFileSymlink(t *testing.T) {
	extractor, temp := newExtractor(t)
	destination := filepath.Join(temp, "destination")
	if err := os.Mkdir(destination, 0o777); err != nil {
		t.Fatalf("could not create destination: %v", err)
	}
	link := createSymlinkOrSkipTest(t, filepath.Join(destination, "link"), destination)
	err := extractor.Extract(badTar, link)
	var extraction *ArchiveExtractionError
	if !errors.As(err, &extraction) {
		t.Fatalf("Extract returned %v, want an ArchiveExtractionError", err)
	}
}

// assertBadZipThrowsException requires the malicious entry to be reported as a
// bad archive entry rather than as an ordinary extraction failure — the
// distinction the original draws by throwing a RuntimeException there.
func assertBadZipThrowsException(t *testing.T, targetDir string) {
	t.Helper()
	err := NewDefaultArchiveExtractor().Extract(badZip, targetDir)
	var badEntry *BadArchiveEntryError
	if !errors.As(err, &badEntry) {
		t.Fatalf("Extract returned %v, want a BadArchiveEntryError", err)
	}
	var extraction *ArchiveExtractionError
	if errors.As(err, &extraction) {
		t.Errorf("Extract returned %v, which must not be an ArchiveExtractionError", err)
	}
}

func assertGoodZipExtractedTo(t *testing.T, targetDir, temp string) {
	t.Helper()
	if err := NewDefaultArchiveExtractor().Extract(goodZip, targetDir); err != nil {
		t.Fatalf("Extract returned %v, want no error", err)
	}
	const nameOfFileInZip = "zip"
	if !isFile(filepath.Join(temp, nameOfFileInZip)) {
		t.Error("zip content not found in target directory")
	}
}

// createRelPath returns a path that reaches the same directory by way of its
// parent, so extraction has to normalise before it compares.
func createRelPath(t *testing.T, orig string) string {
	t.Helper()
	canonical, err := filepath.EvalSymlinks(orig)
	if err != nil {
		t.Fatalf("could not canonicalise %q: %v", orig, err)
	}
	dirName := filepath.Base(canonical)
	result := filepath.Join(canonical, "..", dirName) + string(os.PathSeparator) + ".." + string(os.PathSeparator) + dirName
	// Ensure the result is different from the input...
	if result == canonical {
		t.Fatalf("relative path %q is not different from %q", result, canonical)
	}
	// ...but still points at the same directory.
	resolved, err := filepath.EvalSymlinks(result)
	if err != nil {
		t.Fatalf("could not canonicalise %q: %v", result, err)
	}
	if resolved != canonical {
		t.Fatalf("relative path %q resolves to %q, want %q", result, resolved, canonical)
	}
	return result
}

func createSymlinkOrSkipTest(t *testing.T, link, target string) string {
	t.Helper()
	if err := os.Symlink(target, link); err != nil {
		t.Skip("symlinks not supported")
	}
	return link
}

func TestExtractMSIReportsThatTheInstallerCouldNotBeLaunched(t *testing.T) {
	// msiexec exists only on Windows, so everywhere else this exercises the
	// could-not-launch path, which the original reports as an I/O failure.
	directory := t.TempDir()
	archive := filepath.Join(directory, "installer.msi")
	writeFixture(t, archive, []byte("not really an installer"))

	err := NewDefaultArchiveExtractor().Extract(archive, filepath.Join(directory, "out"))

	if err == nil {
		t.Skip("msiexec is available, so the launch did not fail")
	}
	var extraction *ArchiveExtractionError
	if !errors.As(err, &extraction) {
		t.Fatalf("error = %v, want an ArchiveExtractionError", err)
	}
	if extraction.Error() != "Could not extract archive: '"+archive+"'" {
		t.Errorf("message = %q, want it to name the archive", extraction.Error())
	}
}

func TestArchiveErrorsCarryTheirMessagesAndCauses(t *testing.T) {
	cause := errors.New("underlying")
	extraction := newArchiveExtractionError("outer", cause)
	if extraction.Error() != "outer" {
		t.Errorf("message = %q, want %q", extraction.Error(), "outer")
	}
	if !errors.Is(extraction, cause) {
		t.Error("the cause is not reachable through the chain")
	}

	badEntry := &BadArchiveEntryError{Message: "Bad zip entry"}
	if badEntry.Error() != "Bad zip entry" {
		t.Errorf("message = %q, want %q", badEntry.Error(), "Bad zip entry")
	}
}

func TestProcessAndDownloadErrorsCarryTheirCauses(t *testing.T) {
	cause := errors.New("underlying")

	process := newProcessExecutionError("", cause)
	if process.Error() != "underlying" {
		t.Errorf("message = %q, want the cause's", process.Error())
	}
	if !errors.Is(process, cause) {
		t.Error("the process failure's cause is not reachable")
	}

	download := newDownloadError("outer", cause)
	if download.Error() != "outer" {
		t.Errorf("message = %q, want %q", download.Error(), "outer")
	}
	if !errors.Is(download, cause) {
		t.Error("the download failure's cause is not reachable")
	}

	installation := newInstallationError("outer", cause)
	if !errors.Is(installation, cause) {
		t.Error("the installation failure's cause is not reachable")
	}
	task := newTaskRunnerError("outer", cause)
	if !errors.Is(task, cause) {
		t.Error("the task failure's cause is not reachable")
	}
	if got := newIOError("could not %s", "read").Error(); got != "could not read" {
		t.Errorf("message = %q, want %q", got, "could not read")
	}
}
