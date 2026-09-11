package frontend

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// These exercise what used to be someone else's test suite: the download, the
// unpacking, the on-disk layout and the exact wording of every installer
// message.

func installNode(t *testing.T, config InstallConfig, server string, npmVersion string) error {
	t.Helper()
	return NewNodeInstaller(config, NewDefaultArchiveExtractor(), NewDefaultFileDownloader(NewProxyConfig(nil))).
		SetNodeVersion(testNodeVersion).
		SetNodeDownloadRoot(server + "/node/").
		SetNpmVersion(npmVersion).
		Install()
}

func TestNodeInstallerPlacesTheBinaryAndCachesTheArchive(t *testing.T) {
	config, directory := newTestInstallConfig(t)
	server := distServer(t)

	requireNoError(t, installNode(t, config, server, ""))

	want := []string{
		"d cache",
		"d node",
		"f cache/node-" + testNodeVersion + "-" + testNodeClassifier + ".tar.gz",
		"f node/node",
	}
	if got := treeOf(t, directory); !reflect.DeepEqual(want, got) {
		t.Errorf("installed tree = %v, want %v", got, want)
	}
	info, err := os.Stat(filepath.Join(directory, "node", "node"))
	requireNoError(t, err)
	if info.Mode().Perm()&0o111 != 0o111 {
		t.Errorf("node binary mode = %v, want it executable by owner, group and other",
			info.Mode().Perm())
	}
}

func TestNodeInstallerCopiesTheBundledNpmWhenProvided(t *testing.T) {
	config, directory := newTestInstallConfig(t)
	server := distServer(t)

	output := captureLog(t, 0, func() {
		requireNoError(t, installNode(t, config, server, "provided"))
	})

	if !strings.Contains(output, "Extracting NPM") {
		t.Errorf("log does not mention extracting NPM:\n%s", output)
	}
	for _, script := range []string{"npm", "npm.cmd", "npx"} {
		if !isFile(filepath.Join(directory, "node", script)) {
			t.Errorf("%s was not copied next to node", script)
		}
	}
	if !isFile(filepath.Join(directory, "node", "node_modules", "npm", "bin", "npm-cli.js")) {
		t.Error("the bundled npm module was not copied")
	}
}

func TestNodeInstallerRejectsProvidedNpmBeforeNodeFour(t *testing.T) {
	config, _ := newTestInstallConfig(t)
	server := distServer(t)

	err := NewNodeInstaller(config, NewDefaultArchiveExtractor(), NewDefaultFileDownloader(NewProxyConfig(nil))).
		SetNodeVersion("v3.9.9").
		SetNodeDownloadRoot(server + "/node/").
		SetNpmVersion("provided").
		Install()

	const want = "NPM version is 'provided' but Node didn't include NPM prior to v4.0.0"
	if err == nil || err.Error() != want {
		t.Fatalf("error = %v, want %q", err, want)
	}
}

func TestNPMInstallerRejectsProvidedNpmBeforeNodeFour(t *testing.T) {
	config, _ := newTestInstallConfig(t)

	err := NewNPMInstaller(config, NewDefaultArchiveExtractor(),
		NewDefaultFileDownloader(NewProxyConfig(nil))).
		SetNodeVersion("v3.9.9").
		SetNpmVersion("provided").
		Install()

	const want = "NPM version is 'provided' but Node didn't include NPM prior to v4.0.0"
	if err == nil || err.Error() != want {
		t.Fatalf("error = %v, want %q", err, want)
	}
}

func TestNodeInstallerWarnsAboutAVersionWithoutTheVPrefix(t *testing.T) {
	config, _ := newTestInstallConfig(t)
	server := distServer(t)

	output := captureLog(t, 0, func() {
		_ = NewNodeInstaller(config, NewDefaultArchiveExtractor(), NewDefaultFileDownloader(NewProxyConfig(nil))).
			SetNodeVersion("18.20.4").
			SetNodeDownloadRoot(server + "/node/").
			Install()
	})

	if !strings.Contains(output, "Node version does not start with naming convention 'v'.") {
		t.Errorf("log does not carry the naming warning:\n%s", output)
	}
}

