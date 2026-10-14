package tui

import (
	"strings"

	"github.com/charmbracelet/x/ansi"
)

// truncate cuts s to at most width cells, by display width, never wrapping.
//
// In: s, the text, without newlines; width, the maximum width in cells.
// Out: s unchanged if it fits; otherwise s cut so that, with a trailing `…`,
// it is at most width cells wide (a wide character that would straddle the
// cut is dropped). An empty string if width < 1.
// Side effects: none.
func truncate(s string, width int) string {
	if width < 1 {
		return ""
	}
	return ansi.Truncate(s, width, "…")
}

// padRight pads s with spaces on the right up to width cells.
//
// In: s, the text; width, the target width in cells, by display width.
// Out: s followed by the spaces needed to reach width; s unchanged if it is
// already at least that wide.
// Side effects: none.
func padRight(s string, width int) string {
	return s + strings.Repeat(" ", max(width-ansi.StringWidth(s), 0))
}

// bannerFor builds the banner text lines for a terminal width
// (EC-002).
//
// In: width, the terminal width in cells.
// Out: the six lettering lines and the two description lines when width is
// at least the lettering's width; otherwise the single line `AICLI` and the
// two description lines. In both forms every line is truncated with `…` to
// width, and none is wrapped. No blank separator line is included.
// Side effects: none.
func bannerFor(width int) []string {
	var lines []string
	if width >= bannerWidth() {
		lines = append(lines, bannerLines...)
	} else {
		lines = append(lines, bannerNarrow)
	}
	lines = append(lines, bannerDescription...)
	for i, l := range lines {
		lines[i] = truncate(l, width)
	}
	return lines
}
