package tui

import (
	"reflect"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// choiceInput is the input used by the choice field tests.
var choiceInput = FeatureInput{Name: "input-1", Description: "Description of input 1", Choices: []string{"openai", "anthropic", "ollama"}}

// TestChoiceFieldView checks the focused view (every choice with its
// marker), the unfocused view (only the selected one) and the plain output.
//
// Covers: FR-016
func TestChoiceFieldView(t *testing.T) {
	f := newChoiceField(choiceInput, false, 40).Focus()
	want := []string{"> input-1", "  Description of input 1", "  (•) openai", "  ( ) anthropic", "  ( ) ollama"}
	if got := f.View(10); !reflect.DeepEqual(got, want) || f.Height() != 5 {
		t.Errorf("focused = %q, height %d", got, f.Height())
	}
	f = f.Blur()
	want = []string{"  input-1", "  Description of input 1", "  (•) openai"}
	if got := f.View(10); !reflect.DeepEqual(got, want) || f.Height() != 3 {
		t.Errorf("unfocused = %q, height %d", got, f.Height())
	}
	if strings.Contains(strings.Join(f.View(10), ""), "\x1b") {
		t.Error("plain view has ESC")
	}
	if styled := newChoiceField(choiceInput, true, 40).Focus(); !strings.Contains(strings.Join(styled.View(10), ""), "\x1b") {
		t.Error("styled view has no ESC")
	}
}

// TestChoiceFieldKeys checks that the four movement keys move the selection
// and stop at the ends, and that other keys do nothing.
//
// Covers: FR-016, EC-011
func TestChoiceFieldKeys(t *testing.T) {
	f := newChoiceField(choiceInput, false, 40).Focus()
	if f.Value() != "openai" {
		t.Fatalf("start = %q", f.Value())
	}
	f = f.Update(tea.KeyMsg{Type: tea.KeyUp}).Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
	if f.Value() != "openai" {
		t.Errorf("moved above the first: %q", f.Value())
	}
	f = f.Update(tea.KeyMsg{Type: tea.KeyDown})
	if f.Value() != "anthropic" {
		t.Errorf("down = %q", f.Value())
	}
	f = f.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")}).Update(tea.KeyMsg{Type: tea.KeyDown})
	if f.Value() != "ollama" {
		t.Errorf("past the last = %q", f.Value())
	}
	for _, k := range []tea.KeyMsg{{Type: tea.KeyRunes, Runes: []rune("x")}, {Type: tea.KeyLeft}, {Type: tea.KeyHome}, {Type: tea.KeyEnter}} {
		if g := f.Update(k); g.Value() != "ollama" {
			t.Errorf("%v acted: %q", k, g.Value())
		}
	}
	if g := f.SetValue("anthropic"); g.Value() != "anthropic" || g.SetValue("none").Value() != "anthropic" {
		t.Error("SetValue")
	}
}

// TestChoiceFieldScroll checks a list taller than the space: the choices
// scroll inside the field and the selection stays visible.
//
// Covers: EC-010, FR-016
func TestChoiceFieldScroll(t *testing.T) {
	in := FeatureInput{Name: "n", Description: "d", Choices: []string{"a", "b", "c", "d", "e"}}
	f := newChoiceField(in, false, 40).Focus()
	tail := func(f choiceField, space int) []string { return f.View(space)[choiceFirstRow:] }
	if got := tail(f, 4); !reflect.DeepEqual(got, []string{"  (•) a", "  ( ) b"}) {
		t.Errorf("top = %q", got)
	}
	for i := 0; i < 4; i++ {
		f = f.Update(tea.KeyMsg{Type: tea.KeyDown})
	}
	if got := tail(f, 4); !reflect.DeepEqual(got, []string{"  ( ) d", "  (•) e"}) {
		t.Errorf("bottom = %q", got)
	}
	if got := f.View(1); len(got) != 3 || got[2] != "  (•) e" {
		t.Errorf("no room = %q", got)
	}
	if got := f.View(0); len(got) != 3 {
		t.Errorf("space 0 = %q", got)
	}
}

// TestChoiceFieldClick checks that a click on an option selects it, also in
// a scrolled list, and that clicks elsewhere change nothing.
//
// Covers: FR-015, FR-016
func TestChoiceFieldClick(t *testing.T) {
	f := newChoiceField(choiceInput, false, 40).Focus()
	if g := f.Click(4, 10); g.Value() != "ollama" {
		t.Errorf("click = %q", g.Value())
	}
	for _, row := range []int{0, 1, 5, -1} {
		if g := f.Click(row, 10); g.Value() != "openai" {
			t.Errorf("row %d changed to %q", row, g.Value())
		}
	}
	in := FeatureInput{Name: "n", Description: "d", Choices: []string{"a", "b", "c", "d", "e"}}
	s := newChoiceField(in, false, 40).Focus()
	for i := 0; i < 4; i++ {
		s = s.Update(tea.KeyMsg{Type: tea.KeyDown})
	}
	if g := s.Click(2, 4); g.Value() != "d" {
		t.Errorf("scrolled click = %q", g.Value())
	}
}

// TestChoiceFieldResize checks that a narrower terminal truncates the lines
// with `…` instead of wrapping them.
//
// Covers: EC-010
func TestChoiceFieldResize(t *testing.T) {
	f := newChoiceField(choiceInput, false, 40).Focus().resize(8)
	for _, l := range f.View(10) {
		if len([]rune(l)) > 8 {
			t.Errorf("line %q wider than 8", l)
		}
	}
	if got := f.View(10)[3]; got != "  ( ) a…" {
		t.Errorf("truncated = %q", got)
	}
}