func TestNodeInstallerReportsADownloadFailure(t *testing.T) {
	config, _ := newTestInstallConfig(t)
	server := distServer(t)

	err := NewNodeInstaller(config, NewDefaultArchiveExtractor(), NewDefaultFileDownloader(NewProxyConfig(nil))).
		SetNodeVersion("v99.0.0").
		SetNodeDownloadRoot(server + "/node/").
		Install()

	if err == nil || err.Error() != "Could not download Node.js" {
		t.Fatalf("error = %v, want %q", err, "Could not download Node.js")
	}
	if cause := errors.Unwrap(err); cause == nil || cause.Error() != "Got error code 404 from the server." {
		t.Fatalf("cause = %v, want the 404 message", cause)
	}
}

func TestNodeInstallerSkipsAnAlreadyInstalledVersion(t *testing.T) {
	config, directory := newTestInstallConfig(t)
	server := distServer(t)
	requireNoError(t, installNode(t, config, server, ""))

	output := captureLog(t, 0, func() {
		requireNoError(t, installNode(t, config, server, ""))
	})

	want := "Node " + testNodeVersion + " is already installed."
	if !strings.Contains(output, want) {
		t.Errorf("log does not report the version as installed:\n%s", output)
	}
	_ = directory
}

func TestNodeInstallerReinstallsADifferentVersion(t *testing.T) {
	config, directory := newTestInstallConfig(t)
	server := distServer(t)
	requireNoError(t, installNode(t, config, server, ""))

	output := captureLog(t, 0, func() {
		_ = NewNodeInstaller(config, NewDefaultArchiveExtractor(), NewDefaultFileDownloader(NewProxyConfig(nil))).
			SetNodeVersion("v20.0.0").
			SetNodeDownloadRoot(server + "/node/").
			Install()
	})

	want := "Node " + testNodeVersion + " was installed, but we need version v20.0.0"
	if !strings.Contains(output, want) {
		t.Errorf("log does not report the version mismatch:\n%s", output)
	}
	_ = directory
}

func TestNodeInstallerWarnsWhenTheVersionProbeFails(t *testing.T) {
	config, directory := newTestInstallConfig(t)
	server := distServer(t)
	requireNoError(t, installNode(t, config, server, ""))
	// A node that exits non-zero: the probe cannot answer, so the installer
	// reinstalls rather than trusting what is there.
	writeFixture(t, filepath.Join(directory, "node", "node"), []byte("#!/bin/sh\nexit 4\n"))
	requireNoError(t, os.Chmod(filepath.Join(directory, "node", "node"), 0o755))

	output := captureLog(t, 0, func() {
		requireNoError(t, installNode(t, config, server, ""))
	})

	if !strings.Contains(output, "Unable to determine current node version:") {
		t.Errorf("log does not report the failed probe:\n%s", output)
	}
}

func TestNPMInstallerUnpacksAndCopiesItsScripts(t *testing.T) {
	config, directory := newTestInstallConfig(t)
	server := distServer(t)
	requireNoError(t, installNode(t, config, server, ""))

	requireNoError(t, NewNPMInstaller(config, NewDefaultArchiveExtractor(),
		NewDefaultFileDownloader(NewProxyConfig(nil))).
		SetNodeVersion(testNodeVersion).
		SetNpmVersion(testNpmVersion).
		SetNpmDownloadRoot(server+"/npm/").
		Install())

	for _, path := range []string{
		"node/node_modules/npm/package.json",
		"node/node_modules/npm/bin/npm-cli.js",
		"node/npm",
		"node/npx",
	} {
		if !isFile(filepath.Join(directory, filepath.FromSlash(path))) {
			t.Errorf("%s is missing from the installation", path)
		}
	}
	if exists(filepath.Join(directory, "node", "node_modules", "package")) {
		t.Error("the extracted package directory was not renamed to npm")
	}
}

func TestNPMInstallerSkipsAnAlreadyInstalledVersion(t *testing.T) {
	config, _ := newTestInstallConfig(t)
	server := distServer(t)
	installer := func() *NPMInstaller {
		return NewNPMInstaller(config, NewDefaultArchiveExtractor(),
			NewDefaultFileDownloader(NewProxyConfig(nil))).
			SetNodeVersion(testNodeVersion).
			SetNpmVersion(testNpmVersion).
			SetNpmDownloadRoot(server + "/npm/")
	}
	requireNoError(t, installer().Install())

	output := captureLog(t, 0, func() { requireNoError(t, installer().Install()) })

	if !strings.Contains(output, "NPM "+testNpmVersion+" is already installed.") {
		t.Errorf("log does not report npm as installed:\n%s", output)
	}
}

