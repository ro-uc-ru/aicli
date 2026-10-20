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
