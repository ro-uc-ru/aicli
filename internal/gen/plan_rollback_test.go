package gen_test

import (
	"context"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"syscall"
	"testing"

	"github.com/ro-uc-ru/aicli/internal/gen"
	"github.com/ro-uc-ru/aicli/internal/scaffold"
)

// planFaultFS wraps the OS filesystem and injects faults into ApplyPlan runs.
type planFaultFS struct {
	gen.OSFeatureFS
	// opens counts the OpenFile calls per path.
	opens map[string]int
	// failWrite makes Write fail on the n-th opening (from 1) of name.
	failWrite func(name string, n int) bool
	// failRemove makes Remove fail with EIO for paths ending with these suffixes.
	failRemove []string
	// onOpen runs before every OpenFile with the path and its opening count.
	onOpen func(name string, n int)
}

// OpenFile counts the opening, runs the hook, opens the file and wraps it
// when its write must fail.
// In: name, flag, perm as for gen.FeatureFS. Out: the handle or the error.
// Side effects: may create or truncate the file and run the hook.
func (f *planFaultFS) OpenFile(name string, flag int, perm os.FileMode) (io.WriteCloser, error) {
	if f.opens == nil {
		f.opens = map[string]int{}
	}
	f.opens[name]++
	n := f.opens[name]
	if f.onOpen != nil {
		f.onOpen(name, n)
	}
	w, err := f.OSFeatureFS.OpenFile(name, flag, perm)
	if err == nil && f.failWrite != nil && f.failWrite(name, n) {
		return brokenWriter{w}, nil
	}
	return w, err
}

// Remove fails with EIO for the configured suffixes, else delegates to the OS.
// In: name, the path. Out: the error. Side effects: may delete name.
func (f *planFaultFS) Remove(name string) error {
	for _, s := range f.failRemove {
		if strings.HasSuffix(name, s) {
			return &fs.PathError{Op: "remove", Path: name, Err: syscall.EIO}
		}
	}
	return f.OSFeatureFS.Remove(name)
}

// brokenWriter is a WriteCloser whose Write always fails.
type brokenWriter struct{ io.WriteCloser }

// Write always fails with "boom".
// In: p, ignored. Out: 0 and the error. Side effects: none.
func (brokenWriter) Write([]byte) (int, error) { return 0, errors.New("boom") }

// rollbackProject writes a project whose pyproject.toml is a link to a 0600
// file inside the root, and whose .env.example has mode 0640.
// In: t, the running test; root, the project root. Out: none.
// Side effects: writes files and a link.
func rollbackProject(t *testing.T, root string) {
	t.Helper()
	gen.WritePlanFile(t, root, "config/pyproject.toml", "deps = [\n  # aicli:dependencies\n]\n")
	if err := os.Chmod(filepath.Join(root, "config", "pyproject.toml"), 0o600); err != nil {
		t.Fatal(err)
	}
	gen.SymlinkPlan(t, root, filepath.Join("config", "pyproject.toml"), "pyproject.toml")
	gen.WritePlanFile(t, root, ".env.example", "APP=1\r\n# aicli:env\r\n")
	if err := os.Chmod(filepath.Join(root, ".env.example"), 0o640); err != nil {
		t.Fatal(err)
	}
}

// suffix reports a failWrite or onOpen match on the end of a path.
// In: name, the path; s, the "/" suffix. Out: true on a match.
// Side effects: none.
func suffix(name, s string) bool { return strings.HasSuffix(filepath.ToSlash(name), s) }

