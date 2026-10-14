package root

import (
	"bytes"
	"context"
	"flag"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/spf13/cobra"

	"github.com/ro-uc-ru/aicli/internal/home"
	"github.com/ro-uc-ru/aicli/internal/project"
	"github.com/ro-uc-ru/aicli/internal/tui"
)

// update regenerates the golden files instead of comparing against them.
var update = flag.Bool("update", false, "regenerate golden files")

// probe records whether the test command ran and what it read from its
// context.
type probe struct {
	ran      bool
	ownHook  bool
	root     string
	rootSeen bool
}

// newTestRoot builds a root with an injected getwd and a test command named
// "probe", registered only in tests, which records that its RunE ran.
//
// In: t, the test; getwd, the working-directory stub; p, the probe to fill.
// Out: the root command.
// Side effects: none.
func newTestRoot(t *testing.T, getwd func() (string, error), p *probe) *cobra.Command {
	t.Helper()
	cmd := newRoot(".", getwd)
	cmd.AddCommand(&cobra.Command{
		Use:  "probe",
		Args: cobra.NoArgs,
		RunE: func(c *cobra.Command, _ []string) error {
			p.ran = true
			p.root, p.rootSeen = project.RootFrom(c.Context())
			return nil
		},
	})
	cmd.AddCommand(&cobra.Command{
		Use: "guarded",
		PersistentPreRunE: func(*cobra.Command, []string) error {
			p.ownHook = true
			return nil
		},
		RunE: func(c *cobra.Command, _ []string) error {
			p.ran = true
			p.root, p.rootSeen = project.RootFrom(c.Context())
			return nil
		},
	})
	return cmd
}

// entry is one line of a directory snapshot.
type entry struct {
	mode  fs.FileMode
	size  int64
	mtime time.Time
}

// snapshot records path, mode, size and mtime of everything below dir.
//
// In: t, the test; dir, the directory to walk. Out: entries keyed by
// relative path. Side effects: none.
func snapshot(t *testing.T, dir string) map[string]entry {
	t.Helper()
	snap := map[string]entry{}
	err := filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			if d != nil && d.IsDir() {
				return nil // unreadable directory: its own entry was recorded
			}
			return err
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(dir, p)
		if err != nil {
			return err
		}
		snap[rel] = entry{info.Mode(), info.Size(), info.ModTime()}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return snap
}

