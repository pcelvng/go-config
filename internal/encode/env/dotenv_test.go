package env

import (
	"errors"
	"strings"
	"testing"

	"github.com/hydronica/trial"
)

func TestReadDotenvMap(t *testing.T) {
	fn := func(in string) (map[string]string, error) {
		return readDotenvMap(strings.NewReader(in))
	}
	cases := trial.Cases[string, map[string]string]{
		"default": {
			Input: `NAME=apply
VALUE=23
ENABLE=true
TIME="2010-08-10T00:00:00Z"
FLOAT64=99.9
DURA="10s"`,
			Expected: map[string]string{
				"NAME":    "apply",
				"VALUE":   "23",
				"ENABLE":  "true",
				"TIME":    "2010-08-10T00:00:00Z",
				"FLOAT64": "99.9",
				"DURA":    "10s",
			},
		},
		"empty": {
			Input:    "",
			Expected: map[string]string{},
		},
		"whitespace_only": {
			Input:    "\n\n  \t  \n",
			Expected: map[string]string{},
		},
		"comments_and_blank_lines": {
			Input: `# leading comment

KEY1=a
  # indented comment
KEY2=b

# trailing section
KEY3=c`,
			Expected: map[string]string{
				"KEY1": "a",
				"KEY2": "b",
				"KEY3": "c",
			},
		},
		"single_quoted_value": {
			Input:    `MSG='hello world'`,
			Expected: map[string]string{"MSG": "hello world"},
		},
		"single_quoted_escapes": {
			Input:    `APOS='it\'s fine'`,
			Expected: map[string]string{"APOS": "it's fine"},
		},
		"single_quoted_backslash": {
			Input:    `P='C:\\share\\path'`,
			Expected: map[string]string{"P": `C:\share\path`},
		},
		"double_quoted_with_escapes": {
			Input: `LINE="a\nb\tc\""
PATH="C:\\temp"`,
			Expected: map[string]string{
				"LINE": "a\nb\tc\"",
				"PATH": "C:\\temp",
			},
		},
		"double_quoted_with_comment": {
			Input: `LINE="a\nb\tc\""#hello world
PATH="C:\\temp"`,
			Expected: map[string]string{
				"LINE": "a\nb\tc\"",
				"PATH": `C:\temp`,
			},
		},
		"double_quoted_with_comment+quote": {
			Input: `LINE="a\nb\tc\""#hello world"
PATH="C:\\temp"`,
			Expected: map[string]string{
				"LINE": "a\nb\tc\"",
				"PATH": `C:\temp`,
			},
		},
		"inline_space_hash_strips_comment": {
			Input:    `FOO=bar baz # not used`,
			Expected: map[string]string{"FOO": "bar baz"},
		},
		"inline_tab_before_hash_strips_comment": {
			Input:    "FOO=bar\t#tail",
			Expected: map[string]string{"FOO": "bar"},
		},
		"quoted_value_preserves_hash": {
			Input:    `FOO="bar # still inside"`,
			Expected: map[string]string{"FOO": "bar # still inside"},
		},
		"url_hash_not_comment": {
			Input:    `REF=https://host.example/path#fragment`,
			Expected: map[string]string{"REF": "https://host.example/path#fragment"},
		},
		"line_without_equals_skipped": {
			Input: `NOT_A_VAR_LINE
OK=yes`,
			Expected: map[string]string{"OK": "yes"},
		},
		"empty_key_skipped": {
			Input: `=nope
A=ok`,
			Expected: map[string]string{"A": "ok"},
		},
		"first_equals_splits_key_and_value": {
			Input:    `URL=https://x.example?q=a=b`,
			Expected: map[string]string{"URL": "https://x.example?q=a=b"},
		},
		"trimmed_key_and_value": {
			Input:    "  KEY  =  spaced  ",
			Expected: map[string]string{"KEY": "spaced"},
		},
		"duplicate_key_last_wins": {
			Input: `X=first
X=second`,
			Expected: map[string]string{"X": "second"},
		},
		"unquoted_value_trimmed": {
			Input:    "PORT=8080",
			Expected: map[string]string{"PORT": "8080"},
		},
		"empty_double_quoted": {
			Input:    `EMPTY=""`,
			Expected: map[string]string{"EMPTY": ""},
		},
		"unterminated_double_quote": {
			Input: `OK=1
BAD="no closing quote
FINE=2`,
			ExpectedErr: errors.New("unterminated quoted"),
		},
		"unterminated_double_quote_at_eof": {
			Input:       `X="still open`,
			ExpectedErr: errors.New("unterminated quoted"),
		},
		"unterminated_single_quote": {
			Input:       "X='oops",
			ExpectedErr: errors.New("unterminated quoted"),
		},
	}
	trial.New(fn, cases).SubTest(t)
}

func TestUnescapeEnvDoubleQuoted(t *testing.T) {
	fn := func(in string) (string, error) {
		return unescape2Quoted(in), nil
	}
	cases := trial.Cases[string, string]{
		"empty": {
			Input:    "",
			Expected: "",
		},
		"no_escapes": {
			Input:    `hello`,
			Expected: "hello",
		},
		"newline_tab_cr": {
			Input:    `a\nb\tc\rd`,
			Expected: "a\nb\tc\rd",
		},
		"quote_and_backslash": {
			Input:    `say \"hi\" and \\`,
			Expected: `say "hi" and \`,
		},
		"unknown_escape_drops_backslash": {
			Input:    `a\zb`,
			Expected: "azb",
		},
		"trailing_backslash": {
			Input:    `x\`,
			Expected: `x\`,
		},
		"apostrophe_and_backslash": {
			Input:    `it\'s \\one`,
			Expected: `it's \one`,
		},
	}
	trial.New(fn, cases).SubTest(t)
}

func TestUnescapeEnvSingleQuoted(t *testing.T) {
	fn := func(in string) (string, error) {
		return unescape1Quoted(in), nil
	}
	cases := trial.Cases[string, string]{
		"empty": {
			Input:    "",
			Expected: "",
		},
		"no_escapes": {
			Input:    `hello`,
			Expected: "hello",
		},
		"apostrophe_and_backslash": {
			Input:    `it\'s \\one`,
			Expected: `it's \one`,
		},
		"backslash_n_is_not_newline": {
			Input:    `a\nb`,
			Expected: "anb",
		},
		"unknown_escape_drops_backslash": {
			Input:    `a\zb`,
			Expected: "azb",
		},
		"trailing_backslash": {
			Input:    `x\`,
			Expected: `x\`,
		},
	}
	trial.New(fn, cases).SubTest(t)
}
