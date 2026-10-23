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
	"github.com/ro-uc-ru/aicli/internal/tui"
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
	pv, err := f.Check([]string{"a", "x"})
	if err != nil || !reflect.DeepEqual(pv, tui.Preview{Create: paths}) {
		t.Fatalf("Check on free paths = %+v, %v", pv, err)
	}
	if err := os.MkdirAll(filepath.Join(root, "features"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "features", "x.md"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	_, err = f.Check([]string{"a", "x"})
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
	if _, err := Features(add)[0].Check([]string{"a"}); err == nil {
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

// multiTree builds an add tree with the multi-choice test feature and a temp
// project root holding both injection targets with their markers.
//
// In: t, the test. Out: the tree and the root. Side effects: writes the
// target files under a temp dir and sets the tree context.
func multiTree(t *testing.T) (*cobra.Command, string) {
	t.Helper()
	add := featuretest.NewTree(featuretest.NewMulti())
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "targets"), 0o755); err != nil {
		t.Fatal(err)
	}
	for p, m := range map[string]string{featuretest.MultiTargetA: featuretest.MultiMarkerA, featuretest.MultiTargetB: featuretest.MultiMarkerB} {
		if err := os.WriteFile(filepath.Join(root, p), []byte("top\n# aicli:"+m+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	add.SetContext(project.WithRoot(context.Background(), root))
	return add, root
}

// TestFeaturesMultiAddedAndCheck checks the multi-choice data of a feature,
// that Added counts a file, a directory and a broken link as added, that
// Check returns the preview of the plan and that neither writes.
//
// Covers: FR-007, FR-011, EC-005
func TestFeaturesMultiAddedAndCheck(t *testing.T) {
	add, root := multiTree(t)
	f := Features(add)[0]
	in := f.Inputs[0]
	if !in.Multi || in.ItemNoun != featuretest.MultiNoun || !reflect.DeepEqual(in.Choices, featuretest.MultiChoices) {
		t.Fatalf("input = %+v", in)
	}
	if got := f.Added(); !reflect.DeepEqual(got, []bool{false, false, false}) {
		t.Fatalf("Added on empty project = %v", got)
	}
	if err := os.MkdirAll(filepath.Join(root, "items"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, featuretest.MultiItemFile("alpha")), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(root, featuretest.MultiItemFile("beta")), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("missing", filepath.Join(root, featuretest.MultiItemFile("gamma"))); err != nil {
		t.Fatal(err)
	}
	if got := f.Added(); !reflect.DeepEqual(got, []bool{true, true, true}) {
		t.Fatalf("Added = %v, want all true", got)
	}
	if err := os.Remove(filepath.Join(root, featuretest.MultiItemFile("alpha"))); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(root, featuretest.MultiItemFile("beta"))); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(root, featuretest.MultiItemFile("gamma"))); err != nil {
		t.Fatal(err)
	}
	before := snapshot(t, root)
	pv, err := f.Check([]string{"alpha", "beta"})
	if err != nil {
		t.Fatalf("Check = %v", err)
	}
	want := tui.Preview{
		Create: []string{"shared/one.txt", "shared/two.txt", "items/alpha.txt", "items/alpha.md", "items/beta.txt", "items/beta.md"},
		Update: []string{featuretest.MultiTargetA, featuretest.MultiTargetB},
	}
	if !reflect.DeepEqual(pv, want) {
		t.Fatalf("Check = %+v, want %+v", pv, want)
	}
	if after := snapshot(t, root); !reflect.DeepEqual(before, after) {
		t.Fatalf("Check changed the tree:\n%v\n%v", before, after)
	}
	if err := os.WriteFile(filepath.Join(root, featuretest.MultiItemFile("beta")), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	before = snapshot(t, root)
	if _, err := f.Check([]string{"alpha", "beta"}); err == nil || !strings.Contains(err.Error(), "already added") {
		t.Fatalf("Check with added item = %v", err)
	}
	if after := snapshot(t, root); !reflect.DeepEqual(before, after) {
		t.Fatalf("Check changed the tree:\n%v\n%v", before, after)
	}
}

// TestFeaturesAddedNilWithoutItems checks that a feature with no item files
// has no Added function.
//
// Covers: FR-007
func TestFeaturesAddedNilWithoutItems(t *testing.T) {
	if f := Features(featuretest.NewTree(featuretest.NewMain()))[0]; f.Added != nil {
		t.Fatal("Added is set for a feature with no item files")
	}
}
