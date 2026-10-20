package root

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/ro-uc-ru/aicli/internal/featuretest"
)

// TestStaticHomeListsAddFeatures runs the home screen with the featuretest
// tree registered under `add` and checks that `add <feature>` is listed for
// listed features only: hidden and deprecated features are absent, and a
// feature registered only in tests appears with no change to the feature's
// code.
//
// In: t, the test. Out: none. Side effects: none.
// Covers: FR-004, EC-002
func TestStaticHomeListsAddFeatures(t *testing.T) {
	getwd := func() (string, error) { return t.TempDir(), nil }
	cmd := newRoot(".", getwd)
	addCmd, _, err := cmd.Find([]string{"add"})
	if err != nil || addCmd == cmd {
		t.Fatalf("add command not found: %v", err)
	}
	featuretest.Register(addCmd, featuretest.Valid()...)
	var out, errb bytes.Buffer
	if code := run(context.Background(), cmd, []string{}, &out, &errb); code != 0 || errb.Len() != 0 {
		t.Fatalf("code=%d stderr=%q, want 0 and empty", code, errb.String())
	}
	got := out.String()
	for _, want := range []string{"add  ", "add " + featuretest.NewMain().Name(), "add " + featuretest.NewNoInputs().Name()} {
		if !strings.Contains(got, want) {
			t.Errorf("home does not list %q:\n%s", want, got)
		}
	}
	for _, hidden := range []string{featuretest.NewHidden().Name(), featuretest.NewDeprecated().Name()} {
		if strings.Contains(got, hidden) {
			t.Errorf("home lists unlisted feature %q:\n%s", hidden, got)
		}
	}
}
