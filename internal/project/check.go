package project

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// Check decides whether the current working directory is a project root.
//
// In: getwd, the function that resolves the working directory (os.Getwd in
// production, a stub in tests).
// Out: on success, the root exactly as getwd returned it (FR-009) and a nil
// error. Otherwise an empty root and one of: *WorkingDirError (getwd failed;
// no marker is inspected), *InspectError (the first marker, in the order of
// Markers, that could not be inspected, even if others are missing), or
// *NotProjectError (no inspection failure and at least one marker missing).
// Side effects: stats paths only; it opens, reads, writes, creates or removes
// nothing and spawns no process (FR-004, FR-005).
func Check(getwd func() (string, error)) (string, error) {
	root, err := getwd()
	if err != nil {
		return "", &WorkingDirError{Err: unwrapCause(err)}
	}

	var missing []string
	for _, m := range markers {
		present, err := inspect(root, m)
		if err != nil {
			return "", err
		}
		if !present {
			missing = append(missing, m)
		}
	}
	if len(missing) > 0 {
		return "", &NotProjectError{Missing: missing}
	}
	return root, nil
}

// inspect classifies one marker below root.
//
// In: root, the working directory; marker, a marker path written with '/'.
// Out: true if the marker exists as a regular file (following symbolic
// links); false if it is missing (absent, a broken link, not a regular file,
// or a parent component exists and is not a directory); a *InspectError
// carrying the unwrapped system error for any other failure.
// Side effects: os.Stat calls only.
func inspect(root, marker string) (bool, error) {
	path := filepath.Join(root, filepath.FromSlash(marker))
	info, err := os.Stat(path)
	if err == nil {
		return info.Mode().IsRegular(), nil
	}
	if errors.Is(err, fs.ErrNotExist) {
		return false, nil
	}
	if parentIsNotDir(root, marker) {
		return false, nil
	}
	return false, &InspectError{Path: marker, Err: unwrapCause(err)}
}

// unwrapCause strips the operation and path that os wraps around a system
// error, so that the message reads e.g. "permission denied" and not
// "stat /x/app/main.py: permission denied" or "getwd: no such file or
// directory".
//
// In: err, an error from os.Getwd or os.Stat.
// Out: the inner error of an *fs.PathError or *os.SyscallError; err itself
// for any other type.
// No side effects.
func unwrapCause(err error) error {
	var pe *fs.PathError
	if errors.As(err, &pe) {
		return pe.Err
	}
	var se *os.SyscallError
	if errors.As(err, &se) {
		return se.Err
	}
	return err
}

// parentIsNotDir reports whether a parent component of marker exists below
// root and is not a directory (EC-010).
//
// In: root, the working directory; marker, a marker path written with '/'.
// Out: true if some parent component, checked from root down, stats fine and
// is not a directory; false otherwise, including when a stat fails.
// Side effects: os.Stat calls only.
func parentIsNotDir(root, marker string) bool {
	parts := strings.Split(marker, "/")
	dir := root
	for _, p := range parts[:len(parts)-1] {
		dir = filepath.Join(dir, p)
		info, err := os.Stat(dir)
		if err != nil {
			return false
		}
		if !info.IsDir() {
			return true
		}
	}
	return false
}
