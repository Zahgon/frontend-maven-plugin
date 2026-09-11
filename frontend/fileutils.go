package frontend

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
)

// The installers lean on commons-io for recursive copy and delete and on Jackson
// for one field of a package.json. Both are small enough to carry here rather
// than take a dependency for, and both have observable edges — a copy that drops
// the executable bit leaves an unusable node — so they are spelled out.

// copyDirectory is FileUtils.copyDirectory: source's contents are copied into
// destination, which is created if missing, preserving permissions.
func copyDirectory(source, destination string) error {
	info, err := os.Stat(source)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(destination, info.Mode().Perm()); err != nil {
		return err
	}
	entries, err := os.ReadDir(source)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		sourcePath := filepath.Join(source, entry.Name())
		destinationPath := filepath.Join(destination, entry.Name())
		switch {
		case entry.IsDir():
			if err := copyDirectory(sourcePath, destinationPath); err != nil {
				return err
			}
		case entry.Type()&os.ModeSymlink != 0:
			target, err := os.Readlink(sourcePath)
			if err != nil {
				return err
			}
			_ = os.Remove(destinationPath)
			if err := os.Symlink(target, destinationPath); err != nil {
				return err
			}
		default:
			if err := copyFile(sourcePath, destinationPath); err != nil {
				return err
			}
		}
	}
	return nil
}

// deleteDirectory is FileUtils.deleteDirectory: the tree goes, and a directory
// that was never there is not an error.
func deleteDirectory(path string) error {
	if !exists(path) {
		return nil
	}
	return os.RemoveAll(path)
}

// moveFile is Files.move with REPLACE_EXISTING, falling back to a copy when the
// two paths are on different filesystems.
func moveFile(source, destination string) error {
	if err := os.Rename(source, destination); err == nil {
		return nil
	}
	if err := copyFile(source, destination); err != nil {
		return err
	}
	return os.Remove(source)
}

// renameTo is File.renameTo: it reports success rather than raising, because the
// installers branch on the boolean and fall back to copying.
func renameTo(source, destination string) bool {
	return os.Rename(source, destination) == nil
}

// setExecutable is File.setExecutable(true, ownerOnly): the owner always gains
// the bit, and everyone does when ownerOnly is false.
func setExecutable(path string, ownerOnly bool) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	mode := info.Mode().Perm() | 0o100
	if !ownerOnly {
		mode |= 0o010 | 0o001
	}
	return os.Chmod(path, mode) == nil
}

// readPackageVersion reads the "version" field out of a package.json. A missing
// field is reported as absent rather than as an error, which is the distinction
// the installers act on.
func readPackageVersion(path string) (string, bool, error) {
	handle, err := os.Open(path)
	if err != nil {
		return "", false, err
	}
	defer handle.Close()
	data, err := io.ReadAll(handle)
	if err != nil {
		return "", false, err
	}
	var parsed map[string]any
	if err := json.Unmarshal(data, &parsed); err != nil {
		return "", false, err
	}
	value, ok := parsed["version"]
	if !ok {
		return "", false, nil
	}
	return stringify(value), true, nil
}

// stringify renders a decoded JSON value the way Object.toString would, so that
// a version written as a bare number still compares as text.
func stringify(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	case nil:
		return "null"
	default:
		encoded, err := json.Marshal(typed)
		if err != nil {
			return ""
		}
		return string(encoded)
	}
}
