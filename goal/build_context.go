package goal

import (
	"io/fs"
	"os"
	"path/filepath"
)

// BuildContext is the incremental-build interface the original gets from
// plexus-build-api: an IDE integration supplies one that knows which files
// changed, and a command-line build gets one that says nothing changed
// incrementally, so everything runs.
type BuildContext interface {
	// IsIncremental reports whether this build is an incremental one.
	IsIncremental() bool
	// HasDelta reports whether the named file changed since the last build.
	HasDelta(path string) bool
	// Scan lists the files below dir that changed since the last build.
	Scan(dir string) []string
	// Refresh tells the host that dir's contents were rewritten.
	Refresh(dir string)
}

// FullBuildContext is the context a command-line build runs under: nothing is
// incremental, so every goal executes and every delta check answers true.
type FullBuildContext struct{}

// NewFullBuildContext builds the non-incremental context.
func NewFullBuildContext() *FullBuildContext {
	return &FullBuildContext{}
}

// IsIncremental is always false for a full build.
func (c *FullBuildContext) IsIncremental() bool { return false }

// HasDelta is always true for a full build: every file is treated as changed.
func (c *FullBuildContext) HasDelta(string) bool { return true }

// Scan lists every file below dir.
func (c *FullBuildContext) Scan(dir string) []string {
	found := []string{}
	if dir == "" {
		return found
	}
	_ = filepath.WalkDir(dir, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if !entry.IsDir() {
			relative, relErr := filepath.Rel(dir, path)
			if relErr == nil {
				found = append(found, relative)
			}
		}
		return nil
	})
	return found
}

// Refresh notes that a directory was rewritten. A command-line build has no IDE
// to tell, so the record only goes to the debug log.
func (c *FullBuildContext) Refresh(dir string) {
	log.Debug("Refreshed {}", dir)
}

// ShouldExecute decides whether a file-watching goal has anything to do.
//
// Without a build context, or outside an incremental build, it always does. In
// an incremental build it runs when one of the trigger files changed, when no
// source directory was configured, or when the source directory scan finds
// anything at all.
func ShouldExecute(buildContext BuildContext, triggerfiles []string, srcdir string) bool {
	// With no buildContext, or when this is not an incremental build, always
	// execute.
	if buildContext == nil || !buildContext.IsIncremental() {
		return true
	}

	for _, triggerfile := range triggerfiles {
		if buildContext.HasDelta(triggerfile) {
			return true
		}
	}

	if srcdir == "" {
		return true
	}

	// Check for changes in the srcdir.
	includedFiles := buildContext.Scan(srcdir)
	return len(includedFiles) > 0
}

// fileExists is a small helper the goals share.
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
