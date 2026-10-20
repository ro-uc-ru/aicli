package feature

import "strings"

// EquivalentCommand builds the command line that a user could type to run a
// feature with the given values (FR-018, EC-015, EC-016).
//
// In: name, the feature name; values, the input values in declaration order.
// Out: "aicli add <name>" followed by each value. A value is shown as is when
// it is non-empty ASCII made of letters, digits and "_ @ % + = : , . / -";
// otherwise it is wrapped in single quotes with each "'" written as quote, backslash, quote, quote (four characters).
// "--" is inserted before the first value when any value starts with "-".
// No side effects.
func EquivalentCommand(name string, values []string) string {
	parts := []string{"aicli", "add", name}
	for _, v := range values {
		if strings.HasPrefix(v, "-") {
			parts = append(parts, "--")
			break
		}
	}
	for _, v := range values {
		parts = append(parts, shellWord(v))
	}
	return strings.Join(parts, " ")
}

// shellWord quotes one value for a POSIX shell.
//
// In: v, the value. Out: v itself if it is non-empty and made only of ASCII
// letters, digits and "_ @ % + = : , . / -"; otherwise v in single quotes
// with each "'" replaced by quote, backslash, quote, quote.
// No side effects.
func shellWord(v string) string {
	if v != "" && strings.IndexFunc(v, func(r rune) bool { return !isPlain(r) }) < 0 {
		return v
	}
	return "'" + strings.ReplaceAll(v, "'", `'\''`) + "'"
}

// isPlain reports whether r may appear unquoted in a shell word.
//
// In: r, the rune. Out: true for ASCII letters, digits and "_ @ % + = : , . / -".
// No side effects.
func isPlain(r rune) bool {
	switch {
	case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
		return true
	}
	return strings.ContainsRune("_@%+=:,./-", r)
}
