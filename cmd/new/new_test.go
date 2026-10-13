package newcmd

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/ro-uc-ru/aicli/internal/scaffold"
	"github.com/ro-uc-ru/aicli/internal/tui"
)

// execute runs the command with args in baseDir using in-memory writers.
//
// In: t, the test; baseDir, the parent directory; args, the arguments.
// Out: stdout and stderr contents and the command error.
// Side effects: may create a project in baseDir.
func execute(t *testing.T, baseDir string, args ...string) (stdout, stderr string, err error) {
	t.Helper()
	cmd := NewCommand(baseDir)
	var o, e bytes.Buffer
	cmd.SetOut(&o)
	cmd.SetErr(&e)
	cmd.SetArgs(args)
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	err = cmd.Execute()
	return o.String(), e.String(), err
}

// assertEmptyDir fails the test if dir contains any entry.
//
// In: t, the test; dir, the directory. Out: none.
// Side effects: none.
func assertEmptyDir(t *testing.T, dir string) {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("expected no files, found %d", len(entries))
	}
}

// TestArgErrors checks the errors for 0, "" and 2 arguments: no files, empty
// stdout (FR-012, FR-024, EC-003).
// Covers: FR-012, FR-024, EC-003.
func TestArgErrors(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
		is   error
	}{
		{"no args", nil, "", scaffold.ErrMissingName},
		{"empty name", []string{""}, "", scaffold.ErrMissingName},
		{"two args", []string{"a", "b"}, "too many arguments\nUsage: aicli new <name>", nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			stdout, _, err := execute(t, dir, tt.args...)
			if err == nil {
				t.Fatal("expected an error")
			}
			if tt.is != nil && !errors.Is(err, tt.is) {
				t.Fatalf("error = %v, want %v", err, tt.is)
			}
			if tt.want != "" && err.Error() != tt.want {
				t.Fatalf("error = %q, want %q", err.Error(), tt.want)
			}
			if stdout != "" {
				t.Fatalf("stdout = %q, want empty", stdout)
			}
			assertEmptyDir(t, dir)
		})
	}
}

// TestUsageContinuation checks that the FR-024 error renders with the
// indented usage continuation line.
// Covers: FR-023, FR-024.
func TestUsageContinuation(t *testing.T) {
	_, _, err := execute(t, t.TempDir(), "a", "b")
	got := tui.RenderError(err.Error(), false)
	want := "Error: too many arguments\n  Usage: aicli new <name>\n"
	if got != want {
		t.Fatalf("rendered = %q, want %q", got, want)
	}
}

// TestSuccess checks the exact stdout on success with a non-TTY writer and
// that the project was created (FR-001, FR-015).
// Covers: FR-001, FR-015.
func TestSuccess(t *testing.T) {
	dir := t.TempDir()
	stdout, stderr, err := execute(t, dir, "myapp")
	if err != nil {
		t.Fatal(err)
	}
	if want := tui.RenderSuccess("myapp", false); stdout != want {
		t.Fatalf("stdout = %q, want %q", stdout, want)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
	if _, err := os.Stat(filepath.Join(dir, "myapp", "pyproject.toml")); err != nil {
		t.Fatal(err)
	}
}

// TestSameFilesWithAndWithoutTTY checks EC-006: the generated files are the
// same whether or not stdout is a terminal. A pty is not available portably,
// so the terminal case is a TTY-looking environment (TERM set, NO_COLOR
// unset) with a non-file writer, which must change nothing on disk.
// Covers: EC-006.
func TestSameFilesWithAndWithoutTTY(t *testing.T) {
	list := func(env map[string]string) []string {
		for k, v := range env {
			t.Setenv(k, v)
		}
		dir := t.TempDir()
		if _, _, err := execute(t, dir, "myapp"); err != nil {
			t.Fatal(err)
		}
		var files []string
		err := filepath.WalkDir(dir, func(p string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			rel, _ := filepath.Rel(dir, p)
			files = append(files, rel)
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
		return files
	}
	plain := list(map[string]string{"TERM": "dumb", "NO_COLOR": "1"})
	tty := list(map[string]string{"TERM": "xterm-256color", "NO_COLOR": ""})
	if len(plain) == 0 || len(plain) != len(tty) {
		t.Fatalf("file lists differ: %d vs %d", len(plain), len(tty))
	}
	for i := range plain {
		if plain[i] != tty[i] {
			t.Fatalf("file %d: %q vs %q", i, plain[i], tty[i])
		}
	}
}

// TestFailureStdoutEmpty checks that an existing destination fails with an
// empty stdout (FR-023).
// Covers: FR-014, FR-023.
func TestFailureStdoutEmpty(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "myapp"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "myapp", "x"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	stdout, _, err := execute(t, dir, "myapp")
	if !errors.Is(err, scaffold.ErrDestinationExists) {
		t.Fatalf("error = %v", err)
	}
	if stdout != "" {
		t.Fatalf("stdout = %q, want empty", stdout)
	}
}

// TestInvalidName checks that an invalid name fails with empty stdout.
// Covers: FR-013, FR-023.
func TestInvalidName(t *testing.T) {
	dir := t.TempDir()
	stdout, _, err := execute(t, dir, "Bad Name")
	if !errors.Is(err, scaffold.ErrInvalidName) || stdout != "" {
		t.Fatalf("err = %v, stdout = %q", err, stdout)
	}
	assertEmptyDir(t, dir)
}
