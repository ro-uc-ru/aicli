package tui

// FeatureInput is one input of a Feature, as the add screens show it. It is
// plain data, with no Cobra type (ARC-7).
type FeatureInput struct {
	// Name is the input name, shown as the field label.
	Name string
	// Description is the one-line description shown under the field.
	Description string
	// Choices is the ordered list of allowed values; empty for free text.
	Choices []string
	// Validator checks a trimmed, non-empty value and returns an error
	// message, or "" if it is valid. It is nil if the input has none.
	Validator func(string) string
}

// Feature is one `aicli add` feature, as the add screens and the static list
// see it. It is the only interface between internal/add and internal/tui.
type Feature struct {
	// Name is the command name of the feature.
	Name string
	// Short is the one-line description; empty when the feature has none.
	Short string
	// Listed reports whether the feature is available and not hidden, so it
	// appears in the selector and in the static list (EC-002).
	Listed bool
	// Inputs are the inputs of the feature, in declaration order.
	Inputs []FeatureInput
	// Plan computes the project-relative paths the feature would create from
	// the input values (one per input, trimmed). It returns the paths in plan
	// order, or the plan error. It writes nothing.
	Plan func(values []string) ([]string, error)
	// Check runs the read-only check of the feature files for the given
	// paths (FR-022, FR-023). It returns nil if they can be created, or the
	// error to show. It writes nothing.
	Check func(paths []string) error
	// Command builds the equivalent command line for the input values (one
	// per input, trimmed), as the user could type it (FR-018). When nil, the
	// confirmation shows `aicli add <name>` alone.
	Command func(values []string) string
}
