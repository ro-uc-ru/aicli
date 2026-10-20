// Package add holds what `aicli add` does: it discovers the features
// registered under the add command, chooses static or interactive mode and
// runs the chosen feature. It knows no feature by name (FR-002).
package add

import (
	"errors"

	"github.com/spf13/cobra"

	"github.com/ro-uc-ru/aicli/internal/feature"
	"github.com/ro-uc-ru/aicli/internal/gen"
	"github.com/ro-uc-ru/aicli/internal/project"
	"github.com/ro-uc-ru/aicli/internal/tui"
)

// errNoRoot is returned by a feature's Check when the add command's context
// carries no project root.
var errNoRoot = errors.New("project root not found")

// Features builds the features of the add command from the command tree.
//
// In: add, the `add` command whose direct subcommands are the features. The
// project root for each feature's Check is read from add.Context() when Check
// runs (see project.WithRoot), not when Features is called.
// Out: one tui.Feature per direct subcommand that was built by
// feature.NewCommand, in Cobra order (the order of add.Commands()). Listed is
// true when the command is available and not hidden. Plan returns only the
// paths of the feature's plan; Check calls gen.CheckFeatureFiles with the
// project root, or returns an error if there is none. A subcommand without a
// declaration is skipped (it is the contract check's concern, FR-003).
// Side effects: none. The tree is read on every call, so a feature added to it
// later appears on the next call (FR-002).
func Features(add *cobra.Command) []tui.Feature {
	var out []tui.Feature
	for _, c := range add.Commands() {
		decl, ok := feature.Lookup(c)
		if !ok {
			continue
		}
		inputs := make([]tui.FeatureInput, len(decl.Inputs))
		for i, in := range decl.Inputs {
			inputs[i] = tui.FeatureInput{
				Name:        in.Name,
				Description: in.Description,
				Choices:     append([]string(nil), in.Choices...),
				Validator:   in.Validator,
			}
		}
		name := decl.Name
		plan := decl.Plan
		out = append(out, tui.Feature{
			Name:   name,
			Short:  decl.Short,
			Listed: c.IsAvailableCommand(),
			Inputs: inputs,
			Command: func(values []string) string {
				return feature.EquivalentCommand(name, values)
			},
			Plan: func(values []string) ([]string, error) {
				files, err := plan(values)
				if err != nil {
					return nil, err
				}
				paths := make([]string, len(files))
				for i, f := range files {
					paths[i] = f.Path
				}
				return paths, nil
			},
			Check: func(paths []string) error {
				ctx := add.Context()
				if ctx == nil {
					return errNoRoot
				}
				root, ok := project.RootFrom(ctx)
				if !ok {
					return errNoRoot
				}
				return gen.CheckFeatureFiles(root, name, paths)
			},
		})
	}
	return out
}
