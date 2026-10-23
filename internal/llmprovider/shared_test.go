package llmprovider_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ro-uc-ru/aicli/internal/gen"
	"github.com/ro-uc-ru/aicli/internal/llmprovider"
)

// checkSharedGolden renders the shared entry for path and compares it with its
// golden file under testdata/shared, rewriting it with -update.
//
// In: t, the test; path, the project path of a shared entry; markers, the
// markers that must each appear exactly once. Out: the rendered content.
// Side effects: with -update, writes the golden file.
func checkSharedGolden(t *testing.T, path string, markers ...string) string {
	t.Helper()
	entries, err := llmprovider.Plan([]string{"openai"})
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if e.Action != gen.ActionShared || e.Path != path {
			continue
		}
		got, err := gen.RenderEntry(e)
		if err != nil {
			t.Fatal(err)
		}
		golden := filepath.Join("testdata", "shared", filepath.FromSlash(path)+".golden")
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
			t.Errorf("%s differs from its golden file", path)
		}
		for _, m := range markers {
			if n := strings.Count(string(got), m); n != 1 {
				t.Errorf("%s: %q appears %d times, want 1", path, m, n)
			}
		}
		return string(got)
	}
	t.Fatalf("no shared entry for %s", path)
	return ""
}

// TestBaseGolden renders app/ai/llm/base.py, compares it with its golden file,
// and checks that it holds # aicli:capabilities exactly once and imports no
// provider SDK.
//
// Covers: FR-016, FR-022
func TestBaseGolden(t *testing.T) {
	got := checkSharedGolden(t, "app/ai/llm/base.py", "# aicli:capabilities")
	for _, sdk := range []string{"openai", "anthropic", "google"} {
		for _, form := range []string{"import " + sdk, "from " + sdk} {
			if strings.Contains(got, form) {
				t.Errorf("base.py imports a provider SDK: %q", form)
			}
		}
	}
}

// TestRegistryGolden renders app/ai/llm/__init__.py, compares it with its
// golden file, and checks that it imports no provider SDK.
//
// Covers: FR-016, EC-017, EC-020, EC-027, EC-031
func TestRegistryGolden(t *testing.T) {
	got := checkSharedGolden(t, "app/ai/llm/__init__.py")
	for _, sdk := range []string{"openai", "anthropic", "google"} {
		for _, form := range []string{"import " + sdk, "from " + sdk} {
			if strings.Contains(got, form) {
				t.Errorf("__init__.py imports a provider SDK: %q", form)
			}
		}
	}
}

// TestRouteLayerGoldens renders the schemas, services and router templates,
// compares each with its golden file, and checks that each holds its marker
// exactly once.
//
// Covers: FR-018, FR-019, FR-020, FR-022, EC-019, EC-022
func TestRouteLayerGoldens(t *testing.T) {
	checkSharedGolden(t, "app/schemas/llm.py", "# aicli:schemas")
	checkSharedGolden(t, "app/services/llm.py", "# aicli:services")
	checkSharedGolden(t, "app/api/routers/llm.py", "# aicli:routes")
}

// TestProviderGoldens renders the module of every provider with all four
// chosen, compares each with its golden file, and checks that each holds
// # aicli:methods exactly once and imports no provider SDK at the top level.
//
// Covers: FR-016, FR-017, FR-022, EC-023, EC-026
func TestProviderGoldens(t *testing.T) {
	entries, err := llmprovider.Plan([]string{"openai", "claude", "gemini", "ollama"})
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"openai", "claude", "gemini", "ollama"} {
		path := "app/ai/llm/" + name + ".py"
		for _, e := range entries {
			if e.Path != path {
				continue
			}
			got, err := gen.RenderEntry(e)
			if err != nil {
				t.Fatal(err)
			}
			golden := filepath.Join("testdata", "provider", name+".py.golden")
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
				t.Errorf("%s differs from its golden file", path)
			}
			if n := strings.Count(string(got), "# aicli:methods"); n != 1 {
				t.Errorf("%s: # aicli:methods appears %d times, want 1", path, n)
			}
			for _, line := range strings.Split(string(got), "\n") {
				for _, sdk := range []string{"openai", "anthropic", "google"} {
					if line == "import "+sdk || strings.HasPrefix(line, "from "+sdk) {
						t.Errorf("%s: top-level SDK import %q", path, line)
					}
				}
			}
		}
	}
}

// TestRouterTestGolden renders tests/test_llm_router.py and compares it with
// its golden file.
//
// Covers: FR-018, FR-019, FR-020, FR-021, EC-017, EC-019, EC-020, EC-022, EC-024, EC-027, EC-031
func TestRouterTestGolden(t *testing.T) {
	checkSharedGolden(t, "tests/test_llm_router.py")
}

// TestProviderTestGoldens renders the test module of every provider with all
// four chosen and compares each with its golden file.
//
// Covers: FR-017, FR-021, EC-018, EC-023, EC-024, EC-026
func TestProviderTestGoldens(t *testing.T) {
	entries, err := llmprovider.Plan([]string{"openai", "claude", "gemini", "ollama"})
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"openai", "claude", "gemini", "ollama"} {
		path := "tests/test_llm_" + name + ".py"
		found := false
		for _, e := range entries {
			if e.Path != path {
				continue
			}
			found = true
			got, err := gen.RenderEntry(e)
			if err != nil {
				t.Fatal(err)
			}
			golden := filepath.Join("testdata", "providertest", name+".py.golden")
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
				t.Errorf("%s differs from its golden file", path)
			}
		}
		if !found {
			t.Errorf("no entry for %s", path)
		}
	}
}
