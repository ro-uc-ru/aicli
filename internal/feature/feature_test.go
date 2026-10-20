package feature

import (
	"testing"

	"github.com/spf13/cobra"
)

// TestLookup checks that Lookup finds the declaration of a command built by
// NewCommand and finds none for a plain Cobra command or nil.
//
// Covers: FR-003
func TestLookup(t *testing.T) {
	cmd := NewCommand(Declaration{Name: "feature-1", Short: "s", Inputs: []Input{{Name: "input-1"}}})
	d, ok := Lookup(cmd)
	if !ok || d.Name != "feature-1" || d.Short != "s" || len(d.Inputs) != 1 {
		t.Fatalf("Lookup(built) = %+v, %v", d, ok)
	}
	if d, ok := Lookup(&cobra.Command{Use: "plain"}); ok || d != nil {
		t.Fatalf("Lookup(plain) = %+v, %v; want none", d, ok)
	}
	if _, ok := Lookup(nil); ok {
		t.Fatal("Lookup(nil) found a declaration")
	}
}

// TestNewCommandUseLine checks the usage line under a test add command and
// that any number of arguments is accepted.
//
// Covers: FR-003, FR-006
func TestNewCommandUseLine(t *testing.T) {
	add := &cobra.Command{Use: "add"}
	root := &cobra.Command{Use: "aicli"}
	cmd := NewCommand(Declaration{Name: "feature-1", Inputs: []Input{{Name: "input-1"}, {Name: "input-2"}}})
	add.AddCommand(cmd)
	root.AddCommand(add)
	cmd.InitDefaultHelpFlag()
	if got, want := cmd.UseLine(), "aicli add feature-1 <input-1> <input-2> [flags]"; got != want {
		t.Fatalf("UseLine() = %q, want %q", got, want)
	}
	if err := cmd.Args(cmd, []string{"a", "b", "c", "d"}); err != nil {
		t.Fatalf("Args rejected extra arguments: %v", err)
	}
	if err := cmd.Args(cmd, nil); err != nil {
		t.Fatalf("Args rejected no arguments: %v", err)
	}
}
