package tui_test

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/exp/teatest"
	"github.com/spf13/cobra"

	addpkg "github.com/ro-uc-ru/aicli/internal/add"
	"github.com/ro-uc-ru/aicli/internal/feature"
	"github.com/ro-uc-ru/aicli/internal/featuretest"
	"github.com/ro-uc-ru/aicli/internal/gen"
	"github.com/ro-uc-ru/aicli/internal/llmprovider"
	"github.com/ro-uc-ru/aicli/internal/project"
	"github.com/ro-uc-ru/aicli/internal/scaffold"
	"github.com/ro-uc-ru/aicli/internal/tui"
)

// llmSGR matches the ANSI escape sequences that styling adds.
var llmSGR = regexp.MustCompile(`\x1b\[[0-9;]*[A-Za-z]`)

// llmFooter is the footer of the direct form (design, "Form opened directly").
const llmFooter = "↑/↓ move · space toggle · enter continue · esc quit · ctrl+c quit"

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

// llmAdd applies the llm-provider plan of the given providers to the project,
// as an earlier run would have.
//
// In: t, the test; root, the project root; providers, the choices.
// Out: none. Side effects: writes under root.
func llmAdd(t *testing.T, root string, providers ...string) {
	t.Helper()
	entries, err := llmprovider.Plan(providers)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := gen.ApplyPlan(context.Background(), gen.OSFeatureFS{}, root, "llm-provider", gen.Plan{Entries: entries}); err != nil {
		t.Fatal(err)
	}
}

// llmKey turns a key name into a message: Space is "space", the rest are
// named keys or runes.
//
// In: name. Out: the message. Side effects: none.
func llmKey(name string) tea.Msg {
	switch name {
	case "space":
		return tea.KeyMsg{Type: tea.KeySpace, Runes: []rune(" ")}
	case "enter":
		return tea.KeyMsg{Type: tea.KeyEnter}
	case "esc":
		return tea.KeyMsg{Type: tea.KeyEsc}
	case "down":
		return tea.KeyMsg{Type: tea.KeyDown}
	case "up":
		return tea.KeyMsg{Type: tea.KeyUp}
	case "ctrl+c":
		return tea.KeyMsg{Type: tea.KeyCtrlC}
	}
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(name)}
}

// llmKeys turns key names into messages.
//
// In: names. Out: the messages. Side effects: none.
func llmKeys(names ...string) []tea.Msg {
	out := make([]tea.Msg, len(names))
	for i, n := range names {
		out[i] = llmKey(n)
	}
	return out
}

// llmDrive runs the add screens on the features, starting in the direct form
// of the first one or at the selector, sends msgs, then Ctrl+C.
//
// In: t, the test; features; direct, whether to start in the form; styled; w
// and h, the size; msgs. Out: the last screen and the result.
// Side effects: runs a Bubble Tea program on a test model.
func llmDrive(t *testing.T, features []tui.Feature, direct, styled bool, w, h int, msgs ...tea.Msg) (string, tui.AddResult) {
	t.Helper()
	return llmDriveHook(t, features, direct, styled, w, h, nil, msgs...)
}

// llmDriveHook is llmDrive with a hook that runs after the model is built,
// i.e. after the form has read which items are added, and before any key.
//
// In: as llmDrive, plus hook, which may be nil. Out: as llmDrive.
// Side effects: as llmDrive, plus whatever hook does.
func llmDriveHook(t *testing.T, features []tui.Feature, direct, styled bool, w, h int, hook func(), msgs ...tea.Msg) (string, tui.AddResult) {
	t.Helper()
	var opts []tui.AddOption
	if direct {
		opts = append(opts, tui.WithDirect(0))
	}
	model := tui.NewAddModel(features, styled, w, h, opts...)
	if hook != nil {
		hook()
	}
	tm := teatest.NewTestModel(t, model, teatest.WithInitialTermSize(w, h))
	for _, m := range append(msgs, llmKey("ctrl+c")) {
		tm.Send(m)
		if k, ok := m.(tea.KeyMsg); ok && (k.Type == tea.KeyEnter || k.Type == tea.KeyEsc) {
			time.Sleep(50 * time.Millisecond) // a screen change goes through a command
		}
	}
	final := tm.FinalModel(t, teatest.WithFinalTimeout(5*time.Second)).(tui.AddModel)
	if _, err := io.ReadAll(tm.FinalOutput(t, teatest.WithFinalTimeout(5*time.Second))); err != nil {
		t.Fatal(err)
	}
	return final.View(), final.Result()
}