func TestNPMInstallerReportsAVersionMismatch(t *testing.T) {
	config, _ := newTestInstallConfig(t)
	server := distServer(t)
	requireNoError(t, NewNPMInstaller(config, NewDefaultArchiveExtractor(),
		NewDefaultFileDownloader(NewProxyConfig(nil))).
		SetNodeVersion(testNodeVersion).
		SetNpmVersion(testNpmVersion).
		SetNpmDownloadRoot(server+"/npm/").
		Install())

	output := captureLog(t, 0, func() {
		_ = NewNPMInstaller(config, NewDefaultArchiveExtractor(),
			NewDefaultFileDownloader(NewProxyConfig(nil))).
			SetNodeVersion(testNodeVersion).
			SetNpmVersion("9.0.0").
			SetNpmDownloadRoot(server + "/npm/").
			Install()
	})

	want := "NPM " + testNpmVersion + " was installed, but we need version 9.0.0"
	if !strings.Contains(output, want) {
		t.Errorf("log does not report the version mismatch:\n%s", output)
	}
}

func TestNPMInstallerReportsAPackageJSONWithoutAVersion(t *testing.T) {
	config, directory := newTestInstallConfig(t)
	server := distServer(t)
	writeFixture(t, filepath.Join(directory, "node", "node_modules", "npm", "package.json"), []byte("{}"))

	output := captureLog(t, 0, func() {
		requireNoError(t, NewNPMInstaller(config, NewDefaultArchiveExtractor(),
			NewDefaultFileDownloader(NewProxyConfig(nil))).
			SetNodeVersion(testNodeVersion).
			SetNpmVersion(testNpmVersion).
			SetNpmDownloadRoot(server+"/npm/").
			Install())
	})

	if !strings.Contains(output, "Could not read NPM version from package.json") {
		t.Errorf("log does not report the missing version:\n%s", output)
	}
}

func TestNPMInstallerFailsOnAnUnreadablePackageJSON(t *testing.T) {
	config, directory := newTestInstallConfig(t)
	server := distServer(t)
	writeFixture(t, filepath.Join(directory, "node", "node_modules", "npm", "package.json"), []byte("{not json"))

	err := NewNPMInstaller(config, NewDefaultArchiveExtractor(),
		NewDefaultFileDownloader(NewProxyConfig(nil))).
		SetNodeVersion(testNodeVersion).
		SetNpmVersion(testNpmVersion).
		SetNpmDownloadRoot(server + "/npm/").
		Install()

	if err == nil || err.Error() != "Could not read package.json" {
		t.Fatalf("error = %v, want %q", err, "Could not read package.json")
	}
}

func TestNPMInstallerDoesNothingWhenNpmIsProvided(t *testing.T) {
	config, directory := newTestInstallConfig(t)
	server := distServer(t)
	requireNoError(t, installNode(t, config, server, "provided"))

	requireNoError(t, NewNPMInstaller(config, NewDefaultArchiveExtractor(),
		NewDefaultFileDownloader(NewProxyConfig(nil))).
		SetNodeVersion(testNodeVersion).
		SetNpmVersion("provided").
		Install())

	if isFile(filepath.Join(directory, "cache", "npm-provided.tar.gz")) {
		t.Error("a provided npm must not be downloaded")
	}
}

func TestNPMInstallerDeletesACorruptedArchive(t *testing.T) {
	config, directory := newTestInstallConfig(t)
	server := distServer(t)

	output := captureLog(t, 0, func() {
		err := NewNPMInstaller(config, NewDefaultArchiveExtractor(),
			NewDefaultFileDownloader(NewProxyConfig(nil))).
			SetNodeVersion(testNodeVersion).
			SetNpmVersion("9.9.9").
			SetNpmDownloadRoot(server + "/truncated/").
			Install()
		if err == nil || err.Error() != "Could not extract the npm archive" {
			t.Fatalf("error = %v, want %q", err, "Could not extract the npm archive")
		}
	})

	if !strings.Contains(output, "is corrupted and will be deleted") {
		t.Errorf("log does not report the corrupted archive:\n%s", output)
	}
	if exists(filepath.Join(directory, "cache", "npm-9.9.9.tar.gz")) {
		t.Error("the corrupted archive was not deleted")
	}
}

