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
// If the last input is multi-choice (FR-002, FR-004) it takes every remaining
// argument: "too few" means no value for it ("<feature> requires <input>"),
// there is no "too many" check, each value is checked in the given order as
// above and then for repetition after trimming ("<input> "<value>" is given
// more than once", naming the first repeated value), and its values are
// returned in choice declaration order, after the other inputs' values.
// Side effects: initialises the default help flag of cmd; calls the
// validators of the inputs.
func ValidateValues(cmd *cobra.Command, decl *Declaration, args []string) ([]string, error) {
	cmd.InitDefaultHelpFlag()
	fail := func(format string, a ...any) ([]string, error) {
		return nil, fmt.Errorf("%s\nUsage: %s", fmt.Sprintf(format, a...), cmd.UseLine())
	}
	n := len(decl.Inputs)
	multi := n > 0 && decl.Inputs[n-1].Multi
	if len(args) < n {
		return fail("%s requires %s", decl.Name, decl.Inputs[len(args)].Name)
	}
	if len(args) > n && !multi {
		return fail("%s takes %d argument(s), got %d", decl.Name, n, len(args))
	}
	values := make([]string, len(args))
	for i := range args {
		in := decl.Inputs[min(i, n-1)]
		v := strings.TrimSpace(args[i])
		values[i] = v
		switch {
		case v == "":
			return fail("%s cannot be empty", in.Name)
		case strings.IndexFunc(v, unicode.IsControl) >= 0:
			return fail("%s contains control characters", in.Name)
		case len(in.Choices) > 0 && !contains(in.Choices, v):
			return fail("%s must be one of: %s", in.Name, strings.Join(in.Choices, ", "))
		case in.Multi && contains(values[n-1:i], v):
			return fail("%s %q is given more than once", in.Name, v)
		case in.Validator != nil:
			if msg := in.Validator(v); msg != "" {
				return fail("%s", msg)
			}
		}
	}
	if multi {
		values = append(values[:n-1:n-1], inChoiceOrder(decl.Inputs[n-1].Choices, values[n-1:])...)
	}
	return values, nil
}

// inChoiceOrder sorts distinct values by their position in choices.
//
// In: choices, the declared choices; vals, distinct members of choices.
// Out: a new slice with the elements of vals in the order of choices.
// No side effects.
func inChoiceOrder(choices, vals []string) []string {
	out := make([]string, 0, len(vals))
	for _, c := range choices {
		if contains(vals, c) {
			out = append(out, c)
		}
	}
	return out
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
