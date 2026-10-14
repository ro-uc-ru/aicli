package tui

import (
	"bytes"
	"os"
	"testing"
)

// TestIsInteractiveFalse checks that IsInteractive is false for in-memory
// buffers, for pipes on one or both sides, and with TERM=dumb.
//
// Covers: FR-005, EC-009
func TestIsInteractiveFalse(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = r.Close(); _ = w.Close() })
	r2, w2, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = r2.Close(); _ = w2.Close() })

	xterm := func(string) string { return "xterm-256color" }
	dumb := func(string) string { return "dumb" }
	cases := map[string]func() bool{
		"buffers":         func() bool { return IsInteractive(&bytes.Buffer{}, &bytes.Buffer{}, xterm) },
		"pipe in only":    func() bool { return IsInteractive(r, &bytes.Buffer{}, xterm) },
		"pipe out only":   func() bool { return IsInteractive(&bytes.Buffer{}, w, xterm) },
		"pipes both":      func() bool { return IsInteractive(r2, w2, xterm) },
		"dumb with pipes": func() bool { return IsInteractive(r2, w2, dumb) },
		"dumb buffers":    func() bool { return IsInteractive(&bytes.Buffer{}, &bytes.Buffer{}, dumb) },
	}
	for name, f := range cases {
		t.Run(name, func(t *testing.T) {
			if f() {
				t.Error("IsInteractive = true, want false")
			}
		})
	}
}
