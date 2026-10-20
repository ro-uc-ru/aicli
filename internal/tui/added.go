package tui

import (
	"io"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

// RenderAdded builds the report printed after a feature creates its files
// (FR-025).
//
// In: feature, the feature name; paths, the created paths in list order;
// steps, the one-line next steps in declaration order (may be empty);
// styled, whether to add Lipgloss styling (see ShouldStyle).
// Out: the full text, ending in a newline: one "create  <path>" line per
// path, a blank line, "Added <feature>", and, if steps is not empty, a blank
// line, "Next steps:" and each step indented by two spaces. Styled mode only
// adds ANSI sequences and a "✓ " prefix on the success line, so stripping
// both yields the plain text.
// Side effects: none; the color profile is forced on a private renderer, so
// no global Lipgloss state changes.
func RenderAdded(feature string, paths, steps []string, styled bool) string {
	id := func(s ...string) string { return strings.Join(s, "") }
	verb, title, heading, step := id, id, id, id
	prefix := ""
	if styled {
		r := lipgloss.NewRenderer(io.Discard)
		r.SetColorProfile(termenv.ANSI256)
		verb = r.NewStyle().Foreground(lipgloss.Color("2")).Render
		title = r.NewStyle().Bold(true).Foreground(lipgloss.Color("2")).Render
		heading = r.NewStyle().Bold(true).Foreground(lipgloss.Color("245")).Render
		step = r.NewStyle().Foreground(lipgloss.Color("117")).Render
		prefix = successPrefix
	}

	var b strings.Builder
	for _, p := range paths {
		b.WriteString(verb("create") + "  " + p + "\n")
	}
	b.WriteString("\n" + title(prefix+"Added "+feature) + "\n")
	if len(steps) > 0 {
		b.WriteString("\n" + heading("Next steps:") + "\n")
		for _, s := range steps {
			b.WriteString("  " + step(s) + "\n")
		}
	}
	return b.String()
}
