package tui_test

import (
	"context"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/ro-uc-ru/aicli/internal/tui"
)

// formCalls counts the calls to the Plan of the form flow feature.
var formCalls atomic.Int32

// flowForm returns a feature with a choice input, a text input with a
// validator that rejects "bad", and a plain text input.
//
// In: none. Out: the feature, whose Plan counts its calls in formCalls.
// Side effects: none.
func flowForm() tui.Feature {
	return tui.Feature{Name: "feature-1", Short: "Description of feature 1", Listed: true, Inputs: []tui.FeatureInput{
		{Name: "input-1", Description: "Description of input 1", Choices: []string{"openai", "anthropic", "ollama"}},
		{Name: "input-2", Description: "Description of input 2", Validator: func(v string) string {
			if v == "bad" {
				return "input-2 is bad"
			}
			return ""
		}},
		{Name: "input-3", Description: "Description of input 3"},
	}, Plan: func(v []string) ([]string, error) {
		formCalls.Add(1)
		return []string{"features/" + v[1] + ".txt"}, nil
	}, Command: func(v []string) string { return "aicli add feature-1 " + strings.Join(v, " ") }}
}

// typed returns the message of typed text.
//
// In: s, the text. Out: the message. Side effects: none.
func typed(s string) tea.Msg { return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)} }

// formDrive runs the add model with the form feature, opens the form with
// Enter, sends msgs and quits with Ctrl+C.
//
// In: t, the test; w and h, the size; msgs. Out: the last screen and result.
// Side effects: as driveAdd.
func formDrive(t *testing.T, w, h int, msgs ...tea.Msg) (string, tui.AddResult) {
	t.Helper()
	return driveAddWith(t, []tui.Feature{flowForm()}, w, h, append(append([]tea.Msg{key("enter")}, msgs...), key("ctrl+c"))...)
}

// formView is formDrive returning only the screen.
//
// In: as formDrive. Out: the last screen. Side effects: as driveAdd.
func formView(t *testing.T, w, h int, msgs ...tea.Msg) string {
	t.Helper()
	v, _ := formDrive(t, w, h, msgs...)
	return v
}

// TestAddFormFields checks the header, the description, and the fields in
// declaration order with names and descriptions, the focus starting on the
// first one.
//
// Covers: FR-011
func TestAddFormFields(t *testing.T) {
	v := formView(t, termW, termH)
	lines := strings.Split(v, "\n")
	if lines[0] != "aicli › add › feature-1" || lines[1] != "Description of feature 1" {
		t.Errorf("wrong header:\n%s", v)
	}
	prev := -1
	for _, n := range []string{"input-1", "Description of input 1", "input-2", "Description of input 2", "input-3", "Description of input 3"} {
		row := rowOf(t, v, n)
		if row <= prev {
			t.Errorf("%q out of order:\n%s", n, v)
		}
		prev = row
	}
	if pointerLine(v) != "> input-1" {
		t.Errorf("focus not on the first field:\n%s", v)
	}
	if !strings.Contains(v, "(•) openai") || !strings.Contains(v, "( ) ollama") {
		t.Errorf("focused choice field not expanded:\n%s", v)
	}
}

