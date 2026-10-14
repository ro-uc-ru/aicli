package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// goldenHome compares got with testdata/<name>, rewriting it with -update.
//
// In: t, the test; name, the golden file name; got, the text to compare.
// Out: none. Side effects: with -update, rewrites the golden file.
func goldenHome(t *testing.T, name, got string) {
	t.Helper()
	path := filepath.Join("testdata", name)
	if updating() {
		if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read golden (run with -update to create): %v", err)
	}
	if got != string(want) {
		t.Errorf("output = %q, want %q", got, want)
	}
	if strings.Contains(got, "\x1b") {
		t.Error("output contains an ESC byte")
	}
}

// TestRenderHomeStatic checks the static output for a fixed list with a
// nested path, an entry without a short description, and CJK and emoji
// paths.
//
// Covers: FR-002, FR-003, FR-012, EC-005, EC-010
func TestRenderHomeStatic(t *testing.T) {
	entries := []Entry{
		{Path: "new", Short: "Create a new aicli project"},
		{Path: "db", Short: "Database migrations"},
		{Path: "db migrate", Short: "Apply migrations"},
		{Path: "nodesc"},
		{Path: "日本語", Short: "CJK path"},
		{Path: "🚀go", Short: "Emoji path"},
	}
	goldenHome(t, "home-static.golden", RenderHomeStatic(entries))
}

// TestRenderHomeStaticEmpty checks the static output with no entries.
//
// Covers: FR-012, EC-012
func TestRenderHomeStaticEmpty(t *testing.T) {
	goldenHome(t, "home-static-empty.golden", RenderHomeStatic(nil))
}