func TestYarnInstallerRenamesTheVersionedRootToDist(t *testing.T) {
	config, directory := newTestInstallConfig(t)
	server := distServer(t)

	requireNoError(t, NewYarnInstaller(config, NewDefaultArchiveExtractor(),
		NewDefaultFileDownloader(NewProxyConfig(nil))).
		SetYarnVersion(testYarnVersion).
		SetYarnDownloadRoot(server+"/yarn/").
		Install())

	if !isFile(filepath.Join(directory, "node", "yarn", "dist", "bin", "yarn")) {
		t.Errorf("yarn was not installed under dist: %v", treeOf(t, directory))
	}
}

func TestYarnInstallerAcceptsAnArchiveThatAlreadyUnpacksIntoDist(t *testing.T) {
	config, directory := newTestInstallConfig(t)
	server := distServer(t)

	requireNoError(t, NewYarnInstaller(config, NewDefaultArchiveExtractor(),
		NewDefaultFileDownloader(NewProxyConfig(nil))).
		SetYarnVersion("v2.0.0").
		SetYarnDownloadRoot(server+"/yarn/").
		Install())

	if !isFile(filepath.Join(directory, "node", "yarn", "dist", "bin", "yarn")) {
		t.Error("yarn was not installed under dist")
	}
}

func TestYarnInstallerFailsWhenTheArchiveHasNoDistributionDirectory(t *testing.T) {
	config, _ := newTestInstallConfig(t)
	server := distServer(t)

	err := NewYarnInstaller(config, NewDefaultArchiveExtractor(),
		NewDefaultFileDownloader(NewProxyConfig(nil))).
		SetYarnVersion("v3.0.0").
		SetYarnDownloadRoot(server + "/yarn/").
		Install()

	if err == nil || err.Error() != "Could not extract the Yarn archive" {
		t.Fatalf("error = %v, want %q", err, "Could not extract the Yarn archive")
	}
	cause := errors.Unwrap(err)
	if cause == nil || cause.Error() != "Could not find yarn distribution directory during extract" {
		t.Fatalf("cause = %v, want the missing-distribution message", cause)
	}
}

func TestYarnInstallerRequiresTheVPrefix(t *testing.T) {
	config, _ := newTestInstallConfig(t)
	server := distServer(t)

	err := NewYarnInstaller(config, NewDefaultArchiveExtractor(),
		NewDefaultFileDownloader(NewProxyConfig(nil))).
		SetYarnVersion("1.22.19").
		SetYarnDownloadRoot(server + "/yarn/").
		Install()

	if err == nil || err.Error() != "Yarn version has to start with prefix 'v'." {
		t.Fatalf("error = %v, want the prefix message", err)
	}
}

func TestYarnInstallerReportsADownloadFailure(t *testing.T) {
	config, _ := newTestInstallConfig(t)
	server := distServer(t)

	err := NewYarnInstaller(config, NewDefaultArchiveExtractor(),
		NewDefaultFileDownloader(NewProxyConfig(nil))).
		SetYarnVersion("v9.9.9").
		SetYarnDownloadRoot(server + "/yarn/").
		Install()

	if err == nil || err.Error() != "Could not download Yarn" {
		t.Fatalf("error = %v, want %q", err, "Could not download Yarn")
	}
}

func TestYarnInstallerSkipsAnAlreadyInstalledVersion(t *testing.T) {
	config, _ := newTestInstallConfig(t)
	server := distServer(t)
	requireNoError(t, installNode(t, config, server, ""))
	install := func(version string, berry bool) string {
		return captureLog(t, 0, func() {
			_ = NewYarnInstaller(config, NewDefaultArchiveExtractor(),
				NewDefaultFileDownloader(NewProxyConfig(nil))).
				SetYarnVersion(version).
				SetIsYarnBerry(berry).
				SetYarnDownloadRoot(server + "/yarn/").
				Install()
		})
	}
	install(testYarnVersion, false)

	output := install(testYarnVersion, false)
	if !strings.Contains(output, "Yarn 1.22.19 is already installed.") {
		t.Errorf("log does not report yarn as installed:\n%s", output)
	}

	mismatch := install("v1.0.0", false)
	if !strings.Contains(mismatch, "Yarn 1.22.19 was installed, but we need version v1.0.0") {
		t.Errorf("log does not report the version mismatch:\n%s", mismatch)
	}
}

