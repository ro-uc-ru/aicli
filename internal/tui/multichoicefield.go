package tui

import (
	"slices"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

const (
	// checkSelected, checkOther and checkAdded are the markers before a
	// selected choice, an unselected one and an already added one (FR-006,
	// FR-007).
	checkSelected = "[x] "
	checkOther    = "[ ] "
	checkAdded    = "[✓] "
	// checkAddedSuffix follows the text of an already added choice (FR-007).
	checkAddedSuffix = " (already added)"
	// checkCursor is drawn before the line under the cursor; checkNoCursor
	// takes its place on every other line (FR-006).
	checkCursor   = "> "
	checkNoCursor = "  "
)

// multiChoiceField is a field whose value is one or more of a fixed list of
// choices, shown as a checklist (FR-006, FR-007).
type multiChoiceField struct {
	input    FeatureInput
	styles   addStyles
	focused  bool
	added    []bool
	selected []bool
	cursor   int
	width    int
}

// newMultiChoiceField builds the checklist of one input.
//
// In: in, the input, whose Choices is not empty; added, one flag per choice
// telling whether it is already added (nil or shorter than Choices: the
// missing flags are false); styled, whether to add ANSI colour; width, the
// terminal width in cells.
// Out: an unfocused field with nothing selected and the cursor on the first
// choice that is not already added, or on none (-1) if every choice is.
// Side effects: none.
func newMultiChoiceField(in FeatureInput, added []bool, styled bool, width int) multiChoiceField {
	f := multiChoiceField{
		input:    in,
		styles:   newAddStyles(styled),
		added:    make([]bool, len(in.Choices)),
		selected: make([]bool, len(in.Choices)),
		cursor:   -1,
		width:    width,
	}
	copy(f.added, added)
	f.cursor = f.step(-1, 1)
	return f
}

// step finds the nearest choice that is not already added.
//
// In: from, the index to start from (not included); dir, 1 or -1.
// Out: the index of the first choice that is not already added in that
// direction, or -1 if there is none. Side effects: none.
func (f multiChoiceField) step(from, dir int) int {
	for i := from + dir; i >= 0 && i < len(f.added); i += dir {
		if !f.added[i] {
			return i
		}
	}
	return -1
}

// Focus gives the field the focus.
//
// In: none. Out: the focused field. Side effects: none.
func (f multiChoiceField) Focus() multiChoiceField {
	f.focused = true
	return f
}

// Blur takes the focus away from the field.
//
// In: none. Out: the unfocused field. Side effects: none.
func (f multiChoiceField) Blur() multiChoiceField {
	f.focused = false
	return f
}

// Value returns the selected choices.
//
// In: none. Out: the selected choices in declaration order, whatever order
// they were selected in; empty if none. Side effects: none.
func (f multiChoiceField) Value() []string {
	var v []string
	for i, c := range f.input.Choices {
		if f.selected[i] {
			v = append(v, c)
		}
	}
	return v
}

// SetValue selects choices.
//
// In: v, choice texts. Out: the field with exactly the choices in v
// selected; texts that are not choices, and choices already added, are
// ignored. Side effects: none.
func (f multiChoiceField) SetValue(v []string) multiChoiceField {
	f.selected = make([]bool, len(f.input.Choices))
	for i, c := range f.input.Choices {
		f.selected[i] = !f.added[i] && slices.Contains(v, c)
	}
	return f
}

// AllAdded reports whether every choice is already added (EC-006).
//
// In: none. Out: true if there is no choice that can be selected.
// Side effects: none.
func (f multiChoiceField) AllAdded() bool { return !slices.Contains(f.added, false) }

// Height returns the number of rows the field needs when drawn in full.
//
// In: none. Out: the title, the description and, while focused, one row per
// choice, otherwise one row for the selection. Side effects: none.
func (f multiChoiceField) Height() int {
	if f.focused {
		return choiceFirstRow + len(f.input.Choices)
	}
	return choiceFirstRow + 1
}

// window returns the choices shown while focused.
//
// In: space, the rows available to the whole field, at least 1.
// Out: the index of the first shown choice and how many are shown: all of
// them when they fit below the title and the description, otherwise as many
// as fit (at least one), starting so that the cursor is visible. Side
// effects: none.
func (f multiChoiceField) window(space int) (start, n int) {
	total := len(f.input.Choices)
	n = max(min(space-choiceFirstRow, total), 1)
	if f.cursor >= n {
		start = f.cursor - n + 1
	}
	return start, n
}

// Update handles one key press while the field has the focus (FR-006,
// FR-007, EC-015).
//
// In: msg, the key. Out: the field with the cursor moved to the previous
// choice that is not already added for `↑` and `k`, or to the next for `↓`
// and `j`, staying put at the ends; or with the choice under the cursor
// toggled for `Space`. Any other key is ignored. Side effects: none.
func (f multiChoiceField) Update(msg tea.KeyMsg) multiChoiceField {
	switch msg.String() {
	case "up", "k":
		if i := f.step(f.cursor, -1); f.cursor >= 0 && i >= 0 {
			f.cursor = i
		}
	case "down", "j":
		if i := f.step(f.cursor, 1); f.cursor >= 0 && i >= 0 {
			f.cursor = i
		}
	case " ", "space":
		f = f.toggle(f.cursor)
	}
	return f
}

// toggle flips the selection of one choice.
//
// In: i, a choice index. Out: the field with choice i selected if it was
// not, and deselected if it was; unchanged if i is out of range or the
// choice is already added. Side effects: none.
func (f multiChoiceField) toggle(i int) multiChoiceField {
	if i < 0 || i >= len(f.selected) || f.added[i] {
		return f
	}
	sel := slices.Clone(f.selected)
	sel[i] = !sel[i]
	f.selected = sel
	return f
}

// Click handles a left click on the field (FR-006, FR-007).
//
// In: row, the row of the click relative to the top of the field; space, the
// rows available to the field, as for View. Out: the field with the cursor
// on the clicked choice and that choice toggled if row is on a choice line
// that is not already added; unchanged for the title, the description, a
// row past the last shown choice and an already added choice. Side effects:
// none.
func (f multiChoiceField) Click(row, space int) multiChoiceField {
	start, n := f.window(max(space, 1))
	i := start + row - choiceFirstRow
	if row-choiceFirstRow < 0 || row-choiceFirstRow >= n || i >= len(f.added) || f.added[i] {
		return f
	}
	f.cursor = i
	return f.toggle(i)
}

// View renders the field.
//
// In: space, the rows available to the whole field; below 1 it counts as 1.
// Out: its lines: the title (`> ` before it when focused, two spaces
// otherwise), the description, then, focused, the choices indented by four
// spaces, each with `[x] `, `[ ] ` or `[✓] ` (followed by ` (already
// added)`) before it and `> ` in place of the two spaces before the one
// under the cursor, scrolling inside the field when space is too small with
// the cursor visible; unfocused, one line with the selected choices joined
// with `, `, empty if none. Lines are truncated with `…` to the width,
// never wrapped. When styled, the title, the cursor line, selected choices
// and already added lines use the styles of the design.
// Side effects: none.
func (f multiChoiceField) View(space int) []string {
	indent := strings.Repeat(" ", fieldIndent)
	title, style := indent+f.input.Name, f.styles.Normal
	if f.focused {
		title, style = checkCursor+f.input.Name, f.styles.FieldFocused
	}
	lines := []string{
		style.Render(truncate(title, f.width)),
		f.styles.Dim.Render(truncate(indent+f.input.Description, f.width)),
	}
	if !f.focused {
		line := ""
		if v := f.Value(); len(v) > 0 {
			line = indent + strings.Join(v, ", ")
		}
		return append(lines, f.styles.CheckSelected.Render(truncate(line, f.width)))
	}
	start, n := f.window(max(space, 1))
	for i := start; i < min(start+n, len(f.input.Choices)); i++ {
		lead := indent + checkNoCursor
		if i == f.cursor {
			lead = indent + checkCursor
		}
		text, st := f.input.Choices[i], f.styles.CheckUnselected
		marker := checkOther
		switch {
		case f.added[i]:
			marker, text, st = checkAdded, text+checkAddedSuffix, f.styles.AlreadyAdded
		case f.selected[i]:
			marker, st = checkSelected, f.styles.CheckSelected
		}
		if i == f.cursor {
			st = f.styles.CheckCursor
		}
		lines = append(lines, st.Render(truncate(lead+marker+text, f.width)))
	}
	return lines
}
