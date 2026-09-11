package frontend

import (
	"strings"
	"testing"
)

func TestProxyConfigPicksAProxyForARequest(t *testing.T) {
	insecureHosts := "www.google.ca|www.google.com|*.google.de"
	insecure := NewProxy("id", "http", "localhost", 8888, "u", "p", &insecureHosts)
	secure := NewProxy("s", "https", "sec", 443, "", "", nil)
	config := NewProxyConfig([]*Proxy{insecure, secure})

	if config.IsEmpty() {
		t.Error("a configured proxy list must not read as empty")
	}
	if len(config.Proxies()) != 2 {
		t.Errorf("proxies = %d, want 2", len(config.Proxies()))
	}
	if got := config.SecureProxy(); got != secure {
		t.Errorf("secure proxy = %v, want the https one", got)
	}
	if got := config.InsecureProxy(); got != insecure {
		t.Errorf("insecure proxy = %v, want the http one", got)
	}
	if got := config.ProxyForURL("http://elsewhere/x"); got != insecure {
		t.Errorf("proxy for an unexempted host = %v, want the first one", got)
	}
	// The first proxy exempts the host, so the second one takes the request.
	if got := config.ProxyForURL("http://www.google.ca/x"); got != secure {
		t.Errorf("proxy for an exempted host = %v, want the second one", got)
	}
	// A string with no authority has no host, which no exemption can match.
	if got := config.ProxyForURL("notaurl"); got != insecure {
		t.Errorf("proxy for a hostless URL = %v, want the first one", got)
	}
}

func TestProxyConfigWithNoProxiesSelectsNothing(t *testing.T) {
	config := NewProxyConfig(nil)

	if !config.IsEmpty() {
		t.Error("an empty proxy list must read as empty")
	}
	if config.SecureProxy() != nil || config.InsecureProxy() != nil {
		t.Error("an empty proxy list must select nothing")
	}
	output := captureLog(t, 0, func() {
		if config.ProxyForURL("http://x/y") != nil {
			t.Error("an empty proxy list must select nothing for a URL")
		}
	})
	if !strings.Contains(output, "No proxies configured") {
		t.Errorf("log does not report the empty configuration:\n%s", output)
	}
}

func TestProxyConfigReportsWhenEveryProxyExemptsTheHost(t *testing.T) {
	hosts := "only.host"
	config := NewProxyConfig([]*Proxy{NewProxy("id", "http", "h", 1, "", "", &hosts)})

	output := captureLog(t, 0, func() {
		if config.ProxyForURL("http://only.host/x") != nil {
			t.Error("an exempted host must select no proxy")
		}
	})

	if !strings.Contains(output, "Could not find matching proxy for host: only.host") {
		t.Errorf("log does not report the missing match:\n%s", output)
	}
}

func TestProxyRendersItsUri(t *testing.T) {
	authed := NewProxy("id", "http", "localhost", 8888, "user", "pass", nil)
	anonymous := NewProxy("anon", "https", "secure.example", 443, "", "", nil)

	if got := authed.URI(); got != "http://user:pass@localhost:8888" {
		t.Errorf("URI = %q, want the credentials in the authority", got)
	}
	// The scheme is http even for an https proxy: that is what npm is given.
	if got := anonymous.URI(); got != "http://secure.example:443" {
		t.Errorf("URI = %q, want an http scheme and no credentials", got)
	}
	if !authed.UseAuthentication() || anonymous.UseAuthentication() {
		t.Error("authentication is decided by whether a username is configured")
	}
	if authed.IsSecure() || !anonymous.IsSecure() {
		t.Error("a proxy is secure exactly when its protocol is https")
	}
}

func TestProxyRendersItselfForTheLog(t *testing.T) {
	hosts := "a|b"
	authed := NewProxy("id", "http", "localhost", 8888, "user", "pass", &hosts)
	anonymous := NewProxy("anon", "https", "sec", 443, "", "", nil)

	want := "id{protocol='http', host='localhost', port=8888, nonProxyHosts='a|b'," +
		" with username/passport authentication}"
	if got := authed.String(); got != want {
		t.Errorf("rendering = %q, want %q", got, want)
	}
	// An exemption list that was never configured renders as absent.
	want = "anon{protocol='https', host='sec', port=443, nonProxyHosts='null'}"
	if got := anonymous.String(); got != want {
		t.Errorf("rendering = %q, want %q", got, want)
	}
}

func TestProxyMatchesNonProxyHostsAsWholeNames(t *testing.T) {
	hosts := "www.google.ca|www.google.com|*.google.de"
	proxy := NewProxy("id", "http", "h", 1, "", "", &hosts)

	for host, want := range map[string]bool{
		"www.google.ca":  true,
		"www.google.com": true,
		"x.google.de":    true,
		"a.b.google.de":  true,
		"google.de":      false,
		"www.google.dev": false,
		"wwwXgoogle.ca":  false,
		"other.host":     false,
		"":               false,
		"WWW.GOOGLE.CA":  false,
	} {
		if got := proxy.IsNonProxyHost(host); got != want {
			t.Errorf("IsNonProxyHost(%q) = %t, want %t", host, got, want)
		}
	}

	unset := NewProxy("id", "http", "h", 1, "", "", nil)
	if unset.IsNonProxyHost("anything") {
		t.Error("a proxy with no exemption list exempts nothing")
	}
	if unset.GetNonProxyHosts() != "" {
		t.Errorf("GetNonProxyHosts = %q, want empty", unset.GetNonProxyHosts())
	}
	if proxy.GetNonProxyHosts() != hosts {
		t.Errorf("GetNonProxyHosts = %q, want %q", proxy.GetNonProxyHosts(), hosts)
	}
}
