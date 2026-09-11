package frontend

import (
	"reflect"
	"testing"
)

func TestImplode(t *testing.T) {
	const separator = "Bar"
	elements := []string{}

	if got := Implode(separator, elements); got != "" {
		t.Errorf("Implode(%q, %v) = %q, want %q", separator, elements, got, "")
	}

	elements = append(elements, "foo")
	elements = append(elements, "bar")
	if got := Implode(separator, elements); got != "foo bar" {
		t.Errorf("Implode(%q, %v) = %q, want %q", separator, elements, got, "foo bar")
	}
}

func TestIsRelative(t *testing.T) {
	if !IsRelative("foo/bar") {
		t.Error(`IsRelative("foo/bar") = false, want true`)
	}

	for _, path := range []string{"/foo/bar", "file:foo/bar", `C:\SYSTEM`} {
		if IsRelative(path) {
			t.Errorf("IsRelative(%q) = true, want false", path)
		}
	}
}

func TestMerge(t *testing.T) {
	want := []string{"foo", "bar"}
	got := Merge([]string{"foo"}, []string{"bar"})
	if !reflect.DeepEqual(want, got) {
		t.Errorf("Merge = %v, want %v", got, want)
	}
}

func TestPrepend(t *testing.T) {
	want := []string{"foo", "bar"}
	got := Prepend("foo", []string{"bar"})
	if !reflect.DeepEqual(want, got) {
		t.Errorf("Prepend = %v, want %v", got, want)
	}
}
