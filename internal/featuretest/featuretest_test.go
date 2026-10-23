package featuretest

import (
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/ro-uc-ru/aicli/internal/feature"
)

// TestValidFeaturesFollowContract checks that CheckContract gives no error on
// a tree with every valid test feature, and that the features have the
// documented shape.
//
// Covers: FR-002, FR-003, EC-002, EC-006, EC-007, EC-008
func TestValidFeaturesFollowContract(t *testing.T) {
	add := NewTree(Valid()...)
	if errs := feature.CheckContract(add); len(errs) != 0 {
		t.Fatalf("valid tree gave errors: %v", errs)
	}
	byName := map[string]*cobra.Command{}
	for _, c := range add.Commands() {
		byName[c.Name()] = c
	}
	if c := byName[Hidden]; c == nil || !c.Hidden {
		t.Fatal("hidden feature is not hidden")
	}
	if c := byName[Deprecated]; c == nil || c.Deprecated == "" || c.IsAvailableCommand() {
		t.Fatal("deprecated feature is not unavailable")
	}
	if d, _ := feature.Lookup(byName[NoInputs]); len(d.Inputs) != 0 {
		t.Fatal("NoInputs declares inputs")
	}
	if byName[NoShort].Short != "" {
		t.Fatal("NoShort has a short description")
	}
	if len(byName[Wide].Name()) == len([]rune(byName[Wide].Name())) {
		t.Fatal("Wide has no multi-byte characters")
	}
	files, err := mustDecl(t, byName[Main]).Plan([]string{"one", "x"})
	if err != nil || len(files) != 2 {
		t.Fatalf("main plan = %v, %v", files, err)
	}
	if _, err := mustDecl(t, byName[Main]).Plan([]string{FailChoice, "x"}); err == nil || err.Error() != PlanError {
		t.Fatalf("fail choice plan error = %v", err)
	}
}

// mustDecl returns the declaration of cmd or fails the test.
//
// In: t, the test; cmd, a feature command. Out: its declaration.
// Side effects: fails t if cmd has none.
func mustDecl(t *testing.T, cmd *cobra.Command) *feature.Declaration {
	t.Helper()
	d, ok := feature.Lookup(cmd)
	if !ok {
		t.Fatalf("%s has no declaration", cmd.Name())
	}
	return d
}

// TestBrokenFeaturesGiveExactlyTheirRule checks that, next to the valid
// features, each rule-breaking feature gives exactly one error that names it
// and its rule.
//
// Covers: FR-003
func TestBrokenFeaturesGiveExactlyTheirRule(t *testing.T) {
	for _, b := range BrokenFeatures() {
		t.Run(b.Rule, func(t *testing.T) {
			add := NewTree(append(Valid(), b.Cmd)...)
			errs := feature.CheckContract(add)
			if len(errs) != 1 {
				t.Fatalf("got %d errors %v, want 1", len(errs), errs)
			}
			msg := errs[0].Error()
			if !strings.Contains(msg, "feature "+b.Cmd.Name()+":") || !strings.Contains(msg, b.Message) {
				t.Fatalf("error %q does not name %s and %q", msg, b.Cmd.Name(), b.Message)
			}
		})
	}
}

