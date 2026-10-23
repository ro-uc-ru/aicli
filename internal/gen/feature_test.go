package gen

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"syscall"
	"testing"
	"testing/fstest"
)

// TestValidateFeatureFilesRejects checks every rejected path form.
//
// Covers: FR-022, EC-021, EC-022
func TestValidateFeatureFilesRejects(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(root, "escape")); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "afile"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name  string
		paths []string
		want  string
	}{
		{"empty list", nil, "demo declares no files"},
		{"empty path", []string{""}, "empty"},
		{"absolute", []string{"/etc/x"}, "relative"},
		{"backslash", []string{`a\b`}, "separator"},
		{"leading dot", []string{"./a"}, "not allowed"},
		{"dot component", []string{"a/./b"}, "not allowed"},
		{"dotdot component", []string{"a/../b"}, "not allowed"},
		{"leading dotdot", []string{"../b"}, "not allowed"},
		{"trailing slash", []string{"a/"}, "empty component"},
		{"doubled slash", []string{"a//b"}, "empty component"},
		{"duplicate", []string{"a/b", "c", "a/b"}, "more than once"},
		{"symlink outside", []string{"escape/x.py"}, "outside"},
		{"parent is file", []string{"afile/x.py"}, "not a directory"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateFeatureFiles(root, "demo", tt.paths)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("got %v, want error containing %q", err, tt.want)
			}
			for _, p := range tt.paths {
				if p != "" && tt.name != "empty list" && tt.name != "duplicate" && !strings.Contains(err.Error(), fmt.Sprintf("%q", p)) {
					t.Errorf("error %q does not name path %q", err, p)
				}
			}
		})
	}
}

// TestValidateFeatureFilesAccepts checks valid paths, including missing parents.
//
// Covers: FR-022
func TestValidateFeatureFilesAccepts(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "app"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := ValidateFeatureFiles(root, "demo", []string{"app/x.py", "new/deep/y.py", "z.txt"}); err != nil {
		t.Fatal(err)
	}
}

