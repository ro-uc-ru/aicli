package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

const (
	// choiceSelected and choiceOther are the markers before the selected
	// choice and before the others (FR-016).
	choiceSelected = "(•) "
	choiceOther    = "( ) "
	// choiceFirstRow is the row, inside the field, of the first choice line:
	// after the title and the description.
	choiceFirstRow = 2
)

// choiceField is a field whose value is one of a fixed list of choices
// (FR-016).
type choiceField struct {
	input    FeatureInput
	styles   addStyles
	focused  bool
	selected int
	width    int
}

// newChoiceField builds the choice field of one input.
//
// In: in, the input, whose Choices is not empty; styled, whether to add ANSI
// colour; width, the terminal width in cells.
// Out: an unfocused field with the first choice selected.
// Side effects: none.
func newChoiceField(in FeatureInput, styled bool, width int) choiceField {
	return choiceField{input: in, styles: newAddStyles(styled), width: width}
}

// resize sets the terminal width the field is drawn for.
//
// In: width, the terminal width in cells. Out: the resized field.
// Side effects: none.
func (f choiceField) resize(width int) choiceField {
	f.width = width
	return f
}

// Focus gives the field the focus.
//
// In: none. Out: the focused field. Side effects: none.
func (f choiceField) Focus() choiceField {
	f.focused = true
	return f
}

// Blur takes the focus away from the field.
//
// In: none. Out: the unfocused field. Side effects: none.
func (f choiceField) Blur() choiceField {
	f.focused = false
	return f
}

// Value returns the selected choice.
//
// In: none. Out: the selected choice text. Side effects: none.
func (f choiceField) Value() string { return f.input.Choices[f.selected] }

// SetValue selects a choice.
//
// In: v, a choice text. Out: the field with v selected; unchanged if v is
// not one of the choices. Side effects: none.
func (f choiceField) SetValue(v string) choiceField {
	for i, c := range f.input.Choices {
		if c == v {
			f.selected = i
		}
	}
	return f
}

// Height returns the number of rows the field needs when drawn in full.
//
// In: none. Out: the title, the description and, while focused, one row per
// choice, otherwise one row for the selected choice. Side effects: none.
func (f choiceField) Height() int {
	if f.focused {
		return choiceFirstRow + len(f.input.Choices)
	}
	return choiceFirstRow + 1
}

// window returns the choices shown.
//
// In: space, the rows available to the whole field, at least 1. Out: the
// index of the first shown choice and how many are shown: all of them when
// they fit below the title and the description, otherwise as many as fit
// (at least one), starting so that the selection is visible. Not focused,
// only the selected choice is shown. Side effects: none.
func (f choiceField) window(space int) (start, n int) {
	if !f.focused {
		return f.selected, 1
	}
	total := len(f.input.Choices)
	n = max(min(space-choiceFirstRow, total), 1)
	if f.selected >= n {
		start = f.selected - n + 1
	}
	return start, n
}

// Update handles one key press while the field has the focus (FR-016,
// EC-011).
//
// In: msg, the key. Out: the field with the selection moved to the previous
// choice for `↑` and `k`, or to the next for `↓` and `j`, staying put at the
// first or last. Any other key is ignored. Side effects: none.
func (f choiceField) Update(msg tea.KeyMsg) choiceField {
	switch msg.String() {
	case "up", "k":
		f.selected = max(f.selected-1, 0)
	case "down", "j":
		f.selected = min(f.selected+1, len(f.input.Choices)-1)
	}
	return f
}

// Click handles a left click on the field (FR-015).
//
// In: row, the row of the click relative to the top of the field; space, the
// rows available to the field, as for View. Out: the field with the clicked
// choice selected if row is on a choice line; unchanged for the title and
// the description, and for any row past the last shown choice. Side
// effects: none.
func (f choiceField) Click(row, space int) choiceField {
	start, n := f.window(space)
	if i := row - choiceFirstRow; i >= 0 && i < n {
		f.selected = start + i
	}
	return f
}

// View renders the field.
//
// In: space, the rows available to the whole field; below 1 it counts as 1.
// Out: its lines: the title (`> ` before it when focused, two spaces
// otherwise), the description, then the choices indented by two spaces with
// `(•) ` before the selected one and `( ) ` before the others. Not focused,
// only the selected choice; focused, all of them, scrolling inside the field
// when space is too small, with the selection visible. Lines are truncated
// with `…` to the width, never wrapped. The title and the selected choice
// are highlighted when styled.
// Side effects: none.
func (f choiceField) View(space int) []string {
	indent := strings.Repeat(" ", fieldIndent)
	title, style := indent+f.input.Name, f.styles.Normal
	if f.focused {
		title, style = "> "+f.input.Name, f.styles.FieldFocused
	}
	lines := []string{
		style.Render(truncate(title, f.width)),
		f.styles.Dim.Render(truncate(indent+f.input.Description, f.width)),
	}
	start, n := f.window(max(space, 1))
	for i := start; i < start+n; i++ {
		marker, st := choiceOther, f.styles.Normal
		if i == f.selected {
			marker, st = choiceSelected, f.styles.Choice
		}
		lines = append(lines, st.Render(truncate(indent+marker+f.input.Choices[i], f.width)))
	}
	return lines
}
