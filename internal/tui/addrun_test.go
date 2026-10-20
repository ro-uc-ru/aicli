package tui

import (
	"context"
	"errors"
	"os"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/ro-uc-ru/aicli/internal/scaffold"
)

// syncBuffer is a write-safe buffer that tests can poll.
type syncBuffer struct {
	mu sync.Mutex
	b  strings.Builder
}

// Write appends p. In: p, the bytes. Out: len(p), nil. Side effects: none.
func (s *syncBuffer) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.Write(p)
}

// String returns what was written so far. In: none. Out: the text.
// Side effects: none.
func (s *syncBuffer) String() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.String()
}

// runAdd runs RunAdd in a goroutine.
//
// In: ctx, the run context; features, the features; in, the input stream.
// Out: the output buffer and a channel delivering the result and the error.
// Side effects: starts a goroutine that ends with RunAdd.
func runAdd(ctx context.Context, features []Feature, in *os.File) (*syncBuffer, <-chan addOutcome) {
	out := &syncBuffer{}
	done := make(chan addOutcome, 1)
	go func() {
		res, err := RunAdd(ctx, features, in, out, false)
		done <- addOutcome{res, err}
	}()
	return out, done
}

// addOutcome is what RunAdd returned.
type addOutcome struct {
	res AddResult
	err error
}

// waitAdd returns the outcome of a run, failing the test on a timeout.
func waitAdd(t *testing.T, done <-chan addOutcome) addOutcome {
	t.Helper()
	select {
	case o := <-done:
		return o
	case <-time.After(5 * time.Second):
		t.Fatal("RunAdd did not return")
		return addOutcome{}
	}
}

// waitText waits until the output contains s, failing the test on a timeout.
func waitText(t *testing.T, out *syncBuffer, s string) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for !strings.Contains(out.String(), s) {
		if time.Now().After(deadline) {
			t.Fatalf("output never contained %q: %q", s, out.String())
		}
		time.Sleep(10 * time.Millisecond)
	}
}

// addFeatures returns two features: one with a text input and one with none.
func addFeatures() []Feature {
	var calls int
	f1 := confirmFeature(&calls, "", nil)
	f2 := Feature{Name: "feature-2", Short: "No inputs", Listed: true,
		Plan: func([]string) ([]string, error) { return []string{"only.py"}, nil }}
	return []Feature{f1, f2}
}

// TestRunAddQuit checks that `q` returns the quit action and that the output
// leaves the alternate screen.
//
// Covers: FR-010
func TestRunAddQuit(t *testing.T) {
	r, w := pipe(t)
	out, done := runAdd(context.Background(), addFeatures(), r)
	waitText(t, out, "aicli › add")
	_, _ = w.WriteString("q")
	o := waitAdd(t, done)
	if o.err != nil || o.res.Action != ActionQuit {
		t.Fatalf("outcome = %+v", o)
	}
	if !strings.Contains(out.String(), altScreenExit) {
		t.Errorf("output does not leave the alternate screen: %q", out.String())
	}
}

// TestRunAddSignal checks that a context cancelled with a signal error
// returns that cause after leaving the alternate screen.
//
// Covers: EC-020
func TestRunAddSignal(t *testing.T) {
	r, _ := pipe(t)
	ctx, cancel := context.WithCancelCause(context.Background())
	out, done := runAdd(ctx, addFeatures(), r)
	waitText(t, out, "aicli › add")
	cancel(&scaffold.SignalError{Signal: os.Interrupt})
	o := waitAdd(t, done)
	var se *scaffold.SignalError
	if !errors.As(o.err, &se) {
		t.Fatalf("err = %v, want SignalError", o.err)
	}
	if !strings.Contains(out.String(), altScreenExit) {
		t.Errorf("output does not leave the alternate screen: %q", out.String())
	}
}

// TestRunAddClosedInput checks that a closed input file returns the start
// error.
//
// Covers: FR-021
func TestRunAddClosedInput(t *testing.T) {
	r, _ := pipe(t)
	_ = r.Close()
	_, done := runAdd(context.Background(), addFeatures(), r)
	o := waitAdd(t, done)
	if o.err == nil || !strings.HasPrefix(o.err.Error(), "cannot start the add screen: ") {
		t.Fatalf("err = %v", o.err)
	}
}

// TestRunAddScripted checks a scripted run: select the first feature, fill
// its input, continue, and press `y`.
//
// Covers: FR-010, FR-019
func TestRunAddScripted(t *testing.T) {
	r, w := pipe(t)
	out, done := runAdd(context.Background(), addFeatures(), r)
	waitText(t, out, "feature-1")
	_, _ = w.WriteString("\r")
	waitText(t, out, "> input-1")
	_, _ = w.WriteString("  abc ")
	waitText(t, out, "abc")
	_, _ = w.WriteString("\r")
	waitText(t, out, "Will create:")
	_, _ = w.WriteString("y")
	o := waitAdd(t, done)
	want := AddResult{Action: ActionAdd, Index: 0, Values: []string{"abc"}}
	if o.err != nil || !reflect.DeepEqual(o.res, want) {
		t.Fatalf("outcome = %+v, want %+v", o, want)
	}
	if !strings.Contains(out.String(), altScreenExit) {
		t.Error("output does not leave the alternate screen")
	}
}

