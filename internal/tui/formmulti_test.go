package tui

import (
	"reflect"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// multiFeature returns a feature whose only input is a multi-choice input;
// added marks which of its four choices are already added.
func multiFeature(added []bool) Feature {
	return Feature{Name: "feature-1", Short: "Add things.", Listed: true, Inputs: []FeatureInput{
		{Name: "provider", Description: "LLM providers to add", Choices: []string{"openai", "claude", "gemini", "ollama"},
			Multi: true, ItemNoun: "providers", Added: added},
	}}
}

// multiFeatureWithText returns a feature with a text input followed by a
// multi-choice input.
func multiFeatureWithText() Feature {
	f := multiFeature(nil)
	f.Inputs = append([]FeatureInput{{Name: "name", Description: "Name"}}, f.Inputs...)
	return f
}

// spacePress is the Space key as the form receives it.
var spacePress = tea.KeyMsg{Type: tea.KeySpace, Runes: []rune(" ")}

// TestFormMultiValues checks that the multi-choice value reaches the
// confirmation as one value per selected choice, in declaration order, and
// that the kept values restore the selection.
//
// Covers: FR-006
func TestFormMultiValues(t *testing.T) {
	m := newFormModel(multiFeatureWithText(), nil, false, 80, 30)
	m, _ = press(m, text("app"))
	m, _ = press(m, keyEnter)
	for _, k := range []tea.KeyMsg{text("j"), text("j"), text("j"), spacePress, text("k"), text("k"), text("k"), spacePress} {
		m, _ = press(m, k)
	}
	_, msg := press(m, keyEnter)
	if got := msg.(formDoneMsg).Values; !reflect.DeepEqual(got, []string{"app", "openai", "ollama"}) {
		t.Errorf("done values = %q", got)
	}
	kept := m.values(false)
	again := newFormModel(multiFeatureWithText(), kept, false, 80, 30)
	if got := again.values(false); !reflect.DeepEqual(got, kept) {
		t.Errorf("restored = %q, want %q", got, kept)
	}
}

// TestFormMultiValidation checks that only Enter validates the multi-choice
// field, and that Shift+Tab and a click leave it without validating.
//
// Covers: FR-006
func TestFormMultiValidation(t *testing.T) {
	m := newFormModel(multiFeatureWithText(), nil, false, 80, 30)
	m, _ = press(m, text("app"))
	m, _ = press(m, keyEnter)
	if strings.Contains(m.View(), "select at least one") {
		t.Error("error shown before Enter")
	}
	m, msg := press(m, keyEnter)
	if msg != nil || !strings.Contains(m.View(), "  select at least one provider") {
		t.Errorf("Enter with nothing: msg %v\n%s", msg, m.View())
	}
	m, _ = press(m, spacePress)
	if _, msg = press(m, keyEnter); msg == nil {
		t.Error("Enter with a selection did not finish")
	}
	m, _ = press(m, spacePress) // deselect again
	m, _ = press(m, keyEnter)
	m, _ = press(m, keyShiftTab)
	if m.focus != 0 {
		t.Errorf("focus after Shift+Tab = %d", m.focus)
	}
	fresh := newFormModel(multiFeatureWithText(), nil, false, 80, 30)
	fresh, _ = press(fresh, keyShiftTab)
	fresh, _ = press(fresh, keyTab)
	if strings.Contains(fresh.View(), "select at least one") {
		t.Error("Tab validated")
	}
	l := fresh.layout()
	click := tea.MouseMsg{Action: tea.MouseActionPress, Button: tea.MouseButtonLeft, X: 3, Y: l.AreaTop}
	fresh = fresh.updateMouse(click)
	if fresh.focus != 0 || strings.Contains(fresh.View(), "select at least one") {
		t.Errorf("click on the text field: focus %d\n%s", fresh.focus, fresh.View())
	}
}

// TestFormMultiAllAdded checks the all-added form: the message in place of
// the error, no cursor, Enter doing nothing and not in the footer.
//
// Covers: EC-006, FR-007
func TestFormMultiAllAdded(t *testing.T) {
	m := newFormModel(multiFeature([]bool{true, true, true, true}), nil, false, 80, 30)
	v := m.View()
	for _, want := range []string{"    [✓] openai (already added)", "  All providers are already added.\n", "\nesc back · ctrl+c quit"} {
		if !strings.Contains(v, want) {
			t.Errorf("view lacks %q:\n%s", want, v)
		}
	}
	if strings.Contains(v, "> [") || strings.Contains(m.footer(), "enter") {
		t.Errorf("cursor or enter shown:\n%s", v)
	}
	after, msg := press(m, keyEnter)
	if msg != nil || strings.Contains(after.View(), "select at least one") {
		t.Errorf("Enter acted: %v\n%s", msg, after.View())
	}
	m.direct = true
	if !strings.HasSuffix(m.footer(), "esc quit · ctrl+c quit") {
		t.Errorf("direct footer = %q", m.footer())
	}
}

// TestFormMultiFooter checks that the footer lists `space toggle`, and the
// Esc word and behaviour in direct mode and from the selector.
//
// Covers: FR-006, FR-005, EC-016
func TestFormMultiFooter(t *testing.T) {
	m := newFormModel(multiFeature(nil), nil, false, 80, 30)
	if got := m.footer(); got != "↑/↓ move · space toggle · enter continue · esc back · ctrl+c quit" {
		t.Errorf("footer = %q", got)
	}
	if _, msg := press(m, keyEsc); !isFormBack(msg) {
		t.Errorf("Esc from the selector produced %T", msg)
	}
	m.direct = true
	if got := m.footer(); got != "↑/↓ move · space toggle · enter continue · esc quit · ctrl+c quit" {
		t.Errorf("direct footer = %q", got)
	}
	_, cmd := m.Update(keyEsc)
	if cmd == nil {
		t.Fatal("Esc in direct mode produced no command")
	}
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Error("Esc in direct mode did not quit")
	}
	two := newFormModel(multiFeatureWithText(), nil, false, 80, 30)
	two = two.moveFocus(1)
	if got := two.footer(); got != "↑/↓ move · space toggle · enter continue · shift+tab back · esc back · ctrl+c quit" {
		t.Errorf("two-field footer = %q", got)
	}
}

// TestFormMultiClick checks that a click on a checklist line moves the
// cursor and toggles it, and does nothing on an already added line.
//
// Covers: FR-006, FR-007
func TestFormMultiClick(t *testing.T) {
	m := newFormModel(multiFeature([]bool{true, false, false, false}), nil, false, 80, 30)
	top := m.layout().AreaTop
	click := func(m formModel, row int) formModel {
		return m.updateMouse(tea.MouseMsg{Action: tea.MouseActionPress, Button: tea.MouseButtonLeft, X: 5, Y: top + row})
	}
	m = click(m, choiceFirstRow+2)
	if got := m.values(true); !reflect.DeepEqual(got, []string{"gemini"}) {
		t.Errorf("click = %q", got)
	}
	m = click(m, choiceFirstRow)
	if got := m.values(true); !reflect.DeepEqual(got, []string{"gemini"}) {
		t.Errorf("click on added = %q", got)
	}
}

// isFormBack reports whether a message is a formBackMsg.
//
// In: msg, any message. Out: true if it is a formBackMsg. Side effects:
// none.
func isFormBack(msg tea.Msg) bool {
	_, ok := msg.(formBackMsg)
	return ok
}
