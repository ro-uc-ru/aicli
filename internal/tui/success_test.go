package tui

import (
	"bytes"
	"flag"
	"os"
	"regexp"
	"strings"
	"testing"
)

// updating reports whether the golden files are to be regenerated instead of
// compared. The `-update` flag is registered by the teatest golden package,
// which the external flow tests import, so it is looked up, not redefined.
//
// In: none. Out: true when `-update` was given. Side effects: none.
func updating() bool {
	f := flag.Lookup("update")
	if f == nil {
		return false
	}
	g, ok := f.Value.(flag.Getter)
	if !ok {
		return false
	}
	b, _ := g.Get().(bool)
	return b
}

// ansiRE matches ANSI CSI escape sequences.
var ansiRE = regexp.MustCompile(`\x1b\[[0-9;]*[A-Za-z]`)

// TestRenderSuccessPlain checks the plain output against the golden file.
//
// In: t, the running test. Out: none.
// Side effects: with -update, rewrites testdata/success.golden.
// Covers: FR-015.
func TestRenderSuccessPlain(t *testing.T) {
	got := RenderSuccess("myapp", false)
	if updating() {
		if err := os.WriteFile("testdata/success.golden", []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile("testdata/success.golden")
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

// TestRenderSuccessStyled checks that the styled output carries ANSI and the
// "✓ " prefix, and that stripping both yields exactly the plain output.
//
// In: t, the running test. Out: none. Side effects: none.
// Covers: FR-015.
func TestRenderSuccessStyled(t *testing.T) {
	styled := RenderSuccess("myapp", true)
	if !ansiRE.MatchString(styled) {
		t.Fatal("styled output has no ANSI sequences")
	}
	stripped := strings.Replace(ansiRE.ReplaceAllString(styled, ""), successPrefix, "", 1)
	if want := RenderSuccess("myapp", false); stripped != want {
		t.Fatalf("stripped styled output differs from plain:\n%q\n%q", stripped, want)
	}
}

// TestShouldStyle covers the non-TTY writer and the NO_COLOR and TERM rules.
//
// In: t, the running test. Out: none. Side effects: opens a pty-less file.
// Covers: FR-015, FR-023.
func TestShouldStyle(t *testing.T) {
	env := func(m map[string]string) func(string) string {
		return func(k string) string { return m[k] }
	}
	if ShouldStyle(&bytes.Buffer{}, env(map[string]string{"TERM": "xterm"})) {
		t.Error("non-file writer must not be styled")
	}
	f, err := os.Open(os.DevNull)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()
	if ShouldStyle(f, env(map[string]string{"TERM": "xterm"})) {
		t.Error("/dev/null must not count as a TTY")
	}
	if ShouldStyle(f, env(map[string]string{"NO_COLOR": "1", "TERM": "xterm"})) {
		t.Error("NO_COLOR=1 must disable styling")
	}
	if ShouldStyle(f, env(map[string]string{"TERM": "dumb"})) {
		t.Error("TERM=dumb must disable styling")
	}
}
