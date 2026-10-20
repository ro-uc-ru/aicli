package tui

import "testing"

// TestComputeFormLayout checks the form layout for a tall terminal, for
// short terminals that scroll whole field blocks while keeping the focused
// field (with its error) visible, and for the hiding order down to one row.
//
// Covers: FR-011, EC-010
func TestComputeFormLayout(t *testing.T) {
	type want struct {
		header, desc, ind, footer bool
		top, area, first, last    int
	}
	// Three fields: 5 rows (choice), 3 rows, 4 rows (3 plus an error).
	h3 := []int{5, 3, 4}
	cases := []struct {
		name    string
		height  int
		desc    bool
		heights []int
		focus   int
		want    want
	}{
		{"tall, all visible", 40, true, h3, 0, want{true, true, true, true, 3, 14, 0, 3}},
		{"exact fit", 19, true, h3, 2, want{true, true, true, true, 3, 14, 0, 3}},
		{"scrolls, focus first", 14, true, h3, 0, want{true, true, true, true, 3, 9, 0, 2}},
		{"scrolls, focus last with error", 14, true, h3, 2, want{true, true, true, true, 3, 9, 1, 3}},
		{"focus middle", 11, true, h3, 1, want{true, true, true, true, 3, 6, 1, 2}},
		{"focused block taller than area", 8, true, h3, 0, want{true, true, true, true, 3, 3, 0, 1}},
		{"1 row area", 6, true, h3, 2, want{true, true, true, true, 3, 1, 2, 3}},
		{"description hidden", 5, true, h3, 0, want{true, false, true, true, 2, 1, 0, 1}},
		{"indicators hidden", 3, true, h3, 0, want{true, false, false, true, 1, 1, 0, 1}},
		{"footer hidden", 2, true, h3, 0, want{true, false, false, false, 1, 1, 0, 1}},
		{"header hidden", 1, true, h3, 2, want{false, false, false, false, 0, 1, 2, 3}},
		{"height 0", 0, true, h3, 0, want{false, false, false, false, 0, 1, 0, 1}},
		{"no description line", 40, false, h3, 0, want{true, false, true, true, 2, 14, 0, 3}},
		{"focus out of range", 40, true, h3, 9, want{true, true, true, true, 3, 14, 0, 3}},
		{"no fields", 40, true, nil, 0, want{true, true, true, true, 3, 1, 0, 0}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			l := computeFormLayout(c.height, c.desc, c.heights, c.focus)
			got := want{l.Header, l.Description, l.Indicators, l.Footer, l.AreaTop, l.AreaHeight, l.First, l.Last}
			if got != c.want {
				t.Errorf("layout = %+v, want %+v", got, c.want)
			}
		})
	}
}

// TestFormLayoutMore checks the scroll-indicator flags.
//
// Covers: FR-011
func TestFormLayoutMore(t *testing.T) {
	l := computeFormLayout(14, true, []int{5, 3, 4}, 2)
	if above, below := l.More(3); !above || below {
		t.Errorf("More = %v, %v", above, below)
	}
	l = computeFormLayout(14, true, []int{5, 3, 4}, 0)
	if above, below := l.More(3); above || !below {
		t.Errorf("More = %v, %v", above, below)
	}
}
