package frontend

import "testing"

const (
	proxyID            = "id"
	proxyProtocol      = "http"
	proxyHost          = "localhost"
	proxyPort          = 8888
	proxyUsername      = "someusername"
	proxyPassword      = "somepassword"
	npmNonProxyHosts   = "www.google.ca|www.google.com|*.google.de"
	npmRegistryURLTest = "www.npm.org"
	expectedProxyURL   = "http://someusername:somepassword@localhost:8888"
)

var expectedNonProxyHosts = []string{"www.google.ca", "www.google.com", ".google.de"}

// runBuildArguments builds npm's arguments for a single configured proxy. A nil
// nonProxyHost stands for the null the original passes.
func runBuildArguments(nonProxyHost *string, registryURL string) []string {
	proxyList := []*Proxy{
		NewProxy(proxyID, proxyProtocol, proxyHost, proxyPort, proxyUsername, proxyPassword, nonProxyHost),
	}
	return BuildNpmArguments(NewProxyConfig(proxyList), registryURL)
}

// stringPtr makes an addressable copy, so a test can pass a present value where
// the parameter is nullable.
func stringPtr(value string) *string { return &value }

// assertHasItem fails unless want appears in strings.
func assertHasItem(t *testing.T, strings []string, want string) {
	t.Helper()
	if !contains(strings, want) {
		t.Errorf("arguments %v do not contain %q", strings, want)
	}
}

func TestBuildArgumentsBasicTest(t *testing.T) {
	strings := runBuildArguments(stringPtr(npmNonProxyHosts), npmRegistryURLTest)

	assertHasItem(t, strings, "--proxy="+expectedProxyURL)
	assertHasItem(t, strings, "--https-proxy="+expectedProxyURL)
	for _, expectedNonProxyHost := range expectedNonProxyHosts {
		assertHasItem(t, strings, "--noproxy="+expectedNonProxyHost)
	}
	assertHasItem(t, strings, "--registry="+npmRegistryURLTest)
	if len(strings) != 6 {
		t.Errorf("got %d arguments (%v), want 6", len(strings), strings)
	}
}

func TestBuildArgumentsEmptyRegistryUrl(t *testing.T) {
	strings := runBuildArguments(stringPtr(npmNonProxyHosts), "")

	assertHasItem(t, strings, "--proxy="+expectedProxyURL)
	assertHasItem(t, strings, "--https-proxy="+expectedProxyURL)
	for _, expectedNonProxyHost := range expectedNonProxyHosts {
		assertHasItem(t, strings, "--noproxy="+expectedNonProxyHost)
	}
	if len(strings) != 5 {
		t.Errorf("got %d arguments (%v), want 5", len(strings), strings)
	}
}

func TestBuildArgumentsNullRegistryUrl(t *testing.T) {
	// A registry URL that was never configured reaches the builder as the empty
	// string, which is what Java's null becomes here.
	strings := runBuildArguments(stringPtr(npmNonProxyHosts), "")

	assertHasItem(t, strings, "--proxy="+expectedProxyURL)
	assertHasItem(t, strings, "--https-proxy="+expectedProxyURL)
	for _, expectedNonProxyHost := range expectedNonProxyHosts {
		assertHasItem(t, strings, "--noproxy="+expectedNonProxyHost)
	}
	if len(strings) != 5 {
		t.Errorf("got %d arguments (%v), want 5", len(strings), strings)
	}
}

func TestBuildArgumentsEmptyNoProxy(t *testing.T) {
	strings := runBuildArguments(stringPtr(""), "")

	assertHasItem(t, strings, "--proxy="+expectedProxyURL)
	assertHasItem(t, strings, "--https-proxy="+expectedProxyURL)
	if len(strings) != 2 {
		t.Errorf("got %d arguments (%v), want 2", len(strings), strings)
	}
}

func TestBuildArgumentsNullNoProxy(t *testing.T) {
	strings := runBuildArguments(nil, "")

	assertHasItem(t, strings, "--proxy="+expectedProxyURL)
	assertHasItem(t, strings, "--https-proxy="+expectedProxyURL)
	if len(strings) != 2 {
		t.Errorf("got %d arguments (%v), want 2", len(strings), strings)
	}
}
