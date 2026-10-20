// Package featuretest builds test-only `aicli add` features: a complete valid
// one, one that breaks each rule of the feature contract (FR-003), and the
// edge cases of the listing (hidden, deprecated, no inputs, no short
// description, wide characters). It is imported only by tests; a test of this
// package checks that no non-test file imports it.
package featuretest

import (
	"errors"
	"testing/fstest"

	"github.com/spf13/cobra"

	"github.com/ro-uc-ru/aicli/internal/feature"
	"github.com/ro-uc-ru/aicli/internal/gen"
)

// Names of the test features.
const (
	// Main is the name of the complete test feature.
	Main = "feature-1"
	// NoInputs is the name of the feature that declares no inputs.
	NoInputs = "feature-no-inputs"
	// NoShort is the name of the feature without a short description.
	NoShort = "feature-no-short"
	// Wide is the name of the feature whose name has wide characters.
	Wide = "機能-wide"
	// Hidden is the name of the hidden feature.
	Hidden = "feature-hidden"
	// Deprecated is the name of the deprecated feature.
	Deprecated = "feature-deprecated"
)

// Values of the main feature.
const (
	// FailChoice is the choice of the main feature for which its plan fails.
	FailChoice = "fail"
	// PlanError is the message of the plan failure for FailChoice.
	PlanError = "the fail choice is not supported"
	// BadText is the text value that the validator of the main feature rejects.
	BadText = "bad"
	// ValidatorError is the message of the validator for BadText.
	ValidatorError = "input-2 must not be bad"
)

// NextSteps are the next steps declared by the main feature.
var NextSteps = []string{"Set FEATURE_KEY in .env", "Run uv sync"}

// templates is the in-memory template filesystem of the test features.
var templates = fstest.MapFS{
	"a.tmpl": {Data: []byte("a {{.Choice}} {{.Text}}\n")},
	"b.tmpl": {Data: []byte("b {{.Text}}\n")},
}

// plan computes the two feature files of the main feature.
//
// In: v, the values (choice, text). Out: the files
// "features/<text>.txt" and "features/<text>.md", or an error with PlanError
// when the choice is FailChoice. No side effects.
func plan(v []string) ([]gen.FeatureFile, error) {
	if v[0] == FailChoice {
		return nil, errors.New(PlanError)
	}
	data := map[string]string{"Choice": v[0], "Text": v[1]}
	return []gen.FeatureFile{
		{Path: "features/" + v[1] + ".txt", Templates: templates, Template: "a.tmpl", Data: data},
		{Path: "features/" + v[1] + ".md", Templates: templates, Template: "b.tmpl", Data: data},
	}, nil
}

// onePath is a plan returning one fixed file, for features whose plan is not
// the subject of the test.
//
// In: none used. Out: the single file "features/one.txt". No side effects.
func onePath([]string) ([]gen.FeatureFile, error) {
	return []gen.FeatureFile{{Path: "features/one.txt", Templates: templates, Template: "b.tmpl", Data: map[string]string{"Text": "x"}}}, nil
}

// validator rejects BadText.
//
// In: v, the trimmed value. Out: ValidatorError for BadText, "" otherwise.
// No side effects.
func validator(v string) string {
	if v == BadText {
		return ValidatorError
	}
	return ""
}

// base returns a valid declaration with the given name, one text input and
// the single-file plan.
//
// In: name, the feature name. Out: the declaration. No side effects.
func base(name string) feature.Declaration {
	return feature.Declaration{
		Name:   name,
		Short:  "Test feature " + name,
		Inputs: []feature.Input{{Name: "input-1", Description: "The first input"}},
		Plan:   onePath,
	}
}

// NewMain builds the complete test feature: a choice input (one, two and
// FailChoice), a text input with a validator, two feature files and next steps.
//
// In: none. Out: a new command named Main. No side effects.
func NewMain() *cobra.Command {
	return feature.NewCommand(feature.Declaration{
		Name:  Main,
		Short: "Add the main test feature",
		Long:  "Long description of the main test feature.",
		Inputs: []feature.Input{
			{Name: "input-1", Description: "A choice", Choices: []string{"one", "two", FailChoice}},
			{Name: "input-2", Description: "A text", Validator: validator},
		},
		Plan:      plan,
		NextSteps: NextSteps,
	})
}

// NewNoInputs builds a valid feature that declares no inputs.
//
// In: none. Out: a new command named NoInputs. No side effects.
func NewNoInputs() *cobra.Command {
	d := base(NoInputs)
	d.Inputs = nil
	return feature.NewCommand(d)
}

