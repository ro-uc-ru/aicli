// Package addcmd implements the `aicli add` command. It holds only the Cobra
// command and its wiring; the logic lives in internal/add.
package addcmd

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/ro-uc-ru/aicli/internal/add"
)

// NewCommand builds the `aicli add` grouping command, whose subcommands are
// the features.
//
// In: none. The command runs with the context Cobra sets on it, which carries
// the project root after the root command's check.
// Out: the command, accepting any number of arguments. Its RunE returns the
// error of add.Show: the unknown-feature error, the screens' or the chosen
// feature's error, or a write error.
// Side effects: none at construction; running it may write to stdout and, in
// interactive mode, create the files of the chosen feature.
func NewCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "add",
		Short: "Add an AI/LLM feature to this project.",
		Args:  cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return add.Show(cmd.Context(), cmd, args, cmd.InOrStdin(), cmd.OutOrStdout(), os.Getenv)
		},
	}
}
