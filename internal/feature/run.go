package feature

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/ro-uc-ru/aicli/internal/gen"
	"github.com/ro-uc-ru/aicli/internal/project"
	"github.com/ro-uc-ru/aicli/internal/tui"
)

// errNoDeclaration is returned by Run for a command not built by NewCommand.
var errNoDeclaration = errors.New("command is not a registered feature")

// errNoRoot is returned by Run when the context carries no project root.
var errNoRoot = errors.New("project root not found")

// featureFS is the filesystem through which Run creates the feature files.
var featureFS gen.FeatureFS = gen.OSFeatureFS{}

// SetFS replaces the filesystem through which Run creates files. It is a test
// seam: only test programs (such as the signal test binary) call it, to make
// creation block or fail at a chosen point.
//
// In: fsys, the filesystem to use from now on.
// Out: none. Side effects: changes the package filesystem for every later
// Run; it is not safe to call while a Run is in progress.
func SetFS(fsys gen.FeatureFS) { featureFS = fsys }

// Run runs a feature: it validates the values, computes the plan, applies it
// and prints the report (FR-006, FR-019, FR-025, FR-004, FR-011, FR-014).
//
// In: ctx, the run context, which must carry the project root (see
// project.WithRoot) and is passed to the file creation for cancellation;
// cmd, the feature command built by NewCommand; values, the raw positional
// arguments; out, where the report is written; getenv, the environment
// lookup used to decide whether the report is styled.
// Out: nil on success. Otherwise an error whose message is the plain text
// after "Error: ": a ValidateValues error; the message of a failing Plan; or
// an error of gen.ApplyPlan (which includes the values already added). On
// any error nothing is written to out and nothing is left created or
// modified.
// Side effects: creates and modifies the feature files under the project
// root (see gen.ApplyPlan); writes the report, with one line per path and the
// next steps of the values, to out in one write on success.
func Run(ctx context.Context, cmd *cobra.Command, values []string, out io.Writer, getenv func(string) string) error {
	decl, ok := Lookup(cmd)
	if !ok {
		return fmt.Errorf("%s: %w", cmd.Name(), errNoDeclaration)
	}
	trimmed, err := ValidateValues(cmd, decl, values)
	if err != nil {
		return err
	}
	files, err := decl.Plan(trimmed)
	if err != nil {
		return err
	}
	root, ok := project.RootFrom(ctx)
	if !ok {
		return errNoRoot
	}
	lines, err := gen.ApplyPlan(ctx, featureFS, root, decl.Name, BuildPlan(decl, files, trimmed))
	if err != nil {
		return err
	}
	var steps []string
	if decl.NextSteps != nil {
		steps = decl.NextSteps(trimmed)
	}
	_, err = io.WriteString(out, tui.RenderAdded(decl.Name, lines, steps, tui.ShouldStyle(out, getenv)))
	return err
}

// BuildPlan wraps the entries of a feature in a gen.Plan with the item noun
// and the item files of the values of its multi-choice input.
//
// In: decl, the declaration; entries, the output of decl.Plan; values, the
// validated values (the multi-choice ones last, in choice order).
// Out: the plan. Items follow the order of values and only include values
// that have an item file; ItemNoun is empty if no input is multi-choice.
// No side effects.
func BuildPlan(decl *Declaration, entries []gen.FeatureFile, values []string) gen.Plan {
	plan := gen.Plan{Entries: entries}
	n := len(decl.Inputs)
	if n == 0 || !decl.Inputs[n-1].Multi {
		return plan
	}
	in := decl.Inputs[n-1]
	plan.ItemNoun = in.ItemNoun
	for _, v := range values[n-1:] {
		if p := in.ItemFiles[v]; p != "" {
			plan.Items = append(plan.Items, gen.Item{Value: v, Path: p})
		}
	}
	return plan
}