func TestYarnInstallerAcceptsAnyBerryMajorVersion(t *testing.T) {
	config, _ := newTestInstallConfig(t)
	server := distServer(t)
	requireNoError(t, installNode(t, config, server, ""))
	requireNoError(t, NewYarnInstaller(config, NewDefaultArchiveExtractor(),
		NewDefaultFileDownloader(NewProxyConfig(nil))).
		SetYarnVersion("v2.0.0").
		SetYarnDownloadRoot(server+"/yarn/").
		Install())

	output := captureLog(t, 0, func() {
		requireNoError(t, NewYarnInstaller(config, NewDefaultArchiveExtractor(),
			NewDefaultFileDownloader(NewProxyConfig(nil))).
			SetYarnVersion("v4.1.1").
			SetIsYarnBerry(true).
			SetYarnDownloadRoot(server+"/yarn/").
			Install())
	})

	if !strings.Contains(output, "Yarn Berry 2.0.0 is installed.") {
		t.Errorf("log does not accept the installed Berry release:\n%s", output)
	}
}

func TestPnpmInstallerLinksItsExecutable(t *testing.T) {
	config, directory := newTestInstallConfig(t)
	server := distServer(t)

	requireNoError(t, NewPnpmInstaller(config, NewDefaultArchiveExtractor(),
		NewDefaultFileDownloader(NewProxyConfig(nil))).
		SetNodeVersion(testNodeVersion).
		SetPnpmVersion("v"+testPnpmVersion).
		SetPnpmDownloadRoot(server+"/pnpm/").
		Install())

	link := filepath.Join(directory, "node", "pnpm")
	info, err := os.Lstat(link)
	requireNoError(t, err)
	if info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("%s is not a symbolic link", link)
	}
	target, err := os.Readlink(link)
	requireNoError(t, err)
	if !strings.HasSuffix(target, filepath.FromSlash("node/node_modules/pnpm/bin/pnpm.cjs")) {
		t.Errorf("link points at %q, want the pnpm entry point", target)
	}
}

func TestPnpmInstallerSkipsLinkingWhenAnExecutableIsAlreadyThere(t *testing.T) {
	config, directory := newTestInstallConfig(t)
	server := distServer(t)
	requireNoError(t, NewPnpmInstaller(config, NewDefaultArchiveExtractor(),
		NewDefaultFileDownloader(NewProxyConfig(nil))).
		SetPnpmVersion(testPnpmVersion).
		SetPnpmDownloadRoot(server+"/pnpm/").
		Install())

	output := captureLog(t, 0, func() {
		requireNoError(t, NewPnpmInstaller(config, NewDefaultArchiveExtractor(),
			NewDefaultFileDownloader(NewProxyConfig(nil))).
			SetPnpmVersion(testPnpmVersion).
			SetPnpmDownloadRoot(server+"/pnpm/").
			Install())
	})

	if !strings.Contains(output, "PNPM "+testPnpmVersion+" is already installed.") {
		t.Errorf("log does not report pnpm as installed:\n%s", output)
	}
	if !strings.Contains(output, "Existing pnpm executable found, skipping linking.") {
		t.Errorf("log does not report the existing executable:\n%s", output)
	}
	_ = directory
}

func TestPnpmInstallerFailsWithoutAnInstallation(t *testing.T) {
	config, directory := newTestInstallConfig(t)
	// A package.json that satisfies the version check without any entry point
	// behind it, which is the shape a half-deleted installation leaves.
	writeFixture(t, filepath.Join(directory, "node", "node_modules", "pnpm", "package.json"),
		packageJSON(t, testPnpmVersion))

	err := NewPnpmInstaller(config, NewDefaultArchiveExtractor(),
		NewDefaultFileDownloader(NewProxyConfig(nil))).
		SetPnpmVersion(testPnpmVersion).
		Install()

	want := "Could not link to pnpm executable, no pnpm installation found."
	if err == nil || err.Error() != want {
		t.Fatalf("error = %v, want %q", err, want)
	}
}

func TestPnpmInstallerReportsADownloadFailure(t *testing.T) {
	config, _ := newTestInstallConfig(t)
	server := distServer(t)

	err := NewPnpmInstaller(config, NewDefaultArchiveExtractor(),
		NewDefaultFileDownloader(NewProxyConfig(nil))).
		SetPnpmVersion(testPnpmVersion).
		SetPnpmDownloadRoot(server + "/missing/").
		Install()

	if err == nil || err.Error() != "Could not download pnpm" {
		t.Fatalf("error = %v, want %q", err, "Could not download pnpm")
	}
}

