package goal

import (
	"path/filepath"
	"testing"
)

// stubBuildContext stands in for an IDE's incremental build context.
type stubBuildContext struct {
	incremental bool
	changed     map[string]bool
	scanned     []string
	refreshed   []string
}

func (c *stubBuildContext) IsIncremental() bool { return c.incremental }

func (c *stubBuildContext) HasDelta(path string) bool { return c.changed[path] }

func (c *stubBuildContext) Scan(string) []string { return c.scanned }

func (c *stubBuildContext) Refresh(dir string) { c.refreshed = append(c.refreshed, dir) }

func TestFullBuildContextRunsEverything(t *testing.T) {
	directory := t.TempDir()
	writeFile(t, filepath.Join(directory, "nested", "file.txt"), "x")
	context := NewFullBuildContext()

	if context.IsIncremental() {
		t.Error("a command-line build is never incremental")
	}
	if !context.HasDelta("anything") {
		t.Error("a full build treats every file as changed")
	}
	if got := context.Scan(directory); len(got) != 1 || got[0] != filepath.Join("nested", "file.txt") {
		t.Errorf("scan = %v, want the one nested file", got)
	}
	if got := context.Scan(""); len(got) != 0 {
		t.Errorf("scan of no directory = %v, want none", got)
	}
	context.Refresh(directory)
}

func TestShouldExecuteAlwaysRunsOutsideAnIncrementalBuild(t *testing.T) {
	if !ShouldExecute(nil, nil, "") {
		t.Error("no build context means the goal runs")
	}
	if !ShouldExecute(&stubBuildContext{incremental: false}, nil, "src") {
		t.Error("a non-incremental build means the goal runs")
	}
}

func TestShouldExecuteFollowsTheTriggerFilesAndSourceDirectory(t *testing.T) {
	changed := &stubBuildContext{incremental: true, changed: map[string]bool{"Gruntfile.js": true}}
	if !ShouldExecute(changed, []string{"Gruntfile.js"}, "src") {
		t.Error("a changed trigger file means the goal runs")
	}

	unchanged := &stubBuildContext{incremental: true, changed: map[string]bool{}}
	if !ShouldExecute(unchanged, []string{"Gruntfile.js"}, "") {
		t.Error("no source directory means the goal runs")
	}
	if ShouldExecute(unchanged, []string{"Gruntfile.js"}, "src") {
		t.Error("an unchanged trigger file and an empty scan means the goal stands down")
	}

	scanned := &stubBuildContext{incremental: true, changed: map[string]bool{}, scanned: []string{"a.js"}}
	if !ShouldExecute(scanned, []string{"Gruntfile.js"}, "src") {
		t.Error("a non-empty source scan means the goal runs")
	}
}

func TestFileExists(t *testing.T) {
	directory := t.TempDir()
	path := filepath.Join(directory, "present")
	writeFile(t, path, "x")

	if !fileExists(path) {
		t.Error("an existing file must read as present")
	}
	if fileExists(filepath.Join(directory, "absent")) {
		t.Error("a missing file must read as absent")
	}
}
