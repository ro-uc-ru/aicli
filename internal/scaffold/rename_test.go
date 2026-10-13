//go:build linux || darwin

package scaffold

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// TestRenameNoReplaceExistingEmptyDir checks that renaming onto an existing
// empty directory fails with ErrDestinationExists and changes nothing.
// In: t, the running test. Out: none. Side effects: uses a temp dir.
// Covers: EC-010.
func TestRenameNoReplaceExistingEmptyDir(t *testing.T) {
	base := t.TempDir()
	src, dst := filepath.Join(base, "src"), filepath.Join(base, "dst")
	for _, d := range []string{src, dst} {
		if err := os.Mkdir(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(src, "f"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	err := renameNoReplace(src, dst)
	if !errors.Is(err, ErrDestinationExists) {
		t.Fatalf("got %v, want ErrDestinationExists", err)
	}
	if _, err := os.Stat(filepath.Join(src, "f")); err != nil {
		t.Errorf("source changed: %v", err)
	}
	if entries, err := os.ReadDir(dst); err != nil || len(entries) != 0 {
		t.Errorf("destination touched: %v, %v", entries, err)
	}
}

// TestRenameNoReplaceMissingDestination checks that renaming onto a missing
// path succeeds.
// In: t, the running test. Out: none. Side effects: uses a temp dir.
// Covers: EC-010.
func TestRenameNoReplaceMissingDestination(t *testing.T) {
	base := t.TempDir()
	src, dst := filepath.Join(base, "src"), filepath.Join(base, "dst")
	if err := os.Mkdir(src, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := renameNoReplace(src, dst); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(dst); err != nil {
		t.Errorf("destination missing: %v", err)
	}
	if _, err := os.Lstat(src); !os.IsNotExist(err) {
		t.Errorf("source still present: %v", err)
	}
}
