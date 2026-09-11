package frontend

import "testing"

func TestBuildArgumentBasicTest(t *testing.T) {
	arguments := BuildNpxNpmArguments(NewProxyConfig(nil), "")
	if len(arguments) != 0 {
		t.Errorf("got %d arguments (%v), want 0", len(arguments), arguments)
	}
}

func TestBuildArgumentWithRegistryUrl(t *testing.T) {
	arguments := BuildNpxNpmArguments(NewProxyConfig(nil), npmRegistryURLTest)
	if len(arguments) != 2 {
		t.Fatalf("got %d arguments (%v), want 2", len(arguments), arguments)
	}
	assertHasItem(t, arguments, "--")
	assertHasItem(t, arguments, "--registry="+npmRegistryURLTest)
}
