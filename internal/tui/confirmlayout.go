package tui

const (
	// confirmListOverhead is the height of the file list block without its
	// paths: the blank line before it, the heading and the two
	// scroll-indicator lines (FR-018, EC-010).
	confirmListOverhead = 4
)

// confirmLayout is the vertical arrangement of the confirmation screen, top
// to bottom: the header, the description line and a blank line, the
// equivalent command, the file list block (a blank line, the heading, the
// indicator line above, the paths, the indicator line below), the buttons
// and the footer (FR-018).
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
	// HeadingRow is the screen row of the list heading, when List is set.
	HeadingRow int
	// PathsTop is the screen row of the first visible path, when List is
	// set.
	PathsTop int
	// PathRows is the number of visible paths, at least 1 when List is set;
	// fewer than there are when they scroll.
	PathRows int
	// ButtonsRow is the screen row of the buttons.
	ButtonsRow int
	// Scrolls reports whether the paths or the command are cut, so the
	// footer also lists `↑/↓ scroll`.
	Scrolls bool
}

// computeConfirmLayout decides which parts of the confirmation fit (FR-018,
// EC-010).
//
// In: height, the terminal height in rows; description, whether the feature
// has a description; cmdLines, the number of lines of the wrapped
// equivalent command, at least 1; pathCount, the number of lines of the file
// list (paths, or the lines of the error), at least 1.
// Out: the layout. Parts are hidden, in this order, until the whole command,
// the buttons and, if the list is shown, its heading, indicator lines and
// one path fit: the file list block, the description, the footer, the
// header. If the command and the buttons still do not fit, the command
// shows as many lines as fit above the buttons, from its first, and never
// fewer than one. Paths scroll in the rows left after the command and the
// list overhead. The buttons are always shown.
// Side effects: none.
func computeConfirmLayout(height int, description bool, cmdLines, pathCount int) confirmLayout {
	cmdLines, pathCount = max(cmdLines, 1), max(pathCount, 1)
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
		l.PathRows = max(min(pathCount, height-used()-cmdLines-confirmListOverhead), 1)
		l.HeadingRow = row + 1
		l.PathsTop = row + 3
		row += confirmListOverhead + l.PathRows
	}
	l.ButtonsRow = row
	l.Scrolls = l.CmdRows < cmdLines || (l.List && l.PathRows < pathCount)
	return l
}
