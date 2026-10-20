package tui

import (
	"os"
	"strings"
	"testing"
)

// staticFeatures mirrors the test features of internal/featuretest (which
// cannot be imported here without a cycle): a normal one, one without a short
// description, one with wide characters, and a hidden one that is not listed.
var staticFeatures = []Feature{
	{Name: "feature-1", Short: "Add the main test feature", Listed: true},
	{Name: "feature-no-short", Listed: true},
	{Name: "機能-wide", Short: "広い説明 😀", Listed: true},
	{Name: "feature-hidden", Short: "Hidden one", Listed: false},
}

// checkStaticGolden compares got with testdata/<name>.golden, rewriting it
// with -update, and checks that got has no ESC byte.
//
// In: t, the running test; name, the golden name; got, the output.
// Out: none. Side effects: with -update, rewrites the golden file.
func checkStaticGolden(t *testing.T, name, got string) {
	t.Helper()
	path := "testdata/" + name + ".golden"
	if updating() {
		if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if got != string(want) {
		t.Fatalf("%s differs from golden:\n%s", name, got)
	}
	if strings.ContainsRune(got, '\x1b') {
		t.Fatalf("%s contains an ESC byte", name)
	}
}

// TestRenderAddStatic checks the static list for the test features, including
// a wide-character name and a feature without description, and that unlisted
// features are left out.
//
// Covers: FR-005, EC-006, EC-007
func TestRenderAddStatic(t *testing.T) {
	checkStaticGolden(t, "add-static", RenderAddStatic(staticFeatures))
}

// TestRenderAddStaticEmpty checks the EC-003 text for an empty list and for a
// list with no listed feature.
//
// Covers: FR-005, EC-003
func TestRenderAddStaticEmpty(t *testing.T) {
	checkStaticGolden(t, "add-static-empty", RenderAddStatic(nil))
	if got := RenderAddStatic([]Feature{{Name: "x", Listed: false}}); got != RenderAddStatic(nil) {
		t.Fatalf("unlisted-only output = %q", got)
	}
}
