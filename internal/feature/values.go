package feature

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/spf13/cobra"
)

// ValidateValues checks the positional arguments of a direct call
// `aicli add <feature> ...` (FR-006, FR-014, FR-016).
//
// In: cmd, the feature command, used for its name and usage line; decl, its
// declaration; args, the raw positional arguments.
// Out: the arguments trimmed of leading and trailing whitespace, and nil if
// all checks pass. Otherwise nil and the first failure, checked in this
// order: too few arguments ("<feature> requires <input>", naming the first
// missing input); too many ("<feature> takes <n> argument(s), got <m>");
// then, input by input in declaration order, an empty value, a value with
// control characters (C0, C1 or DEL), a value not among the choices (exact
// match, EC-009) and a validator message. The validator is called only with
// a non-empty value free of control characters. Each error message is the
// plain text after "Error: " and its second line is "Usage: " followed by
// the usage line, built after the help flag is initialised so it ends in
// "[flags]".
// Side effects: initialises the default help flag of cmd; calls the
// validators of the inputs.
func ValidateValues(cmd *cobra.Command, decl *Declaration, args []string) ([]string, error) {
	cmd.InitDefaultHelpFlag()
	fail := func(format string, a ...any) ([]string, error) {
		return nil, fmt.Errorf("%s\nUsage: %s", fmt.Sprintf(format, a...), cmd.UseLine())
	}
	n := len(decl.Inputs)
	if len(args) < n {
		return fail("%s requires %s", decl.Name, decl.Inputs[len(args)].Name)
	}
	if len(args) > n {
		return fail("%s takes %d argument(s), got %d", decl.Name, n, len(args))
	}
	values := make([]string, n)
	for i, in := range decl.Inputs {
		v := strings.TrimSpace(args[i])
		values[i] = v
		switch {
		case v == "":
			return fail("%s cannot be empty", in.Name)
		case strings.IndexFunc(v, unicode.IsControl) >= 0:
			return fail("%s contains control characters", in.Name)
		case len(in.Choices) > 0 && !contains(in.Choices, v):
			return fail("%s must be one of: %s", in.Name, strings.Join(in.Choices, ", "))
		case in.Validator != nil:
			if msg := in.Validator(v); msg != "" {
				return fail("%s", msg)
			}
		}
	}
	return values, nil
}

// contains reports whether list has an element equal to s.
//
// In: list, the elements; s, the value. Out: true on an exact match.
// No side effects.
func contains(list []string, s string) bool {
	for _, e := range list {
		if e == s {
			return true
		}
	}
	return false
}