func TestPnpmInstallerPrefersTheEsmEntryPoint(t *testing.T) {
	config, directory := newTestInstallConfig(t)
	server := distServer(t)
	requireNoError(t, NewPnpmInstaller(config, NewDefaultArchiveExtractor(),
		NewDefaultFileDownloader(NewProxyConfig(nil))).
		SetPnpmVersion(testPnpmVersion).
		SetPnpmDownloadRoot(server+"/pnpm/").
		Install())
	writeFixture(t, filepath.Join(directory, "node", "node_modules", "pnpm", "bin", "pnpm.mjs"),
		[]byte("// pnpm mjs\n"))

	executable := NewInstallNodeExecutorConfig(config).PnpmExecutablePath()

	if !strings.HasSuffix(executable, "pnpm.mjs") {
		t.Errorf("executable path = %q, want the .mjs entry point", executable)
	}
}

func TestCorepackInstallerLinksItsExecutable(t *testing.T) {
	config, directory := newTestInstallConfig(t)
	server := distServer(t)

	requireNoError(t, NewCorepackInstaller(config, NewDefaultArchiveExtractor(),
		NewDefaultFileDownloader(NewProxyConfig(nil))).
		SetNodeVersion(testNodeVersion).
		SetCorepackVersion(testCorepackVer).
		SetCorepackDownloadRoot(server+"/corepack/").
		Install())

	link := filepath.Join(directory, "node", "corepack")
	info, err := os.Lstat(link)
	requireNoError(t, err)
	if info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("%s is not a symbolic link", link)
	}
}

func TestCorepackInstallerTrustsTheBundledCopyWhenProvided(t *testing.T) {
	config, directory := newTestInstallConfig(t)
	server := distServer(t)
	requireNoError(t, installNode(t, config, server, "provided"))

	requireNoError(t, NewCorepackInstaller(config, NewDefaultArchiveExtractor(),
		NewDefaultFileDownloader(NewProxyConfig(nil))).
		SetCorepackVersion("provided").
		SetCorepackDownloadRoot(server+"/corepack/").
		Install())

	if exists(filepath.Join(directory, "cache", "corepack-provided.tar.gz")) {
		t.Error("a provided corepack must not be downloaded")
	}
	if !exists(filepath.Join(directory, "node", "corepack")) {
		t.Error("the corepack launcher was not linked")
	}
}

func TestCorepackInstallerReportsAVersionMismatch(t *testing.T) {
	config, _ := newTestInstallConfig(t)
	server := distServer(t)
	requireNoError(t, NewCorepackInstaller(config, NewDefaultArchiveExtractor(),
		NewDefaultFileDownloader(NewProxyConfig(nil))).
		SetCorepackVersion(testCorepackVer).
		SetCorepackDownloadRoot(server+"/corepack/").
		Install())

	output := captureLog(t, 0, func() {
		_ = NewCorepackInstaller(config, NewDefaultArchiveExtractor(),
			NewDefaultFileDownloader(NewProxyConfig(nil))).
			SetCorepackVersion("0.20.0").
			SetCorepackDownloadRoot(server + "/corepack/").
			Install()
	})

	want := "corepack " + testCorepackVer + " was installed, but we need version 0.20.0"
	if !strings.Contains(output, want) {
		t.Errorf("log does not report the version mismatch:\n%s", output)
	}
}

func TestBunInstallerPlacesTheBinary(t *testing.T) {
	config, directory := newTestInstallConfig(t)
	server := distServer(t)

	requireNoError(t, NewBunInstaller(config, NewDefaultArchiveExtractor(),
		NewDefaultFileDownloader(NewProxyConfig(nil))).
		SetBunVersion(testBunVersion).
		SetBunDownloadRoot(server+"/bun/").
		Install())

	binary := filepath.Join(directory, "bun", "bun")
	info, err := os.Stat(binary)
	requireNoError(t, err)
	if info.Mode().Perm()&0o111 != 0o111 {
		t.Errorf("bun binary mode = %v, want it executable by owner, group and other",
			info.Mode().Perm())
	}
	if exists(filepath.Join(directory, filepath.Base(createBunTargetArchitecturePath()))) {
		t.Error("the extracted architecture directory was not cleaned up")
	}
}

