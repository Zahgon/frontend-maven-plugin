package goal

import (
	"path/filepath"
	"testing"

	"github.com/eirslett/frontend-maven-plugin/frontend"
)

func TestSessionSystemPropertiesOverrideConfiguredValues(t *testing.T) {
	session, _ := newTestSession(t)
	session.SystemProperties["npmRegistryURL"] = "http://override"

	if got := session.SystemProperty("npmRegistryURL", "http://configured"); got != "http://override" {
		t.Errorf("property = %q, want the override", got)
	}
	if got := session.SystemProperty("absent", "http://configured"); got != "http://configured" {
		t.Errorf("property = %q, want the configured value", got)
	}
	var absent *Session
	if got := absent.SystemProperty("any", "fallback"); got != "fallback" {
		t.Errorf("property = %q, want the fallback", got)
	}
}

func TestIsYarnrcYamlFilePresentChecksEveryProjectRoot(t *testing.T) {
	session, directory := newTestSession(t)
	working := filepath.Join(directory, "frontend")

	if IsYarnrcYamlFilePresent(session, working) {
		t.Error("no .yarnrc.yml means no Yarn Berry")
	}
	if IsYarnrcYamlFilePresent(nil, working) {
		t.Error("no session means no Yarn Berry")
	}

	writeFile(t, filepath.Join(working, ".yarnrc.yml"), "nodeLinker: node-modules\n")
	if !IsYarnrcYamlFilePresent(session, working) {
		t.Error("a .yarnrc.yml in the working directory means Yarn Berry")
	}

	session, directory = newTestSession(t)
	writeFile(t, filepath.Join(directory, ".yarnrc.yml"), "")
	if !IsYarnrcYamlFilePresent(session, filepath.Join(directory, "elsewhere")) {
		t.Error("a .yarnrc.yml at the project root means Yarn Berry")
	}

	session, directory = newTestSession(t)
	session.MultiModuleProjectDirectory = filepath.Join(directory, "reactor")
	writeFile(t, filepath.Join(session.MultiModuleProjectDirectory, ".yarnrc.yml"), "")
	if !IsYarnrcYamlFilePresent(session, filepath.Join(directory, "elsewhere")) {
		t.Error("a .yarnrc.yml at the reactor root means Yarn Berry")
	}

	session, directory = newTestSession(t)
	session.ExecutionRootDirectory = filepath.Join(directory, "execroot")
	writeFile(t, filepath.Join(session.ExecutionRootDirectory, ".yarnrc.yml"), "")
	if !IsYarnrcYamlFilePresent(session, filepath.Join(directory, "elsewhere")) {
		t.Error("a .yarnrc.yml at the execution root means Yarn Berry")
	}
}

func TestRepositoryCacheResolverUsesTheArtifactLayout(t *testing.T) {
	repository := filepath.Join(t.TempDir(), "repository")
	resolver := NewRepositoryCacheResolver(repository)

	got := resolver.Resolve(frontend.NewClassifiedCacheDescriptor("node", "v18.0.0", "linux-x64", "tar.gz"))
	want := filepath.Join(repository, "com", "github", "eirslett", "node", "18.0.0",
		"node-18.0.0-linux-x64.tar.gz")
	if got != want {
		t.Errorf("resolved to %q, want %q", got, want)
	}

	got = resolver.Resolve(frontend.NewCacheDescriptor("npm", "8.6.0", "tar.gz"))
	want = filepath.Join(repository, "com", "github", "eirslett", "npm", "8.6.0", "npm-8.6.0.tar.gz")
	if got != want {
		t.Errorf("resolved to %q, want %q", got, want)
	}
}

func TestTrimLeadingVDropsOnlyTheVersionPrefix(t *testing.T) {
	for version, want := range map[string]string{
		"v1.2.3": "1.2.3",
		"1.2.3":  "1.2.3",
		"":       "",
		"v":      "",
	} {
		if got := trimLeadingV(version); got != want {
			t.Errorf("trimLeadingV(%q) = %q, want %q", version, got, want)
		}
	}
}
