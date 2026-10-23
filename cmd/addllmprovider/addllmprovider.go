// Package addllmprovider implements the `aicli add llm-provider` command. It
// holds only the Cobra command; the declaration lives in internal/llmprovider.
package addllmprovider

import (
	"github.com/spf13/cobra"

	"github.com/ro-uc-ru/aicli/internal/feature"
	"github.com/ro-uc-ru/aicli/internal/llmprovider"
)

// NewCommand builds the `aicli add llm-provider` command.
//
// In: none.
// Out: the feature command built from llmprovider.Declaration().
// Side effects: registers the declaration in the feature registry.
func NewCommand() *cobra.Command {
	return feature.NewCommand(llmprovider.Declaration())
}
