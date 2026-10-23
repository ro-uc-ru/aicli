package tui

import (
	"io"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"

	"github.com/ro-uc-ru/aicli/internal/gen"
)

// RenderAdded builds the report printed after a feature creates its files
// (FR-025).
//
// In: feature, the feature name; lines, the report lines in plan order;
// steps, the one-line next steps in declaration order (may be empty);
// styled, whether to add Lipgloss styling (see ShouldStyle).
// Out: the full text, ending in a newline: one "<verb>  <path>" line per
// report line, the verb (create, skip or update) padded to 6 characters
// (FR-011), a blank line, "Added <feature>", and, if steps is not empty, a blank
// line, "Next steps:" and each step indented by two spaces. Styled mode only
// adds ANSI sequences and a "✓ " prefix on the success line, so stripping
// both yields the plain text. Styled, create is green, update is 117 and skip
// is 245.
// Side effects: none; the color profile is forced on a private renderer, so
// no global Lipgloss state changes.
func RenderAdded(feature string, lines []gen.ReportLine, steps []string, styled bool) string {
	id := func(s ...string) string { return strings.Join(s, "") }
	title, heading, step := id, id, id
	verbs := map[gen.Verb]func(...string) string{gen.VerbCreate: id, gen.VerbSkip: id, gen.VerbUpdate: id}
	prefix := ""
	if styled {
		r := lipgloss.NewRenderer(io.Discard)
		r.SetColorProfile(termenv.ANSI256)
		verbs[gen.VerbCreate] = styleFunc(r.NewStyle().Foreground(lipgloss.Color("2")))
		verbs[gen.VerbUpdate] = styleFunc(r.NewStyle().Foreground(lipgloss.Color("117")))
		verbs[gen.VerbSkip] = styleFunc(r.NewStyle().Foreground(lipgloss.Color("245")))
		title = r.NewStyle().Bold(true).Foreground(lipgloss.Color("2")).Render
		heading = r.NewStyle().Bold(true).Foreground(lipgloss.Color("245")).Render
		step = r.NewStyle().Foreground(lipgloss.Color("117")).Render
		prefix = successPrefix
	}

	var b strings.Builder
	for _, l := range lines {
		word := string(l.Verb)
		b.WriteString(verbs[l.Verb](word) + strings.Repeat(" ", 8-len(word)) + l.Path + "\n")
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

// styleFunc adapts a Lipgloss style to the variadic render signature used by
// RenderAdded.
//
// In: st, the style. Out: a function that joins its arguments and renders
// them with st. No side effects.
func styleFunc(st lipgloss.Style) func(...string) string {
	return func(s ...string) string { return st.Render(strings.Join(s, "")) }
}
