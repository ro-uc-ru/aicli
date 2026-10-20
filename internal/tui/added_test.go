package tui

import (
	"os"
	"strings"
	"testing"
)

// addedCase is one input of the RenderAdded golden tests.
type addedCase struct {
	name  string
	steps []string
}

var addedCases = []addedCase{
	{"added", nil},
	{"added-steps", []string{"Set OPENAI_API_KEY in .env", "uv sync"}},
}

// TestRenderAddedPlain checks the plain report, with and without next steps,
// against the golden files.
//
// In: t, the running test. Out: none.
// Side effects: with -update, rewrites testdata/added*.golden.
// Covers: FR-025.
func TestRenderAddedPlain(t *testing.T) {
	for _, tc := range addedCases {
		got := RenderAdded("feature-1", []string{"a/b.py", "c.txt"}, tc.steps, false)
		path := "testdata/" + tc.name + ".golden"
		if updating() {
			if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
				t.Fatal(err)
			}
		}
		want, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if got != string(want) {
			t.Fatalf("%s: plain output differs from golden:\n%s", tc.name, got)
		}
		if ansiRE.MatchString(got) {
			t.Fatalf("%s: plain output contains ANSI sequences", tc.name)
		}
	}
}

// TestRenderAddedStyled checks that the styled report carries ANSI and the
// "✓ " prefix, and that stripping both yields exactly the plain report.
//
// In: t, the running test. Out: none. Side effects: none.
// Covers: FR-025.
func TestRenderAddedStyled(t *testing.T) {
	for _, tc := range addedCases {
		paths := []string{"a/b.py", "c.txt"}
		got := RenderAdded("feature-1", paths, tc.steps, true)
		if !ansiRE.MatchString(got) || !strings.Contains(got, successPrefix+"Added") {
			t.Fatalf("%s: styled output lacks ANSI or prefix: %q", tc.name, got)
		}
		stripped := strings.Replace(ansiRE.ReplaceAllString(got, ""), successPrefix, "", 1)
		if want := RenderAdded("feature-1", paths, tc.steps, false); stripped != want {
			t.Fatalf("%s: stripped styled output differs:\n%q\n%q", tc.name, stripped, want)
		}
	}
}