// TestValidateFeatureFilesRootViaSymlink checks a root reached through a
// symbolic link, and that an inside symlinked directory is still accepted.
//
// Covers: FR-022, EC-022
func TestValidateFeatureFilesRootViaSymlink(t *testing.T) {
	real := t.TempDir()
	if err := os.Mkdir(filepath.Join(real, "app"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(real, "app"), filepath.Join(real, "alias")); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(t.TempDir(), "link")
	if err := os.Symlink(real, link); err != nil {
		t.Fatal(err)
	}
	if err := ValidateFeatureFiles(link, "demo", []string{"app/x.py", "alias/y.py"}); err != nil {
		t.Fatal(err)
	}
}

// TestCheckFeatureFilesExisting checks that an existing file, a directory and
// a broken link are all reported in list order, and that nothing is written.
//
// Covers: FR-023, FR-018
func TestCheckFeatureFilesExisting(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "a.py"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(root, "dir"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(root, "nowhere"), filepath.Join(root, "broken")); err != nil {
		t.Fatal(err)
	}
	before := listTree(t, root)
	err := CheckFeatureFiles(root, "demo", []string{"broken", "free.py", "a.py", "dir"})
	want := "cannot add demo, these files already exist:\nbroken\na.py\ndir"
	if err == nil || err.Error() != want {
		t.Fatalf("got %v, want %q", err, want)
	}
	if after := listTree(t, root); !reflect.DeepEqual(before, after) {
		t.Errorf("tree changed: %v -> %v", before, after)
	}
}

// TestCheckFeatureFilesValidation checks that the validation errors of
// ValidateFeatureFiles surface and that free paths pass.
//
// Covers: FR-023, FR-018
func TestCheckFeatureFilesValidation(t *testing.T) {
	root := t.TempDir()
	if err := CheckFeatureFiles(root, "demo", nil); err == nil || err.Error() != "demo declares no files" {
		t.Errorf("empty list: got %v", err)
	}
	if err := CheckFeatureFiles(root, "demo", []string{"a/../b"}); err == nil || !strings.Contains(err.Error(), "not allowed") {
		t.Errorf("bad path: got %v", err)
	}
	if err := CheckFeatureFiles(root, "demo", []string{"a/b.py", "c.py"}); err != nil {
		t.Errorf("free paths: got %v", err)
	}
}

// listTree lists the entries directly under dir with their Lstat modes, without
// following links.
// In: t, the running test; dir, the directory. Out: name -> mode string.
// Side effects: fails the test on read errors.
func listTree(t *testing.T, dir string) map[string]string {
	t.Helper()
	ents, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	out := map[string]string{}
	for _, e := range ents {
		info, err := os.Lstat(filepath.Join(dir, e.Name()))
		if err != nil {
			t.Fatal(err)
		}
		out[e.Name()] = info.Mode().String()
	}
	return out
}

// featureTemplates is the template source of the CreateFeatureFiles tests.
var featureTemplates = fstest.MapFS{
	"chat.tmpl":   {Data: []byte("name={{.Name}}\nkind={{.Kind}}\n")},
	"readme.tmpl": {Data: []byte("# {{.Name}}\n")},
	"bad.tmpl":    {Data: []byte("{{.Missing}}")},
}

// featureData is the template data of the CreateFeatureFiles tests.
type featureData struct{ Name, Kind string }

// TestCreateFeatureFilesGolden compares the created tree (contents and modes)
// with testdata/feature, under a restrictive umask.
//
// Covers: FR-024
func TestCreateFeatureFilesGolden(t *testing.T) {
	old := syscall.Umask(0o077)
	defer syscall.Umask(old)
	root := t.TempDir()
	d := featureData{Name: "demo", Kind: "chat"}
	files := []FeatureFile{
		{Path: "app/routes/chat.py", Templates: featureTemplates, Template: "chat.tmpl", Data: d},
		{Path: "README.feature", Templates: featureTemplates, Template: "readme.tmpl", Data: d},
	}
	got, err := CreateFeatureFiles(context.Background(), OSFeatureFS{}, root, "demo", files)
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"app/routes/chat.py", "README.feature"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("created %v, want %v", got, want)
	}
	for _, f := range files {
		want, err := os.ReadFile(filepath.Join("testdata", "feature", filepath.FromSlash(f.Path)))
		if err != nil {
			t.Fatal(err)
		}
		p := filepath.Join(root, filepath.FromSlash(f.Path))
		have, err := os.ReadFile(p)
		if err != nil || !bytes.Equal(have, want) {
			t.Errorf("%s: got %q (%v), want %q", f.Path, have, err, want)
		}
		if info, _ := os.Stat(p); info.Mode().Perm() != FileMode {
			t.Errorf("%s: mode %v, want %v", f.Path, info.Mode().Perm(), FileMode)
		}
	}
	for _, dir := range []string{"app", "app/routes"} {
		if info, err := os.Stat(filepath.Join(root, dir)); err != nil || info.Mode().Perm() != DirMode {
			t.Errorf("%s: %v %v, want mode %v", dir, info, err, DirMode)
		}
	}
}

// TestCreateFeatureFilesRenderFailure checks that a template that fails to
// render creates nothing and names the path and the cause.
//
// Covers: FR-024
func TestCreateFeatureFilesRenderFailure(t *testing.T) {
	root := t.TempDir()
	d := featureData{Name: "demo"}
	files := []FeatureFile{
		{Path: "ok.txt", Templates: featureTemplates, Template: "readme.tmpl", Data: d},
		{Path: "sub/bad.txt", Templates: featureTemplates, Template: "bad.tmpl", Data: d},
	}
	before := listTree(t, root)
	_, err := CreateFeatureFiles(context.Background(), OSFeatureFS{}, root, "demo", files)
	if err == nil || !strings.Contains(err.Error(), "sub/bad.txt") || !strings.Contains(err.Error(), "Missing") {
		t.Fatalf("got %v, want an error naming the path and the cause", err)
	}
	if after := listTree(t, root); !reflect.DeepEqual(before, after) {
		t.Errorf("tree changed: %v -> %v", before, after)
	}
}

// racyFS creates a file just before the first OpenFile, as a concurrent
// process would after the existence check.
type racyFS struct {
	OSFeatureFS
	hook string
	done bool
}

