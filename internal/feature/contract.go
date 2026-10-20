package feature

import (
	"fmt"

	"github.com/spf13/cobra"
)

// CheckContract verifies that every feature under add follows the feature
// contract (FR-003).
//
// In: add, the `add` command; its direct runnable subcommands, except
// "help", are the features.
// Out: one error per broken rule, in Cobra order and then rule order, each
// naming the feature and the rule. The rules are: the feature declares
// aliases; an input name is empty or duplicated; an input declares an empty
// choice list (non-nil and zero length); a choice is empty or duplicated; an
// input declares both choices and a validator; the feature has no
// declaration or its declaration has no plan. A valid tree gives nil.
// No side effects.
func CheckContract(add *cobra.Command) []error {
	var errs []error
	for _, cmd := range add.Commands() {
		if !cmd.Runnable() || cmd.Name() == "help" {
			continue
		}
		errs = append(errs, checkFeature(cmd)...)
	}
	return errs
}

// checkFeature applies the FR-003 rules to one feature command.
//
// In: cmd, a direct runnable subcommand of add.
// Out: one error per broken rule, nil if none.
// No side effects.
func checkFeature(cmd *cobra.Command) []error {
	name := cmd.Name()
	var errs []error
	fail := func(format string, args ...any) {
		errs = append(errs, fmt.Errorf("feature %s: %s", name, fmt.Sprintf(format, args...)))
	}
	if len(cmd.Aliases) > 0 {
		fail("declares aliases")
	}
	decl, ok := Lookup(cmd)
	if !ok {
		fail("has no declaration")
		return errs
	}
	if decl.Plan == nil {
		fail("has no plan")
	}
	names := map[string]bool{}
	for i, in := range decl.Inputs {
		switch {
		case in.Name == "":
			fail("input %d has an empty name", i+1)
		case names[in.Name]:
			fail("input name %q is duplicated", in.Name)
		}
		names[in.Name] = true
		if in.Choices != nil && len(in.Choices) == 0 {
			fail("input %q declares an empty choice list", in.Name)
		}
		choices := map[string]bool{}
		for _, c := range in.Choices {
			switch {
			case c == "":
				fail("input %q declares an empty choice", in.Name)
			case choices[c]:
				fail("input %q declares the choice %q twice", in.Name, c)
			}
			choices[c] = true
		}
		if len(in.Choices) > 0 && in.Validator != nil {
			fail("input %q declares both choices and a validator", in.Name)
		}
	}
	return errs
}
