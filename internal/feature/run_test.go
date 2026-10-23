package feature

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/spf13/cobra"

	"github.com/ro-uc-ru/aicli/internal/gen"
	"github.com/ro-uc-ru/aicli/internal/project"
)

// runFixture builds a test feature that writes one file named after its
// input, and a context carrying a temp project root.
func runFixture(t *testing.T, plan func([]string) ([]gen.FeatureFile, error)) (*cobra.Command, context.Context, string) {
	t.Helper()
	root := t.TempDir()
	if plan == nil {
		tpl := fstest.MapFS{"t.tmpl": {Data: []byte("hello {{.Name}}\n")}}
		plan = func(v []string) ([]gen.FeatureFile, error) {
			return []gen.FeatureFile{{Path: "out/" + v[0] + ".txt", Templates: tpl, Template: "t.tmpl", Data: map[string]string{"Name": v[0]}}}, nil
		}
	}
	cmd := NewCommand(Declaration{
		Name: "feature-1",
		Inputs: []Input{{Name: "input-1", Validator: func(v string) string {
			if v == "bad" {
				return "input-1 is bad"
			}
			return ""
		}}},
		Plan:      plan,
		NextSteps: func([]string) []string { return []string{"do this"} },
	})
	return cmd, project.WithRoot(context.Background(), root), root
}

// TestRunCreatesFilesAndReports checks that valid values create the files and
// print the plain report in one write.
//
// Covers: FR-006, FR-019, FR-025
func TestRunCreatesFilesAndReports(t *testing.T) {
	cmd, ctx, root := runFixture(t, nil)
	var out bytes.Buffer
	if err := Run(ctx, cmd, []string{" abc "}, &out, func(string) string { return "" }); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(root, "out", "abc.txt"))
	if err != nil || string(b) != "hello abc\n" {
		t.Fatalf("file = %q, %v", b, err)
	}
	want := "create  out/abc.txt\n\nAdded feature-1\n\nNext steps:\n  do this\n"
	if out.String() != want {
		t.Fatalf("report = %q, want %q", out.String(), want)
	}
}

// TestRunFailuresCreateNothing checks that invalid values, a failing plan and
// a missing root create nothing and write nothing to out.
//
// Covers: FR-006, FR-019, FR-025
func TestRunFailuresCreateNothing(t *testing.T) {
	failing := func([]string) ([]gen.FeatureFile, error) { return nil, errors.New("unsupported combination") }
	tests := []struct {
		name string
		plan func([]string) ([]gen.FeatureFile, error)
		args []string
		want string
	}{
		{"too few", nil, nil, "feature-1 requires input-1"},
		{"too many", nil, []string{"a", "b"}, "takes 1 argument(s), got 2"},
		{"empty", nil, []string{" "}, "input-1 cannot be empty"},
		{"validator", nil, []string{"bad"}, "input-1 is bad"},
		{"plan fails", failing, []string{"a"}, "unsupported combination"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cmd, ctx, root := runFixture(t, tc.plan)
			var out bytes.Buffer
			err := Run(ctx, cmd, tc.args, &out, func(string) string { return "" })
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("err = %v, want %q", err, tc.want)
			}
			if out.Len() != 0 {
				t.Fatalf("out = %q, want empty", out.String())
			}
			if entries, _ := os.ReadDir(root); len(entries) != 0 {
				t.Fatalf("root not empty: %v", entries)
			}
		})
	}
	cmd, _, _ := runFixture(t, nil)
	var out bytes.Buffer
	if err := Run(context.Background(), cmd, []string{"a"}, &out, os.Getenv); !errors.Is(err, errNoRoot) || out.Len() != 0 {
		t.Fatalf("no root: err = %v, out = %q", err, out.String())
	}
	if err := Run(ctx0(), &cobra.Command{Use: "plain"}, nil, &out, os.Getenv); !errors.Is(err, errNoDeclaration) {
		t.Fatalf("plain command: err = %v", err)
	}
}

// ctx0 returns a background context.
//
// In: none. Out: context.Background(). No side effects.
func ctx0() context.Context { return context.Background() }

// TestRunE checks that the RunE built by NewCommand runs the feature through
// Run with the context of the command and its output writer.
//
// Covers: FR-006, FR-019
func TestRunE(t *testing.T) {
	cmd, ctx, root := runFixture(t, nil)
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetContext(ctx)
	if err := cmd.RunE(cmd, []string{"x"}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "out", "x.txt")); err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(out.String(), "Added feature-1\n\nNext steps:\n  do this\n") {
		t.Fatalf("out = %q", out.String())
	}
}

// TestRunNextStepsReceiveValues checks that NextSteps is called with the
// trimmed input values and its result is printed.
//
// Covers: FR-014
func TestRunNextStepsReceiveValues(t *testing.T) {
	var got []string
	cmd, ctx, _ := runFixture(t, nil)
	d, _ := Lookup(cmd)
	d.NextSteps = func(v []string) []string { got = v; return []string{"step for " + v[0]} }
	var out bytes.Buffer
	if err := Run(ctx, cmd, []string{" ok "}, &out, func(string) string { return "" }); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if len(got) != 1 || got[0] != "ok" {
		t.Fatalf("NextSteps values = %q, want [ok]", got)
	}
	if !strings.Contains(out.String(), "step for ok") {
		t.Fatalf("output lacks next step: %q", out.String())
	}
}
