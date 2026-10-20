package add

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"strings"
	"syscall"
	"testing"

	"github.com/spf13/cobra"

	"github.com/ro-uc-ru/aicli/internal/featuretest"
	"github.com/ro-uc-ru/aicli/internal/project"
	"github.com/ro-uc-ru/aicli/internal/scaffold"
	"github.com/ro-uc-ru/aicli/internal/tui"
)

// xterm is a getenv that reports a capable terminal.
func xterm(string) string { return "xterm-256color" }

// TestShowUnknownFeature checks the FR-007 error with and without a close
// listed feature.
//
// Covers: FR-007
func TestShowUnknownFeature(t *testing.T) {
	add := featuretest.NewTree(featuretest.NewMain(), featuretest.NewNoInputs())
	var out bytes.Buffer
	err := Show(context.Background(), add, []string{"featur-1"}, &bytes.Buffer{}, &out, xterm)
	want := "unknown feature \"featur-1\"\nDid you mean \"feature-1\"?\nRun `aicli add` to see the available features."
	if err == nil || err.Error() != want {
		t.Fatalf("error = %v, want %q", err, want)
	}
	if out.Len() != 0 {
		t.Fatalf("wrote %q", out.String())
	}

	err = Show(context.Background(), add, []string{"zzzzzzzz"}, &bytes.Buffer{}, &out, xterm)
	want = "unknown feature \"zzzzzzzz\"\nRun `aicli add` to see the available features."
	if err == nil || err.Error() != want {
		t.Fatalf("error = %v, want %q", err, want)
	}
}

// TestShowUnknownOnlyUnlisted checks that features that are not listed are
// never suggested, and that the error is the same with no listed feature.
//
// Covers: FR-007, EC-004
func TestShowUnknownOnlyUnlisted(t *testing.T) {
	add := featuretest.NewTree(featuretest.NewHidden(), featuretest.NewDeprecated())
	err := Show(context.Background(), add, []string{"feature-hidde"}, &bytes.Buffer{}, &bytes.Buffer{}, xterm)
	want := "unknown feature \"feature-hidde\"\nRun `aicli add` to see the available features."
	if err == nil || err.Error() != want {
		t.Fatalf("error = %v, want %q", err, want)
	}
	err = Show(context.Background(), featuretest.NewTree(), []string{"x"}, &bytes.Buffer{}, &bytes.Buffer{}, xterm)
	if err == nil || strings.Contains(err.Error(), "Did you mean") {
		t.Fatalf("error = %v", err)
	}
}

// TestShowStaticBuffers checks the static output for in-memory buffers.
//
// Covers: FR-005
func TestShowStaticBuffers(t *testing.T) {
	add := featuretest.NewTree(featuretest.Valid()...)
	var out bytes.Buffer
	if err := Show(context.Background(), add, nil, &bytes.Buffer{}, &out, xterm); err != nil {
		t.Fatal(err)
	}
	if out.String() != tui.RenderAddStatic(Features(add)) || strings.Contains(out.String(), "feature-hidden") {
		t.Fatalf("output = %q", out.String())
	}
}

// TestShowNoListedFeature checks the EC-003 text with no listed feature.
//
// Covers: EC-003
func TestShowNoListedFeature(t *testing.T) {
	var out bytes.Buffer
	if err := Show(context.Background(), featuretest.NewTree(), nil, &bytes.Buffer{}, &out, xterm); err != nil {
		t.Fatal(err)
	}
	want := "aicli › add\nAdd an AI/LLM feature to this project.\n\nNo features available.\n"
	if out.String() != want {
		t.Fatalf("output = %q", out.String())
	}
}

// TestShowMixedStreamsAreStatic checks that a pipe on one side and a terminal
// on the other gives static mode. The terminal side is /dev/tty; the subtest
// is skipped when the process has no controlling terminal.
//
// Covers: FR-005, EC-005
func TestShowMixedStreamsAreStatic(t *testing.T) {
	add := featuretest.NewTree(featuretest.NewMain())
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = r.Close(); _ = w.Close() })
	tty, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
	if err != nil {
		t.Skipf("no controlling terminal: %v", err)
	}
	t.Cleanup(func() { _ = tty.Close() })

	if err := Show(context.Background(), add, nil, tty, w, xterm); err != nil {
		t.Fatalf("pipe out: %v", err)
	}
	_ = w.Close()
	var got bytes.Buffer
	if _, err := got.ReadFrom(r); err != nil || got.String() != tui.RenderAddStatic(Features(add)) {
		t.Fatalf("pipe out gave %q (%v)", got.String(), err)
	}
	var out bytes.Buffer
	if err := Show(context.Background(), add, nil, r, &out, xterm); err != nil || out.Len() == 0 {
		t.Fatalf("pipe in: %q, %v", out.String(), err)
	}
}

