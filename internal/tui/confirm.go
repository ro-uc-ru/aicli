package tui

import (
	"strings"
	"unicode"
	"unicode/utf8"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
)

const (
	// footerConfirm and footerBlocked are the key hints of the confirmation
	// and of the blocked confirmation (FR-018).
	footerConfirm = "←/→/tab choose · enter confirm · y add · n cancel · esc back · ctrl+c quit"
	footerBlocked = "enter cancel · n cancel · esc back · ctrl+c quit"
	// hintScroll is added in front of the footer when the paths or the
	// command scroll (EC-010).
	hintScroll = "↑/↓ scroll · "
	// headingCreate and headingUpdate are the headings of the two lists of
	// the confirmation (FR-011).
	headingCreate = "Will create:"
	headingUpdate = "Will update:"
	// plainErrorPrefix is the prefix of an error message, removed before it
	// is shown in the confirmation.
	plainErrorPrefix = "Error: "
)

// confirmResult is how the confirmation ended.
type confirmResult int

const (
	// confirmPending means the confirmation is still shown.
	confirmPending confirmResult = iota
	// confirmAdd means the user pressed `Add`.
	confirmAdd
	// confirmCancel means the user pressed `Cancel`.
	confirmCancel
	// confirmBack means the user pressed `Esc`.
	confirmBack
)

// confirmModel is the model of the confirmation screen (FR-018).
type confirmModel struct {
	feature Feature
	values  []string
	styles  addStyles
	width   int
	height  int

	// command is the wrapped equivalent command.
	command []string
	// rest holds the rows of the list: `Will create:`, its indented paths, a
	// blank row, `Will update:` and its indented paths (each heading only
	// when it has paths), or, when blocked, the lines of the error.
	rest    []string
	blocked bool
	// focusAdd is true when `Add` has the focus.
	focusAdd bool
	// cmdOffset and restOffset are the first visible line of the command
	// and of the file list.
	cmdOffset, restOffset int
	// result is how the confirmation ended; confirmPending until then.
	result confirmResult
}

// newConfirmModel builds the confirmation of a feature for the entered
// values, computing the plan and running the checks again.
//
// In: feature, the feature; values, one per input, trimmed (nil without
// inputs); styled, whether to add ANSI colour; width and height, the
// initial terminal size in cells.
// Out: the model, with the focus on `Cancel`, and tea.DisableMouse, so the
// user can select and copy the command (FR-018). If the plan or the check
// fails, the model is blocked and shows the error instead of the paths.
// Side effects: calls feature.Plan, feature.Check and feature.Command; they
// write nothing.
func newConfirmModel(feature Feature, values []string, styled bool, width, height int) (confirmModel, tea.Cmd) {
	m := confirmModel{feature: feature, values: values, styles: newAddStyles(styled), width: width, height: height}
	cmd := "aicli add " + feature.Name
	if feature.Command != nil {
		cmd = feature.Command(values)
	}
	m.command = wrapCommand(cmd, width)
	var preview Preview
	var err error
	if feature.Plan != nil {
		preview.Create, err = feature.Plan(values)
	}
	if err == nil && feature.Check != nil {
		preview, err = feature.Check(values)
	}
	if err != nil {
		m.blocked = true
		m.rest = errorLines(err)
		return m, tea.DisableMouse
	}
	m.rest = previewRows(preview)
	return m, tea.DisableMouse
}

// previewRows lays out the two lists of the confirmation as rows.
//
// In: p, the preview. Out: `Will create:` and its paths indented by two
// spaces, a blank row, `Will update:` and its paths; each heading only when
// it has paths, and the blank row only when both are shown. Side effects:
// none.
func previewRows(p Preview) []string {
	var rows []string
	for _, l := range []struct {
		heading string
		paths   []string
	}{{headingCreate, p.Create}, {headingUpdate, p.Update}} {
		if len(l.paths) == 0 {
			continue
		}
		if len(rows) > 0 {
			rows = append(rows, "")
		}
		rows = append(rows, l.heading)
		for _, path := range l.paths {
			rows = append(rows, "  "+path)
		}
	}
	return rows
}

