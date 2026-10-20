package tui

import (
	"strings"

	"github.com/charmbracelet/x/ansi"
)

// addHeader is the header and description line shared by the static output
// and the EC-003 text.
const addHeader = "aicli › add\n" + addDescription + "\n"

// addDescription is the description line under the add header.
const addDescription = "Add an AI/LLM feature to this project."

// RenderAddStatic builds the plain text that `aicli add` prints when it is not
// interactive, or when no feature is listed (FR-005, EC-003).
//
// In: features, the features of the add command, in Cobra order; only those
// with Listed set are shown.
// Out: the text, ending in a newline. With at least one listed feature: the
// header `aicli › add`, the description line, a blank line, one line per
// listed feature (the name padded with spaces to the widest name by display
// width, two spaces, then the short description; the name alone, without
// trailing spaces, if it has none), a blank line and the closing line
// "Run `aicli add <feature> --help` to see its inputs.". With none: the header,
// the description line, a blank line and "No features available.". The text is
// always plain (no ANSI) and ignores the terminal width.
// Side effects: none.
func RenderAddStatic(features []Feature) string {
	var listed []Feature
	width := 0
	for _, f := range features {
		if f.Listed {
			listed = append(listed, f)
			width = max(width, ansi.StringWidth(f.Name))
		}
	}
	var b strings.Builder
	b.WriteString(addHeader + "\n")
	if len(listed) == 0 {
		b.WriteString("No features available.\n")
		return b.String()
	}
	for _, f := range listed {
		if f.Short == "" {
			b.WriteString(f.Name + "\n")
			continue
		}
		b.WriteString(padRight(f.Name, width) + "  " + f.Short + "\n")
	}
	b.WriteString("\nRun `aicli add <feature> --help` to see its inputs.\n")
	return b.String()
}
