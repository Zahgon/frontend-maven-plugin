package frontend

import (
	"reflect"
	"testing"
)

// assertParsed compares a parse result against the expected argument vector.
func assertParsed(t *testing.T, parser *ArgumentsParser, args string, want []string) {
	t.Helper()
	got := parser.Parse(args)
	if !reflect.DeepEqual(want, got) {
		t.Errorf("Parse(%q) = %#v, want %#v", args, got, want)
	}
}

func TestNoArguments(t *testing.T) {
	parser := NewArgumentsParser()

	// The original passes Java's null, the literal "null", and the empty
	// string; a Go string cannot be null, so the empty string covers both it
	// and the empty case.
	if got := len(parser.Parse("")); got != 0 {
		t.Errorf(`Parse("") returned %d arguments, want 0`, got)
	}
	if got := len(parser.Parse("null")); got != 0 {
		t.Errorf(`Parse("null") returned %d arguments, want 0`, got)
	}
}

func TestMultipleArgumentsNoQuotes(t *testing.T) {
	parser := NewArgumentsParser()

	assertParsed(t, parser, "foo", []string{"foo"})
	assertParsed(t, parser, "foo bar", []string{"foo", "bar"})
	assertParsed(t, parser, "foo bar foobar", []string{"foo", "bar", "foobar"})
}

func TestMultipleArgumentsWithQuotes(t *testing.T) {
	parser := NewArgumentsParser()

	assertParsed(t, parser, `foo "bar foobar"`, []string{"foo", `"bar foobar"`})
	assertParsed(t, parser, `"foo bar" foobar`, []string{`"foo bar"`, "foobar"})
	assertParsed(t, parser, "foo 'bar foobar'", []string{"foo", "'bar foobar'"})
	assertParsed(t, parser, "'foo bar' foobar", []string{"'foo bar'", "foobar"})
	// Unclosed quotes.
	assertParsed(t, parser, `foo "bar foobar`, []string{"foo", `"bar foobar`})
}

func TestArgumentsWithMixedQuotes(t *testing.T) {
	parser := NewArgumentsParser()

	assertParsed(t, parser, `foo "bar 'foo bar'"`, []string{"foo", `"bar 'foo bar'"`})
	assertParsed(t, parser, `foo "bar 'foo" 'bar `, []string{"foo", `"bar 'foo"`, "'bar "})
}

func TestRepeatedArgumentsAreAccepted(t *testing.T) {
	parser := NewArgumentsParser()

	assertParsed(t, parser, "echo echo", []string{"echo", "echo"})
}

func TestAdditionalArgumentsNoIntersection(t *testing.T) {
	parser := NewArgumentsParserWith([]string{"foo", "bar"})

	assertParsed(t, parser, "foobar", []string{"foobar", "foo", "bar"})
}

func TestAdditionalArgumentsWithIntersection(t *testing.T) {
	parser := NewArgumentsParserWith([]string{"foo", "foobar"})

	assertParsed(t, parser, "bar foobar", []string{"bar", "foobar", "foo"})
}
