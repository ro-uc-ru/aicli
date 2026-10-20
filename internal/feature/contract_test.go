package feature

import (
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/ro-uc-ru/aicli/internal/gen"
)

// plan is a no-op plan for test features.
func plan([]string) ([]gen.FeatureFile, error) { return nil, nil }

// okDecl returns a valid declaration named name.
func okDecl(name string) Declaration {
	return Declaration{Name: name, Plan: plan, Inputs: []Input{
		{Name: "a"},
		{Name: "b", Choices: []string{"x", "y"}},
		{Name: "c", Validator: func(string) string { return "" }},
	}}
}

// TestCheckContract checks that a valid feature gives no error and that each
// broken rule gives exactly one error naming the feature and the rule.
//
// Covers: FR-003
func TestCheckContract(t *testing.T) {
	val := func(string) string { return "" }
	tests := []struct {
		name  string
		build func() *cobra.Command
		want  string
	}{
		{"aliases", func() *cobra.Command {
			c := NewCommand(okDecl("bad"))
			c.Aliases = []string{"b"}
			return c
		}, "declares aliases"},
		{"empty input name", func() *cobra.Command {
			d := okDecl("bad")
			d.Inputs = []Input{{Name: ""}}
			return NewCommand(d)
		}, "empty name"},
		{"duplicate input name", func() *cobra.Command {
			d := okDecl("bad")
			d.Inputs = []Input{{Name: "a"}, {Name: "a"}}
			return NewCommand(d)
		}, `input name "a" is duplicated`},
		{"empty choice list", func() *cobra.Command {
			d := okDecl("bad")
			d.Inputs = []Input{{Name: "a", Choices: []string{}}}
			return NewCommand(d)
		}, "empty choice list"},
		{"empty choice", func() *cobra.Command {
			d := okDecl("bad")
			d.Inputs = []Input{{Name: "a", Choices: []string{"x", ""}}}
			return NewCommand(d)
		}, "empty choice"},
		{"duplicate choice", func() *cobra.Command {
			d := okDecl("bad")
			d.Inputs = []Input{{Name: "a", Choices: []string{"x", "x"}}}
			return NewCommand(d)
		}, `the choice "x" twice`},
		{"choices and validator", func() *cobra.Command {
			d := okDecl("bad")
			d.Inputs = []Input{{Name: "a", Choices: []string{"x"}, Validator: val}}
			return NewCommand(d)
		}, "both choices and a validator"},
		{"no declaration", func() *cobra.Command {
			return &cobra.Command{Use: "bad", RunE: func(*cobra.Command, []string) error { return nil }}
		}, "no declaration"},
		{"no plan", func() *cobra.Command {
			d := okDecl("bad")
			d.Plan = nil
			return NewCommand(d)
		}, "no plan"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			add := &cobra.Command{Use: "add"}
			add.AddCommand(NewCommand(okDecl("good")), tc.build())
			errs := CheckContract(add)
			if len(errs) != 1 {
				t.Fatalf("got %d errors %v, want exactly 1", len(errs), errs)
			}
			msg := errs[0].Error()
			if !strings.Contains(msg, "feature bad:") || !strings.Contains(msg, tc.want) {
				t.Fatalf("error %q does not name the feature and %q", msg, tc.want)
			}
		})
	}
}

// TestCheckContractValid checks that a valid tree gives no error and that
// help and non-runnable subcommands are skipped.
//
// Covers: FR-003
func TestCheckContractValid(t *testing.T) {
	add := &cobra.Command{Use: "add"}
	add.AddCommand(NewCommand(okDecl("one")), NewCommand(okDecl("two")), &cobra.Command{Use: "group"})
	add.InitDefaultHelpCmd()
	if errs := CheckContract(add); len(errs) != 0 {
		t.Fatalf("valid tree gave errors: %v", errs)
	}
}
