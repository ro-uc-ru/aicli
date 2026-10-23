// Package featuretest builds test-only `aicli add` features: a complete valid
// one, one that breaks each rule of the feature contract (FR-003), and the
// edge cases of the listing (hidden, deprecated, no inputs, no short
// description, wide characters). It is imported only by tests; a test of this
// package checks that no non-test file imports it.
package featuretest

import (
	"errors"
	"strings"
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
	// Multi is the name of the valid multi-choice test feature.
	Multi = "feature-multi"
)

// Shape of the multi-choice test feature.
const (
	// MultiInput is the name of its multi-choice input.
	MultiInput = "item"
	// MultiNoun is its plural item noun.
	MultiNoun = "items"
	// MultiMarkerA and MultiMarkerB are the markers its inject entries use,
	// in MultiTargetA and MultiTargetB respectively.
	MultiMarkerA = "first"
	// MultiMarkerB is the second marker; see MultiMarkerA.
	MultiMarkerB = "second"
	// MultiTargetA is the file that holds MultiMarkerA.
	MultiTargetA = "targets/a.txt"
	// MultiTargetB is the file that holds MultiMarkerB.
	MultiTargetB = "targets/b.txt"
)

// MultiChoices are the choices of the multi-choice test feature, in order.
var MultiChoices = []string{"alpha", "beta", "gamma"}

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

	"shared.tmpl":     {Data: []byte("shared\n")},
	"shared-bad.tmpl": {Data: []byte("shared {{.Values}}\n")},
	"item.tmpl":       {Data: []byte("item {{.Value}}\n")},
	"inject.tmpl":     {Data: []byte("line {{.Value}}\n")},
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
		NextSteps: func([]string) []string { return NextSteps },
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

// MultiItemFile returns the item file of a choice of the multi-choice feature.
//
// In: choice, one of MultiChoices. Out: "items/<choice>.txt". No side effects.
func MultiItemFile(choice string) string { return "items/" + choice + ".txt" }

// multiItemFiles maps each choice of MultiChoices to its item file.
func multiItemFiles() map[string]string {
	m := map[string]string{}
	for _, c := range MultiChoices {
		m[c] = MultiItemFile(c)
	}
	return m
}

// multiPlan computes the entries of the multi-choice feature.
//
// In: v, the values (the chosen items, in choice order). Out: two shared
// entries ("shared/one.txt", "shared/two.txt"), then, per value, a create
// entry for its item file and one for "items/<value>.md", then, per value,
// one inject entry at MultiMarkerA of MultiTargetA and one at MultiMarkerB of
// MultiTargetB. No side effects.
func multiPlan(v []string) ([]gen.FeatureFile, error) {
	return multiEntries(v, "shared.tmpl", nil), nil
}

// multiEntries builds the entries of the multi-choice feature.
//
// In: v, the values; sharedTmpl, the template of both shared entries (it may
// render "{{.Values}}"); omitItem, the choices whose item file is left out of
// the entries (nil for none). Out: the entries as in multiPlan.
// No side effects.
func multiEntries(v []string, sharedTmpl string, omitItem map[string]bool) []gen.FeatureFile {
	all := map[string]string{"Values": strings.Join(v, ",")}
	files := []gen.FeatureFile{
		{Path: "shared/one.txt", Templates: templates, Template: sharedTmpl, Data: all, Action: gen.ActionShared},
		{Path: "shared/two.txt", Templates: templates, Template: sharedTmpl, Data: all, Action: gen.ActionShared},
	}
	for _, x := range v {
		d := map[string]string{"Value": x}
		if !omitItem[x] {
			files = append(files, gen.FeatureFile{Path: MultiItemFile(x), Templates: templates, Template: "item.tmpl", Data: d})
		}
		files = append(files, gen.FeatureFile{Path: "items/" + x + ".md", Templates: templates, Template: "item.tmpl", Data: d})
	}
	for _, x := range v {
		d := map[string]string{"Value": x}
		files = append(files,
			gen.FeatureFile{Path: MultiTargetA, Templates: templates, Template: "inject.tmpl", Data: d, Action: gen.ActionInject, Marker: MultiMarkerA},
			gen.FeatureFile{Path: MultiTargetB, Templates: templates, Template: "inject.tmpl", Data: d, Action: gen.ActionInject, Marker: MultiMarkerB})
	}
	return files
}

