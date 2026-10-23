package root

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ro-uc-ru/aicli/internal/scaffold"
	"github.com/ro-uc-ru/aicli/internal/tui"
)

// runReal runs the real command tree with dir as the working directory.
//
// In: t, the test; dir, the project root; args, the arguments.
// Out: the exit code, stdout and stderr. Side effects: as run.
func runReal(t *testing.T, dir string, args ...string) (int, string, string) {
	t.Helper()
	cmd := newRoot(".", func() (string, error) { return dir, nil })
	var out, errb bytes.Buffer
	code := run(context.Background(), cmd, args, &out, &errb)
	return code, out.String(), errb.String()
}

// scaffolded generates a real project skeleton in a temp dir.
//
// In: t, the test. Out: the project root. Side effects: writes files.
func scaffolded(t *testing.T) string {
	t.Helper()
	base := t.TempDir()
	if err := scaffold.Run(context.Background(), base, "demo"); err != nil {
		t.Fatal(err)
	}
	return filepath.Join(base, "demo")
}

// editFile replaces the content of a project file.
//
// In: t, the test; dir, the root; rel, the "/" path; fn, the edit.
// Out: none. Side effects: rewrites the file.
func editFile(t *testing.T, dir, rel string, fn func(string) string) {
	t.Helper()
	p := filepath.Join(dir, filepath.FromSlash(rel))
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(fn(string(b))), 0o644); err != nil {
		t.Fatal(err)
	}
}

// TestAddLLMProviderErrors runs `aicli add llm-provider` through run in a
// scaffolded project for each failing case and compares stderr with a golden
// file. Each fails with exit code 1, empty stdout and nothing created or
// modified; its styled form, stripped of ANSI sequences and of the `✗ `
// prefix, equals the plain one.
//
// In: t, the test; the golden files are testdata/add-llm-err-<name>.golden.
// Out: none. Side effects: writes golden files with -update.
// Covers: FR-002, FR-003, FR-010, EC-001, EC-002, EC-003, EC-004, EC-010, EC-014
func TestAddLLMProviderErrors(t *testing.T) {
	p := "llm-provider"
	cases := []struct {
		name  string
		setup func(t *testing.T, dir string)
		args  []string
	}{
		{"missing", nil, []string{"add", p}},
		{"repeated", nil, []string{"add", p, "openai", "openai"}},
		{"repeated-trimmed", nil, []string{"add", p, " openai", "openai"}},
		{"case", nil, []string{"add", p, "OpenAI"}},
		{"vendor", nil, []string{"add", p, "anthropic"}},
		{"added", func(t *testing.T, dir string) {
			if code, _, e := runReal(t, dir, "add", p, "openai"); code != 0 {
				t.Fatal(e)
			}
		}, []string{"add", p, "openai", "claude"}},
		{"fallback-exists", func(t *testing.T, dir string) {
			f := filepath.Join(dir, "tests", "test_llm_claude.py")
			if err := os.WriteFile(f, []byte("x"), 0o644); err != nil {
				t.Fatal(err)
			}
		}, []string{"add", p, "claude"}},
		{"target-missing", func(t *testing.T, dir string) {
			if err := os.Remove(filepath.Join(dir, ".env.example")); err != nil {
				t.Fatal(err)
			}
		}, []string{"add", p, "openai"}},
		{"target-not-file", func(t *testing.T, dir string) {
			f := filepath.Join(dir, ".env.example")
			if err := os.Remove(f); err != nil {
				t.Fatal(err)
			}
			if err := os.Mkdir(f, 0o755); err != nil {
				t.Fatal(err)
			}
		}, []string{"add", p, "openai"}},
		{"marker-missing", func(t *testing.T, dir string) {
			editFile(t, dir, "app/api/router.py", func(s string) string {
				return strings.ReplaceAll(s, "# aicli:router-imports\n", "")
			})
		}, []string{"add", p, "openai"}},
		{"marker-repeated", func(t *testing.T, dir string) {
			editFile(t, dir, ".env.example", func(s string) string {
				return s + "# aicli:env\n"
			})
		}, []string{"add", p, "openai"}},
		{"target-link-outside", func(t *testing.T, dir string) {
			out := filepath.Join(t.TempDir(), "outside")
			if err := os.WriteFile(out, []byte("# aicli:env\n"), 0o644); err != nil {
				t.Fatal(err)
			}
			f := filepath.Join(dir, ".env.example")
			if err := os.Remove(f); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(out, f); err != nil {
				t.Fatal(err)
			}
		}, []string{"add", p, "openai"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			dir := scaffolded(t)
			if c.setup != nil {
				c.setup(t, dir)
			}
			before := snapshot(t, dir)
			code, out, errb := runReal(t, dir, c.args...)
			if code != 1 || out != "" {
				t.Fatalf("code=%d stdout=%q, want 1 and empty", code, out)
			}
			if !strings.HasPrefix(errb, "Error: ") || ansiSeq.MatchString(errb) {
				t.Errorf("stderr is not plain: %q", errb)
			}
			compareGolden(t, "add-llm-err-"+c.name+".golden", []byte(errb))
			wantUnchanged(t, dir, before)

			msg := strings.TrimSuffix(strings.TrimPrefix(errb, "Error: "), "\n")
			msg = strings.ReplaceAll(msg, "\n  ", "\n")
			styled := tui.RenderError(msg, true)
			if !ansiSeq.MatchString(styled) {
				t.Fatalf("styled error has no ANSI sequence: %q", styled)
			}
			stripped := strings.Replace(ansiSeq.ReplaceAllString(styled, ""), "✗ ", "", 1)
			if stripped != errb {
				t.Errorf("styled stripped = %q, want %q", stripped, errb)
			}
		})
	}
}