// llmDirect is llmDrive in the direct form, unstyled, on a 80x40 terminal.
//
// In: t, the test; add, the add command; msgs. Out: the last screen.
// Side effects: as llmDrive.
func llmDirect(t *testing.T, add *cobra.Command, msgs ...tea.Msg) string {
	t.Helper()
	v, _ := llmDrive(t, addpkg.Features(add), true, false, 80, 40, msgs...)
	return v
}

// wantAll fails the test for each text the screen lacks.
//
// In: t; v, the screen; texts. Out: none. Side effects: none.
func wantAll(t *testing.T, v string, texts ...string) {
	t.Helper()
	for _, s := range texts {
		if !strings.Contains(v, s) {
			t.Errorf("missing %q in:\n%s", s, v)
		}
	}
}

// TestLLMDirectForm checks the header, the description, the field, the
// checklist and the footer of the direct form.
//
// Covers: FR-005
func TestLLMDirectForm(t *testing.T) {
	add, _ := llmProject(t)
	v := llmDirect(t, add)
	want := []string{
		"aicli › add › llm-provider",
		"Add one or more LLM providers.",
		"",
		"> provider",
		"  LLM providers to add",
		"  > [ ] openai",
		"    [ ] claude",
		"    [ ] gemini",
		"    [ ] ollama",
		"",
		llmFooter,
	}
	lines := strings.Split(v, "\n")
	if len(lines) < len(want) {
		t.Fatalf("screen too short:\n%s", v)
	}
	for i, w := range want {
		if strings.TrimRight(lines[i], " ") != w {
			t.Errorf("line %d = %q, want %q", i, lines[i], w)
		}
	}
	if strings.Contains(v, "\x1b") || strings.Contains(v, "tab") {
		t.Errorf("unstyled single-field form has escapes or tab hints:\n%s", v)
	}
}

// TestLLMKeys checks Space, the arrow and letter keys, the cursor stopping at
// both ends, deselecting, and a click, with the `> `, `[x]` and `[ ]` markers.
//
// Covers: FR-005, FR-006, EC-015
func TestLLMKeys(t *testing.T) {
	add, _ := llmProject(t)
	v := llmDirect(t, add, llmKeys("space", "down", "down", "down", "space")...)
	wantAll(t, v, "    [x] openai", "    [ ] claude", "    [ ] gemini", "  > [x] ollama")

	v = llmDirect(t, add, llmKeys("up", "j", "j", "k", "space", "down", "down", "down", "down", "space")...)
	wantAll(t, v, "    [ ] openai", "    [x] claude", "    [ ] gemini", "  > [x] ollama")

	v = llmDirect(t, add, llmKeys("space", "space")...)
	wantAll(t, v, "  > [ ] openai")

	row := -1
	for i, l := range strings.Split(llmDirect(t, add), "\n") {
		if strings.Contains(l, "[ ] gemini") {
			row = i
		}
	}
	click := tea.MouseMsg{Action: tea.MouseActionPress, Button: tea.MouseButtonLeft, X: 8, Y: row}
	v = llmDirect(t, add, click)
	wantAll(t, v, "    [ ] openai", "  > [x] gemini")
}

// TestLLMEmptySelection checks that Enter with nothing selected keeps the form
// and shows the error under the field.
//
// Covers: FR-006
func TestLLMEmptySelection(t *testing.T) {
	add, _ := llmProject(t)
	v := llmDirect(t, add, llmKey("enter"))
	wantAll(t, v, "aicli › add › llm-provider", "  > [ ] openai", "  select at least one provider")
	if strings.Contains(v, "Will create:") {
		t.Errorf("the confirmation was shown:\n%s", v)
	}
}