// TestShowCancelled checks that a pre-cancelled context writes nothing and
// returns the cause.
//
// Covers: EC-020
func TestShowCancelled(t *testing.T) {
	cause := errors.New("signal")
	ctx, cancel := context.WithCancelCause(context.Background())
	cancel(cause)
	var out bytes.Buffer
	add := featuretest.NewTree(featuretest.NewMain())
	if err := Show(ctx, add, nil, &bytes.Buffer{}, &out, xterm); !errors.Is(err, cause) {
		t.Fatalf("error = %v, want cause", err)
	}
	if out.Len() != 0 {
		t.Fatalf("wrote %q", out.String())
	}
}

// failWriter fails every write.
type failWriter struct{}

// Write always fails. In: p ignored. Out: 0 and an error. No side effects.
func (failWriter) Write([]byte) (int, error) { return 0, errors.New("boom") }

// TestShowWriteError checks that a write failure is returned with context.
//
// Covers: FR-005
func TestShowWriteError(t *testing.T) {
	add := featuretest.NewTree(featuretest.NewMain())
	err := Show(context.Background(), add, nil, &bytes.Buffer{}, failWriter{}, xterm)
	if err == nil || !strings.Contains(err.Error(), "boom") {
		t.Fatalf("error = %v", err)
	}
}

// TestShowNoListedFeatureNeverInteractive checks that with no listed feature
// (only unlisted ones, or none) Show prints the same plain text as in static
// mode, with no ANSI or alternate-screen sequence, and never starts the
// interactive screens, even on a terminal.
//
// Covers: FR-005, EC-003
func TestShowNoListedFeatureNeverInteractive(t *testing.T) {
	old := runScreens
	runScreens = func(context.Context, []tui.Feature, io.Reader, io.Writer, bool) (tui.AddResult, error) {
		t.Error("the interactive screens must not start")
		return tui.AddResult{}, nil
	}
	t.Cleanup(func() { runScreens = old })
	want := "aicli › add\nAdd an AI/LLM feature to this project.\n\nNo features available.\n"
	for name, add := range map[string]*cobra.Command{
		"none":          featuretest.NewTree(),
		"only unlisted": featuretest.NewTree(featuretest.NewHidden(), featuretest.NewDeprecated()),
	} {
		var out bytes.Buffer
		if err := Show(context.Background(), add, nil, &bytes.Buffer{}, &out, xterm); err != nil {
			t.Fatal(err)
		}
		if out.String() != want || strings.ContainsRune(out.String(), '\x1b') {
			t.Errorf("%s: output = %q, want %q", name, out.String(), want)
		}
	}
	tty, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
	if err != nil {
		t.Skipf("no controlling terminal: %v", err)
	}
	t.Cleanup(func() { _ = tty.Close() })
	if !tui.IsInteractive(tty, tty, xterm) {
		t.Skip("/dev/tty is not a terminal here")
	}
	if err := Show(context.Background(), featuretest.NewTree(), nil, tty, tty, xterm); err != nil {
		t.Fatalf("on a terminal: %v", err)
	}
}

// TestShowSignalWhileScreensShown checks that a context cancelled with a
// *scaffold.SignalError while the screens are shown makes Show return that
// cause and print nothing, so that run exits with 128 plus the signal number.
//
// Covers: EC-020
func TestShowSignalWhileScreensShown(t *testing.T) {
	old := runScreens
	t.Cleanup(func() { runScreens = old })
	runScreens = func(ctx context.Context, _ []tui.Feature, _ io.Reader, _ io.Writer, _ bool) (tui.AddResult, error) {
		<-ctx.Done()
		return tui.AddResult{}, context.Cause(ctx)
	}
	add := featuretest.NewTree(featuretest.NewMain())
	ctx, cancel := context.WithCancelCause(project.WithRoot(context.Background(), t.TempDir()))
	go cancel(&scaffold.SignalError{Signal: syscall.SIGINT})
	var out bytes.Buffer
	err := runInteractive(ctx, add, Features(add), &bytes.Buffer{}, &out, xterm)
	var se *scaffold.SignalError
	if !errors.As(err, &se) || se.ExitCode() != 130 {
		t.Fatalf("err = %v, want the SignalError for SIGINT", err)
	}
	if out.Len() != 0 {
		t.Errorf("printed %q, want nothing", out.String())
	}
}
