//go:build linux || darwin

package scaffold

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/ro-uc-ru/aicli/internal/gen"
)

// listDir returns the names in dir.
// In: t, the running test; dir, a directory. Out: the entry names.
// Side effects: fails the test on read errors.
func listDir(t *testing.T, dir string) []string {
	t.Helper()
	es, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, e := range es {
		names = append(names, e.Name())
	}
	return names
}

// TestRunSuccess checks that Run creates the project and no staging leftovers.
// In: t, the running test. Out: none. Side effects: uses a temp dir.
// Covers: FR-001.
func TestRunSuccess(t *testing.T) {
	base := t.TempDir()
	if err := Run(context.Background(), base, "myapp"); err != nil {
		t.Fatal(err)
	}
	if got := listDir(t, base); len(got) != 1 || got[0] != "myapp" {
		t.Fatalf("base contains %v", got)
	}
	info, err := os.Stat(filepath.Join(base, "myapp", "pyproject.toml"))
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o644 {
		t.Errorf("file mode %v", info.Mode().Perm())
	}
	root, _ := os.Stat(filepath.Join(base, "myapp"))
	if root.Mode().Perm() != 0o755 {
		t.Errorf("root mode %v", root.Mode().Perm())
	}
}

// TestRunInvalidName checks that validation errors surface and write nothing.
// In: t, the running test. Out: none. Side effects: uses a temp dir.
// Covers: FR-013, EC-001.
func TestRunInvalidName(t *testing.T) {
	base := t.TempDir()
	if err := Run(context.Background(), base, ""); !errors.Is(err, ErrMissingName) {
		t.Errorf("got %v", err)
	}
	if err := Run(context.Background(), base, "../x"); !errors.Is(err, ErrInvalidName) {
		t.Errorf("got %v", err)
	}
	if got := listDir(t, base); len(got) != 0 {
		t.Errorf("base contains %v", got)
	}
}

// TestRunExistingDestination checks that every kind of existing destination
// is rejected and left untouched.
// In: t, the running test. Out: none. Side effects: uses temp dirs.
// Covers: FR-014, EC-004, EC-007, EC-008.
func TestRunExistingDestination(t *testing.T) {
	tests := map[string]func(t *testing.T, dest string){
		"file": func(t *testing.T, dest string) {
			if err := os.WriteFile(dest, []byte("keep"), 0o644); err != nil {
				t.Fatal(err)
			}
		},
		"empty dir": func(t *testing.T, dest string) {
			if err := os.Mkdir(dest, 0o755); err != nil {
				t.Fatal(err)
			}
		},
		"non-empty dir": func(t *testing.T, dest string) {
			if err := os.Mkdir(dest, 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(dest, "f"), []byte("keep"), 0o644); err != nil {
				t.Fatal(err)
			}
		},
		"symlink": func(t *testing.T, dest string) {
			if err := os.Symlink(t.TempDir(), dest); err != nil {
				t.Fatal(err)
			}
		},
		"broken symlink": func(t *testing.T, dest string) {
			if err := os.Symlink(filepath.Join(t.TempDir(), "missing"), dest); err != nil {
				t.Fatal(err)
			}
		},
	}
	for name, setup := range tests {
		t.Run(name, func(t *testing.T) {
			base := t.TempDir()
			dest := filepath.Join(base, "myapp")
			setup(t, dest)
			before, _ := os.Lstat(dest)
			err := Run(context.Background(), base, "myapp")
			if !errors.Is(err, ErrDestinationExists) {
				t.Fatalf("got %v, want ErrDestinationExists", err)
			}
			after, aerr := os.Lstat(dest)
			if aerr != nil || !os.SameFile(before, after) || before.Mode() != after.Mode() {
				t.Errorf("destination changed: %v", aerr)
			}
			if got := listDir(t, base); len(got) != 1 {
				t.Errorf("base contains %v", got)
			}
		})
	}
}

