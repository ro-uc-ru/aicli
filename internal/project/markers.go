// Package project detects whether the current working directory is the root
// of an aicli project (spec 002).
package project

import (
	"fmt"
	"strings"
)

// markers lists the project markers in the order of the spec Definitions,
// written relative to the project root with '/' (FR-001, FR-003, FR-006).
var markers = []string{
	"pyproject.toml",
	"alembic.ini",
	"app/main.py",
	"migrations/env.py",
}

// Markers returns the project markers in their fixed order.
// In: none. Out: a fresh slice of the four marker paths, relative to a
// project root and written with '/'; callers may modify it freely.
// No side effects.
func Markers() []string {
	return append([]string(nil), markers...)
}

// WorkingDirError reports that the current working directory could not be
// determined (FR-006, EC-008).
type WorkingDirError struct {
	// Err is the system error returned when resolving the directory.
	Err error
}

// Error renders the single-line message of FR-006.
// In: none. Out: "cannot determine the current directory: <cause>".
// No side effects.
func (e *WorkingDirError) Error() string {
	return "cannot determine the current directory: " + e.Err.Error()
}

// Unwrap exposes the underlying system error.
// In: none. Out: the cause. No side effects.
func (e *WorkingDirError) Unwrap() error { return e.Err }

// InspectError reports that a project marker could not be inspected for a
// reason other than being missing (FR-006, EC-007, EC-011, EC-012).
type InspectError struct {
	// Path is the marker as written in the Definitions (relative, with '/').
	Path string
	// Err is the system error, unwrapped from *fs.PathError so that its text
	// reads e.g. "permission denied".
	Err error
}

// Error renders the single-line message of FR-006.
// In: none. Out: "cannot inspect <path>: <cause>". No side effects.
func (e *InspectError) Error() string {
	return fmt.Sprintf("cannot inspect %s: %s", e.Path, e.Err.Error())
}

// Unwrap exposes the underlying system error.
// In: none. Out: the cause. No side effects.
func (e *InspectError) Unwrap() error { return e.Err }

// NotProjectError reports that the working directory is not a project root
// and that no inspection failure occurred (FR-003).
type NotProjectError struct {
	// Missing lists the missing markers in the order of Markers.
	Missing []string
}

// Error renders the message of FR-003 without the "Error: " prefix and
// without indentation on the continuation lines, so that tui.RenderError
// adds both.
// In: none. Out: three lines separated by "\n", without a trailing newline:
// the reason, "Missing: <comma-separated markers>" and the hint.
// No side effects.
func (e *NotProjectError) Error() string {
	return "not inside an aicli project\n" +
		"Missing: " + strings.Join(e.Missing, ", ") + "\n" +
		"Run this command from the root of a project created with `aicli new`."
}
