package root

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ro-uc-ru/aicli/internal/gen"
	"github.com/ro-uc-ru/aicli/internal/tui"
)

// parseReport splits the plain report of a successful run into its path
// lines and next steps, to render the same report styled.
//
// In: t, the test; out, the plain stdout. Out: the report lines and the
// next steps. Side effects: none.
func parseReport(t *testing.T, out string) ([]gen.ReportLine, []string) {
	t.Helper()
	head, tail, _ := strings.Cut(out, "\n\nAdded ")
	var lines []gen.ReportLine
	for _, l := range strings.Split(head, "\n") {
		verb, path, ok := strings.Cut(l, " ")
		if !ok {
			t.Fatalf("bad report line %q", l)
		}
		lines = append(lines, gen.ReportLine{Verb: gen.Verb(verb), Path: strings.TrimSpace(path)})
	}
	var steps []string
	if _, s, ok := strings.Cut(tail, "Next steps:\n"); ok {
		for _, l := range strings.Split(strings.TrimSuffix(s, "\n"), "\n") {
			steps = append(steps, strings.TrimPrefix(l, "  "))
		}
	}
	return lines, steps
}

// TestAddLLMProviderReports runs `aicli add llm-provider` through run in a
// scaffolded project and compares the report with a plain and a styled golden
// file: a first run with `openai ollama`; a later run with `claude` (skip and
// update lines); `ollama` after `openai` (pyproject.toml skipped); and a run
// where injection changes nothing. Stdout carries the report, stderr is empty,
// and the styled form stripped of ANSI sequences and of `✓ ` equals the plain
// one.
//
// In: t, the test; the goldens are testdata/add-llm-<name>.golden and
// add-llm-<name>-styled.golden. Out: none. Side effects: writes golden files
// with -update.
// Covers: FR-004, FR-011, FR-014, EC-008, EC-009, EC-011
func TestAddLLMProviderReports(t *testing.T) {
	p := "llm-provider"
	cases := []struct {
		name  string
		setup [][]string // calls that must succeed first
		prep  func(t *testing.T, dir string)
		args  []string
	}{
		{"first", nil, nil, []string{"add", p, "openai", "ollama"}},
		{"later", [][]string{{"add", p, "openai", "ollama"}}, nil, []string{"add", p, "claude"}},
		{"ollama-after-openai", [][]string{{"add", p, "openai"}}, nil, []string{"add", p, "ollama"}},
		{"nothing-injected", [][]string{{"add", p, "openai"}}, func(t *testing.T, dir string) {
			for _, f := range []string{"app/ai/llm/openai.py", "tests/test_llm_openai.py", "app/api/routers/llm.py"} {
				if err := os.Remove(filepath.Join(dir, filepath.FromSlash(f))); err != nil {
					t.Fatal(err)
				}
			}
		}, []string{"add", p, "openai"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			dir := scaffolded(t)
			for _, s := range c.setup {
				if code, _, e := runReal(t, dir, s...); code != 0 {
					t.Fatal(e)
				}
			}
			if c.prep != nil {
				c.prep(t, dir)
			}
			code, out, errb := runReal(t, dir, c.args...)
			if code != 0 || errb != "" {
				t.Fatalf("code=%d stderr=%q, want 0 and empty", code, errb)
			}
			if ansiSeq.MatchString(out) {
				t.Errorf("stdout is not plain: %q", out)
			}
			compareGolden(t, "add-llm-"+c.name+".golden", []byte(out))

			lines, steps := parseReport(t, out)
			styled := tui.RenderAdded(p, lines, steps, true)
			if !ansiSeq.MatchString(styled) {
				t.Fatalf("styled report has no ANSI sequence: %q", styled)
			}
			compareGolden(t, "add-llm-"+c.name+"-styled.golden", []byte(styled))
			stripped := strings.Replace(ansiSeq.ReplaceAllString(styled, ""), "✓ ", "", 1)
			if stripped != out {
				t.Errorf("styled stripped = %q, want %q", stripped, out)
			}
		})
	}
}

// TestAddLLMProviderOrderIndependent runs `ollama openai` and `openai ollama`
// in two fresh projects and checks that stdout is identical.
//
// In: t, the test. Out: none. Side effects: writes in temp directories.
// Covers: FR-004
func TestAddLLMProviderOrderIndependent(t *testing.T) {
	var outs []string
	for _, order := range [][]string{{"openai", "ollama"}, {"ollama", "openai"}} {
		code, out, errb := runReal(t, scaffolded(t), append([]string{"add", "llm-provider"}, order...)...)
		if code != 0 || errb != "" {
			t.Fatalf("code=%d stderr=%q", code, errb)
		}
		outs = append(outs, out)
	}
	if outs[0] != outs[1] {
		t.Errorf("outputs differ:\n%s\n---\n%s", outs[0], outs[1])
	}
}