// TestAddFormNavigation checks Tab, Enter and Shift+Tab, validation on leaving
// a field and the error persistence, and that Shift+Tab does not validate.
//
// Covers: FR-013, FR-014, FR-015
func TestAddFormNavigation(t *testing.T) {
	for _, k := range []string{"tab", "enter"} {
		if got := pointerLine(formView(t, termW, termH, key(k))); got != "> input-2" {
			t.Errorf("%s: focus = %q, want input-2", k, got)
		}
	}
	if got := pointerLine(formView(t, termW, termH, key("tab"), key("shift+tab"))); got != "> input-1" {
		t.Errorf("shift+tab: focus = %q", got)
	}
	v := formView(t, termW, termH, key("tab"), key("shift+tab"), key("tab"), key("shift+tab"))
	if strings.Contains(v, "cannot be empty") {
		t.Errorf("shift+tab validated:\n%s", v)
	}
	v = formView(t, termW, termH, key("tab"), key("tab"))
	if pointerLine(v) != "> input-2" || !strings.Contains(v, "input-2 cannot be empty") {
		t.Errorf("an empty value must stay on the field with its error:\n%s", v)
	}
	v = formView(t, termW, termH, key("tab"), typed("   "), key("enter"))
	if pointerLine(v) != "> input-2" || !strings.Contains(v, "input-2 cannot be empty") {
		t.Errorf("a blank value must be empty:\n%s", v)
	}
	v = formView(t, termW, termH, key("tab"), typed("bad"), key("tab"))
	if pointerLine(v) != "> input-2" || !strings.Contains(v, "input-2 is bad") || strings.Contains(v, "cannot be empty") {
		t.Errorf("the validator error must show under the field:\n%s", v)
	}
	if got := pointerLine(formView(t, termW, termH, key("tab"), typed(" good "), key("tab"))); got != "> input-3" {
		t.Errorf("a valid value must move on: %q", got)
	}
	// Tab on the last field does nothing; Enter validates every field.
	v = formView(t, termW, termH, key("tab"), typed("ok"), key("tab"), key("tab"))
	if pointerLine(v) != "> input-3" {
		t.Errorf("tab on the last field moved:\n%s", v)
	}
	v = formView(t, termW, termH, key("tab"), typed("ok"), key("tab"), key("enter"))
	if pointerLine(v) != "> input-3" || !strings.Contains(v, "input-3 cannot be empty") {
		t.Errorf("enter on the last field must stay on the invalid field:\n%s", v)
	}
	v = formView(t, termW, termH, key("tab"), typed("ok"), key("tab"), typed("x"), key("enter"))
	if !strings.Contains(v, "Will create:") || !strings.Contains(v, "features/ok.txt") || !strings.Contains(v, "aicli add feature-1 openai ok x") {
		t.Errorf("a valid form must show the confirmation:\n%s", v)
	}
}

// TestAddFormFirstInvalid checks that Enter on the last field moves the focus
// to the first invalid field.
//
// Covers: FR-015
func TestAddFormFirstInvalid(t *testing.T) {
	// Fill input-2 with a bad value by going back after leaving it blank-free.
	v := formView(t, termW, termH,
		key("tab"), typed("x"), key("tab"), typed("z"),
		key("shift+tab"), key("backspace"), typed("bad"), key("shift+tab"), key("shift+tab"), key("tab"), key("tab"), key("enter"))
	if pointerLine(v) != "> input-2" || !strings.Contains(v, "input-2 is bad") {
		t.Errorf("focus must go to the first invalid field:\n%s", v)
	}
}

// TestAddFormClicks checks the clicks on the title, the description and the
// input line of a field, none of which validates, and the clicks outside
// every field.
//
// Covers: FR-015, EC-011
func TestAddFormClicks(t *testing.T) {
	base := formView(t, termW, termH)
	for _, part := range []string{"input-3", "Description of input 3"} {
		y := rowOf(t, base, part)
		v := formView(t, termW, termH, click(y))
		if pointerLine(v) != "> input-3" || strings.Contains(v, "cannot be empty") {
			t.Errorf("click on %q: focus = %q:\n%s", part, pointerLine(v), v)
		}
	}
	y := rowOf(t, base, "Description of input 2") + 1 // the input line
	v := formView(t, termW, termH, click(y))
	if pointerLine(v) != "> input-2" || strings.Contains(v, "cannot be empty") {
		t.Errorf("click on the input line: %q", pointerLine(v))
	}
	for _, y := range []int{0, 1, 2, rowOf(t, base, "( ) ollama") + 1, termH - 1} {
		if got := formView(t, termW, termH, click(y)); got != base {
			t.Errorf("click at row %d changed the form:\n%s", y, got)
		}
	}
}

