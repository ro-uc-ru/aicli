package llmprovider_test

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/ro-uc-ru/aicli/internal/gen"
	"github.com/ro-uc-ru/aicli/internal/llmprovider"
)

// describe renders a plan entry as "<action> <path> [marker] [line]".
//
// In: e, an entry. Out: the text. No side effects.
func describe(e gen.FeatureFile) string {
	s := fmt.Sprintf("%d %s %s", e.Action, e.Path, e.Marker)
	if d, ok := e.Data.(llmprovider.Data); ok {
		s += " " + d.SDK + d.Line
	}
	return s
}

// plan returns the described entries for values.
//
// In: t, the test; values, the providers. Out: one text per entry.
// No side effects.
func plan(t *testing.T, values ...string) []string {
	t.Helper()
	entries, err := llmprovider.Plan(values)
	if err != nil {
		t.Fatal(err)
	}
	out := make([]string, len(entries))
	for i, e := range entries {
		out[i] = describe(e)
	}
	return out
}

// TestDeclaration checks the name, short text and the multi-choice input.
//
// Covers: FR-012
func TestDeclaration(t *testing.T) {
	d := llmprovider.Declaration()
	if d.Name != "llm-provider" || d.Short != "Add one or more LLM providers." {
		t.Fatalf("name/short: %q %q", d.Name, d.Short)
	}
	if len(d.Inputs) != 1 {
		t.Fatalf("inputs: %d", len(d.Inputs))
	}
	in := d.Inputs[0]
	if in.Name != "provider" || in.Description != "LLM providers to add" || !in.Multi || in.ItemNoun != "providers" {
		t.Fatalf("input: %+v", in)
	}
	if want := []string{"openai", "claude", "gemini", "ollama"}; !reflect.DeepEqual(in.Choices, want) {
		t.Fatalf("choices: %v", in.Choices)
	}
	for _, c := range in.Choices {
		if in.ItemFiles[c] != "app/ai/llm/"+c+".py" {
			t.Fatalf("item file of %s: %q", c, in.ItemFiles[c])
		}
	}
}

// TestPlanOrderOne checks the FR-013 order with one provider.
//
// Covers: FR-013
func TestPlanOrderOne(t *testing.T) {
	got := plan(t, "claude")
	want := []string{
		"1 app/ai/llm/__init__.py ", "1 app/ai/llm/base.py ", "1 app/services/llm.py ",
		"1 app/schemas/llm.py ", "1 app/api/routers/llm.py ", "1 tests/test_llm_router.py ",
		"0 app/ai/llm/claude.py  ", "0 tests/test_llm_claude.py  ",
		"2 app/api/router.py router-imports", "2 app/api/router.py routers",
		"2 pyproject.toml dependencies anthropic",
		"2 .env.example env ANTHROPIC_API_KEY=", "2 .env.example env ANTHROPIC_MODEL=",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got\n%q\nwant\n%q", got, want)
	}
}

// TestPlanOrderAll checks the order with all four providers, given in any
// order, and that the openai SDK appears once.
//
// Covers: FR-013, EC-008
func TestPlanOrderAll(t *testing.T) {
	got := plan(t, "ollama", "gemini", "claude", "openai")
	want := []string{
		"1 app/ai/llm/__init__.py ", "1 app/ai/llm/base.py ", "1 app/services/llm.py ",
		"1 app/schemas/llm.py ", "1 app/api/routers/llm.py ", "1 tests/test_llm_router.py ",
		"0 app/ai/llm/openai.py  ", "0 tests/test_llm_openai.py  ",
		"0 app/ai/llm/claude.py  ", "0 tests/test_llm_claude.py  ",
		"0 app/ai/llm/gemini.py  ", "0 tests/test_llm_gemini.py  ",
		"0 app/ai/llm/ollama.py  ", "0 tests/test_llm_ollama.py  ",
		"2 app/api/router.py router-imports", "2 app/api/router.py routers",
		"2 pyproject.toml dependencies openai", "2 pyproject.toml dependencies anthropic",
		"2 pyproject.toml dependencies google-genai",
		"2 .env.example env OPENAI_API_KEY=", "2 .env.example env OPENAI_MODEL=",
		"2 .env.example env ANTHROPIC_API_KEY=", "2 .env.example env ANTHROPIC_MODEL=",
		"2 .env.example env GEMINI_API_KEY=", "2 .env.example env GEMINI_MODEL=",
		"2 .env.example env OLLAMA_MODEL=",
		"2 .env.example env # OLLAMA_BASE_URL=http://localhost:11434/v1",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got\n%q\nwant\n%q", got, want)
	}
}

// TestPlanSDKOnce checks that openai and ollama share one dependency entry.
//
// Covers: FR-013, EC-008
func TestPlanSDKOnce(t *testing.T) {
	n := 0
	for _, s := range plan(t, "openai", "ollama") {
		if s == "2 pyproject.toml dependencies openai" {
			n++
		}
	}
	if n != 1 {
		t.Fatalf("openai SDK entries: %d", n)
	}
}

// TestNextSteps checks the next steps of one provider and of openai ollama.
//
// Covers: FR-014
func TestNextSteps(t *testing.T) {
	got := llmprovider.NextSteps([]string{"ollama", "openai"})
	want := []string{
		"Run uv sync to install the provider SDKs",
		"Set OPENAI_API_KEY, OPENAI_MODEL in .env",
		"Set OLLAMA_MODEL in .env",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %q", got)
	}
}
