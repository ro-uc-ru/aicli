package scaffold

import (
	"context"
	"fmt"
	"io"
	"os"
	"syscall"

	"github.com/ro-uc-ru/aicli/internal/gen"
)

// SignalError reports that a run was interrupted by a signal (EC-009). The
// caller cancels the run context with it as the cause
// (context.WithCancelCause) so Run can return it.
type SignalError struct {
	// Signal is the signal received (SIGINT or SIGTERM).
	Signal os.Signal
}

// Error describes the interruption.
// In: none. Out: the message. No side effects.
func (e *SignalError) Error() string {
	return fmt.Sprintf("interrupted by signal %v", e.Signal)
}

// ExitCode returns the shell exit code for the signal: 130 for SIGINT, 143
// for SIGTERM, 128+n in general, and 1 if the signal is not a syscall.Signal.
// In: none. Out: the exit code. No side effects.
func (e *SignalError) ExitCode() int {
	if s, ok := e.Signal.(syscall.Signal); ok {
		return 128 + int(s)
	}
	return 1
}

// cancelled reports whether ctx is done.
// In: ctx, the run context. Out: nil if still live; otherwise the
// *SignalError set as the cancellation cause, or an error wrapping ctx.Err().
// No side effects.
func cancelled(ctx context.Context) error {
	if ctx.Err() == nil {
		return nil
	}
	if se, ok := context.Cause(ctx).(*SignalError); ok {
		return se
	}
	return fmt.Errorf("run cancelled: %w", ctx.Err())
}

// ctxFS wraps a gen.FS and checks the context before each creation, so a
// cancellation is noticed between writes.
type ctxFS struct {
	gen.FS
	ctx context.Context
}

// Mkdir checks ctx, then creates the directory through the wrapped FS.
// In: name, perm as in gen.FS. Out: the cancellation or wrapped FS error.
// Side effects: creates the directory.
func (c ctxFS) Mkdir(name string, perm os.FileMode) error {
	if err := cancelled(c.ctx); err != nil {
		return err
	}
	return c.FS.Mkdir(name, perm)
}

// OpenFile checks ctx, then creates the file through the wrapped FS.
// In: name, flag, perm as in gen.FS. Out: the handle, or the cancellation or
// wrapped FS error. Side effects: creates the file.
func (c ctxFS) OpenFile(name string, flag int, perm os.FileMode) (io.WriteCloser, error) {
	if err := cancelled(c.ctx); err != nil {
		return nil, err
	}
	return c.FS.OpenFile(name, flag, perm)
}
