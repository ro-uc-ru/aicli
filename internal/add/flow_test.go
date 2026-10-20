package add

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/ro-uc-ru/aicli/internal/featuretest"
)

const (
	// altExit is the sequence that leaves the alternate screen.
	altExit = "\x1b[?1049l"
	// flowTimeout bounds every wait of the flow tests.
	flowTimeout = 5 * time.Second
)

// syncOut is a writer safe for use by the program and the test.
type syncOut struct {
	mu  sync.Mutex
	buf strings.Builder
}

// Write appends p. In: p, the bytes. Out: len(p) and nil. Side effects: locks.
func (b *syncOut) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

// String returns what was written. In: none. Out: the text. Side effects: locks.
func (b *syncOut) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

// flow is a running interactive session on a pipe.
type flow struct {
	t    *testing.T
	in   *os.File
	out  *syncOut
	done chan error
}

// startFlow runs runInteractive with the real screens on a pipe, with the
// main test feature, in a new project root.
//
// In: t, the test. Out: the session, the project root. Side effects: starts a
// goroutine running the program and creates a temp directory.
func startFlow(t *testing.T) (*flow, string) {
	t.Helper()
	add, ctx, root := interactiveSetup(t)
	add.SetContext(ctx)
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = r.Close(); _ = w.Close() })
	f := &flow{t: t, in: w, out: &syncOut{}, done: make(chan error, 1)}
	go func() { f.done <- runInteractive(ctx, add, Features(add), r, f.out, xterm) }()
	f.wait("> " + featuretest.Main)
	return f, root
}

// wait blocks until the output contains text.
//
// In: text. Out: none; the test fails after flowTimeout. Side effects: none.
func (f *flow) wait(text string) {
	f.t.Helper()
	f.waitCount(text, 1)
}

// waitCount blocks until the output contains text n times.
//
// In: text; n. Out: none; the test fails after flowTimeout. Side effects: none.
func (f *flow) waitCount(text string, n int) {
	f.t.Helper()
	deadline := time.Now().Add(flowTimeout)
	for strings.Count(f.out.String(), text) < n {
		if time.Now().After(deadline) {
			f.t.Fatalf("never saw %q (x%d) in %q", text, n, f.out.String())
		}
		time.Sleep(10 * time.Millisecond)
	}
}

// send types s into the program.
//
// In: s. Out: none. Side effects: writes to the pipe.
func (f *flow) send(s string) {
	f.t.Helper()
	if _, err := f.in.WriteString(s); err != nil {
		f.t.Fatal(err)
	}
}

// fillToConfirm selects the main feature, accepts the first choice, types
// name and shows the confirmation.
//
// In: name, the value of the text input. Out: none. Side effects: as send.
func (f *flow) fillToConfirm(name string) {
	f.send("\r")
	f.wait("aicli › add › " + featuretest.Main)
	f.send("\r")
	f.send(name)
	f.send("\r")
	f.wait("Will create:")
}

// finish waits for the program to end.
//
// In: none. Out: the error runInteractive returned; the test fails after
// flowTimeout. Side effects: none.
func (f *flow) finish() error {
	f.t.Helper()
	select {
	case err := <-f.done:
		return err
	case <-time.After(flowTimeout):
		f.t.Fatalf("the session did not end: %q", f.out.String())
		return nil
	}
}

// after returns what was written after the last exit from the alternate
// screen, once it has been left.
//
// In: none. Out: that text; the test fails if the screen was never left.
// Side effects: none.
func (f *flow) after() string {
	f.t.Helper()
	s := f.out.String()
	i := strings.LastIndex(s, altExit)
	if i < 0 {
		f.t.Fatalf("the alternate screen was not left: %q", s)
	}
	rest := s[i+len(altExit):]
	// Drop the cursor and mouse control sequences the program emits as it exits.
	for _, seq := range []string{"\x1b[?25h", "\x1b[?1002l", "\x1b[?1006l"} {
		rest = strings.ReplaceAll(rest, seq, "")
	}
	return rest
}

// entries lists the files below root.
//
// In: root. Out: the slash paths of the files. Side effects: none.
func entries(t *testing.T, root string) []string {
	t.Helper()
	var out []string
	err := filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
		if err == nil && !d.IsDir() {
			rel, _ := filepath.Rel(root, p)
			out = append(out, filepath.ToSlash(rel))
		}
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	return out
}

