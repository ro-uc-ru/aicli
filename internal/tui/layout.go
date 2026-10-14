package tui

// view identifies which screen of the home program is shown.
type view int

const (
	// viewList is the list of commands, topped by the banner.
	viewList view = iota
	// viewDetail is the help of one command, topped by a header.
	viewDetail
)

const (
	// bannerRowsWide is the height of the large banner: the lettering, the
	// two description lines and a blank separator line.
	bannerRowsWide = 9
	// bannerRowsNarrow is the height of the narrow banner: the `AICLI` line,
	// the two description lines and a blank separator line.
	bannerRowsNarrow = 4
	// headerRows is the height of the detail header: the header line and a
	// blank separator line (FR-019).
	headerRows = 2
	// indicatorRows is the height of the two scroll-indicator lines, one
	// above and one below the area (FR-020).
	indicatorRows = 2
	// footerRows is the height of the key-hint footer (FR-017).
	footerRows = 1
)

// layout is the vertical arrangement of the screen, top to bottom: banner
// (list view) or header (detail view), the indicator line above, the area,
// the indicator line below, and the footer.
type layout struct {
	// Banner reports whether the banner is shown (list view only).
	Banner bool
	// BannerRows is the height of the banner, 0 when it is hidden.
	BannerRows int
	// Header reports whether the detail header is shown (detail view only).
	Header bool
	// Indicators reports whether both scroll-indicator lines are shown.
	Indicators bool
	// Footer reports whether the footer is shown.
	Footer bool
	// AreaTop is the screen row of the first row of the list or help area.
	AreaTop int
	// AreaHeight is the number of rows of the area, at least 1.
	AreaHeight int
}

// computeLayout decides which parts fit on the screen (EC-003).
//
// In: width, the terminal width in cells, which only selects the wide or
// narrow banner; height, the terminal height in rows; v, the view shown;
// contentLen, the number of entries (list) or help lines (detail).
// Out: the layout. Parts are hidden, in this order, until the area gets at
// least one row: the banner, the indicator lines, the footer, then (detail
// view) the header. The area is as tall as the room left, but no taller than
// the content, and never less than one row, even when height is below 1.
// Side effects: none.
func computeLayout(width, height int, v view, contentLen int) layout {
	l := layout{Footer: true, Indicators: true}
	if v == viewList {
		l.Banner = true
		l.BannerRows = bannerRowsWide
		if width < bannerWidth() {
			l.BannerRows = bannerRowsNarrow
		}
	} else {
		l.Header = true
	}
	used := func() int {
		n := 0
		if l.Banner {
			n += l.BannerRows
		}
		if l.Header {
			n += headerRows
		}
		if l.Indicators {
			n += indicatorRows
		}
		if l.Footer {
			n += footerRows
		}
		return n
	}
	if height-used() < 1 && l.Banner {
		l.Banner, l.BannerRows = false, 0
	}
	if height-used() < 1 {
		l.Indicators = false
	}
	if height-used() < 1 {
		l.Footer = false
	}
	if height-used() < 1 {
		l.Header = false
	}
	l.AreaHeight = max(min(height-used(), max(contentLen, 1)), 1)
	if l.Banner {
		l.AreaTop += l.BannerRows
	}
	if l.Header {
		l.AreaTop += headerRows
	}
	if l.Indicators {
		l.AreaTop++
	}
	return l
}

// entryAtRow maps a screen row to the entry shown on it (FR-007, EC-004).
//
// In: l, the list view layout; row, the screen row of a mouse click, from 0;
// offset, the index of the first visible entry; entries, the number of
// entries.
// Out: the entry index and true if row is inside the area and shows an
// entry; 0 and false for the banner, the indicator lines, the footer, empty
// space or any row outside the screen.
// Side effects: none.
func entryAtRow(l layout, row, offset, entries int) (int, bool) {
	if row < l.AreaTop || row >= l.AreaTop+l.AreaHeight {
		return 0, false
	}
	i := offset + row - l.AreaTop
	if i < 0 || i >= entries {
		return 0, false
	}
	return i, true
}
