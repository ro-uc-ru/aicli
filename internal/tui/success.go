// Package tui renders the styled or plain terminal output of the CLI. It is
// the only package that imports Lipgloss (ARC-7) and spawns no processes.
package tui

import (
	"io"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
	"golang.org/x/term"
)

// ShouldStyle reports whether output written to w may carry ANSI styling.
//
// In: w, the destination writer (stdout for success output, stderr for
// errors); getenv, the environment lookup (typically os.Getenv).
// Out: true only if w is a terminal, NO_COLOR is unset or empty and TERM is
// not "dumb"; false otherwise, including when w is not an *os.File.
// Side effects: none beyond querying the terminal state of w.
func ShouldStyle(w io.Writer, getenv func(string) string) bool {
	f, ok := w.(*os.File)
	if !ok || !term.IsTerminal(int(f.Fd())) {
		return false
	}
	if getenv("NO_COLOR") != "" {
		return false
	}
	return getenv("TERM") != "dumb"
}

// successPrefix is prepended to the first line of the styled success output.
const successPrefix = "✓ "

// RenderSuccess builds the success message of `aicli new` (FR-015).
//
// In: name, the created project name, substituted verbatim; styled, whether
// to add Lipgloss styling (see ShouldStyle).
// Out: the full text, ending in a newline. Plain mode is the exact FR-015
// text; styled mode only adds ANSI sequences and a "✓ " prefix on the first
// line, so stripping both yields the plain text.
// Side effects: none; the color profile is forced on a private renderer, so
// no global Lipgloss state changes.
func RenderSuccess(name string, styled bool) string {
	type line struct{ text, optional string }
	steps := []line{
		{"cd " + name, ""},
		{"cp .env.example .env", "   (optional)"},
		{"uv sync", ""},
		{"uv run uvicorn app.main:app --reload", ""},
	}

	if !styled {
		var b strings.Builder
		b.WriteString("Created project " + name + "\n\nNext steps:\n")
		for _, s := range steps {
			b.WriteString("  " + s.text + s.optional + "\n")
		}
		return b.String()
	}

	r := lipgloss.NewRenderer(io.Discard)
	r.SetColorProfile(termenv.ANSI256)
	title := r.NewStyle().Bold(true).Foreground(lipgloss.Color("2"))
	heading := r.NewStyle().Bold(true).Foreground(lipgloss.Color("245"))
	command := r.NewStyle().Foreground(lipgloss.Color("117"))
	dim := r.NewStyle().Faint(true)

	var b strings.Builder
	b.WriteString(title.Render(successPrefix+"Created project "+name) + "\n\n")
	b.WriteString(heading.Render("Next steps:") + "\n")
	for _, s := range steps {
		b.WriteString("  " + command.Render(s.text))
		if s.optional != "" {
			b.WriteString(dim.Render(s.optional))
		}
		b.WriteString("\n")
	}
	return b.String()
}
