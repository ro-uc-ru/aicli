package gen_test

import (
	"context"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"testing/fstest"

	"github.com/ro-uc-ru/aicli/internal/gen"
	"github.com/ro-uc-ru/aicli/internal/scaffold"
)

// rollbackTemplates holds the single template used by the rollback tests.
var rollbackTemplates = fstest.MapFS{"t.tmpl": {Data: []byte("x\n")}}

// faultFS wraps the OS filesystem and injects faults by relative file name.
type faultFS struct {
	gen.OSFeatureFS
	// failWrite makes Write fail for files whose path ends with this suffix.
	failWrite string
	// failRemove makes Remove fail with EIO for paths ending with these suffixes.
	failRemove []string
	// onOpen runs before every OpenFile with the path being opened.
	onOpen func(name string)
	// onChmod runs after every successful Chmod with the path.
	onChmod func(name string)
}

// OpenFile runs the onOpen hook, opens the file and wraps it when its write
// must fail.
// In: name, flag, perm as for gen.FeatureFS. Out: the handle or the error.
// Side effects: may create the file and run the hook.
func (f *faultFS) OpenFile(name string, flag int, perm os.FileMode) (io.WriteCloser, error) {
	if f.onOpen != nil {
		f.onOpen(name)
	}
	w, err := f.OSFeatureFS.OpenFile(name, flag, perm)
	if err == nil && f.failWrite != "" && strings.HasSuffix(name, f.failWrite) {
		return failingWriter{w}, nil
	}
	return w, err
}

// Chmod delegates to the OS and then runs the onChmod hook.
// In: name, mode as for gen.FeatureFS. Out: the OS error.
// Side effects: changes the mode and may run the hook.
func (f *faultFS) Chmod(name string, mode os.FileMode) error {
	if err := f.OSFeatureFS.Chmod(name, mode); err != nil {
		return err
	}
	if f.onChmod != nil {
		f.onChmod(name)
	}
	return nil
}

// Remove fails with EIO for the configured suffixes, else delegates to the OS.
// In: name, the path. Out: the error. Side effects: may delete name.
func (f *faultFS) Remove(name string) error {
	for _, s := range f.failRemove {
		if strings.HasSuffix(name, s) {
			return &fs.PathError{Op: "remove", Path: name, Err: syscall.EIO}
		}
	}
	return f.OSFeatureFS.Remove(name)
}

// failingWriter is a WriteCloser whose Write always fails.
type failingWriter struct{ io.WriteCloser }

// Write always fails with "boom".
// In: p, ignored. Out: 0 and the error. Side effects: none.
func (failingWriter) Write([]byte) (int, error) { return 0, errors.New("boom") }

// rollbackFiles builds feature files for the given paths.
// In: paths, the targets. Out: the feature files. Side effects: none.
func rollbackFiles(paths ...string) []gen.FeatureFile {
	out := make([]gen.FeatureFile, len(paths))
	for i, p := range paths {
		out[i] = gen.FeatureFile{Path: p, Templates: rollbackTemplates, Template: "t.tmpl"}
	}
	return out
}

// entries lists every path under dir, relative and sorted by the walk.
// In: t, the test; dir, the root. Out: the relative paths. Side effects: none.
func entries(t *testing.T, dir string) []string {
	t.Helper()
	var out []string
	err := filepath.WalkDir(dir, func(p string, _ fs.DirEntry, err error) error {
		if err == nil && p != dir {
			out = append(out, filepath.ToSlash(p[len(dir)+1:]))
		}
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	return out
}

// TestRollbackOnWriteFailure checks that a forced write failure leaves nothing
// of this run.
//
// Covers: FR-024
func TestRollbackOnWriteFailure(t *testing.T) {
	root := t.TempDir()
	fsys := &faultFS{failWrite: "c.txt"}
	_, err := gen.CreateFeatureFiles(context.Background(), fsys, root, "demo", rollbackFiles("sub/a.txt", "b.txt", "deep/er/c.txt"))
	if err == nil || err.Error() != "write file deep/er/c.txt: boom" {
		t.Fatalf("got %v", err)
	}
	if got := entries(t, root); len(got) != 0 {
		t.Errorf("left behind: %v", got)
	}
}

// TestRollbackKeepsNonEmptyParent checks that a directory with something else
// inside is kept and not reported.
//
// Covers: FR-024
func TestRollbackKeepsNonEmptyParent(t *testing.T) {
	root := t.TempDir()
	fsys := &faultFS{failWrite: "b.txt"}
	fsys.onOpen = func(name string) {
		if strings.HasSuffix(name, "b.txt") {
			if err := os.WriteFile(filepath.Join(root, "sub", "other.txt"), []byte("o"), 0o644); err != nil {
				t.Fatal(err)
			}
		}
	}
	_, err := gen.CreateFeatureFiles(context.Background(), fsys, root, "demo", rollbackFiles("sub/a.txt", "sub/b.txt"))
	if err == nil || strings.Contains(err.Error(), "could not remove") {
		t.Fatalf("got %v", err)
	}
	got := entries(t, root)
	if len(got) != 2 || got[0] != "sub" || got[1] != "sub/other.txt" {
		t.Errorf("left %v, want sub and sub/other.txt", got)
	}
}

// TestRollbackCouldNotRemove checks the "could not remove:" lines against a
// golden file, in creation order.
//
// Covers: FR-024
func TestRollbackCouldNotRemove(t *testing.T) {
	root := t.TempDir()
	fsys := &faultFS{failWrite: "c.txt", failRemove: []string{"sub/a.txt", "b.txt"}}
	_, err := gen.CreateFeatureFiles(context.Background(), fsys, root, "demo", rollbackFiles("sub/a.txt", "b.txt", "c.txt"))
	want, rerr := os.ReadFile(filepath.Join("testdata", "feature", "could_not_remove.golden"))
	if rerr != nil {
		t.Fatal(rerr)
	}
	if err == nil || err.Error()+"\n" != string(want) {
		t.Fatalf("got %q, want %q", err, want)
	}
}

// TestRollbackOnSignal checks that a context cancelled with a
// *scaffold.SignalError between two files completes the rollback and returns
// that cause.
//
// Covers: FR-024, EC-023
func TestRollbackOnSignal(t *testing.T) {
	root := t.TempDir()
	ctx, cancel := context.WithCancelCause(context.Background())
	cause := &scaffold.SignalError{Signal: syscall.SIGINT}
	fsys := &faultFS{}
	fsys.onChmod = func(name string) {
		if strings.HasSuffix(name, "a.txt") {
			cancel(cause)
		}
	}
	_, err := gen.CreateFeatureFiles(ctx, fsys, root, "demo", rollbackFiles("sub/a.txt", "b.txt"))
	var se *scaffold.SignalError
	if !errors.As(err, &se) || se != cause {
		t.Fatalf("got %v, want the signal cause", err)
	}
	if got := entries(t, root); len(got) != 0 {
		t.Errorf("left behind: %v", got)
	}
}
