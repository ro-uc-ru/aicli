package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
)

// testEntries returns five entries, one of them a subcommand.
//
// In: none. Out: the entries. Side effects: none.
func testEntries() []Entry {
	return []Entry{
		{Path: "new", Short: "Create a new project"},
		{Path: "db", Short: "Database"},
		{Path: "db migrate", Short: "Apply migrations"},
		{Path: "nodesc"},
		{Path: "last", Short: "Last one"},
	}
}

// send applies messages to the model in order.
//
// In: m, the model; msgs, the messages. Out: the final model and the
// command returned by the last message. Side effects: none.
func send(m listModel, msgs ...tea.Msg) (listModel, tea.Cmd) {
	var cmd tea.Cmd
	for _, msg := range msgs {
		m, cmd = m.Update(msg)
	}
	return m, cmd
}

// key builds the key message for a key name as KeyMsg.String reports it.
//
// In: name, e.g. "down", "j", "enter", "ctrl+c". Out: the message.
// Side effects: none.
func key(name string) tea.KeyMsg {
	switch name {
	case "down":
		return tea.KeyMsg{Type: tea.KeyDown}
	case "up":
		return tea.KeyMsg{Type: tea.KeyUp}
	case "home":
		return tea.KeyMsg{Type: tea.KeyHome}
	case "end":
		return tea.KeyMsg{Type: tea.KeyEnd}
	case "enter":
		return tea.KeyMsg{Type: tea.KeyEnter}
	case "ctrl+c":
		return tea.KeyMsg{Type: tea.KeyCtrlC}
	}
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(name)}
}

// isQuit reports whether a command is tea.Quit.
//
// In: cmd, the command. Out: true if it produces tea.QuitMsg.
// Side effects: runs cmd.
func isQuit(cmd tea.Cmd) bool {
	if cmd == nil {
		return false
	}
	_, ok := cmd().(tea.QuitMsg)
	return ok
}

// opened returns the index carried by an open command.
//
// In: cmd, the command. Out: the entry index and true if cmd produces
// openMsg. Side effects: runs cmd.
func opened(cmd tea.Cmd) (int, bool) {
	if cmd == nil {
		return 0, false
	}
	msg, ok := cmd().(openMsg)
	return msg.Index, ok
}

// click builds a left-button press at row y.
//
// In: y, the screen row. Out: the message. Side effects: none.
func click(y int) tea.MouseMsg {
	return tea.MouseMsg{X: 3, Y: y, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft}
}

// TestListPointerKeys checks the start position and the movement keys,
// including the stops at both ends.
//
// Covers: FR-005, FR-006
func TestListPointerKeys(t *testing.T) {
	m := newListModel(testEntries(), false, 80, 40)
	if m.pointer != 0 || m.offset != 0 {
		t.Fatalf("start = %d, %d", m.pointer, m.offset)
	}
	steps := []struct {
		keys []string
		want int
	}{
		{[]string{"up"}, 0},
		{[]string{"k"}, 0},
		{[]string{"down"}, 1},
		{[]string{"j", "j"}, 3},
		{[]string{"down", "down", "down"}, 4},
		{[]string{"k"}, 3},
		{[]string{"up"}, 2},
		{[]string{"end"}, 4},
		{[]string{"home"}, 0},
	}
	for _, s := range steps {
		for _, k := range s.keys {
			m, _ = send(m, key(k))
		}
		if m.pointer != s.want {
			t.Errorf("after %v pointer = %d, want %d", s.keys, m.pointer, s.want)
		}
	}
}

