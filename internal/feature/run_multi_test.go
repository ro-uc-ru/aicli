package feature_test

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ro-uc-ru/aicli/internal/feature"
	"github.com/ro-uc-ru/aicli/internal/featuretest"
	"github.com/ro-uc-ru/aicli/internal/project"
)

// multiRoot builds a temp project root with the two injection targets of the
// multi-choice test feature, each holding its marker.
//
// In: t, the test. Out: the root and a context carrying it.
// Side effects: writes the target files under a temp dir.
func multiRoot(t *testing.T) (string, context.Context) {
	t.Helper()
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "targets"), 0o755); err != nil {
		t.Fatal(err)
	}
	for p, m := range map[string]string{featuretest.MultiTargetA: featuretest.MultiMarkerA, featuretest.MultiTargetB: featuretest.MultiMarkerB} {
		if err := os.WriteFile(filepath.Join(root, p), []byte("top\n# aicli:"+m+"\nbottom\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root, project.WithRoot(context.Background(), root)
}

// runMulti runs the multi-choice test feature with args.
//
// In: ctx, the context with the root; args, the positional arguments.
// Out: the report written and the error. Side effects: those of Run.
func runMulti(ctx context.Context, args ...string) (string, error) {
	var out bytes.Buffer
	err := feature.Run(ctx, featuretest.NewMulti(), args, &out, func(string) string { return "" })
	return out.String(), err
}

// snapshot reads every regular file under root.
//
// In: t, the test; root, the directory. Out: path to content.
// No side effects.
func snapshot(t *testing.T, root string) map[string]string {
	t.Helper()
	m := map[string]string{}
	err := filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		b, err := os.ReadFile(p)
		m[p] = string(b)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	return m
}

// TestRunMultiOrderAndSteps checks that values in any order give the same
// files and the same report, and that the next steps depend on the values.
//
// Covers: FR-004, FR-011, FR-014
func TestRunMultiOrderAndSteps(t *testing.T) {
	rootA, ctxA := multiRoot(t)
	outA, err := runMulti(ctxA, "gamma", "alpha")
	if err != nil {
		t.Fatal(err)
	}
	rootB, ctxB := multiRoot(t)
	outB, err := runMulti(ctxB, "alpha", "gamma")
	if err != nil {
		t.Fatal(err)
	}
	if outA != outB {
		t.Fatalf("reports differ:\n%s\n%s", outA, outB)
	}
	a, b := snapshot(t, rootA), snapshot(t, rootB)
	if len(a) != len(b) {
		t.Fatalf("file sets differ: %d vs %d", len(a), len(b))
	}
	for p, c := range a {
		rel, _ := filepath.Rel(rootA, p)
		if b[filepath.Join(rootB, rel)] != c {
			t.Fatalf("%s differs", rel)
		}
	}
	for _, want := range []string{
		"create  shared/one.txt", "create  items/alpha.txt", "update  targets/a.txt", "update  targets/b.txt",
		"Set alpha in .env\n  Set gamma in .env",
	} {
		if !strings.Contains(outA, want) {
			t.Fatalf("report lacks %q:\n%s", want, outA)
		}
	}
	if strings.Contains(outA, "beta") {
		t.Fatalf("report mentions an unchosen value:\n%s", outA)
	}
	got := a[filepath.Join(rootA, featuretest.MultiTargetA)]
	if got != "top\n# aicli:first\nline alpha\nline gamma\nbottom\n" {
		t.Fatalf("target = %q", got)
	}
}

// TestRunMultiAlreadyAdded checks that a value whose item file exists fails
// and creates, modifies and prints nothing, while a later run with another
// value reports the shared files as skipped.
//
// Covers: FR-004, FR-011
func TestRunMultiAlreadyAdded(t *testing.T) {
	root, ctx := multiRoot(t)
	if _, err := runMulti(ctx, "alpha"); err != nil {
		t.Fatal(err)
	}
	before := snapshot(t, root)
	out, err := runMulti(ctx, "alpha", "beta")
	if err == nil || !strings.Contains(err.Error(), "already added") || !strings.Contains(err.Error(), "alpha") {
		t.Fatalf("err = %v", err)
	}
	if out != "" {
		t.Fatalf("out = %q, want empty", out)
	}
	after := snapshot(t, root)
	if len(after) != len(before) {
		t.Fatalf("files changed: %d vs %d", len(after), len(before))
	}
	for p, c := range before {
		if after[p] != c {
			t.Fatalf("%s was modified", p)
		}
	}
	out, err = runMulti(ctx, "beta")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "skip    shared/one.txt") || !strings.Contains(out, "create  items/beta.txt") {
		t.Fatalf("report:\n%s", out)
	}
}
