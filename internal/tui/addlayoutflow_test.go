package tui_test

import (
	"fmt"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"

	"github.com/ro-uc-ru/aicli/internal/tui"
)

// manyPaths returns a feature with one text input whose plan lists n paths.
//
// In: n, the number of paths. Out: the feature. Side effects: none.
func manyPaths(n int) tui.Feature {
	f := confirmFeature(nil, nil, false)
	f.Plan = func([]string) ([]string, error) {
		paths := make([]string, n)
		for i := range paths {
			paths[i] = fmt.Sprintf("features/file-%02d.txt", i+1)
		}
		return paths, nil
	}
	plan := f.Plan
	f.Check = func(v []string) (tui.Preview, error) {
		paths, err := plan(v)
		return tui.Preview{Create: paths}, err
	}
	return f
}

// sizedConfirm opens the form of f at the given size, types "val", shows the
// confirmation, sends msgs and quits with Ctrl+C.
//
// In: t, the test; f, the feature; w and h, the size; msgs. Out: the last
// screen. Side effects: as driveAdd.
func sizedConfirm(t *testing.T, f tui.Feature, w, h int, msgs ...tea.Msg) string {
	t.Helper()
	all := append(append(toConfirm(), msgs...), key("ctrl+c"))
	v, _ := driveAddWith(t, []tui.Feature{f}, w, h, all...)
	return v
}

// maxWidth returns the widest line of a screen, in cells.
//
// In: v, the screen. Out: the width. Side effects: none.
func maxWidth(v string) int {
	w := 0
	for _, l := range strings.Split(v, "\n") {
		w = max(w, ansi.StringWidth(l))
	}
	return w
}

// repeat returns n copies of the key.
//
// In: k, the key name; n, the count. Out: the messages. Side effects: none.
func repeat(k string, n int) []tea.Msg {
	out := make([]tea.Msg, n)
	for i := range out {
		out[i] = key(k)
	}
	return out
}

// TestAddLayoutFileListScrolls checks, on a short terminal, that the file
// list scrolls, heading included, with `↓`/`↑` and `j`/`k`, with the
// indicator lines, the `↑/↓ scroll` hint, the buttons always visible and the
// screen never taller than the terminal.
//
// Covers: FR-011, EC-010
func TestAddLayoutFileListScrolls(t *testing.T) {
	const h = 14
	f := manyPaths(30)
	top := sizedConfirm(t, f, termW, h)
	lines := strings.Split(top, "\n")
	if len(lines) > h {
		t.Fatalf("the screen is %d lines, taller than %d:\n%s", len(lines), h, top)
	}
	for _, w := range []string{"aicli › add › feature-1", "Will create:", "  features/file-01.txt", "↓ more", "↑/↓ scroll", "[ Cancel ]"} {
		if !strings.Contains(top, w) {
			t.Errorf("missing %q:\n%s", w, top)
		}
	}
	if strings.Contains(top, "↑ more") || strings.Contains(top, "file-30") {
		t.Errorf("the top of the list needs only ↓ more:\n%s", top)
	}
	for _, k := range []string{"down", "j"} {
		v := sizedConfirm(t, f, termW, h, key(k))
		if !strings.Contains(v, "↑ more") || strings.Contains(v, "Will create:") {
			t.Errorf("%s must scroll the heading and the paths together:\n%s", k, v)
		}
	}
	bottom := sizedConfirm(t, f, termW, h, repeat("down", 60)...)
	if !strings.Contains(bottom, "file-30.txt") || !strings.Contains(bottom, "↑ more") || strings.Contains(bottom, "↓ more") {
		t.Errorf("the end of the list needs only ↑ more and the last path:\n%s", bottom)
	}
	if len(strings.Split(bottom, "\n")) > h || !strings.Contains(bottom, "[ Cancel ]") {
		t.Errorf("the buttons must stay and the screen fit:\n%s", bottom)
	}
	back := sizedConfirm(t, f, termW, h, append(repeat("down", 60), repeat("k", 60)...)...)
	if back != top {
		t.Errorf("scrolling back must restore the top:\n%s", back)
	}
	if up := sizedConfirm(t, f, termW, h, key("up"), key("k")); up != top {
		t.Errorf("scrolling up at the top must do nothing:\n%s", up)
	}
	tall := sizedConfirm(t, f, termW, 60)
	if strings.Contains(tall, "more") || strings.Contains(tall, "scroll") || !strings.Contains(tall, "file-30.txt") {
		t.Errorf("a tall terminal shows every path and no hint:\n%s", tall)
	}
}

// TestAddLayoutChoiceScrolls checks that a focused choice field taller than
// the space left scrolls inside the field, keeping the selected choice
// visible, with the screen never taller than the terminal.
//
// Covers: EC-010
func TestAddLayoutChoiceScrolls(t *testing.T) {
	choices := make([]string, 12)
	for i := range choices {
		choices[i] = fmt.Sprintf("choice-%02d", i+1)
	}
	f := tui.Feature{Name: "feature-1", Short: "Description of feature 1", Listed: true,
		Inputs: []tui.FeatureInput{{Name: "input-1", Description: "Description of input 1", Choices: choices}},
		Plan:   func([]string) ([]string, error) { return []string{"a.txt"}, nil }}
	const h = 9
	view := func(msgs ...tea.Msg) string {
		v, _ := driveAddWith(t, []tui.Feature{f}, termW, h, append(append([]tea.Msg{key("enter")}, msgs...), key("ctrl+c"))...)
		return v
	}
	selected := func(v string) string {
		for _, l := range strings.Split(v, "\n") {
			if strings.Contains(l, "(•)") {
				return strings.TrimSpace(strings.Replace(l, "(•)", "", 1))
			}
		}
		return ""
	}
	for n := 0; n < len(choices); n++ {
		v := view(repeat("down", n)...)
		if want := choices[n]; selected(v) != want {
			t.Fatalf("after %d downs the selected choice %q must be visible, got %q:\n%s", n, want, selected(v), v)
		}
		if got := len(strings.Split(v, "\n")); got > h {
			t.Fatalf("after %d downs the form is %d lines, taller than %d:\n%s", n, got, h, v)
		}
		if !strings.Contains(v, "> input-1") {
			t.Fatalf("the focused title must stay visible:\n%s", v)
		}
	}
	if v := view(repeat("down", 11)...); strings.Contains(v, "choice-01") {
		t.Errorf("the first choices must have scrolled out:\n%s", v)
	}
	if v := view(append(repeat("down", 11), repeat("up", 11)...)...); selected(v) != "choice-01" {
		t.Errorf("scrolling back must reach the first choice:\n%s", v)
	}
}