// TestLLMAlreadyAdded checks that the line of an added provider is marked and
// skipped by the cursor, and that with every provider added there is no
// cursor, the message appears and `enter` leaves the footer.
//
// Covers: FR-007, EC-006
func TestLLMAlreadyAdded(t *testing.T) {
	add, root := llmProject(t)
	llmAdd(t, root, "openai")
	v := llmDirect(t, add, llmKeys("down", "up", "up", "k")...)
	wantAll(t, v, "    [✓] openai (already added)", "  > [ ] claude", "    [ ] gemini", "    [ ] ollama")

	llmAdd(t, root, "claude", "gemini", "ollama")
	v = llmDirect(t, add, llmKeys("down", "space", "enter")...)
	wantAll(t, v, "    [✓] openai (already added)", "    [✓] claude (already added)",
		"    [✓] gemini (already added)", "    [✓] ollama (already added)", "  All providers are already added.", "esc quit · ctrl+c quit")
	if strings.Contains(v, "enter") || strings.Contains(v, "> [") || strings.Contains(v, "Will create:") {
		t.Errorf("a form with nothing to add must show no cursor, no enter and no confirmation:\n%s", v)
	}
}

// TestLLMEscape checks that Esc in the direct form quits with nothing added,
// and that Esc in a form opened from the selector returns to it.
//
// Covers: FR-005, EC-016
func TestLLMEscape(t *testing.T) {
	add, _ := llmProject(t)
	_, res := llmDrive(t, addpkg.Features(add), true, false, 80, 40, llmKey("esc"))
	if res.Action != tui.ActionQuit {
		t.Errorf("Esc in the direct form: %+v", res)
	}
	v, res := llmDrive(t, addpkg.Features(add), false, false, 80, 40, llmKeys("enter")...)
	wantAll(t, v, "esc back · ctrl+c quit")
	if res.Action != tui.ActionQuit {
		t.Errorf("result: %+v", res)
	}
	v, _ = llmDrive(t, addpkg.Features(add), false, false, 80, 40, llmKeys("enter", "esc")...)
	wantAll(t, v, "aicli › add", "> llm-provider  Add one or more LLM providers.")
	if strings.Contains(v, "aicli › add › llm-provider") {
		t.Errorf("still in the form:\n%s", v)
	}
}

// TestLLMConfirmation checks the confirmation of a first run (values in
// choice order whatever the order ticked) and of a later run.
//
// Covers: FR-004, FR-011
func TestLLMConfirmation(t *testing.T) {
	add, root := llmProject(t)
	first := llmKeys("down", "down", "down", "space", "up", "up", "up", "space", "enter")
	v := llmDirect(t, add, first...)
	lines := strings.Split(v, "\n")
	want := []string{
		"aicli › add › llm-provider",
		"Add one or more LLM providers.",
		"",
		"aicli add llm-provider openai ollama",
		"",
		"Will create:",
		"  app/ai/llm/__init__.py", "  app/ai/llm/base.py", "  app/services/llm.py", "  app/schemas/llm.py",
		"  app/api/routers/llm.py", "  tests/test_llm_router.py", "  app/ai/llm/openai.py", "  tests/test_llm_openai.py",
		"  app/ai/llm/ollama.py", "  tests/test_llm_ollama.py",
		"",
		"Will update:",
		"  app/api/router.py", "  pyproject.toml", "  .env.example",
		"",
	}
	for i, w := range want {
		if i >= len(lines) || strings.TrimRight(lines[i], " ") != w {
			t.Fatalf("line %d differs, want %q; screen:\n%s", i, w, v)
		}
	}
	wantAll(t, v, "  Add    [ Cancel ]", "←/→/tab choose · enter confirm · y add · n cancel · esc back · ctrl+c quit")

	llmAdd(t, root, "openai", "ollama")
	v = llmDirect(t, add, llmKeys("space", "enter")...)
	wantAll(t, v, "aicli add llm-provider claude", "Will create:\n  app/ai/llm/claude.py\n  tests/test_llm_claude.py\n",
		"Will update:\n  pyproject.toml\n  .env.example\n")
	if strings.Contains(v, "app/api/router.py") || strings.Contains(v, "llm/base.py") {
		t.Errorf("unchanged files must not be listed:\n%s", v)
	}
}

