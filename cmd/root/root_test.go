package root

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"
	"testing"

	"github.com/ro-uc-ru/aicli/internal/scaffold"
	"github.com/ro-uc-ru/aicli/internal/tui"
)

// ansiRe matches ANSI escape sequences.
var ansiRe = regexp.MustCompile(`\x1b\[[0-9;]*m`)

// inTempDir changes the working directory to a new temp dir for the test.
//
// In: t, the test. Out: the directory. Side effects: chdir, restored on exit.
func inTempDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	old, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(old) })
	return dir
}

// TestFailuresExitOne checks exit 1, an "Error: " stderr and empty stdout for
// an invalid name, an existing destination and a write failure (FR-023).
// Covers: FR-013, FR-014, FR-016, FR-023.
func TestFailuresExitOne(t *testing.T) {
	inTempDir(t)
	if err := os.Mkdir("exists", 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join("exists", "f"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	cases := map[string][]string{
		"invalid name":   {"new", "Bad Name"},
		"existing dest":  {"new", "exists"},
		"too many args":  {"new", "a", "b"},
		"missing name":   {"new"},
		"unknown subcmd": {"bogus"},
	}
	for name, args := range cases {
		t.Run(name, func(t *testing.T) {
			var out, errb bytes.Buffer
			code := Execute(context.Background(), args, &out, &errb)
			if code != 1 {
				t.Fatalf("code = %d", code)
			}
			if out.Len() != 0 {
				t.Fatalf("stdout = %q", out.String())
			}
			if got := ansiRe.ReplaceAllString(errb.String(), ""); !strings.HasPrefix(got, "Error: ") {
				t.Fatalf("stderr = %q", got)
			}
		})
	}
}

// TestWriteFailure checks exit 1 and an Error message when the working
// directory is not writable.
// Covers: FR-016, EC-005.
func TestWriteFailure(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("root ignores directory permissions")
	}
	dir := inTempDir(t)
	if err := os.Chmod(dir, 0o500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(dir, 0o755) })
	var out, errb bytes.Buffer
	code := Execute(context.Background(), []string{"new", "myapp"}, &out, &errb)
	if code != 1 || out.Len() != 0 || !strings.HasPrefix(errb.String(), "Error: ") {
		t.Fatalf("code=%d stdout=%q stderr=%q", code, out.String(), errb.String())
	}
}

// TestSignalExitCodes checks 130 and 143 with no output (EC-009).
// Covers: FR-023, EC-009.
func TestSignalExitCodes(t *testing.T) {
	for sig, want := range map[syscall.Signal]int{syscall.SIGINT: 130, syscall.SIGTERM: 143} {
		dir := inTempDir(t)
		ctx, cancel := context.WithCancelCause(context.Background())
		cancel(&scaffold.SignalError{Signal: sig})
		var out, errb bytes.Buffer
		code := Execute(ctx, []string{"new", "myapp"}, &out, &errb)
		if code != want || out.Len() != 0 || errb.Len() != 0 {
			t.Fatalf("%v: code=%d stdout=%q stderr=%q", sig, code, out.String(), errb.String())
		}
		if entries, _ := os.ReadDir(dir); len(entries) != 0 {
			t.Fatalf("%v: left %d entries", sig, len(entries))
		}
	}
}

// TestBuiltBinary builds the binary and runs `aicli new myapp` in a temp
// directory: exit 0 and the FR-015 text.
// Covers: FR-001, FR-015.
func TestBuiltBinary(t *testing.T) {
	bin := filepath.Join(t.TempDir(), "aicli")
	build := exec.Command("go", "build", "-o", bin, "github.com/ro-uc-ru/aicli")
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build: %v\n%s", err, out)
	}
	work := t.TempDir()
	run := exec.Command(bin, "new", "myapp")
	run.Dir = work
	var out bytes.Buffer
	run.Stdout = &out
	if err := run.Run(); err != nil {
		t.Fatal(err)
	}
	if want := tui.RenderSuccess("myapp", false); out.String() != want {
		t.Fatalf("stdout = %q, want %q", out.String(), want)
	}
	if _, err := os.Stat(filepath.Join(work, "myapp", "pyproject.toml")); err != nil {
		t.Fatal(err)
	}
}
