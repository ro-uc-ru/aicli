//go:build integration

package scaffold

import (
	"context"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// uv runs `uv <args...>` in dir through os/exec with an argument list.
//
// In: t, the test; dir, the working directory; args, the uv arguments.
// Out: the combined output. The test fails if uv exits non-zero.
// Side effects: spawns uv; `uv sync` uses the network to install the
// generated project's dependencies (TST-6).
func uv(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("uv", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("uv %s: %v\n%s", strings.Join(args, " "), err, out)
	}
	return string(out)
}

// TestIntegrationGeneratedProject scaffolds a project into a temp directory
// and runs its own install, tests, lint and migrations (FR-006, FR-008,
// FR-020, FR-021, FR-026, FR-027, EC-013, EC-014, TST-3, TST-5).
// Covers: FR-004, FR-005, FR-006, FR-008, FR-020, FR-021, FR-026, FR-027, EC-013, EC-014.
func TestIntegrationGeneratedProject(t *testing.T) {
	if _, err := exec.LookPath("uv"); err != nil {
		t.Skip("uv is not on PATH")
	}
	base := t.TempDir()
	if err := Run(context.Background(), base, "myapp"); err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(base, "myapp")

	uv(t, dir, "sync")
	uv(t, dir, "run", "pytest")
	uv(t, dir, "run", "ruff", "check", ".")
	uv(t, dir, "run", "alembic", "upgrade", "head")

	// app/logging.py must not shadow the standard library module.
	got := strings.TrimSpace(uv(t, dir, "run", "python", "-c",
		"import logging; print(logging.__file__)"))
	if rel, err := filepath.Rel(dir, got); err == nil && !strings.HasPrefix(rel, "..") {
		t.Fatalf("standard logging resolved inside the project: %s", got)
	}
}