// TestAddLayoutCommandScrolls checks that a wrapped equivalent command taller
// than the terminal scrolls with `↑`/`↓` and `k`/`j` above the buttons,
// starting at its first line, with the file list hidden and no part of the
// command ever cut.
//
// Covers: EC-010
func TestAddLayoutCommandScrolls(t *testing.T) {
	const w, h = 20, 5
	f := confirmFeature(nil, nil, false)
	words := make([]string, 8)
	for i := range words {
		words[i] = fmt.Sprintf("word%d-%s", i+1, strings.Repeat("x", 6))
	}
	f.Command = func([]string) string { return "aicli add feature-1 -- " + strings.Join(words, " ") }
	top := sizedConfirm(t, f, w, h)
	lines := strings.Split(top, "\n")
	if len(lines) > h {
		t.Fatalf("the screen is %d lines, taller than %d:\n%s", len(lines), h, top)
	}
	if lines[0] != "aicli add feature-1" {
		t.Errorf("the command must start at its first line:\n%s", top)
	}
	if strings.Contains(top, "Will create") || !strings.Contains(top, "[ Cancel ]") || strings.Contains(top, "…") {
		t.Errorf("the list is hidden, the buttons stay and the command is not truncated:\n%s", top)
	}
	var seen strings.Builder
	for n := 0; n <= 12; n++ {
		v := sizedConfirm(t, f, w, h, repeat("j", n)...)
		if got := len(strings.Split(v, "\n")); got > h {
			t.Fatalf("after %d scrolls the screen is %d lines:\n%s", n, got, v)
		}
		if !strings.Contains(v, "[ Cancel ]") || maxWidth(v) > w || strings.Contains(v, "…") {
			t.Fatalf("after %d scrolls the buttons must stay and nothing be cut:\n%s", n, v)
		}
		seen.WriteString(v)
	}
	for _, word := range words {
		if !strings.Contains(seen.String(), word[:6]) {
			t.Errorf("scrolling must reveal %q", word[:6])
		}
	}
	if up := sizedConfirm(t, f, w, h, key("k"), key("up")); up != top {
		t.Errorf("scrolling up at the top must do nothing:\n%s", up)
	}
	if back := sizedConfirm(t, f, w, h, append(repeat("down", 12), repeat("up", 12)...)...); back != top {
		t.Errorf("scrolling back must restore the top:\n%s", back)
	}
}

// TestAddLayoutTruncation checks, on a narrow terminal, that the header, the
// descriptions, the choices, the field errors and the paths are truncated with
// `…` and never wrapped in the selector, the form and the confirmation.
//
// Covers: EC-010
func TestAddLayoutTruncation(t *testing.T) {
	const w = 24
	long := strings.Repeat("long", 10)
	f := tui.Feature{Name: "feature-" + long, Short: "Short " + long, Listed: true,
		Inputs: []tui.FeatureInput{
			{Name: "choice-" + long, Description: "Choice description " + long, Choices: []string{"first-" + long, "second"}},
			{Name: "text-" + long, Description: "Text description " + long, Validator: func(string) string { return "error " + long }},
		},
		Plan:    func([]string) ([]string, error) { return []string{"features/" + long + "/" + long + ".txt"}, nil },
		Command: func(v []string) string { return "aicli add " + long + " " + strings.Join(v, " ") }}
	run := func(msgs ...tea.Msg) string {
		v, _ := driveAddWith(t, []tui.Feature{f}, w, 20, append(msgs, key("ctrl+c"))...)
		return v
	}
	check := func(name, v string, want ...string) {
		t.Helper()
		if got := maxWidth(v); got > w {
			t.Errorf("%s: a line is %d cells wide, over %d:\n%s", name, got, w, v)
		}
		for _, s := range want {
			if !strings.Contains(v, s) {
				t.Errorf("%s: missing %q:\n%s", name, s, v)
			}
		}
	}
	sel := run()
	check("selector", sel, "…")
	form := run(key("enter"))
	check("form", form, "…", "(•) first-")
	invalid := run(key("enter"), key("tab"), typed("x"), key("tab"), key("enter"))
	check("form error", invalid, "…", "error long")
	for _, s := range []string{"feature-l…", "Short lo", "Choice des", "Text desc"} {
		if !strings.Contains(form, s) {
			t.Errorf("form: the start %q must remain:\n%s", s, form)
		}
	}
	confirm := run(key("enter"), key("tab"), typed("x"), key("enter"))
	if strings.Contains(confirm, "Will create:") {
		check("confirmation", confirm, "…", "  features/long")
	} else {
		check("confirmation", confirm, "…")
	}
}
