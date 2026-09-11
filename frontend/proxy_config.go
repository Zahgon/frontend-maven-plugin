package frontend

import (
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/eirslett/frontend-maven-plugin/internal/logging"
)

var proxyConfigLogger = logging.GetLogger("ProxyConfig")

// ProxyConfig is the set of proxies a build was configured with, and the rules
// for picking one for a given request.
type ProxyConfig struct {
	proxies []*Proxy
}

// NewProxyConfig wraps a proxy list, which may be empty.
func NewProxyConfig(proxies []*Proxy) *ProxyConfig {
	return &ProxyConfig{proxies: proxies}
}

// IsEmpty reports whether no proxy is configured at all.
func (c *ProxyConfig) IsEmpty() bool {
	return len(c.proxies) == 0
}

// Proxies exposes the configured proxies in order.
func (c *ProxyConfig) Proxies() []*Proxy {
	return c.proxies
}

// ProxyForURL picks the first proxy whose non-proxy-host list does not exempt
// the requested host, or nil when none applies.
func (c *ProxyConfig) ProxyForURL(requestURL string) *Proxy {
	if len(c.proxies) == 0 {
		proxyConfigLogger.Info("No proxies configured")
		return nil
	}
	host := uriHost(requestURL)
	for _, proxy := range c.proxies {
		if !proxy.IsNonProxyHost(host) {
			return proxy
		}
	}
	proxyConfigLogger.Info("Could not find matching proxy for host: {}", host)
	return nil
}

// SecureProxy is the first proxy declared with the https protocol, or nil.
func (c *ProxyConfig) SecureProxy() *Proxy {
	for _, proxy := range c.proxies {
		if proxy.IsSecure() {
			return proxy
		}
	}
	return nil
}

// InsecureProxy is the first proxy not declared with the https protocol, or nil.
func (c *ProxyConfig) InsecureProxy() *Proxy {
	for _, proxy := range c.proxies {
		if !proxy.IsSecure() {
			return proxy
		}
	}
	return nil
}

// uriHost is URI.create(url).getHost(): the authority's host, empty when the
// string carries no authority at all.
func uriHost(requestURL string) string {
	parsed, err := url.Parse(requestURL)
	if err != nil {
		return ""
	}
	return parsed.Hostname()
}

// proxyURIScheme is the scheme every proxy URI carries, even one declared as
// https: that is what the original emits, and npm, yarn, pnpm and bun are all
// configured with it verbatim.
const proxyURIScheme = "http://"

// Proxy is one configured proxy server.
type Proxy struct {
	ID       string
	Protocol string
	Host     string
	Port     int
	Username string
	Password string
	// NonProxyHosts is nil when the settings entry declares none, which the log
	// line renders differently from a list that is present but empty.
	NonProxyHosts *string
}

// NewProxy builds a proxy from the fields a settings file supplies. A nil
// nonProxyHosts means the entry declared none at all.
func NewProxy(id, protocol, host string, port int, username, password string, nonProxyHosts *string) *Proxy {
	return &Proxy{
		ID:            id,
		Protocol:      protocol,
		Host:          host,
		Port:          port,
		Username:      username,
		Password:      password,
		NonProxyHosts: nonProxyHosts,
	}
}

// UseAuthentication reports whether this proxy carries credentials.
func (p *Proxy) UseAuthentication() bool {
	return p.Username != ""
}

// URI renders the proxy as the URL a package manager is told to use.
//
// The scheme is always http, even for a proxy declared as https: that is what
// the original emits, and npm and yarn are configured with it verbatim.
func (p *Proxy) URI() string {
	authority := p.Host
	if p.Port >= 0 {
		authority = p.Host + ":" + strconv.Itoa(p.Port)
	}
	if p.UseAuthentication() {
		return proxyURIScheme + p.Username + ":" + p.Password + "@" + authority
	}
	return proxyURIScheme + authority
}

// IsSecure reports whether the proxy was declared with the https protocol.
func (p *Proxy) IsSecure() bool {
	return p.Protocol == "https"
}

// IsNonProxyHost reports whether host is exempted from this proxy.
//
// The exemption list is "|"-separated and its entries are glob-ish: "." is
// literal and "*" matches anything. The whole host must match, not a prefix.
func (p *Proxy) IsNonProxyHost(host string) bool {
	if host == "" || p.GetNonProxyHosts() == "" {
		return false
	}
	for _, pattern := range strings.Split(p.GetNonProxyHosts(), "|") {
		if pattern == "" {
			continue
		}
		pattern = strings.ReplaceAll(pattern, ".", `\.`)
		pattern = strings.ReplaceAll(pattern, "*", ".*")
		matcher, err := regexp.Compile("^(?:" + pattern + ")$")
		if err != nil {
			continue
		}
		if matcher.MatchString(host) {
			return true
		}
	}
	return false
}

// GetNonProxyHosts is the exemption list as configured.
//
// npm documents a comma-separated list at
// https://docs.npmjs.com/misc/config#noproxy while a Maven settings file
// usually uses a bar-separated one. npm accepts the bar-separated form
// regardless of what its documentation says, so no conversion happens here.
func (p *Proxy) GetNonProxyHosts() string {
	if p.NonProxyHosts == nil {
		return ""
	}
	return *p.NonProxyHosts
}

// String renders the proxy for the log, never disclosing the password.
func (p *Proxy) String() string {
	authentication := ""
	if p.UseAuthentication() {
		authentication = ", with username/passport authentication"
	}
	nonProxyHosts := "null"
	if p.NonProxyHosts != nil {
		nonProxyHosts = *p.NonProxyHosts
	}
	return fmt.Sprintf("%s{protocol='%s', host='%s', port=%d, nonProxyHosts='%s'%s}",
		p.ID, p.Protocol, p.Host, p.Port, nonProxyHosts, authentication)
}
