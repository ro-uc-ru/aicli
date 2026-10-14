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
	"github.com/ro-uc-ru/aicli/internal/home"
	"github.com/ro-uc-ru/aicli/internal/project"
	"github.com/ro-uc-ru/aicli/internal/scaffold"
	"github.com/ro-uc-ru/aicli/internal/tui"
)

// init turns on Cobra's traverse run hooks once, at package load, so that a
// command's own PersistentPreRunE runs after the root's project check instead
// of replacing it (FR-002).
//
// In: none. Out: none.
// Side effects: sets the package-global cobra.EnableTraverseRunHooks.
func init() {
	cobra.EnableTraverseRunHooks = true
}

// NewRoot builds the root command with every subcommand registered.
//
// In: baseDir, the directory in which generated projects are created.
// Out: the root command, with SilenceUsage and SilenceErrors set so that
// Execute alone reports errors.
// Side effects: none.
func NewRoot(baseDir string) *cobra.Command {
	return newRoot(baseDir, os.Getwd)
}

// newRoot builds the root command with an injectable working directory.
//
// In: baseDir, as in NewRoot; getwd, resolves the working directory for the
// project root check (os.Getwd in production, a stub in tests).
// Out: the root command, whose PersistentPreRunE runs the project root check
// before every non-exempt command (FR-002, FR-007), with Cobra's traverse
// run hooks enabled so that a command's own PersistentPreRunE does not
// replace it. The root is runnable: with no arguments it shows the home
// screen through home.Show (FR-001).
// Side effects: none; Cobra's traverse run hooks are enabled once by init.
func newRoot(baseDir string, getwd func() (string, error)) *cobra.Command {
	cmd := &cobra.Command{
		Use:           "aicli",
		Short:         "Scaffold FastAPI backends for AI/LLM apps",
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRunE: func(c *cobra.Command, _ []string) error {
			return checkProject(c, getwd)
		},
		RunE: func(c *cobra.Command, _ []string) error {
			return home.Show(c.Context(), c, c.InOrStdin(), c.OutOrStdout(), os.Getenv)
		},
	}
	cmd.AddCommand(newcmd.NewCommand(baseDir))
	return cmd
}

// checkProject runs the project root check for the executed command unless
// its top-level ancestor is exempt.
//
// In: c, the executed command; getwd, the working-directory resolver.
// Out: nil if the command is exempt or the working directory is a project
// root; otherwise the error from project.Check, unchanged.
// Side effects: on success, stores the root in c's context (project.RootFrom).
func checkProject(c *cobra.Command, getwd func() (string, error)) error {
	top := c
	for top.HasParent() && top.Parent().HasParent() {
		top = top.Parent()
	}
	if project.IsExempt(top.Name()) {
		return nil
	}
	root, err := project.Check(getwd)
	if err != nil {
		return err
	}
	ctx := c.Context()
	if ctx == nil {
		ctx = context.Background()
	}
	c.SetContext(project.WithRoot(ctx, root))
	return nil
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
	return run(ctx, NewRoot("."), args, stdout, stderr)
}

// run executes a prepared root command and returns the process exit code.
//
// In: ctx, args, stdout and stderr, as in Execute; cmd, the root command to
// run (tests pass one built by newRoot with extra commands registered).
// Out: the exit code, as in Execute.
// Side effects: as in Execute.
func run(ctx context.Context, cmd *cobra.Command, args []string, stdout, stderr io.Writer) int {
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