func TestBunInstallerSkipsAnAlreadyInstalledVersion(t *testing.T) {
	config, _ := newTestInstallConfig(t)
	server := distServer(t)
	install := func(version string) string {
		return captureLog(t, 0, func() {
			_ = NewBunInstaller(config, NewDefaultArchiveExtractor(),
				NewDefaultFileDownloader(NewProxyConfig(nil))).
				SetBunVersion(version).
				SetBunDownloadRoot(server + "/bun/").
				Install()
		})
	}
	install(testBunVersion)

	if output := install(testBunVersion); !strings.Contains(output, "Bun 1.0.0 is already installed.") {
		t.Errorf("log does not report bun as installed:\n%s", output)
	}
	if output := install("v2.0.0"); !strings.Contains(output, "Bun 1.0.0 was installed, but we need version v2.0.0") {
		t.Errorf("log does not report the version mismatch:\n%s", output)
	}
}

func TestBunInstallerWarnsAboutAVersionWithoutTheVPrefix(t *testing.T) {
	config, _ := newTestInstallConfig(t)
	server := distServer(t)

	output := captureLog(t, 0, func() {
		_ = NewBunInstaller(config, NewDefaultArchiveExtractor(),
			NewDefaultFileDownloader(NewProxyConfig(nil))).
			SetBunVersion("1.0.0").
			SetBunDownloadRoot(server + "/bun/").
			Install()
	})

	if !strings.Contains(output, "Bun version does not start with naming convention 'v'.") {
		t.Errorf("log does not carry the naming warning:\n%s", output)
	}
}

func TestBunInstallerReportsADownloadFailure(t *testing.T) {
	config, _ := newTestInstallConfig(t)
	server := distServer(t)

	err := NewBunInstaller(config, NewDefaultArchiveExtractor(),
		NewDefaultFileDownloader(NewProxyConfig(nil))).
		SetBunVersion("v9.9.9").
		SetBunDownloadRoot(server + "/bun/").
		Install()

	if err == nil || err.Error() != "Could not download bun" {
		t.Fatalf("error = %v, want %q", err, "Could not download bun")
	}
}

func TestBunInstallerDefaultsItsDownloadRoot(t *testing.T) {
	installer := NewBunInstaller(nil, nil, nil).SetBunVersion(testBunVersion)

	want := DefaultBunDownloadRoot + "bun-" + testBunVersion + createBunTargetArchitecturePath() + ".zip"
	if got := installer.createDownloadURL(); got != want {
		t.Errorf("download URL = %q, want %q", got, want)
	}
}

func TestInstallersAcceptCredentialsAndHeaders(t *testing.T) {
	config, _ := newTestInstallConfig(t)
	headers := map[string]string{"X-Probe": "yes"}

	// The setters are fluent: the original returns `this`, so a chain has to
	// come back as the very same installer, not merely as something non-nil.
	node := NewNodeInstaller(config, nil, nil)
	if got := node.SetUserName("u").SetPassword("p").SetHTTPHeaders(headers); got != node {
		t.Error("NodeInstaller setters did not return the same installer")
	}
	npm := NewNPMInstaller(config, nil, nil)
	if got := npm.SetUserName("u").SetPassword("p").SetHTTPHeaders(headers); got != npm {
		t.Error("NPMInstaller setters did not return the same installer")
	}
	yarn := NewYarnInstaller(config, nil, nil)
	if got := yarn.SetUserName("u").SetPassword("p").SetHTTPHeaders(headers); got != yarn {
		t.Error("YarnInstaller setters did not return the same installer")
	}
	pnpm := NewPnpmInstaller(config, nil, nil)
	if got := pnpm.SetNodeVersion("v18").SetUserName("u").SetPassword("p").
		SetHTTPHeaders(headers); got != pnpm {
		t.Error("PnpmInstaller setters did not return the same installer")
	}
	corepack := NewCorepackInstaller(config, nil, nil)
	if got := corepack.SetNodeVersion("v18").SetUserName("u").SetPassword("p").
		SetHTTPHeaders(headers); got != corepack {
		t.Error("CorepackInstaller setters did not return the same installer")
	}
	bun := NewBunInstaller(config, nil, nil)
	if got := bun.SetUserName("u").SetPassword("p").SetHTTPHeaders(headers); got != bun {
		t.Error("BunInstaller setters did not return the same installer")
	}
}
