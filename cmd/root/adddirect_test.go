package root

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/ro-uc-ru/aicli/internal/featuretest"
	"github.com/ro-uc-ru/aicli/internal/tui"
)

// ansiSeq matches the ANSI escape sequences that styling adds.
var ansiSeq = regexp.MustCompile("\x1b\\[[0-9;]*m")

// TestAddDirectCreatesFiles runs `aicli add <feature>` with valid values in a
// project root: the files are created with the rendered content, stdout is
// the report of FR-025 (golden file) and stderr is empty. Surrounding
// whitespace of the arguments is trimmed.
//
// In: t, the test. Out: none. Side effects: writes in a temp directory.
// Covers: FR-006, FR-025
func TestAddDirectCreatesFiles(t *testing.T) {
	dir := projectDir(t)
	code, out, errb := runAdd(t, dir, "add", featuretest.Main, " one ", " name ")
	if code != 0 || errb != "" {
		t.Fatalf("code=%d stderr=%q, want 0 and empty", code, errb)
	}
	compareGolden(t, "add-direct-ok.golden", []byte(out))
	for path, want := range map[string]string{
		"features/name.txt": "a one name\n",
		"features/name.md":  "b name\n",
	} {
		got, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(path)))
		if err != nil || string(got) != want {
			t.Errorf("%s = %q, %v; want %q", path, got, err, want)
		}
	}
}

// TestAddDirectErrors runs the failing direct calls in a project root and
// compares stderr with golden files. Each fails with exit code 1, empty
// stdout and no file created; its styled form, stripped of ANSI sequences and
// of the `✗ ` prefix, equals the plain one.
//
// In: t, the test; the golden files are testdata/add-err-<name>.golden.
// Out: none. Side effects: writes golden files with -update.
// Covers: FR-006, FR-007, FR-023, FR-026, EC-009, EC-013
func TestAddDirectErrors(t *testing.T) {
	m := featuretest.Main
	cases := []struct {
		name string
		pre  []string // a call that must succeed first
		args []string
	}{
		{"too-few", nil, []string{"add", m, "one"}},
		{"too-few-none", nil, []string{"add", m}},
		{"too-many", nil, []string{"add", m, "one", "x", "y"}},
		{"empty", nil, []string{"add", m, "one", "   "}},
		{"control", nil, []string{"add", m, "one", "a\tb"}},
		{"choice", nil, []string{"add", m, "ONE", "x"}},
		{"validator", nil, []string{"add", m, "one", featuretest.BadText}},
		{"unknown-suggest", nil, []string{"add", "feature-2"}},
		{"unknown-plain", nil, []string{"add", "zzz"}},
		{"exists", []string{"add", m, "one", "x"}, []string{"add", m, "two", "x"}},
		{"plan-fails", nil, []string{"add", m, featuretest.FailChoice, "x"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			dir := projectDir(t)
			if c.pre != nil {
				if code, _, errb := runAdd(t, dir, c.pre...); code != 0 {
					t.Fatalf("setup failed: %s", errb)
				}
			}
			before := snapshot(t, dir)
			code, out, errb := runAdd(t, dir, c.args...)
			if code != 1 || out != "" {
				t.Fatalf("code=%d stdout=%q, want 1 and empty", code, out)
			}
			if !strings.HasPrefix(errb, "Error: ") || ansiSeq.MatchString(errb) {
				t.Errorf("stderr is not plain: %q", errb)
			}
			compareGolden(t, "add-err-"+c.name+".golden", []byte(errb))
			if c.pre == nil && len(snapshot(t, dir)) != len(before) {
				t.Error("files were created")
			}

			msg := strings.TrimSuffix(strings.TrimPrefix(errb, "Error: "), "\n")
			msg = strings.ReplaceAll(msg, "\n  ", "\n")
			styled := tui.RenderError(msg, true)
			if !ansiSeq.MatchString(styled) {
				t.Fatalf("styled error has no ANSI sequence: %q", styled)
			}
			stripped := strings.Replace(ansiSeq.ReplaceAllString(styled, ""), "✗ ", "", 1)
			if stripped != errb {
				t.Errorf("styled stripped = %q, want %q", stripped, errb)
			}
		})
	}
}

// TestAddHiddenAndDeprecatedRunDirectly runs a hidden and a deprecated
// feature directly: each creates its file and prints only the report of
// FR-025, with no deprecation notice and nothing on stderr.
//
// In: t, the test. Out: none. Side effects: writes in temp directories.
// Covers: EC-002, FR-025
func TestAddHiddenAndDeprecatedRunDirectly(t *testing.T) {
	for _, name := range []string{featuretest.Hidden, featuretest.Deprecated} {
		t.Run(name, func(t *testing.T) {
			dir := projectDir(t)
			code, out, errb := runAdd(t, dir, "add", name, "value")
			want := "create  features/one.txt\n\nAdded " + name + "\n"
			if code != 0 || errb != "" || out != want {
				t.Errorf("code=%d stdout=%q stderr=%q, want 0, %q and empty", code, out, errb, want)
			}
			if _, err := os.Stat(filepath.Join(dir, "features", "one.txt")); err != nil {
				t.Error(err)
			}
		})
	}
}
