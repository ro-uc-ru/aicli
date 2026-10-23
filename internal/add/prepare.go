package add

import (
	"context"
	"os"

	"github.com/spf13/cobra"

	"github.com/ro-uc-ru/aicli/internal/tui"
)

// isInteractive decides whether the direct form may run. It is a variable so
// that tests can force either mode without a terminal.
var isInteractive = tui.IsInteractive

// directAnnotation marks a feature command whose RunE Prepare has wrapped.
const directAnnotation = "aicli.add.direct"

// Prepare hides the deprecated features so that running one prints no
// deprecation notice (EC-002), and wraps the RunE of each feature with inputs
// so that a bare call opens its form directly (FR-005).
//
// In: root, the root command of the tree; its direct subcommand named "add"
// is the one prepared. A nil root or a tree without "add" is a no-op.
// Out: none.
// Side effects: for every direct subcommand of add with Deprecated set, clears
// Deprecated and sets Hidden. The feature stays unavailable and unlisted and
// still runs when called directly. Cobra prints its notice before any hook
// runs, so this must happen before the tree is executed. For every feature
// declared with inputs, replaces RunE with directRunE around the original and
// marks the command, so calling Prepare again wraps nothing twice.
func Prepare(root *cobra.Command) {
	if root == nil {
		return
	}
	for _, c := range root.Commands() {
		if c.Name() != "add" {
			continue
		}
		for _, f := range c.Commands() {
			if f.Deprecated != "" {
				f.Deprecated = ""
				f.Hidden = true
			}
			wrapDirect(c, f)
		}
	}
}

// wrapDirect wraps the RunE of one feature command, once.
//
// In: addCmd, the `add` command; f, a direct subcommand of it. Out: none.
// Side effects: unless f is not a registered feature, has no inputs, has no
// RunE or is already marked, sets f.RunE to a function that, with no
// positional argument, no changed flag and interactive mode, runs the add
// screens on f alone in direct mode (handling the result as runInteractive
// does) and otherwise calls the original RunE; it also marks f.
func wrapDirect(addCmd, f *cobra.Command) {
	if f.RunE == nil || f.Annotations[directAnnotation] != "" {
		return
	}
	probe := Features(addCmd)
	hasInputs := false
	for _, ft := range probe {
		if ft.Name == f.Name() && len(ft.Inputs) > 0 {
			hasInputs = true
		}
	}
	if !hasInputs {
		return
	}
	if f.Annotations == nil {
		f.Annotations = map[string]string{}
	}
	f.Annotations[directAnnotation] = "1"
	orig := f.RunE
	f.RunE = func(c *cobra.Command, args []string) error {
		in, out := c.InOrStdin(), c.OutOrStdout()
		if len(args) > 0 || c.Flags().NFlag() > 0 || !isInteractive(in, out, os.Getenv) {
			return orig(c, args)
		}
		ctx := c.Context()
		if ctx == nil {
			ctx = context.Background()
		}
		addCmd.SetContext(ctx)
		for _, ft := range Features(addCmd) {
			if ft.Name != c.Name() {
				continue
			}
			only := []tui.Feature{ft}
			res, err := runScreens(ctx, only, in, out, tui.ShouldStyle(out, os.Getenv), tui.WithDirect(0))
			if err != nil {
				return err
			}
			return handleResult(ctx, addCmd, only, res, out, os.Getenv)
		}
		return orig(c, args)
	}
}
