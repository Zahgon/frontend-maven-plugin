package goal

import (
	"path/filepath"
	"strings"
	"testing"
)

const settingsWithProxies = `<?xml version="1.0" encoding="UTF-8"?>
<settings>
  <proxies>
    <proxy>
      <id>active-http</id>
      <active>true</active>
      <protocol>http</protocol>
      <host>proxy.example</host>
      <port>8080</port>
      <username>user</username>
      <password>secret</password>
      <nonProxyHosts>*.internal|localhost</nonProxyHosts>
    </proxy>
    <proxy>
      <id>inactive</id>
      <active>false</active>
      <protocol>http</protocol>
      <host>never.example</host>
      <port>3128</port>
    </proxy>
    <proxy>
      <id>implicitly-active</id>
      <protocol>https</protocol>
      <host>secure.example</host>
      <port>443</port>
    </proxy>
  </proxies>
  <servers>
    <server>
      <id>downloads</id>
      <username>server-user</username>
      <password>server-password</password>
      <configuration>
        <httpHeaders>
          <property><name>X-One</name><value>1</value></property>
          <property><name>X-Two</name><value>2</value></property>
        </httpHeaders>
      </configuration>
    </server>
    <server>
      <id>plain</id>
      <username>plain-user</username>
      <password>plain-password</password>
    </server>
  </servers>
  <profiles>
    <profile><id>ignored</id></profile>
  </profiles>
</settings>
`

func loadTestSettings(t *testing.T) *Settings {
	t.Helper()
	path := filepath.Join(t.TempDir(), "settings.xml")
	writeFile(t, path, settingsWithProxies)
	settings, err := LoadSettings(path)
	requireNoError(t, err)
	return settings
}

func TestLoadSettingsReadsProxiesAndServers(t *testing.T) {
	settings := loadTestSettings(t)

	if len(settings.Proxies) != 3 {
		t.Fatalf("proxies = %d, want 3", len(settings.Proxies))
	}
	if !settings.Proxies[0].IsActive() {
		t.Error("an explicitly active proxy must read as active")
	}
	if settings.Proxies[1].IsActive() {
		t.Error("an explicitly inactive proxy must read as inactive")
	}
	if !settings.Proxies[2].IsActive() {
		t.Error("a proxy without an <active> element must read as active")
	}
	if len(settings.Servers) != 2 {
		t.Fatalf("servers = %d, want 2", len(settings.Servers))
	}
}

func TestLoadSettingsToleratesAMissingFile(t *testing.T) {
	settings, err := LoadSettings(filepath.Join(t.TempDir(), "absent.xml"))
	requireNoError(t, err)
	if len(settings.Proxies) != 0 {
		t.Error("a missing settings file must yield no proxies")
	}

	settings, err = LoadSettings("")
	requireNoError(t, err)
	if settings == nil {
		t.Error("an unnamed settings file must yield empty settings")
	}
}

func TestLoadSettingsReportsUnparsableXml(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.xml")
	writeFile(t, path, "<settings>")

	if _, err := LoadSettings(path); err == nil {
		t.Error("unparsable settings must be reported")
	}
}

func TestProxyConfigForKeepsOnlyActiveProxies(t *testing.T) {
	session, _ := newTestSession(t)
	session.Settings = loadTestSettings(t)

	var config = ProxyConfigFor(session)
	output := captureLog(t, func() { config = ProxyConfigFor(session) })

	if len(config.Proxies()) != 2 {
		t.Fatalf("proxies = %d, want the two active ones", len(config.Proxies()))
	}
	if config.Proxies()[0].ID != "active-http" || config.Proxies()[1].ID != "implicitly-active" {
		t.Errorf("proxies = %v, want the active ones in order", config.Proxies())
	}
	if got := config.Proxies()[0].URI(); got != "http://user:secret@proxy.example:8080" {
		t.Errorf("proxy URI = %q, want the credentials in the authority", got)
	}
	if !strings.Contains(output, "Found proxies: [active-http{") {
		t.Errorf("log does not report the proxies as a list:\n%s", output)
	}
	// The second proxy declares no exemption list, which the rendering shows.
	if !strings.Contains(output, "nonProxyHosts='null'") {
		t.Errorf("log does not render the absent exemption list:\n%s", output)
	}
}