// TestLLMConfirmationScrolls checks that on a short terminal both headings and
// their paths scroll together as one list.
//
// Covers: FR-011
func TestLLMConfirmationScrolls(t *testing.T) {
	add, _ := llmProject(t)
	const h = 14
	enter := llmKeys("space", "enter")
	top, _ := llmDrive(t, addpkg.Features(add), true, false, 80, h, enter...)
	wantAll(t, top, "Will create:", "↓ more", "↑/↓ scroll", "[ Cancel ]")
	if strings.Contains(top, "Will update:") || len(strings.Split(top, "\n")) > h {
		t.Errorf("the top of the list must not show the second heading:\n%s", top)
	}
	var scroll []tea.Msg
	for range 40 {
		scroll = append(scroll, llmKey("down"))
	}
	bottom, _ := llmDrive(t, addpkg.Features(add), true, false, 80, h, append(enter, scroll...)...)
	wantAll(t, bottom, "Will update:", "  .env.example", "↑ more", "[ Cancel ]")
	if strings.Contains(bottom, "Will create:") || strings.Contains(bottom, "↓ more") || len(strings.Split(bottom, "\n")) > h {
		t.Errorf("the end of the list:\n%s", bottom)
	}
}

// TestLLMItemFileAppearsBeforeConfirmation checks that an item file created
// after the form opened blocks the confirmation with the FR-003 error and only
// Cancel.
//
// Covers: FR-007, EC-007
func TestLLMItemFileAppearsBeforeConfirmation(t *testing.T) {
	add, root := llmProject(t)
	p := filepath.Join(root, "app", "ai", "llm", "openai.py")
	create := func() {
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte("mine"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	v, _ := llmDriveHook(t, addpkg.Features(add), true, false, 80, 40, create, llmKeys("space", "down", "space", "enter")...)
	wantAll(t, v, "aicli add llm-provider openai claude",
		"Cannot add llm-provider, these providers are already added:", "already added:\n  openai (app/ai/llm/openai.py)\n", "[ Cancel ]")
	if strings.Contains(v, "Add ") && strings.Contains(v, "  Add  ") {
		t.Errorf("the Add button must be gone:\n%s", v)
	}
}

// sig returns the ANSI sequences of the first line containing text, joined.
//
// In: t; v, a styled screen; text. Out: the signature; the test fails if no
// line contains text. Side effects: none.
func sig(t *testing.T, v, text string) string {
	t.Helper()
	for _, l := range strings.Split(v, "\n") {
		if strings.Contains(llmSGR.ReplaceAllString(l, ""), text) {
			return strings.Join(llmSGR.FindAllString(l, -1), "")
		}
	}
	t.Fatalf("no line with %q in:\n%q", text, v)
	return ""
}

// TestLLMStyled checks the colours of the design table: cursor 117 bold,
// selected 117, unselected 255, already added 245, the all-added message 245,
// the empty-selection error red, `Will update:` 245 like `Will create:`.
//
// Covers: FR-005, FR-007, FR-011, EC-006
func TestLLMStyled(t *testing.T) {
	add, root := llmProject(t)
	drive := func(msgs ...tea.Msg) string {
		v, _ := llmDrive(t, addpkg.Features(add), true, true, 80, 40, msgs...)
		return v
	}
	has := func(what, got string, codes ...string) {
		t.Helper()
		for _, c := range codes {
			if !strings.Contains(got, c) {
				t.Errorf("%s: %q lacks %q", what, got, c)
			}
		}
	}
	v := drive(llmKeys("space", "down")...)
	has("cursor line", sig(t, v, "> [ ] claude"), "1;38;5;117")
	if sig(t, v, "[x] openai") == sig(t, v, "> [ ] claude") {
		t.Error("selected and cursor lines share one style")
	}
	has("selected", sig(t, v, "[x] openai"), "38;5;117")
	has("unselected", sig(t, v, "[ ] gemini"), "38;5;255")

	has("error", sig(t, drive(llmKey("enter")), "select at least one provider"), "[31m")

	v = drive(llmKeys("space", "enter")...)
	has("Will create", sig(t, v, "Will create:"), "38;5;245")
	has("Will update", sig(t, v, "Will update:"), "38;5;245")

	llmAdd(t, root, "openai")
	has("already added", sig(t, drive(), "[✓] openai (already added)"), "38;5;245")
	llmAdd(t, root, "claude", "gemini", "ollama")
	has("all added", sig(t, drive(), "All providers are already added."), "38;5;245")

	for verb, code := range map[gen.Verb]string{gen.VerbCreate: "[32m", gen.VerbUpdate: "38;5;117", gen.VerbSkip: "38;5;245"} {
		r := tui.RenderAdded("llm-provider", []gen.ReportLine{{Verb: verb, Path: "a/b"}}, nil, true)
		has("report "+string(verb), sig(t, r, string(verb)), code)
	}
}
