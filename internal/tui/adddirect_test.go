package tui

import (
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// directFeatures returns two features: a plain one and a multi-choice one
// whose Added function counts its calls and marks the first choice added.
func directFeatures(calls *int) []Feature {
	multi := multiFeature(nil)
	multi.Added = func() []bool {
		*calls++
		return []bool{true, false, false, false}
	}
	return []Feature{{Name: "plain", Listed: true}, multi}
}

// feed feeds messages to a model, running each command it returns and
// feeding back the form and selector messages they produce; it returns the
// final model.
func feed(m tea.Model, msgs ...tea.Msg) tea.Model {
	for len(msgs) > 0 {
		var cmd tea.Cmd
		m, cmd = m.Update(msgs[0])
		msgs = msgs[1:]
		if cmd == nil {
			continue
		}
		switch out := cmd().(type) {
		case formDoneMsg, formBackMsg, selectMsg:
			msgs = append([]tea.Msg{out}, msgs...)
		}
	}
	return m
}

// TestAddDirectStart checks that a model started with WithDirect shows the
// form of that feature, with the already added choices read once, and that
// Esc quits with nothing added.
//
// Covers: FR-005, FR-007, EC-016
func TestAddDirectStart(t *testing.T) {
	calls := 0
	m := NewAddModel(directFeatures(&calls), false, 80, 30, WithDirect(1))
	if m.screen != screenForm || m.current != 1 || calls != 1 {
		t.Fatalf("screen %v, current %d, Added calls %d", m.screen, m.current, calls)
	}
	v := m.View()
	for _, want := range []string{"aicli › add › feature-1", "[✓] openai (already added)", "esc quit"} {
		if !strings.Contains(v, want) {
			t.Errorf("view lacks %q:\n%s", want, v)
		}
	}
	m = feed(m, tea.KeyMsg{Type: tea.KeyDown}, spacePress).(AddModel)
	if calls != 1 {
		t.Errorf("Added called %d times while typing", calls)
	}
	_, cmd := m.Update(keyEsc)
	if cmd == nil {
		t.Fatal("Esc produced no command")
	}
	if _, ok := cmd().(tea.QuitMsg); !ok || m.Result().Action != ActionQuit {
		t.Error("Esc did not quit")
	}
}

// TestAddDirectConfirmBack checks that Esc in the confirmation returns to
// the form, whose Esc still quits.
//
// Covers: FR-005, EC-016
func TestAddDirectConfirmBack(t *testing.T) {
	calls := 0
	fs := directFeatures(&calls)
	fs[1].Plan = func([]string) ([]string, error) { return []string{"a.txt"}, nil }
	var m tea.Model = NewAddModel(fs, false, 80, 30, WithDirect(1))
	m = feed(m, spacePress, keyEnter)
	if am := m.(AddModel); am.screen != screenConfirm || !reflect.DeepEqual(am.confirm.values, []string{"claude"}) {
		t.Fatalf("screen %v, values %q", am.screen, am.confirm.values)
	}
	m = feed(m, keyEsc)
	if m.(AddModel).screen != screenForm {
		t.Fatal("Esc in the confirmation did not return to the form")
	}
	if _, cmd := m.Update(keyEsc); cmd == nil || !isQuit(cmd) {
		t.Error("Esc in the form did not quit")
	}
}

// TestAddSelectorEscBack checks that, started on the selector, Esc in the
// form returns to the selector, and that Added is read each time the form
// opens, not before.
//
// Covers: FR-005, FR-007, EC-016
func TestAddSelectorEscBack(t *testing.T) {
	calls := 0
	var m tea.Model = NewAddModel(directFeatures(&calls), false, 80, 30)
	if calls != 0 {
		t.Fatalf("Added called %d times before the form opened", calls)
	}
	m = feed(m, selectMsg{Index: 1})
	if am := m.(AddModel); am.screen != screenForm || calls != 1 || am.direct {
		t.Fatalf("screen %v, calls %d", am.screen, calls)
	}
	m = feed(m, keyEsc)
	if am := m.(AddModel); am.screen != screenSelector || am.Result().Action != ActionQuit {
		t.Errorf("screen after Esc = %v", am.screen)
	}
}

// TestRunAddDirectEsc checks, through the real program, that Esc on a
// direct form leaves the alternate screen and returns the quit action.
//
// Covers: FR-005, EC-016
func TestRunAddDirectEsc(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = r.Close(); _ = w.Close() })
	calls := 0
	out := &syncBuffer{}
	done := make(chan addOutcome, 1)
	go func() {
		res, err := RunAdd(t.Context(), directFeatures(&calls), r, out, false, WithDirect(1))
		done <- addOutcome{res, err}
	}()
	deadline := time.Now().Add(5 * time.Second)
	for !strings.Contains(out.String(), "esc quit") {
		if time.Now().After(deadline) {
			t.Fatalf("form never shown: %q", out.String())
		}
		time.Sleep(10 * time.Millisecond)
	}
	if _, err := w.WriteString("\x1b"); err != nil {
		t.Fatal(err)
	}
	if o := waitAdd(t, done); o.err != nil || o.res.Action != ActionQuit {
		t.Fatalf("result = %+v, %v", o.res, o.err)
	}
	const altScreenEnter = "\x1b[?1049h"
	s := out.String()
	if exit := strings.LastIndex(s, altScreenExit); exit < strings.LastIndex(s, altScreenEnter) || !strings.Contains(s, altScreenEnter) {
		t.Errorf("alternate screen not entered and left: %q", s)
	}
}
