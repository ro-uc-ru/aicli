package add

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/ro-uc-ru/aicli/internal/featuretest"
	"github.com/ro-uc-ru/aicli/internal/project"
	"github.com/ro-uc-ru/aicli/internal/tui"
)

// TestPrepareDeprecatedRunsSilently checks that after Prepare a deprecated
// feature run directly through Cobra prints only the success report, and is
// still not listed.
//
// Covers: EC-002
func TestPrepareDeprecatedRunsSilently(t *testing.T) {
	add := featuretest.NewTree(featuretest.NewDeprecated())
	root := add.Parent()
	rootDir := t.TempDir()

	Prepare(root)
	Prepare(root)
	Prepare(nil)

	var out, errOut bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&errOut)
	root.SetArgs([]string{"add", featuretest.Deprecated, "x"})
	if err := root.ExecuteContext(project.WithRoot(context.Background(), rootDir)); err != nil {
		t.Fatalf("execute: %v", err)
	}
	if strings.Contains(out.String()+errOut.String(), "use feature-1") {
		t.Fatalf("deprecation notice printed: %q %q", out.String(), errOut.String())
	}
	if !strings.Contains(out.String(), "Added "+featuretest.Deprecated) || errOut.Len() != 0 {
		t.Fatalf("stdout = %q, stderr = %q", out.String(), errOut.String())
	}
	if _, err := os.Lstat(filepath.Join(rootDir, "features", "one.txt")); err != nil {
		t.Fatalf("feature did not run: %v", err)
	}
	for _, f := range Features(add) {
		if f.Listed {
			t.Fatalf("%s is listed", f.Name)
		}
	}
}

// TestPrepareLeavesOthers checks that Prepare changes neither a normal feature
// nor a tree without add.
//
// Covers: EC-002
func TestPrepareLeavesOthers(t *testing.T) {
	add := featuretest.NewTree(featuretest.NewMain())
	Prepare(add.Parent())
	if f := Features(add); len(f) != 1 || !f[0].Listed {
		t.Fatalf("features = %+v", f)
	}
}

// directSetup builds a tree with the main and the no-input features, prepares
// it, and stubs the screens with a recorder.
//
// In: t, the test; interactive, whether the terminal check passes; res, what
// the stub screens return. Out: the root command, the project root, and a
// pointer to the features the stub was called with (nil if never called).
// Side effects: replaces runScreens and isInteractive until the test ends.
func directSetup(t *testing.T, interactive bool, res tui.AddResult) (*cobra.Command, string, *[]tui.Feature) {
	t.Helper()
	add := featuretest.NewTree(featuretest.NewMain(), featuretest.NewNoInputs())
	Prepare(add.Parent())
	var got []tui.Feature
	oldS, oldI := runScreens, isInteractive
	runScreens = func(_ context.Context, f []tui.Feature, _ io.Reader, _ io.Writer, _ bool, opts ...tui.AddOption) (tui.AddResult, error) {
		got = f
		if len(opts) != 1 {
			t.Errorf("options = %d, want 1 (WithDirect)", len(opts))
		}
		return res, nil
	}
	isInteractive = func(io.Reader, io.Writer, func(string) string) bool { return interactive }
	t.Cleanup(func() { runScreens, isInteractive = oldS, oldI })
	return add.Parent(), t.TempDir(), &got
}

// runDirect executes the root with args and returns stdout and the error.
//
// In: root, the root command; dir, the project root; args, the arguments.
// Out: stdout and the error. Side effects: executes the command.
func runDirect(root *cobra.Command, dir string, args ...string) (string, error) {
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&bytes.Buffer{})
	root.SetArgs(args)
	err := root.ExecuteContext(project.WithRoot(context.Background(), dir))
	return out.String(), err
}

// TestPrepareDirectForm checks that a bare call in interactive mode opens the
// form of that feature alone, and handles quit, cancel and add.
//
// Covers: FR-005, EC-016
func TestPrepareDirectForm(t *testing.T) {
	root, dir, got := directSetup(t, true, tui.AddResult{Action: tui.ActionQuit})
	out, err := runDirect(root, dir, "add", featuretest.Main)
	if err != nil || out != "" {
		t.Fatalf("quit: out = %q, err = %v", out, err)
	}
	if len(*got) != 1 || (*got)[0].Name != featuretest.Main {
		t.Fatalf("screens got %+v", *got)
	}

	root, dir, _ = directSetup(t, true, tui.AddResult{Action: tui.ActionCancel})
	if out, err = runDirect(root, dir, "add", featuretest.Main); err != nil || out != cancelMessage {
		t.Fatalf("cancel: out = %q, err = %v", out, err)
	}

	root, dir, _ = directSetup(t, true, tui.AddResult{Action: tui.ActionAdd, Values: []string{"one", "x"}})
	if out, err = runDirect(root, dir, "add", featuretest.Main); err != nil || !strings.Contains(out, "Added "+featuretest.Main) {
		t.Fatalf("add: out = %q, err = %v", out, err)
	}
	if _, err := os.Lstat(filepath.Join(dir, "features", "x.txt")); err != nil {
		t.Fatalf("feature did not run: %v", err)
	}
}

// TestPrepareDirectSkips checks that static mode, arguments, flags and a
// feature with no inputs do not open the form.
//
// Covers: FR-005, EC-001
func TestPrepareDirectSkips(t *testing.T) {
	root, dir, _ := directSetup(t, false, tui.AddResult{Action: tui.ActionQuit})
	_, err := runDirect(root, dir, "add", featuretest.Main)
	if err == nil || strings.SplitN(err.Error(), "\n", 2)[0] != featuretest.Main+" requires input-1" {
		t.Fatalf("static mode err = %v", err)
	}

	root, dir, got := directSetup(t, true, tui.AddResult{Action: tui.ActionQuit})
	if out, err := runDirect(root, dir, "add", featuretest.Main, "one", "x"); err != nil || !strings.Contains(out, "Added") {
		t.Fatalf("with arguments: out = %q, err = %v", out, err)
	}
	if out, err := runDirect(root, dir, "add", featuretest.NoInputs); err != nil || !strings.Contains(out, "Added") {
		t.Fatalf("no inputs: out = %q, err = %v", out, err)
	}
	if *got != nil {
		t.Fatalf("screens ran: %+v", *got)
	}

	root, dir, got = directSetup(t, true, tui.AddResult{Action: tui.ActionQuit})
	root.PersistentFlags().Bool("flag", false, "")
	_, err = runDirect(root, dir, "add", featuretest.Main, "--flag")
	if err == nil || *got != nil {
		t.Fatalf("with flag: err = %v, screens = %+v", err, *got)
	}
}

// TestPrepareWrapsOnce checks that calling Prepare again does not wrap the
// RunE of a feature a second time.
//
// Covers: FR-005
func TestPrepareWrapsOnce(t *testing.T) {
	add := featuretest.NewTree(featuretest.NewMain())
	Prepare(add.Parent())
	f := add.Commands()[0]
	first := reflect.ValueOf(f.RunE).Pointer()
	n := len(f.Annotations)
	Prepare(add.Parent())
	if reflect.ValueOf(f.RunE).Pointer() != first || len(f.Annotations) != n {
		t.Fatal("Prepare wrapped twice")
	}
	if f.Annotations[directAnnotation] == "" {
		t.Fatal("feature not marked")
	}
}
