// Package goal is the plugin's user-facing layer: the eighteen goals the
// original exposes as Maven mojos, their parameters, and the build settings they
// read.
//
// The original gets all of this from the Maven container — parameter injection,
// settings decryption, incremental-build deltas, the local artifact repository.
// Go has no such container, so the layer supplies the same inputs from a
// settings file and an explicit session, and the goal names, parameter names,
// defaults and messages carry over unchanged.
package goal

import (
	"encoding/xml"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/eirslett/frontend-maven-plugin/frontend"
	"github.com/eirslett/frontend-maven-plugin/internal/logging"
)

var settingsLogger = logging.GetLogger("MojoUtils")

// Settings is the build's settings file: the proxies to route downloads
// through and the servers that hold their credentials.
type Settings struct {
	XMLName xml.Name  `xml:"settings"`
	Proxies []Proxy   `xml:"proxies>proxy"`
	Servers []Server  `xml:"servers>server"`
	Profile []Profile `xml:"profiles>profile"`
}

// Profile is carried so that a settings file with profiles still parses; none of
// its content affects this plugin.
type Profile struct {
	ID string `xml:"id"`
}

// Proxy is one <proxy> entry.
type Proxy struct {
	ID            string  `xml:"id"`
	Active        *bool   `xml:"active"`
	Protocol      string  `xml:"protocol"`
	Host          string  `xml:"host"`
	Port          int     `xml:"port"`
	Username      string  `xml:"username"`
	Password      string  `xml:"password"`
	NonProxyHosts *string `xml:"nonProxyHosts"`
}

// IsActive reports whether the proxy participates in this build. An entry with
// no <active> element is active, which is Maven's default.
func (p Proxy) IsActive() bool {
	return p.Active == nil || *p.Active
}

// Server is one <server> entry: credentials, plus the free-form configuration
// this plugin reads HTTP headers out of.
type Server struct {
	ID            string        `xml:"id"`
	Username      string        `xml:"username"`
	Password      string        `xml:"password"`
	Configuration Configuration `xml:"configuration"`
}

// Configuration is a server's <configuration> block.
type Configuration struct {
	HTTPHeaders *HTTPHeaders `xml:"httpHeaders"`
}

// HTTPHeaders is the <httpHeaders> block of a server configuration.
type HTTPHeaders struct {
	Properties []HeaderProperty `xml:"property"`
}

// HeaderProperty is one <property> inside <httpHeaders>.
type HeaderProperty struct {
	Name  string `xml:"name"`
	Value string `xml:"value"`
}

// LoadSettings reads a settings file. A path that does not exist yields empty
// settings rather than an error: a build without one is perfectly ordinary.
func LoadSettings(path string) (*Settings, error) {
	if path == "" {
		return &Settings{}, nil
	}
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return &Settings{}, nil
	}
	if err != nil {
		return nil, err
	}
	settings := &Settings{}
	if err := xml.Unmarshal(data, settings); err != nil {
		return nil, err
	}
	return settings, nil
}

// DefaultSettingsPath is the settings file a build reads when none is named:
// ~/.m2/settings.xml, the same place the original's container looks.
func DefaultSettingsPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".m2", "settings.xml")
}

// ProxyConfigFor turns the session's active proxies into the library's proxy
// configuration.
func ProxyConfigFor(session *Session) *frontend.ProxyConfig {
	if session == nil || session.Settings == nil || len(session.Settings.Proxies) == 0 {
		return frontend.NewProxyConfig(nil)
	}

	proxies := make([]*frontend.Proxy, 0, len(session.Settings.Proxies))
	for _, mavenProxy := range session.Settings.Proxies {
		if !mavenProxy.IsActive() {
			continue
		}
		decrypted := decryptProxy(mavenProxy)
		proxies = append(proxies, frontend.NewProxy(
			decrypted.ID, decrypted.Protocol, decrypted.Host, decrypted.Port,
			decrypted.Username, decrypted.Password, decrypted.NonProxyHosts))
	}

	settingsLogger.Info("Found proxies: {}", renderProxyList(proxies))
	return frontend.NewProxyConfig(proxies)
}

// renderProxyList renders a proxy list the way the original's log line does,
// which is java.util.List.toString.
func renderProxyList(proxies []*frontend.Proxy) string {
	rendered := make([]string, 0, len(proxies))
	for _, proxy := range proxies {
		rendered = append(rendered, proxy.String())
	}
	return "[" + strings.Join(rendered, ", ") + "]"
}

// DecryptServer finds the server a goal's serverId names.
func DecryptServer(serverID string, session *Session) *Server {
	if serverID == "" {
		return nil
	}
	if session == nil || session.Settings == nil {
		settingsLogger.Warn("Could not find server '" + serverID + "' in settings.xml")
		return nil
	}
	for index := range session.Settings.Servers {
		if session.Settings.Servers[index].ID == serverID {
			decrypted := decryptServer(session.Settings.Servers[index])
			return &decrypted
		}
	}
	settingsLogger.Warn("Could not find server '" + serverID + "' in settings.xml")
	return nil
}

// HTTPHeadersOf reads a server's configured HTTP headers.
func HTTPHeadersOf(server *Server) map[string]string {
	result := map[string]string{}
	if server == nil || server.Configuration.HTTPHeaders == nil {
		return result
	}
	for _, property := range server.Configuration.HTTPHeaders.Properties {
		result[property.Name] = property.Value
	}
	return result
}

// decryptProxy and decryptServer stand where the original calls Maven's
// SettingsDecrypter. Decrypting a settings-security.xml-encrypted password is
// declared out of scope for this port, so a value is passed through as written;
// a plaintext settings file — which is what every one of this plugin's own
// integration tests uses — behaves identically.
func decryptProxy(proxy Proxy) Proxy {
	return proxy
}

func decryptServer(server Server) Server {
	return server
}

// portOf renders a proxy port for a URL.
func portOf(port int) string {
	return strconv.Itoa(port)
}
