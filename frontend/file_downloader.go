package frontend

import (
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/eirslett/frontend-maven-plugin/internal/logging"
)

// FileDownloader fetches one artifact to a local path.
type FileDownloader interface {
	Download(downloadURL, destination, userName, password string, headers map[string]string) error
}

var fileDownloaderLogger = logging.GetLogger("FileDownloader")

// DefaultFileDownloader fetches over HTTP, or straight off the filesystem for a
// "file:" URL, honouring the build's proxy configuration.
type DefaultFileDownloader struct {
	proxyConfig *ProxyConfig
}

// NewDefaultFileDownloader builds a downloader that routes through proxyConfig.
func NewDefaultFileDownloader(proxyConfig *ProxyConfig) *DefaultFileDownloader {
	return &DefaultFileDownloader{proxyConfig: proxyConfig}
}

// Download fetches downloadURL to destination, creating the parent directories.
// Any status other than 200 is a failure, as is any transport or filesystem
// error; both are reported against the URL that was actually requested.
func (d *DefaultFileDownloader) Download(downloadURL, destination, userName, password string, httpHeaders map[string]string) error {
	fixedDownloadURL := separatorsToUnix(downloadURL)

	parsed, err := url.Parse(fixedDownloadURL)
	if err != nil {
		return newDownloadError("Could not download "+fixedDownloadURL, err)
	}

	if strings.EqualFold(parsed.Scheme, "file") {
		if err := copyFile(fileURIPath(parsed), destination); err != nil {
			return newDownloadError("Could not download "+fixedDownloadURL, err)
		}
		return nil
	}

	response, err := d.execute(fixedDownloadURL, userName, password, httpHeaders)
	if err != nil {
		return newDownloadError("Could not download "+fixedDownloadURL, err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return newDownloadError("Got error code "+strconv.Itoa(response.StatusCode)+" from the server.", nil)
	}

	data, err := io.ReadAll(response.Body)
	if err != nil {
		return newDownloadError("Could not download "+fixedDownloadURL, err)
	}
	if parent := fullPathNoEndSeparator(destination); parent != "" {
		_ = os.MkdirAll(parent, 0o777)
	}
	if err := os.WriteFile(destination, data, 0o666); err != nil {
		return newDownloadError("Could not download "+fixedDownloadURL, err)
	}
	return nil
}

func (d *DefaultFileDownloader) execute(requestURL, userName, password string, httpHeaders map[string]string) (*http.Response, error) {
	request, err := http.NewRequest(http.MethodGet, requestURL, nil)
	if err != nil {
		return nil, err
	}

	transport := &http.Transport{
		// The original disables content compression, so the bytes on the wire
		// are the bytes written to the cache.
		DisableCompression: true,
		// Standing in for HttpClients.useSystemProperties(), which picks the
		// proxy up from the JVM's system properties.
		Proxy: http.ProxyFromEnvironment,
	}

	proxy := d.proxyConfig.ProxyForURL(requestURL)
	if proxy != nil {
		fileDownloaderLogger.Info("Downloading via proxy " + proxy.String())
		proxyURL, parseErr := url.Parse("http://" + proxy.Host + ":" + strconv.Itoa(proxy.Port))
		if parseErr != nil {
			return nil, parseErr
		}
		if proxy.UseAuthentication() {
			proxyURL.User = url.UserPassword(proxy.Username, proxy.Password)
		}
		transport.Proxy = http.ProxyURL(proxyURL)
	} else {
		fileDownloaderLogger.Info("No proxy was configured, downloading directly")
	}

	for name, value := range httpHeaders {
		fileDownloaderLogger.Info("Using HTTP-Header (" + name + ") from settings.xml")
		request.Header.Add(name, value)
	}

	if userName != "" && password != "" {
		fileDownloaderLogger.Info("Using credentials (" + userName + ") from settings.xml")
		// Sent pre-emptively, as the original's auth cache does, so a server
		// that never issues a challenge still sees the credentials.
		request.SetBasicAuth(userName, password)
	}

	client := &http.Client{Transport: transport, Timeout: downloadTimeout}
	return client.Do(request)
}

const downloadTimeout = 30 * time.Minute

// separatorsToUnix is FilenameUtils.separatorsToUnix.
func separatorsToUnix(path string) string {
	return strings.ReplaceAll(path, `\`, "/")
}

// fullPathNoEndSeparator is FilenameUtils.getFullPathNoEndSeparator: the
// directory part of a path, without its trailing separator.
func fullPathNoEndSeparator(path string) string {
	index := strings.LastIndexAny(path, `/\`)
	if index < 0 {
		return ""
	}
	if index == 0 {
		return path[:1]
	}
	return path[:index]
}

// fileURIPath is `new File(URI)`: the filesystem path a file: URI names.
func fileURIPath(parsed *url.URL) string {
	if parsed.Path != "" {
		return parsed.Path
	}
	return parsed.Opaque
}

func copyFile(source, destination string) error {
	if parent := fullPathNoEndSeparator(destination); parent != "" {
		if err := os.MkdirAll(parent, 0o777); err != nil {
			return err
		}
	}
	in, err := os.Open(source)
	if err != nil {
		return err
	}
	defer in.Close()
	info, err := in.Stat()
	if err != nil {
		return err
	}
	out, err := os.OpenFile(destination, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, info.Mode().Perm())
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}
