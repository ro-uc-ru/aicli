package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// selectorFeatures are the features used by the selector tests.
var selectorFeatures = []Feature{
	{Name: "feature-1", Short: "Description of feature 1", Listed: true},
	{Name: "f-2", Short: "Description of feature 2", Listed: true},
	{Name: "日本語", Listed: true},
}

// selectorKey builds the key message for a key name used by the tests.
func selectorKey(name string) tea.KeyMsg {
	switch name {
	case "enter":
		return tea.KeyMsg{Type: tea.KeyEnter}
	case "esc":
		return tea.KeyMsg{Type: tea.KeyEsc}
	case "ctrl+c":
		return tea.KeyMsg{Type: tea.KeyCtrlC}
	case "down":
		return tea.KeyMsg{Type: tea.KeyDown}
	}
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(name)}
}

// TestSelectorView checks the header, the description line, the entries
// padded by display width, the pointer on the first entry, the footer, and
// the absence of the banner.
//
// Covers: FR-008, EC-006
func TestSelectorView(t *testing.T) {
	m := newSelectorModel(selectorFeatures, false, 80, 24)
	lines := strings.Split(m.View(), "\n")
	want := []string{
		"aicli › add",
		"Add an AI/LLM feature to this project.",
		"",
		"",
		"> feature-1  Description of feature 1",
		"  f-2        Description of feature 2",
		"  日本語",
		"",
		"↑/↓ move · enter select · q quit",
	}
	if len(lines) != len(want) {
		t.Fatalf("lines = %q", lines)
	}
	for i := range want {
		if lines[i] != want[i] {
			t.Errorf("line %d = %q, want %q", i, lines[i], want[i])
		}
	}
	if strings.Contains(m.View(), "AICLI") {
		t.Error("selector shows the banner")
	}
}

// TestSelectorViewHidesParts checks the view on a short terminal: only the
// parts the layout keeps are drawn.
//
// Covers: FR-008, EC-010
func TestSelectorViewHidesParts(t *testing.T) {
	m := newSelectorModel(selectorFeatures, false, 80, 5)
	got := m.View()
	if strings.Contains(got, "Add an AI/LLM") || !strings.HasPrefix(got, "aicli › add") {
		t.Errorf("view = %q", got)
	}
	m = newSelectorModel(selectorFeatures, false, 10, 24)
	if v := m.View(); !strings.Contains(v, "Add an AI…") {
		t.Errorf("description not truncated: %q", v)
	}
}

// TestSelectorSelect checks that Enter and a click on an entry produce a
// selection of that entry, and that a click outside the entries does not.
//
// Covers: FR-009, FR-010, EC-011
func TestSelectorSelect(t *testing.T) {
	m := newSelectorModel(selectorFeatures, false, 80, 24)
	m, _ = m.Update(selectorKey("down"))
	_, cmd := m.Update(selectorKey("enter"))
	if msg := cmd(); msg != (selectMsg{Index: 1}) {
		t.Errorf("enter gave %v", msg)
	}
	// Entries start at row 4 (header, description, blank, indicator).
	m2, cmd := m.Update(tea.MouseMsg{Action: tea.MouseActionPress, Button: tea.MouseButtonLeft, Y: 6})
	if msg := cmd(); msg != (selectMsg{Index: 2}) || m2.pointer != 2 {
		t.Errorf("click gave %v, pointer %d", msg, m2.pointer)
	}
	for _, y := range []int{0, 3, 20} {
		if _, cmd := m.Update(tea.MouseMsg{Action: tea.MouseActionPress, Button: tea.MouseButtonLeft, Y: y}); cmd != nil {
			t.Errorf("click on row %d acted", y)
		}
	}
	if _, cmd := m.Update(selectorKey("x")); cmd != nil {
		t.Error("unlisted key acted")
	}
}

// TestSelectorQuit checks the three quit keys.
//
// Covers: FR-009, FR-010
func TestSelectorQuit(t *testing.T) {
	for _, k := range []string{"q", "esc", "ctrl+c"} {
		m := newSelectorModel(selectorFeatures, false, 80, 24)
		_, cmd := m.Update(selectorKey(k))
		if cmd == nil {
			t.Fatalf("%s: no command", k)
		}
		if _, ok := cmd().(tea.QuitMsg); !ok {
			t.Errorf("%s did not quit", k)
		}
	}
}
