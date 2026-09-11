package frontend

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// The recursive copy, the delete and the package.json read replace commons-io
// and Jackson; each has an edge the installers depend on.

func TestCopyDirectoryPreservesModesAndLinks(t *testing.T) {
	directory := t.TempDir()
	source := filepath.Join(directory, "source")
	writeFixture(t, filepath.Join(source, "plain.txt"), []byte("plain"))
	writeFixture(t, filepath.Join(source, "nested", "script.sh"), []byte("#!/bin/sh\n"))
	requireNoError(t, os.Chmod(filepath.Join(source, "nested", "script.sh"), 0o755))
	linkSupported := os.Symlink("plain.txt", filepath.Join(source, "link")) == nil

	destination := filepath.Join(directory, "destination")
	requireNoError(t, copyDirectory(source, destination))

	want := []string{"d nested", "f nested/script.sh", "f plain.txt"}
	if linkSupported {
		want = []string{"d nested", "f nested/script.sh", "l link -> plain.txt", "f plain.txt"}
	}
	got := treeOf(t, destination)
	if len(got) != len(want) {
		t.Fatalf("copied tree = %v, want %d entries", got, len(want))
	}
	info, err := os.Stat(filepath.Join(destination, "nested", "script.sh"))
	requireNoError(t, err)
	if info.Mode().Perm()&0o100 == 0 {
		t.Errorf("copied script mode = %v, want it executable", info.Mode().Perm())
	}
}

func TestCopyDirectoryReportsAMissingSource(t *testing.T) {
	if err := copyDirectory(filepath.Join(t.TempDir(), "absent"), t.TempDir()); err == nil {
		t.Error("copying a missing directory must fail")
	}
}

func TestDeleteDirectoryIgnoresAMissingTree(t *testing.T) {
	directory := t.TempDir()
	writeFixture(t, filepath.Join(directory, "tree", "file"), []byte("x"))

	requireNoError(t, deleteDirectory(filepath.Join(directory, "tree")))
	requireNoError(t, deleteDirectory(filepath.Join(directory, "never-existed")))

	if exists(filepath.Join(directory, "tree")) {
		t.Error("the tree was not deleted")
	}
}

func TestMoveFileAndRenameTo(t *testing.T) {
	directory := t.TempDir()
	source := filepath.Join(directory, "source")
	writeFixture(t, source, []byte("body"))
	destination := filepath.Join(directory, "destination")

	requireNoError(t, moveFile(source, destination))

	if exists(source) || !isFile(destination) {
		t.Error("the file was not moved")
	}
	if err := moveFile(filepath.Join(directory, "absent"), destination); err == nil {
		t.Error("moving a missing file must fail")
	}
	if !renameTo(destination, filepath.Join(directory, "renamed")) {
		t.Error("renaming an existing file must succeed")
	}
	if renameTo(filepath.Join(directory, "absent"), filepath.Join(directory, "other")) {
		t.Error("renaming a missing file must report failure")
	}
}

func TestSetExecutableGrantsTheOwnerOrEveryone(t *testing.T) {
	directory := t.TempDir()
	path := filepath.Join(directory, "script")
	writeFixture(t, path, []byte("#!/bin/sh\n"))
	requireNoError(t, os.Chmod(path, 0o600))

	if !setExecutable(path, true) {
		t.Fatal("setExecutable reported failure")
	}
	info, err := os.Stat(path)
	requireNoError(t, err)
	if info.Mode().Perm()&0o100 == 0 || info.Mode().Perm()&0o011 != 0 {
		t.Errorf("mode = %v, want owner-only execute", info.Mode().Perm())
	}

	if !setExecutable(path, false) {
		t.Fatal("setExecutable reported failure")
	}
	info, err = os.Stat(path)
	requireNoError(t, err)
	if info.Mode().Perm()&0o111 != 0o111 {
		t.Errorf("mode = %v, want execute for everyone", info.Mode().Perm())
	}

	if setExecutable(filepath.Join(directory, "absent"), true) {
		t.Error("setExecutable on a missing file must report failure")
	}
}

func TestReadPackageVersion(t *testing.T) {
	directory := t.TempDir()
	present := filepath.Join(directory, "present.json")
	writeFixture(t, present, []byte(`{"name":"npm","version":"8.6.0"}`))
	absent := filepath.Join(directory, "absent-field.json")
	writeFixture(t, absent, []byte(`{"name":"npm"}`))
	broken := filepath.Join(directory, "broken.json")
	writeFixture(t, broken, []byte(`{`))
	numeric := filepath.Join(directory, "numeric.json")
	writeFixture(t, numeric, []byte(`{"version":3}`))

	version, ok, err := readPackageVersion(present)
	requireNoError(t, err)
	if !ok || version != "8.6.0" {
		t.Errorf("version = %q (present=%t), want 8.6.0", version, ok)
	}

	_, ok, err = readPackageVersion(absent)
	requireNoError(t, err)
	if ok {
		t.Error("a package.json without a version must report the field as absent")
	}

	if _, _, err = readPackageVersion(broken); err == nil {
		t.Error("unparsable JSON must be reported")
	}
	if _, _, err = readPackageVersion(filepath.Join(directory, "missing.json")); err == nil {
		t.Error("a missing file must be reported")
	}

	// A version written as a bare number still compares as text.
	version, ok, err = readPackageVersion(numeric)
	requireNoError(t, err)
	if !ok || version != "3" {
		t.Errorf("version = %q, want %q", version, "3")
	}
}

func TestStringifyRendersDecodedValues(t *testing.T) {
	for value, want := range map[any]string{
		"text": "text",
		nil:    "null",
		3.0:    "3",
		true:   "true",
	} {
		if got := stringify(value); got != want {
			t.Errorf("stringify(%v) = %q, want %q", value, got, want)
		}
	}
	if got := stringify([]any{1.0, 2.0}); got != "[1,2]" {
		t.Errorf("stringify of a list = %q, want %q", got, "[1,2]")
	}
	if !reflect.DeepEqual(stringify(map[string]any{}), "{}") {
		t.Errorf("stringify of a map = %q, want %q", stringify(map[string]any{}), "{}")
	}
}