// wrapCommand wraps the equivalent command to the width without cutting
// anything.
//
// In: cmd, the command; width, the terminal width. Out: its lines, broken at
// spaces and, for a word wider than the width, inside it, none wider than
// the width and without trailing spaces; at least one.
// Side effects: none.
func wrapCommand(cmd string, width int) []string {
	width = max(width, 1)
	lines := strings.Split(ansi.Hardwrap(ansi.Wordwrap(cmd, width, ""), width, true), "\n")
	for i, l := range lines {
		lines[i] = strings.TrimRight(l, " ")
	}
	return lines
}

// errorLines turns an error into the lines the confirmation shows.
//
// In: err, a non-nil error. Out: its message split into lines, without the
// `Error: ` prefix and with the first letter capitalised; at least one line.
// Every line after the first is indented by two spaces, as in the plain error
// of ERR-5, unless the message already indented it.
// Side effects: none.
func errorLines(err error) []string {
	msg := strings.TrimPrefix(err.Error(), plainErrorPrefix)
	if r, n := utf8.DecodeRuneInString(msg); n > 0 {
		msg = string(unicode.ToUpper(r)) + msg[n:]
	}
	lines := strings.Split(msg, "\n")
	for i := 1; i < len(lines); i++ {
		if !strings.HasPrefix(lines[i], " ") {
			lines[i] = "  " + lines[i]
		}
	}
	return lines
}

// Result returns how the confirmation ended.
//
// In: none. Out: confirmPending while it is shown, otherwise the button or
// key that closed it. Side effects: none.
func (m confirmModel) Result() confirmResult { return m.result }

// Init is part of the Bubble Tea model contract.
//
// In: none. Out: nil, no initial command. Side effects: none.
func (m confirmModel) Init() tea.Cmd { return nil }

// layout returns the current screen layout.
//
// In: none. Out: the layout for the size, the command and the list. Side
// effects: none.
func (m confirmModel) layout() confirmLayout {
	return computeConfirmLayout(m.height, m.feature.Short != "", len(m.command), len(m.rest))
}

// Update handles one message (FR-018, EC-011).
//
// In: msg, a key or window-size message; mouse and other messages are
// ignored, since the mouse is released.
// Out: the updated model and a command. `←`, `→` and `Tab` move the focus
// between the buttons; `Enter` presses the focused one, `y` presses `Add`,
// `n` presses `Cancel` (the result is set, with no command); `Esc` sets
// confirmBack and returns tea.EnableMouseCellMotion; `Ctrl+C` quits;
// `↑`/`↓`/`k`/`j` scroll what is cut. When blocked only `Cancel` exists, so
// `y`, `←`, `→` and `Tab` do nothing. Other keys do nothing.
// Side effects: none.
func (m confirmModel) Update(msg tea.Msg) (confirmModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.command = wrapCommand(m.rawCommand(), m.width)
		return m.clampOffsets(), nil
	case tea.KeyMsg:
		return m.updateKey(msg)
	}
	return m, nil
}

// rawCommand returns the equivalent command on one line.
//
// In: none. Out: the command as built, unwrapped. Side effects: none.
func (m confirmModel) rawCommand() string {
	if m.feature.Command == nil {
		return "aicli add " + m.feature.Name
	}
	return m.feature.Command(m.values)
}

// updateKey handles a key press.
//
// In: msg, the key. Out: the updated model and command, as for Update.
// Side effects: none.
func (m confirmModel) updateKey(msg tea.KeyMsg) (confirmModel, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit
	case "esc":
		m.result = confirmBack
		return m, tea.EnableMouseCellMotion
	case "n":
		m.result = confirmCancel
	case "enter":
		m.result = confirmCancel
		if m.focusAdd && !m.blocked {
			m.result = confirmAdd
		}
	case "y":
		if !m.blocked {
			m.result = confirmAdd
		}
	case "left", "right", "tab":
		if !m.blocked {
			m.focusAdd = !m.focusAdd
		}
	case "up", "k":
		m = m.scroll(-1)
	case "down", "j":
		m = m.scroll(1)
	}
	return m, nil
}

