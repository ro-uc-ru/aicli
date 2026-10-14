package project

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"syscall"
	"testing"
)

// fullProject writes the four markers as regular files in a new temp dir.
func fullProject(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	for _, m := range Markers() {
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

// checkDir runs Check with a getwd stub returning dir.
func checkDir(dir string) (string, error) {
	return Check(func() (string, error) { return dir, nil })
}

// wantMissing asserts that err is a *NotProjectError listing exactly want.
func wantMissing(t *testing.T, err error, want ...string) {
	t.Helper()
	var np *NotProjectError
	if !errors.As(err, &np) {
		t.Fatalf("err = %v (%T), want *NotProjectError", err, err)
	}
	if !slices.Equal(np.Missing, want) {
		t.Fatalf("Missing = %v, want %v", np.Missing, want)
	}
}

// replace removes the marker and puts mk(path) in its place.
func replace(t *testing.T, dir, marker string, mk func(path string) error) {
	t.Helper()
	p := filepath.Join(dir, filepath.FromSlash(marker))
	if err := os.RemoveAll(p); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := mk(p); err != nil {
		t.Fatal(err)
	}
}

// TestCheckFullProject covers a complete project, edited contents and extra
// files, and the returned root.
//
// Covers: FR-001, FR-005, EC-009
func TestCheckFullProject(t *testing.T) {
	dir := fullProject(t)
	if err := os.WriteFile(filepath.Join(dir, "app", "main.py"), []byte("syntax error ((("), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "extra", "deep"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "README"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	root, err := checkDir(dir)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if root != dir {
		t.Fatalf("root = %q, want %q", root, dir)
	}
}

// TestCheckEmptyDir covers an empty directory: all four markers missing.
//
// Covers: FR-001, EC-004
func TestCheckEmptyDir(t *testing.T) {
	_, err := checkDir(t.TempDir())
	wantMissing(t, err, Markers()...)
}

// TestCheckSubsets covers every subset of present markers, each listing
// exactly the missing ones in order.
//
// Covers: FR-001, EC-005
func TestCheckSubsets(t *testing.T) {
	all := Markers()
	for mask := 0; mask < 1<<len(all); mask++ {
		dir := t.TempDir()
		var want []string
		for i, m := range all {
			if mask&(1<<i) == 0 {
				want = append(want, m)
				continue
			}
			replace(t, dir, m, func(p string) error { return os.WriteFile(p, nil, 0o644) })
		}
		root, err := checkDir(dir)
		if len(want) == 0 {
			if err != nil || root != dir {
				t.Fatalf("mask %d: root=%q err=%v", mask, root, err)
			}
			continue
		}
		wantMissing(t, err, want...)
	}
}

// TestCheckNonRegularMarkers covers a marker that is a directory, a symlink
// to a file, a symlink to a directory, a broken symlink and a FIFO.
//
// Covers: FR-001, EC-002, EC-003, EC-013
func TestCheckNonRegularMarkers(t *testing.T) {
	const m = "alembic.ini"
	cases := []struct {
		name    string
		mk      func(dir, path string) error
		missing bool
	}{
		{"directory", func(_, p string) error { return os.Mkdir(p, 0o755) }, true},
		{"symlink to file", func(d, p string) error {
			target := filepath.Join(d, "target.ini")
			if err := os.WriteFile(target, nil, 0o644); err != nil {
				return err
			}
			return os.Symlink(target, p)
		}, false},
		{"symlink to directory", func(d, p string) error { return os.Symlink(d, p) }, true},
		{"broken symlink", func(d, p string) error { return os.Symlink(filepath.Join(d, "nope"), p) }, true},
		{"fifo", func(_, p string) error { return syscall.Mkfifo(p, 0o644) }, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			dir := fullProject(t)
			replace(t, dir, m, func(p string) error { return c.mk(dir, p) })
			_, err := checkDir(dir)
			if c.missing {
				wantMissing(t, err, m)
			} else if err != nil {
				t.Fatalf("Check: %v", err)
			}
		})
	}
}

// TestCheckParentIsFile covers `app` being a regular file.
//
// Covers: FR-001, EC-010
func TestCheckParentIsFile(t *testing.T) {
	dir := fullProject(t)
	if err := os.RemoveAll(filepath.Join(dir, "app")); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "app"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := checkDir(dir)
	wantMissing(t, err, "app/main.py")
}

// TestCheckNestedAndSubdir covers a project nested inside another one and a
// subdirectory of a project (no parent is consulted).
//
// Covers: FR-001, EC-001, EC-006
func TestCheckNestedAndSubdir(t *testing.T) {
	outer := fullProject(t)
	inner := filepath.Join(outer, "inner")
	if err := os.Mkdir(inner, 0o755); err != nil {
		t.Fatal(err)
	}
	_, err := checkDir(inner)
	wantMissing(t, err, Markers()...)

	for _, m := range Markers() {
		p := filepath.Join(inner, filepath.FromSlash(m))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, nil, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if root, err := checkDir(inner); err != nil || root != inner {
		t.Fatalf("nested: root=%q err=%v", root, err)
	}

	sub := filepath.Join(outer, "app", "services")
	if err := os.Mkdir(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	_, err = checkDir(sub)
	wantMissing(t, err, Markers()...)
}

// TestCheckRootUnchangedThroughSymlink covers the root being returned
// exactly as getwd gave it, without resolving symbolic links.
//
// Covers: FR-001, FR-009, EC-006
func TestCheckRootUnchangedThroughSymlink(t *testing.T) {
	dir := fullProject(t)
	link := filepath.Join(t.TempDir(), "link")
	if err := os.Symlink(dir, link); err != nil {
		t.Fatal(err)
	}
	root, err := checkDir(link)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if root != link {
		t.Fatalf("root = %q, want %q", root, link)
	}
}

// denyApp makes the app directory of a project unsearchable and restores its
// mode on cleanup so the temp dir can be removed. It skips the test as root,
// where mode 000 is not enforced.
func denyApp(t *testing.T, dir string) {
	t.Helper()
	if os.Geteuid() == 0 {
		t.Skip("mode 000 is not enforced for root")
	}
	app := filepath.Join(dir, "app")
	if err := os.Chmod(app, 0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(app, 0o755) })
}

// wantInspect asserts that err is a *InspectError for path and returns it.
func wantInspect(t *testing.T, err error, path string) *InspectError {
	t.Helper()
	var ie *InspectError
	if !errors.As(err, &ie) {
		t.Fatalf("err = %v (%T), want *InspectError", err, err)
	}
	if ie.Path != path {
		t.Fatalf("Path = %q, want %q", ie.Path, path)
	}
	return ie
}

// TestCheckGetwdFails covers a failing getwd: its error is returned as a
// working-directory error and no marker is inspected, even though the
// project in the directory would otherwise yield an inspection failure.
//
// Covers: FR-006, EC-008
func TestCheckGetwdFails(t *testing.T) {
	dir := fullProject(t)
	denyApp(t, dir)
	root, err := Check(func() (string, error) { return dir, syscall.ENOENT })
	if root != "" {
		t.Fatalf("root = %q, want empty", root)
	}
	var we *WorkingDirError
	if !errors.As(err, &we) {
		t.Fatalf("err = %v (%T), want *WorkingDirError", err, err)
	}
	want := "cannot determine the current directory: no such file or directory"
	if err.Error() != want {
		t.Fatalf("Error() = %q, want %q", err.Error(), want)
	}
}

// TestCheckPermissionDenied covers an unsearchable app/ directory: an
// inspection failure with the exact message, not a missing marker.
//
// Covers: FR-006, EC-007
func TestCheckPermissionDenied(t *testing.T) {
	dir := fullProject(t)
	denyApp(t, dir)
	_, err := checkDir(dir)
	_ = wantInspect(t, err, "app/main.py")
	if want := "cannot inspect app/main.py: permission denied"; err.Error() != want {
		t.Fatalf("Error() = %q, want %q", err.Error(), want)
	}
}

// TestCheckCircularSymlink covers alembic.ini being a symbolic link to
// itself: an inspection failure naming the marker.
//
// Covers: FR-006, EC-011
func TestCheckCircularSymlink(t *testing.T) {
	dir := fullProject(t)
	replace(t, dir, "alembic.ini", func(p string) error { return os.Symlink(p, p) })
	_, err := checkDir(dir)
	ie := wantInspect(t, err, "alembic.ini")
	if !errors.Is(ie, syscall.ELOOP) {
		t.Fatalf("cause = %v, want ELOOP", ie.Err)
	}
	if got, prefix := err.Error(), "cannot inspect alembic.ini: "; len(got) <= len(prefix) || got[:len(prefix)] != prefix {
		t.Fatalf("Error() = %q, want prefix %q", got, prefix)
	}
}

// TestCheckInspectionWinsOverMissing covers pyproject.toml missing plus an
// unsearchable app/: the inspection failure is reported and the "not a
// project" error is not.
//
// Covers: FR-006, EC-012
func TestCheckInspectionWinsOverMissing(t *testing.T) {
	dir := fullProject(t)
	if err := os.Remove(filepath.Join(dir, "pyproject.toml")); err != nil {
		t.Fatal(err)
	}
	denyApp(t, dir)
	_, err := checkDir(dir)
	_ = wantInspect(t, err, "app/main.py")
	var np *NotProjectError
	if errors.As(err, &np) {
		t.Fatal("got a *NotProjectError alongside the inspection failure")
	}
}

// TestCheckGetwdCauseUnwrapped covers the errors os.Getwd returns on Linux
// (*os.SyscallError) and on other paths (*fs.PathError): the message carries
// only the system error text.
//
// Covers: FR-006, EC-008
func TestCheckGetwdCauseUnwrapped(t *testing.T) {
	want := "cannot determine the current directory: no such file or directory"
	for name, in := range map[string]error{
		"syscall error": os.NewSyscallError("getwd", syscall.ENOENT),
		"path error":    &fs.PathError{Op: "getwd", Path: ".", Err: syscall.ENOENT},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := Check(func() (string, error) { return "", in })
			if err == nil || err.Error() != want {
				t.Fatalf("err = %v, want %q", err, want)
			}
		})
	}
}