func TestProxyConfigForIsEmptyWithoutSettings(t *testing.T) {
	if !ProxyConfigFor(nil).IsEmpty() {
		t.Error("no session means no proxies")
	}
	session, _ := newTestSession(t)
	if !ProxyConfigFor(session).IsEmpty() {
		t.Error("empty settings mean no proxies")
	}
	session.Settings = nil
	if !ProxyConfigFor(session).IsEmpty() {
		t.Error("absent settings mean no proxies")
	}
}

func TestDecryptServerFindsTheNamedServer(t *testing.T) {
	session, _ := newTestSession(t)
	session.Settings = loadTestSettings(t)

	server := DecryptServer("downloads", session)
	if server == nil {
		t.Fatal("the configured server was not found")
	}
	if server.Username != "server-user" || server.Password != "server-password" {
		t.Errorf("credentials = %q/%q, want the configured ones", server.Username, server.Password)
	}

	headers := HTTPHeadersOf(server)
	if headers["X-One"] != "1" || headers["X-Two"] != "2" {
		t.Errorf("headers = %v, want both configured properties", headers)
	}
	if got := HTTPHeadersOf(DecryptServer("plain", session)); len(got) != 0 {
		t.Errorf("headers = %v, want none for a server without a configuration", got)
	}
	if got := HTTPHeadersOf(nil); len(got) != 0 {
		t.Errorf("headers = %v, want none for no server", got)
	}
}

func TestDecryptServerReportsAMissingServer(t *testing.T) {
	session, _ := newTestSession(t)
	session.Settings = loadTestSettings(t)

	if DecryptServer("", session) != nil {
		t.Error("an unnamed server must resolve to nothing")
	}

	output := captureLog(t, func() {
		if DecryptServer("absent", session) != nil {
			t.Error("a server that is not configured must resolve to nothing")
		}
	})
	if !strings.Contains(output, "Could not find server 'absent' in settings.xml") {
		t.Errorf("log does not report the missing server:\n%s", output)
	}

	output = captureLog(t, func() {
		if DecryptServer("absent", nil) != nil {
			t.Error("no session means no server")
		}
	})
	if !strings.Contains(output, "Could not find server 'absent' in settings.xml") {
		t.Errorf("log does not report the missing server:\n%s", output)
	}
}

func TestDefaultSettingsPathIsUnderTheHomeDirectory(t *testing.T) {
	if got := DefaultSettingsPath(); got != "" && !strings.HasSuffix(got, filepath.Join(".m2", "settings.xml")) {
		t.Errorf("default settings path = %q, want it under .m2", got)
	}
	if got := DefaultLocalRepository(); got != "" && !strings.HasSuffix(got, filepath.Join(".m2", "repository")) {
		t.Errorf("default local repository = %q, want it under .m2", got)
	}
}

func TestDecryptionPassesValuesThrough(t *testing.T) {
	// Decrypting a settings-security.xml-encrypted value is out of scope, so a
	// value reaches the downloader as it was written.
	proxy := Proxy{ID: "id", Password: "{encrypted}"}
	if decryptProxy(proxy).Password != "{encrypted}" {
		t.Error("the proxy password was rewritten")
	}
	server := Server{ID: "id", Password: "{encrypted}"}
	if decryptServer(server).Password != "{encrypted}" {
		t.Error("the server password was rewritten")
	}
	if portOf(8080) != "8080" {
		t.Errorf("portOf = %q, want %q", portOf(8080), "8080")
	}
}