// multiNextSteps returns one next step per value.
//
// In: v, the values. Out: "Set <value> in .env" per value. No side effects.
func multiNextSteps(v []string) []string {
	var out []string
	for _, x := range v {
		out = append(out, "Set "+x+" in .env")
	}
	return out
}

// multiDecl returns a valid declaration of a multi-choice feature.
//
// In: name, the feature name. Out: the declaration: one multi-choice input
// over MultiChoices with its item noun and item files, the multiPlan plan and
// steps that depend on the values. No side effects.
func multiDecl(name string) feature.Declaration {
	return feature.Declaration{
		Name:  name,
		Short: "Test feature " + name,
		Inputs: []feature.Input{{
			Name: MultiInput, Description: "The items", Choices: MultiChoices,
			Multi: true, ItemNoun: MultiNoun, ItemFiles: multiItemFiles(),
		}},
		Plan:      multiPlan,
		NextSteps: multiNextSteps,
	}
}

// NewMulti builds the valid multi-choice test feature: item files, two shared
// entries, create entries per value and inject entries at two markers.
//
// In: none. Out: a new command named Multi. No side effects.
func NewMulti() *cobra.Command { return feature.NewCommand(multiDecl(Multi)) }

// BrokenMultiFeatures returns one new feature per multi-choice rule of the
// contract (FR-001), each named "broken-multi-<n>" and breaking only its rule:
// the multi-choice input is not the last; it has no choices; it has an empty
// choice; it has a repeated choice; it has a validator; an item file is not a
// create entry of the plan; a shared entry depends on the values.
//
// In: none. Out: the features. No side effects.
func BrokenMultiFeatures() []Broken {
	mk := func(name string, edit func(*feature.Declaration)) *cobra.Command {
		d := multiDecl(name)
		edit(&d)
		return feature.NewCommand(d)
	}
	multi := func(d *feature.Declaration) *feature.Input { return &d.Inputs[len(d.Inputs)-1] }
	return []Broken{
		{"multi not last", "must be the last input", mk("broken-multi-not-last", func(d *feature.Declaration) {
			d.Inputs = append(d.Inputs, feature.Input{Name: "extra"})
		})},
		{"multi no choices", "declares no choices", mk("broken-multi-no-choices", func(d *feature.Declaration) {
			multi(d).Choices = nil
		})},
		{"multi empty choice", "declares an empty choice", mk("broken-multi-empty-choice", func(d *feature.Declaration) {
			multi(d).Choices = []string{"alpha", ""}
		})},
		{"multi repeated choice", "twice", mk("broken-multi-repeated-choice", func(d *feature.Declaration) {
			multi(d).Choices = []string{"alpha", "alpha"}
		})},
		{"multi validator", "both choices and a validator", mk("broken-multi-validator", func(d *feature.Declaration) {
			multi(d).Validator = func(string) string { return "" }
		})},
		{"multi item file not created", "item file", mk("broken-multi-item-file", func(d *feature.Declaration) {
			d.Plan = func(v []string) ([]gen.FeatureFile, error) {
				return multiEntries(v, "shared.tmpl", map[string]bool{"beta": true}), nil
			}
		})},
		{"multi shared depends on values", "depends on the values", mk("broken-multi-shared", func(d *feature.Declaration) {
			d.Plan = func(v []string) ([]gen.FeatureFile, error) {
				return multiEntries(v, "shared-bad.tmpl", nil), nil
			}
		})},
	}
}
