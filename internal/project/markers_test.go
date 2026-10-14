package project

import (
	"errors"
	"io/fs"
	"slices"
	"syscall"
	"testing"

	"github.com/ro-uc-ru/aicli/internal/tui"
)

// TestMarkers checks the marker list content, order and isolation.
//
// Covers: FR-003, FR-006
func TestMarkers(t *testing.T) {
	want := []string{"pyproject.toml", "alembic.ini", "app/main.py", "migrations/env.py"}
	got := Markers()
	if !slices.Equal(got, want) {
		t.Fatalf("Markers() = %v, want %v", got, want)
	}
	got[0] = "x"
	if Markers()[0] != "pyproject.toml" {
		t.Fatal("Markers() exposes its backing array")
	}
}

// TestWorkingDirError checks the exact text of the working-directory error.
//
// Covers: FR-006
func TestWorkingDirError(t *testing.T) {
	err := &WorkingDirError{Err: syscall.ENOENT}
	want := "cannot determine the current directory: no such file or directory"
	if err.Error() != want {
		t.Fatalf("Error() = %q, want %q", err.Error(), want)
	}
	if !errors.Is(err, syscall.ENOENT) {
		t.Fatal("cause is not unwrappable")
	}
	if got, want := tui.RenderError(err.Error(), false), "Error: "+want+"\n"; got != want {
		t.Fatalf("RenderError = %q, want %q", got, want)
	}
}

// TestInspectError checks the exact text of the inspection error, with the
// cause unwrapped from a *fs.PathError.
//
// Covers: FR-006
func TestInspectError(t *testing.T) {
	pe := &fs.PathError{Op: "stat", Path: "/tmp/x/app/main.py", Err: syscall.EACCES}
	err := &InspectError{Path: "app/main.py", Err: pe.Err}
	want := "cannot inspect app/main.py: permission denied"
	if err.Error() != want {
		t.Fatalf("Error() = %q, want %q", err.Error(), want)
	}
	if !errors.Is(err, syscall.EACCES) {
		t.Fatal("cause is not unwrappable")
	}
	if got, want := tui.RenderError(err.Error(), false), "Error: "+want+"\n"; got != want {
		t.Fatalf("RenderError = %q, want %q", got, want)
	}
}

// TestNotProjectError checks the exact FR-003 text and that RenderError in
// plain form yields the FR-003 block verbatim.
//
// Covers: FR-003
func TestNotProjectError(t *testing.T) {
	err := &NotProjectError{Missing: Markers()}
	want := "not inside an aicli project\n" +
		"Missing: pyproject.toml, alembic.ini, app/main.py, migrations/env.py\n" +
		"Run this command from the root of a project created with `aicli new`."
	if err.Error() != want {
		t.Fatalf("Error() = %q, want %q", err.Error(), want)
	}
	block := "Error: not inside an aicli project\n" +
		"  Missing: pyproject.toml, alembic.ini, app/main.py, migrations/env.py\n" +
		"  Run this command from the root of a project created with `aicli new`.\n"
	if got := tui.RenderError(err.Error(), false); got != block {
		t.Fatalf("RenderError = %q, want %q", got, block)
	}

	one := &NotProjectError{Missing: []string{"migrations/env.py"}}
	if got, want := tui.RenderError(one.Error(), false), "Error: not inside an aicli project\n  Missing: migrations/env.py\n  Run this command from the root of a project created with `aicli new`.\n"; got != want {
		t.Fatalf("RenderError = %q, want %q", got, want)
	}
}
