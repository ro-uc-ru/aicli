package tui

import "testing"

// TestComputeConfirmLayout checks the confirmation layout: everything
// visible, the list rows scrolling together (both lists, one list, or the
// error lines count as plain rows), the list hidden, and a wrapped command
// taller than the terminal that keeps the buttons and one command line.
//
// Covers: FR-011, FR-018, EC-010
func TestComputeConfirmLayout(t *testing.T) {
	type want struct {
		header, desc, list, footer bool
		cmdTop, cmdRows            int
		listTop, listRows          int
		buttons                    int
		scrolls                    bool
	}
	cases := []struct {
		name              string
		height            int
		desc              bool
		cmdLines, rowsLen int
		want              want
	}{
		// header 0, desc 1, blank 2, cmd 3, up 4, rows 5-6, down 7, buttons 8, footer 9.
		{"everything visible, one list", 30, true, 1, 2, want{true, true, true, true, 3, 1, 5, 2, 8, false}},
		// Will create: + 2 paths, blank, Will update: + 2 paths.
		{"both lists visible", 30, true, 1, 8, want{true, true, true, true, 3, 1, 5, 8, 14, false}},
		{"exact fit", 10, true, 1, 2, want{true, true, true, true, 3, 1, 5, 2, 8, false}},
		{"both lists scroll together", 10, true, 1, 8, want{true, true, true, true, 3, 1, 5, 2, 8, true}},
		{"rows scroll", 9, true, 1, 2, want{true, true, true, true, 3, 1, 5, 1, 7, true}},
		{"rows scroll, long list", 12, true, 2, 20, want{true, true, true, true, 3, 2, 6, 3, 10, true}},
		{"list hidden", 8, true, 1, 2, want{true, true, false, true, 3, 1, 0, 0, 4, false}},
		{"list hidden, 2-line command", 9, true, 2, 5, want{true, true, false, true, 3, 2, 0, 0, 5, false}},
		{"description hidden", 5, true, 2, 5, want{true, false, false, true, 1, 2, 0, 0, 3, false}},
		{"footer hidden", 4, true, 2, 5, want{true, false, false, false, 1, 2, 0, 0, 3, false}},
		{"header hidden", 3, true, 2, 5, want{false, false, false, false, 0, 2, 0, 0, 2, false}},
		{"command scrolls above the buttons", 2, true, 5, 3, want{false, false, false, false, 0, 1, 0, 0, 1, true}},
		{"command scrolls, 3 rows", 3, true, 5, 3, want{false, false, false, false, 0, 2, 0, 0, 2, true}},
		{"height 0", 0, true, 5, 3, want{false, false, false, false, 0, 1, 0, 0, 1, true}},
		{"no description", 30, false, 1, 2, want{true, false, true, true, 1, 1, 3, 2, 6, false}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			l := computeConfirmLayout(c.height, c.desc, c.cmdLines, c.rowsLen)
			got := want{l.Header, l.Description, l.List, l.Footer, l.CmdTop, l.CmdRows,
				l.ListTop, l.ListRows, l.ButtonsRow, l.Scrolls}
			if got != c.want {
				t.Errorf("layout = %+v, want %+v", got, c.want)
			}
		})
	}
}
