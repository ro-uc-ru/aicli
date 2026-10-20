package add

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"

	"github.com/ro-uc-ru/aicli/internal/feature"
	"github.com/ro-uc-ru/aicli/internal/tui"
)

// cancelMessage is what Show writes to stdout when the user cancels on the
// confirmation (FR-018).
const cancelMessage = "Nothing added.\n"

// runScreens runs the interactive add screens. It is a variable so that tests
// can replace the terminal program with a stub.
var runScreens = tui.RunAdd

// runFeature runs a chosen feature. It is a variable so that tests can
// replace the feature runner with a stub.
var runFeature = feature.Run

// Show is what `aicli add` does when run by itself or with an unknown name.
// The project root check has already run (FR-001).
//
// In: ctx, the run context, whose cancellation cause is returned and whose
// cancellation prints nothing (EC-020); addCmd, the `add` command, whose
// subcommands are the features; args, the arguments after `add`; in and out,
// the input and output streams; getenv, the environment lookup (typically
// os.Getenv).
// Out: nil on success. The context's cause if ctx is already cancelled. For a
// first argument that is not the name of any feature (listed or not), the
// FR-007 error:
// `unknown feature "<arg>"`, then, only when addCmd.SuggestionsFor finds
// listed features, `Did you mean "<a>", "<b>"?`, then
// "Run `aicli add` to see the available features." (continuation lines have
// no indentation; the error printer adds it, FR-026). The write error,
// wrapped with context. In interactive mode (tui.IsInteractive) with at least
// one listed feature: the error of the screens (tui.RunAdd), or, when the user
// adds, the error of feature.Run, with nothing written to out by Show.
// Side effects: writes the EC-003 text (no listed feature, both modes) or the
// static list (static mode) to out in a single write. In interactive mode it
// takes over the terminal on in and out while the screens run; after they
// return, `quit` writes nothing, `cancel` writes "Nothing added." plain to out
// and `add` runs the selected feature with ctx and out, which creates its
// files and writes its report (FR-010, FR-018, FR-019). A value starting with
// `-` is passed to the feature as a value.
func Show(ctx context.Context, addCmd *cobra.Command, args []string, in io.Reader, out io.Writer, getenv func(string) string) error {
	features := Features(addCmd)
	if len(args) > 0 && !isFeature(features, args[0]) {
		return unknownFeature(addCmd, args[0])
	}
	if err := context.Cause(ctx); err != nil {
		return err
	}
	anyListed := false
	for _, f := range features {
		anyListed = anyListed || f.Listed
	}
	if anyListed && tui.IsInteractive(in, out, getenv) {
		return runInteractive(ctx, addCmd, features, in, out, getenv)
	}
	if _, err := io.WriteString(out, tui.RenderAddStatic(features)); err != nil {
		return fmt.Errorf("write add list: %w", err)
	}
	return nil
}

// runInteractive runs the add screens and acts on how they ended.
//
// In: ctx, the run context; addCmd, the `add` command; features, the features
// of addCmd; in and out, the terminal streams; getenv, the environment lookup
// used to decide on styling.
// Out: nil for `quit` and `cancel`; for `add`, the error of the feature run;
// the screens' error otherwise; an error if the selected feature is not a
// subcommand of addCmd.
// Side effects: as described for Show.
func runInteractive(ctx context.Context, addCmd *cobra.Command, features []tui.Feature, in io.Reader, out io.Writer, getenv func(string) string) error {
	res, err := runScreens(ctx, features, in, out, tui.ShouldStyle(out, getenv))
	if err != nil {
		return err
	}
	switch res.Action {
	case tui.ActionCancel:
		if _, err := io.WriteString(out, cancelMessage); err != nil {
			return fmt.Errorf("write cancel message: %w", err)
		}
	case tui.ActionAdd:
		if res.Index < 0 || res.Index >= len(features) {
			return fmt.Errorf("selected feature %d is out of range", res.Index)
		}
		name := features[res.Index].Name
		for _, c := range addCmd.Commands() {
			if c.Name() == name {
				return runFeature(ctx, c, res.Values, out, getenv)
			}
		}
		return fmt.Errorf("feature %q not found", name)
	}
	return nil
}

// isFeature reports whether name is the name of any feature, listed or not.
//
// In: features, the features; name, the name to look for. Out: true if found.
// No side effects.
func isFeature(features []tui.Feature, name string) bool {
	for _, f := range features {
		if f.Name == name {
			return true
		}
	}
	return false
}

// cobraDefaultDistance is the edit distance Cobra uses for its own suggestions
// when SuggestionsMinimumDistance is not set.
const cobraDefaultDistance = 2

// unknownFeature builds the FR-007 error.
//
// In: addCmd, the `add` command; arg, the unknown name. Out: the error. The
// suggestions come from addCmd.SuggestionsFor, which Cobra restricts to
// available, non-hidden commands, i.e. the listed features, in list order.
// Side effects: sets addCmd.SuggestionsMinimumDistance to Cobra's default (2)
// when it is not set, as Cobra does itself before suggesting.
func unknownFeature(addCmd *cobra.Command, arg string) error {
	lines := []string{fmt.Sprintf("unknown feature %q", arg)}
	if addCmd.SuggestionsMinimumDistance <= 0 {
		addCmd.SuggestionsMinimumDistance = cobraDefaultDistance
	}
	if s := addCmd.SuggestionsFor(arg); len(s) > 0 {
		quoted := make([]string, len(s))
		for i, n := range s {
			quoted[i] = fmt.Sprintf("%q", n)
		}
		lines = append(lines, "Did you mean "+strings.Join(quoted, ", ")+"?")
	}
	lines = append(lines, "Run `aicli add` to see the available features.")
	return errors.New(strings.Join(lines, "\n"))
}
