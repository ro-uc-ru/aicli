package tui

import (
	"io"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

// homeStyles holds the Lipgloss styles of the home screen, one per row of
// the design's colour table (FR-013).
type homeStyles struct {
	// Banner styles the large lettering.
	Banner lipgloss.Style
	// BannerDesc styles the banner description lines.
	BannerDesc lipgloss.Style
	// Entry styles an entry that is not under the pointer.
	Entry lipgloss.Style
	// Parent styles the parent prefix of a subcommand entry.
	Parent lipgloss.Style
	// Selected styles the entry under the pointer, pointer included.
	Selected lipgloss.Style
	// Header styles the detail view header.
	Header lipgloss.Style
	// Heading styles the headings of the detail view (`Usage:`, `Flags:`).
	Heading lipgloss.Style
	// Flag styles the flags of the detail view.
	Flag lipgloss.Style
	// Footer styles the key hints.
	Footer lipgloss.Style
	// Indicator styles the scroll indicators.
	Indicator lipgloss.Style
}

// newHomeStyles builds the home screen styles.
//
// In: styled, whether to add ANSI colour (see ShouldStyle).
// Out: the styles. When styled they use ANSI-256 colours on a private
// renderer; otherwise every style is a no-op, so the rendered text is the
// same and only ANSI sequences differ.
// Side effects: none; no global Lipgloss state changes.
func newHomeStyles(styled bool) homeStyles {
	r := lipgloss.NewRenderer(io.Discard)
	if !styled {
		r.SetColorProfile(termenv.Ascii)
		n := r.NewStyle()
		return homeStyles{n, n, n, n, n, n, n, n, n, n}
	}
	r.SetColorProfile(termenv.ANSI256)
	accent := lipgloss.Color("117")
	dim := lipgloss.Color("245")
	normal := lipgloss.Color("255")
	return homeStyles{
		Banner:     r.NewStyle().Foreground(accent).Bold(true),
		BannerDesc: r.NewStyle().Foreground(dim),
		Entry:      r.NewStyle().Foreground(normal),
		Parent:     r.NewStyle().Foreground(dim),
		Selected:   r.NewStyle().Foreground(accent).Bold(true).Background(lipgloss.Color("236")),
		Header:     r.NewStyle().Foreground(accent).Bold(true),
		Heading:    r.NewStyle().Foreground(accent).Bold(true),
		Flag:       r.NewStyle().Foreground(normal),
		Footer:     r.NewStyle().Foreground(dim),
		Indicator:  r.NewStyle().Foreground(dim),
	}
}
