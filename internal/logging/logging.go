// Package logging provides the levelled, brace-placeholder logger the front-end
// plugin logs through.
//
// The Java original logs through SLF4J and lets the Maven container decide how a
// record is rendered. Go has no such container, so this package supplies both
// halves: the SLF4J-shaped call surface (four levels, "{}" placeholders,
// per-class loggers) and a Maven-flavoured renderer, so that the *content* of
// every message — which is contract — is produced by exactly the same format
// strings as the original.
package logging

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
)

// Level orders the four severities SLF4J exposes to this codebase.
type Level int

// The levels, in increasing severity.
const (
	Debug Level = iota
	Info
	Warn
	Error
)

var levelNames = map[Level]string{
	Debug: "DEBUG",
	Info:  "INFO",
	Warn:  "WARNING",
	Error: "ERROR",
}

// String renders the level the way Maven labels it on the console. A level
// outside the four known ones reads as Info, which is where the threshold sits
// by default.
func (l Level) String() string {
	if name, ok := levelNames[l]; ok {
		return name
	}
	return levelNames[Info]
}

// ParseLevel maps a level name, in any case, onto a Level. Unknown names fall
// back to Info, which is the level the plugin runs at by default.
func ParseLevel(raw string) Level {
	switch strings.ToUpper(strings.TrimSpace(raw)) {
	case "DEBUG", "TRACE":
		return Debug
	case "WARN", "WARNING":
		return Warn
	case "ERROR":
		return Error
	default:
		return Info
	}
}

var (
	mu        sync.Mutex
	out       io.Writer = os.Stdout
	threshold           = Info
)

// SetOutput redirects every logger's output. It returns the previous writer so
// a caller — a test, usually — can put the old one back.
func SetOutput(w io.Writer) io.Writer {
	mu.Lock()
	defer mu.Unlock()
	previous := out
	out = w
	return previous
}

// SetLevel raises or lowers the threshold below which records are dropped, and
// returns the previous threshold.
func SetLevel(l Level) Level {
	mu.Lock()
	defer mu.Unlock()
	previous := threshold
	threshold = l
	return previous
}

// Logger is one named logging channel, the equivalent of an SLF4J Logger
// obtained for a class.
type Logger struct {
	name string
}

// GetLogger returns the logger named for a component, mirroring
// LoggerFactory.getLogger(getClass()).
func GetLogger(name string) *Logger {
	return &Logger{name: name}
}

// Name reports the channel name this logger was created with.
func (l *Logger) Name() string {
	return l.name
}

// Debug logs at DEBUG, substituting args into the "{}" placeholders of pattern.
func (l *Logger) Debug(pattern string, args ...any) {
	l.log(Debug, pattern, args...)
}

// Info logs at INFO.
func (l *Logger) Info(pattern string, args ...any) {
	l.log(Info, pattern, args...)
}

// Warn logs at WARNING.
func (l *Logger) Warn(pattern string, args ...any) {
	l.log(Warn, pattern, args...)
}

// Error logs at ERROR.
func (l *Logger) Error(pattern string, args ...any) {
	l.log(Error, pattern, args...)
}

// Enabled reports whether a record at the given level would be emitted.
func (l *Logger) Enabled(level Level) bool {
	mu.Lock()
	defer mu.Unlock()
	return level >= threshold
}

func (l *Logger) log(level Level, pattern string, args ...any) {
	if !l.Enabled(level) {
		return
	}
	mu.Lock()
	defer mu.Unlock()
	fmt.Fprintf(out, "[%s] %s\n", level, Format(pattern, args...))
}

// Format substitutes args, in order, into the "{}" placeholders of pattern. A
// surplus placeholder is left as it stands and a surplus argument is appended in
// parentheses, which is what SLF4J's MessageFormatter does.
func Format(pattern string, args ...any) string {
	if len(args) == 0 {
		return pattern
	}
	var b strings.Builder
	next := 0
	for i := 0; i < len(pattern); i++ {
		if pattern[i] == '{' && i+1 < len(pattern) && pattern[i+1] == '}' && next < len(args) {
			b.WriteString(render(args[next]))
			next++
			i++
			continue
		}
		b.WriteByte(pattern[i])
	}
	for ; next < len(args); next++ {
		b.WriteString(" (")
		b.WriteString(render(args[next]))
		b.WriteString(")")
	}
	return b.String()
}

func render(arg any) string {
	switch value := arg.(type) {
	case nil:
		return "null"
	case string:
		return value
	case error:
		return value.Error()
	case fmt.Stringer:
		return value.String()
	default:
		return fmt.Sprintf("%v", value)
	}
}
