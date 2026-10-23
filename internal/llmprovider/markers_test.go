package llmprovider_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ro-uc-ru/aicli/internal/gen"
	"github.com/ro-uc-ru/aicli/internal/llmprovider"
	"github.com/ro-uc-ru/aicli/internal/scaffold"
)

// TestSkeletonHasMarkers scaffolds a project and checks that the plan of
// every provider passes CheckPlan, so each marker exists exactly once.
//
// Covers: FR-015
func TestSkeletonHasMarkers(t *testing.T) {
	root := scaffoldProject(t)
	entries, err := llmprovider.Plan([]string{"openai", "claude", "gemini", "ollama"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := gen.CheckPlan(root, "llm-provider", gen.Plan{Entries: entries}); err != nil {
		t.Fatal(err)
	}
}

// TestOldProjectFailsCheck removes the router marker from a scaffolded
// project, as one generated before this spec lacks it, and checks the error.
//
// Covers: FR-015, EC-010
func TestOldProjectFailsCheck(t *testing.T) {
	root := scaffoldProject(t)
	p := filepath.Join(root, "app", "api", "router.py")
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	old := strings.ReplaceAll(string(b), "# aicli:router-imports\n", "")
	if err := os.WriteFile(p, []byte(old), 0o644); err != nil {
		t.Fatal(err)
	}
	entries, _ := llmprovider.Plan([]string{"openai"})
	_, err = gen.CheckPlan(root, "llm-provider", gen.Plan{Entries: entries})
	want := `cannot add llm-provider, app/api/router.py has no "# aicli:router-imports" marker`
	if err == nil || err.Error() != want {
		t.Fatalf("got %v, want %q", err, want)
	}
}

// scaffoldProject generates the base skeleton into a temp dir.
//
// In: t, the test. Out: the project root. Side effects: writes under a temp dir.
func scaffoldProject(t *testing.T) string {
	t.Helper()
	base := t.TempDir()
	if err := scaffold.Run(context.Background(), base, "demo"); err != nil {
		t.Fatal(err)
	}
	return filepath.Join(base, "demo")
}