// TestListOpenAndQuit checks Enter, q and Ctrl+C, and that other keys do
// nothing.
//
// Covers: FR-006, FR-007, EC-007
func TestListOpenAndQuit(t *testing.T) {
	m := newListModel(testEntries(), false, 80, 40)
	m, _ = send(m, key("j"), key("j"))
	if _, cmd := send(m, key("enter")); true {
		if i, ok := opened(cmd); !ok || i != 2 {
			t.Errorf("enter opened %d, %v; want 2", i, ok)
		}
	}
	for _, k := range []string{"q", "ctrl+c"} {
		if _, cmd := send(m, key(k)); !isQuit(cmd) {
			t.Errorf("%s did not quit", k)
		}
	}
	for _, k := range []string{"x", "esc", "left", "pgdown", "enter "} {
		got, cmd := send(m, key(k))
		if cmd != nil || got.pointer != m.pointer || got.offset != m.offset {
			t.Errorf("key %q changed the state", k)
		}
	}
}

// TestListScrollKeepsPointerVisible checks that keyboard moves scroll the
// list just enough to show the pointer, in both directions.
//
// Covers: FR-011
func TestListScrollKeepsPointerVisible(t *testing.T) {
	m := newListModel(testEntries(), false, 80, 14) // area of 2 rows
	if h := m.layout().AreaHeight; h != 2 {
		t.Fatalf("area = %d", h)
	}
	m, _ = send(m, key("down"), key("down"), key("down"))
	if m.pointer != 3 || m.offset != 2 {
		t.Errorf("pointer, offset = %d, %d; want 3, 2", m.pointer, m.offset)
	}
	m, _ = send(m, key("end"))
	if m.pointer != 4 || m.offset != 3 {
		t.Errorf("end: %d, %d; want 4, 3", m.pointer, m.offset)
	}
	m, _ = send(m, key("home"))
	if m.pointer != 0 || m.offset != 0 {
		t.Errorf("home: %d, %d; want 0, 0", m.pointer, m.offset)
	}
	m, _ = send(m, key("end"), key("up"), key("up"), key("up"), key("up"))
	if m.pointer != 0 || m.offset != 0 {
		t.Errorf("up: %d, %d; want 0, 0", m.pointer, m.offset)
	}
}

// TestListMouse checks the wheel (offset only, clamped), a click on an entry
// (moves and opens) and clicks elsewhere and motion (ignored).
//
// Covers: FR-007, FR-011, EC-004, EC-007
func TestListMouse(t *testing.T) {
	m := newListModel(testEntries(), false, 80, 14)
	wheel := func(b tea.MouseButton) tea.MouseMsg {
		return tea.MouseMsg{Action: tea.MouseActionPress, Button: b}
	}
	m, _ = send(m, wheel(tea.MouseButtonWheelDown), wheel(tea.MouseButtonWheelDown))
	if m.offset != 2 || m.pointer != 0 {
		t.Errorf("wheel down: offset %d pointer %d; want 2, 0", m.offset, m.pointer)
	}
	m, _ = send(m, wheel(tea.MouseButtonWheelDown), wheel(tea.MouseButtonWheelDown),
		wheel(tea.MouseButtonWheelDown), wheel(tea.MouseButtonWheelDown))
	if m.offset != 3 {
		t.Errorf("wheel clamp: offset %d; want 3", m.offset)
	}
	m, _ = send(m, wheel(tea.MouseButtonWheelUp), wheel(tea.MouseButtonWheelUp),
		wheel(tea.MouseButtonWheelUp), wheel(tea.MouseButtonWheelUp), wheel(tea.MouseButtonWheelUp))
	if m.offset != 0 {
		t.Errorf("wheel up clamp: offset %d; want 0", m.offset)
	}

	top := m.layout().AreaTop
	m, _ = send(m, wheel(tea.MouseButtonWheelDown))
	got, cmd := send(m, click(top+1))
	if i, ok := opened(cmd); !ok || i != 2 || got.pointer != 2 {
		t.Errorf("click: opened %d, %v pointer %d; want 2", i, ok, got.pointer)
	}
	for name, msg := range map[string]tea.MouseMsg{
		"banner":  click(1),
		"footer":  click(13),
		"past":    click(top + 5),
		"release": {Y: top, Action: tea.MouseActionRelease, Button: tea.MouseButtonLeft},
		"motion":  {Y: top, Action: tea.MouseActionMotion, Button: tea.MouseButtonNone},
		"right":   {Y: top, Action: tea.MouseActionPress, Button: tea.MouseButtonRight},
	} {
		got, cmd := send(m, msg)
		if cmd != nil || got.pointer != m.pointer || got.offset != m.offset {
			t.Errorf("%s changed the state", name)
		}
	}
}

