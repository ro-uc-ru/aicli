package home

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/ro-uc-ru/aicli/internal/tui"
)

// failWriter is a writer that always fails.
type failWriter struct{ err error }

// Write always returns the configured error.
func (f failWriter) Write([]byte) (int, error) { return 0, f.err }

// TestShowStatic checks that static Show with buffers writes exactly the
// static text and no ESC byte.
//
// Covers: FR-012, FR-014, EC-009
func TestShowStatic(t *testing.T) {
	root := newTree()
	var out bytes.Buffer
	err := Show(context.Background(), root, strings.NewReader(""), &out, func(string) string { return "" })
	if err != nil {
		t.Fatal(err)
	}
	if want := tui.RenderHomeStatic(Entries(root)); out.String() != want {
		t.Errorf("output = %q, want %q", out.String(), want)
	}
	if strings.ContainsRune(out.String(), '\x1b') {
		t.Error("output contains ESC byte")
	}
}

// TestShowCancelled checks that a pre-cancelled context returns its cause
// and writes nothing.
//
// Covers: FR-012, EC-014
func TestShowCancelled(t *testing.T) {
	cause := errors.New("interrupted")
	ctx, cancel := context.WithCancelCause(context.Background())
	cancel(cause)
	var out bytes.Buffer
	err := Show(ctx, newTree(), strings.NewReader(""), &out, func(string) string { return "" })
	if !errors.Is(err, cause) {
		t.Errorf("err = %v, want cause %v", err, cause)
	}
	if out.Len() != 0 {
		t.Errorf("wrote %d bytes, want 0", out.Len())
	}
}

// TestShowWriteError checks that a failing writer's error is returned with
// context.
//
// Covers: FR-012, EC-009
func TestShowWriteError(t *testing.T) {
	boom := errors.New("boom")
	err := Show(context.Background(), newTree(), strings.NewReader(""), failWriter{boom}, func(string) string { return "" })
	if !errors.Is(err, boom) || err.Error() == boom.Error() {
		t.Errorf("err = %v, want wrapped boom", err)
	}
}
