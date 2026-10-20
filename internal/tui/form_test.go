package tui

import (
	"reflect"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// formFeature returns a feature with a choice input and two text inputs, the
// second with a validator that rejects "bad".
func formFeature(short string) Feature {
	return Feature{Name: "feature-1", Short: short, Listed: true, Inputs: []FeatureInput{
		{Name: "input-1", Description: "Description of input 1", Choices: []string{"openai", "anthropic", "ollama"}},
		{Name: "input-2", Description: "Description of input 2", Validator: func(v string) string {
			if v == "bad" {
				return "input-2 is bad"
			}
			return ""
		}},
		{Name: "input-3", Description: "Description of input 3"},
	}}
}

// press sends a key to the form and returns the model and the message its
// command produced, if any.
func press(m formModel, k tea.KeyMsg) (formModel, tea.Msg) {
	m, cmd := m.Update(k)
	if cmd == nil {
		return m, nil
	}
	return m, cmd()
}

// text builds the key message of typed text.
func text(s string) tea.KeyMsg { return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)} }

var (
	keyTab      = tea.KeyMsg{Type: tea.KeyTab}
	keyShiftTab = tea.KeyMsg{Type: tea.KeyShiftTab}
	keyEnter    = tea.KeyMsg{Type: tea.KeyEnter}
	keyEsc      = tea.KeyMsg{Type: tea.KeyEsc}
	keyDown     = tea.KeyMsg{Type: tea.KeyDown}
)

// TestFormView checks the header, the description, the fields with the
// focus marker, the indicator lines and the footer, and EC-008 (no
// description, no blank line in its place).
//
// Covers: FR-011, FR-012, EC-008
func TestFormView(t *testing.T) {
	m := newFormModel(formFeature("Description of feature 1"), nil, false, 80, 30)
	got := m.View()
	for _, want := range []string{
		"aicli › add › feature-1\nDescription of feature 1\n\n> input-1\n  Description of input 1\n  (•) openai\n  ( ) anthropic\n  ( ) ollama\n\n  input-2\n",
		"\n\ntab/enter next · ↑/↓ choose · shift+tab back · esc back · ctrl+c quit",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("view lacks %q:\n%s", want, got)
		}
	}
	m = newFormModel(formFeature(""), nil, false, 80, 30)
	if got := m.View(); !strings.HasPrefix(got, "aicli › add › feature-1\n\n> input-1") {
		t.Errorf("view without description = %q", got[:60])
	}
}

// TestFormFooters checks the per-field footers.
//
// Covers: FR-011
func TestFormFooters(t *testing.T) {
	m := newFormModel(formFeature("d"), nil, false, 80, 30)
	m, _ = press(m, keyTab)
	if !strings.HasSuffix(m.View(), "\ntab/enter next · shift+tab back · esc back · ctrl+c quit") {
		t.Errorf("text footer: %q", m.View())
	}
	m.fields[1].text = m.fields[1].text.SetValue("x")
	m, _ = press(m, keyTab)
	if !strings.HasSuffix(m.View(), "\nenter continue · shift+tab back · esc back · ctrl+c quit") {
		t.Errorf("last footer: %q", m.View())
	}
}

// TestFormNavigation checks that Tab and Enter validate the field they
// leave, that Shift+Tab does not, and that a choice moves on with arrows.
//
// Covers: FR-013, FR-015
func TestFormNavigation(t *testing.T) {
	m := newFormModel(formFeature("d"), nil, false, 80, 30)
	m, _ = press(m, keyDown)
	m, _ = press(m, keyEnter)
	if m.focus != 1 || m.fields[0].value() != "anthropic" {
		t.Fatalf("focus = %d, choice = %q", m.focus, m.fields[0].value())
	}
	m, _ = press(m, keyTab)
	if m.focus != 1 || !strings.Contains(m.View(), "  input-2 cannot be empty") {
		t.Errorf("empty field left: focus %d\n%s", m.focus, m.View())
	}
	m, _ = press(m, text("bad"))
	m, _ = press(m, keyEnter)
	if m.focus != 1 || !strings.Contains(m.View(), "  input-2 is bad") {
		t.Errorf("invalid field left: focus %d\n%s", m.focus, m.View())
	}
	m, _ = press(m, keyShiftTab)
	m, _ = press(m, keyShiftTab)
	if m.focus != 0 {
		t.Errorf("shift+tab: focus %d", m.focus)
	}
	m, _ = press(m, keyTab)
	m, _ = press(m, tea.KeyMsg{Type: tea.KeyBackspace})
	m, _ = press(m, tea.KeyMsg{Type: tea.KeyBackspace})
	m, _ = press(m, tea.KeyMsg{Type: tea.KeyBackspace})
	m, _ = press(m, text("good"))
	m, _ = press(m, keyTab)
	if m.focus != 2 {
		t.Errorf("valid field: focus %d", m.focus)
	}
	m, msg := press(m, keyTab)
	if m.focus != 2 || msg != nil {
		t.Errorf("tab on the last field acted: focus %d, %v", m.focus, msg)
	}
}

