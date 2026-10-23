package tui

import (
	"errors"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// confirmFeature returns a feature whose plan echoes the first value in two
// paths. calls counts the Plan and Check calls; existing, if non-empty, makes
// Check fail with the FR-023 error naming that path; planErr, if set, makes
// Plan fail.
func confirmFeature(calls *int, existing string, planErr error) Feature {
	return Feature{
		Name: "feature-1", Short: "Description of feature 1", Listed: true,
		Inputs: []FeatureInput{{Name: "input-1"}},
		Command: func(v []string) string {
			return "aicli add feature-1 " + strings.Join(v, " ")
		},
		Plan: func(v []string) ([]string, error) {
			*calls++
			if planErr != nil {
				return nil, planErr
			}
			return []string{"app/" + v[0] + ".py", "tests/test_" + v[0] + ".py"}, nil
		},
		Check: func(v []string) (Preview, error) {
			*calls++
			if existing != "" {
				return Preview{}, errors.New("Error: cannot add feature-1, these files already exist:\n  " + existing)
			}
			return Preview{Create: []string{"app/" + v[0] + ".py", "tests/test_" + v[0] + ".py"}}, nil
		},
	}
}

// pressC sends a key to the confirmation.
func pressC(m confirmModel, k tea.KeyMsg) (confirmModel, tea.Cmd) { return m.Update(k) }

// TestConfirmView checks the whole screen: header, description, command,
// heading, paths and buttons with the focus on Cancel, footer; and the plain
// output has no ESC.
//
// Covers: FR-012, FR-018
func TestConfirmView(t *testing.T) {
	var calls int
	m, cmd := newConfirmModel(confirmFeature(&calls, "", nil), []string{"x"}, false, 80, 30)
	want := strings.Join([]string{
		"aicli › add › feature-1",
		"Description of feature 1",
		"",
		"aicli add feature-1 x",
		"",
		"Will create:",
		"  app/x.py",
		"  tests/test_x.py",
		"",
		"  Add    [ Cancel ]",
		"←/→/tab choose · enter confirm · y add · n cancel · esc back · ctrl+c quit",
	}, "\n")
	if got := m.View(); got != want {
		t.Errorf("view:\n%s\nwant:\n%s", got, want)
	}
	if cmd == nil || cmd() != tea.DisableMouse() {
		t.Error("entering did not release the mouse")
	}
	if styled, _ := newConfirmModel(confirmFeature(&calls, "", nil), []string{"x"}, true, 80, 30); !strings.Contains(styled.View(), "\x1b") {
		t.Error("styled view has no ESC")
	}
}

// TestConfirmNoDescription checks that without a short description there is
// no description line and no blank line in its place.
//
// Covers: EC-008
func TestConfirmNoDescription(t *testing.T) {
	var calls int
	f := confirmFeature(&calls, "", nil)
	f.Short = ""
	m, _ := newConfirmModel(f, []string{"x"}, false, 80, 30)
	if got := m.View(); !strings.HasPrefix(got, "aicli › add › feature-1\naicli add feature-1 x\n") {
		t.Errorf("view = %q", got)
	}
}

// TestConfirmKeys checks the button keys: the focus starts on Cancel, the
// arrows and Tab move it, Enter presses the focused button, y adds, n
// cancels, and other keys do nothing.
//
// Covers: FR-018, EC-011
func TestConfirmKeys(t *testing.T) {
	var calls int
	start, _ := newConfirmModel(confirmFeature(&calls, "", nil), []string{"x"}, false, 80, 30)
	enter, _ := pressC(start, tea.KeyMsg{Type: tea.KeyEnter})
	if enter.Result() != confirmCancel {
		t.Errorf("enter on Cancel = %v", enter.Result())
	}
	for _, k := range []tea.KeyMsg{{Type: tea.KeyTab}, {Type: tea.KeyLeft}, {Type: tea.KeyRight}} {
		moved, _ := pressC(start, k)
		if !strings.Contains(moved.View(), "[ Add ]    Cancel  ") {
			t.Errorf("%v: %q", k, moved.View())
		}
		if done, _ := pressC(moved, tea.KeyMsg{Type: tea.KeyEnter}); done.Result() != confirmAdd {
			t.Errorf("enter on Add = %v", done.Result())
		}
	}
	if y, _ := pressC(start, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("y")}); y.Result() != confirmAdd {
		t.Errorf("y = %v", y.Result())
	}
	moved, _ := pressC(start, tea.KeyMsg{Type: tea.KeyTab})
	if n, _ := pressC(moved, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("n")}); n.Result() != confirmCancel {
		t.Errorf("n = %v", n.Result())
	}
	for _, k := range []tea.KeyMsg{{Type: tea.KeyRunes, Runes: []rune("x")}, {Type: tea.KeyShiftTab}, {Type: tea.KeyHome}} {
		if g, cmd := pressC(start, k); g.Result() != confirmPending || cmd != nil || g.View() != start.View() {
			t.Errorf("%v acted", k)
		}
	}
	if _, cmd := pressC(start, tea.KeyMsg{Type: tea.KeyCtrlC}); cmd == nil {
		t.Error("ctrl+c did not quit")
	} else if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Error("ctrl+c did not quit")
	}
	if g, _ := start.Update(tea.MouseMsg{Action: tea.MouseActionPress, Button: tea.MouseButtonLeft, Y: 10}); g.Result() != confirmPending || g.View() != start.View() {
		t.Error("mouse acted")
	}
}

