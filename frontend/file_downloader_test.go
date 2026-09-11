package frontend

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The downloader replaces httpclient; its status handling, credential handling
// and proxy routing are all part of what the plugin promises.

func TestFileDownloaderWritesTheBodyAndCreatesParents(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("payload"))
	}))
	defer server.Close()
	destination := filepath.Join(t.TempDir(), "nested", "deeper", "file.bin")

	err := NewDefaultFileDownloader(NewProxyConfig(nil)).
		Download(server.URL+"/thing", destination, "", "", nil)

	requireNoError(t, err)
	body, err := os.ReadFile(destination)
	requireNoError(t, err)
	if string(body) != "payload" {
		t.Errorf("downloaded %q, want %q", body, "payload")
	}
}

func TestFileDownloaderReportsANonOkStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	err := NewDefaultFileDownloader(NewProxyConfig(nil)).
		Download(server.URL+"/missing", filepath.Join(t.TempDir(), "out"), "", "", nil)

	if err == nil || err.Error() != "Got error code 404 from the server." {
		t.Fatalf("error = %v, want the 404 message", err)
	}
}

func TestFileDownloaderReportsAnUnreachableServer(t *testing.T) {
	// Port 0 is never listening, so the request cannot even be made.
	target := "http://127.0.0.1:0/thing"

	err := NewDefaultFileDownloader(NewProxyConfig(nil)).
		Download(target, filepath.Join(t.TempDir(), "out"), "", "", nil)

	if err == nil || !strings.HasPrefix(err.Error(), "Could not download "+target) {
		t.Fatalf("error = %v, want a download failure naming the URL", err)
	}
}

func TestFileDownloaderSendsCredentialsAndHeaders(t *testing.T) {
	var gotUser, gotPassword, gotHeader string
	var gotBasic bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUser, gotPassword, gotBasic = r.BasicAuth()
		gotHeader = r.Header.Get("X-Probe")
		_, _ = w.Write([]byte("ok"))
	}))
	defer server.Close()

	output := captureLog(t, 0, func() {
		requireNoError(t, NewDefaultFileDownloader(NewProxyConfig(nil)).
			Download(server.URL, filepath.Join(t.TempDir(), "out"), "user", "secret",
				map[string]string{"X-Probe": "yes"}))
	})

	if !gotBasic || gotUser != "user" || gotPassword != "secret" {
		t.Errorf("credentials = %q/%q (basic=%t), want user/secret", gotUser, gotPassword, gotBasic)
	}
	if gotHeader != "yes" {
		t.Errorf("header = %q, want %q", gotHeader, "yes")
	}
	if !strings.Contains(output, "Using credentials (user) from settings.xml") {
		t.Errorf("log does not report the credentials:\n%s", output)
	}
	if !strings.Contains(output, "Using HTTP-Header (X-Probe) from settings.xml") {
		t.Errorf("log does not report the header:\n%s", output)
	}
	if !strings.Contains(output, "No proxy was configured, downloading directly") {
		t.Errorf("log does not report the direct download:\n%s", output)
	}
}

func TestFileDownloaderRoutesThroughAProxy(t *testing.T) {
	var proxied bool
	proxy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		proxied = true
		_, _ = w.Write([]byte("through-proxy"))
	}))
	defer proxy.Close()
	proxyURL, err := url.Parse(proxy.URL)
	requireNoError(t, err)
	port := proxyURL.Port()
	config := NewProxyConfig([]*Proxy{
		NewProxy("id", "http", proxyURL.Hostname(), atoiOrZero(port), "u", "p", nil),
	})
	destination := filepath.Join(t.TempDir(), "out")

	output := captureLog(t, 0, func() {
		requireNoError(t, NewDefaultFileDownloader(config).
			Download("http://example.invalid/thing", destination, "", "", nil))
	})

	if !proxied {
		t.Error("the request did not go through the proxy")
	}
	if !strings.Contains(output, "Downloading via proxy id{") {
		t.Errorf("log does not report the proxy:\n%s", output)
	}
	body, err := os.ReadFile(destination)
	requireNoError(t, err)
	if string(body) != "through-proxy" {
		t.Errorf("downloaded %q, want %q", body, "through-proxy")
	}
}

func TestFileDownloaderCopiesAFileUrl(t *testing.T) {
	directory := t.TempDir()
	source := filepath.Join(directory, "source.bin")
	writeFixture(t, source, []byte("local"))
	destination := filepath.Join(directory, "copied", "out.bin")

	requireNoError(t, NewDefaultFileDownloader(NewProxyConfig(nil)).
		Download("file://"+source, destination, "", "", nil))

	body, err := os.ReadFile(destination)
	requireNoError(t, err)
	if string(body) != "local" {
		t.Errorf("copied %q, want %q", body, "local")
	}
}

func TestFileDownloaderReportsAMissingFileUrl(t *testing.T) {
	target := "file://" + filepath.Join(t.TempDir(), "absent.bin")

	err := NewDefaultFileDownloader(NewProxyConfig(nil)).
		Download(target, filepath.Join(t.TempDir(), "out"), "", "", nil)

	var download *DownloadError
	if !errors.As(err, &download) {
		t.Fatalf("error = %v, want a DownloadError", err)
	}
	if !strings.HasPrefix(download.Error(), "Could not download file://") {
		t.Errorf("message = %q, want it to name the URL", download.Error())
	}
}

func TestFileDownloaderNormalisesBackslashes(t *testing.T) {
	if got := separatorsToUnix(`C:\a\b`); got != "C:/a/b" {
		t.Errorf("separatorsToUnix = %q, want %q", got, "C:/a/b")
	}
	for path, want := range map[string]string{
		"/a/b/c.txt": "/a/b",
		"c.txt":      "",
		"/c.txt":     "/",
	} {
		if got := fullPathNoEndSeparator(path); got != want {
			t.Errorf("fullPathNoEndSeparator(%q) = %q, want %q", path, got, want)
		}
	}
}

func TestFileUriPathFallsBackToAnOpaqueUri(t *testing.T) {
	opaque, err := url.Parse("file:relative/path")
	requireNoError(t, err)
	if got := fileURIPath(opaque); got != "relative/path" {
		t.Errorf("fileURIPath = %q, want %q", got, "relative/path")
	}
	hierarchical, err := url.Parse("file:///abs/path")
	requireNoError(t, err)
	if got := fileURIPath(hierarchical); got != "/abs/path" {
		t.Errorf("fileURIPath = %q, want %q", got, "/abs/path")
	}
}

func atoiOrZero(value string) int {
	total := 0
	for _, c := range value {
		if c < '0' || c > '9' {
			return 0
		}
		total = total*10 + int(c-'0')
	}
	return total
}
