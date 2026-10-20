package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	tea "github.com/charmbracelet/bubbletea"
)

// typeText sends text to the field as typed runes.
func typeText(f textField, s string) textField {
	return f.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)})
}

// TestTextFieldView checks the title markers, the description, the input
// line and the absence of ESC in plain mode.
//
// Covers: FR-011
func TestTextFieldView(t *testing.T) {
	f := newTextField(FeatureInput{Name: "input-2", Description: "Description of input 2"}, false, 40)
	got := strings.Join(f.View(), "\n")
	if want := "  input-2\n  Description of input 2\n  "; !strings.HasPrefix(got, want) {
		t.Errorf("unfocused view = %q", got)
	}
	f = f.Focus()
	if got := f.View()[0]; got != "> input-2" {
		t.Errorf("focused title = %q", got)
	}
	if strings.Contains(strings.Join(f.View(), ""), "\x1b") {
		t.Error("plain view has ESC")
	}
}

// TestTextFieldLongValue checks that a value wider than the field scrolls
// with the cursor visible while the full value is kept.
//
// Covers: FR-011, EC-012
func TestTextFieldLongValue(t *testing.T) {
	f := newTextField(FeatureInput{Name: "n"}, false, 20).Focus()
	long := strings.Repeat("abcdefghij", 5) + "END"
	f = typeText(f, long)
	if f.Value() != long {
		t.Fatalf("value = %q", f.Value())
	}
	line := f.View()[2]
	if w := ansi.StringWidth(line); w > 20 {
		t.Errorf("input line is %d cells wide, terminal is 20", w)
	}
	if !strings.Contains(line, "END") {
		t.Errorf("cursor end not visible: %q", line)
	}
	for i := 0; i < len(long); i++ {
		f = f.Update(tea.KeyMsg{Type: tea.KeyLeft})
	}
	if line := f.View()[2]; !strings.Contains(line, "abc") || ansi.StringWidth(line) > 20 {
		t.Errorf("cursor at start not visible: %q", line)
	}
	if f.Value() != long {
		t.Error("value changed by moving the cursor")
	}
}

// TestTextFieldPasteControls checks that a paste with newlines, tabs and DEL
// loses those characters and keeps the rest, and that a paste of controls
// only changes nothing.
//
// Covers: EC-013, EC-011
func TestTextFieldPasteControls(t *testing.T) {
	f := newTextField(FeatureInput{Name: "n"}, false, 40).Focus()
	f = f.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("ab\ncd\r\te\x7ff\x85g"), Paste: true})
	if f.Value() != "abcdefg" {
		t.Errorf("value = %q", f.Value())
	}
	f = f.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("\n\n"), Paste: true})
	if f.Value() != "abcdefg" || f.err != "" {
		t.Errorf("controls-only paste changed the field: %q", f.Value())
	}
}

// TestTextFieldEmptyError checks that a blank value gives `<input> cannot be
// empty` and shows it under the field without calling the validator.
//
// Covers: FR-013, FR-014
func TestTextFieldEmptyError(t *testing.T) {
	called := false
	f := newTextField(FeatureInput{Name: "name", Validator: func(string) string { called = true; return "bad" }}, false, 40).Focus()
	f = typeText(f, "   ")
	f, msg := f.Validate()
	if msg != "name cannot be empty" || called {
		t.Errorf("msg = %q, validator called = %v", msg, called)
	}
	if v := f.View(); len(v) != 4 || v[3] != "  name cannot be empty" {
		t.Errorf("view = %q", v)
	}
}

// TestTextFieldValidator checks that the validator gets the trimmed value,
// that its message is shown, and that a valid value clears the error.
//
// Covers: FR-013, FR-014
func TestTextFieldValidator(t *testing.T) {
	var got string
	f := newTextField(FeatureInput{Name: "name", Validator: func(v string) string {
		got = v
		if v == "ok" {
			return ""
		}
		return "must be ok"
	}}, false, 40).Focus()
	f = typeText(f, "  no ")
	f, msg := f.Validate()
	if msg != "must be ok" || got != "no" || f.View()[3] != "  must be ok" {
		t.Errorf("msg = %q, validator saw %q, view = %q", msg, got, f.View())
	}
	if f.Value() != "  no " {
		t.Errorf("value changed: %q", f.Value())
	}
	f = f.SetValue(" ok ")
	f, msg = f.Validate()
	if msg != "" || len(f.View()) != 3 {
		t.Errorf("msg = %q, view = %q", msg, f.View())
	}
}

// TestTextFieldUnlistedKeys checks that keys outside the listed ones do
// nothing, while editing keys work.
//
// Covers: EC-011
func TestTextFieldUnlistedKeys(t *testing.T) {
	f := typeText(newTextField(FeatureInput{Name: "n"}, false, 40).Focus(), "abc")
	for _, k := range []tea.KeyMsg{
		{Type: tea.KeyCtrlA}, {Type: tea.KeyCtrlE}, {Type: tea.KeyCtrlW}, {Type: tea.KeyCtrlK},
		{Type: tea.KeyCtrlU}, {Type: tea.KeyUp}, {Type: tea.KeyDown}, {Type: tea.KeyTab},
		{Type: tea.KeyEnter}, {Type: tea.KeyEsc}, {Type: tea.KeyPgUp},
		{Type: tea.KeyLeft, Alt: true}, {Type: tea.KeyRunes, Runes: []rune("x"), Alt: true},
	} {
		if g := f.Update(k); g.Value() != "abc" || g.ti.Position() != 3 {
			t.Errorf("%v acted: %q at %d", k, g.Value(), g.ti.Position())
		}
	}
	f = f.Update(tea.KeyMsg{Type: tea.KeyLeft}).Update(tea.KeyMsg{Type: tea.KeyBackspace})
	f = f.Update(tea.KeyMsg{Type: tea.KeyHome}).Update(tea.KeyMsg{Type: tea.KeyDelete})
	if f.Value() != "c" {
		t.Errorf("value = %q", f.Value())
	}
	f = typeText(f, "x y")
	if f.Value() != "x yc" {
		t.Errorf("value = %q", f.Value())
	}
}
