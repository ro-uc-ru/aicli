package llmprovider_test

import (
	"flag"
	"os"
	"path/filepath"
	"testing"

	"github.com/ro-uc-ru/aicli/internal/gen"
	"github.com/ro-uc-ru/aicli/internal/llmprovider"
)

var update = flag.Bool("update", false, "regenerate golden files")

// TestInjectGoldens renders every injected entry of a plan with all four
// providers and compares each rendered block with its golden file.
//
// Covers: FR-008, FR-013
func TestInjectGoldens(t *testing.T) {
	entries, err := llmprovider.Plan([]string{"openai", "claude", "gemini", "ollama"})
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]int{}
	for _, e := range entries {
		if e.Action != gen.ActionInject {
			continue
		}
		got, err := gen.RenderEntry(e)
		if err != nil {
			t.Fatal(err)
		}
		name := e.Marker
		d, _ := e.Data.(llmprovider.Data)
		if d.SDK != "" {
			name += "-" + d.SDK
		} else if d.Line != "" {
			seen[name]++
			name += "-" + string(rune('a'+seen[name]-1))
		}
		path := filepath.Join("testdata", "inject", name+".golden")
		if *update {
			if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, got, 0o644); err != nil {
				t.Fatal(err)
			}
		}
		want, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if string(want) != string(got) {
			t.Errorf("%s: got %q, want %q", name, got, want)
		}
	}
}