// TestListResizeKeepsPointerVisible checks that a window-size message keeps
// the pointer and scrolls so that it stays visible.
//
// Covers: FR-005, FR-011
func TestListResizeKeepsPointerVisible(t *testing.T) {
	m := newListModel(testEntries(), false, 80, 40)
	m, _ = send(m, key("end"), tea.WindowSizeMsg{Width: 80, Height: 14})
	if m.pointer != 4 || m.offset != 3 {
		t.Errorf("shrink: %d, %d; want 4, 3", m.pointer, m.offset)
	}
	m, _ = send(m, tea.WindowSizeMsg{Width: 80, Height: 40})
	if m.pointer != 4 || m.offset != 0 {
		t.Errorf("grow: %d, %d; want 4, 0", m.pointer, m.offset)
	}
}

// TestListViewRows checks the pointer prefix, the padding, the footer, the
// banner and the absence of ANSI when not styled.
//
// Covers: FR-005, FR-013, FR-017
func TestListViewRows(t *testing.T) {
	m := newListModel(testEntries(), false, 80, 40)
	view := m.View()
	if strings.Contains(view, "\x1b") {
		t.Error("unstyled view has ESC")
	}
	lines := strings.Split(view, "\n")
	top := m.layout().AreaTop
	want := []string{
		"> new         Create a new project",
		"  db          Database",
		"  db migrate  Apply migrations",
		"  nodesc",
		"  last        Last one",
	}
	for i, w := range want {
		if lines[top+i] != w {
			t.Errorf("row %d = %q, want %q", i, lines[top+i], w)
		}
	}
	if lines[0] != bannerLines[0] {
		t.Errorf("banner line = %q", lines[0])
	}
	if last := lines[len(lines)-1]; last != footerList {
		t.Errorf("footer = %q", last)
	}
	m, _ = send(m, key("j"))
	lines = strings.Split(m.View(), "\n")
	if !strings.HasPrefix(lines[top], "  new") || !strings.HasPrefix(lines[top+1], "> db") {
		t.Errorf("pointer rows = %q, %q", lines[top], lines[top+1])
	}
}

// TestListViewStyled checks that the selected row differs from the others
// when styled, that the highlight follows the pointer, and that stripping
// ANSI gives the unstyled text.
//
// Covers: FR-013
func TestListViewStyled(t *testing.T) {
	plain := newListModel(testEntries(), false, 80, 40)
	styled := newListModel(testEntries(), true, 80, 40)
	if got := styleAnsiRe.ReplaceAllString(styled.View(), ""); got != plain.View() {
		t.Errorf("stripped styled view differs:\n%q\n%q", got, plain.View())
	}
	top := styled.layout().AreaTop
	rows := strings.Split(styled.View(), "\n")
	if !strings.Contains(rows[top], "\x1b") || rows[top][:2] == "  " {
		t.Errorf("selected row not styled: %q", rows[top])
	}
	moved, _ := send(styled, key("j"))
	after := strings.Split(moved.View(), "\n")
	if rows[top] == after[top] || rows[top+1] == after[top+1] {
		t.Error("highlight did not follow the pointer")
	}
	// The parent prefix of "db migrate" is dimmed: its row has more than one
	// styled segment while a top-level row has one.
	if strings.Count(rows[top+2], "\x1b[0m") < 2 {
		t.Errorf("parent prefix not dimmed: %q", rows[top+2])
	}
}