// NewNoShort builds a valid feature without a short description (EC-007).
//
// In: none. Out: a new command named NoShort. No side effects.
func NewNoShort() *cobra.Command {
	d := base(NoShort)
	d.Short = ""
	return feature.NewCommand(d)
}

// NewWide builds a valid feature whose name and short description contain
// characters wider than one cell (EC-006).
//
// In: none. Out: a new command named Wide. No side effects.
func NewWide() *cobra.Command {
	d := base(Wide)
	d.Short = "広い説明 😀"
	return feature.NewCommand(d)
}

// NewHidden builds a valid hidden feature (EC-002).
//
// In: none. Out: a new command named Hidden with Hidden set. No side effects.
func NewHidden() *cobra.Command {
	c := feature.NewCommand(base(Hidden))
	c.Hidden = true
	return c
}

// NewDeprecated builds a valid deprecated feature (EC-002).
//
// In: none. Out: a new command named Deprecated with a Deprecated notice.
// No side effects.
func NewDeprecated() *cobra.Command {
	c := feature.NewCommand(base(Deprecated))
	c.Deprecated = "use feature-1"
	return c
}

// Valid returns one new instance of every valid test feature, in a fixed
// order: Main, NoInputs, NoShort, Wide, Hidden, Deprecated.
//
// In: none. Out: the commands. No side effects.
func Valid() []*cobra.Command {
	return []*cobra.Command{NewMain(), NewNoInputs(), NewNoShort(), NewWide(), NewHidden(), NewDeprecated()}
}

// Broken is a test feature that breaks exactly one rule of the contract.
type Broken struct {
	// Rule names the broken rule.
	Rule string
	// Message is the text that the CheckContract error for this rule contains.
	Message string
	// Cmd is the feature command.
	Cmd *cobra.Command
}

// BrokenFeatures returns one new feature per contract rule, each named
// "broken-<n>" and breaking only its rule: aliases; empty input name;
// duplicate input name; empty choice list; empty choice; duplicate choice;
// choices with a validator; no declaration; no plan.
//
// In: none. Out: the features. No side effects.
func BrokenFeatures() []Broken {
	mk := func(name string, edit func(*feature.Declaration)) *cobra.Command {
		d := base(name)
		edit(&d)
		return feature.NewCommand(d)
	}
	aliases := mk("broken-aliases", func(*feature.Declaration) {})
	aliases.Aliases = []string{"alias"}
	noDecl := &cobra.Command{Use: "broken-no-declaration", RunE: func(*cobra.Command, []string) error { return nil }}
	noop := func(string) string { return "" }
	return []Broken{
		{"aliases", "declares aliases", aliases},
		{"empty input name", "empty name", mk("broken-empty-name", func(d *feature.Declaration) {
			d.Inputs = []feature.Input{{Name: ""}}
		})},
		{"duplicate input name", "is duplicated", mk("broken-dup-name", func(d *feature.Declaration) {
			d.Inputs = []feature.Input{{Name: "a"}, {Name: "a"}}
		})},
		{"empty choice list", "empty choice list", mk("broken-empty-list", func(d *feature.Declaration) {
			d.Inputs = []feature.Input{{Name: "a", Choices: []string{}}}
		})},
		{"empty choice", "declares an empty choice", mk("broken-empty-choice", func(d *feature.Declaration) {
			d.Inputs = []feature.Input{{Name: "a", Choices: []string{"x", ""}}}
		})},
		{"duplicate choice", "twice", mk("broken-dup-choice", func(d *feature.Declaration) {
			d.Inputs = []feature.Input{{Name: "a", Choices: []string{"x", "x"}}}
		})},
		{"choices and validator", "both choices and a validator", mk("broken-both", func(d *feature.Declaration) {
			d.Inputs = []feature.Input{{Name: "a", Choices: []string{"x"}, Validator: noop}}
		})},
		{"no declaration", "no declaration", noDecl},
		{"no plan", "no plan", mk("broken-no-plan", func(d *feature.Declaration) { d.Plan = nil })},
	}
}

// Register adds features under an add command.
//
// In: add, the `add` command; cmds, the feature commands.
// Out: none.
// Side effects: adds cmds as subcommands of add.
func Register(add *cobra.Command, cmds ...*cobra.Command) {
	add.AddCommand(cmds...)
}

// NewTree builds `aicli add` with the given features registered.
//
// In: cmds, the feature commands. Out: the add command, already attached to a
// new root command named "aicli". Side effects: none outside the new tree.
func NewTree(cmds ...*cobra.Command) *cobra.Command {
	root := &cobra.Command{Use: "aicli"}
	add := &cobra.Command{Use: "add", Short: "Add an AI/LLM feature to this project."}
	root.AddCommand(add)
	Register(add, cmds...)
	return add
}
