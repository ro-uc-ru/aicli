package tui_test

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/spf13/cobra"

	"github.com/ro-uc-ru/aicli/internal/home"
)

// wideTree builds a tree whose only command, `wide`, has long help lines: a
// long paragraph, a word with no spaces, an indented line and a long flag
// description.
//
// In: none. Out: the root command. Side effects: none.
func wideTree() *cobra.Command {
	root := &cobra.Command{Use: "aicli", Short: "root"}
	c := &cobra.Command{
		Use:   "wide",
		Short: "Long lines",
		Long: strings.Repeat("A long paragraph of help text that keeps going. ", 6) + "\n\n" +
			"Word: " + strings.Repeat("x", 90) + "\n" +
			"Examples:\n  aicli wide --name " + strings.Repeat("value ", 20) + "\n\nshort line",
		Run: func(*cobra.Command, []string) {},
	}
	c.Flags().String("name", "", "a flag description that is long enough to need more room than a narrow terminal has")
	root.AddCommand(c)
	return root
}

// helpArea returns the help area of a detail screen: the screen without the
// header and its blank line, the indicator lines and the footer, stripped of
// ANSI.
//
// In: v, a detail screen. Out: the help lines joined by newlines. Side
// effects: none.
func helpArea(v string) string {
	lines := strings.Split(ansi.Strip(v), "\n")
	return strings.Join(lines[3:len(lines)-2], "\n")
}

// TestFlowDetailEqualsHelp checks that the help area equals the entry's
// help when the terminal is at least as wide as its longest line, and that
// when narrower the only difference is wrapping, styled and unstyled.
//
// Covers: FR-008, EC-006
func TestFlowDetailEqualsHelp(t *testing.T) {
	entries := home.Entries(wideTree())
	if len(entries) != 1 {
		t.Fatalf("want one entry, got %d", len(entries))
	}
	want, err := entries[0].Help()
	if err != nil {
		t.Fatal(err)
	}
	want = strings.TrimRight(want, "\n")
	longest := 0
	for _, l := range strings.Split(want, "\n") {
		longest = max(longest, ansi.StringWidth(l))
	}
	if longest < 120 {
		t.Fatalf("the test help must have long lines, longest is %d", longest)
	}
	for _, styled := range []bool{false, true} {
		const h = 300
		wide := drive(t, entries, styled, longest, h, keys("enter", "q")...).View
		if got := strings.TrimRight(helpArea(wide), "\n"); got != want {
			t.Errorf("styled=%v wide: help area differs from the help:\n%s\n--- want:\n%s", styled, got, want)
		}
		const w = 50
		narrow := drive(t, entries, styled, w, h, keys("enter", "q")...).View
		area := strings.TrimRight(helpArea(narrow), "\n")
		squeeze := func(s string) string { return strings.Join(strings.Fields(s), "") }
		if squeeze(area) != squeeze(want) {
			t.Errorf("styled=%v narrow: text lost or changed by wrapping:\n%s", styled, area)
		}
		shown := map[string]bool{}
		for _, l := range strings.Split(area, "\n") {
			if ansi.StringWidth(l) > w {
				t.Errorf("styled=%v narrow: line wider than the terminal: %q", styled, l)
			}
			shown[l] = true
		}
		for _, l := range strings.Split(want, "\n") {
			if ansi.StringWidth(l) <= w && !shown[l] {
				t.Errorf("styled=%v narrow: a line that fits was changed: %q", styled, l)
			}
		}
	}
}
