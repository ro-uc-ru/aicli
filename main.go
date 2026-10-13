// Command aicli scaffolds convention-based FastAPI backends for AI/LLM apps.
package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/ro-uc-ru/aicli/cmd/root"
	"github.com/ro-uc-ru/aicli/internal/scaffold"
)

// main runs aicli and exits with its code.
//
// In: none (reads os.Args). Out: none; the process exit code is the one
// returned by root.Execute. Side effects: installs SIGINT and SIGTERM
// handlers that cancel the run context with a *scaffold.SignalError, runs the
// command and exits the process.
func main() {
	ctx, cancel := context.WithCancelCause(context.Background())
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		s := <-sigs
		cancel(&scaffold.SignalError{Signal: s})
	}()
	code := root.Execute(ctx, os.Args[1:], os.Stdout, os.Stderr)
	cancel(nil)
	os.Exit(code)
}