// TestApplyPlanRollbackSecondInject checks that a write failure on the second
// injected file restores both injected files and removes every created path.
//
// Covers: FR-011, EC-013, EC-029
func TestApplyPlanRollbackSecondInject(t *testing.T) {
	root := t.TempDir()
	rollbackProject(t, root)
	before := gen.SnapshotTree(t, root)
	fsys := &planFaultFS{failWrite: func(name string, n int) bool { return suffix(name, ".env.example") && n == 1 }}
	_, err := gen.ApplyPlan(context.Background(), fsys, root, "llm-provider", gen.ProviderPlan("openai"))
	if err == nil || err.Error() != "update file .env.example: boom" {
		t.Fatalf("got %v", err)
	}
	if after := gen.SnapshotTree(t, root); !reflect.DeepEqual(before, after) {
		t.Errorf("tree not restored:\n%v\n->\n%v", before, after)
	}
}

// TestApplyPlanRollbackAfterAll checks that a failure after every creation
// and injection restores every byte and mode and keeps the link a link.
//
// Covers: FR-011, EC-013, EC-029
func TestApplyPlanRollbackAfterAll(t *testing.T) {
	root := t.TempDir()
	rollbackProject(t, root)
	before := gen.SnapshotTree(t, root)
	fsys := &planFaultFS{failWrite: func(name string, _ int) bool { return suffix(name, "tests/test_llm_claude.py") }}
	_, err := gen.ApplyPlan(context.Background(), fsys, root, "llm-provider", gen.ProviderPlan("openai", "claude"))
	if err == nil || err.Error() != "write file tests/test_llm_claude.py: boom" {
		t.Fatalf("got %v", err)
	}
	if after := gen.SnapshotTree(t, root); !reflect.DeepEqual(before, after) {
		t.Errorf("tree not restored:\n%v\n->\n%v", before, after)
	}
}

// TestApplyPlanRollbackCouldNotRestore checks the rollback error when a
// created file cannot be removed and an injected file cannot be restored,
// against testdata/feature/could_not_restore.golden.
//
// Covers: FR-011, EC-013
func TestApplyPlanRollbackCouldNotRestore(t *testing.T) {
	root := t.TempDir()
	rollbackProject(t, root)
	fsys := &planFaultFS{
		failWrite: func(name string, n int) bool {
			return suffix(name, "tests/test_llm_openai.py") || (suffix(name, "config/pyproject.toml") && n == 2)
		},
		failRemove: []string{"app/ai/llm/openai.py"},
	}
	_, err := gen.ApplyPlan(context.Background(), fsys, root, "llm-provider", gen.ProviderPlan("openai"))
	if err == nil {
		t.Fatal("expected an error")
	}
	path := filepath.Join("testdata", "feature", "could_not_restore.golden")
	if *gen.Update {
		if werr := os.WriteFile(path, []byte(err.Error()+"\n"), 0o644); werr != nil {
			t.Fatal(werr)
		}
	}
	want, rerr := os.ReadFile(path)
	if rerr != nil {
		t.Fatal(rerr)
	}
	if err.Error()+"\n" != string(want) {
		t.Fatalf("got %q, want %q", err, want)
	}
}

// TestApplyPlanRollbackOnSignal checks that a context cancelled with a
// *scaffold.SignalError between two steps completes the rollback and returns
// that cause.
//
// Covers: FR-011, EC-013, EC-029
func TestApplyPlanRollbackOnSignal(t *testing.T) {
	root := t.TempDir()
	rollbackProject(t, root)
	before := gen.SnapshotTree(t, root)
	ctx, cancel := context.WithCancelCause(context.Background())
	cause := &scaffold.SignalError{Signal: syscall.SIGINT}
	fsys := &planFaultFS{onOpen: func(name string, n int) {
		if suffix(name, "config/pyproject.toml") && n == 1 {
			cancel(cause)
		}
	}}
	_, err := gen.ApplyPlan(ctx, fsys, root, "llm-provider", gen.ProviderPlan("openai"))
	var se *scaffold.SignalError
	if !errors.As(err, &se) || se != cause {
		t.Fatalf("got %v, want the signal cause", err)
	}
	if after := gen.SnapshotTree(t, root); !reflect.DeepEqual(before, after) {
		t.Errorf("tree not restored:\n%v\n->\n%v", before, after)
	}
}
