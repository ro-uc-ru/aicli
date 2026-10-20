package tui

// formLayout is the vertical arrangement of the form screen, top to bottom:
// the header, the description line, the scroll-indicator line above, the
// fields, the scroll-indicator line below and the footer (FR-011).
type formLayout struct {
	// Header reports whether the header is shown.
	Header bool
	// Description reports whether the feature description line is shown.
	Description bool
	// Indicators reports whether both scroll-indicator lines are shown.
	Indicators bool
	// Footer reports whether the footer is shown.
	Footer bool
	// AreaTop is the screen row of the first row of the fields area.
	AreaTop int
	// AreaHeight is the number of rows of the fields area, at least 1.
	AreaHeight int
	// First is the index of the first visible field; Last is the index
	// after the last visible field. The focused field is always in
	// [First, Last).
	First, Last int
}

// More reports whether fields are hidden above and below the window.
//
// In: total, the number of fields. Out: above, true if First > 0; below,
// true if Last < total. Side effects: none.
func (l formLayout) More(total int) (above, below bool) {
	return l.First > 0, l.Last < total
}

// computeFormLayout decides which parts of the form fit and which fields
// are visible (FR-011, EC-010).
//
// In: height, the terminal height in rows; description, whether the feature
// has a description line; heights, the height in rows of each field block
// (title, description, input or choices, and error if any), in declaration
// order; focus, the index of the focused field, clamped into range.
// Out: the layout. Parts are hidden, in this order, until the area gets at
// least one row: the description line, the indicator lines, the footer, the
// header. Fields are one blank row apart and scroll as whole blocks: the
// window starts at the first field and moves down just enough to show the
// focused one in full, then takes as many following fields as fit. A
// focused block taller than the area is the only one shown, from its top.
// The area is no taller than the fields need and never less than one row,
// even when height is below 1.
// Side effects: none.
func computeFormLayout(height int, description bool, heights []int, focus int) formLayout {
	l := formLayout{Header: true, Description: description, Indicators: true, Footer: true}
	used := func() int {
		n := 0
		for _, show := range []bool{l.Header, l.Description, l.Footer} {
			if show {
				n++
			}
		}
		if l.Indicators {
			n += indicatorRows
		}
		return n
	}
	hide := []*bool{&l.Description, &l.Indicators, &l.Footer, &l.Header}
	for _, p := range hide {
		if height-used() >= 1 {
			break
		}
		*p = false
	}
	total := 0
	for _, h := range heights {
		total += h
	}
	if len(heights) > 1 {
		total += len(heights) - 1
	}
	l.AreaHeight = max(min(height-used(), max(total, 1)), 1)
	if l.Header {
		l.AreaTop++
	}
	if l.Description {
		l.AreaTop++
	}
	if l.Indicators {
		l.AreaTop++
	}
	if len(heights) == 0 {
		return l
	}
	focus = max(0, min(focus, len(heights)-1))
	span := func(first, last int) int {
		n := 0
		for _, h := range heights[first:last] {
			n += h
		}
		return n + max(last-first-1, 0)
	}
	l.First = 0
	for l.First < focus && span(l.First, focus+1) > l.AreaHeight {
		l.First++
	}
	l.Last = focus + 1
	for l.Last < len(heights) && span(l.First, l.Last+1) <= l.AreaHeight {
		l.Last++
	}
	return l
}
