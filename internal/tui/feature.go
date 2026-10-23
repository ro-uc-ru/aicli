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
	// Multi reports whether the input takes one or more of its Choices,
	// shown as a checklist (FR-006). Only the last input may be Multi.
	Multi bool
	// ItemNoun is the plural noun of the things the choices add, used in
	// `All <items> are already added.` (EC-006); empty for none.
	ItemNoun string
	// Added tells, per choice and in order, whether it is already added
	// (FR-007); nil or shorter than Choices means the rest are not.
	Added []bool
	// Validator checks a trimmed, non-empty value and returns an error
	// message, or "" if it is valid. It is nil if the input has none.
	Validator func(string) string
}

// Preview is what a feature would do to the project, as the confirmation
// lists it (FR-011). It is plain data, with no gen type (ARC-7).
type Preview struct {
	// Create holds the project-relative paths that would be created, in
	// plan order.
	Create []string
	// Update holds the project-relative paths of existing files that would
	// be modified, in plan order.
	Update []string
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
	// Added reports, per choice of the multi-choice input, whether its item
	// file already exists (FR-007). It is called once, when the form opens,
	// and only stats files. When nil, no choice is already added.
	Added func() []bool
	// Check runs the read-only check of the feature files for the given
	// input values (one per input, trimmed; FR-022, FR-023). It returns what
	// the feature would create and update, or the error to show. It writes
	// nothing. When nil, the confirmation lists the paths of Plan as
	// created and nothing as updated.
	Check func(values []string) (Preview, error)
	// Command builds the equivalent command line for the input values (one
	// per input, trimmed), as the user could type it (FR-018). When nil, the
	// confirmation shows `aicli add <name>` alone.
	Command func(values []string) string
}
