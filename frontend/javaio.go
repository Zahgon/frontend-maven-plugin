package frontend

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// The original addresses every install location through java.io.File, whose
// path handling is not quite Go's. File's constructor collapses duplicated
// separators and drops a trailing one but leaves ".." segments in place, and
// getCanonicalPath resolves both those segments and any symlink on the way. The
// installer's escape checks and the "install directory + /node" concatenations
// depend on that exact behaviour, so it is reproduced here rather than
// approximated with filepath.Clean.

const separator = string(os.PathSeparator)

// newFile normalises a path the way `new File(String)` does: duplicated
// separators collapse and a trailing separator is dropped, but ".." stays.
func newFile(path string) string {
	if path == "" {
		return ""
	}
	normalised := strings.ReplaceAll(path, "/", separator)
	if separator != "/" {
		normalised = strings.ReplaceAll(normalised, "\\", separator)
	}
	prefix := ""
	if strings.HasPrefix(normalised, separator) {
		prefix = separator
	}
	segments := make([]string, 0, 8)
	for _, segment := range strings.Split(normalised, separator) {
		if segment != "" {
			segments = append(segments, segment)
		}
	}
	joined := prefix + strings.Join(segments, separator)
	if joined == "" && prefix == "" {
		return ""
	}
	return joined
}

// childFile is `new File(parent, child)`.
func childFile(parent, child string) string {
	if parent == "" {
		return newFile(child)
	}
	return newFile(parent + separator + child)
}

// absolutePath is File.getAbsolutePath: the working directory is prepended to a
// relative path, and the result is otherwise left alone — ".." included.
func absolutePath(path string) string {
	normalised := newFile(path)
	if filepath.IsAbs(normalised) {
		return normalised
	}
	cwd, err := os.Getwd()
	if err != nil {
		return normalised
	}
	return newFile(cwd + separator + normalised)
}

// parentPath is File.getParent: the path minus its last segment, or "" when
// there is no parent.
func parentPath(path string) string {
	normalised := newFile(path)
	index := strings.LastIndex(normalised, separator)
	switch {
	case index < 0:
		return ""
	case index == 0:
		return separator
	default:
		return normalised[:index]
	}
}

// canonicalPath is File.getCanonicalPath: absolute, with ".", ".." and every
// symlink resolved. A path that does not exist yet still resolves — the longest
// existing prefix is followed through its symlinks and the remainder is appended
// — because extraction canonicalises destinations before creating them.
func canonicalPath(path string) (string, error) {
	absolute := absolutePath(path)
	resolved, err := filepath.EvalSymlinks(absolute)
	if err == nil {
		return filepath.Clean(resolved), nil
	}
	if !errors.Is(err, fs.ErrNotExist) {
		return "", err
	}
	cleaned := filepath.Clean(absolute)
	remainder := ""
	for current := cleaned; ; {
		parent := filepath.Dir(current)
		if parent == current {
			return cleaned, nil
		}
		remainder = filepath.Join(filepath.Base(current), remainder)
		if base, baseErr := filepath.EvalSymlinks(parent); baseErr == nil {
			return filepath.Join(base, remainder), nil
		} else if !errors.Is(baseErr, fs.ErrNotExist) {
			return "", baseErr
		}
		current = parent
	}
}

// exists reports whether the path resolves to anything at all.
func exists(path string) bool {
	_, err := os.Lstat(path)
	if err == nil {
		return true
	}
	if _, statErr := os.Stat(path); statErr == nil {
		return true
	}
	return false
}

// isDirectory is File.isDirectory: symlinks are followed, so a link to a
// directory reports true.
func isDirectory(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

// isFile is File.isFile.
func isFile(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.Mode().IsRegular()
}

// canWrite is File.canWrite for a directory: whether a new entry can be created
// in it. Permission bits alone do not answer that on every filesystem, so the
// question is settled by trying.
func canWrite(dir string) bool {
	probe, err := os.CreateTemp(dir, ".fmp-write-probe-*")
	if err != nil {
		return false
	}
	name := probe.Name()
	_ = probe.Close()
	_ = os.Remove(name)
	return true
}

// extension is plexus FileUtils.getExtension: the text after the last dot of the
// last path segment, empty when that segment has no dot.
func extension(path string) string {
	base := path
	if index := strings.LastIndexAny(path, `/\`); index >= 0 {
		base = path[index+1:]
	}
	index := strings.LastIndex(base, ".")
	if index < 0 {
		return ""
	}
	return base[index+1:]
}

// relativise is Path.relativize: the route from base to target, expressed with
// ".." segments where it has to climb.
func relativise(base, target string) string {
	relative, err := filepath.Rel(filepath.Clean(base), filepath.Clean(target))
	if err != nil {
		return filepath.Clean(target)
	}
	return relative
}

// startsWithPath is Path.startsWith: a comparison by whole segments, so that
// "/tmp/abc" is not considered to sit inside "/tmp/ab".
func startsWithPath(child, parent string) bool {
	if parent == "" {
		return false
	}
	child = filepath.Clean(child)
	parent = filepath.Clean(parent)
	if child == parent {
		return true
	}
	if !strings.HasSuffix(parent, separator) {
		parent += separator
	}
	return strings.HasPrefix(child, parent)
}

// asError is errors.As with the target's type inferred, so call sites reading
// "does this failure carry an X" stay one line long.
func asError[T error](err error, target *T) bool {
	return errors.As(err, target)
}
