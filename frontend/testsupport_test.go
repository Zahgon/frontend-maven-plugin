package frontend

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/eirslett/frontend-maven-plugin/internal/logging"
)

// Shared fixtures for the tests that exercise the download-and-unpack path the
// original delegated to commons-compress, commons-io and httpclient.

const (
	testNodeVersion    = "v18.20.4"
	testNodeClassifier = "linux-x64"
	testNodeDir        = "node-" + testNodeVersion + "-" + testNodeClassifier
	testNpmVersion     = "8.6.0"
	testPnpmVersion    = "7.9.0"
	testCorepackVer    = "0.19.0"
	testYarnVersion    = "v1.22.19"
	testBunVersion     = "v1.0.0"
)

// testPlatform is a fixed Linux/x64 platform, so every download URL and unpacked
// directory name is the same on every machine the suite runs on.
func testPlatform() *Platform {
	return GuessPlatformWith(OSLinux, ArchX64, func() bool { return false })
}

type tarEntry struct {
	name       string
	body       []byte
	directory  bool
	executable bool
}

func writeTarGz(t *testing.T, path string, entries []tarEntry) {
	t.Helper()
	var buffer bytes.Buffer
	gzipWriter := gzip.NewWriter(&buffer)
	tarWriter := tar.NewWriter(gzipWriter)
	for _, entry := range entries {
		header := &tar.Header{Name: entry.name, Mode: 0o644, Size: int64(len(entry.body))}
		if entry.directory {
			header.Typeflag = tar.TypeDir
			header.Mode = 0o755
			header.Size = 0
		} else if entry.executable {
			header.Mode = 0o755
		}
		if err := tarWriter.WriteHeader(header); err != nil {
			t.Fatal(err)
		}
		if !entry.directory {
			if _, err := tarWriter.Write(entry.body); err != nil {
				t.Fatal(err)
			}
		}
	}
	if err := tarWriter.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gzipWriter.Close(); err != nil {
		t.Fatal(err)
	}
	writeFixture(t, path, buffer.Bytes())
}

func writeFixture(t *testing.T, path string, body []byte) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o777); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, body, 0o666); err != nil {
		t.Fatal(err)
	}
}

func packageJSON(t *testing.T, version string) []byte {
	t.Helper()
	body, err := json.Marshal(map[string]string{"version": version})
	if err != nil {
		t.Fatal(err)
	}
	return body
}

