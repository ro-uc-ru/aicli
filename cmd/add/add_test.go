package addcmd

import (
	"bytes"
	"context"
	"testing"

	"github.com/spf13/cobra"

	"github.com/ro-uc-ru/aicli/internal/project"
)

// TestAddNoFeatures checks that `add` with no feature prints the EC-003 text.
//
// Covers: FR-001, FR-002
func TestAddNoFeatures(t *testing.T) {
	root := &cobra.Command{Use: "aicli"}
	root.AddCommand(NewCommand())
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetArgs([]string{"add"})
	if err := root.ExecuteContext(project.WithRoot(context.Background(), t.TempDir())); err != nil {
		t.Fatal(err)
	}
	want := "aicli › add\nAdd an AI/LLM feature to this project.\n\nNo features available.\n"
	if out.String() != want {
		t.Fatalf("output = %q, want %q", out.String(), want)
	}
}