// TestAddFormChoice checks the choice field: the keys, the stops at both
// ends, the collapsed form once it loses the focus, and the click on an
// option, which also takes the focus.
//
// Covers: FR-016
func TestAddFormChoice(t *testing.T) {
	selected := func(v string) string {
		for _, l := range strings.Split(v, "\n") {
			if strings.Contains(l, "(•)") {
				return strings.TrimSpace(strings.Replace(l, "(•)", "", 1))
			}
		}
		return ""
	}
	cases := []struct {
		name string
		keys []string
		want string
	}{
		{"initial", nil, "openai"},
		{"down", []string{"down"}, "anthropic"},
		{"j", []string{"j", "j"}, "ollama"},
		{"stops at last", []string{"down", "down", "down", "j"}, "ollama"},
		{"up", []string{"down", "down", "up"}, "anthropic"},
		{"k", []string{"down", "k"}, "openai"},
		{"stops at first", []string{"up", "k"}, "openai"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := selected(formView(t, termW, termH, keys(c.keys...)...)); got != c.want {
				t.Errorf("selected = %q, want %q", got, c.want)
			}
		})
	}
	v := formView(t, termW, termH, key("down"), key("tab"))
	if strings.Contains(v, "( )") || selected(v) != "anthropic" {
		t.Errorf("an unfocused choice shows only the selected option:\n%s", v)
	}
	base := formView(t, termW, termH)
	v = formView(t, termW, termH, click(rowOf(t, base, "ollama")))
	if selected(v) != "ollama" || pointerLine(v) != "> input-1" {
		t.Errorf("click on an option must select it:\n%s", v)
	}
	// A click on the collapsed choice of another field takes the focus.
	v = formView(t, termW, termH, key("tab"), key("shift+tab"), key("tab"), click(rowOf(t, formView(t, termW, termH, key("tab")), "(•) openai")))
	if pointerLine(v) != "> input-1" {
		t.Errorf("click on a collapsed choice must focus it:\n%s", v)
	}
}

// TestAddFormLongValue checks the horizontal scroll of a long value, which is
// never wrapped nor truncated, and that the full value reaches the
// equivalent command.
//
// Covers: FR-011, EC-012
func TestAddFormLongValue(t *testing.T) {
	const w = 40
	long := strings.Repeat("0123456789", 6) + "END"
	v := formView(t, w, termH, key("tab"), typed(long))
	line := v[strings.Index(v, "Description of input 2"):]
	line = strings.Split(line, "\n")[1]
	if !strings.Contains(line, "END") || len([]rune(line)) > w || strings.Contains(line, "0123456789012345678901234567890123456789012") {
		t.Errorf("the line must scroll to keep the cursor visible: %q", line)
	}
	if h := formView(t, w, termH, key("tab"), typed(long), key("home")); !strings.Contains(h, "0123456789") {
		t.Errorf("Home must scroll back to the start:\n%s", h)
	}
	v = formView(t, 200, termH, key("tab"), typed(long), key("tab"), typed("x"), key("enter"))
	if !strings.Contains(strings.ReplaceAll(v, "\n", ""), "openai "+long+" x") {
		t.Errorf("the full value must reach the equivalent command:\n%s", v)
	}
}

// TestAddFormPaste checks that control characters typed or pasted into a text
// field are discarded and the rest kept, and a multi-line paste never submits.
//
// Covers: EC-013
func TestAddFormPaste(t *testing.T) {
	paste := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a\nb\r\tc\x7fd\u0085e"), Paste: true}
	v := formView(t, termW, termH, key("tab"), paste)
	if pointerLine(v) != "> input-2" || !strings.Contains(v, "  abcde") {
		t.Errorf("control characters must be discarded:\n%s", v)
	}
	if strings.Contains(v, "Will create") {
		t.Errorf("a paste submitted the form:\n%s", v)
	}
	if got := formView(t, termW, termH, key("tab"), typed("x\ny"), key("tab")); pointerLine(got) != "> input-3" {
		t.Errorf("typed newline must be discarded, then the value xy is valid:\n%s", got)
	}
}

