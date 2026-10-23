package add

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"

	"github.com/ro-uc-ru/aicli/internal/feature"
	"github.com/ro-uc-ru/aicli/internal/featuretest"
	"github.com/ro-uc-ru/aicli/internal/llmprovider"
	"github.com/ro-uc-ru/aicli/internal/project"
	"github.com/ro-uc-ru/aicli/internal/scaffold"
)

// llmProject scaffolds a project and returns its root and an `add` command
// that holds the real llm-provider feature, with the root in its context.
//
// In: t, the test. Out: the add command and the project root.
// Side effects: writes under a temp dir.
func llmProject(t *testing.T) (*cobra.Command, string) {
	t.Helper()
	base := t.TempDir()
	if err := scaffold.Run(context.Background(), base, "demo"); err != nil {
		t.Fatal(err)
	}
	root := filepath.Join(base, "demo")
	add := featuretest.NewTree(feature.NewCommand(llmprovider.Declaration()))
	add.SetContext(project.WithRoot(context.Background(), root))
	return add, root
}

// llmStartDirect runs the real `aicli add llm-provider` command with no
// arguments on a pipe, as in an interactive terminal.
//
// In: t, the test; add, the add command. Out: the session. Side effects:
// starts a goroutine running the command; stubs isInteractive until the test
// ends.
func llmStartDirect(t *testing.T, add *cobra.Command) *flow {
	t.Helper()
	old := isInteractive
	isInteractive = func(io.Reader, io.Writer, func(string) string) bool { return true }
	t.Cleanup(func() { isInteractive = old })
	Prepare(add.Parent())
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = r.Close(); _ = w.Close() })
	f := &flow{t: t, in: w, out: &syncOut{}, done: make(chan error, 1)}
	root := add.Parent()
	root.SetIn(r)
	root.SetOut(f.out)
	root.SetErr(io.Discard)
	root.SetArgs([]string{"add", "llm-provider"})
	go func() { f.done <- root.ExecuteContext(add.Context()) }()
	f.wait("> provider")
	return f
}

// TestLLMFlowEscCreatesNothing checks that Esc in the direct form restores the
// terminal, prints nothing and creates nothing.
//
// Covers: FR-005, EC-016
func TestLLMFlowEscCreatesNothing(t *testing.T) {
	add, root := llmProject(t)
	before := entries(t, root)
	f := llmStartDirect(t, add)
	f.send("\x1b")
	if err := f.finish(); err != nil {
		t.Fatal(err)
	}
	if got := f.after(); got != "" {
		t.Errorf("something is left on the normal screen: %q", got)
	}
	if got := entries(t, root); len(got) != len(before) {
		t.Errorf("files changed: %v -> %v", before, got)
	}
}

// TestLLMFlowAdd checks that Add creates the files and prints the report once
// the terminal is restored.
//
// Covers: FR-011, FR-014
func TestLLMFlowAdd(t *testing.T) {
	add, root := llmProject(t)
	f := llmStartDirect(t, add)
	f.send(" ")
	f.send("\r")
	f.wait("Will create:")
	f.send("y")
	if err := f.finish(); err != nil {
		t.Fatal(err)
	}
	want := "create  app/ai/llm/__init__.py\ncreate  app/ai/llm/base.py\ncreate  app/services/llm.py\n" +
		"create  app/schemas/llm.py\ncreate  app/api/routers/llm.py\ncreate  tests/test_llm_router.py\n" +
		"create  app/ai/llm/openai.py\ncreate  tests/test_llm_openai.py\n" +
		"update  app/api/router.py\nupdate  pyproject.toml\nupdate  .env.example\n\n" +
		"Added llm-provider\n\nNext steps:\n  Run uv sync to install the provider SDKs\n  Set OPENAI_API_KEY, OPENAI_MODEL in .env\n"
	if got := f.after(); got != want {
		t.Errorf("after the terminal was restored:\n%q\nwant\n%q", got, want)
	}
	if _, err := os.Stat(filepath.Join(root, "app", "ai", "llm", "openai.py")); err != nil {
		t.Error(err)
	}
}
