package tui

import (
	"strings"

	"github.com/charmbracelet/x/ansi"
)

// bannerLines is the large `AICLI` lettering of the home screen (FR-002).
var bannerLines = []string{
	` █████╗ ██╗ ██████╗██╗     ██╗`,
	`██╔══██╗██║██╔════╝██║     ██║`,
	`███████║██║██║     ██║     ██║`,
	`██╔══██║██║██║     ██║     ██║`,
	`██║  ██║██║╚██████╗███████╗██║`,
	`╚═╝  ╚═╝╚═╝ ╚═════╝╚══════╝╚═╝`,
}

// bannerDescription is the two fixed description lines shown under the
// lettering (FR-002).
var bannerDescription = []string{
	"Scaffolding for FastAPI backends for AI/LLM apps.",
	"Convention over configuration, driven by specs.",
}

// bannerNarrow is the single-line banner used when the terminal is narrower
// than the large lettering (EC-002).
const bannerNarrow = "AICLI"

// noCommandsText is printed in place of the list when there is no entry
// (EC-012).
const noCommandsText = "No commands available."

// RenderHomeStatic builds the static home output: the plain banner, a blank
// line and one line per entry (FR-002, FR-003, FR-012).
//
// In: entries, the commands to list, in display order.
// Out: the full text, ending in a newline, with no ANSI sequences. Each
// entry line is the path padded by display width to the widest path (EC-010),
// two spaces and the short description, or the path alone when it has none
// (EC-005). With no entries the list is `No commands available.` (EC-012).
// The terminal width is ignored: nothing is truncated or wrapped.
// Side effects: none.
func RenderHomeStatic(entries []Entry) string {
	var b strings.Builder
	for _, l := range bannerLines {
		b.WriteString(l + "\n")
	}
	for _, l := range bannerDescription {
		b.WriteString(l + "\n")
	}
	b.WriteString("\n")
	if len(entries) == 0 {
		b.WriteString(noCommandsText + "\n")
		return b.String()
	}
	width := 0
	for _, e := range entries {
		width = max(width, ansi.StringWidth(e.Path))
	}
	for _, e := range entries {
		if e.Short == "" {
			b.WriteString(e.Path + "\n")
			continue
		}
		b.WriteString(padRight(e.Path, width) + "  " + e.Short + "\n")
	}
	return b.String()
}

// bannerWidth is the display width of the large lettering.
//
// In: none. Out: the width in cells of the widest lettering line.
// Side effects: none.
func bannerWidth() int {
	w := 0
	for _, l := range bannerLines {
		w = max(w, ansi.StringWidth(l))
	}
	return w
}
