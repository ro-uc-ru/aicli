package add

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// snapshot records every path below root with its mode and, for a file, its
// content.
//
// In: t, the test; root, the directory. Out: a map from slash path to
// "mode:content". Side effects: none.
func snapshot(t *testing.T, root string) map[string]string {
	t.Helper()
	snap := map[string]string{}
	err := filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(root, p)
		content := ""
		if !d.IsDir() {
			b, err := os.ReadFile(p)
			if err != nil {
				return err
			}
			content = string(b)
		}
		snap[filepath.ToSlash(rel)] = info.Mode().String() + ":" + content
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return snap
}

// TestFlowQuitLeavesDirectoryUnchanged shows the confirmation of the real
// feature (which reads the file system to check the plan) in a project root
// with existing files, quits with Ctrl+C, and then does the same ending with
// `n`: the snapshot of the directory (paths, modes and contents) is identical
// before and after, and the working directory of the process is unchanged.
//
// Covers: FR-020
func TestFlowQuitLeavesDirectoryUnchanged(t *testing.T) {
	for _, end := range []string{"\x03", "n"} {
		f, root := startFlow(t)
		if err := os.MkdirAll(filepath.Join(root, "features"), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, "features", "other.txt"), []byte("keep\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, "pyproject.toml"), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
		wd, err := os.Getwd()
		if err != nil {
			t.Fatal(err)
		}
		before := snapshot(t, root)
		f.fillToConfirm("name")
		f.send(end)
		if err := f.finish(); err != nil {
			t.Fatal(err)
		}
		after := snapshot(t, root)
		if len(after) != len(before) {
			t.Errorf("ending with %q: before %v, after %v", end, before, after)
		}
		for p, v := range before {
			if after[p] != v {
				t.Errorf("ending with %q: %s changed from %q to %q", end, p, v, after[p])
			}
		}
		if now, _ := os.Getwd(); now != wd {
			t.Errorf("working directory changed from %s to %s", wd, now)
		}
	}
}

// forbiddenImports are the packages the selector, the form and the
// confirmation must not import: no process, no network.
var forbiddenImports = []string{"os/exec", "net", "net/http"}

// addScreenFiles are the non-test files of internal/tui that make the add
// screens, relative to internal/tui.
var addScreenFiles = []string{
	"addrun.go", "addstatic.go", "addstyles.go", "added.go", "choicefield.go",
	"confirm.go", "confirmlayout.go", "feature.go", "form.go", "formlayout.go",
	"selector.go", "textfield.go",
}

// TestAddScreensImportNoProcessOrNetwork parses the Go files of internal/add
// and of the add screens in internal/tui and fails on an import of os/exec,
// net or net/http (or any package below net).
//
// Covers: FR-020
func TestAddScreensImportNoProcessOrNetwork(t *testing.T) {
	var files []string
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if n := e.Name(); strings.HasSuffix(n, ".go") && !strings.HasSuffix(n, "_test.go") {
			files = append(files, n)
		}
	}
	if len(files) == 0 {
		t.Fatal("no Go file found in internal/add")
	}
	for _, n := range addScreenFiles {
		files = append(files, filepath.Join("..", "tui", n))
	}
	fset := token.NewFileSet()
	for _, f := range files {
		file, err := parser.ParseFile(fset, f, nil, parser.ImportsOnly)
		if err != nil {
			t.Errorf("%s: %v", f, err)
			continue
		}
		for _, imp := range file.Imports {
			path, err := strconv.Unquote(imp.Path.Value)
			if err != nil {
				t.Fatal(err)
			}
			for _, bad := range forbiddenImports {
				if path == bad || strings.HasPrefix(path, "net/") {
					t.Errorf("%s imports %s", f, path)
				}
			}
		}
	}
}
