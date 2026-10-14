package home

import (
	"bytes"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// newTree builds a test tree: new, db (grouping) with migrate and a hidden
// child, a hidden command, a deprecated command, and a command with no short.
func newTree() *cobra.Command {
	run := func(*cobra.Command, []string) {}
	root := &cobra.Command{Use: "aicli", Short: "root"}
	root.AddCommand(&cobra.Command{Use: "new", Short: "Create a project", Run: run})
	db := &cobra.Command{Use: "db", Short: "Database tools"}
	db.AddCommand(&cobra.Command{Use: "migrate", Short: "Run migrations", Run: run})
	db.AddCommand(&cobra.Command{Use: "secret", Short: "x", Hidden: true, Run: run})
	root.AddCommand(db)
	root.AddCommand(&cobra.Command{Use: "hid", Short: "hidden", Hidden: true, Run: run})
	root.AddCommand(&cobra.Command{Use: "old", Short: "old", Deprecated: "no", Run: run})
	root.AddCommand(&cobra.Command{Use: "bare", Run: run})
	return root
}

// paths returns the entry paths of root in order.
func paths(root *cobra.Command) []string {
	var p []string
	for _, e := range Entries(root) {
		p = append(p, e.Path)
	}
	return p
}

// TestEntriesOrderAndExclusions checks pre-order nesting, grouping commands
// and every exclusion.
//
// Covers: FR-003, FR-004
func TestEntriesOrderAndExclusions(t *testing.T) {
	root := newTree()
	run := func(*cobra.Command, []string) {}
	root.AddCommand(&cobra.Command{Use: "help", Short: "h", Run: run})
	comp := &cobra.Command{Use: "completion", Short: "c", Run: run}
	comp.AddCommand(&cobra.Command{Use: "bash", Short: "b", Run: run})
	root.AddCommand(comp)
	root.AddCommand(&cobra.Command{Use: "__complete", Short: "c", Run: run})

	got := strings.Join(paths(root), "|")
	want := "bare|db|db migrate|new"
	if got != want {
		t.Errorf("paths = %q, want %q", got, want)
	}
}

// TestEntriesEmptyAndNoShort checks an empty tree and a missing description.
//
// Covers: EC-001, EC-005
func TestEntriesEmptyAndNoShort(t *testing.T) {
	if got := Entries(&cobra.Command{Use: "aicli"}); len(got) != 0 {
		t.Errorf("empty tree gave %d entries", len(got))
	}
	for _, e := range Entries(newTree()) {
		if e.Path == "bare" && e.Short != "" {
			t.Errorf("bare Short = %q, want empty", e.Short)
		}
	}
}

// TestHelpMatchesCobra checks the help text equals `<path> --help` run
// through Cobra on the same tree.
//
// Covers: FR-008
func TestHelpMatchesCobra(t *testing.T) {
	for _, path := range []string{"new", "db migrate", "db"} {
		root := newTree()
		var help func() (string, error)
		for _, e := range Entries(root) {
			if e.Path == path {
				help = e.Help
			}
		}
		got, err := help()
		if err != nil {
			t.Fatalf("%s: %v", path, err)
		}

		ref := newTree()
		var buf bytes.Buffer
		ref.SetOut(&buf)
		ref.SetErr(&buf)
		ref.SetArgs(append(strings.Fields(path), "--help"))
		if err := ref.Execute(); err != nil {
			t.Fatal(err)
		}
		if got != buf.String() {
			t.Errorf("%s: help differs\n got: %q\nwant: %q", path, got, buf.String())
		}
	}
}

// TestHelpBrokenTemplate checks the FR-016 error and that the writers are
// restored after the failure.
//
// Covers: FR-016
func TestHelpBrokenTemplate(t *testing.T) {
	root := newTree()
	for _, c := range root.Commands() {
		if c.Name() == "new" {
			c.SetHelpTemplate("{{ .Missing }}")
		}
	}
	var help func() (string, error)
	for _, e := range Entries(root) {
		if e.Path == "new" {
			help = e.Help
		}
	}
	_, err := help()
	if err == nil || !strings.HasPrefix(err.Error(), "cannot show help for new: ") {
		t.Fatalf("err = %v, want cannot show help for new: ...", err)
	}
	if strings.Contains(err.Error(), "\n") {
		t.Errorf("error has more than one line: %q", err)
	}
	for _, c := range root.Commands() {
		if c.Name() == "new" {
			if c.OutOrStdout() == nil || c.ErrOrStderr() == nil {
				t.Fatal("writers unusable")
			}
			var b bytes.Buffer
			c.SetOut(nil)
			if c.OutOrStdout() == &b {
				t.Error("writer not restored")
			}
		}
	}
}
