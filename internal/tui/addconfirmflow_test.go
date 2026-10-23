package tui_test

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/ro-uc-ru/aicli/internal/tui"
)

// confirmFeature returns a feature with one text input whose plan lists two
// paths and whose check fails with check, if not nil. With noInputs it
// declares no inputs.
//
// In: check, the error of the read-only check; plan, the error of the plan;
// noInputs. Out: the feature. Side effects: none.
func confirmFeature(check, plan error, noInputs bool) tui.Feature {
	f := tui.Feature{Name: "feature-1", Short: "Description of feature 1", Listed: true,
		Inputs: []tui.FeatureInput{{Name: "input-1", Description: "Description of input 1"}},
		Plan: func(v []string) ([]string, error) {
			if plan != nil {
				return nil, plan
			}
			return []string{"features/one.txt", "features/two.md"}, nil
		},
		Check: func([]string) (tui.Preview, error) {
			return tui.Preview{Create: []string{"features/one.txt", "features/two.md"}}, check
		},
		Command: func(v []string) string { return "aicli add feature-1 -- " + strings.Join(v, " ") },
	}
	if noInputs {
		f.Inputs = nil
		f.Command = func([]string) string { return "aicli add feature-1" }
	}
	return f
}

// confirmDrive runs the add model with the feature, sends msgs and, if the
// model has not quit by itself, ends with Ctrl+C.
//
// In: t, the test; f, the feature; quit, whether msgs already end the
// program; msgs. Out: the last screen and the result. Side effects: as
// driveAdd.
func confirmDrive(t *testing.T, f tui.Feature, quit bool, msgs ...tea.Msg) (string, tui.AddResult) {
	t.Helper()
	if !quit {
		msgs = append(msgs, key("ctrl+c"))
	}
	return driveAddWith(t, []tui.Feature{f}, termW, termH, msgs...)
}

// toConfirm are the messages that open the form, fill its only field with
// "val" and show the confirmation.
func toConfirm() []tea.Msg { return []tea.Msg{key("enter"), typed("val"), key("enter")} }

// TestAddConfirmScreen checks the header, the description, the equivalent
// command, `Will create:` with the paths in plan order and indented, and the
// buttons with the focus on `Cancel`.
//
// Covers: FR-012, FR-018
func TestAddConfirmScreen(t *testing.T) {
	v, r := confirmDrive(t, confirmFeature(nil, nil, false), false, toConfirm()...)
	if r.Action != tui.ActionQuit {
		t.Errorf("result = %+v", r)
	}
	lines := strings.Split(v, "\n")
	want := []string{
		"aicli › add › feature-1",
		"Description of feature 1",
		"",
		"aicli add feature-1 -- val",
		"",
		"Will create:",
		"  features/one.txt",
		"  features/two.md",
	}
	for i, w := range want {
		if i >= len(lines) || lines[i] != w {
			t.Fatalf("line %d: want %q in:\n%s", i, w, v)
		}
	}
	if !strings.Contains(v, "  Add    [ Cancel ]") {
		t.Errorf("buttons or focus on Cancel missing:\n%s", v)
	}
	if !strings.Contains(v, "enter confirm") || !strings.Contains(v, "esc back") {
		t.Errorf("footer missing:\n%s", v)
	}
}

// TestAddConfirmKeys checks the focus and the keys: `←`, `→` and `Tab` move
// the focus, `Enter` presses the focused button, `y` presses `Add`, `n`
// presses `Cancel`, and other keys do nothing.
//
// Covers: FR-018
func TestAddConfirmKeys(t *testing.T) {
	f := confirmFeature(nil, nil, false)
	add := tui.ActionAdd
	cancel := tui.ActionCancel
	cases := []struct {
		name string
		keys []string
		want tui.AddAction
	}{
		{"enter on Cancel", []string{"enter"}, cancel},
		{"tab then enter", []string{"tab", "enter"}, add},
		{"right then enter", []string{"right", "enter"}, add},
		{"left then enter", []string{"left", "enter"}, add},
		{"tab twice then enter", []string{"tab", "tab", "enter"}, cancel},
		{"y", []string{"y"}, add},
		{"n", []string{"tab", "n"}, cancel},
		{"unlisted keys do nothing", []string{"x", "a", " ", "backspace", "enter"}, cancel},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, r := confirmDrive(t, f, true, append(toConfirm(), keys(c.keys...)...)...)
			if r.Action != c.want {
				t.Errorf("action = %q, want %q", r.Action, c.want)
			}
			if c.want == add && (len(r.Values) != 1 || r.Values[0] != "val" || r.Index != 0) {
				t.Errorf("result = %+v, want index 0 and the typed value", r)
			}
		})
	}
	v, _ := confirmDrive(t, f, false, append(toConfirm(), key("tab"))...)
	if !strings.Contains(v, "[ Add ]    Cancel") {
		t.Errorf("Tab must focus Add:\n%s", v)
	}
	v, _ = confirmDrive(t, f, false, append(toConfirm(), key("tab"), key("right"))...)
	if !strings.Contains(v, "  Add    [ Cancel ]") {
		t.Errorf("→ must move the focus back:\n%s", v)
	}
}