// TestRunTwice checks that the second run fails and keeps the first result.
// In: t, the running test. Out: none. Side effects: uses a temp dir.
// Covers: FR-014, EC-007.
func TestRunTwice(t *testing.T) {
	base := t.TempDir()
	if err := Run(context.Background(), base, "myapp"); err != nil {
		t.Fatal(err)
	}
	marker := filepath.Join(base, "myapp", "marker")
	if err := os.WriteFile(marker, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := Run(context.Background(), base, "myapp"); !errors.Is(err, ErrDestinationExists) {
		t.Fatalf("got %v", err)
	}
	if _, err := os.Stat(marker); err != nil {
		t.Errorf("first result changed: %v", err)
	}
}

// TestRunNonWritableBase checks that an unwritable base directory fails.
// In: t, the running test. Out: none. Side effects: uses a temp dir.
// Covers: FR-016, EC-005.
func TestRunNonWritableBase(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root ignores directory permissions")
	}
	base := t.TempDir()
	if err := os.Chmod(base, 0o555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(base, 0o755) })
	if err := Run(context.Background(), base, "myapp"); err == nil {
		t.Fatal("want error")
	}
	if got := listDir(t, base); len(got) != 0 {
		t.Errorf("base contains %v", got)
	}
}

// failFS is a gen.FS that fails OpenFile after n successful calls.
type failFS struct {
	gen.OSFS
	n int
}

// OpenFile fails once the allowance is used up.
// In: name, flag, perm as in gen.FS. Out: the file or an injected error.
// Side effects: creates the file while the allowance lasts.
func (f *failFS) OpenFile(name string, flag int, perm os.FileMode) (io.WriteCloser, error) {
	if f.n--; f.n < 0 {
		return nil, errors.New("injected failure")
	}
	return f.OSFS.OpenFile(name, flag, perm)
}

// TestRunFailingWrite checks that a failed write leaves nothing behind.
// In: t, the running test. Out: none. Side effects: uses a temp dir.
// Covers: FR-016.
func TestRunFailingWrite(t *testing.T) {
	base := t.TempDir()
	err := run(context.Background(), base, "myapp", hooks{fs: &failFS{n: 3}})
	if err == nil || !strings.Contains(err.Error(), "injected failure") {
		t.Fatalf("got %v", err)
	}
	if got := listDir(t, base); len(got) != 0 {
		t.Errorf("base contains %v", got)
	}
}

// TestRunDestinationAppears checks that a destination created between the
// check and the rename, even an empty directory, is not replaced.
// In: t, the running test. Out: none. Side effects: uses a temp dir.
// Covers: EC-010.
func TestRunDestinationAppears(t *testing.T) {
	base := t.TempDir()
	dest := filepath.Join(base, "myapp")
	err := run(context.Background(), base, "myapp", hooks{afterCheck: func() {
		if err := os.Mkdir(dest, 0o755); err != nil {
			t.Error(err)
		}
	}})
	if !errors.Is(err, ErrDestinationExists) {
		t.Fatalf("got %v", err)
	}
	if got := listDir(t, dest); len(got) != 0 {
		t.Errorf("destination replaced: %v", got)
	}
	if got := listDir(t, base); len(got) != 1 {
		t.Errorf("base contains %v", got)
	}
}

// TestRunConcurrent checks that N concurrent runs yield exactly one success.
// In: t, the running test. Out: none. Side effects: uses a temp dir.
// Covers: FR-014, EC-010.
func TestRunConcurrent(t *testing.T) {
	base := t.TempDir()
	const n = 16
	var wg sync.WaitGroup
	var ok, exists atomic.Int32
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			switch err := Run(context.Background(), base, "myapp"); {
			case err == nil:
				ok.Add(1)
			case errors.Is(err, ErrDestinationExists):
				exists.Add(1)
			default:
				t.Errorf("unexpected: %v", err)
			}
		}()
	}
	wg.Wait()
	if ok.Load() != 1 || exists.Load() != n-1 {
		t.Fatalf("ok=%d exists=%d", ok.Load(), exists.Load())
	}
	if got := listDir(t, base); len(got) != 1 {
		t.Errorf("base contains %v", got)
	}
}

// TestRunStagingNameLength checks that the staging name length does not
// depend on the project name: it stays within the pattern plus MkdirTemp's
// random suffix (at most 10 digits) for both a short and a 255-byte name.
// In: t, the running test. Out: none. Side effects: uses temp dirs.
// Covers: EC-012.
func TestRunStagingNameLength(t *testing.T) {
	limit := len(strings.TrimSuffix(stagingPattern, "*")) + 10
	for _, name := range []string{"a", strings.Repeat("b", 255)} {
		err := run(context.Background(), t.TempDir(), name, hooks{beforeRename: func(s string) {
			if got := len(filepath.Base(s)); got > limit {
				t.Errorf("staging name %q has %d bytes, limit %d", filepath.Base(s), got, limit)
			}
		}})
		if err != nil {
			t.Fatal(err)
		}
	}
}
