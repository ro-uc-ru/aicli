//go:build linux || darwin

package scaffold

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"syscall"
	"testing"

	"github.com/ro-uc-ru/aicli/internal/gen"
)

// cancelFS is a gen.FS that calls cancel when OpenFile is called the n-th time.
type cancelFS struct {
	gen.OSFS
	n      int
	cancel func()
}

// OpenFile counts calls and cancels on the n-th one before delegating.
// In: name, flag, perm as in gen.FS. Out: the handle or error.
// Side effects: may call cancel; creates the file.
func (f *cancelFS) OpenFile(name string, flag int, perm os.FileMode) (io.WriteCloser, error) {
	if f.n--; f.n == 0 {
		f.cancel()
	}
	return f.OSFS.OpenFile(name, flag, perm)
}

// TestRunCancel cancels with a signal cause at each hook point and after the
// rename, and checks that no destination and no staging directory remain and
// that the typed error exposes the signal.
// In: t, the running test. Out: none. Side effects: uses temp dirs.
// Covers: FR-016, EC-009.
func TestRunCancel(t *testing.T) {
	sigs := []syscall.Signal{syscall.SIGINT, syscall.SIGTERM}
	points := map[string]func(cancel func(), h *hooks){
		"after check":   func(cancel func(), h *hooks) { h.afterCheck = cancel },
		"before rename": func(cancel func(), h *hooks) { h.beforeRename = func(string) { cancel() } },
		"after rename":  func(cancel func(), h *hooks) { h.afterRename = cancel },
		"between writes": func(cancel func(), h *hooks) {
			h.fs = &cancelFS{n: 3, cancel: cancel}
		},
	}
	for point, set := range points {
		for _, sig := range sigs {
			t.Run(point+"/"+sig.String(), func(t *testing.T) {
				base := t.TempDir()
				ctx, cancel := context.WithCancelCause(context.Background())
				var h hooks
				set(func() { cancel(&SignalError{Signal: sig}) }, &h)
				err := run(ctx, base, "myapp", h)
				var se *SignalError
				if !errors.As(err, &se) || se.Signal != sig {
					t.Fatalf("got %v, want SignalError %v", err, sig)
				}
				if want := 128 + int(sig); se.ExitCode() != want {
					t.Errorf("exit code %d, want %d", se.ExitCode(), want)
				}
				if got := listDir(t, base); len(got) != 0 {
					t.Errorf("base contains %v", got)
				}
			})
		}
	}
}

// TestRunCancelBeforeStart checks that an already cancelled context writes
// nothing and that a plain cancellation wraps context.Canceled.
// In: t, the running test. Out: none. Side effects: uses a temp dir.
// Covers: EC-009.
func TestRunCancelBeforeStart(t *testing.T) {
	base := t.TempDir()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := Run(ctx, base, "myapp"); !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v", err)
	}
	if _, err := os.Lstat(filepath.Join(base, "myapp")); !os.IsNotExist(err) {
		t.Errorf("destination exists: %v", err)
	}
	if got := listDir(t, base); len(got) != 0 {
		t.Errorf("base contains %v", got)
	}
}