// TestListIndicators checks the scroll indicators on the reserved lines.
//
// Covers: FR-020
func TestListIndicators(t *testing.T) {
	m := newListModel(testEntries(), false, 80, 14)
	top := m.layout().AreaTop
	lines := strings.Split(m.View(), "\n")
	if lines[top-1] != "" || lines[top+2] != moreBelow {
		t.Errorf("at top: above %q below %q", lines[top-1], lines[top+2])
	}
	m, _ = send(m, key("down"), key("down"))
	lines = strings.Split(m.View(), "\n")
	if lines[top-1] != moreAbove || lines[top+2] != moreBelow {
		t.Errorf("middle: above %q below %q", lines[top-1], lines[top+2])
	}
	m, _ = send(m, key("end"))
	lines = strings.Split(m.View(), "\n")
	if lines[top-1] != moreAbove || lines[top+2] != "" {
		t.Errorf("bottom: above %q below %q", lines[top-1], lines[top+2])
	}
}

// TestListNarrow checks that lines are truncated, never wrapped, in a
// narrow terminal, with the narrow banner.
//
// Covers: FR-005
func TestListNarrow(t *testing.T) {
	m := newListModel(testEntries(), false, 12, 40)
	for i, l := range strings.Split(m.View(), "\n") {
		if ansiWidth(l) > 12 {
			t.Errorf("line %d too wide: %q", i, l)
		}
	}
	if first := strings.Split(m.View(), "\n")[0]; first != bannerNarrow {
		t.Errorf("first line = %q", first)
	}
}

// TestListEmpty checks the empty list: the message, the footer, and that
// only q and Ctrl+C act.
//
// Covers: EC-001, EC-004
func TestListEmpty(t *testing.T) {
	m := newListModel(nil, false, 80, 40)
	lines := strings.Split(m.View(), "\n")
	top := m.layout().AreaTop
	if lines[top] != noCommandsText || lines[len(lines)-1] != footerListEmpty {
		t.Errorf("view = %q", lines)
	}
	for _, msg := range []tea.Msg{key("enter"), key("down"), key("end"), key("j"), click(top)} {
		got, cmd := send(m, msg)
		if cmd != nil || got.pointer != 0 || got.offset != 0 {
			t.Errorf("%v acted on an empty list", msg)
		}
	}
	for _, k := range []string{"q", "ctrl+c"} {
		if _, cmd := send(m, key(k)); !isQuit(cmd) {
			t.Errorf("%s did not quit", k)
		}
	}
}

// ansiWidth is the display width of s.
//
// In: s, the text. Out: its width in cells. Side effects: none.
func ansiWidth(s string) int { return ansi.StringWidth(s) }

// TestListResizeKeepsEntry checks that shrinking and growing the terminal
// keep the pointer on the same entry and visible, including a resize that
// forces the offset back.
//
// Covers: FR-015
func TestListResizeKeepsEntry(t *testing.T) {
	m := newListModel(testEntries(), false, 80, 14) // area of 2
	m, _ = send(m, key("j"), key("j"), key("j"))    // pointer 3, offset 2
	for _, size := range []tea.WindowSizeMsg{{Width: 80, Height: 13}, {Width: 30, Height: 13}, {Width: 80, Height: 30}, {Width: 80, Height: 12}} {
		m, _ = send(m, size)
		l := m.layout()
		if m.pointer != 3 || m.pointer < m.offset || m.pointer >= m.offset+l.AreaHeight {
			t.Errorf("%v: pointer %d offset %d area %d", size, m.pointer, m.offset, l.AreaHeight)
		}
	}
	m, _ = send(m, tea.WindowSizeMsg{Width: 80, Height: 40})
	if m.offset != 0 || m.pointer != 3 {
		t.Errorf("grow: pointer %d offset %d", m.pointer, m.offset)
	}
}
