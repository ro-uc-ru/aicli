// Package root builds the aicli Cobra root command, registers every command
// package under cmd/ and turns the result into a process exit code.
package root

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"

	newcmd "github.com/ro-uc-ru/aicli/cmd/new"
	"github.com/ro-uc-ru/aicli/internal/scaffold"
	"github.com/ro-uc-ru/aicli/internal/tui"
)

// NewRoot builds the root command with every subcommand registered.
//
// In: baseDir, the directory in which generated projects are created.
// Out: the root command, with SilenceUsage and SilenceErrors set so that
// Execute alone reports errors.
// Side effects: none.
func NewRoot(baseDir string) *cobra.Command {
	cmd := &cobra.Command{
		Use:           "aicli",
		Short:         "Scaffold FastAPI backends for AI/LLM apps",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	cmd.AddCommand(newcmd.NewCommand(baseDir))
	return cmd
}

// Execute runs aicli and returns the process exit code.
//
// In: ctx, the run context; the caller cancels it with a
// *scaffold.SignalError as cause on SIGINT or SIGTERM; args, the command-line
// arguments without the program name; stdout and stderr, the output writers.
// Out: 0 on success; the signal's 128+n code, printing nothing, if the error
// is a *scaffold.SignalError (EC-009); 1 for any other error, after writing
// it to stderr with tui.RenderError, styled only if stderr allows it (FR-023).
// Side effects: runs the command, which may create a project on disk, and
// writes to stdout or stderr.
func Execute(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	cmd := NewRoot(".")
	cmd.SetArgs(args)
	cmd.SetOut(stdout)
	cmd.SetErr(stderr)
	err := cmd.ExecuteContext(ctx)
	if err == nil {
		return 0
	}
	var se *scaffold.SignalError
	if errors.As(err, &se) {
		return se.ExitCode()
	}
	styled := tui.ShouldStyle(stderr, os.Getenv)
	_, _ = fmt.Fprint(stderr, tui.RenderError(err.Error(), styled))
	return 1
}
