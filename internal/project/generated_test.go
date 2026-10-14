package project

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ro-uc-ru/aicli/internal/gen"
)

// TestGeneratedProjectIsRoot renders and writes the real skeleton with
// internal/gen into a temp dir and checks that it is a project root; then
// removes each marker in turn and checks that the check fails listing it.
//
// Covers: FR-008
func TestGeneratedProjectIsRoot(t *testing.T) {
	entries, err := gen.Render("myapp")
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	if err := gen.Write(gen.OSFS{}, dir, entries); err != nil {
		t.Fatal(err)
	}

	root, err := checkDir(dir)
	if err != nil || root != dir {
		t.Fatalf("generated project: root=%q err=%v, want %q and nil", root, err, dir)
	}

	for _, m := range Markers() {
		p := filepath.Join(dir, filepath.FromSlash(m))
		saved, err := os.ReadFile(p)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.Remove(p); err != nil {
			t.Fatal(err)
		}
		_, err = checkDir(dir)
		wantMissing(t, err, m)
		if err := os.WriteFile(p, saved, 0o644); err != nil {
			t.Fatal(err)
		}
	}
}
