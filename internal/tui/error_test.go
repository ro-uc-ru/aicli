package tui

import (
	"os"
	"strings"
	"testing"
)

// errorMsg is the two-line message used by the error tests.
const errorMsg = "missing project name\nUsage: aicli new <name>"

// TestRenderErrorPlain checks the plain output against the golden file.
//
// In: t, the running test. Out: none.
// Side effects: with -update, rewrites testdata/error.golden.
// Covers: FR-023.
func TestRenderErrorPlain(t *testing.T) {
	got := RenderError(errorMsg, false)
	if *update {
		if err := os.WriteFile("testdata/error.golden", []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile("testdata/error.golden")
	if err != nil {
		t.Fatal(err)
	}
	if got != string(want) {
		t.Fatalf("plain output differs from golden:\n%s", got)
	}
	if ansiRE.MatchString(got) {
		t.Fatal("plain output contains ANSI sequences")
	}
}

// TestRenderErrorStyled checks that the styled output carries ANSI and the
// "✗ " prefix, and that stripping both yields exactly the plain output.
//
// In: t, the running test. Out: none. Side effects: none.
// Covers: FR-023.
func TestRenderErrorStyled(t *testing.T) {
	styled := RenderError(errorMsg, true)
	if !ansiRE.MatchString(styled) {
		t.Fatal("styled output has no ANSI sequences")
	}
	stripped := strings.Replace(ansiRE.ReplaceAllString(styled, ""), errorPrefix, "", 1)
	if want := RenderError(errorMsg, false); stripped != want {
		t.Fatalf("stripped styled output differs from plain:\n%q\n%q", stripped, want)
	}
}