// distServer lays out one release of every tool this plugin installs and serves
// it over HTTP, returning the server's base URL.
func distServer(t *testing.T) string {
	t.Helper()
	root := t.TempDir()

	nodeEntries := []tarEntry{
		{name: testNodeDir + "/", directory: true},
		{name: testNodeDir + "/bin/", directory: true},
		{name: testNodeDir + "/bin/node", body: []byte("#!/bin/sh\necho " + testNodeVersion + "\n"), executable: true},
		{name: testNodeDir + "/lib/", directory: true},
		{name: testNodeDir + "/lib/node_modules/", directory: true},
		{name: testNodeDir + "/lib/node_modules/npm/", directory: true},
		{name: testNodeDir + "/lib/node_modules/npm/package.json", body: packageJSON(t, "9.9.9")},
		{name: testNodeDir + "/lib/node_modules/npm/bin/", directory: true},
		{name: testNodeDir + "/lib/node_modules/npm/bin/npm", body: []byte("#!/bin/sh\necho npm\n"), executable: true},
		{name: testNodeDir + "/lib/node_modules/npm/bin/npm.cmd", body: []byte("@echo npm\r\n")},
		{name: testNodeDir + "/lib/node_modules/npm/bin/npx", body: []byte("#!/bin/sh\necho npx\n"), executable: true},
		{name: testNodeDir + "/lib/node_modules/npm/bin/npm-cli.js", body: []byte("// npm-cli\n")},
		{name: testNodeDir + "/lib/node_modules/npm/bin/npx-cli.js", body: []byte("// npx-cli\n")},
		{name: testNodeDir + "/lib/node_modules/corepack/", directory: true},
		{name: testNodeDir + "/lib/node_modules/corepack/package.json", body: packageJSON(t, "0.17.0")},
		{name: testNodeDir + "/lib/node_modules/corepack/dist/", directory: true},
		{name: testNodeDir + "/lib/node_modules/corepack/dist/corepack.js", body: []byte("// corepack\n")},
	}
	writeTarGz(t, filepath.Join(root, "node", testNodeVersion, testNodeDir+".tar.gz"), nodeEntries)

	writeTarGz(t, filepath.Join(root, "npm", "npm-"+testNpmVersion+".tgz"), []tarEntry{
		{name: "package/", directory: true},
		{name: "package/package.json", body: packageJSON(t, testNpmVersion)},
		{name: "package/bin/", directory: true},
		{name: "package/bin/npm", body: []byte("#!/bin/sh\necho npm\n"), executable: true},
		{name: "package/bin/npm.cmd", body: []byte("@echo npm\r\n")},
		{name: "package/bin/npx", body: []byte("#!/bin/sh\necho npx\n"), executable: true},
		{name: "package/bin/npm-cli.js", body: []byte("// npm-cli\n")},
		{name: "package/bin/npx-cli.js", body: []byte("// npx-cli\n")},
	})

	writeTarGz(t, filepath.Join(root, "pnpm", "pnpm-"+testPnpmVersion+".tgz"), []tarEntry{
		{name: "package/", directory: true},
		{name: "package/package.json", body: packageJSON(t, testPnpmVersion)},
		{name: "package/bin/", directory: true},
		{name: "package/bin/pnpm.cjs", body: []byte("// pnpm cjs\n")},
	})

	writeTarGz(t, filepath.Join(root, "corepack", "corepack-"+testCorepackVer+".tgz"), []tarEntry{
		{name: "package/", directory: true},
		{name: "package/package.json", body: packageJSON(t, testCorepackVer)},
		{name: "package/dist/", directory: true},
		{name: "package/dist/corepack.js", body: []byte("// corepack\n")},
	})

	writeTarGz(t, filepath.Join(root, "yarn", testYarnVersion, "yarn-"+testYarnVersion+".tar.gz"), []tarEntry{
		{name: "yarn-" + testYarnVersion + "/", directory: true},
		{name: "yarn-" + testYarnVersion + "/bin/", directory: true},
		{name: "yarn-" + testYarnVersion + "/bin/yarn", body: []byte("#!/bin/sh\necho 1.22.19\n"), executable: true},
	})

	// A Yarn archive that already unpacks into "dist", and one that unpacks into
	// neither shape.
	writeTarGz(t, filepath.Join(root, "yarn", "v2.0.0", "yarn-v2.0.0.tar.gz"), []tarEntry{
		{name: "dist/", directory: true},
		{name: "dist/bin/", directory: true},
		{name: "dist/bin/yarn", body: []byte("#!/bin/sh\necho 2.0.0\n"), executable: true},
	})
	writeTarGz(t, filepath.Join(root, "yarn", "v3.0.0", "yarn-v3.0.0.tar.gz"), []tarEntry{
		{name: "elsewhere/", directory: true},
		{name: "elsewhere/yarn", body: []byte("#!/bin/sh\n"), executable: true},
	})

	writeBunZip(t, filepath.Join(root, "bun", "bun-"+testBunVersion,
		filepath.Base(createBunTargetArchitecturePath())+".zip"))

	// A Node.js release from before Node bundled npm, for the "provided" check.
	writeTarGz(t, filepath.Join(root, "node", "v3.9.9", "node-v3.9.9-"+testNodeClassifier+".tar.gz"), []tarEntry{
		{name: "node-v3.9.9-" + testNodeClassifier + "/", directory: true},
		{name: "node-v3.9.9-" + testNodeClassifier + "/bin/", directory: true},
		{name: "node-v3.9.9-" + testNodeClassifier + "/bin/node",
			body: []byte("#!/bin/sh\necho v3.9.9\n"), executable: true},
	})

	// An archive whose download was cut off halfway, for the corrupted-download
	// path.
	writeFixture(t, filepath.Join(root, "truncated", "npm-9.9.9.tgz"), truncatedTarGz(t))

	return startFileServer(t, root)
}

