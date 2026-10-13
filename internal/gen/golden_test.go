package gen

import (
	"bytes"
	"flag"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"testing"
)

// update regenerates the golden files instead of comparing against them.
var update = flag.Bool("update", false, "regenerate golden files")

// goldenDir is the directory holding the expected tree for the name "myapp".
const goldenDir = "testdata/golden"

// TestGoldenTree compares the whole generated tree of "myapp" with the golden
// files: same set of files and byte-identical content. With -update it
// rewrites the golden directory from the current output.
//
// In: t, the running test. Out: none.
// Side effects: with -update, removes and recreates testdata/golden.
// Covers: FR-007, FR-009, FR-010, FR-017, FR-018, FR-025.
func TestGoldenTree(t *testing.T) {
	entries, err := Render("myapp")
	if err != nil {
		t.Fatal(err)
	}
	if *update {
		if err := os.RemoveAll(goldenDir); err != nil {
			t.Fatal(err)
		}
		for _, e := range entries {
			if e.Kind != KindFile {
				continue
			}
			p := filepath.Join(goldenDir, filepath.FromSlash(e.Path))
			if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(p, e.Content, 0o644); err != nil {
				t.Fatal(err)
			}
		}
	}

	want := map[string][]byte{}
	for _, e := range entries {
		if e.Kind == KindFile {
			want[e.Path] = e.Content
		}
	}
	have := map[string]bool{}
	err = filepath.WalkDir(goldenDir, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		rel, err := filepath.Rel(goldenDir, p)
		if err != nil {
			return err
		}
		have[filepath.ToSlash(rel)] = true
		return nil
	})
	if err != nil {
		t.Fatalf("walk %s (run with -update to create): %v", goldenDir, err)
	}

	var paths []string
	for p := range want {
		paths = append(paths, p)
	}
	for p := range have {
		if _, ok := want[p]; !ok {
			paths = append(paths, p)
		}
	}
	sort.Strings(paths)
	for _, p := range paths {
		content, generated := want[p]
		switch {
		case !have[p]:
			t.Errorf("%s: missing golden file", p)
		case !generated:
			t.Errorf("%s: golden file not generated", p)
		default:
			got, err := os.ReadFile(filepath.Join(goldenDir, filepath.FromSlash(p)))
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(got, content) {
				t.Errorf("%s: content differs from golden", p)
			}
		}
	}
}