// TestNoNonTestFileImportsFeaturetest parses the imports of every non-test Go
// file of the module and fails if one imports this package.
//
// Covers: FR-002, FR-003
func TestNoNonTestFileImportsFeaturetest(t *testing.T) {
	root := filepath.Join("..", "..")
	const target = "github.com/ro-uc-ru/aicli/internal/featuretest"
	fset := token.NewFileSet()
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() && (d.Name() == ".git" || d.Name() == "testdata") {
			return filepath.SkipDir
		}
		if d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		src, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		f, err := parser.ParseFile(fset, path, src, parser.ImportsOnly)
		if err != nil {
			return err
		}
		for _, imp := range f.Imports {
			if p, _ := strconv.Unquote(imp.Path.Value); p == target {
				t.Errorf("%s imports %s", path, target)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

// TestMultiFeatureShapeAndContract checks that the valid multi-choice feature
// passes the existing contract rules, that its usage line shows the input as
// <input>..., and that its plan has the documented entries for any values.
//
// Covers: FR-001
func TestMultiFeatureShapeAndContract(t *testing.T) {
	add := NewTree(append(Valid(), NewMulti())...)
	if errs := feature.CheckContract(add); len(errs) != 0 {
		t.Fatalf("tree with the multi feature gave errors: %v", errs)
	}
	var cmd *cobra.Command
	for _, c := range add.Commands() {
		if c.Name() == Multi {
			cmd = c
		}
	}
	if cmd == nil {
		t.Fatal("multi feature not registered")
	}
	cmd.InitDefaultHelpFlag()
	if got, want := cmd.UseLine(), "aicli add "+Multi+" <"+MultiInput+">... [flags]"; got != want {
		t.Fatalf("UseLine() = %q, want %q", got, want)
	}
	d := mustDecl(t, cmd)
	in := d.Inputs[0]
	if !in.Multi || in.ItemNoun != MultiNoun || len(in.ItemFiles) != len(MultiChoices) {
		t.Fatalf("input = %+v", in)
	}
	files, err := d.Plan([]string{"alpha", "gamma"})
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, f := range files {
		got = append(got, strconv.Itoa(int(f.Action))+" "+f.Path+" "+f.Marker)
	}
	want := []string{
		"1 shared/one.txt ", "1 shared/two.txt ",
		"0 items/alpha.txt ", "0 items/alpha.md ", "0 items/gamma.txt ", "0 items/gamma.md ",
		"2 targets/a.txt first", "2 targets/b.txt second", "2 targets/a.txt first", "2 targets/b.txt second",
	}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("entries = %q, want %q", got, want)
	}
	if steps := d.NextSteps([]string{"alpha", "gamma"}); len(steps) != 2 || steps[1] != "Set gamma in .env" {
		t.Fatalf("steps = %q", steps)
	}
}

// TestBrokenMultiFeaturesBuild checks that every rule-breaking multi-choice
// feature is built with a declaration, a distinct name and a message.
//
// Covers: FR-001
func TestBrokenMultiFeaturesBuild(t *testing.T) {
	seen := map[string]bool{}
	for _, b := range BrokenMultiFeatures() {
		if _, ok := feature.Lookup(b.Cmd); !ok || b.Message == "" || seen[b.Cmd.Name()] {
			t.Fatalf("broken feature %q is malformed", b.Rule)
		}
		seen[b.Cmd.Name()] = true
	}
	if len(seen) != 7 {
		t.Fatalf("got %d broken multi-choice features, want 7", len(seen))
	}
}

// TestBrokenMultiFeaturesGiveExactlyTheirRule checks that, next to the valid
// features (the multi-choice one included), each rule-breaking multi-choice
// feature gives exactly one error that names it and its rule, and that the
// valid multi-choice feature gives none.
//
// Covers: FR-001
func TestBrokenMultiFeaturesGiveExactlyTheirRule(t *testing.T) {
	valid := append(Valid(), NewMulti())
	if errs := feature.CheckContract(NewTree(valid...)); len(errs) != 0 {
		t.Fatalf("valid tree gave errors: %v", errs)
	}
	for _, b := range BrokenMultiFeatures() {
		t.Run(b.Rule, func(t *testing.T) {
			errs := feature.CheckContract(NewTree(append(append([]*cobra.Command(nil), valid...), b.Cmd)...))
			if len(errs) != 1 {
				t.Fatalf("got %d errors %v, want 1", len(errs), errs)
			}
			msg := errs[0].Error()
			if !strings.Contains(msg, "feature "+b.Cmd.Name()+":") || !strings.Contains(msg, b.Message) {
				t.Fatalf("error %q does not name %s and %q", msg, b.Cmd.Name(), b.Message)
			}
		})
	}
}
