package root

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"github.com/ro-uc-ru/aicli/internal/featuretest"
	"github.com/ro-uc-ru/aicli/internal/scaffold"
)

// TestSignalDuringCreation builds the test-only program of
// internal/featuretest/testdata/signalbin, which blocks between the two files
// of the test feature, sends it SIGINT and then SIGTERM during the creation
// and checks that each run exits with 128 plus the signal number, prints
// nothing and leaves no file or directory of the run.
//
// In: t, the test. Out: none. Side effects: builds a binary and runs it in
// temp directories.
// Covers: EC-020, EC-023
func TestSignalDuringCreation(t *testing.T) {
	bin := filepath.Join(t.TempDir(), "signalbin")
	build := exec.Command("go", "build", "-o", bin, "github.com/ro-uc-ru/aicli/internal/featuretest/testdata/signalbin")
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build: %v\n%s", err, out)
	}
	for sig, want := range map[syscall.Signal]int{syscall.SIGINT: 130, syscall.SIGTERM: 143} {
		t.Run(sig.String(), func(t *testing.T) {
			dir := projectDir(t)
			before := listTree(t, dir)
			ready := filepath.Join(t.TempDir(), "ready")
			cmd := exec.Command(bin, ready, featuretest.Main, "one", "name")
			cmd.Dir = dir
			var out, errb bytes.Buffer
			cmd.Stdout, cmd.Stderr = &out, &errb
			if err := cmd.Start(); err != nil {
				t.Fatal(err)
			}
			done := make(chan error, 1)
			go func() { done <- cmd.Wait() }()
			deadline := time.Now().Add(10 * time.Second)
			for {
				if _, err := os.Stat(ready); err == nil {
					break
				}
				if time.Now().After(deadline) {
					_ = cmd.Process.Kill()
					t.Fatal("the program never reached the block between files")
				}
				time.Sleep(10 * time.Millisecond)
			}
			if _, err := os.Stat(filepath.Join(dir, "features", "name.txt")); err != nil {
				t.Fatalf("the first file must exist while blocked: %v", err)
			}
			if err := cmd.Process.Signal(sig); err != nil {
				t.Fatal(err)
			}
			var err error
			select {
			case err = <-done:
			case <-time.After(10 * time.Second):
				_ = cmd.Process.Kill()
				t.Fatal("the program did not exit after the signal")
			}
			ee, ok := err.(*exec.ExitError)
			if !ok || ee.ExitCode() != want {
				t.Fatalf("exit = %v, want code %d", err, want)
			}
			if out.Len() != 0 || errb.Len() != 0 {
				t.Errorf("printed stdout=%q stderr=%q, want nothing", out.String(), errb.String())
			}
			if after := listTree(t, dir); !equalStrings(before, after) {
				t.Errorf("files left by the run: before %v, after %v", before, after)
			}
		})
	}
}

// listTree lists every path under dir, relative and sorted by WalkDir.
//
// In: t, the test; dir, the directory. Out: the relative paths.
// Side effects: none.
func listTree(t *testing.T, dir string) []string {
	t.Helper()
	var paths []string
	err := filepath.WalkDir(dir, func(p string, _ os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(dir, p)
		paths = append(paths, rel)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return paths
}

// equalStrings reports whether two slices hold the same strings in order.
//
// In: a and b, the slices. Out: true if equal. Side effects: none.
func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// TestAddCancelledContext runs `aicli add` in static mode and a direct
// `aicli add feature-1` with a context already cancelled with a
// *scaffold.SignalError: through run each gives 128 plus the signal number,
// prints nothing and creates nothing.
//
// In: t, the test. Out: none. Side effects: writes in a temp directory.
// Covers: EC-020, EC-023
func TestAddCancelledContext(t *testing.T) {
	for sig, want := range map[syscall.Signal]int{syscall.SIGINT: 130, syscall.SIGTERM: 143} {
		for _, args := range [][]string{{"add"}, {"add", featuretest.Main, "one", "name"}} {
			dir := projectDir(t)
			before := listTree(t, dir)
			cmd := newRoot(".", func() (string, error) { return dir, nil })
			addCmd, _, err := cmd.Find([]string{"add"})
			if err != nil || addCmd == cmd {
				t.Fatalf("add command not found: %v", err)
			}
			featuretest.Register(addCmd, featuretest.Valid()...)
			ctx, cancel := context.WithCancelCause(context.Background())
			cancel(&scaffold.SignalError{Signal: sig})
			var out, errb bytes.Buffer
			code := run(ctx, cmd, args, &out, &errb)
			if code != want || out.Len() != 0 || errb.Len() != 0 {
				t.Errorf("%v %v: code=%d stdout=%q stderr=%q, want %d and nothing", sig, args, code, out.String(), errb.String(), want)
			}
			if after := listTree(t, dir); !equalStrings(before, after) {
				t.Errorf("%v %v: files left: %v", sig, args, after)
			}
		}
	}
}
