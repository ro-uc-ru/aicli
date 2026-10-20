package add

import "github.com/spf13/cobra"

// Prepare hides the deprecated features so that running one prints no
// deprecation notice (EC-002).
//
// In: root, the root command of the tree; its direct subcommand named "add"
// is the one prepared. A nil root or a tree without "add" is a no-op.
// Out: none.
// Side effects: for every direct subcommand of add with Deprecated set, clears
// Deprecated and sets Hidden. The feature stays unavailable and unlisted and
// still runs when called directly. Cobra prints its notice before any hook
// runs, so this must happen before the tree is executed. Calling it again has
// no further effect.
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
		}
	}
}
