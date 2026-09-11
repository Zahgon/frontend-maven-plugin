package frontend

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Windows takes different download URLs and a different on-disk layout, and lays
// a proxy script where the other platforms make a symbolic link. None of that
// needs Windows to exercise, only a platform that says it is Windows.

func windowsInstallConfig(t *testing.T) (InstallConfig, string) {
	t.Helper()
	directory := t.TempDir()
	platform := GuessPlatformWith(OSWindows, ArchX64, func() bool { return false })
	return NewInstallConfig(directory, directory,
		NewDirectoryCacheResolver(childFile(directory, "cache")), platform), directory
}

// windowsDistServer serves the two shapes a Windows Node.js download takes: the
// bare node.exe and the full archive.
func windowsDistServer(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	writeFixture(t, filepath.Join(root, "node", testNodeVersion, "win-x64", "node.exe"),
		[]byte("MZ fake node"))

	var buffer bytes.Buffer
	zipWriter := zip.NewWriter(&buffer)
	archiveDir := "node-" + testNodeVersion + "-win-x64"
	for name, body := range map[string]string{
		archiveDir + "/node.exe":                               "MZ fake node",
		archiveDir + "/node_modules/npm/package.json":          `{"version":"9.9.9"}`,
		archiveDir + "/node_modules/npm/bin/npm-cli.js":        "// npm-cli",
		archiveDir + "/node_modules/pnpm/package.json":         `{"version":"9.9.9"}`,
		archiveDir + "/node_modules/pnpm/bin/pnpm.cjs":         "// pnpm",
		archiveDir + "/node_modules/corepack/package.json":     `{"version":"0.17.0"}`,
		archiveDir + "/node_modules/corepack/dist/corepack.js": "// corepack",
	} {
		writer, err := zipWriter.Create(name)
		requireNoError(t, err)
		_, err = writer.Write([]byte(body))
		requireNoError(t, err)
	}
	requireNoError(t, zipWriter.Close())
	writeFixture(t, filepath.Join(root, "node", testNodeVersion, archiveDir+".zip"), buffer.Bytes())

	return startFileServer(t, root)
}

func TestNodeInstallerDownloadsTheBareBinaryOnWindows(t *testing.T) {
	config, directory := windowsInstallConfig(t)
	server := windowsDistServer(t)

	requireNoError(t, NewNodeInstaller(config, NewDefaultArchiveExtractor(),
		NewDefaultFileDownloader(NewProxyConfig(nil))).
		SetNodeVersion(testNodeVersion).
		SetNodeDownloadRoot(server+"/node/").
		Install())

	if !isFile(filepath.Join(directory, "node", "node.exe")) {
		t.Errorf("node.exe was not installed: %v", treeOf(t, directory))
	}
	// The bare binary is cached under the "exe" extension, not an archive one.
	if !isFile(filepath.Join(directory, "cache", "node-"+testNodeVersion+"-win-x64.exe")) {
		t.Errorf("the binary was not cached as an exe: %v", treeOf(t, directory))
	}
}

func TestNodeInstallerReportsAFailedWindowsBinaryDownload(t *testing.T) {
	config, _ := windowsInstallConfig(t)
	server := windowsDistServer(t)

	err := NewNodeInstaller(config, NewDefaultArchiveExtractor(),
		NewDefaultFileDownloader(NewProxyConfig(nil))).
		SetNodeVersion("v99.0.0").
		SetNodeDownloadRoot(server + "/node/").
		Install()

	if err == nil || !strings.HasPrefix(err.Error(), "Could not download Node.js from: ") {
		t.Fatalf("error = %v, want the Windows download message", err)
	}
}

func TestNodeInstallerUnpacksTheWindowsArchiveWhenNpmIsProvided(t *testing.T) {
	config, directory := windowsInstallConfig(t)
	server := windowsDistServer(t)

	requireNoError(t, NewNodeInstaller(config, NewDefaultArchiveExtractor(),
		NewDefaultFileDownloader(NewProxyConfig(nil))).
		SetNodeVersion(testNodeVersion).
		SetNodeDownloadRoot(server+"/node/").
		SetNpmVersion("provided").
		Install())

	if !isFile(filepath.Join(directory, "node", "node.exe")) {
		t.Errorf("node.exe was not installed: %v", treeOf(t, directory))
	}
	if !isFile(filepath.Join(directory, "node", "node_modules", "npm", "bin", "npm-cli.js")) {
		t.Errorf("the bundled node_modules was not copied: %v", treeOf(t, directory))
	}
}

func TestNodeInstallerReportsAFailedWindowsArchiveDownload(t *testing.T) {
	config, _ := windowsInstallConfig(t)
	server := windowsDistServer(t)

	err := NewNodeInstaller(config, NewDefaultArchiveExtractor(),
		NewDefaultFileDownloader(NewProxyConfig(nil))).
		SetNodeVersion("v99.0.0").
		SetNodeDownloadRoot(server + "/node/").
		SetNpmVersion("provided").
		Install()

	if err == nil || err.Error() != "Could not download Node.js" {
		t.Fatalf("error = %v, want %q", err, "Could not download Node.js")
	}
}

