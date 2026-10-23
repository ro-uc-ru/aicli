package tui

import (
	"io"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

// addStyles holds the Lipgloss styles of the add screens (selector, form and
// confirmation), one per row of the colour table of the add-screen design
// (FR-006, FR-007, FR-008, FR-011, FR-016, FR-018).
type addStyles struct {
	// Header styles the headers `aicli › add` and `aicli › add › <feature>`.
	Header lipgloss.Style
	// Dim styles the description lines, field descriptions, footers,
	// scroll indicators and the `Will create:` heading.
	Dim lipgloss.Style
	// Normal styles unselected entries, unfocused field titles, input text,
	// the equivalent command, the paths and the unselected choices.
	Normal lipgloss.Style
	// Selected styles the entry under the pointer, pointer included.
	Selected lipgloss.Style
	// FieldFocused styles the title of the focused form field.
	FieldFocused lipgloss.Style
	// Choice styles the choice marker `(•) ` and the selected choice.
	Choice lipgloss.Style
	// Error styles validation errors and the `Cannot add …` block, in the
	// red of the `Error:` label of ERR-5.
	Error lipgloss.Style
	// Add styles the `Add` button, green; AddButton makes it bold
	// when it has the focus.
	Add lipgloss.Style
	// Cancel styles the `Cancel` button, red.
	Cancel lipgloss.Style
	// Bold is applied to the focused button.
	Bold lipgloss.Style
	// CheckCursor styles the cursor `> ` of a multi-choice checklist and the
	// choice under it (FR-006).
	CheckCursor lipgloss.Style
	// CheckSelected styles the marker `[x] ` and a selected choice (FR-006).
	CheckSelected lipgloss.Style
	// CheckUnselected styles the marker `[ ] ` and an unselected choice
	// (FR-006).
	CheckUnselected lipgloss.Style
	// AlreadyAdded styles a whole `[✓] <choice> (already added)` line (FR-007).
	AlreadyAdded lipgloss.Style
	// AllAdded styles the `All … already added.` message (EC-006).
	AllAdded lipgloss.Style
	// WillUpdate styles the `Will update:` heading of the confirmation
	// (FR-011).
	WillUpdate lipgloss.Style
}

// newAddStyles builds the add screen styles.
//
// In: styled, whether to add ANSI colour (see ShouldStyle).
// Out: the styles. When styled they use ANSI-256 colours on a private
// renderer; otherwise every style is a no-op, so the rendered text is the
// same and only ANSI sequences differ.
// Side effects: none; no global Lipgloss state changes.
func newAddStyles(styled bool) addStyles {
	r := lipgloss.NewRenderer(io.Discard)
	if !styled {
		r.SetColorProfile(termenv.Ascii)
		n := r.NewStyle()
		return addStyles{n, n, n, n, n, n, n, n, n, n, n, n, n, n, n, n}
	}
	r.SetColorProfile(termenv.ANSI256)
	accent := lipgloss.Color("117")
	dim := lipgloss.Color("245")
	normal := lipgloss.Color("255")
	return addStyles{
		Header:       r.NewStyle().Foreground(accent).Bold(true),
		Dim:          r.NewStyle().Foreground(dim),
		Normal:       r.NewStyle().Foreground(normal),
		Selected:     r.NewStyle().Foreground(accent).Bold(true).Background(lipgloss.Color("236")),
		FieldFocused: r.NewStyle().Foreground(accent).Bold(true),
		Choice:       r.NewStyle().Foreground(accent),
		Error:        r.NewStyle().Foreground(lipgloss.Color("1")),
		Add:          r.NewStyle().Foreground(lipgloss.Color("2")),
		Cancel:       r.NewStyle().Foreground(lipgloss.Color("1")),
		Bold:         r.NewStyle().Bold(true),

		CheckCursor:     r.NewStyle().Foreground(accent).Bold(true),
		CheckSelected:   r.NewStyle().Foreground(accent),
		CheckUnselected: r.NewStyle().Foreground(normal),
		AlreadyAdded:    r.NewStyle().Foreground(dim),
		AllAdded:        r.NewStyle().Foreground(dim),
		WillUpdate:      r.NewStyle().Foreground(dim),
	}
}

// AddButton renders a confirmation button.
//
// In: label, the button text; add, true for the `Add` button and false for
// `Cancel`; focused, whether the button has the focus.
// Out: `[ <label> ]` when focused, `  <label>  ` otherwise (same width),
// in the button's colour, and bold when focused.
// Side effects: none.
func (s addStyles) AddButton(label string, add, focused bool) string {
	style, text := s.Cancel, "  "+label+"  "
	if add {
		style = s.Add
	}
	if focused {
		style, text = style.Inherit(s.Bold), "[ "+label+" ]"
	}
	return style.Render(text)
}
