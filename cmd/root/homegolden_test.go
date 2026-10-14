package root

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// compareGolden compares got with a golden file of testdata.
//
// In: t, the test; name, the golden file name; got, the output. Out: none;
// the test fails on a difference or a missing file. Side effects: with
// -update, writes testdata/<name>.
func compareGolden(t *testing.T, name string, got []byte) {
	t.Helper()
	golden := filepath.Join("testdata", name)
	if *update {
		if err := os.MkdirAll("testdata", 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(golden, got, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(golden)
	if err != nil {
		t.Fatalf("read golden (run with -update to create): %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Errorf("output differs from %s:\n%s\n--- want:\n%s", golden, got, want)
	}
}

// TestStaticHomeGoldenRealTree runs the real `aicli` with no arguments on
// buffers (static mode) and compares stdout with the golden file.
//
// In: t, the test; run with -update to regenerate the golden file.
// Out: none. Side effects: with -update, writes testdata/home-real.golden.
// Covers: FR-001, FR-012
func TestStaticHomeGoldenRealTree(t *testing.T) {
	var out, errb bytes.Buffer
	if code := Execute(context.Background(), []string{}, &out, &errb); code != 0 || errb.Len() != 0 {
		t.Fatalf("code=%d stderr=%q, want 0 and empty", code, errb.String())
	}
	compareGolden(t, "home-real.golden", out.Bytes())
}

// TestStaticHomeGoldenTestTree runs the test tree, which registers `probe`
// only in tests, with no arguments and compares stdout with the golden
// file: `probe` is listed with no change to the feature's code.
//
// In: t, the test; run with -update to regenerate the golden file.
// Out: none. Side effects: with -update, writes testdata/home-testtree.golden.
// Covers: FR-004, FR-012
func TestStaticHomeGoldenTestTree(t *testing.T) {
	getwd := func() (string, error) { return t.TempDir(), nil }
	var out, errb bytes.Buffer
	if code := run(context.Background(), newTestRoot(t, getwd, &probe{}), []string{}, &out, &errb); code != 0 || errb.Len() != 0 {
		t.Fatalf("code=%d stderr=%q, want 0 and empty", code, errb.String())
	}
	if !bytes.Contains(out.Bytes(), []byte("probe")) {
		t.Fatalf("probe is not listed:\n%s", out.String())
	}
	compareGolden(t, "home-testtree.golden", out.Bytes())
}

// TestBuiltBinaryHome builds the binary and runs `aicli` with no arguments,
// stdin and stdout not terminals: exit 0 and the real-tree golden.
//
// In: t, the test. Out: none.
// Side effects: builds a binary and runs it in a temp directory.
// Covers: FR-001, EC-009
func TestBuiltBinaryHome(t *testing.T) {
	bin := filepath.Join(t.TempDir(), "aicli")
	if out, err := exec.Command("go", "build", "-o", bin, "github.com/ro-uc-ru/aicli").CombinedOutput(); err != nil {
		t.Fatalf("build: %v\n%s", err, out)
	}
	run := exec.Command(bin)
	run.Dir = t.TempDir()
	var out, errb bytes.Buffer
	run.Stdout, run.Stderr = &out, &errb
	if err := run.Run(); err != nil {
		t.Fatalf("run: %v, stderr %q", err, errb.String())
	}
	if errb.Len() != 0 {
		t.Errorf("stderr = %q, want empty", errb.String())
	}
	want, err := os.ReadFile(filepath.Join("testdata", "home-real.golden"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(out.Bytes(), want) {
		t.Errorf("stdout differs from home-real.golden:\n%s", out.String())
	}
}
