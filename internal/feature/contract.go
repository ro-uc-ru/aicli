package feature

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/ro-uc-ru/aicli/internal/gen"
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
// declaration or its declaration has no plan. For a multi-choice input
// (FR-001) the rules also are: it is not the last input; it declares no
// choices; it has a validator (the rule above); one of its item files is not
// a create entry of the plan for every non-empty set of values that includes
// its choice; a shared entry renders different bytes for different non-empty
// sets of values. A valid tree gives nil.
// No side effects, except those of the Plan functions, which are called with
// sample values.
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
		if in.Multi && i != len(decl.Inputs)-1 {
			fail("multi-choice input %q must be the last input", in.Name)
		}
		if in.Multi && len(in.Choices) == 0 {
			fail("multi-choice input %q declares no choices", in.Name)
		} else if in.Choices != nil && len(in.Choices) == 0 {
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
	if n := len(decl.Inputs); n > 0 && decl.Plan != nil && multiChoicesValid(decl.Inputs[n-1]) {
		multiPlanErrors(decl, fail)
	}
	return errs
}

// multiPlanErrors applies the plan rules of a multi-choice input (FR-001).
//
// In: decl, a declaration whose last input is multi-choice, with distinct,
// non-empty choices; fail, the function that records one broken rule.
// Out: none; it calls fail at most once per rule (per choice for the item
// files, once for the shared entries). The Plan is called once for every
// non-empty subset of the choices, in choice order, with the first choice (or
// "x") for each ordinary input; a Plan error is reported as a broken rule.
// Side effects: calls decl.Plan.
func multiPlanErrors(decl *Declaration, fail func(string, ...any)) {
	last := len(decl.Inputs) - 1
	in := decl.Inputs[last]
	prefix := make([]string, last)
	for i := range prefix {
		prefix[i] = "x"
		if c := decl.Inputs[i].Choices; len(c) > 0 {
			prefix[i] = c[0]
		}
	}
	reported := map[string]bool{}
	shared := map[string][]byte{}
	sharedDone := false
	for mask := 1; mask < 1<<len(in.Choices); mask++ {
		var subset []string
		for i, c := range in.Choices {
			if mask&(1<<i) != 0 {
				subset = append(subset, c)
			}
		}
		entries, err := decl.Plan(append(append([]string(nil), prefix...), subset...))
		if err != nil {
			fail("plan failed for %s %q: %v", in.Name, strings.Join(subset, " "), err)
			return
		}
		created := map[string]bool{}
		for _, e := range entries {
			if e.Action == gen.ActionCreate {
				created[e.Path] = true
			}
		}
		for _, c := range subset {
			if p := in.ItemFiles[c]; p != "" && !created[p] && !reported[c] {
				reported[c] = true
				fail("item file %q of choice %q is not a create entry of the plan for %s %q", p, c, in.Name, strings.Join(subset, " "))
			}
		}
		if sharedDone {
			continue
		}
		for _, e := range entries {
			if e.Action != gen.ActionShared {
				continue
			}
			got, err := gen.RenderEntry(e)
			if err != nil {
				fail("shared entry cannot be rendered: %v", err)
				sharedDone = true
				break
			}
			if prev, ok := shared[e.Path]; ok && !bytes.Equal(prev, got) {
				fail("shared entry %q depends on the values of %q", e.Path, in.Name)
				sharedDone = true
				break
			}
			shared[e.Path] = got
		}
	}
}

// multiChoicesValid reports whether an input is multi-choice with a non-empty
// list of distinct, non-empty choices, which the plan rules need.
//
// In: in, the last input of a feature. Out: true if so. No side effects.
func multiChoicesValid(in Input) bool {
	if !in.Multi || len(in.Choices) == 0 {
		return false
	}
	seen := map[string]bool{}
	for _, c := range in.Choices {
		if c == "" || seen[c] {
			return false
		}
		seen[c] = true
	}
	return true
}
