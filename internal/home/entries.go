// Package home builds the content of the home screen from the Cobra command
// tree and chooses how to show it.
package home

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"

	"github.com/ro-uc-ru/aicli/internal/tui"
)

// excludedNames lists the commands that are never shown on the home screen.
// An excluded command also excludes its whole subtree.
var excludedNames = map[string]bool{
	"help":                          true,
	"completion":                    true,
	cobra.ShellCompRequestCmd:       true,
	cobra.ShellCompNoDescRequestCmd: true,
}

// Entries builds the home screen entries from a command tree.
//
// In: root, the root command of the tree.
// Out: one tui.Entry per listed command, in depth-first pre-order with
// siblings in the order Cobra gives them; empty (nil) if no command is
// listed. A command is listed if it is available (not hidden, not
// deprecated), is not help, completion or __complete, and none of its
// ancestors is excluded. Each entry's Path is the command path without the
// root's name.
// Side effects: none at call time; the Help function of each entry modifies
// and then restores the writers of its command when called.
func Entries(root *cobra.Command) []tui.Entry {
	var entries []tui.Entry
	walk(root, root, &entries)
	return entries
}

// walk appends the listed descendants of parent to entries, depth first.
//
// In: root, the tree root (for path computation); parent, the command whose
// children are visited; entries, the destination slice.
// Out: none; entries grows with the listed descendants.
// Side effects: modifies *entries.
func walk(root, parent *cobra.Command, entries *[]tui.Entry) {
	for _, c := range parent.Commands() {
		if excludedNames[c.Name()] || !c.IsAvailableCommand() {
			continue
		}
		path := strings.TrimPrefix(c.CommandPath(), root.Name()+" ")
		*entries = append(*entries, tui.Entry{
			Path:  path,
			Short: c.Short,
			Help:  helpProducer(c, path),
		})
		walk(root, c, entries)
	}
}

// helpProducer returns the function that produces the help text of cmd.
//
// In: cmd, the command whose help is produced; path, its path without the
// root name, used in error messages.
// Out: a function returning the help text, the same as `<path> --help`, or
// an error "cannot show help for <path>: <first line written to err>" if
// the help function reported a failure.
// Side effects: the returned function initialises cmd's default help flag,
// points cmd's out and err writers at buffers while it runs, and restores
// both writers to unset afterwards.
func helpProducer(cmd *cobra.Command, path string) func() (string, error) {
	return func() (string, error) {
		var out, errBuf bytes.Buffer
		cmd.InitDefaultHelpFlag()
		cmd.SetOut(&out)
		cmd.SetErr(&errBuf)
		defer func() {
			cmd.SetOut(nil)
			cmd.SetErr(nil)
		}()
		cmd.HelpFunc()(cmd, nil)
		if msg := strings.TrimSpace(errBuf.String()); msg != "" {
			first, _, _ := strings.Cut(msg, "\n")
			return "", fmt.Errorf("cannot show help for %s: %s", path, first)
		}
		return out.String(), nil
	}
}

// Show presents the home screen for a command tree.
//
// In: ctx, the context whose cancellation aborts static output; root, the
// root command of the tree; in, the input stream; out, the output stream;
// getenv, the environment lookup (typically os.Getenv).
// Out: nil on success. In interactive mode (tui.IsInteractive), the error of
// tui.RunHome. Otherwise the context's cause if ctx is already cancelled, or
// the write error wrapped with context.
// Side effects: interactive mode takes over the terminal until the user
// quits. Static mode writes the plain banner and list to out in a single
// write and nothing else; no file, process or network is used.
func Show(ctx context.Context, root *cobra.Command, in io.Reader, out io.Writer, getenv func(string) string) error {
	entries := Entries(root)
	if tui.IsInteractive(in, out, getenv) {
		return tui.RunHome(ctx, entries, in, out, tui.ShouldStyle(out, getenv))
	}
	text := tui.RenderHomeStatic(entries)
	if err := context.Cause(ctx); err != nil {
		return err
	}
	if _, err := io.WriteString(out, text); err != nil {
		return fmt.Errorf("write home screen: %w", err)
	}
	return nil
}
