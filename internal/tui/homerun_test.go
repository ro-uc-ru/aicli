package tui

import (
	"bytes"
	"context"
	"errors"
	"os"
	"strings"
	"syscall"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/ro-uc-ru/aicli/internal/scaffold"
)

// altScreenExit is the sequence that leaves the alternate screen.
const altScreenExit = "\x1b[?1049l"

// runHome runs RunHome with a pipe as input in a goroutine.
//
// In: t, the test; ctx, the run context; entries, the commands; in, the
// input stream. Out: the buffer receiving the output and a channel that
// delivers the result; the test fails if there is none within 5 seconds.
// Side effects: starts a goroutine that ends with RunHome.
func runHome(ctx context.Context, entries []Entry, in *os.File) (*bytes.Buffer, <-chan error) {
	var out bytes.Buffer
	done := make(chan error, 1)
	go func() { done <- RunHome(ctx, entries, in, &out, false) }()
	return &out, done
}

// wait returns the result of a run, failing the test on a timeout.
//
// In: t, the test; done, the result channel. Out: the error. Side effects: none.
func wait(t *testing.T, done <-chan error) error {
	t.Helper()
	select {
	case err := <-done:
		return err
	case <-time.After(5 * time.Second):
		t.Fatal("RunHome did not return")
		return nil
	}
}

// pipe returns a pipe whose ends are closed when the test ends.
//
// In: t, the test. Out: the read and write ends. Side effects: opens a pipe.
func pipe(t *testing.T) (r, w *os.File) {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = r.Close(); _ = w.Close() })
	return r, w
}

// TestRunHomeQuit checks that `q` returns nil and the output ends by
// leaving the alternate screen.
//
// Covers: FR-010
func TestRunHomeQuit(t *testing.T) {
	r, w := pipe(t)
	out, done := runHome(context.Background(), testEntries(), r)
	_, _ = w.WriteString("q")
	if err := wait(t, done); err != nil {
		t.Fatalf("err = %v", err)
	}
	if !strings.Contains(out.String(), altScreenExit) {
		t.Errorf("output does not leave the alternate screen: %q", out.String())
	}
}

// TestRunHomeCancel checks that a context cancelled with a cause returns
// that cause and the terminal is restored.
//
// Covers: EC-014
func TestRunHomeCancel(t *testing.T) {
	r, _ := pipe(t)
	ctx, cancel := context.WithCancelCause(context.Background())
	out, done := runHome(ctx, testEntries(), r)
	time.Sleep(200 * time.Millisecond)
	cause := &scaffold.SignalError{Signal: syscall.SIGINT}
	cancel(cause)
	if err := wait(t, done); !errors.Is(err, cause) {
		t.Fatalf("err = %v, want the cause", err)
	}
	if !strings.Contains(out.String(), altScreenExit) {
		t.Errorf("output does not leave the alternate screen: %q", out.String())
	}
}

// TestRunHomeHelpFailure checks that a failing help producer is returned
// after the alternate screen is left.
//
// Covers: FR-016
func TestRunHomeHelpFailure(t *testing.T) {
	boom := errors.New("boom")
	entries := []Entry{{Path: "bad", Short: "fails", Help: func() (string, error) { return "", boom }}}
	r, w := pipe(t)
	out, done := runHome(context.Background(), entries, r)
	_, _ = w.WriteString("\r")
	if err := wait(t, done); !errors.Is(err, boom) {
		t.Fatalf("err = %v, want boom", err)
	}
	if !strings.Contains(out.String(), altScreenExit) {
		t.Errorf("output does not leave the alternate screen: %q", out.String())
	}
}

// TestRunHomeClosedInput checks that a closed input file returns the start
// error.
//
// Covers: FR-016
func TestRunHomeClosedInput(t *testing.T) {
	r, _ := pipe(t)
	_ = r.Close()
	_, done := runHome(context.Background(), testEntries(), r)
	err := wait(t, done)
	if err == nil || !strings.HasPrefix(err.Error(), "cannot start the home screen: ") {
		t.Fatalf("err = %v", err)
	}
}

// TestHomeModelFlow checks opening an entry, the commands for the mouse,
// going back with the pointer unchanged, and the stored help error.
//
// Covers: FR-007, FR-010, FR-016
func TestHomeModelFlow(t *testing.T) {
	entries := []Entry{
		{Path: "a", Short: "A", Help: func() (string, error) { return "help of a\n", nil }},
		{Path: "b", Short: "B", Help: func() (string, error) { return "help of b\n", nil }},
	}
	var m tea.Model = NewHomeModel(entries, false, 80, 40)
	step := func(msg tea.Msg) tea.Cmd {
		var cmd tea.Cmd
		m, cmd = m.Update(msg)
		return cmd
	}
	step(key("j"))
	cmd := step(key("enter"))
	if cmd == nil {
		t.Fatal("no command from enter")
	}
	open := cmd()
	if cmd = step(open); cmd == nil || cmd() != tea.DisableMouse() {
		t.Error("opening did not return the disable-mouse command")
	}
	if v := m.View(); !strings.Contains(v, "aicli › b") || !strings.Contains(v, "help of b") {
		t.Errorf("detail view = %q", v)
	}
	if cmd = step(tea.MouseMsg{Action: tea.MouseActionPress, Button: tea.MouseButtonLeft}); cmd != nil {
		t.Error("mouse acted in the detail view")
	}
	if cmd = step(tea.KeyMsg{Type: tea.KeyEsc}); cmd == nil || cmd() != tea.EnableMouseCellMotion() {
		t.Error("going back did not return the enable-mouse command")
	}
	if hm := m.(HomeModel); hm.list.pointer != 1 || hm.inDetail {
		t.Errorf("after back: pointer %d inDetail %v", hm.list.pointer, hm.inDetail)
	}

	boom := errors.New("boom")
	m = NewHomeModel([]Entry{{Path: "x", Help: func() (string, error) { return "", boom }}}, false, 80, 40)
	cmd = step(openMsg{Index: 0})
	if !isQuit(cmd) || !errors.Is(m.(HomeModel).Err(), boom) {
		t.Error("a help failure did not quit with the error")
	}
}
