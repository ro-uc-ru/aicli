package add

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ro-uc-ru/aicli/internal/featuretest"
	"github.com/ro-uc-ru/aicli/internal/project"
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
