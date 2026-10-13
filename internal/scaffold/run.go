package scaffold

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/ro-uc-ru/aicli/internal/gen"
)

// stagingPattern is the MkdirTemp pattern of the staging directory; its
// length does not depend on the project name (EC-012).
const stagingPattern = ".aicli-new-*"

// hooks are the internal test seams of run. The zero value is production
// behavior.
type hooks struct {
	// fs is the filesystem the skeleton is written through; nil means gen.OSFS.
	fs gen.FS
	// afterCheck runs after the destination check and before staging is created.
	afterCheck func()
	// beforeRename runs after the skeleton is written, with the staging path.
	beforeRename func(staging string)
	// afterRename runs right after the successful rename.
	afterRename func()
}

// Run creates the project skeleton at baseDir/name. It writes everything into
// a staging directory inside baseDir and moves it to the destination with a
// no-replace rename, so the destination exists only when the project is
// complete and an existing destination is never modified.
//
// In: ctx, the run context, checked between steps and between writes; baseDir, an existing
// directory (not the process working directory, so callers and tests are
// independent); name, the project name.
// Out: nil on success; ErrMissingName or an error wrapping ErrInvalidName for
// a bad name; an error wrapping ErrDestinationExists if baseDir/name exists in
// any form (including a broken symlink) or appears before the rename;
// *SignalError if ctx was cancelled with one as its cause (EC-009), or an
// error wrapping ctx.Err() for any other cancellation; otherwise an error with
// context (e.g. baseDir not writable, a write failed).
// Side effects: creates and renames a staging directory in baseDir and writes
// the skeleton into it; on any error the staging directory is removed and,
// if the rename already happened, so is the destination this run created.
func Run(ctx context.Context, baseDir, name string) error {
	return run(ctx, baseDir, name, hooks{})
}

// run is Run with test hooks.
//
// In: ctx, baseDir and name as in Run; h, the hooks.
// Out: as Run. Side effects: as Run.
func run(ctx context.Context, baseDir, name string, h hooks) (err error) {
	if err := ValidateName(name); err != nil {
		return err
	}
	if err := cancelled(ctx); err != nil {
		return err
	}
	dest := filepath.Join(baseDir, name)
	if _, err := os.Lstat(dest); err == nil {
		return fmt.Errorf("%q: %w", dest, ErrDestinationExists)
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("check destination %q: %w", dest, err)
	}
	if h.afterCheck != nil {
		h.afterCheck()
	}
	if err := cancelled(ctx); err != nil {
		return err
	}
	entries, err := gen.Render(name)
	if err != nil {
		return fmt.Errorf("render skeleton: %w", err)
	}
	staging, err := os.MkdirTemp(baseDir, stagingPattern)
	if err != nil {
		return fmt.Errorf("create staging directory in %q: %w", baseDir, err)
	}
	renamed := false
	defer func() {
		if err == nil {
			return
		}
		if renamed {
			_ = os.RemoveAll(dest)
		} else {
			_ = os.RemoveAll(staging)
		}
	}()
	if err = os.Chmod(staging, gen.DirMode); err != nil {
		return fmt.Errorf("set mode of staging directory: %w", err)
	}
	var fsys gen.FS = gen.OSFS{}
	if h.fs != nil {
		fsys = h.fs
	}
	if err = gen.Write(ctxFS{FS: fsys, ctx: ctx}, staging, entries); err != nil {
		if ce := cancelled(ctx); ce != nil {
			err = ce
			return err
		}
		return fmt.Errorf("write skeleton: %w", err)
	}
	if h.beforeRename != nil {
		h.beforeRename(staging)
	}
	if err = cancelled(ctx); err != nil {
		return err
	}
	if err = renameNoReplace(staging, dest); err != nil {
		return err
	}
	renamed = true
	if h.afterRename != nil {
		h.afterRename()
	}
	err = cancelled(ctx)
	return err
}
