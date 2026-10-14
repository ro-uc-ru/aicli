package root

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// homeMarkers are texts that only the home screen prints: the banner's
// description line and its lettering.
var homeMarkers = []string{"Scaffolding for FastAPI backends", "██", "No commands available."}

// notHome fails the test if a stream shows any part of the home screen.
//
// In: t, the test; name, the stream's name; s, its text. Out: none.
// Side effects: none.
func notHome(t *testing.T, name, s string) {
	t.Helper()
	for _, m := range homeMarkers {
		if strings.Contains(s, m) {
			t.Errorf("%s shows the home screen (%q):\n%s", name, m, s)
		}
	}
}

// TestRootWithArgsKeepsBehaviour runs the real root with an unknown command,
// one with a close name, an unknown flag and `new --help`: each gives the
// output it gave before the root became runnable, and none shows the home
// screen.
//
// In: t, the test. Out: none. Side effects: none.
// Covers: EC-008
func TestRootWithArgsKeepsBehaviour(t *testing.T) {
	baseline, err := os.ReadFile(filepath.Join("testdata", "new-help-before.golden"))
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name    string
		args    []string
		code    int
		stdout  string
		stderr  string
		message string
	}{
		{"unknown command", []string{"unknowncmd"}, 1, "",
			"Error: unknown command \"unknowncmd\" for \"aicli\"\n", "Cobra's unknown-command error"},
		{"unknown command with suggestion", []string{"newx"}, 1, "",
			"Error: unknown command \"newx\" for \"aicli\"\n  \n  Did you mean this?\n  \tnew\n", "suggestions"},
		{"unknown flag", []string{"--bogus"}, 1, "",
			"Error: unknown flag: --bogus\n", "Cobra's flag error"},
		{"command help", []string{"new", "--help"}, 0, string(baseline), "", "the help of new"},
		{"command help short flag", []string{"new", "-h"}, 0, string(baseline), "", "the help of new"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var out, errb bytes.Buffer
			code := Execute(context.Background(), c.args, &out, &errb)
			if code != c.code || out.String() != c.stdout || errb.String() != c.stderr {
				t.Errorf("%s: code=%d stdout=%q stderr=%q\nwant code=%d stdout=%q stderr=%q",
					c.message, code, out.String(), errb.String(), c.code, c.stdout, c.stderr)
			}
			notHome(t, "stdout", out.String())
			notHome(t, "stderr", errb.String())
		})
	}
}

// TestRootHelpHasNoHome checks that the root help, with its single new Usage
// line, is the help and not the home screen.
//
// In: t, the test. Out: none. Side effects: none.
// Covers: EC-008
func TestRootHelpHasNoHome(t *testing.T) {
	var out, errb bytes.Buffer
	if code := Execute(context.Background(), []string{"--help"}, &out, &errb); code != 0 || errb.Len() != 0 {
		t.Fatalf("code=%d stderr=%q", code, errb.String())
	}
	if !strings.Contains(out.String(), "  aicli [flags]\n") {
		t.Errorf("root help lacks the `aicli [flags]` Usage line:\n%s", out.String())
	}
	notHome(t, "root help", out.String())
}
