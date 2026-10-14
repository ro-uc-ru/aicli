package tui

import (
	"fmt"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// sampleHelp is a short help text with a heading and flags.
const sampleHelp = "Create a new aicli project\n\nUsage:\n  aicli new <name> [flags]\n\nFlags:\n  -h, --help   help for new\n"

// longHelp builds a help text of n numbered lines.
//
// In: n, the number of lines. Out: the text. Side effects: none.
func longHelp(n int) string {
	var b strings.Builder
	for i := 1; i <= n; i++ {
		fmt.Fprintf(&b, "line %02d\n", i)
	}
	return b.String()
}

// areaLines returns the help lines of a rendered detail view, without the
// header, the indicator lines and the footer.
//
// In: m, the model. Out: the visible help lines. Side effects: none.
func areaLines(m detailModel) []string {
	l := computeLayout(m.width, m.height, viewDetail, len(m.lines))
	lines := strings.Split(m.View(), "\n")
	top := l.AreaTop
	return lines[top : top+l.AreaHeight]
}

// sendD applies messages to the detail model in order.
//
// In: m, the model; msgs, the messages. Out: the final model and the
// command of the last message. Side effects: none.
func sendD(m detailModel, msgs ...tea.Msg) (detailModel, tea.Cmd) {
	var cmd tea.Cmd
	for _, msg := range msgs {
		m, cmd = m.Update(msg)
	}
	return m, cmd
}

// TestDetailShowsHelp checks the header, the footer and that the help area
// equals the help text when the terminal is wide enough.
//
// Covers: FR-008, FR-017, FR-019
func TestDetailShowsHelp(t *testing.T) {
	m, _ := newDetailModel(false, 80, 40).open("db migrate", sampleHelp)
	lines := strings.Split(m.View(), "\n")
	if lines[0] != "aicli › db migrate" || lines[1] != "" {
		t.Errorf("header = %q", lines[:2])
	}
	if last := lines[len(lines)-1]; last != footerDetail {
		t.Errorf("footer = %q", last)
	}
	got := strings.Join(areaLines(m), "\n")
	if want := strings.TrimSuffix(sampleHelp, "\n"); got != want {
		t.Errorf("help area = %q, want %q", got, want)
	}
	if strings.Contains(m.View(), "\x1b") {
		t.Error("unstyled view has ESC")
	}
}

// TestDetailHeaderTruncated checks that a header wider than the terminal is
// truncated with `…`.
//
// Covers: EC-002, FR-019
func TestDetailHeaderTruncated(t *testing.T) {
	m, _ := newDetailModel(false, 10, 40).open("a-very-long-path", sampleHelp)
	if first := strings.Split(m.View(), "\n")[0]; first != "aicli › a…" {
		t.Errorf("header = %q", first)
	}
}

// TestDetailWraps checks that long lines are wrapped by display width and
// no character is lost, and that lines that fit are untouched.
//
// Covers: EC-006
func TestDetailWraps(t *testing.T) {
	help := "short\n" + strings.Repeat("abcdefghij", 3) + "\n日本語日本語\n"
	m, _ := newDetailModel(false, 10, 40).open("x", help)
	want := []string{"short", "abcdefghij", "abcdefghij", "abcdefghij", "日本語日本", "語"}
	if strings.Join(m.lines, "|") != strings.Join(want, "|") {
		t.Errorf("lines = %q, want %q", m.lines, want)
	}
	for _, l := range m.lines {
		if ansiWidth(l) > 10 {
			t.Errorf("line %q wider than 10", l)
		}
	}
}

// TestDetailEmphasis checks that styling only adds ANSI sequences around
// headings and flags.
//
// Covers: FR-008
func TestDetailEmphasis(t *testing.T) {
	plain, _ := newDetailModel(false, 80, 40).open("new", sampleHelp)
	styled, _ := newDetailModel(true, 80, 40).open("new", sampleHelp)
	pl, sl := areaLines(plain), areaLines(styled)
	for i := range pl {
		if got := styleAnsiRe.ReplaceAllString(sl[i], ""); got != pl[i] {
			t.Errorf("line %d: stripped %q, plain %q", i, got, pl[i])
		}
	}
	for _, i := range []int{2, 5, 6} { // "Usage:", "Flags:", the flags line
		if !strings.Contains(sl[i], "\x1b") {
			t.Errorf("line %d not emphasised: %q", i, sl[i])
		}
	}
	if strings.Contains(sl[0], "\x1b") || strings.Contains(sl[3], "\x1b") {
		t.Error("plain text lines were emphasised")
	}
}

// TestDetailScrollKeys checks the scroll keys, Home and End, the indicators
// and that other keys, letters and the mouse do nothing.
//
// Covers: FR-009, FR-020, EC-007
func TestDetailScrollKeys(t *testing.T) {
	m, _ := newDetailModel(false, 80, 11).open("x", longHelp(30)) // area of 6
	if h := m.vp.Height; h != 6 {
		t.Fatalf("area = %d", h)
	}
	lines := strings.Split(m.View(), "\n")
	if lines[2] != "" || lines[9] != moreBelow {
		t.Errorf("top indicators: %q / %q", lines[2], lines[9])
	}
	steps := []struct {
		msg  tea.Msg
		want int
	}{
		{key("down"), 1}, {key("j"), 2}, {key("up"), 1}, {key("k"), 0}, {key("k"), 0},
		{tea.KeyMsg{Type: tea.KeyPgDown}, 6}, {tea.KeyMsg{Type: tea.KeyPgUp}, 0},
		{key("end"), 24}, {key("down"), 24}, {key("home"), 0},
	}
	for i, s := range steps {
		m, _ = sendD(m, s.msg)
		if m.vp.YOffset != s.want {
			t.Errorf("step %d: offset %d, want %d", i, m.vp.YOffset, s.want)
		}
	}
	m, _ = sendD(m, key("end"))
	lines = strings.Split(m.View(), "\n")
	if lines[2] != moreAbove || lines[9] != "" {
		t.Errorf("bottom indicators: %q / %q", lines[2], lines[9])
	}
	m, _ = sendD(m, key("home"), key("j"))
	for name, msg := range map[string]tea.Msg{
		"x": key("x"), "enter": key("enter"), "space": key(" "),
		"wheel": tea.MouseMsg{Action: tea.MouseActionPress, Button: tea.MouseButtonWheelDown},
		"click": tea.MouseMsg{Action: tea.MouseActionPress, Button: tea.MouseButtonLeft},
	} {
		got, cmd := sendD(m, msg)
		if cmd != nil || got.vp.YOffset != m.vp.YOffset || got.closed {
			t.Errorf("%s changed the state", name)
		}
	}
}

// TestDetailOpenResets checks that every open starts at the top with its
// own help, and returns the disable-mouse command.
//
// Covers: FR-009, EC-011
func TestDetailOpenResets(t *testing.T) {
	m, cmd := newDetailModel(false, 80, 10).open("a", longHelp(30))
	if cmd == nil || cmd() != tea.DisableMouse() {
		t.Error("open did not return the disable-mouse command")
	}
	m, _ = sendD(m, key("end"))
	if m.vp.YOffset == 0 {
		t.Fatal("did not scroll")
	}
	m, _ = m.open("a", longHelp(30))
	if m.vp.YOffset != 0 {
		t.Errorf("reopen offset = %d", m.vp.YOffset)
	}
	m, _ = m.open("b", "other help\n")
	if m.path != "b" || m.vp.YOffset != 0 || areaLines(m)[0] != "other help" {
		t.Errorf("second entry: path %q lines %q", m.path, areaLines(m))
	}
}

// TestDetailBackAndQuit checks that Esc, Left and Backspace close the view
// with the enable-mouse command, and that q and Ctrl+C quit.
//
// Covers: FR-009, FR-010
func TestDetailBackAndQuit(t *testing.T) {
	m, _ := newDetailModel(false, 80, 10).open("a", sampleHelp)
	for _, k := range []tea.KeyMsg{{Type: tea.KeyEsc}, {Type: tea.KeyLeft}, {Type: tea.KeyBackspace}} {
		got, cmd := sendD(m, k)
		if !got.closed || cmd == nil || cmd() != tea.EnableMouseCellMotion() {
			t.Errorf("%v: closed %v, enable-mouse command missing", k, got.closed)
		}
	}
	for _, k := range []string{"q", "ctrl+c"} {
		if _, cmd := sendD(m, key(k)); !isQuit(cmd) {
			t.Errorf("%s did not quit", k)
		}
	}
}

// TestDetailHidesParts checks that a short terminal hides indicators, the
// footer and the header in this order, keeping one help line.
//
// Covers: EC-003, FR-019
func TestDetailHidesParts(t *testing.T) {
	m, _ := newDetailModel(false, 80, 2).open("x", longHelp(30))
	if got := m.View(); got != "line 01\nline 02" {
		t.Errorf("height 2 view = %q", got)
	}
	m, _ = newDetailModel(false, 80, 1).open("x", longHelp(30))
	if got := m.View(); got != "line 01" {
		t.Errorf("height 1 view = %q", got)
	}
}

// TestDetailResize checks that a resize re-wraps the help and keeps the
// first visible line: shrinking, growing, and a resize that forces the
// clamp so the end of the text is not above the bottom.
//
// Covers: FR-015
func TestDetailResize(t *testing.T) {
	help := "intro\n" + strings.Repeat("0123456789", 3) + "\n" + longHelp(12)
	m, _ := newDetailModel(false, 80, 8).open("x", help) // area of 3
	if len(m.lines) != 14 {
		t.Fatalf("lines = %d", len(m.lines))
	}
	// Scroll so that the first visible line is the long line, then shrink.
	m, _ = sendD(m, key("j"))
	m, _ = sendD(m, tea.WindowSizeMsg{Width: 10, Height: 8})
	if m.vp.YOffset != 1 || m.lines[m.vp.YOffset] != "0123456789" {
		t.Fatalf("shrink: offset %d, first %q", m.vp.YOffset, m.lines[m.vp.YOffset])
	}
	m, _ = sendD(m, key("j"), key("j"))
	if m.vp.YOffset != 3 || m.lines[3] != "0123456789" {
		t.Fatalf("setup: offset %d", m.vp.YOffset)
	}
	// Growing re-joins the segments: the first visible line stays in the same source line.
	m, _ = sendD(m, tea.WindowSizeMsg{Width: 80, Height: 8})
	if m.vp.YOffset != 1 || m.lines[1] != strings.Repeat("0123456789", 3) {
		t.Errorf("grow: offset %d, first %q", m.vp.YOffset, m.lines[m.vp.YOffset])
	}
	// A taller terminal forces the clamp: the end of the text stays at the bottom.
	m, _ = sendD(m, key("end"))
	m, _ = sendD(m, tea.WindowSizeMsg{Width: 80, Height: 12}) // area of 7
	if want := len(m.lines) - m.vp.Height; m.vp.YOffset != want || !m.vp.AtBottom() {
		t.Errorf("clamp: offset %d, want %d", m.vp.YOffset, want)
	}
	// Everything fits: back to the top.
	m, _ = sendD(m, tea.WindowSizeMsg{Width: 80, Height: 40})
	if m.vp.YOffset != 0 {
		t.Errorf("fits: offset %d", m.vp.YOffset)
	}
}
