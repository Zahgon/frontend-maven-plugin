package frontend

import (
	"regexp"
	"strings"
)

var windowsAbsolutePath = regexp.MustCompile(`^[a-zA-Z]:\\.*$`)

// Merge concatenates two argument lists into a fresh one.
func Merge(first, second []string) []string {
	result := make([]string, 0, len(first)+len(second))
	result = append(result, first...)
	result = append(result, second...)
	return result
}

// Prepend puts one argument in front of a list, returning a fresh one.
func Prepend(first string, list []string) []string {
	return Merge([]string{first}, list)
}

// Normalize rewrites a "/"-separated path for the local filesystem.
func Normalize(path string) string {
	return strings.ReplaceAll(path, "/", separator)
}

// Implode joins elements with a single space.
//
// The separator argument is accepted and ignored — that is what the original
// does, and its test suite pins the behaviour by passing "Bar" and expecting
// "foo bar". Preserving the quirk keeps every log line the plugin emits
// byte-identical.
func Implode(separator string, elements []string) string {
	_ = separator
	var b strings.Builder
	for i, element := range elements {
		if i > 0 {
			b.WriteString(" ")
		}
		b.WriteString(element)
	}
	return b.String()
}

// IsRelative reports whether a task location still has to be resolved against
// the working or install directory. A POSIX absolute path, a "file:" URI and a
// Windows drive-letter path are all already absolute.
func IsRelative(path string) bool {
	return !strings.HasPrefix(path, "/") &&
		!strings.HasPrefix(path, "file:") &&
		!windowsAbsolutePath.MatchString(path)
}
