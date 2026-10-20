package feature

import (
	"reflect"
	"strings"
	"testing"
)

// posixSplit splits s into words as a POSIX shell does for the subset used by
// EquivalentCommand: unquoted text, single quotes and backslash-escaped
// characters outside quotes. It returns an error text for unterminated quotes.
//
// In: s, the command line. Out: the words and "" or an error text.
// No side effects.
func posixSplit(s string) ([]string, string) {
	var words []string
	var cur strings.Builder
	inWord, inQuote := false, false
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case inQuote:
			if c == '\'' {
				inQuote = false
			} else {
				cur.WriteByte(c)
			}
		case c == '\'':
			inQuote, inWord = true, true
		case c == '\\' && i+1 < len(s):
			i++
			cur.WriteByte(s[i])
			inWord = true
		case c == ' ':
			if inWord {
				words = append(words, cur.String())
				cur.Reset()
				inWord = false
			}
		default:
			cur.WriteByte(c)
			inWord = true
		}
	}
	if inQuote {
		return nil, "unterminated quote"
	}
	if inWord {
		words = append(words, cur.String())
	}
	return words, ""
}

// TestEquivalentCommand checks the quoting rule and the "--" separator, and
// that splitting the output like a POSIX shell gives back the original values.
//
// Covers: FR-018, EC-015, EC-016
func TestEquivalentCommand(t *testing.T) {
	tests := []struct {
		name   string
		values []string
		want   string
	}{
		{"none", nil, "aicli add f"},
		{"plain", []string{"abc", "A_1@%+=:,./-x"}, "aicli add f abc A_1@%+=:,./-x"},
		{"spaces", []string{"hola mundo"}, "aicli add f 'hola mundo'"},
		{"quote", []string{"it's"}, `aicli add f 'it'\''s'`},
		{"empty", []string{"", "a"}, "aicli add f '' a"},
		{"non-ascii", []string{"añb"}, "aicli add f 'añb'"},
		{"metachar", []string{"a;b", "$HOME", "`x`", "a|b", "*", "a\"b", `a\b`, "a&b", "(x)", "a#b", "~"},
			`aicli add f 'a;b' '$HOME' '` + "`x`" + `' 'a|b' '*' 'a"b' 'a\b' 'a&b' '(x)' 'a#b' '~'`},
		{"leading dash", []string{"-x", "b"}, "aicli add f -- -x b"},
		{"dash later", []string{"a", "-x"}, "aicli add f -- a -x"},
		{"dash inside", []string{"a-b"}, "aicli add f a-b"},
		{"only dash", []string{"-"}, "aicli add f -- -"},
		{"newline", []string{"a\nb"}, "aicli add f 'a\nb'"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := EquivalentCommand("f", tc.values)
			if got != tc.want {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
			words, msg := posixSplit(got)
			if msg != "" {
				t.Fatal(msg)
			}
			want := []string{"aicli", "add", "f"}
			for _, v := range tc.values {
				if strings.HasPrefix(v, "-") {
					want = append(want, "--")
					break
				}
			}
			want = append(want, tc.values...)
			if !reflect.DeepEqual(words, want) {
				t.Fatalf("split = %q, want %q", words, want)
			}
		})
	}
}