// TestOutsideProjectGolden runs a non-exempt command in an empty directory:
// exit 1, empty stdout, the plain FR-003 block on stderr (golden file), the
// command's RunE never ran and the directory is unchanged.
//
// Covers: FR-002, FR-003, FR-004, EC-004
func TestOutsideProjectGolden(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "keep.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	// Snapshot after creating keep.txt: it is not a marker, so EC-004's
	// "all four missing" still holds.
	before := snapshot(t, dir)

	var p probe
	cmd := newTestRoot(t, func() (string, error) { return dir, nil }, &p)
	var out, errb bytes.Buffer
	code := run(context.Background(), cmd, []string{"probe"}, &out, &errb)

	if code != 1 {
		t.Errorf("exit code = %d, want 1", code)
	}
	if out.Len() != 0 {
		t.Errorf("stdout = %q, want empty", out.String())
	}
	if p.ran {
		t.Error("RunE ran outside a project")
	}

	golden := filepath.Join("testdata", "not-a-project.golden")
	if *update {
		if err := os.MkdirAll("testdata", 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(golden, errb.Bytes(), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(golden)
	if err != nil {
		t.Fatalf("read golden (run with -update to create): %v", err)
	}
	if !bytes.Equal(errb.Bytes(), want) {
		t.Errorf("stderr = %q, want %q", errb.String(), want)
	}

	wantUnchanged(t, dir, before)
}

// wantUnchanged fails the test if the directory differs from its snapshot.
//
// In: t, the test; dir, the directory; before, its earlier snapshot.
// Out: none. Side effects: none.
func wantUnchanged(t *testing.T, dir string, before map[string]entry) {
	t.Helper()
	after := snapshot(t, dir)
	if len(before) != len(after) {
		t.Fatalf("snapshot size changed: %d -> %d", len(before), len(after))
	}
	for k, b := range before {
		a, ok := after[k]
		if !ok || a != b {
			t.Errorf("%s changed: %+v -> %+v (present: %v)", k, b, a, ok)
		}
	}
}

// projectDir creates a full project skeleton (the four markers) in a temp dir.
//
// In: t, the test. Out: the directory. Side effects: writes files.
func projectDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	for _, m := range []string{"pyproject.toml", "alembic.ini", "app/main.py", "migrations/env.py"} {
		p := filepath.Join(dir, filepath.FromSlash(m))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

// runProbe runs the probe command through run with the given getwd stub.
//
// In: t, the test; getwd, the stub. Out: exit code, stdout, stderr and the
// probe. Side effects: none beyond the command's own.
func runProbe(t *testing.T, getwd func() (string, error)) (int, string, string, *probe) {
	t.Helper()
	p := &probe{}
	var out, errb bytes.Buffer
	code := run(context.Background(), newTestRoot(t, getwd, p), []string{"probe"}, &out, &errb)
	return code, out.String(), errb.String(), p
}

// wantFailure asserts exit 1, empty stdout, RunE not run and a stderr that
// equals exact, or starts with prefix when exact is empty.
func wantFailure(t *testing.T, code int, out, errs string, p *probe, exact, prefix string) {
	t.Helper()
	if code != 1 {
		t.Errorf("exit code = %d, want 1", code)
	}
	if out != "" {
		t.Errorf("stdout = %q, want empty", out)
	}
	if p.ran {
		t.Error("RunE ran")
	}
	if exact != "" && errs != exact {
		t.Errorf("stderr = %q, want %q", errs, exact)
	}
	if prefix != "" && !strings.HasPrefix(errs, prefix) {
		t.Errorf("stderr = %q, want prefix %q", errs, prefix)
	}
}

// TestInspectionPermissionDenied runs a command in a project whose app/ is
// unsearchable: the exact plain FR-006 message. Skipped as root.
//
// Covers: FR-004, FR-006, EC-007
func TestInspectionPermissionDenied(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("mode 000 is not enforced for root")
	}
	dir := projectDir(t)
	app := filepath.Join(dir, "app")
	if err := os.Chmod(app, 0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(app, 0o755) })
	before := snapshot(t, dir)

	code, out, errs, p := runProbe(t, func() (string, error) { return dir, nil })
	wantFailure(t, code, out, errs, p, "Error: cannot inspect app/main.py: permission denied\n", "")
	wantUnchanged(t, dir, before)
}

// TestInspectionCircularSymlink runs a command in a project whose
// alembic.ini links to itself: the message names the marker.
//
// Covers: FR-004, FR-006, EC-011
func TestInspectionCircularSymlink(t *testing.T) {
	dir := projectDir(t)
	ini := filepath.Join(dir, "alembic.ini")
	if err := os.Remove(ini); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(ini, ini); err != nil {
		t.Fatal(err)
	}
	before := snapshot(t, dir)
	code, out, errs, p := runProbe(t, func() (string, error) { return dir, nil })
	wantFailure(t, code, out, errs, p, "", "Error: cannot inspect alembic.ini: ")
	wantUnchanged(t, dir, before)
}

// TestInspectionGetwdFails runs a command whose working directory cannot be
// determined: the first FR-006 message, no "not a project" claim.
//
// Covers: FR-004, FR-006, EC-008
func TestInspectionGetwdFails(t *testing.T) {
	dir := projectDir(t)
	before := snapshot(t, dir)
	code, out, errs, p := runProbe(t, func() (string, error) { return "", syscall.ENOENT })
	wantUnchanged(t, dir, before)
	wantFailure(t, code, out, errs, p, "", "Error: cannot determine the current directory: ")
	if strings.Contains(errs, "not inside an aicli project") {
		t.Errorf("stderr claims not a project: %q", errs)
	}
}

// TestInsideProject runs a command in a project root: RunE runs and reads
// from its context a root equal to the value getwd returned, unresolved.
//
// Covers: FR-002, FR-009
func TestInsideProject(t *testing.T) {
	dir := projectDir(t)
	link := filepath.Join(t.TempDir(), "link")
	if err := os.Symlink(dir, link); err != nil {
		t.Fatal(err)
	}
	code, out, errs, p := runProbe(t, func() (string, error) { return link, nil })
	if code != 0 || out != "" || errs != "" {
		t.Fatalf("code=%d stdout=%q stderr=%q, want 0 and empty output", code, out, errs)
	}
	if !p.ran || !p.rootSeen || p.root != link {
		t.Fatalf("ran=%v rootSeen=%v root=%q, want true true %q", p.ran, p.rootSeen, p.root, link)
	}
}

// TestOwnPersistentPreRunEStillChecked runs a command that defines its own
// PersistentPreRunE outside a project: the check still fails it and neither
// the hook nor RunE runs.
//
// Covers: FR-002
func TestOwnPersistentPreRunEStillChecked(t *testing.T) {
	dir := t.TempDir()
	p := &probe{}
	var out, errb bytes.Buffer
	cmd := newTestRoot(t, func() (string, error) { return dir, nil }, p)
	code := run(context.Background(), cmd, []string{"guarded"}, &out, &errb)
	wantFailure(t, code, out.String(), errb.String(), p, "", "Error: not inside an aicli project\n")
	if p.ownHook {
		t.Error("the command's own PersistentPreRunE ran before the check failed")
	}

	// Inside a project both the check and the command's own hook run.
	proj := projectDir(t)
	p = &probe{}
	out.Reset()
	errb.Reset()
	cmd = newTestRoot(t, func() (string, error) { return proj, nil }, p)
	if code := run(context.Background(), cmd, []string{"guarded"}, &out, &errb); code != 0 {
		t.Fatalf("exit code = %d, stderr %q", code, errb.String())
	}
	if !p.ownHook || !p.ran || p.root != proj {
		t.Fatalf("ownHook=%v ran=%v root=%q, want true true %q", p.ownHook, p.ran, p.root, proj)
	}
}

// TestExemptCommandsOutsideProject runs every exempt form in a directory that
// is not a project: each succeeds and the check (the getwd stub) is never
// consulted.
//
// Covers: FR-007
func TestExemptCommandsOutsideProject(t *testing.T) {
	cases := map[string][]string{
		"new":              {"new", "myapp"},
		"help":             {"help"},
		"help probe":       {"help", "probe"},
		"probe --help":     {"probe", "--help"},
		"probe -h":         {"probe", "-h"},
		"completion bash":  {"completion", "bash"},
		"__complete probe": {"__complete", "probe", ""},
	}
	for name, args := range cases {
		t.Run(name, func(t *testing.T) {
			inTempDir(t)
			calls := 0
			getwd := func() (string, error) {
				calls++
				return t.TempDir(), nil
			}
			p := &probe{}
			var out, errb bytes.Buffer
			code := run(context.Background(), newTestRoot(t, getwd, p), args, &out, &errb)
			if code != 0 {
				t.Fatalf("exit code = %d, stderr %q", code, errb.String())
			}
			if calls != 0 {
				t.Errorf("the check ran %d time(s) for an exempt command", calls)
			}
			if p.ran {
				t.Error("the test command's RunE ran")
			}
		})
	}
}

// TestUnknownCommandNoCheck runs an unknown command outside a project: it
// gets Cobra's unknown-command error and the check is not run.
//
// Covers: FR-007, EC-014
func TestUnknownCommandNoCheck(t *testing.T) {
	calls := 0
	getwd := func() (string, error) {
		calls++
		return t.TempDir(), nil
	}
	p := &probe{}
	var out, errb bytes.Buffer
	code := run(context.Background(), newTestRoot(t, getwd, p), []string{"bogus"}, &out, &errb)
	if code != 1 || out.Len() != 0 {
		t.Fatalf("code=%d stdout=%q, want 1 and empty", code, out.String())
	}
	if want := `Error: unknown command "bogus" for "aicli"`; !strings.HasPrefix(errb.String(), want) {
		t.Errorf("stderr = %q, want prefix %q", errb.String(), want)
	}
	if calls != 0 {
		t.Errorf("the check ran %d time(s) for an unknown command", calls)
	}
}

// TestBareRootNoCheck runs `aicli` with no arguments and buffers outside a
// project: it prints the static home screen, exits 0 and does not run the
// check (the home screen is exempt).
//
// Covers: FR-001, EC-008
func TestBareRootNoCheck(t *testing.T) {
	calls := 0
	getwd := func() (string, error) {
		calls++
		return t.TempDir(), nil
	}
	var out, errb bytes.Buffer
	root := newTestRoot(t, getwd, &probe{})
	code := run(context.Background(), root, []string{}, &out, &errb)
	if code != 0 || errb.Len() != 0 {
		t.Fatalf("code=%d stderr=%q, want 0 and empty", code, errb.String())
	}
	if want := tui.RenderHomeStatic(home.Entries(root)); out.String() != want {
		t.Errorf("stdout = %q, want the static home screen %q", out.String(), want)
	}
	if calls != 0 {
		t.Errorf("the check ran %d time(s)", calls)
	}
}

// TestParseErrorsBeforeCheck runs a non-exempt command outside a project with
// an invalid flag and with an unexpected argument: Cobra's own error is
// reported, not the FR-003 message, and the check does not run, since it
// happens after Cobra has parsed flags and arguments.
//
// Covers: FR-002
func TestParseErrorsBeforeCheck(t *testing.T) {
	cases := map[string]struct {
		args []string
		want string
	}{
		"invalid flag":        {[]string{"probe", "--bogus"}, "Error: unknown flag: --bogus"},
		"unexpected argument": {[]string{"probe", "extra"}, `Error: unknown command "extra" for "aicli probe"`},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			calls := 0
			getwd := func() (string, error) {
				calls++
				return t.TempDir(), nil
			}
			p := &probe{}
			var out, errb bytes.Buffer
			code := run(context.Background(), newTestRoot(t, getwd, p), c.args, &out, &errb)
			if code != 1 || out.Len() != 0 {
				t.Fatalf("code=%d stdout=%q, want 1 and empty", code, out.String())
			}
			if !strings.HasPrefix(errb.String(), c.want) {
				t.Errorf("stderr = %q, want prefix %q", errb.String(), c.want)
			}
			if calls != 0 || p.ran {
				t.Errorf("check calls=%d ran=%v, want 0 false", calls, p.ran)
			}
		})
	}
}
