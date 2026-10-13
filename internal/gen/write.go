package gen

import (
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
)

const (
	// DirMode is the permission of every generated directory (FR-022).
	DirMode os.FileMode = 0o755
	// FileMode is the permission of every generated file (FR-022).
	FileMode os.FileMode = 0o644
)

// FS is the minimal file-creation interface used by Write, injectable so
// tests can force failures.
type FS interface {
	// Mkdir creates one directory.
	// In: name, the directory path; perm, the requested mode.
	// Out: an error if it cannot be created or already exists.
	// Side effects: creates the directory.
	Mkdir(name string, perm os.FileMode) error
	// OpenFile creates a file for writing.
	// In: name, the file path; flag, the os.O_* flags; perm, the requested mode.
	// Out: a writable handle, or an error.
	// Side effects: creates the file.
	OpenFile(name string, flag int, perm os.FileMode) (io.WriteCloser, error)
	// Chmod sets the mode of a path exactly, regardless of the umask.
	// In: name, the path; mode, the mode to set.
	// Out: an error if the mode cannot be set.
	// Side effects: changes the permissions of name.
	Chmod(name string, mode os.FileMode) error
}

// OSFS is the FS backed by the real filesystem.
type OSFS struct{}

// Mkdir implements FS with os.Mkdir.
// In: name, the directory path; perm, the requested mode.
// Out: the os.Mkdir error. Side effects: creates the directory.
func (OSFS) Mkdir(name string, perm os.FileMode) error { return os.Mkdir(name, perm) }

// OpenFile implements FS with os.OpenFile.
// In: name, the file path; flag, the os.O_* flags; perm, the requested mode.
// Out: the file handle, or the os.OpenFile error. Side effects: creates the file.
func (OSFS) OpenFile(name string, flag int, perm os.FileMode) (io.WriteCloser, error) {
	return os.OpenFile(name, flag, perm)
}

// Chmod implements FS with os.Chmod.
// In: name, the path; mode, the mode to set.
// Out: the os.Chmod error. Side effects: changes the permissions of name.
func (OSFS) Chmod(name string, mode os.FileMode) error { return os.Chmod(name, mode) }

// Write dumps entries into baseDir. Directories get DirMode and files
// FileMode, set explicitly with Chmod so the umask has no effect. Existing
// files are never overwritten (O_EXCL).
//
// In: fsys, the filesystem to write through; baseDir, an existing directory;
// entries, as returned by Render (parents precede children).
// Out: nil, or the first error, wrapped with the entry path.
// Side effects: creates directories and files under baseDir; on error it
// leaves what it already wrote (the caller removes it).
func Write(fsys FS, baseDir string, entries []Entry) error {
	for _, e := range entries {
		target := filepath.Join(baseDir, filepath.FromSlash(path.Clean(e.Path)))
		if e.Kind == KindDir {
			if err := fsys.Mkdir(target, DirMode); err != nil {
				return fmt.Errorf("create directory %s: %w", e.Path, err)
			}
			if err := fsys.Chmod(target, DirMode); err != nil {
				return fmt.Errorf("set mode of directory %s: %w", e.Path, err)
			}
			continue
		}
		if err := writeFile(fsys, target, e); err != nil {
			return err
		}
	}
	return nil
}

// writeFile creates one file with its content and mode.
//
// In: fsys, the filesystem; target, the full path; e, the file entry.
// Out: nil, or an error wrapped with the entry path.
// Side effects: creates the file at target.
func writeFile(fsys FS, target string, e Entry) (err error) {
	f, err := fsys.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_EXCL, FileMode)
	if err != nil {
		return fmt.Errorf("create file %s: %w", e.Path, err)
	}
	defer func() {
		if cerr := f.Close(); err == nil && cerr != nil {
			err = fmt.Errorf("close file %s: %w", e.Path, cerr)
		}
	}()
	if _, err = f.Write(e.Content); err != nil {
		return fmt.Errorf("write file %s: %w", e.Path, err)
	}
	if err = fsys.Chmod(target, FileMode); err != nil {
		return fmt.Errorf("set mode of file %s: %w", e.Path, err)
	}
	return nil
}
