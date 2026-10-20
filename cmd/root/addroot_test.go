package root

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/ro-uc-ru/aicli/internal/featuretest"
)

// runAdd runs the root with the featuretest features registered under `add`.
//
// In: t, the test; dir, the working directory seen by the project check;
// args, the arguments. Out: the exit code, stdout and stderr.
// Side effects: as run.
func runAdd(t *testing.T, dir string, args ...string) (int, string, string) {
	t.Helper()
	cmd := newRoot(".", func() (string, error) { return dir, nil })
	addCmd, _, err := cmd.Find([]string{"add"})
	if err != nil || addCmd == cmd {
		t.Fatalf("add command not found: %v", err)
	}
	featuretest.Register(addCmd, featuretest.Valid()...)
	var out, errb bytes.Buffer
	code := run(context.Background(), cmd, args, &out, &errb)
	return code, out.String(), errb.String()
}

// TestAddOutsideProject runs `aicli add`, `aicli add <feature>` with valid
// values and `aicli add <unknown>` in a directory that is not a project: each
// fails with the message of spec 002 (golden file), shows no screen on
// stdout and creates nothing.
//
// In: t, the test. Out: none. Side effects: none outside a temp directory.
// Covers: FR-001, EC-001
func TestAddOutsideProject(t *testing.T) {
	want, err := os.ReadFile(filepath.Join("testdata", "not-a-project.golden"))
	if err != nil {
		t.Fatal(err)
	}
	cases := map[string][]string{
		"add":         {"add"},
		"add feature": {"add", featuretest.Main, "one", "name"},
		"add unknown": {"add", "unknown-feature"},
	}
	for name, args := range cases {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			before := snapshot(t, dir)
			code, out, errb := runAdd(t, dir, args...)
			if code != 1 {
				t.Errorf("exit code = %d, want 1", code)
			}
			if out != "" {
				t.Errorf("stdout = %q, want empty", out)
			}
			if errb != string(want) {
				t.Errorf("stderr = %q, want %q", errb, want)
			}
			if !reflect.DeepEqual(before, snapshot(t, dir)) {
				t.Error("the directory changed")
			}
		})
	}
}

// TestAddHelpOutsideProject runs `aicli add --help` and `aicli help add`
// outside a project: both succeed and print the help of `add`.
//
// In: t, the test. Out: none. Side effects: none.
// Covers: FR-001
func TestAddHelpOutsideProject(t *testing.T) {
	for name, args := range map[string][]string{
		"add --help": {"add", "--help"},
		"help add":   {"help", "add"},
	} {
		t.Run(name, func(t *testing.T) {
			code, out, errb := runAdd(t, t.TempDir(), args...)
			if code != 0 || errb != "" {
				t.Fatalf("code=%d stderr=%q, want 0 and empty", code, errb)
			}
			if !bytes.Contains([]byte(out), []byte("Add an AI/LLM feature to this project.")) {
				t.Errorf("help of add not printed:\n%s", out)
			}
		})
	}
}