// TestConfirmEsc checks that Esc goes back and turns the mouse capture on
// again.
//
// Covers: FR-018
func TestConfirmEsc(t *testing.T) {
	var calls int
	m, _ := newConfirmModel(confirmFeature(&calls, "", nil), []string{"x"}, false, 80, 30)
	m, cmd := pressC(m, tea.KeyMsg{Type: tea.KeyEsc})
	if m.Result() != confirmBack || cmd == nil || cmd() != tea.EnableMouseCellMotion() {
		t.Errorf("result %v, cmd %v", m.Result(), cmd)
	}
}

// TestConfirmBlockedExisting checks the confirmation for a file that exists:
// the error in place of the paths, only Cancel, and y, ←, → and Tab doing
// nothing.
//
// Covers: FR-018
func TestConfirmBlockedExisting(t *testing.T) {
	var calls int
	m, _ := newConfirmModel(confirmFeature(&calls, "app/x.py", nil), []string{"x"}, false, 80, 30)
	got := m.View()
	for _, want := range []string{"\nCannot add feature-1, these files already exist:\n", "\n  app/x.py\n", "\n[ Cancel ]\n", "\nenter cancel · n cancel · esc back · ctrl+c quit"} {
		if !strings.Contains(got, want) {
			t.Errorf("view lacks %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "Will create:") || strings.Contains(got, "Add ") {
		t.Errorf("blocked view shows the normal parts:\n%s", got)
	}
	for _, k := range []tea.KeyMsg{{Type: tea.KeyRunes, Runes: []rune("y")}, {Type: tea.KeyLeft}, {Type: tea.KeyRight}, {Type: tea.KeyTab}} {
		if g, _ := pressC(m, k); g.Result() != confirmPending || g.View() != got {
			t.Errorf("%v acted when blocked", k)
		}
	}
	if g, _ := pressC(m, tea.KeyMsg{Type: tea.KeyEnter}); g.Result() != confirmCancel {
		t.Errorf("enter = %v", g.Result())
	}
	if styled, _ := newConfirmModel(confirmFeature(&calls, "app/x.py", nil), []string{"x"}, true, 80, 30); !strings.Contains(styled.View(), "\x1b") {
		t.Error("styled blocked view has no ESC")
	}
}

// TestConfirmBlockedPlan checks that a failing plan shows its message
// without `Error: ` and capitalised, and that Check is not run.
//
// Covers: FR-018
func TestConfirmBlockedPlan(t *testing.T) {
	var calls int
	m, _ := newConfirmModel(confirmFeature(&calls, "", errors.New("Error: bad value")), []string{"x"}, false, 80, 30)
	if got := m.View(); !strings.Contains(got, "\nBad value\n") || strings.Contains(got, "Error:") || !strings.Contains(got, "\n[ Cancel ]\n") {
		t.Errorf("view:\n%s", got)
	}
	if calls != 1 {
		t.Errorf("Plan and Check calls = %d, want 1", calls)
	}
}

// TestConfirmRecomputes checks that each new confirmation computes the plan
// and the check again and shows the command of the new values.
//
// Covers: EC-017, FR-018
func TestConfirmRecomputes(t *testing.T) {
	var calls int
	f := confirmFeature(&calls, "", nil)
	first, _ := newConfirmModel(f, []string{"old"}, false, 80, 30)
	second, _ := newConfirmModel(f, []string{"new"}, false, 80, 30)
	if calls != 4 {
		t.Errorf("calls = %d, want 4", calls)
	}
	if !strings.Contains(first.View(), "aicli add feature-1 old") || !strings.Contains(second.View(), "aicli add feature-1 new") || !strings.Contains(second.View(), "app/new.py") {
		t.Errorf("views:\n%s\n%s", first.View(), second.View())
	}
}

// TestConfirmWrapAndScroll checks that a long command is wrapped and never
// truncated, and that the list rows, heading included, scroll with the
// arrows, with the scroll hint in the footer.
//
// Covers: EC-010, FR-011, FR-018
func TestConfirmWrapAndScroll(t *testing.T) {
	var calls int
	f := confirmFeature(&calls, "", nil)
	f.Command = func([]string) string { return "aicli add feature-1 " + strings.Repeat("word ", 8) }
	m, _ := newConfirmModel(f, []string{"x"}, false, 20, 30)
	if got := strings.Join(m.command, " "); !strings.Contains(got, "word word") || len(m.command) < 3 {
		t.Errorf("command = %q", m.command)
	}
	for _, c := range m.command {
		if len(c) > 20 {
			t.Errorf("line %q wider than 20", c)
		}
	}
	f.Command = func([]string) string { return "aicli add feature-1 x" }
	f.Plan = func([]string) ([]string, error) { return []string{"p1", "p2", "p3", "p4", "p5"}, nil }
	f.Check = func([]string) (Preview, error) { return Preview{Create: []string{"p1", "p2", "p3", "p4", "p5"}}, nil }
	f.Short = ""
	m, _ = newConfirmModel(f, []string{"x"}, false, 80, 8)
	// header, cmd, up, heading, 1 path, down, buttons, footer
	v := m.View()
	if !strings.Contains(v, "Will create:") || !strings.Contains(v, "  p1") || strings.Contains(v, "  p2") || !strings.Contains(v, "↓ more") || !strings.Contains(v, "↑/↓ scroll · ←/→/tab") {
		t.Errorf("view:\n%s", v)
	}
	m, _ = pressC(m, tea.KeyMsg{Type: tea.KeyDown})
	m, _ = pressC(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	v = m.View()
	if !strings.Contains(v, "  p3") || !strings.Contains(v, "↑ more") || strings.Contains(v, "Will create:") {
		t.Errorf("scrolled view:\n%s", v)
	}
	for i := 0; i < 10; i++ {
		m, _ = pressC(m, tea.KeyMsg{Type: tea.KeyDown})
	}
	if !strings.Contains(m.View(), "  p5") || strings.Contains(m.View(), "↓ more") {
		t.Errorf("end view:\n%s", m.View())
	}
	m, _ = pressC(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
	if !strings.Contains(m.View(), "  p4") {
		t.Errorf("up view:\n%s", m.View())
	}
}

// TestConfirmCommandScrolls checks that on a very short terminal the command
// scrolls above the buttons, from its first line.
//
// Covers: EC-010
func TestConfirmCommandScrolls(t *testing.T) {
	var calls int
	f := confirmFeature(&calls, "", nil)
	f.Command = func([]string) string { return "aaaa bbbb cccc dddd" }
	m, _ := newConfirmModel(f, []string{"x"}, false, 5, 2)
	if got := m.View(); got != "aaaa\n"+m.buttons() {
		t.Errorf("view = %q", got)
	}
	m, _ = pressC(m, tea.KeyMsg{Type: tea.KeyDown})
	if !strings.HasPrefix(m.View(), "bbbb\n") {
		t.Errorf("scrolled view = %q", m.View())
	}
}