// OpenFile plants the hook file once, then delegates to the OS.
// In: name, flag, perm as for FeatureFS. Out: the OS result.
// Side effects: creates racyFS.hook on the first call.
func (r *racyFS) OpenFile(name string, flag int, perm os.FileMode) (io.WriteCloser, error) {
	if !r.done {
		r.done = true
		if err := os.WriteFile(r.hook, []byte("theirs"), 0o644); err != nil {
			return nil, err
		}
	}
	return r.OSFeatureFS.OpenFile(name, flag, perm)
}

// TestCreateFeatureFilesRace checks that a file appearing after the check is
// never replaced, the run fails and no file of this run is left.
//
// Covers: FR-024, EC-018
func TestCreateFeatureFilesRace(t *testing.T) {
	root := t.TempDir()
	d := featureData{Name: "demo"}
	files := []FeatureFile{
		{Path: "a.txt", Templates: featureTemplates, Template: "readme.tmpl", Data: d},
		{Path: "b.txt", Templates: featureTemplates, Template: "readme.tmpl", Data: d},
	}
	fsys := &racyFS{hook: filepath.Join(root, "a.txt")}
	if _, err := CreateFeatureFiles(context.Background(), fsys, root, "demo", files); err == nil {
		t.Fatal("expected an error")
	}
	if b, _ := os.ReadFile(filepath.Join(root, "a.txt")); string(b) != "theirs" {
		t.Errorf("a.txt replaced: %q", b)
	}
	if _, err := os.Lstat(filepath.Join(root, "b.txt")); !os.IsNotExist(err) {
		t.Errorf("b.txt left behind: %v", err)
	}
}

// TestValidateEntriesActions checks the duplicate rules of the file actions
// and the resolution of injection targets reached through symbolic links.
//
// Covers: FR-008, EC-014
func TestValidateEntriesActions(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	if err := os.WriteFile(filepath.Join(outside, "router.py"), []byte("# aicli:routers\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(root, "app"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "app", "router.py"), []byte("# aicli:routers\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(outside, "router.py"), filepath.Join(root, "app", "out.py")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(root, "app", "router.py"), filepath.Join(root, "app", "in.py")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(root, "missing.py"), filepath.Join(root, "app", "broken.py")); err != nil {
		t.Fatal(err)
	}
	inject := func(p string) FeatureFile { return FeatureFile{Path: p, Action: ActionInject, Marker: "routers"} }
	tests := []struct {
		name    string
		entries []FeatureFile
		want    string
	}{
		{"two injects on one path", []FeatureFile{inject("app/router.py"), {Path: "app/x.py"}, inject("app/router.py")}, ""},
		{"inject through inside link", []FeatureFile{inject("app/in.py")}, ""},
		{"inject on broken link left to the check", []FeatureFile{inject("app/broken.py")}, ""},
		{"inject on missing file left to the check", []FeatureFile{inject("app/none.py")}, ""},
		{"create and inject", []FeatureFile{{Path: "app/router.py"}, inject("app/router.py")}, `invalid path "app/router.py": listed more than once`},
		{"inject and create", []FeatureFile{inject("app/router.py"), {Path: "app/router.py"}}, `invalid path "app/router.py": listed more than once`},
		{"shared duplicate", []FeatureFile{{Path: "app/s.py", Action: ActionShared}, {Path: "app/s.py", Action: ActionShared}}, `invalid path "app/s.py": listed more than once`},
		{"shared and inject", []FeatureFile{{Path: "app/s.py", Action: ActionShared}, inject("app/s.py")}, `invalid path "app/s.py": listed more than once`},
		{"inject link outside", []FeatureFile{inject("app/out.py")}, `invalid path "app/out.py": resolves outside the project`},
		{"empty plan", nil, "demo declares no files"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateEntries(root, "demo", tt.entries)
			if tt.want == "" {
				if err != nil {
					t.Fatalf("got %v, want nil", err)
				}
				return
			}
			if err == nil || err.Error() != tt.want {
				t.Fatalf("got %v, want %q", err, tt.want)
			}
		})
	}
}