// TestAddFormShortTerminal checks the scrolling of the fields on a short
// terminal: the focused field stays visible, with the indicators.
//
// Covers: FR-011, EC-010
func TestAddFormShortTerminal(t *testing.T) {
	const h = 9
	top := formView(t, termW, h)
	if pointerLine(top) != "> input-1" || !strings.Contains(top, "↓ more") || strings.Contains(top, "↑ more") {
		t.Errorf("top of a short form needs only ↓ more:\n%s", top)
	}
	bottom := formView(t, termW, h, key("tab"), typed("x"), key("tab"))
	if pointerLine(bottom) != "> input-3" || !strings.Contains(bottom, "↑ more") || strings.Contains(bottom, "↓ more") {
		t.Errorf("last field needs only ↑ more:\n%s", bottom)
	}
	if lines := strings.Split(bottom, "\n"); len(lines) > h {
		t.Errorf("the form is %d lines, taller than %d:\n%s", len(lines), h, bottom)
	}
	back := formView(t, termW, h, key("tab"), typed("x"), key("tab"), key("shift+tab"), key("shift+tab"))
	if back != top {
		t.Errorf("going back must restore the top:\n%s", back)
	}
	grown := formView(t, termW, h, key("tab"), typed("x"), key("tab"), tea.WindowSizeMsg{Width: termW, Height: termH})
	if strings.Contains(grown, "more") || pointerLine(grown) != "> input-3" {
		t.Errorf("growing must show every field:\n%s", grown)
	}
}

// TestAddFormBackKeepsValues checks that Esc returns to the selector with the
// pointer on the feature, that reopening it shows the same values with the
// focus on the first field, and that nothing runs.
//
// Covers: FR-017, EC-014
func TestAddFormBackKeepsValues(t *testing.T) {
	before := formCalls.Load()
	sel, res := driveAddWith(t, []tui.Feature{flowForm()}, termW, termH,
		key("enter"), key("down"), key("tab"), typed("mine"), key("esc"), key("q"))
	if res.Action != tui.ActionQuit || !strings.HasPrefix(pointerLine(sel), "> feature-1") || strings.Contains(sel, "aicli › add ›") {
		t.Errorf("Esc must return to the selector with the pointer kept (%+v):\n%s", res, sel)
	}
	again, _ := driveAddWith(t, []tui.Feature{flowForm()}, termW, termH,
		key("enter"), key("down"), key("tab"), typed("mine"), key("esc"), key("enter"), key("ctrl+c"))
	if pointerLine(again) != "> input-1" || !strings.Contains(again, "(•) anthropic") || !strings.Contains(again, "  mine") {
		t.Errorf("reopening must show the kept values, focus on the first field:\n%s", again)
	}
	if formCalls.Load() != before {
		t.Error("a feature plan ran")
	}
}

// TestAddFormCtrlC checks that Ctrl+C after filling fields runs nothing and
// creates nothing.
//
// Covers: FR-010, EC-014
func TestAddFormCtrlC(t *testing.T) {
	before := formCalls.Load()
	_, r := formDrive(t, termW, termH, key("down"), key("tab"), typed("mine"), key("tab"), typed("x"))
	if r.Action != tui.ActionQuit || r.Values != nil {
		t.Errorf("result = %+v, want quit", r)
	}
	if formCalls.Load() != before {
		t.Error("a feature plan ran")
	}
}

// TestAddFormMouseCaptured checks, through the real program, that the mouse
// stays captured in the form: enabled once at the start and released only
// after the form was drawn.
//
// Covers: FR-015
func TestAddFormMouseCaptured(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = r.Close(); _ = w.Close() })
	out := &syncBuffer{}
	done := make(chan error, 1)
	go func() {
		_, err := tui.RunAdd(context.Background(), []tui.Feature{flowForm()}, r, out, false)
		done <- err
	}()
	wait := func(text string) {
		deadline := time.Now().Add(5 * time.Second)
		for !strings.Contains(out.String(), text) {
			if time.Now().After(deadline) {
				t.Fatalf("never saw %q in %q", text, out.String())
			}
			time.Sleep(10 * time.Millisecond)
		}
	}
	wait("> feature-1")
	if _, err := w.WriteString("\r"); err != nil {
		t.Fatal(err)
	}
	wait("aicli › add › feature-1")
	if s := out.String(); strings.Count(s, mouseOn) != 1 || strings.Contains(s, mouseOff) {
		t.Errorf("the mouse must stay captured in the form: %q", s)
	}
	if _, err := w.WriteString("\x03"); err != nil {
		t.Fatal(err)
	}
	if err := <-done; err != nil {
		t.Fatal(err)
	}
}
