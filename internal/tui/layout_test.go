package tui

import "testing"

// TestComputeLayout checks the parts and the area for both views from one row
// of height up to the full layout, with the wide and the narrow banner.
//
// Covers: EC-003, FR-019, FR-020
func TestComputeLayout(t *testing.T) {
	type want struct {
		banner, header, ind, footer bool
		bannerRows, top, area       int
	}
	cases := []struct {
		name          string
		width, height int
		v             view
		content       int
		want          want
	}{
		{"list full", 80, 30, viewList, 3, want{true, false, true, true, 9, 10, 3}},
		{"list scrolls under the banner", 80, 14, viewList, 20, want{true, false, true, true, 9, 10, 2}},
		{"list banner kept, 1 row area", 80, 13, viewList, 20, want{true, false, true, true, 9, 10, 1}},
		{"list narrow banner", 20, 14, viewList, 20, want{true, false, true, true, 4, 5, 7}},
		{"list banner hidden", 80, 12, viewList, 20, want{false, false, true, true, 0, 1, 9}},
		{"list banner hidden, 1 row area", 80, 4, viewList, 20, want{false, false, true, true, 0, 1, 1}},
		{"list indicators hidden", 80, 3, viewList, 20, want{false, false, false, true, 0, 0, 2}},
		{"list footer hidden", 80, 1, viewList, 20, want{false, false, false, false, 0, 0, 1}},
		{"list height 0", 80, 0, viewList, 20, want{false, false, false, false, 0, 0, 1}},
		{"list empty", 80, 30, viewList, 0, want{true, false, true, true, 9, 10, 1}},
		{"detail full", 80, 30, viewDetail, 40, want{false, true, true, true, 0, 3, 25}},
		{"detail short help", 80, 30, viewDetail, 5, want{false, true, true, true, 0, 3, 5}},
		{"detail 1 row area", 80, 6, viewDetail, 40, want{false, true, true, true, 0, 3, 1}},
		{"detail indicators hidden", 80, 5, viewDetail, 40, want{false, true, false, true, 0, 2, 2}},
		{"detail footer hidden", 80, 3, viewDetail, 40, want{false, true, false, false, 0, 2, 1}},
		{"detail header hidden", 80, 2, viewDetail, 40, want{false, false, false, false, 0, 0, 2}},
		{"detail 1 row", 80, 1, viewDetail, 40, want{false, false, false, false, 0, 0, 1}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			l := computeLayout(c.width, c.height, c.v, c.content)
			got := want{l.Banner, l.Header, l.Indicators, l.Footer, l.BannerRows, l.AreaTop, l.AreaHeight}
			if got != c.want {
				t.Errorf("layout = %+v, want %+v", got, c.want)
			}
		})
	}
}

// TestEntryAtRow checks the row to entry mapping with a scroll offset: rows
// outside the area and rows past the last entry map to no entry.
//
// Covers: FR-007, EC-004
func TestEntryAtRow(t *testing.T) {
	l := computeLayout(80, 20, viewList, 10) // banner 0-8, indicator 9, area 10..
	if l.AreaTop != 10 {
		t.Fatalf("AreaTop = %d", l.AreaTop)
	}
	cases := []struct {
		name        string
		row, offset int
		entries     int
		wantI       int
		wantOK      bool
	}{
		{"banner row", 2, 0, 10, 0, false},
		{"indicator row", 9, 0, 10, 0, false},
		{"first area row", 10, 0, 10, 0, true},
		{"second area row", 11, 0, 10, 1, true},
		{"with offset", 11, 4, 10, 5, true},
		{"last entry", 10, 9, 10, 9, true},
		{"past the last entry", 12, 9, 10, 0, false},
		{"below the area", 10 + l.AreaHeight, 0, 10, 0, false},
		{"footer row", 19, 0, 10, 0, false},
		{"negative row", -1, 0, 10, 0, false},
		{"empty list", 10, 0, 0, 0, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			i, ok := entryAtRow(l, c.row, c.offset, c.entries)
			if i != c.wantI || ok != c.wantOK {
				t.Errorf("entryAtRow = %d, %v; want %d, %v", i, ok, c.wantI, c.wantOK)
			}
		})
	}
}
