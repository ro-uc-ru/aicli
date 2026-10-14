package tui

import (
	"regexp"
	"strings"
	"testing"
)

// styleAnsiRe matches ANSI SGR escape sequences.
var styleAnsiRe = regexp.MustCompile(`\x1b\[[0-9;]*m`)

// TestHomeStyles checks that unstyled output has no ESC byte, that styled
// output stripped of ANSI equals the unstyled one, and that the selected
// style differs from the normal one.
//
// Covers: FR-013
func TestHomeStyles(t *testing.T) {
	const sample = "> new  Create a project"
	plain, styled := newHomeStyles(false), newHomeStyles(true)

	if got := plain.Selected.Render(sample); strings.Contains(got, "\x1b") {
		t.Errorf("unstyled output has ESC: %q", got)
	}
	for name, pair := range map[string][2]string{
		"banner":    {plain.Banner.Render(sample), styled.Banner.Render(sample)},
		"entry":     {plain.Entry.Render(sample), styled.Entry.Render(sample)},
		"selected":  {plain.Selected.Render(sample), styled.Selected.Render(sample)},
		"footer":    {plain.Footer.Render(sample), styled.Footer.Render(sample)},
		"indicator": {plain.Indicator.Render(sample), styled.Indicator.Render(sample)},
	} {
		if !strings.Contains(pair[1], "\x1b") {
			t.Errorf("%s: styled output has no ESC", name)
		}
		if got := styleAnsiRe.ReplaceAllString(pair[1], ""); got != pair[0] || got != sample {
			t.Errorf("%s: stripped = %q, plain = %q", name, got, pair[0])
		}
	}
	if styled.Selected.Render(sample) == styled.Entry.Render(sample) {
		t.Error("selected style equals the normal style")
	}
}
