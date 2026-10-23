// Package feature defines the contract that every `aicli add <feature>`
// follows: the declaration of its inputs and plan, the helper that builds its
// Cobra command, and the registry that maps a command back to its declaration.
package feature

import (
	"context"
	"os"
	"strings"
	"sync"

	"github.com/spf13/cobra"

	"github.com/ro-uc-ru/aicli/internal/gen"
)

// Input is one named positional argument that a feature declares (FR-003).
type Input struct {
	// Name is the input name shown as <name> in the usage line.
	Name string
	// Description is the one-line description shown by the form.
	Description string
	// Choices is the ordered list of allowed values; empty if the input is free text.
	Choices []string
	// Validator checks a trimmed, non-empty value and returns an error message,
	// or "" if the value is valid. It is nil if the input has none. An input
	// declares either Choices or Validator, never both.
	Validator func(string) string
	// Multi marks the input as multi-choice: it accepts one or more distinct
	// values from Choices. Only the last input of a feature may be Multi.
	Multi bool
	// ItemNoun is the plural noun for the chosen values (e.g. "providers");
	// empty if the input declares none. Only meaningful if Multi is true.
	ItemNoun string
	// ItemFiles maps a choice to its item file, a path that must be a create
	// entry of the plan for any set of values including that choice.
	ItemFiles map[string]string
}

// Declaration is everything a feature declares about itself (FR-003).
type Declaration struct {
	// Name is the command name, e.g. "feature-1" in `aicli add feature-1`.
	Name string
	// Short is the one-line description of the feature.
	Short string
	// Long is the long description shown by --help.
	Long string
	// Inputs are the positional arguments, in order.
	Inputs []Input
	// Plan computes the files the feature creates from the input values
	// (one per input, trimmed). It returns the files or an error whose
	// message is shown after "Error: ".
	Plan func(values []string) ([]gen.FeatureFile, error)
	// NextSteps returns the one-line instructions printed after the files are
	// created, from the input values (one per input, trimmed; the last holds
	// every value of a multi-choice input).
	NextSteps func(values []string) []string
}

var (
	// registryMu guards registry.
	registryMu sync.RWMutex
	// registry maps each feature command to its declaration.
	registry = map[*cobra.Command]*Declaration{}
)

// NewCommand builds the Cobra command of a feature and registers decl.
//
// In: decl, the feature declaration; it is copied, so later changes to the
// caller's value do not affect the registered one.
// Out: a command whose Use is "<name> <input-1> <input-2> ..." (each input
// name wrapped in angle brackets, followed by "..." for a multi-choice
// input), with Short and Long from decl, accepting
// any number of arguments (the count is checked by ValidateValues, FR-006),
// and a RunE that only calls Run with os.Getenv.
// Side effects: stores the declaration in the package registry, keyed by the
// returned command.
func NewCommand(decl Declaration) *cobra.Command {
	d := decl
	d.Inputs = append([]Input(nil), decl.Inputs...)
	parts := []string{d.Name}
	for _, in := range d.Inputs {
		name := "<" + in.Name + ">"
		if in.Multi {
			name += "..."
		}
		parts = append(parts, name)
	}
	cmd := &cobra.Command{
		Use:   strings.Join(parts, " "),
		Short: d.Short,
		Long:  d.Long,
		Args:  cobra.ArbitraryArgs,
		RunE: func(c *cobra.Command, args []string) error {
			ctx := c.Context()
			if ctx == nil {
				ctx = context.Background()
			}
			return Run(ctx, c, args, c.OutOrStdout(), os.Getenv)
		},
	}
	registryMu.Lock()
	registry[cmd] = &d
	registryMu.Unlock()
	return cmd
}

// Lookup reads back the declaration registered by NewCommand.
//
// In: cmd, any Cobra command (may be nil).
// Out: the declaration and true if cmd was built by NewCommand; nil and false
// otherwise. The declaration must be treated as read-only.
// No side effects.
func Lookup(cmd *cobra.Command) (*Declaration, bool) {
	registryMu.RLock()
	defer registryMu.RUnlock()
	d, ok := registry[cmd]
	return d, ok
}
