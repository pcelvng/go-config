package env

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
	"unicode"
	"unicode/utf8"
)

// LoadEnvFile opens path, parses dotenv lines into a map, and unmarshals into v via Decoder.
// Callers may use readDotenvMap + Decoder directly for other flows.
func LoadEnvFile(path string, v interface{}) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	m, err := readDotenvMap(f)
	if err != nil {
		return err
	}
	d := &Decoder{
		GetVal: func(k string) string { return m[k] },
	}
	return d.Unmarshal(v)
}

// readDotenvMap reads r line by line. Each non-empty, non-comment line is split on the first
// '=' into key and value (trimmed). Malformed quoted values return an error and a nil map.
func readDotenvMap(r io.Reader) (map[string]string, error) {
	sc := bufio.NewScanner(r)
	vars := make(map[string]string)
	lineNo := 0
	for sc.Scan() {
		lineNo++
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		eq := strings.Index(line, "=")
		if eq < 0 {
			continue
		}
		key := strings.TrimSpace(line[:eq])
		if key == "" {
			continue
		}
		val, err := parseDotenvLineValue(line[eq+1:])
		if err != nil {
			return nil, fmt.Errorf("dotenv: line %d: %w", lineNo, err)
		}
		vars[key] = val
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	return vars, nil
}

func parseDotenvLineValue(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if len(raw) == 0 {
		return "", nil
	}

	//what does our quote look like?
	q := raw[0]
	if q != '"' && q != '\'' {
		return trimUnquotedInlineComment(raw), nil
	}

	closeIdx, err := findClosingQuote(raw, q)
	if err != nil {
		return "", err
	}
	if err := validateQuotedValueSuffix(raw, closeIdx+1); err != nil {
		return "", err
	}
	if q == '\'' { //literal
		return unescape1Quoted(raw[1:closeIdx]), nil
	}
	return unescape2Quoted(raw[1:closeIdx]), nil

}

// findClosingQuote returns the index of the closing quote in s. literal is true for
// single-quoted values (delimiter '\”), false for double-quoted ('"'). Backslash escapes
// the next byte (same rules as the corresponding unescapeEnv* function).
func findClosingQuote(s string, delimiter byte) (int, error) {
	if len(s) < 2 || s[0] != delimiter {
		return 0, fmt.Errorf("unterminated quoted value")
	}
	for i := 1; i < len(s); i++ {
		if s[i] == '\\' && i+1 < len(s) {
			i++
			continue
		}
		if s[i] == delimiter {
			return i, nil
		}
	}
	return 0, fmt.Errorf("unterminated quoted value")
}

// validateQuotedValueSuffix requires bytes after the closing quote to be empty or an inline
// comment (trimmed remainder empty or starting with '#').
func validateQuotedValueSuffix(s string, from int) error {
	rest := strings.TrimSpace(s[from:])
	if rest == "" || strings.HasPrefix(rest, "#") {
		return nil
	}
	return fmt.Errorf("unexpected trailing content after quoted value")
}

// trimUnquotedInlineComment removes a trailing comment starting at the first '#' preceded by Unicode whitespace.
func trimUnquotedInlineComment(s string) string {
	i := 0
	for i < len(s) {
		if s[i] == '#' && i > 0 {
			r, _ := utf8.DecodeLastRuneInString(s[:i])
			if unicode.IsSpace(r) {
				return strings.TrimSpace(s[:i])
			}
		}
		_, sz := utf8.DecodeRuneInString(s[i:])
		i += sz
	}
	return strings.TrimSpace(s)
}

func unescape2Quoted(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] == '\\' && i+1 < len(s) {
			switch s[i+1] {
			case 'n':
				b.WriteByte('\n')
			case 'r':
				b.WriteByte('\r')
			case 't':
				b.WriteByte('\t')
			case '"', '\\':
				b.WriteByte(s[i+1])
			default:
				b.WriteByte(s[i+1])
			}
			i++
			continue
		}
		b.WriteByte(s[i])
	}
	return b.String()
}

func unescape1Quoted(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] == '\\' && i+1 < len(s) {
			switch s[i+1] {
			case '\'':
				b.WriteByte('\'')
			case '\\':
				b.WriteByte('\\')
			default:
				b.WriteByte(s[i+1])
			}
			i++
			continue
		}
		b.WriteByte(s[i])
	}
	return b.String()
}

func needsEnvQuotes(s string) bool {
	if s == "" {
		return true
	}
	for _, r := range s {
		if unicode.IsSpace(r) || r == '#' || r == '=' || r == '"' || r == '\'' || r == '\\' {
			return true
		}
	}
	return false
}

func quoteEnvString(s string) string {
	var b strings.Builder
	b.WriteByte('"')
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '\\':
			b.WriteString(`\\`)
		case '"':
			b.WriteString(`\"`)
		case '\n':
			b.WriteString(`\n`)
		case '\r':
			b.WriteString(`\r`)
		case '\t':
			b.WriteString(`\t`)
		default:
			b.WriteByte(s[i])
		}
	}
	b.WriteByte('"')
	return b.String()
}

func formatEnvScalar(s string) string {
	if needsEnvQuotes(s) {
		return quoteEnvString(s)
	}
	return s
}
