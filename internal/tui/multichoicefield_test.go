package tui

import (
	"reflect"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// multiInput is the input used by the multi-choice field tests.
var multiInput = FeatureInput{Name: "provider", Description: "LLM providers to add", Choices: []string{"openai", "claude", "gemini", "ollama"}}

// spaceKey is the Space key message.
var spaceKey = tea.KeyMsg{Type: tea.KeySpace, Runes: []rune(" ")}

// TestMultiChoiceFieldView checks the focused view with markers, cursor and
// already added choices, the unfocused view, and plain vs styled output.
//
// Covers: FR-006, FR-007, EC-005
func TestMultiChoiceFieldView(t *testing.T) {
	f := newMultiChoiceField(multiInput, []bool{true, false, false, false}, false, 40).Focus()
	want := []string{
		"> provider", "  LLM providers to add",
		"    [✓] openai (already added)", "  > [ ] claude", "    [ ] gemini", "    [ ] ollama",
	}
	if got := f.View(10); !reflect.DeepEqual(got, want) || f.Height() != 6 {
		t.Errorf("focused = %q, height %d", got, f.Height())
	}
	f = f.Update(spaceKey).Update(key("j")).Update(spaceKey)
	if got := f.View(10)[3:]; !reflect.DeepEqual(got, []string{"    [x] claude", "  > [x] gemini", "    [ ] ollama"}) {
		t.Errorf("selected = %q", got)
	}
	f = f.Blur()
	want = []string{"  provider", "  LLM providers to add", "  claude, gemini"}
	if got := f.View(10); !reflect.DeepEqual(got, want) || f.Height() != 3 {
		t.Errorf("unfocused = %q, height %d", got, f.Height())
	}
	if got := newMultiChoiceField(multiInput, nil, false, 40).View(10)[2]; got != "" {
		t.Errorf("unfocused with nothing = %q", got)
	}
	if strings.Contains(strings.Join(f.Focus().View(10), ""), "\x1b") {
		t.Error("plain view has ESC")
	}
	if s := newMultiChoiceField(multiInput, nil, true, 40).Focus(); !strings.Contains(strings.Join(s.View(10), ""), "\x1b") {
		t.Error("styled view has no ESC")
	}
}

// TestMultiChoiceFieldKeys checks cursor movement that skips already added
// lines and stops at the ends, and that other keys do nothing.
//
// Covers: FR-006, FR-007
func TestMultiChoiceFieldKeys(t *testing.T) {
	f := newMultiChoiceField(multiInput, []bool{true, false, true, false}, false, 40).Focus()
	if f.cursor != 1 {
		t.Fatalf("start cursor = %d", f.cursor)
	}
	f = f.Update(tea.KeyMsg{Type: tea.KeyUp}).Update(key("k"))
	if f.cursor != 1 {
		t.Errorf("moved above the first = %d", f.cursor)
	}
	f = f.Update(tea.KeyMsg{Type: tea.KeyDown})
	if f.cursor != 3 {
		t.Errorf("down skipped = %d", f.cursor)
	}
	f = f.Update(key("j")).Update(tea.KeyMsg{Type: tea.KeyDown})
	if f.cursor != 3 {
		t.Errorf("past the last = %d", f.cursor)
	}
	if g := f.Update(key("k")); g.cursor != 1 {
		t.Errorf("up skipped = %d", g.cursor)
	}
	for _, k := range []tea.KeyMsg{key("x"), {Type: tea.KeyLeft}, {Type: tea.KeyEnter}} {
		if g := f.Update(k); !reflect.DeepEqual(g, f) {
			t.Errorf("%v acted", k)
		}
	}
}

// TestMultiChoiceFieldToggle checks that Space selects and deselects, and
// that the value keeps the declaration order whatever the selection order.
//
// Covers: FR-006, EC-015
func TestMultiChoiceFieldToggle(t *testing.T) {
	f := newMultiChoiceField(multiInput, nil, false, 40).Focus()
	if len(f.Value()) != 0 {
		t.Fatalf("start = %q", f.Value())
	}
	f = f.Update(key("j")).Update(key("j")).Update(key("j")).Update(spaceKey) // ollama
	f = f.Update(key("k")).Update(key("k")).Update(key("k")).Update(spaceKey) // openai
	f = f.Update(key("j")).Update(key("j")).Update(spaceKey)                  // gemini
	if got := f.Value(); !reflect.DeepEqual(got, []string{"openai", "gemini", "ollama"}) {
		t.Errorf("value = %q", got)
	}
	f = f.Update(spaceKey)
	if got := f.Value(); !reflect.DeepEqual(got, []string{"openai", "ollama"}) {
		t.Errorf("after deselecting = %q", got)
	}
}

// TestMultiChoiceFieldClick checks that a click moves the cursor and
// toggles, and does nothing on an already added line, the title, the
// description or past the last choice.
//
// Covers: FR-006, FR-007
func TestMultiChoiceFieldClick(t *testing.T) {
	f := newMultiChoiceField(multiInput, []bool{true, false, false, false}, false, 40).Focus()
	f = f.Click(choiceFirstRow+2, 10)
	if f.cursor != 2 || !reflect.DeepEqual(f.Value(), []string{"gemini"}) {
		t.Errorf("click = cursor %d, value %q", f.cursor, f.Value())
	}
	f = f.Click(choiceFirstRow+2, 10)
	if len(f.Value()) != 0 {
		t.Errorf("second click did not deselect: %q", f.Value())
	}
	for _, row := range []int{choiceFirstRow, 0, 1, choiceFirstRow + 4, -1} {
		if g := f.Click(row, 10); !reflect.DeepEqual(g, f) {
			t.Errorf("click on row %d acted", row)
		}
	}
}

// TestMultiChoiceFieldAllAdded checks that, with every choice already
// added, there is no cursor and nothing can be selected.
//
// Covers: FR-006, FR-007, EC-005
func TestMultiChoiceFieldAllAdded(t *testing.T) {
	f := newMultiChoiceField(multiInput, []bool{true, true, true, true}, false, 40).Focus()
	if !f.AllAdded() || f.cursor != -1 {
		t.Fatalf("AllAdded %v, cursor %d", f.AllAdded(), f.cursor)
	}
	f = f.Update(key("j")).Update(spaceKey).Click(choiceFirstRow, 10)
	if len(f.Value()) != 0 {
		t.Errorf("selected = %q", f.Value())
	}
	for _, l := range f.View(10)[choiceFirstRow:] {
		if !strings.HasPrefix(l, "    [✓] ") || !strings.HasSuffix(l, "(already added)") {
			t.Errorf("line = %q", l)
		}
	}
}

// TestMultiChoiceFieldScroll checks a list taller than the space: the
// choices scroll inside the field, the cursor stays visible and a click
// maps to the shown choice.
//
// Covers: EC-015, FR-006
func TestMultiChoiceFieldScroll(t *testing.T) {
	in := FeatureInput{Name: "n", Description: "d", Choices: []string{"a", "b", "c", "d", "e"}}
	f := newMultiChoiceField(in, nil, false, 40).Focus()
	tail := func(f multiChoiceField, space int) []string { return f.View(space)[choiceFirstRow:] }
	if got := tail(f, 4); !reflect.DeepEqual(got, []string{"  > [ ] a", "    [ ] b"}) {
		t.Errorf("top = %q", got)
	}
	for i := 0; i < 4; i++ {
		f = f.Update(key("j"))
	}
	if got := tail(f, 4); !reflect.DeepEqual(got, []string{"    [ ] d", "  > [ ] e"}) {
		t.Errorf("bottom = %q", got)
	}
	if got := tail(f, 0); !reflect.DeepEqual(got, []string{"  > [ ] e"}) {
		t.Errorf("space 0 = %q", got)
	}
	f = f.Click(choiceFirstRow, 4)
	if !reflect.DeepEqual(f.Value(), []string{"d"}) || f.cursor != 3 {
		t.Errorf("click = %q cursor %d", f.Value(), f.cursor)
	}
}
