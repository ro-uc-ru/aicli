package tui

import (
	tea "github.com/charmbracelet/bubbletea"
)

const (
	// footerSelector is the key-hint footer of the selector (FR-008).
	footerSelector = "↑/↓ move · enter select · q quit"
	// footerSelectorEmpty is the footer of a selector with no entry.
	footerSelectorEmpty = "q quit"
)

// selectMsg asks the program to select one feature of the selector
// (FR-010).
type selectMsg struct {
	// Index is the index of the feature in the selector's list.
	Index int
}

// selectorTop renders the header, the description line and the blank line
// of the selector, hiding each part the layout hides (FR-008, EC-010).
//
// In: styles, the styles in use; width, the terminal width; l, the layout.
// Out: the lines, truncated with `…` to the width, never wrapped; nil if
// the layout hides everything. Side effects: none.
func selectorTop(styles homeStyles, width int, l layout) []string {
	var lines []string
	if l.Header {
		lines = append(lines, styles.Header.Render(truncate("aicli › add", width)))
	}
	if l.Description {
		lines = append(lines, styles.BannerDesc.Render(truncate(addDescription, width)), "")
	}
	return lines
}

// selectorListConfig is the configuration of the add selector's list: the
// header block, the selector footers, `q`, `Esc` and `Ctrl+C` to quit, and
// selectMsg as the action (FR-009, FR-010).
var selectorListConfig = listConfig{
	view:        viewSelector,
	top:         selectorTop,
	footer:      footerSelector,
	footerEmpty: footerSelectorEmpty,
	quitKeys:    []string{"q", "esc", "ctrl+c"},
	action: func(i int) tea.Cmd {
		return func() tea.Msg { return selectMsg{Index: i} }
	},
}

// newSelectorModel builds the add selector as a list model (FR-008, FR-009).
//
// In: features, the listed features in Cobra order; styled, whether to add
// ANSI colour; width and height, the initial terminal size in cells.
// Out: the model, with one entry per feature (name padded by display width,
// then the short description) and the pointer on the first (FR-008,
// EC-006).
// Side effects: none.
func newSelectorModel(features []Feature, styled bool, width, height int) listModel {
	entries := make([]Entry, len(features))
	for i, f := range features {
		entries[i] = Entry{Path: f.Name, Short: f.Short}
	}
	return newConfiguredListModel(selectorListConfig, entries, styled, width, height)
}