// TestAddConfirmBackAndEdit checks that Esc returns to the form with the typed
// values kept and that, after an edit, the confirmation shows the new command.
//
// Covers: FR-018, EC-017
func TestAddConfirmBackAndEdit(t *testing.T) {
	f := confirmFeature(nil, nil, false)
	back, _ := confirmDrive(t, f, false, append(toConfirm(), key("esc"))...)
	if !strings.Contains(back, "aicli › add › feature-1") || !strings.Contains(back, "val") || strings.Contains(back, "Will create") {
		t.Errorf("Esc must return to the form with the value:\n%s", back)
	}
	v, _ := confirmDrive(t, f, false, append(toConfirm(), key("esc"), typed("2"), key("enter"))...)
	if !strings.Contains(v, "aicli add feature-1 -- val2") || strings.Contains(v, "-- val\n") {
		t.Errorf("the new command must be shown:\n%s", v)
	}
}

// TestAddConfirmNoInputs checks that a feature with no inputs goes straight to
// the confirmation and that Esc returns to the selector with the pointer on it.
//
// Covers: FR-018, EC-008
func TestAddConfirmNoInputs(t *testing.T) {
	f := confirmFeature(nil, nil, true)
	v, _ := confirmDrive(t, f, false, key("enter"))
	if !strings.Contains(v, "Will create:") || !strings.Contains(v, "aicli add feature-1\n") || !strings.Contains(v, "[ Cancel ]") {
		t.Errorf("a feature with no inputs must show its confirmation:\n%s", v)
	}
	sel, r := driveAddWith(t, []tui.Feature{f}, termW, termH, key("enter"), key("esc"), key("q"))
	if r.Action != tui.ActionQuit || !strings.HasPrefix(pointerLine(sel), "> feature-1") || strings.Contains(sel, "Will create") {
		t.Errorf("Esc must return to the selector (%+v):\n%s", r, sel)
	}
}

// TestAddConfirmBlocked checks the confirmation of a feature whose plan fails
// or whose files already exist: the error lines in place of the list, only
// `Cancel`, and `y`, `←`, `→` and `Tab` doing nothing.
//
// Covers: FR-018
func TestAddConfirmBlocked(t *testing.T) {
	exists := errors.New("cannot add feature-1, these files already exist:\nfeatures/one.txt\nfeatures/two.md")
	cases := map[string]struct {
		f    tui.Feature
		want []string
	}{
		"existing files": {confirmFeature(exists, nil, false), []string{"Cannot add feature-1, these files already exist:", "\n  features/one.txt", "\n  features/two.md"}},
		"failing plan":   {confirmFeature(nil, errors.New("the plan failed"), false), []string{"The plan failed"}},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			v, _ := confirmDrive(t, c.f, false, append(toConfirm(), keys("y", "left", "right", "tab")...)...)
			for _, w := range append(c.want, "[ Cancel ]") {
				if !strings.Contains(v, w) {
					t.Errorf("missing %q:\n%s", w, v)
				}
			}
			if strings.Contains(v, "Will create:") || strings.Contains(v, "Add ") || strings.Contains(v, "Error:") {
				t.Errorf("a blocked confirmation must show the error and only Cancel:\n%s", v)
			}
			_, r := confirmDrive(t, c.f, true, append(toConfirm(), keys("y", "tab", "right", "enter")...)...)
			if r.Action != tui.ActionCancel {
				t.Errorf("Enter must press Cancel, got %+v", r)
			}
			_, r = confirmDrive(t, c.f, true, append(toConfirm(), key("n"))...)
			if r.Action != tui.ActionCancel {
				t.Errorf("n must press Cancel, got %+v", r)
			}
		})
	}
}

// TestAddConfirmMouse checks, through the real program, that the mouse is
// released in the confirmation and captured again after Esc.
//
// Covers: FR-018
func TestAddConfirmMouse(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = r.Close(); _ = w.Close() })
	out := &syncBuffer{}
	done := make(chan error, 1)
	go func() {
		_, err := tui.RunAdd(context.Background(), []tui.Feature{confirmFeature(nil, nil, true)}, r, out, false)
		done <- err
	}()
	wait := func(cond func(string) bool) {
		deadline := time.Now().Add(5 * time.Second)
		for !cond(out.String()) {
			if time.Now().After(deadline) {
				t.Fatalf("condition never held for %q", out.String())
			}
			time.Sleep(10 * time.Millisecond)
		}
	}
	send := func(s string) {
		if _, err := w.WriteString(s); err != nil {
			t.Fatal(err)
		}
	}
	wait(func(s string) bool { return strings.Contains(s, "> feature-1") })
	send("\r")
	wait(func(s string) bool { return strings.Contains(s, "Will create:") })
	if s := out.String(); strings.Count(s, mouseOn) != 1 || !strings.Contains(s[strings.Index(s, mouseOn):], mouseOff) {
		t.Errorf("the mouse must be released in the confirmation: %q", s)
	}
	send("\x1b")
	wait(func(s string) bool { return strings.Count(s, mouseOn) >= 2 })
	send("q")
	if err := <-done; err != nil {
		t.Fatal(err)
	}
}
