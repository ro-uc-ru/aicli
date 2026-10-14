package tui_test

import (
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/ro-uc-ru/aicli/internal/home"
)

// dirEntry is one line of a directory snapshot.
type dirEntry struct {
	mode  fs.FileMode
	size  int64
	mtime time.Time
}

// dirSnapshot records mode, size and mtime of everything below dir.
//
// In: t, the test; dir. Out: the entries keyed by relative path. Side
// effects: none.
func dirSnapshot(t *testing.T, dir string) map[string]dirEntry {
	t.Helper()
	snap := map[string]dirEntry{}
	err := filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(dir, p)
		if err != nil {
			return err
		}
		snap[rel] = dirEntry{info.Mode(), info.Size(), info.ModTime()}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return snap
}

// TestFlowLeavesWorkingDirUnchanged runs an interactive teatest session that
// opens entries, scrolls and goes back, in a directory with a file: the
// directory is identical afterwards.
//
// In: t, the test. Out: none. Side effects: changes the working directory
// for the test.
// Covers: FR-014
func TestFlowLeavesWorkingDirUnchanged(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "keep.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Chdir(dir)
	before := dirSnapshot(t, dir)
	msgs := append(keys("down", "down", "down", "enter", "j", "end", "esc", "end", "enter"), key("q"))
	if v := drive(t, home.Entries(flowTree()), true, termW, termH, msgs...).View; v == "" {
		t.Fatal("the flow rendered nothing")
	}
	if after := dirSnapshot(t, dir); !reflect.DeepEqual(before, after) {
		t.Errorf("working directory changed:\nbefore %v\nafter  %v", before, after)
	}
}
