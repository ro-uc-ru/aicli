package tui

import (
	"io"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

// errorPrefix is prepended to the first line of the styled error output.
const errorPrefix = "✗ "

// RenderError builds the error message written to stderr (FR-023).
//
// In: msg, the error text; its first line is the reason and any further
// lines are continuation lines (e.g. "Usage: aicli new <name>"), given
// without indentation; styled, whether to add Lipgloss styling (see
// ShouldStyle, evaluated on stderr).
// Out: "Error: <reason>" plus each continuation line indented by two spaces,
// ending in a newline. Styled mode only adds ANSI sequences and a "✗ "
// prefix on the first line, so stripping both yields the plain text.
// Side effects: none; the color profile is forced on a private renderer.
func RenderError(msg string, styled bool) string {
	lines := strings.Split(strings.TrimRight(msg, "\n"), "\n")

	var b strings.Builder
	if !styled {
		b.WriteString("Error: " + lines[0] + "\n")
		for _, l := range lines[1:] {
			b.WriteString("  " + l + "\n")
		}
		return b.String()
	}

	r := lipgloss.NewRenderer(io.Discard)
	r.SetColorProfile(termenv.ANSI256)
	label := r.NewStyle().Bold(true).Foreground(lipgloss.Color("1"))
	dim := r.NewStyle().Faint(true)

	b.WriteString(label.Render(errorPrefix+"Error:") + " " + lines[0] + "\n")
	for _, l := range lines[1:] {
		b.WriteString(dim.Render("  "+l) + "\n")
	}
	return b.String()
}
