package goal

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/eirslett/frontend-maven-plugin/frontend"
)

// Session is what the original reads off the Maven session: the settings, the
// directories that locate the project, the local artifact repository, the
// lifecycle phase the goal is running in, and the system properties that can
// override a parameter at runtime.
type Session struct {
	// Settings is the parsed settings file.
	Settings *Settings
	// BaseDir is the current project's base directory.
	BaseDir string
	// MultiModuleProjectDirectory is the root of a multi-module build.
	MultiModuleProjectDirectory string
	// ExecutionRootDirectory is where the build was invoked from.
	ExecutionRootDirectory string
	// LocalRepository is the local artifact repository the repository cache
	// resolver stores downloads in.
	LocalRepository string
	// LifecyclePhase is the phase this goal is bound to, which decides whether
	// skipTests and testFailureIgnore apply.
	LifecyclePhase string
	// SystemProperties are the -D properties of the build.
	SystemProperties map[string]string
	// BuildContext supplies incremental-build deltas; nil means every goal runs.
	BuildContext BuildContext
}

// NewSession builds a session rooted at baseDir with default settings and no
// incremental build context.
func NewSession(baseDir string) *Session {
	return &Session{
		Settings:                    &Settings{},
		BaseDir:                     baseDir,
		MultiModuleProjectDirectory: baseDir,
		ExecutionRootDirectory:      baseDir,
		LocalRepository:             DefaultLocalRepository(),
		SystemProperties:            map[string]string{},
	}
}

// SystemProperty reads a system property, falling back to the value configured
// on the goal. A "-D" override wins, which is how npmRegistryURL is overridden
// at run time.
func (s *Session) SystemProperty(name, fallback string) string {
	if s == nil {
		return fallback
	}
	if value, ok := s.SystemProperties[name]; ok {
		return value
	}
	return fallback
}

// DefaultLocalRepository is ~/.m2/repository, where the original's resolver
// keeps cached downloads.
func DefaultLocalRepository() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".m2", "repository")
}

const yarnrcYamlFileName = ".yarnrc.yml"

// IsYarnrcYamlFilePresent reports whether a .yarnrc.yml exists at the project
// root — in a multi-module build, the reactor root — which is how Yarn Berry is
// detected.
func IsYarnrcYamlFilePresent(session *Session, workingDirectory string) bool {
	if session == nil {
		return false
	}
	filesToCheck := []string{
		filepath.Join(session.BaseDir, yarnrcYamlFileName),
		filepath.Join(session.MultiModuleProjectDirectory, yarnrcYamlFileName),
		filepath.Join(session.ExecutionRootDirectory, yarnrcYamlFileName),
		filepath.Join(workingDirectory, yarnrcYamlFileName),
	}
	for _, candidate := range filesToCheck {
		if _, err := os.Stat(candidate); err == nil {
			return true
		}
	}
	return false
}

// repositoryCacheGroupID is the group the cached downloads are filed under in
// the local repository.
const repositoryCacheGroupID = "com.github.eirslett"

// RepositoryCacheResolver keeps downloads in the local artifact repository,
// under the same path layout an artifact of the same coordinates would occupy.
type RepositoryCacheResolver struct {
	localRepository string
}

// NewRepositoryCacheResolver caches into localRepository.
func NewRepositoryCacheResolver(localRepository string) *RepositoryCacheResolver {
	return &RepositoryCacheResolver{localRepository: localRepository}
}

// Resolve returns the local-repository path for a cached download:
// <group as directories>/<artifact>/<version>/<artifact>-<version>[-<classifier>].<extension>
func (r *RepositoryCacheResolver) Resolve(descriptor *frontend.CacheDescriptor) string {
	version := trimLeadingV(descriptor.Version())
	name := descriptor.Name()

	filename := name + "-" + version
	if descriptor.Classifier() != "" {
		filename += "-" + descriptor.Classifier()
	}
	filename += "." + descriptor.Extension()

	segments := []string{r.localRepository}
	segments = append(segments, strings.Split(repositoryCacheGroupID, ".")...)
	segments = append(segments, name, version, filename)
	return filepath.Join(segments...)
}

// trimLeadingV strips the "v" a Node or Yarn version carries, which is not part
// of an artifact version.
func trimLeadingV(version string) string {
	if len(version) > 0 && version[0] == 'v' {
		return version[1:]
	}
	return version
}
