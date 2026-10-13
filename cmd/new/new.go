// Package newcmd implements the `aicli new` command. It holds only the Cobra
// command, its argument checks and wiring; the logic lives in internal/.
package newcmd

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/ro-uc-ru/aicli/internal/scaffold"
	"github.com/ro-uc-ru/aicli/internal/tui"
)

// errTooManyArgs is the usage error for more than one positional argument
// (FR-024). Its second line is the continuation line shown after the reason.
var errTooManyArgs = errors.New("too many arguments\nUsage: aicli new <name>")

// NewCommand builds the `aicli new <name>` command.
//
// In: baseDir, the directory in which ./<name> is created (the root command
// passes "."). The command runs with the context set on it by Cobra
// (context.Background if none) and writes to cmd.OutOrStdout.
// Out: the command. Its RunE returns scaffold.ErrMissingName for zero
// arguments or "" (FR-012, EC-003), a usage error for more than one argument
// (FR-024), or any error from scaffold.Run; it never writes to stdout on
// failure (FR-023).
// Side effects: none at construction; running it creates the project on disk
// and, on success only, writes the FR-015 text to stdout.
func NewCommand(baseDir string) *cobra.Command {
	return &cobra.Command{
		Use:   "new <name>",
		Short: "Create a new FastAPI project",
		Args:  checkArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return run(cmd, baseDir, args[0])
		},
	}
}

// checkArgs validates the positional arguments of `aicli new`.
//
// In: args, the positional arguments; the command is unused.
// Out: scaffold.ErrMissingName if there are none or the first is "";
// errTooManyArgs if there is more than one; nil for exactly one.
// No side effects.
func checkArgs(_ *cobra.Command, args []string) error {
	switch {
	case len(args) > 1:
		return errTooManyArgs
	case len(args) == 0 || args[0] == "":
		return scaffold.ErrMissingName
	}
	return nil
}

// run scaffolds the project and prints the success text.
//
// In: cmd, the running command (context and stdout); baseDir, the parent
// directory; name, the project name.
// Out: nil on success; otherwise the error from scaffold.Run, or the write
// error of stdout, with context.
// Side effects: creates the project; on success writes RenderSuccess to
// stdout, styled only if stdout is a terminal and the environment allows it.
func run(cmd *cobra.Command, baseDir, name string) error {
	ctx := cmd.Context()
	if ctx == nil {
		ctx = context.Background()
	}
	if err := scaffold.Run(ctx, baseDir, name); err != nil {
		return err
	}
	out := cmd.OutOrStdout()
	styled := tui.ShouldStyle(out, os.Getenv)
	if _, err := fmt.Fprint(out, tui.RenderSuccess(name, styled)); err != nil {
		return fmt.Errorf("write success message: %w", err)
	}
	return nil
}
