package frontend

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/eirslett/frontend-maven-plugin/internal/logging"
)

// ArchiveExtractor unpacks a downloaded distribution.
type ArchiveExtractor interface {
	Extract(archive, destinationDirectory string) error
}

var archiveExtractorLogger = logging.GetLogger("DefaultArchiveExtractor")

// DefaultArchiveExtractor handles the three shapes a distribution arrives in:
// a Windows installer, a zip, and — for everything else — a gzip-compressed tar.
type DefaultArchiveExtractor struct{}

// NewDefaultArchiveExtractor builds the standard extractor.
func NewDefaultArchiveExtractor() *DefaultArchiveExtractor {
	return &DefaultArchiveExtractor{}
}

// prepDestination creates the entry's directory, or its parent, and refuses to
// write where it has no permission.
func (e *DefaultArchiveExtractor) prepDestination(path string, directory bool) error {
	if directory {
		return os.MkdirAll(path, 0o777)
	}
	parent := parentPath(path)
	if !exists(parent) {
		if err := os.MkdirAll(parent, 0o777); err != nil {
			return err
		}
	}
	if !canWrite(parent) {
		return fmt.Errorf("Could not get write permissions for '%s'", absolutePath(parent))
	}
	return nil
}

// Extract unpacks archive into destinationDirectory, dispatching on the
// archive's extension.
func (e *DefaultArchiveExtractor) Extract(archive, destinationDirectory string) error {
	archiveFile := newFile(archive)

	handle, err := os.Open(archiveFile)
	if err != nil {
		return newArchiveExtractionError("Could not extract archive: '"+archive+"'", err)
	}
	defer handle.Close()

	switch extension(absolutePath(archiveFile)) {
	case "msi":
		err = e.extractMSI(archiveFile, destinationDirectory)
	case "zip":
		err = e.extractZip(archiveFile, destinationDirectory)
	default:
		// A tar reader accepts a plain FileInputStream too, if uncompressed
		// ".tar" archives ever need extracting.
		err = e.extractTarGz(handle, destinationDirectory)
	}
	if err == nil {
		return nil
	}
	// A malicious zip entry is reported as itself, never folded into an
	// extraction failure: callers distinguish the two.
	var badEntry *BadArchiveEntryError
	var extraction *ArchiveExtractionError
	if asError(err, &badEntry) || asError(err, &extraction) {
		return err
	}
	return newArchiveExtractionError("Could not extract archive: '"+archive+"'", err)
}

func (e *DefaultArchiveExtractor) extractMSI(archiveFile, destinationDirectory string) error {
	absolute := absolutePath(archiveFile)
	command := exec.Command("msiexec", "/a", absolute, "/qn", `TARGETDIR="`+destinationDirectory+`"`)
	if err := command.Start(); err != nil {
		// The installer could not be launched at all; the original reports that
		// as an ordinary I/O failure, so it is left for Extract to wrap.
		return err
	}
	if err := command.Wait(); err != nil {
		var exitError *exec.ExitError
		if asError(err, &exitError) {
			return newArchiveExtractionError(
				fmt.Sprintf("Could not extract %s; return code %d", absolute, exitError.ExitCode()), nil)
		}
		return newArchiveExtractionError(
			"Unexpected interruption of while waiting for extraction process", err)
	}
	return nil
}

func (e *DefaultArchiveExtractor) extractZip(archiveFile, destinationDirectory string) error {
	destinationPath := filepath.Clean(destinationDirectory)
	reader, err := zip.OpenReader(archiveFile)
	if err != nil {
		return err
	}
	defer reader.Close()

	for _, entry := range reader.File {
		destPath := filepath.Clean(filepath.Join(destinationPath, entry.Name))
		if !startsWithPath(destPath, destinationPath) {
			return &BadArchiveEntryError{Message: "Bad zip entry"}
		}
		if err := e.prepDestination(destPath, isZipDirectory(entry)); err != nil {
			return err
		}
		if isZipDirectory(entry) {
			continue
		}
		if err := copyZipEntry(entry, destPath); err != nil {
			return err
		}
	}
	return nil
}

// isZipDirectory is ZipEntry.isDirectory: the name ends with a slash. A
// zero-length entry without one is a file, not a directory.
func isZipDirectory(entry *zip.File) bool {
	return strings.HasSuffix(entry.Name, "/")
}

func copyZipEntry(entry *zip.File, destPath string) error {
	in, err := entry.Open()
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}

func (e *DefaultArchiveExtractor) extractTarGz(archive io.Reader, destinationDirectory string) error {
	gzipReader, err := gzip.NewReader(archive)
	if err != nil {
		return err
	}
	defer gzipReader.Close()
	tarReader := tar.NewReader(gzipReader)

	// Canonicalising the destination up front keeps symlink resolution
	// consistent across platforms, so an extraction into a symlinked directory
	// is judged against where that link really points.
	canonicalDestinationDirectory, err := canonicalPath(destinationDirectory)
	if err != nil {
		return err
	}

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}

		destPath := childFile(canonicalDestinationDirectory, header.Name)
		isDir := header.Typeflag == tar.TypeDir || strings.HasSuffix(header.Name, "/")
		if err := e.prepDestination(destPath, isDir); err != nil {
			return err
		}

		canonicalDestPath, err := canonicalPath(destPath)
		if err != nil {
			return err
		}
		if !startsWithCaseTolerantPath(canonicalDestPath, canonicalDestinationDirectory) {
			return fmt.Errorf("Expanding %s would create file outside of %s",
				header.Name, canonicalDestinationDirectory)
		}

		if isDir {
			continue
		}
		if err := writeTarEntry(tarReader, header, destPath); err != nil {
			return err
		}
	}
}

func writeTarEntry(tarReader *tar.Reader, header *tar.Header, destPath string) error {
	out, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer out.Close()
	// Only the owner-execute bit is honoured, and only as a whole: File
	// .setExecutable(boolean) grants or revokes execution for the owner alone.
	isExecutable := header.Mode&0o100 > 0
	if err := applyExecutable(destPath, isExecutable); err != nil {
		return err
	}
	_, err = io.Copy(out, tarReader)
	return err
}

func applyExecutable(path string, executable bool) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	mode := info.Mode().Perm()
	if executable {
		mode |= 0o100
	} else {
		mode &^= 0o100
	}
	return os.Chmod(path, mode)
}

// startsWithCaseTolerantPath decides whether an extracted path really sits
// inside the destination, on filesystems that may or may not fold case.
//
// A literal match settles it. Otherwise, if the exact-case path exists but its
// lower-cased spelling does not, the filesystem is case-sensitive and the paths
// genuinely differ; if not, a case-insensitive comparison is the right answer.
func startsWithCaseTolerantPath(destPath, destDir string) bool {
	if strings.HasPrefix(destPath, destDir) {
		return true
	}
	if len(destDir) > len(destPath) {
		return false
	}
	if exists(destPath) && !exists(strings.ToLower(destPath)) {
		return false
	}
	return strings.HasPrefix(strings.ToLower(destPath), strings.ToLower(destDir))
}
