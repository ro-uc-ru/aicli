package tui

import "testing"

// TestComputeConfirmLayout checks the confirmation layout: everything
// visible, paths scrolling, the file list hidden, and a wrapped command
// taller than the terminal that keeps the buttons and one command line.
//
// Covers: FR-018, EC-010
func TestComputeConfirmLayout(t *testing.T) {
	type want struct {
		header, desc, list, footer bool
		cmdTop, cmdRows            int
		heading, pathsTop, paths   int
		buttons                    int
		scrolls                    bool
	}
	cases := []struct {
		name               string
		height             int
		desc               bool
		cmdLines, pathsLen int
		want               want
	}{
		// header 0, desc 1, blank 2, cmd 3, blank 4, heading 5, up 6, paths 7-8, down 9, buttons 10, footer 11.
		{"everything visible", 30, true, 1, 2, want{true, true, true, true, 3, 1, 5, 7, 2, 10, false}},
		{"exact fit", 12, true, 1, 2, want{true, true, true, true, 3, 1, 5, 7, 2, 10, false}},
		{"paths scroll", 11, true, 1, 2, want{true, true, true, true, 3, 1, 5, 7, 1, 9, true}},
		{"paths scroll, long list", 12, true, 2, 20, want{true, true, true, true, 3, 2, 6, 8, 1, 10, true}},
		{"file list hidden", 10, true, 1, 2, want{true, true, false, true, 3, 1, 0, 0, 0, 4, false}},
		{"file list hidden, 2-line command", 9, true, 2, 5, want{true, true, false, true, 3, 2, 0, 0, 0, 5, false}},
		{"description hidden", 5, true, 2, 5, want{true, false, false, true, 1, 2, 0, 0, 0, 3, false}},
		{"footer hidden", 4, true, 2, 5, want{true, false, false, false, 1, 2, 0, 0, 0, 3, false}},
		{"header hidden", 3, true, 2, 5, want{false, false, false, false, 0, 2, 0, 0, 0, 2, false}},
		{"command scrolls above the buttons", 2, true, 5, 3, want{false, false, false, false, 0, 1, 0, 0, 0, 1, true}},
		{"command scrolls, 3 rows", 3, true, 5, 3, want{false, false, false, false, 0, 2, 0, 0, 0, 2, true}},
		{"height 0", 0, true, 5, 3, want{false, false, false, false, 0, 1, 0, 0, 0, 1, true}},
		{"no description", 30, false, 1, 2, want{true, false, true, true, 1, 1, 3, 5, 2, 8, false}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			l := computeConfirmLayout(c.height, c.desc, c.cmdLines, c.pathsLen)
			got := want{l.Header, l.Description, l.List, l.Footer, l.CmdTop, l.CmdRows,
				l.HeadingRow, l.PathsTop, l.PathRows, l.ButtonsRow, l.Scrolls}
			if got != c.want {
				t.Errorf("layout = %+v, want %+v", got, c.want)
			}
		})
	}
}
