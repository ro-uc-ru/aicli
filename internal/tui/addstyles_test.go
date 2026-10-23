package tui

import (
	"strings"
	"testing"
)

// TestAddStyles checks that every add style renders without ESC when not
// styled, that the styled output stripped of ANSI equals the unstyled one,
// and that each styled output has an ESC.
//
// Covers: FR-006, FR-007, FR-008, FR-011, FR-016, FR-018
func TestAddStyles(t *testing.T) {
	const sample = "> feature-1  Description"
	plain, styled := newAddStyles(false), newAddStyles(true)
	pairs := map[string][2]string{
		"header":   {plain.Header.Render(sample), styled.Header.Render(sample)},
		"dim":      {plain.Dim.Render(sample), styled.Dim.Render(sample)},
		"normal":   {plain.Normal.Render(sample), styled.Normal.Render(sample)},
		"selected": {plain.Selected.Render(sample), styled.Selected.Render(sample)},
		"focused":  {plain.FieldFocused.Render(sample), styled.FieldFocused.Render(sample)},
		"choice":   {plain.Choice.Render(sample), styled.Choice.Render(sample)},
		"error":    {plain.Error.Render(sample), styled.Error.Render(sample)},

		"check cursor":     {plain.CheckCursor.Render(sample), styled.CheckCursor.Render(sample)},
		"check selected":   {plain.CheckSelected.Render(sample), styled.CheckSelected.Render(sample)},
		"check unselected": {plain.CheckUnselected.Render(sample), styled.CheckUnselected.Render(sample)},
		"already added":    {plain.AlreadyAdded.Render(sample), styled.AlreadyAdded.Render(sample)},
		"all added":        {plain.AllAdded.Render(sample), styled.AllAdded.Render(sample)},
		"will update":      {plain.WillUpdate.Render(sample), styled.WillUpdate.Render(sample)},
	}
	for name, pair := range pairs {
		if strings.Contains(pair[0], "\x1b") {
			t.Errorf("%s: unstyled output has ESC: %q", name, pair[0])
		}
		if !strings.Contains(pair[1], "\x1b") {
			t.Errorf("%s: styled output has no ESC", name)
		}
		if got := styleAnsiRe.ReplaceAllString(pair[1], ""); got != pair[0] || got != sample {
			t.Errorf("%s: stripped = %q, plain = %q", name, got, pair[0])
		}
	}
	if styled.Selected.Render(sample) == styled.Normal.Render(sample) {
		t.Error("selected style equals the normal style")
	}
	for name, want := range map[string]string{
		"check cursor":     "1;38;5;117m",
		"check selected":   "38;5;117m",
		"check unselected": "38;5;255m",
		"already added":    "38;5;245m",
		"all added":        "38;5;245m",
		"will update":      "38;5;245m",
	} {
		if !strings.Contains(pairs[name][1], want) {
			t.Errorf("%s: styled output %q lacks %s", name, pairs[name][1], want)
		}
	}
}

// TestAddButtons checks the button markers in both modes and that the Add
// and Cancel styles differ, focused or not.
//
// Covers: FR-018
func TestAddButtons(t *testing.T) {
	plain, styled := newAddStyles(false), newAddStyles(true)
	for _, add := range []bool{true, false} {
		for _, focused := range []bool{true, false} {
			label := "Cancel"
			if add {
				label = "Add"
			}
			want := "  " + label + "  "
			if focused {
				want = "[ " + label + " ]"
			}
			p, s := plain.AddButton(label, add, focused), styled.AddButton(label, add, focused)
			if p != want || strings.Contains(p, "\x1b") {
				t.Errorf("plain %s focused=%v = %q, want %q", label, focused, p, want)
			}
			if !strings.Contains(s, "\x1b") {
				t.Errorf("styled %s focused=%v has no ESC", label, focused)
			}
			if got := styleAnsiRe.ReplaceAllString(s, ""); got != want {
				t.Errorf("styled stripped %s focused=%v = %q, want %q", label, focused, got, want)
			}
		}
	}
	for _, focused := range []bool{true, false} {
		if styled.AddButton("X", true, focused) == styled.AddButton("X", false, focused) {
			t.Errorf("Add and Cancel styles equal, focused=%v", focused)
		}
	}
	if styled.AddButton("Add", true, true) == styled.AddButton("Add", true, false) {
		t.Error("focused Add not distinguished by styling")
	}
}
