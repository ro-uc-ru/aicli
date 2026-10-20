package tui

import (
	"slices"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
)

const (
	// footerList is the key-hint footer of the list view (FR-017).
	footerList = "↑/↓ move · enter open · q quit"
	// footerListEmpty is the footer of the list view with no entry (EC-001).
	footerListEmpty = "q quit"
	// moreAbove and moreBelow are the scroll indicators (FR-020).
	moreAbove = "↑ more"
	moreBelow = "↓ more"
)

// openMsg asks the program to open the detail view of one entry (FR-007).
type openMsg struct {
	// Index is the index of the entry in the model's entry list.
	Index int
}

// listConfig is what distinguishes one use of the list component from
// another (FR-009).
type listConfig struct {
	// view is the layout view kind of the list.
	view view
	// top renders the block above the list when the layout shows it.
	// In: styles, the styles in use; width, the terminal width; l, the
	// layout. Out: the lines of the block, including its blank separator.
	top func(styles homeStyles, width int, l layout) []string
	// footer and footerEmpty are the key-hint footers with and without
	// entries.
	footer, footerEmpty string
	// quitKeys are the key names that quit the program.
	quitKeys []string
	// action builds the command run when the entry at index i is chosen by
	// `Enter` or by a click. It is never nil.
	action func(i int) tea.Cmd
}

// homeTop renders the banner of the home screen as the top block of the
// list.
//
// In: styles, the styles in use; width, the terminal width; l, the layout.
// Out: the banner lines and a blank separator, or nil if the layout hides
// the banner. Side effects: none.
func homeTop(styles homeStyles, width int, l layout) []string {
	if !l.Banner {
		return nil
	}
	var lines []string
	b := bannerFor(width)
	for i, s := range b {
		if i < len(b)-len(bannerDescription) {
			lines = append(lines, styles.Banner.Render(s))
		} else {
			lines = append(lines, styles.BannerDesc.Render(s))
		}
	}
	return append(lines, "")
}

// homeListConfig is the configuration of the home screen's list: the
// banner, the home footers, `q` and `Ctrl+C` to quit, and openMsg as the
// action.
var homeListConfig = listConfig{
	view:        viewList,
	top:         homeTop,
	footer:      footerList,
	footerEmpty: footerListEmpty,
	quitKeys:    []string{"q", "ctrl+c"},
	action: func(i int) tea.Cmd {
		return func() tea.Msg { return openMsg{Index: i} }
	},
}

// listModel is the Bubble Tea model of a list view: the home screen's list
// or, with another listConfig, the add selector's.
type listModel struct {
	entries []Entry
	styles  homeStyles
	cfg     listConfig
	width   int
	height  int
	pointer int
	offset  int
}

// newListModel builds the list view model.
//
// In: entries, the commands to list, in display order; styled, whether to
// add ANSI colour; width and height, the initial terminal size in cells.
// Out: the model, with the pointer on the first entry and the offset at 0
// (FR-005).
// Side effects: none.
func newListModel(entries []Entry, styled bool, width, height int) listModel {
	return newConfiguredListModel(homeListConfig, entries, styled, width, height)
}

// newConfiguredListModel builds a list model with its own configuration.
//
// In: cfg, the configuration, with action set; the other arguments as for
// newListModel.
// Out: the model, with the pointer on the first entry and the offset at 0.
// Side effects: none.
func newConfiguredListModel(cfg listConfig, entries []Entry, styled bool, width, height int) listModel {
	return listModel{entries: entries, styles: newHomeStyles(styled), cfg: cfg, width: width, height: height}
}

// layout returns the current screen layout of the list view.
//
// In: none. Out: the layout for the model's size and entry count.
// Side effects: none.
func (m listModel) layout() layout {
	return computeLayout(m.width, m.height, m.cfg.view, len(m.entries))
}

// clampOffset keeps the offset inside [0, entries-area].
//
// In: none. Out: the model with a valid offset. Side effects: none.
func (m listModel) clampOffset() listModel {
	m.offset = max(0, min(m.offset, len(m.entries)-m.layout().AreaHeight))
	return m
}

// showPointer scrolls the list just enough for the pointer to be visible.
//
// In: none. Out: the model with the pointer inside the visible rows.
// Side effects: none.
func (m listModel) showPointer() listModel {
	area := m.layout().AreaHeight
	if m.pointer < m.offset {
		m.offset = m.pointer
	} else if m.pointer >= m.offset+area {
		m.offset = m.pointer - area + 1
	}
	return m.clampOffset()
}

// open builds the command that acts on the entry under the pointer.
//
// In: none. Out: the configured action's command. Side effects: none.
func (m listModel) open() tea.Cmd {
	return m.cfg.action(m.pointer)
}

// Update handles one message (FR-006, FR-007, FR-010, FR-011, EC-001,
// EC-004, EC-007).
//
// In: msg, a key, mouse or window-size message; any other message is
// ignored.
// Out: the updated model and a command: tea.Quit for `q` and `Ctrl+C`, a
// command producing openMsg for `Enter` and for a left click on an entry,
// nil otherwise. The pointer moves with the arrows, `j`, `k`, `Home` and
// `End` and stops at the ends; the wheel changes only the offset; any other
// key, a click outside the entries and mouse motion are ignored. With no
// entry only `q` and `Ctrl+C` act.
// Side effects: none.
func (m listModel) Update(msg tea.Msg) (listModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		return m.showPointer(), nil
	case tea.KeyMsg:
		return m.updateKey(msg)
	case tea.MouseMsg:
		return m.updateMouse(msg)
	}
	return m, nil
}