// TestFlowConfirmationShowsCommand checks the equivalent command and the file
// list of the real feature in the confirmation, with a value that starts with
// `-` (shown after `--`) and one with spaces (shown quoted).
//
// Covers: FR-018, EC-015, EC-016
func TestFlowConfirmationShowsCommand(t *testing.T) {
	f, root := startFlow(t)
	f.fillToConfirm("-a b")
	s := f.out.String()
	for _, want := range []string{"aicli add " + featuretest.Main + " -- one '-a b'", "Will create:", "features/-a b.txt"} {
		if !strings.Contains(s, want) {
			t.Errorf("missing %q in %q", want, s)
		}
	}
	f.send("\x03")
	if err := f.finish(); err != nil {
		t.Fatal(err)
	}
	if got := entries(t, root); len(got) != 0 {
		t.Errorf("files created: %v", got)
	}
}

// TestFlowCancel checks that Cancel restores the terminal, then prints
// `Nothing added.`, and creates nothing.
//
// Covers: FR-018
func TestFlowCancel(t *testing.T) {
	f, root := startFlow(t)
	f.fillToConfirm("name")
	f.send("\r") // the focus starts on Cancel
	if err := f.finish(); err != nil {
		t.Fatal(err)
	}
	if got := f.after(); got != "Nothing added.\n" {
		t.Errorf("after the terminal was restored: %q", got)
	}
	if got := entries(t, root); len(got) != 0 {
		t.Errorf("files created: %v", got)
	}
}

// TestFlowAdd checks that Add restores the terminal, then creates the files
// and prints the report.
//
// Covers: FR-018, FR-019
func TestFlowAdd(t *testing.T) {
	f, root := startFlow(t)
	f.fillToConfirm("name")
	f.send("y")
	if err := f.finish(); err != nil {
		t.Fatal(err)
	}
	want := "create  features/name.txt\ncreate  features/name.md\n\nAdded " + featuretest.Main + "\n\nNext steps:\n  " +
		strings.Join(featuretest.NextSteps, "\n  ") + "\n"
	if got := f.after(); got != want {
		t.Errorf("after the terminal was restored: %q, want %q", got, want)
	}
	if got := entries(t, root); len(got) != 2 {
		t.Errorf("files = %v, want 2", got)
	}
}

// TestFlowFileAppearsBeforeAdd checks that a file created between the
// confirmation and Add makes the run fail with the FR-023 error, replaces
// nothing, creates nothing else and leaves the terminal restored with nothing
// printed.
//
// Covers: FR-018, EC-018, EC-019
func TestFlowFileAppearsBeforeAdd(t *testing.T) {
	f, root := startFlow(t)
	f.fillToConfirm("name")
	path := filepath.Join(root, "features", "name.txt")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("mine"), 0o644); err != nil {
		t.Fatal(err)
	}
	f.send("y")
	err := f.finish()
	want := "cannot add " + featuretest.Main + ", these files already exist:\nfeatures/name.txt"
	if err == nil || err.Error() != want {
		t.Fatalf("error = %v, want %q", err, want)
	}
	if got := f.after(); got != "" {
		t.Errorf("something is left on the normal screen: %q", got)
	}
	if b, _ := os.ReadFile(path); string(b) != "mine" {
		t.Errorf("the existing file was replaced: %q", b)
	}
	if got := entries(t, root); len(got) != 1 {
		t.Errorf("files = %v, want only the existing one", got)
	}
}

// TestFlowBlockedByExistingFile checks that the confirmation of a feature
// already added shows the error and only Cancel, and that nothing is created.
//
// Covers: FR-018
func TestFlowBlockedByExistingFile(t *testing.T) {
	f, root := startFlow(t)
	path := filepath.Join(root, "features", "name.txt")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("mine"), 0o644); err != nil {
		t.Fatal(err)
	}
	f.send("\r")
	f.wait("aicli › add › " + featuretest.Main)
	f.send("\r")
	f.send("name")
	f.send("\r")
	f.wait("Cannot add " + featuretest.Main + ", these files already exist:")
	f.send("y") // does nothing when blocked
	f.send("\r")
	if err := f.finish(); err != nil {
		t.Fatal(err)
	}
	if got := f.after(); got != "Nothing added.\n" {
		t.Errorf("after: %q", got)
	}
	if got := entries(t, root); len(got) != 1 {
		t.Errorf("files = %v", got)
	}
}