// scroll moves the list, or the command when the list is hidden.
//
// In: delta, the number of lines, negative for up. Out: the model with the
// offset moved and clamped; unchanged if nothing is cut. Side effects: none.
func (m confirmModel) scroll(delta int) confirmModel {
	l := m.layout()
	switch {
	case l.List && l.ListRows < len(m.rest):
		m.restOffset += delta
	case l.CmdRows < len(m.command):
		m.cmdOffset += delta
	}
	return m.clampOffsets()
}

// clampOffsets keeps both offsets inside what is cut.
//
// In: none. Out: the model with valid offsets. Side effects: none.
func (m confirmModel) clampOffsets() confirmModel {
	l := m.layout()
	m.cmdOffset = max(0, min(m.cmdOffset, len(m.command)-l.CmdRows))
	rows := len(m.rest)
	if l.List {
		rows = l.ListRows
	}
	m.restOffset = max(0, min(m.restOffset, len(m.rest)-rows))
	return m
}

// footer returns the key hints.
//
// In: l, the layout. Out: the footer text, with `↑/↓ scroll` first when
// something is cut. Side effects: none.
func (m confirmModel) footer(l confirmLayout) string {
	f := footerConfirm
	if m.blocked {
		f = footerBlocked
	}
	if l.Scrolls {
		f = hintScroll + f
	}
	return f
}

// buttons renders the buttons line.
//
// In: none. Out: `Add` and `Cancel` side by side, or `Cancel` alone when
// blocked, each in its own colour. Side effects: none.
func (m confirmModel) buttons() string {
	if m.blocked {
		return m.styles.AddButton("Cancel", false, true)
	}
	return m.styles.AddButton("Add", true, m.focusAdd) + "  " + m.styles.AddButton("Cancel", false, !m.focusAdd)
}

// View renders the confirmation (FR-012, FR-018, EC-008, EC-010).
//
// In: none.
// Out: the screen text, without a trailing newline: the header, the
// description, the visible command lines, the indicator line above (blank
// when nothing is hidden), the visible list rows, the indicator line below,
// the buttons and the footer, each part only if the layout shows it. The
// description and list rows are truncated with `…`; the command is never
// truncated. The headings are dimmed and the error, when blocked, is in the
// error colour.
// Side effects: none.
func (m confirmModel) View() string {
	l := m.layout()
	var lines []string
	if l.Header {
		lines = append(lines, m.styles.Header.Render(truncate("aicli › add › "+m.feature.Name, m.width)))
	}
	if l.Description {
		lines = append(lines, m.styles.Dim.Render(truncate(m.feature.Short, m.width)), "")
	}
	for _, c := range m.command[m.cmdOffset : m.cmdOffset+l.CmdRows] {
		lines = append(lines, m.styles.Normal.Render(c))
	}
	if l.List {
		lines = append(lines, m.indicator(m.restOffset > 0, moreAbove))
		for i := m.restOffset; i < m.restOffset+l.ListRows; i++ {
			if i < len(m.rest) {
				lines = append(lines, m.restLine(m.rest[i]))
			} else {
				lines = append(lines, "")
			}
		}
		lines = append(lines, m.indicator(m.restOffset+l.ListRows < len(m.rest), moreBelow))
	}
	lines = append(lines, m.buttons())
	if l.Footer {
		lines = append(lines, m.styles.Dim.Render(truncate(m.footer(l), m.width)))
	}
	return strings.Join(lines, "\n")
}

// restLine renders one line of the file list.
//
// In: s, the row. Out: the truncated row in the normal colour, in the error
// colour when blocked, or dimmed for a heading (`Will update:` in its own
// style). Side effects: none.
func (m confirmModel) restLine(s string) string {
	switch {
	case m.blocked:
		return m.styles.Error.Render(truncate(s, m.width))
	case s == headingCreate:
		return m.styles.Dim.Render(truncate(s, m.width))
	case s == headingUpdate:
		return m.styles.WillUpdate.Render(truncate(s, m.width))
	}
	return m.styles.Normal.Render(truncate(s, m.width))
}

// indicator renders one scroll-indicator line.
//
// In: show, whether lines are hidden on that side; text, the indicator.
// Out: the styled text, or an empty line. Side effects: none.
func (m confirmModel) indicator(show bool, text string) string {
	if !show {
		return ""
	}
	return m.styles.Dim.Render(truncate(text, m.width))
}
