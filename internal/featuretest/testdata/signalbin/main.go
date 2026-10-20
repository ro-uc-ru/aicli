// Command signalbin is a test-only program for the signal tests of
// `aicli add`. It runs the test feature of internal/featuretest in the
// current directory, treated as the project root, through a filesystem that
// blocks between two files, so that a test can send SIGINT or SIGTERM in the
// middle of the creation (EC-020, EC-023). It handles signals and exits like
// main.go and cmd/root: 128 plus the signal number, printing nothing.
//
// Usage: signalbin <ready-file> <feature-1 arguments...>. It creates
// <ready-file> when it is blocked after the first feature file.
package main

import (
	"context"
	"errors"
	"io"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	addcmd "github.com/ro-uc-ru/aicli/cmd/add"
	"github.com/ro-uc-ru/aicli/internal/feature"
	"github.com/ro-uc-ru/aicli/internal/featuretest"
	"github.com/ro-uc-ru/aicli/internal/gen"
	"github.com/ro-uc-ru/aicli/internal/project"
	"github.com/ro-uc-ru/aicli/internal/scaffold"
)

// blockTimeout is how long the filesystem waits for a signal before giving up.
const blockTimeout = 10 * time.Second

// blockingFS is the real filesystem that, once a file has been opened, blocks
// at the next Lstat (the check before the second file) until a signal has
// been handled.
type blockingFS struct {
	gen.OSFeatureFS
	ready   string
	signal  <-chan struct{}
	opened  bool
	blocked bool
}

// OpenFile opens a file through the real filesystem and records that one
// feature file has been created.
//
// In: name, flag and perm, as for os.OpenFile. Out: the handle or an error.
// Side effects: opens a file; sets opened.
func (b *blockingFS) OpenFile(name string, flag int, perm os.FileMode) (io.WriteCloser, error) {
	w, err := b.OSFeatureFS.OpenFile(name, flag, perm)
	b.opened = b.opened || err == nil
	return w, err
}

// Lstat blocks, once, if a file was already created: it creates the ready
// file and waits for the signal handler to cancel the run context.
//
// In: name, the path. Out: its info or an error. Side effects: the first
// time after a file was created, creates the ready file and waits up to
// blockTimeout.
func (b *blockingFS) Lstat(name string) (os.FileInfo, error) {
	if b.opened && !b.blocked {
		b.blocked = true
		if f, err := os.Create(b.ready); err == nil {
			_ = f.Close()
		}
		select {
		case <-b.signal:
		case <-time.After(blockTimeout):
		}
	}
	return b.OSFeatureFS.Lstat(name)
}

// main runs the test feature and exits with the code of the signal, 1 on any
// other error, or 0.
//
// In: os.Args as described in the package comment. Out: none.
// Side effects: creates files in the current directory, handles SIGINT and
// SIGTERM, exits the process.
func main() {
	if len(os.Args) < 3 {
		os.Exit(2)
	}
	cwd, err := os.Getwd()
	if err != nil {
		os.Exit(2)
	}
	ctx, cancel := context.WithCancelCause(project.WithRoot(context.Background(), cwd))
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	seen := make(chan struct{})
	go func() {
		s := <-sigs
		cancel(&scaffold.SignalError{Signal: s})
		close(seen)
	}()
	feature.SetFS(&blockingFS{ready: os.Args[1], signal: seen})

	root := &cobra.Command{Use: "aicli", SilenceUsage: true, SilenceErrors: true}
	add := addcmd.NewCommand()
	root.AddCommand(add)
	featuretest.Register(add, featuretest.NewMain())
	root.SetArgs(append([]string{"add"}, os.Args[2:]...))
	err = root.ExecuteContext(ctx)
	var se *scaffold.SignalError
	switch {
	case err == nil:
		os.Exit(0)
	case errors.As(err, &se):
		os.Exit(se.ExitCode())
	default:
		os.Exit(1)
	}
}