// step sends a message to the model and returns the new model and command.
func step(m tea.Model, msg tea.Msg) (tea.Model, tea.Cmd) { return m.Update(msg) }

// run feeds a command's message back to the model.
func run(m tea.Model, cmd tea.Cmd) tea.Model {
	if cmd == nil {
		return m
	}
	m, _ = m.Update(cmd())
	return m
}

// TestAddModelFlow checks the model: a feature with inputs goes through the
// form, `Esc` keeps the values, reopening shows them, a feature without
// inputs goes straight to the confirmation and `Esc` there returns to the
// selector, and `Cancel` returns the cancel action.
//
// Covers: FR-010, FR-017, EC-014
func TestAddModelFlow(t *testing.T) {
	var m tea.Model = NewAddModel(addFeatures(), false, 80, 30)
	var cmd tea.Cmd
	m, cmd = step(m, tea.KeyMsg{Type: tea.KeyEnter})
	m = run(m, cmd)
	if !strings.Contains(m.View(), "aicli › add › feature-1") {
		t.Fatalf("form not shown:\n%s", m.View())
	}
	m, _ = step(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("half")})
	m, cmd = step(m, tea.KeyMsg{Type: tea.KeyEsc})
	m = run(m, cmd)
	if !strings.HasPrefix(m.View(), "aicli › add\n") {
		t.Fatalf("selector not shown:\n%s", m.View())
	}
	if r := m.(AddModel).Result(); r.Action != ActionQuit {
		t.Errorf("result after Esc = %+v", r)
	}
	m, cmd = step(m, tea.KeyMsg{Type: tea.KeyEnter})
	m = run(m, cmd)
	if v := m.View(); !strings.Contains(v, "half") || !strings.Contains(v, "> input-1") {
		t.Errorf("reopened form lacks the values:\n%s", v)
	}
	m, cmd = step(m, tea.KeyMsg{Type: tea.KeyEsc})
	m = run(m, cmd)

	m, _ = step(m, tea.KeyMsg{Type: tea.KeyDown})
	m, cmd = step(m, tea.KeyMsg{Type: tea.KeyEnter})
	m = run(m, cmd)
	if !strings.Contains(m.View(), "Will create:") || !strings.Contains(m.View(), "  only.py") {
		t.Fatalf("confirmation not shown:\n%s", m.View())
	}
	m, cmd = step(m, tea.KeyMsg{Type: tea.KeyEsc})
	if cmd == nil || cmd() != tea.EnableMouseCellMotion() {
		t.Error("Esc did not enable the mouse")
	}
	if !strings.HasPrefix(m.View(), "aicli › add\n") {
		t.Fatalf("Esc did not return to the selector:\n%s", m.View())
	}
	m, cmd = step(m, tea.KeyMsg{Type: tea.KeyEnter})
	m = run(m, cmd)
	m, cmd = step(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("n")})
	if r := m.(AddModel).Result(); r.Action != ActionCancel || r.Index != 1 || !isQuit(cmd) {
		t.Errorf("cancel result = %+v", r)
	}
}

// TestAddModelEscBackFromConfirm checks that `Esc` in the confirmation of a
// feature with inputs returns to the form with its values, and that an
// edited value shows a new command on the next confirmation, and that
// `Ctrl+C` never adds.
//
// Covers: FR-017, EC-014
func TestAddModelEscBackFromConfirm(t *testing.T) {
	var m tea.Model = NewAddModel(addFeatures(), false, 80, 30)
	var cmd tea.Cmd
	m, cmd = step(m, tea.KeyMsg{Type: tea.KeyEnter})
	m = run(m, cmd)
	m, _ = step(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("old")})
	m, cmd = step(m, tea.KeyMsg{Type: tea.KeyEnter})
	m = run(m, cmd)
	if !strings.Contains(m.View(), "aicli add feature-1 old") {
		t.Fatalf("confirmation:\n%s", m.View())
	}
	m, _ = step(m, tea.KeyMsg{Type: tea.KeyEsc})
	if v := m.View(); !strings.Contains(v, "> input-1") || !strings.Contains(v, "old") {
		t.Fatalf("form after Esc:\n%s", v)
	}
	m, _ = step(m, tea.KeyMsg{Type: tea.KeyBackspace})
	m, _ = step(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("x")})
	m, cmd = step(m, tea.KeyMsg{Type: tea.KeyEnter})
	m = run(m, cmd)
	if !strings.Contains(m.View(), "aicli add feature-1 olx") {
		t.Errorf("edited confirmation:\n%s", m.View())
	}
	m, cmd = step(m, tea.KeyMsg{Type: tea.KeyCtrlC})
	if r := m.(AddModel).Result(); r.Action != ActionQuit || !isQuit(cmd) {
		t.Errorf("ctrl+c result = %+v", r)
	}
}