func TestPnpmInstallerWritesAProxyScriptOnWindows(t *testing.T) {
	config, directory := windowsInstallConfig(t)
	server := windowsDistServer(t)
	requireNoError(t, NewNodeInstaller(config, NewDefaultArchiveExtractor(),
		NewDefaultFileDownloader(NewProxyConfig(nil))).
		SetNodeVersion(testNodeVersion).
		SetNodeDownloadRoot(server+"/node/").
		SetNpmVersion("provided").
		Install())

	requireNoError(t, NewPnpmInstaller(config, NewDefaultArchiveExtractor(),
		NewDefaultFileDownloader(NewProxyConfig(nil))).
		SetPnpmVersion("v9.9.9").
		Install())

	script, err := os.ReadFile(filepath.Join(directory, "node", "pnpm.cmd"))
	requireNoError(t, err)
	body := string(script)
	for _, want := range []string{
		":: Created by frontend-maven-plugin, please don't edit manually.\r\n",
		"@ECHO OFF\r\n",
		"SETLOCAL\r\n",
		`SET "NODE_EXE=%~dp0\`,
		`SET "PNPM_CLI_JS=%~dp0\`,
		`"%NODE_EXE%" "%PNPM_CLI_JS%" %*`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("proxy script does not contain %q:\n%s", want, body)
		}
	}
}

func TestCorepackInstallerWritesAProxyScriptOnWindows(t *testing.T) {
	config, directory := windowsInstallConfig(t)
	server := windowsDistServer(t)
	requireNoError(t, NewNodeInstaller(config, NewDefaultArchiveExtractor(),
		NewDefaultFileDownloader(NewProxyConfig(nil))).
		SetNodeVersion(testNodeVersion).
		SetNodeDownloadRoot(server+"/node/").
		SetNpmVersion("provided").
		Install())
	requireNoError(t, NewCorepackInstaller(config, NewDefaultArchiveExtractor(),
		NewDefaultFileDownloader(NewProxyConfig(nil))).
		SetCorepackVersion("provided").
		Install())

	script, err := os.ReadFile(filepath.Join(directory, "node", "corepack.cmd"))
	requireNoError(t, err)
	if !strings.Contains(string(script), `SET "COREPACK_CLI_JS=%~dp0\`) {
		t.Errorf("proxy script does not point at corepack:\n%s", script)
	}
}

func TestWindowsInstallerSkipsLinkingWhenAScriptIsAlreadyThere(t *testing.T) {
	config, directory := windowsInstallConfig(t)
	writeFixture(t, filepath.Join(directory, "node", "pnpm.cmd"), []byte("existing"))
	writeFixture(t, filepath.Join(directory, "node", "node_modules", "pnpm", "package.json"),
		packageJSON(t, "7.9.0"))

	output := captureLog(t, 0, func() {
		requireNoError(t, NewPnpmInstaller(config, NewDefaultArchiveExtractor(),
			NewDefaultFileDownloader(NewProxyConfig(nil))).
			SetPnpmVersion("7.9.0").
			Install())
	})

	if !strings.Contains(output, "Existing pnpm executable found, skipping linking.") {
		t.Errorf("log does not report the existing script:\n%s", output)
	}
}

func TestWindowsInstallerFailsWithoutAnEntryPointToProxy(t *testing.T) {
	config, directory := windowsInstallConfig(t)
	writeFixture(t, filepath.Join(directory, "node", "node_modules", "pnpm", "package.json"),
		packageJSON(t, "7.9.0"))

	err := NewPnpmInstaller(config, NewDefaultArchiveExtractor(),
		NewDefaultFileDownloader(NewProxyConfig(nil))).
		SetPnpmVersion("7.9.0").
		Install()

	want := "Could not link to pnpm executable, no pnpm installation found."
	if err == nil || err.Error() != want {
		t.Fatalf("error = %v, want %q", err, want)
	}
}

func TestWindowsExecutorPathsUseBackslashes(t *testing.T) {
	config, directory := windowsInstallConfig(t)

	node := NewInstallNodeExecutorConfig(config)
	if got := node.NodePath(); got != directory+`\node\node.exe` {
		t.Errorf("node path = %q, want the Windows spelling", got)
	}
	yarn := NewInstallYarnExecutorConfig(config, false)
	if got := yarn.YarnPath(); got != directory+`\node\yarn\dist\bin\yarn.cmd` {
		t.Errorf("yarn path = %q, want the Windows spelling", got)
	}
	if got := yarn.NodePath(); got != node.NodePath() {
		t.Errorf("yarn's node path = %q, want %q", got, node.NodePath())
	}
	bun := NewInstallBunExecutorConfig(config)
	if got := bun.BunPath(); got != directory+`\bun\bun.exe` {
		t.Errorf("bun path = %q, want the Windows spelling", got)
	}
	if got := bun.NodePath(); got != node.NodePath() {
		t.Errorf("bun's node path = %q, want %q", got, node.NodePath())
	}
}
