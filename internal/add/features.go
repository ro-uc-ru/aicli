// Package add holds what `aicli add` does: it discovers the features
// registered under the add command, chooses static or interactive mode and
// runs the chosen feature. It knows no feature by name (FR-002).
package add

import (
	"errors"
	"os"
	"path/filepath"

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
// project root for each feature's Added and Check is read from add.Context()
// when they run (see project.WithRoot), not when Features is called.
// Out: one tui.Feature per direct subcommand that was built by
// feature.NewCommand, in Cobra order (the order of add.Commands()). Listed is
// true when the command is available and not hidden. Each input carries Multi
// and ItemNoun. Plan returns only the paths of the feature's plan. Added
// reports, per choice of the last input if it is multi-choice, whether its
// item file exists (a file, a directory or a broken link all count); it is nil
// for a feature with no item files. Check runs gen.CheckPlan on the plan of
// the values, with the item files, against the project root, and returns its
// Preview, or an error if there is no root. A subcommand without a
// declaration is skipped (it is the contract check's concern, FR-003).
// Side effects: none. The tree is read on every call, so a feature added to it
// later appears on the next call (FR-002). Added only stats files.
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
				Multi:       in.Multi,
				ItemNoun:    in.ItemNoun,
				Validator:   in.Validator,
			}
		}
		out = append(out, tui.Feature{
			Name:   decl.Name,
			Short:  decl.Short,
			Listed: c.IsAvailableCommand(),
			Inputs: inputs,
			Command: func(values []string) string {
				return feature.EquivalentCommand(decl.Name, values)
			},
			Plan: func(values []string) ([]string, error) {
				files, err := decl.Plan(values)
				if err != nil {
					return nil, err
				}
				paths := make([]string, len(files))
				for i, f := range files {
					paths[i] = f.Path
				}
				return paths, nil
			},
			Added: addedFunc(add, decl),
			Check: func(values []string) (tui.Preview, error) {
				root, err := rootOf(add)
				if err != nil {
					return tui.Preview{}, err
				}
				files, err := decl.Plan(values)
				if err != nil {
					return tui.Preview{}, err
				}
				pv, err := gen.CheckPlan(root, decl.Name, feature.BuildPlan(decl, files, values))
				if err != nil {
					return tui.Preview{}, err
				}
				return tui.Preview{Create: pv.Create, Update: pv.Update}, nil
			},
		})
	}
	return out
}

// rootOf reads the project root from the context of the add command.
//
// In: add, the add command. Out: the root, or errNoRoot if the command has no
// context or the context carries no root. No side effects.
func rootOf(add *cobra.Command) (string, error) {
	ctx := add.Context()
	if ctx == nil {
		return "", errNoRoot
	}
	root, ok := project.RootFrom(ctx)
	if !ok {
		return "", errNoRoot
	}
	return root, nil
}

// addedFunc builds the Added function of a feature.
//
// In: add, the add command (for the root); decl, the declaration.
// Out: nil if the last input is not multi-choice or has no item files.
// Otherwise a function that returns, per choice of that input, true when its
// item file exists (Lstat, so a directory or a broken link counts). Without a
// root, or if a file cannot be checked, the choice counts as not added; the
// confirmation check reports the real error.
// Side effects: the returned function only stats files.
func addedFunc(add *cobra.Command, decl *feature.Declaration) func() []bool {
	n := len(decl.Inputs)
	if n == 0 || !decl.Inputs[n-1].Multi || len(decl.Inputs[n-1].ItemFiles) == 0 {
		return nil
	}
	in := decl.Inputs[n-1]
	return func() []bool {
		added := make([]bool, len(in.Choices))
		root, err := rootOf(add)
		if err != nil {
			return added
		}
		for i, ch := range in.Choices {
			if p := in.ItemFiles[ch]; p != "" {
				_, statErr := os.Lstat(filepath.Join(root, filepath.FromSlash(p)))
				added[i] = statErr == nil
			}
		}
		return added
	}
}
