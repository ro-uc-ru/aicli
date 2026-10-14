package root

import (
	"bytes"
	"context"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// TestStaticHomeLeavesWorkingDirUnchanged runs `aicli` with no arguments in a
// directory with a file and a subdirectory: afterwards the directory is
// identical (same entries, modes, sizes and times).
//
// In: t, the test. Out: none. Side effects: changes the working directory
// for the test.
// Covers: FR-014
func TestStaticHomeLeavesWorkingDirUnchanged(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "keep.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(dir, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(dir)
	before := snapshot(t, dir)
	var out, errb bytes.Buffer
	if code := Execute(context.Background(), []string{}, &out, &errb); code != 0 {
		t.Fatalf("code=%d stderr=%q", code, errb.String())
	}
	if out.Len() == 0 {
		t.Fatal("the static home screen printed nothing")
	}
	wantUnchanged(t, dir, before)
}

// TestHomeFilesImportNoProcessOrNetwork parses the Go files of internal/home
// and the home files of internal/tui and fails if any imports os/exec, net or
// a net/ package.
//
// In: t, the test. Out: none. Side effects: reads source files.
// Covers: FR-014
func TestHomeFilesImportNoProcessOrNetwork(t *testing.T) {
	var files []string
	entries, err := os.ReadDir(filepath.Join("..", "..", "internal", "home"))
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".go") {
			files = append(files, filepath.Join("..", "..", "internal", "home", e.Name()))
		}
	}
	for _, n := range []string{"banner", "detailview", "home", "homerun", "homestyles", "layout", "listview", "width"} {
		files = append(files, filepath.Join("..", "..", "internal", "tui", n+".go"))
	}
	if len(files) < 10 {
		t.Fatalf("too few files to check: %v", files)
	}
	for _, f := range files {
		parsed, err := parser.ParseFile(token.NewFileSet(), f, nil, parser.ImportsOnly)
		if err != nil {
			t.Fatal(err)
		}
		for _, imp := range parsed.Imports {
			path, err := strconv.Unquote(imp.Path.Value)
			if err != nil {
				t.Fatal(err)
			}
			if path == "os/exec" || path == "net" || strings.HasPrefix(path, "net/") {
				t.Errorf("%s imports %s", f, path)
			}
		}
	}
}
