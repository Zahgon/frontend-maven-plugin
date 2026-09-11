package frontend

// ArgumentsParser splits a single configured argument string into the argument
// vector handed to a front-end tool, and appends any arguments the runner adds
// on the caller's behalf.
type ArgumentsParser struct {
	additionalArguments []string
}

// NewArgumentsParser builds a parser that adds no arguments of its own.
func NewArgumentsParser() *ArgumentsParser {
	return NewArgumentsParserWith(nil)
}

// NewArgumentsParserWith builds a parser that appends additionalArguments to
// every parse result, skipping any that the parsed string already contains.
func NewArgumentsParserWith(additionalArguments []string) *ArgumentsParser {
	return &ArgumentsParser{additionalArguments: additionalArguments}
}

// Parse splits args on whitespace, respecting quoted phrases.
//
// A phrase opened by a single or double quote runs to the matching quote or to
// the end of the string, and the quote characters stay in the produced argument.
// The other kind of quote inside a quoted phrase is ignored. Every character
// except the whitespace that splits is left in place.
//
// Examples:
//
//	"foo bar"             -> ["foo", "bar"]
//	"foo \"bar foobar\""  -> ["foo", "\"bar foobar\""]
//	"foo 'bar"            -> ["foo", "'bar"]
//
// The empty string and the literal "null" both parse to no arguments; "null" is
// what an unset configuration property renders as.
func (p *ArgumentsParser) Parse(args string) []string {
	if args == "" || args == "null" {
		return []string{}
	}

	arguments := make([]string, 0, 8)
	builder := make([]rune, 0, len(args))
	var quote rune
	quoted := false

	for _, c := range args {
		if isJavaWhitespace(c) && !quoted {
			arguments, builder = addArgument(builder, arguments)
			continue
		} else if c == '"' || c == '\'' {
			if !quoted {
				quote = c
				quoted = true
			} else if quote == c {
				quoted = false
			}
			// A quoted argument containing the other kind of quote is left alone.
		}
		builder = append(builder, c)
	}

	arguments, _ = addArgument(builder, arguments)

	for _, argument := range p.additionalArguments {
		if !contains(arguments, argument) {
			arguments = append(arguments, argument)
		}
	}

	return arguments
}

func addArgument(builder []rune, arguments []string) ([]string, []rune) {
	if len(builder) > 0 {
		arguments = append(arguments, string(builder))
		builder = builder[:0]
	}
	return arguments, builder
}

func contains(haystack []string, needle string) bool {
	for _, candidate := range haystack {
		if candidate == needle {
			return true
		}
	}
	return false
}

// isJavaWhitespace answers Character.isWhitespace, which is not Go's
// unicode.IsSpace: Java counts the four information separators as whitespace and
// refuses the three non-breaking spaces. Where an argument string splits is
// contract, so the Java definition is reproduced rather than substituted.
func isJavaWhitespace(c rune) bool {
	switch c {
	case '\u00a0', '\u2007', '\u202f':
		// Space separators that Java explicitly excludes.
		return false
	case '\t', '\n', '\u000b', '\f', '\r',
		'\u001c', '\u001d', '\u001e', '\u001f':
		return true
	case ' ', '\u1680', '\u2028', '\u2029', '\u205f', '\u3000':
		return true
	}
	// The remainder of the space-separator category: U+2000..U+200A.
	return c >= '\u2000' && c <= '\u200a'
}
