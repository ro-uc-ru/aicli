package root

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"
)

// TestRootHelpBaseline compares the plain `aicli --help` output with the
// baseline captured before the root became runnable: once the `aicli [flags]`
// Usage line, the only change allowed, is removed, they must be identical.
// In: t, the test; run with -update to regenerate the golden file.
// Out: none. Side effects: with -update, writes testdata/root-help-before.golden.
// Covers: EC-008.
func TestRootHelpBaseline(t *testing.T) {
	var out, errb bytes.Buffer
	if code := Execute(context.Background(), []string{"--help"}, &out, &errb); code != 0 {
		t.Fatalf("exit code = %d, stderr = %q", code, errb.String())
	}
	golden := filepath.Join("testdata", "root-help-before.golden")
	if *update {
		if err := os.MkdirAll("testdata", 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(golden, out.Bytes(), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(golden)
	if err != nil {
		t.Fatalf("read golden (run with -update to create): %v", err)
	}
	got := bytes.Replace(out.Bytes(), []byte("  aicli [flags]\n"), nil, 1)
	if !bytes.Equal(got, want) {
		t.Errorf("root help = %q, want %q", out.String(), want)
	}
}
