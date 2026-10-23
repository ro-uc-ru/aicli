package tui

const (
	// confirmListOverhead is the height of the list block without its rows:
	// the two scroll-indicator lines, of which the upper one is also the
	// blank line that separates the list from the command (FR-011, EC-010).
	confirmListOverhead = 2
)

// confirmLayout is the vertical arrangement of the confirmation screen, top
// to bottom: the header, the description line and a blank line, the
// equivalent command, the list block (the indicator line above, the list
// rows, the indicator line below), the buttons and the footer. The list rows
// are `Will create:`, its paths, a blank row, `Will update:` and its paths,
// or the lines of the error (FR-011, FR-018).
type confirmLayout struct {
	// Header, Description, List and Footer report which parts are shown.
	// Description covers the description line and the blank line after it;
	// List covers the whole file list block.
	Header, Description, List, Footer bool
	// CmdTop is the screen row of the first visible command line.
	CmdTop int
	// CmdRows is the number of visible command lines, at least 1; fewer
	// than the command has when it scrolls.
	CmdRows int
	// ListTop is the screen row of the first visible list row, when List is
	// set.
	ListTop int
	// ListRows is the number of visible list rows, at least 1 when List is
	// set; fewer than there are when they scroll.
	ListRows int
	// ButtonsRow is the screen row of the buttons.
	ButtonsRow int
	// Scrolls reports whether the list or the command are cut, so the
	// footer also lists `↑/↓ scroll`.
	Scrolls bool
}

// computeConfirmLayout decides which parts of the confirmation fit (FR-018,
// EC-010).
//
// In: height, the terminal height in rows; description, whether the feature
// has a description; cmdLines, the number of lines of the wrapped
// equivalent command, at least 1; rowCount, the total number of list rows:
// both headings, their paths and the blank row between them, or the lines of
// the error; at least 1.
// Out: the layout. Parts are hidden, in this order, until the whole command,
// the buttons and, if the list is shown, its indicator lines and one row
// fit: the list block, the description, the footer, the header. If the command and the buttons still do not fit, the command
// shows as many lines as fit above the buttons, from its first, and never
// fewer than one. The list rows scroll together in the rows left after the
// command and the list overhead. The buttons are always shown.
// Side effects: none.
func computeConfirmLayout(height int, description bool, cmdLines, rowCount int) confirmLayout {
	cmdLines, rowCount = max(cmdLines, 1), max(rowCount, 1)
	l := confirmLayout{Header: true, Description: description, List: true, Footer: true}
	used := func() int {
		n := 1 // buttons
		for _, show := range []bool{l.Header, l.Footer} {
			if show {
				n++
			}
		}
		if l.Description {
			n += descriptionRows
		}
		return n
	}
	fits := func() bool {
		need := used() + cmdLines
		if l.List {
			need += confirmListOverhead + 1
		}
		return need <= height
	}
	for _, p := range []*bool{&l.List, &l.Description, &l.Footer, &l.Header} {
		if fits() {
			break
		}
		*p = false
	}
	l.CmdRows = max(min(cmdLines, height-used()), 1)
	row := 0
	if l.Header {
		row++
	}
	if l.Description {
		row += descriptionRows
	}
	l.CmdTop = row
	row += l.CmdRows
	if l.List {
		l.ListRows = max(min(rowCount, height-used()-cmdLines-confirmListOverhead), 1)
		l.ListTop = row + 1
		row += confirmListOverhead + l.ListRows
	}
	l.ButtonsRow = row
	l.Scrolls = l.CmdRows < cmdLines || (l.List && l.ListRows < rowCount)
	return l
}
