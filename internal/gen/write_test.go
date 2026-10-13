package gen

import (
	"bytes"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
)

// snapshot maps every path under dir to its mode and content.
// In: t, the running test; dir, the root. Out: path -> "mode:content".
// Side effects: fails the test on walk errors.
func snapshot(t *testing.T, dir string) map[string]string {
	t.Helper()
	out := map[string]string{}
	err := filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		var content []byte
		if !d.IsDir() {
			if content, err = os.ReadFile(p); err != nil {
				return err
			}
		}
		out[p[len(dir):]] = info.Mode().String() + ":" + string(content)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return out
}

// writeUnderUmask renders and writes a project under the given umask.
// In: t, the running test; umask, the process umask to apply.
// Out: a snapshot of the written tree. Side effects: changes and restores the umask.
func writeUnderUmask(t *testing.T, umask int) map[string]string {
	old := syscall.Umask(umask)
	defer syscall.Umask(old)
	entries, err := Render("myapp")
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	if err := Write(OSFS{}, dir, entries); err != nil {
		t.Fatal(err)
	}
	return snapshot(t, dir)
}

// TestWriteModesAndDeterminism checks 0755/0644 under umask 077 and 022 and
// byte-identical output across runs.
// In: t, the running test. Out: none. Side effects: temp directories.
// Covers: FR-022.
func TestWriteModesAndDeterminism(t *testing.T) {
	a := writeUnderUmask(t, 0o077)
	b := writeUnderUmask(t, 0o022)
	c := writeUnderUmask(t, 0o022)
	for name, snaps := range map[string]map[string]string{"077": a, "022": b} {
		for p, v := range snaps {
			mode := v[:strings.Index(v, ":")]
			if p == "" {
				continue // the temp root itself
			}
			want := "-rw-r--r--"
			if strings.HasPrefix(mode, "d") {
				want = "drwxr-xr-x"
			}
			if mode != want {
				t.Errorf("umask %s: %s has mode %s, want %s", name, p, mode, want)
			}
		}
	}
	delete(a, "")
	delete(b, "")
	delete(c, "")
	if !equal(a, b) || !equal(b, c) {
		t.Error("runs are not byte-identical")
	}
}

// equal reports whether two snapshots are identical.
// In: a, b, the snapshots. Out: true if equal. Side effects: none.
func equal(a, b map[string]string) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if b[k] != v {
			return false
		}
	}
	return true
}

// failFS wraps OSFS and fails the Nth OpenFile call.
type failFS struct {
	OSFS
	n, calls int
}

// OpenFile fails with errInjected on the configured call.
// In: name, flag, perm as in FS. Out: a handle, or errInjected. Side effects: may create a file.
func (f *failFS) OpenFile(name string, flag int, perm os.FileMode) (io.WriteCloser, error) {
	f.calls++
	if f.calls == f.n {
		return nil, errInjected
	}
	return f.OSFS.OpenFile(name, flag, perm)
}

// errInjected is the error forced by failFS.
var errInjected = errors.New("injected failure")

// TestWriteInjectedFailure checks that a failing write returns an error with context.
// In: t, the running test. Out: none. Side effects: temp directory.
// Covers: FR-016.
func TestWriteInjectedFailure(t *testing.T) {
	entries, err := Render("myapp")
	if err != nil {
		t.Fatal(err)
	}
	err = Write(&failFS{n: 3}, t.TempDir(), entries)
	if !errors.Is(err, errInjected) || !strings.Contains(err.Error(), "create file ") {
		t.Errorf("want wrapped injected error with path context, got %v", err)
	}
}

// TestWriteNoOverwrite checks that an existing file is never overwritten.
// In: t, the running test. Out: none. Side effects: temp directory.
// Covers: FR-016.
func TestWriteNoOverwrite(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "a.txt")
	if err := os.WriteFile(p, []byte("keep"), 0o600); err != nil {
		t.Fatal(err)
	}
	err := Write(OSFS{}, dir, []Entry{{Path: "a.txt", Content: []byte("new"), Kind: KindFile}})
	if err == nil {
		t.Fatal("want error")
	}
	if b, _ := os.ReadFile(p); !bytes.Equal(b, []byte("keep")) {
		t.Error("file was overwritten")
	}
}
