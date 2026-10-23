package llmprovider_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ro-uc-ru/aicli/internal/gen"
	"github.com/ro-uc-ru/aicli/internal/llmprovider"
)

// injectedFiles are the files of the base skeleton that the feature injects
// into, relative to the project root with "/" separators.
var injectedFiles = []string{"pyproject.toml", ".env.example", "app/api/router.py"}

// markerFiles lists the files that must hold each marker of FR-022 exactly
// once.
//
// In: providers, the providers added. Out: marker to project paths.
func markerFiles(providers []string) map[string][]string {
	m := map[string][]string{
		"# aicli:capabilities": {"app/ai/llm/base.py"},
		"# aicli:schemas":      {"app/schemas/llm.py"},
		"# aicli:services":     {"app/services/llm.py"},
		"# aicli:routes":       {"app/api/routers/llm.py"},
	}
	for _, p := range providers {
		m["# aicli:methods"] = append(m["# aicli:methods"], "app/ai/llm/"+p+".py")
	}
	return m
}

// addProviders applies the plan of the given providers to the project.
//
// In: t, the test; root, the project root; providers, the choices to add.
// Out: none. Side effects: writes under root; fails the test on error.
func addProviders(t *testing.T, root string, providers ...string) {
	t.Helper()
	entries, err := llmprovider.Plan(providers)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := gen.ApplyPlan(context.Background(), gen.OSFeatureFS{}, root, "llm-provider", gen.Plan{Entries: entries}); err != nil {
		t.Fatal(err)
	}
}

// TestInjectedFilesGoldens adds `openai ollama` and then `claude` to a
// scaffolded project and compares pyproject.toml, .env.example and
// app/api/router.py with their golden files after each run.
//
// Covers: FR-013, FR-015, EC-008
func TestInjectedFilesGoldens(t *testing.T) {
	root := scaffoldProject(t)
	steps := []struct {
		name      string
		providers []string
	}{
		{"first", []string{"openai", "ollama"}},
		{"second", []string{"claude"}},
	}
	for _, s := range steps {
		addProviders(t, root, s.providers...)
		for _, f := range injectedFiles {
			got, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(f)))
			if err != nil {
				t.Fatal(err)
			}
			golden := filepath.Join("testdata", "injected", s.name, filepath.FromSlash(f)+".golden")
			if *update {
				if err := os.MkdirAll(filepath.Dir(golden), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(golden, got, 0o644); err != nil {
					t.Fatal(err)
				}
			}
			want, err := os.ReadFile(golden)
			if err != nil {
				t.Fatal(err)
			}
			if string(want) != string(got) {
				t.Errorf("%s after %s run differs from its golden file", f, s.name)
			}
		}
	}
	b, err := os.ReadFile(filepath.Join(root, "pyproject.toml"))
	if err != nil {
		t.Fatal(err)
	}
	if n := strings.Count(string(b), `"openai`); n != 1 {
		t.Errorf("openai SDK appears %d times in pyproject.toml, want 1", n)
	}
}

// TestMarkersExactlyOnce checks that every file of FR-022 holds its marker
// exactly once, in a project with all four providers and in one with a single
// provider, none of them having injected lines at those markers.
//
// Covers: FR-022
func TestMarkersExactlyOnce(t *testing.T) {
	cases := [][]string{{"openai", "claude", "gemini", "ollama"}, {"gemini"}}
	for _, providers := range cases {
		root := scaffoldProject(t)
		addProviders(t, root, providers...)
		for marker, files := range markerFiles(providers) {
			for _, f := range files {
				b, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(f)))
				if err != nil {
					t.Fatal(err)
				}
				if n := strings.Count(string(b), marker); n != 1 {
					t.Errorf("%v: %s has %q %d times, want 1", providers, f, marker, n)
				}
			}
		}
	}
}

// TestInjectKeepsCRLFAndIndent converts the injection targets of a scaffolded
// project to \r\n line endings and checks that the inserted lines use the
// same ending, that the indented TOML marker keeps its indentation, and that
// no bare \n appears.
//
// Covers: EC-012
func TestInjectKeepsCRLFAndIndent(t *testing.T) {
	root := scaffoldProject(t)
	for _, f := range injectedFiles {
		p := filepath.Join(root, filepath.FromSlash(f))
		b, err := os.ReadFile(p)
		if err != nil {
			t.Fatal(err)
		}
		crlf := strings.ReplaceAll(string(b), "\n", "\r\n")
		if err := os.WriteFile(p, []byte(crlf), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	addProviders(t, root, "openai", "claude")
	for _, f := range injectedFiles {
		b, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(f)))
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(strings.ReplaceAll(string(b), "\r\n", ""), "\n") {
			t.Errorf("%s has a bare \\n after injection", f)
		}
	}
	b, _ := os.ReadFile(filepath.Join(root, "pyproject.toml"))
	for _, line := range strings.Split(string(b), "\r\n") {
		if strings.Contains(line, `"anthropic`) && !strings.HasPrefix(line, "    ") {
			t.Errorf("injected dependency line lost the marker indentation: %q", line)
		}
	}
}