// updateKey handles a key press in the list view.
//
// In: msg, the key. Out: the updated model and command, as for Update.
// Side effects: none.
func (m listModel) updateKey(msg tea.KeyMsg) (listModel, tea.Cmd) {
	key := msg.String()
	if slices.Contains(m.cfg.quitKeys, key) {
		return m, tea.Quit
	}
	if len(m.entries) == 0 {
		return m, nil
	}
	last := len(m.entries) - 1
	switch key {
	case "down", "j":
		m.pointer = min(m.pointer+1, last)
	case "up", "k":
		m.pointer = max(m.pointer-1, 0)
	case "home":
		m.pointer = 0
	case "end":
		m.pointer = last
	case "enter":
		return m, m.open()
	default:
		return m, nil
	}
	return m.showPointer(), nil
}

// updateMouse handles a mouse event in the list view.
//
// In: msg, the event. Out: the updated model and command, as for Update.
// Side effects: none.
func (m listModel) updateMouse(msg tea.MouseMsg) (listModel, tea.Cmd) {
	if msg.Action != tea.MouseActionPress {
		return m, nil
	}
	switch msg.Button {
	case tea.MouseButtonWheelUp:
		m.offset--
		return m.clampOffset(), nil
	case tea.MouseButtonWheelDown:
		m.offset++
		return m.clampOffset(), nil
	case tea.MouseButtonLeft:
		i, ok := entryAtRow(m.layout(), msg.Y, m.offset, len(m.entries))
		if !ok {
			return m, nil
		}
		m.pointer = i
		return m, m.open()
	}
	return m, nil
}

// Init is part of the Bubble Tea model contract.
//
// In: none. Out: nil, no initial command. Side effects: none.
func (m listModel) Init() tea.Cmd { return nil }

// View renders the list view (FR-002, FR-005, FR-013, FR-017, FR-020).
//
// In: none.
// Out: the screen text, without a trailing newline: the banner when it
// fits, the indicator line above, the visible entries (the pointer entry
// prefixed with `> `, the others with two spaces), the indicator line
// below and the footer. Lines are truncated with `…` to the width and never
// wrapped. With no entry the area shows `No commands available.`.
// Side effects: none.
func (m listModel) View() string {
	l := m.layout()
	var lines []string
	lines = append(lines, m.cfg.top(m.styles, m.width, l)...)
	if l.Indicators {
		lines = append(lines, m.indicator(m.offset > 0, moreAbove))
	}
	if len(m.entries) == 0 {
		lines = append(lines, m.styles.Entry.Render(truncate(noCommandsText, m.width)))
	}
	pathWidth := 0
	for _, e := range m.entries {
		pathWidth = max(pathWidth, ansi.StringWidth(e.Path))
	}
	for i := m.offset; i < min(m.offset+l.AreaHeight, len(m.entries)); i++ {
		lines = append(lines, m.row(i, pathWidth))
	}
	if l.Indicators {
		lines = append(lines, m.indicator(m.offset+l.AreaHeight < len(m.entries), moreBelow))
	}
	if l.Footer {
		footer := m.cfg.footer
		if len(m.entries) == 0 {
			footer = m.cfg.footerEmpty
		}
		lines = append(lines, m.styles.Footer.Render(truncate(footer, m.width)))
	}
	return strings.Join(lines, "\n")
}

// indicator renders one scroll-indicator line.
//
// In: show, whether content is hidden on that side; text, the indicator.
// Out: the styled text, or an empty line. Side effects: none.
func (m listModel) indicator(show bool, text string) string {
	if !show {
		return ""
	}
	return m.styles.Indicator.Render(truncate(text, m.width))
}

// row renders the entry at index i.
//
// In: i, the entry index; pathWidth, the width the paths are padded to.
// Out: the line: the pointer prefix, the padded path, two spaces and the
// short description (the path alone without one), truncated to the width.
// The pointer row takes the selected style; on the others the parent prefix
// of a subcommand path is dimmed. Side effects: none.
func (m listModel) row(i, pathWidth int) string {
	e := m.entries[i]
	prefix := "  "
	if i == m.pointer {
		prefix = "> "
	}
	text := prefix + e.Path
	if e.Short != "" {
		text = prefix + padRight(e.Path, pathWidth) + "  " + e.Short
	}
	text = truncate(text, m.width)
	if i == m.pointer {
		return m.styles.Selected.Render(text)
	}
	cut := 0
	if sp := strings.LastIndex(e.Path, " "); sp >= 0 {
		cut = len(prefix) + sp + 1
	}
	if cut == 0 {
		return m.styles.Entry.Render(text)
	}
	if len(text) >= cut && text[:cut] == (prefix + e.Path)[:cut] {
		return m.styles.Parent.Render(text[:cut]) + m.styles.Entry.Render(text[cut:])
	}
	return m.styles.Parent.Render(text)
}
