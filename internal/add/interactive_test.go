package add

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/ro-uc-ru/aicli/internal/featuretest"
	"github.com/ro-uc-ru/aicli/internal/project"
	"github.com/ro-uc-ru/aicli/internal/tui"
)

// stubScreens replaces the interactive screens with one that returns res and
// err, and restores the real ones when the test ends.
//
// In: t, the test; res and err, what the stub returns. Out: none.
// Side effects: replaces runScreens for the duration of the test.
func stubScreens(t *testing.T, res tui.AddResult, err error) {
	t.Helper()
	old := runScreens
	runScreens = func(context.Context, []tui.Feature, io.Reader, io.Writer, bool, ...tui.AddOption) (tui.AddResult, error) {
		return res, err
	}
	t.Cleanup(func() { runScreens = old })
}

// interactiveSetup builds a tree with the main feature, a project root and a
// context carrying it.
//
// In: t, the test. Out: the add command, the context and the root directory.
// Side effects: creates a temporary directory.
func interactiveSetup(t *testing.T) (*cobra.Command, context.Context, string) {
	t.Helper()
	add := featuretest.NewTree(featuretest.NewMain())
	root := t.TempDir()
	return add, project.WithRoot(context.Background(), root), root
}

// TestRunInteractiveQuit checks that quitting prints nothing and creates no
// file.
//
// Covers: FR-010
func TestRunInteractiveQuit(t *testing.T) {
	add, ctx, root := interactiveSetup(t)
	stubScreens(t, tui.AddResult{Action: tui.ActionQuit}, nil)
	var out bytes.Buffer
	if err := runInteractive(ctx, add, Features(add), &bytes.Buffer{}, &out, xterm); err != nil {
		t.Fatal(err)
	}
	assertEmptyDir(t, root, out.String(), "")
}

// TestRunInteractiveCancel checks that cancelling prints the plain message
// and creates no file.
//
// Covers: FR-018, EC-015
func TestRunInteractiveCancel(t *testing.T) {
	add, ctx, root := interactiveSetup(t)
	stubScreens(t, tui.AddResult{Action: tui.ActionCancel, Index: 0}, nil)
	var out bytes.Buffer
	if err := runInteractive(ctx, add, Features(add), &bytes.Buffer{}, &out, xterm); err != nil {
		t.Fatal(err)
	}
	assertEmptyDir(t, root, out.String(), "Nothing added.\n")
}

// assertEmptyDir checks the output and that dir has no entries.
//
// In: t, the test; dir, the directory; got and want, the output and its
// expected value. Out: none. Side effects: fails the test on a mismatch.
func assertEmptyDir(t *testing.T, dir, got, want string) {
	t.Helper()
	if got != want {
		t.Fatalf("output = %q, want %q", got, want)
	}
	entries, err := os.ReadDir(dir)
	if err != nil || len(entries) != 0 {
		t.Fatalf("dir entries = %v, %v", entries, err)
	}
}

// TestRunInteractiveAdd checks that adding runs the selected feature with the
// values, creates its files and writes its report; a value starting with `-`
// is a value.
//
// Covers: FR-010, FR-019
func TestRunInteractiveAdd(t *testing.T) {
	add, ctx, root := interactiveSetup(t)
	stubScreens(t, tui.AddResult{Action: tui.ActionAdd, Index: 0, Values: []string{"one", "-x"}}, nil)
	var out bytes.Buffer
	if err := runInteractive(ctx, add, Features(add), &bytes.Buffer{}, &out, xterm); err != nil {
		t.Fatal(err)
	}
	for _, p := range []string{"features/-x.txt", "features/-x.md"} {
		if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(p))); err != nil {
			t.Fatalf("missing %s: %v", p, err)
		}
	}
	if !strings.Contains(out.String(), "features/-x.txt") {
		t.Fatalf("report = %q", out.String())
	}
}

// TestRunInteractiveFeatureFailure checks that a feature failure is returned
// with nothing written and nothing created.
//
// Covers: FR-019, EC-019
func TestRunInteractiveFeatureFailure(t *testing.T) {
	add, ctx, root := interactiveSetup(t)
	stubScreens(t, tui.AddResult{Action: tui.ActionAdd, Index: 0, Values: []string{featuretest.FailChoice, "x"}}, nil)
	var out bytes.Buffer
	err := runInteractive(ctx, add, Features(add), &bytes.Buffer{}, &out, xterm)
	if err == nil || err.Error() != featuretest.PlanError {
		t.Fatalf("error = %v", err)
	}
	assertEmptyDir(t, root, out.String(), "")
}

// TestRunInteractiveScreenError checks that the screens' error is returned
// and nothing is written.
//
// Covers: FR-010
func TestRunInteractiveScreenError(t *testing.T) {
	add, ctx, _ := interactiveSetup(t)
	cause := errors.New("no terminal")
	stubScreens(t, tui.AddResult{}, cause)
	var out bytes.Buffer
	if err := runInteractive(ctx, add, Features(add), &bytes.Buffer{}, &out, xterm); !errors.Is(err, cause) || out.Len() != 0 {
		t.Fatalf("error = %v, out = %q", err, out.String())
	}
}
