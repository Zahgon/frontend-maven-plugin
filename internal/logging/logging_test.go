package logging

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

func capture(t *testing.T, level Level, fn func()) string {
	t.Helper()
	var buffer bytes.Buffer
	previousOut := SetOutput(&buffer)
	previousLevel := SetLevel(level)
	defer func() {
		SetOutput(previousOut)
		SetLevel(previousLevel)
	}()
	fn()
	return buffer.String()
}

func TestFormatSubstitutesPlaceholders(t *testing.T) {
	for _, testCase := range []struct {
		pattern string
		args    []any
		want    string
	}{
		{"no placeholders", nil, "no placeholders"},
		{"one {}", []any{"value"}, "one value"},
		{"two {} and {}", []any{"a", "b"}, "two a and b"},
		{"missing {} {}", []any{"only"}, "missing only {}"},
		{"surplus {}", []any{"a", "b"}, "surplus a (b)"},
		{"a null {}", []any{nil}, "a null null"},
		{"an error {}", []any{errors.New("boom")}, "an error boom"},
		{"a number {}", []any{7}, "a number 7"},
		{"a level {}", []any{Warn}, "a level WARNING"},
		{"unbalanced {", []any{"a"}, "unbalanced { (a)"},
	} {
		if got := Format(testCase.pattern, testCase.args...); got != testCase.want {
			t.Errorf("Format(%q, %v) = %q, want %q", testCase.pattern, testCase.args, got, testCase.want)
		}
	}
}

func TestLoggerWritesEachLevelWithItsLabel(t *testing.T) {
	logger := GetLogger("Component")

	output := capture(t, Debug, func() {
		logger.Debug("debug {}", 1)
		logger.Info("info {}", 2)
		logger.Warn("warn {}", 3)
		logger.Error("error {}", 4)
	})

	for _, want := range []string{"[DEBUG] debug 1", "[INFO] info 2", "[WARNING] warn 3", "[ERROR] error 4"} {
		if !strings.Contains(output, want) {
			t.Errorf("output does not contain %q:\n%s", want, output)
		}
	}
	if logger.Name() != "Component" {
		t.Errorf("Name = %q, want %q", logger.Name(), "Component")
	}
}

func TestLoggerDropsRecordsBelowTheThreshold(t *testing.T) {
	logger := GetLogger("Component")

	output := capture(t, Warn, func() {
		logger.Debug("dropped")
		logger.Info("dropped")
		logger.Warn("kept")
	})

	if strings.Contains(output, "dropped") {
		t.Errorf("a record below the threshold was written:\n%s", output)
	}
	if !strings.Contains(output, "kept") {
		t.Errorf("a record at the threshold was dropped:\n%s", output)
	}
	if logger.Enabled(Debug) {
		t.Error("Enabled reported a level below the threshold as enabled")
	}
}

func TestParseLevelAcceptsTheUsualSpellings(t *testing.T) {
	for raw, want := range map[string]Level{
		"debug":    Debug,
		"TRACE":    Debug,
		" info ":   Info,
		"warn":     Warn,
		"WARNING":  Warn,
		"error":    Error,
		"nonsense": Info,
		"":         Info,
	} {
		if got := ParseLevel(raw); got != want {
			t.Errorf("ParseLevel(%q) = %v, want %v", raw, got, want)
		}
	}
}

func TestLevelNamesItself(t *testing.T) {
	for level, want := range map[Level]string{Debug: "DEBUG", Info: "INFO", Warn: "WARNING", Error: "ERROR"} {
		if got := level.String(); got != want {
			t.Errorf("%d.String() = %q, want %q", level, got, want)
		}
	}
	if got := Level(99).String(); got != "INFO" {
		t.Errorf("an unknown level renders as %q, want %q", got, "INFO")
	}
}
