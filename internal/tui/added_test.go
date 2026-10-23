package tui

import (
	"os"
	"strings"
	"testing"

	"github.com/ro-uc-ru/aicli/internal/gen"
)

// createLines builds create report lines for the given paths.
func createLines(paths ...string) []gen.ReportLine {
	var out []gen.ReportLine
	for _, p := range paths {
		out = append(out, gen.ReportLine{Verb: gen.VerbCreate, Path: p})
	}
	return out
}

// allVerbs is a report with the three verbs.
var allVerbs = []gen.ReportLine{
	{Verb: gen.VerbCreate, Path: "app/a.py"},
	{Verb: gen.VerbSkip, Path: "pyproject.toml"},
	{Verb: gen.VerbUpdate, Path: ".env.example"},
}

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
		got := RenderAdded("feature-1", createLines("a/b.py", "c.txt"), tc.steps, false)
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
		paths := createLines("a/b.py", "c.txt")
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

// TestRenderAddedVerbs checks the three verbs against a golden file, plain
// and styled: the verbs are padded to 6 characters plus two spaces, the colours
// are green, 117 and 245, and the styled text stripped of ANSI and of the
// prefix equals the plain one.
//
// In: t, the running test. Out: none.
// Side effects: with -update, rewrites testdata/added-verbs.golden.
// Covers: FR-011.
func TestRenderAddedVerbs(t *testing.T) {
	got := RenderAdded("feature-1", allVerbs, []string{"uv sync"}, false)
	path := "testdata/added-verbs.golden"
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
		t.Fatalf("plain output differs from golden:\n%s", got)
	}
	for _, line := range []string{"create  app/a.py", "skip    pyproject.toml", "update  .env.example"} {
		if !strings.Contains(got, line+"\n") {
			t.Fatalf("missing line %q in:\n%s", line, got)
		}
	}
	styled := RenderAdded("feature-1", allVerbs, []string{"uv sync"}, true)
	for _, c := range []string{"[32mcreate", "38;5;117mupdate", "38;5;245mskip"} {
		if !strings.Contains(styled, c) {
			t.Fatalf("styled output lacks colour %s: %q", c, styled)
		}
	}
	stripped := strings.Replace(ansiRE.ReplaceAllString(styled, ""), successPrefix, "", 1)
	if stripped != got {
		t.Fatalf("stripped styled output differs:\n%q\n%q", stripped, got)
	}
}