// TestFormFinish checks Enter on the last field: invalid fields move the
// focus to the first invalid one; valid ones produce the trimmed values.
//
// Covers: FR-013, FR-015
func TestFormFinish(t *testing.T) {
	m := newFormModel(formFeature("d"), nil, false, 80, 30)
	m = m.moveFocus(2)
	m, msg := press(m, keyEnter)
	if msg != nil || m.focus != 1 {
		t.Fatalf("msg = %v, focus = %d", msg, m.focus)
	}
	v := m.View()
	if !strings.Contains(v, "input-2 cannot be empty") || !strings.Contains(v, "input-3 cannot be empty") {
		t.Errorf("errors missing:\n%s", v)
	}
	m, _ = press(m, text("  two "))
	m = m.moveFocus(2)
	m, _ = press(m, text(" three  "))
	_, msg = press(m, keyEnter)
	if want := (formDoneMsg{Values: []string{"openai", "two", "three"}}); !reflect.DeepEqual(msg, want) {
		t.Errorf("msg = %#v, want %#v", msg, want)
	}
}

// TestFormEscAndQuit checks that Esc keeps the typed values and never asks
// for a run, that Ctrl+C quits, and that reopening shows the values with the
// focus on the first field.
//
// Covers: FR-017, EC-014
func TestFormEscAndQuit(t *testing.T) {
	m := newFormModel(formFeature("d"), nil, false, 80, 30)
	m, _ = press(m, keyDown)
	m, _ = press(m, keyTab)
	m, _ = press(m, text(" half "))
	_, msg := press(m, keyEsc)
	back, ok := msg.(formBackMsg)
	if !ok || !reflect.DeepEqual(back.Values, []string{"anthropic", " half ", ""}) {
		t.Fatalf("esc gave %#v", msg)
	}
	_, msg = press(m, tea.KeyMsg{Type: tea.KeyCtrlC})
	if _, ok := msg.(tea.QuitMsg); !ok {
		t.Errorf("ctrl+c gave %#v", msg)
	}
	again := newFormModel(formFeature("d"), back.Values, false, 80, 30)
	if again.focus != 0 || again.fields[0].value() != "anthropic" || again.fields[1].value() != " half " {
		t.Errorf("reopened: focus %d, values %q", again.focus, again.values(false))
	}
	other := newFormModel(formFeature("d"), nil, false, 80, 30)
	if !reflect.DeepEqual(other.values(false), []string{"openai", "", ""}) {
		t.Errorf("fresh form = %q", other.values(false))
	}
}

// TestFormMouse checks that a click on a field moves the focus without
// validating, that a click on a choice option selects it, and that clicks
// outside the fields do nothing.
//
// Covers: FR-015, EC-011
func TestFormMouse(t *testing.T) {
	click := func(m formModel, y int) formModel {
		m, _ = m.Update(tea.MouseMsg{Action: tea.MouseActionPress, Button: tea.MouseButtonLeft, Y: y})
		return m
	}
	m := newFormModel(formFeature("d"), nil, false, 80, 30)
	// Rows: header 0, description 1, indicator 2, input-1 3-7, blank 8, input-2 9-11.
	m = click(m, 10)
	if m.focus != 1 || strings.Contains(m.View(), "cannot be empty") {
		t.Errorf("click on text field: focus %d", m.focus)
	}
	m = click(m, 5)
	if m.focus != 0 {
		t.Errorf("click back: focus %d", m.focus)
	}
	m = click(m, 7)
	if m.fields[0].value() != "ollama" {
		t.Errorf("option click: %q", m.fields[0].value())
	}
	for _, y := range []int{0, 1, 2, 8, 20, 29} {
		if g := click(m, y); g.focus != m.focus || g.fields[0].value() != "ollama" {
			t.Errorf("click on row %d acted", y)
		}
	}
	wheel, _ := m.Update(tea.MouseMsg{Action: tea.MouseActionPress, Button: tea.MouseButtonWheelDown, Y: 5})
	if wheel.focus != 0 {
		t.Error("wheel acted")
	}
}

// TestFormUnlistedKeys checks that keys outside the lists do nothing in a
// choice field and in a text field.
//
// Covers: EC-011
func TestFormUnlistedKeys(t *testing.T) {
	m := newFormModel(formFeature("d"), nil, false, 80, 30)
	for _, k := range []tea.KeyMsg{text("x"), {Type: tea.KeyCtrlA}, {Type: tea.KeyLeft}, {Type: tea.KeyBackspace}} {
		g, msg := press(m, k)
		if msg != nil || g.focus != 0 || g.fields[0].value() != "openai" {
			t.Errorf("%v acted in the choice field", k)
		}
	}
	m = m.moveFocus(1)
	for _, k := range []tea.KeyMsg{{Type: tea.KeyCtrlA}, {Type: tea.KeyUp}, {Type: tea.KeyPgDown}} {
		g, msg := press(m, k)
		if msg != nil || g.focus != 1 || g.fields[1].value() != "" {
			t.Errorf("%v acted in the text field", k)
		}
	}
}

// TestFormResize checks that a short terminal scrolls the fields keeping the
// focused one visible, and that a narrow one truncates the header.
//
// Covers: FR-011, EC-010
func TestFormResize(t *testing.T) {
	m := newFormModel(formFeature("d"), nil, false, 80, 30)
	m, _ = m.Update(tea.WindowSizeMsg{Width: 14, Height: 11})
	m = m.moveFocus(2)
	got := m.View()
	if !strings.Contains(got, "> input-3") || !strings.Contains(got, "↑ more") || !strings.HasPrefix(got, "aicli › add ›…") {
		t.Errorf("view:\n%s", got)
	}
	if n := strings.Count(got, "\n") + 1; n > 11 {
		t.Errorf("%d lines on 11 rows", n)
	}
}
