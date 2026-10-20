package tui_test

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/ro-uc-ru/aicli/internal/tui"
)

// styleFeatures are the features of the styling flows: one with a choice and
// two text inputs, and one with no inputs.
func styleFeatures() []tui.Feature {
	other := confirmFeature(nil, nil, true)
	other.Name = "feature-2"
	return []tui.Feature{flowForm(), other}
}

// styleView runs msgs from the selector, then quits with Ctrl+C, with styling
// on or off.
//
// In: t, the test; styled; msgs. Out: the last screen. Side effects: as
// driveAdd.
func styleView(t *testing.T, styled bool, msgs ...tea.Msg) string {
	t.Helper()
	v, _ := driveAddStyled(t, styleFeatures(), styled, termW, termH, append(msgs, key("ctrl+c"))...)
	return v
}

// lineSig returns the ANSI sequences of the first line containing text,
// joined, i.e. the style that line is drawn with.
//
// In: t, the test; v, a styled screen; text. Out: the signature; the test
// fails if no line contains text. Side effects: none.
func lineSig(t *testing.T, v, text string) string {
	t.Helper()
	for _, l := range strings.Split(v, "\n") {
		if strings.Contains(sgrRE.ReplaceAllString(l, ""), text) {
			return strings.Join(sgrRE.FindAllString(l, -1), "")
		}
	}
	t.Fatalf("no line with %q in:\n%q", text, v)
	return ""
}

// TestAddUnstyledMarks checks that with styling off the focus is marked by
// `> `, `(•) ` and `[ … ]` and that no colour sequence is emitted anywhere.
//
// Covers: FR-009, FR-011, FR-016, FR-018
func TestAddUnstyledMarks(t *testing.T) {
	scenes := map[string][]tea.Msg{
		"selector": nil,
		"form":     keys("enter"),
		"choice":   keys("enter", "down"),
		"confirm":  keys("down", "enter"),
		"focused":  keys("down", "enter", "tab"),
	}
	for name, msgs := range scenes {
		t.Run(name, func(t *testing.T) {
			if v := styleView(t, false, msgs...); strings.Contains(v, "\x1b") {
				t.Errorf("unstyled screen has escape sequences: %q", v)
			}
		})
	}
	if v := styleView(t, false); !strings.HasPrefix(pointerLine(v), "> feature-1") {
		t.Errorf("the selector must mark the entry with `> `:\n%s", v)
	}
	if v := styleView(t, false, keys("enter", "down")...); !strings.Contains(v, "(•) anthropic") || !strings.Contains(v, "> input-1") {
		t.Errorf("the form must mark the focus and the choice:\n%s", v)
	}
	if v := styleView(t, false, keys("down", "enter")...); !strings.Contains(v, "[ Cancel ]") || !strings.Contains(v, "  Add  ") {
		t.Errorf("the confirmation must mark the focused button:\n%s", v)
	}
	if v := styleView(t, false, keys("down", "enter", "tab")...); !strings.Contains(v, "[ Add ]") {
		t.Errorf("the confirmation must follow the focus:\n%s", v)
	}
}

// TestAddStyledOnlyAddsANSI checks that, stripped of ANSI sequences, the
// styled screens equal the unstyled ones.
//
// Covers: FR-009, FR-011, FR-016, FR-018
func TestAddStyledOnlyAddsANSI(t *testing.T) {
	for name, msgs := range map[string][]tea.Msg{
		"selector": nil,
		"form":     keys("enter", "down"),
		"confirm":  keys("down", "enter", "tab"),
	} {
		t.Run(name, func(t *testing.T) {
			styled := styleView(t, true, msgs...)
			if !sgrRE.MatchString(styled) {
				t.Fatalf("styled screen has no ANSI sequence: %q", styled)
			}
			if got, want := sgrRE.ReplaceAllString(styled, ""), styleView(t, false, msgs...); got != want {
				t.Errorf("stripped styled screen differs:\n%s\n--- want:\n%s", got, want)
			}
		})
	}
}

// TestAddStyledSelector checks that the highlighted entry has a different
// style from the others and that the highlight follows the pointer.
//
// Covers: FR-009
func TestAddStyledSelector(t *testing.T) {
	first := styleView(t, true)
	if lineSig(t, first, "feature-1") == lineSig(t, first, "feature-2") {
		t.Error("the selected entry has the same style as the others")
	}
	moved := styleView(t, true, key("down"))
	if lineSig(t, moved, "feature-2") != lineSig(t, first, "feature-1") || lineSig(t, moved, "feature-1") != lineSig(t, first, "feature-2") {
		t.Error("the highlight did not follow the pointer")
	}
}

// TestAddStyledFormFocus checks that the focused title is highlighted and
// that the highlight moves with the focus.
//
// Covers: FR-011, FR-016
func TestAddStyledFormFocus(t *testing.T) {
	first := styleView(t, true, key("enter"))
	if lineSig(t, first, "input-1") == lineSig(t, first, "input-2") {
		t.Error("the focused title has the same style as the others")
	}
	next := styleView(t, true, key("enter"), key("tab"))
	if lineSig(t, next, "input-2") != lineSig(t, first, "input-1") || lineSig(t, next, "input-1") != lineSig(t, first, "input-2") {
		t.Error("the highlight did not move with the focus")
	}
	if lineSig(t, first, "(•) openai") == lineSig(t, first, "( ) ollama") {
		t.Error("the selected choice has the same style as the others")
	}
}

// TestAddStyledButtons checks that `Add` and `Cancel` are drawn in two
// different colours whether focused or not, and that the focused one differs
// from the unfocused one (bold).
//
// Covers: FR-018
func TestAddStyledButtons(t *testing.T) {
	cancelFocused := styleView(t, true, keys("down", "enter")...)
	addFocused := styleView(t, true, keys("down", "enter", "tab")...)
	var sigs [4]string
	for i, c := range []struct{ v, label string }{
		{cancelFocused, "Add"}, {cancelFocused, "Cancel"}, {addFocused, "Add"}, {addFocused, "Cancel"},
	} {
		for _, l := range strings.Split(c.v, "\n") {
			if strings.Contains(sgrRE.ReplaceAllString(l, ""), "Add") && strings.Contains(sgrRE.ReplaceAllString(l, ""), "Cancel") {
				sigs[i] = buttonSig(l, c.label)
			}
		}
		if sigs[i] == "" {
			t.Fatalf("button %q not found in %q", c.label, c.v)
		}
	}
	addOff, cancelOn, addOn, cancelOff := sigs[0], sigs[1], sigs[2], sigs[3]
	if addOff == cancelOff || addOn == cancelOn {
		t.Errorf("Add and Cancel must use different colours: %q %q / %q %q", addOff, cancelOff, addOn, cancelOn)
	}
	if addOff == addOn || cancelOff == cancelOn {
		t.Errorf("the focused button must be drawn differently: %q %q / %q %q", addOff, addOn, cancelOff, cancelOn)
	}
}

// buttonSig returns the ANSI sequences that immediately precede a button label
// in a line holding both buttons.
//
// In: line, the styled line; label, `Add` or `Cancel`. Out: the sequences
// that style the label's own text. Side effects: none.
func buttonSig(line, label string) string {
	i := strings.Index(line, label)
	j := strings.LastIndex(line[:i], "\x1b[")
	start := j
	for start > 0 && strings.HasSuffix(line[:start], "m") {
		k := strings.LastIndex(line[:start], "\x1b[")
		if k < 0 {
			break
		}
		start = k
	}
	if start < 0 {
		return ""
	}
	return strings.Join(sgrRE.FindAllString(line[start:i], -1), "")
}