// startFileServer serves a directory of fixtures over HTTP for the duration of
// one test.
func startFileServer(t *testing.T, root string) string {
	t.Helper()
	server := httptest.NewServer(http.FileServer(http.Dir(root)))
	t.Cleanup(server.Close)
	return server.URL
}

func writeBunZip(t *testing.T, path string) {
	t.Helper()
	var buffer bytes.Buffer
	zipWriter := zip.NewWriter(&buffer)
	directory := filepath.Base(createBunTargetArchitecturePath())
	if _, err := zipWriter.Create(directory + "/"); err != nil {
		t.Fatal(err)
	}
	header := &zip.FileHeader{Name: directory + "/bun", Method: zip.Deflate}
	header.SetMode(0o755)
	writer, err := zipWriter.CreateHeader(header)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := writer.Write([]byte("#!/bin/sh\necho 1.0.0\n")); err != nil {
		t.Fatal(err)
	}
	if err := zipWriter.Close(); err != nil {
		t.Fatal(err)
	}
	writeFixture(t, path, buffer.Bytes())
}

// truncatedTarGz is a well-formed tar.gz cut off halfway, which is what an
// interrupted download leaves in the cache.
func truncatedTarGz(t *testing.T) []byte {
	t.Helper()
	var buffer bytes.Buffer
	gzipWriter := gzip.NewWriter(&buffer)
	tarWriter := tar.NewWriter(gzipWriter)
	body := bytes.Repeat([]byte("payload-"), 65536)
	header := &tar.Header{Name: "package/big.bin", Mode: 0o644, Size: int64(len(body))}
	if err := tarWriter.WriteHeader(header); err != nil {
		t.Fatal(err)
	}
	if _, err := tarWriter.Write(body); err != nil {
		t.Fatal(err)
	}
	if err := tarWriter.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gzipWriter.Close(); err != nil {
		t.Fatal(err)
	}
	return buffer.Bytes()[:len(buffer.Bytes())/2]
}

// newTestInstallConfig points an installer at a fresh directory on the fixed
// test platform.
func newTestInstallConfig(t *testing.T) (InstallConfig, string) {
	t.Helper()
	directory := t.TempDir()
	return NewInstallConfig(directory, directory,
		NewDirectoryCacheResolver(childFile(directory, "cache")), testPlatform()), directory
}

// captureLog collects everything logged while fn runs.
func captureLog(t *testing.T, level logging.Level, fn func()) string {
	t.Helper()
	var buffer bytes.Buffer
	previousOut := logging.SetOutput(&buffer)
	previousLevel := logging.SetLevel(level)
	defer func() {
		logging.SetOutput(previousOut)
		logging.SetLevel(previousLevel)
	}()
	fn()
	return buffer.String()
}

// treeOf lists a directory as sorted "kind path" lines, so a whole installation
// can be asserted in one comparison.
func treeOf(t *testing.T, root string) []string {
	t.Helper()
	entries := []string{}
	var walk func(dir string)
	walk = func(dir string) {
		children, err := os.ReadDir(dir)
		if err != nil {
			return
		}
		for _, child := range children {
			path := filepath.Join(dir, child.Name())
			relative, _ := filepath.Rel(root, path)
			info, err := os.Lstat(path)
			if err != nil {
				continue
			}
			switch {
			case info.Mode()&os.ModeSymlink != 0:
				target, _ := os.Readlink(path)
				linkTarget, relErr := filepath.Rel(root, target)
				if relErr != nil {
					linkTarget = target
				}
				entries = append(entries, "l "+relative+" -> "+linkTarget)
			case info.IsDir():
				entries = append(entries, "d "+relative)
				walk(path)
			default:
				entries = append(entries, "f "+relative)
			}
		}
	}
	walk(root)
	sort.Strings(entries)
	return entries
}

func requireNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func readAllString(t *testing.T, r io.Reader) string {
	t.Helper()
	body, err := io.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}
	return string(body)
}
