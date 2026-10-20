package add

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/ro-uc-ru/aicli/internal/featuretest"
	"github.com/ro-uc-ru/aicli/internal/project"
)

// TestFeaturesOrderAndListing checks that all features come in Cobra order and
// that hidden and deprecated ones are not listed.
//
// Covers: FR-002, EC-002
func TestFeaturesOrderAndListing(t *testing.T) {
	add := featuretest.NewTree(featuretest.Valid()...)
	got := Features(add)
	var all, listed []string
	for _, f := range got {
		all = append(all, f.Name)
		if f.Listed {
			listed = append(listed, f.Name)
		}
	}
	var want []string
	for _, c := range add.Commands() {
		want = append(want, c.Name())
	}
	if !reflect.DeepEqual(all, want) || len(all) != 6 {
		t.Fatalf("features = %v, want %v", all, want)
	}
	wantListed := []string{featuretest.Main, featuretest.NoInputs, featuretest.NoShort, featuretest.Wide}
	if !reflect.DeepEqual(sorted(listed), sorted(wantListed)) {
		t.Fatalf("listed = %v, want %v", listed, wantListed)
	}
}

// sorted returns a sorted copy of s.
func sorted(s []string) []string {
	c := append([]string(nil), s...)
	for i := range c {
		for j := i + 1; j < len(c); j++ {
			if c[j] < c[i] {
				c[i], c[j] = c[j], c[i]
			}
		}
	}
	return c
}

// TestFeaturesNoShort checks that a feature without a short description has an
// empty one.
//
// Covers: EC-007
func TestFeaturesNoShort(t *testing.T) {
	for _, f := range Features(featuretest.NewTree(featuretest.NewNoShort())) {
		if f.Name == featuretest.NoShort && f.Short != "" {
			t.Fatalf("Short = %q, want empty", f.Short)
		}
	}
}

// TestFeaturesReadTreeEachCall checks that a feature added to the tree after a
// first call appears on the next call.
//
// Covers: FR-002
func TestFeaturesReadTreeEachCall(t *testing.T) {
	add := featuretest.NewTree(featuretest.NewNoInputs())
	if n := len(Features(add)); n != 1 {
		t.Fatalf("first call = %d features, want 1", n)
	}
	featuretest.Register(add, featuretest.NewMain())
	got := Features(add)
	if len(got) != 2 {
		t.Fatalf("second call = %d features, want 2", len(got))
	}
	found := false
	for _, f := range got {
		found = found || f.Name == featuretest.Main
	}
	if !found {
		t.Fatal("added feature missing")
	}
}

// TestFeaturesInputsPlanCheck checks inputs, the paths-only Plan and the
// read-only Check with the project root.
//
// Covers: FR-002, EC-002
func TestFeaturesInputsPlanCheck(t *testing.T) {
	add := featuretest.NewTree(featuretest.NewMain())
	root := t.TempDir()
	add.SetContext(project.WithRoot(context.Background(), root))
	f := Features(add)[0]
	if len(f.Inputs) != 2 || f.Inputs[0].Name != "input-1" || len(f.Inputs[0].Choices) != 3 || f.Inputs[1].Validator == nil {
		t.Fatalf("inputs = %+v", f.Inputs)
	}
	paths, err := f.Plan([]string{"one", "x"})
	if err != nil || !reflect.DeepEqual(paths, []string{"features/x.txt", "features/x.md"}) {
		t.Fatalf("Plan = %v, %v", paths, err)
	}
	if _, err := f.Plan([]string{featuretest.FailChoice, "x"}); err == nil || err.Error() != featuretest.PlanError {
		t.Fatalf("Plan error = %v", err)
	}
	if err := f.Check(paths); err != nil {
		t.Fatalf("Check on free paths = %v", err)
	}
	if err := os.MkdirAll(filepath.Join(root, "features"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "features", "x.md"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	err = f.Check(paths)
	if err == nil || !strings.Contains(err.Error(), "features/x.md") {
		t.Fatalf("Check on existing file = %v", err)
	}
	if _, statErr := os.Lstat(filepath.Join(root, "features", "x.txt")); statErr == nil {
		t.Fatal("Check wrote a file")
	}
}

// TestFeaturesCheckWithoutRoot checks that Check reports a missing project
// root instead of panicking.
//
// Covers: FR-002
func TestFeaturesCheckWithoutRoot(t *testing.T) {
	add := featuretest.NewTree(featuretest.NewMain())
	if err := Features(add)[0].Check([]string{"a"}); err == nil {
		t.Fatal("want an error without a root")
	}
}

// TestFeaturesSkipsUndeclared checks that a plain Cobra subcommand is skipped.
//
// Covers: FR-002
func TestFeaturesSkipsUndeclared(t *testing.T) {
	add := featuretest.NewTree(featuretest.NewMain())
	add.AddCommand(&cobra.Command{Use: "plain", Run: func(*cobra.Command, []string) {}})
	if n := len(Features(add)); n != 1 {
		t.Fatalf("got %d features, want 1", n)
	}
}
