package project

import "slices"

// exemptCommands is the single list of top-level commands that do not
// perform the project root check (FR-007). A spec that adds a command which
// needs an exemption declares it here.
var exemptCommands = []string{"new", "help", "version", "completion", "__complete"}

// IsExempt reports whether a command is exempt from the project root check.
//
// In: name, the name of the top-level command being run (not its flags and
// not its subcommands, which share the exemption of their top-level command).
// Out: true if name is in the exempt list, false for any other name.
// No side effects.
func IsExempt(name string) bool {
	return slices.Contains(exemptCommands, name)
}
