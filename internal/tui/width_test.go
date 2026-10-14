package tui

import (
	"testing"

	"github.com/charmbracelet/x/ansi"
)

// TestTruncate checks truncation with `…` by display width around the cut
// point for ASCII, CJK and emoji text.
//
// Covers: EC-002, EC-010
func TestTruncate(t *testing.T) {
	cases := []struct {
		name  string
		in    string
		width int
		want  string
	}{
		{"ascii fits", "hello", 5, "hello"},
		{"ascii wider", "hello", 4, "hel…"},
		{"ascii much wider", "hello world", 6, "hello…"},
		{"width 1", "hello", 1, "…"},
		{"width 0", "hello", 0, ""},
		{"cjk fits", "日本語", 6, "日本語"},
		{"cjk cut on boundary", "日本語", 5, "日本…"},
		{"cjk cut mid character", "日本語", 4, "日…"},
		{"emoji fits", "🚀🚀", 4, "🚀🚀"},
		{"emoji cut", "🚀🚀🚀", 5, "🚀🚀…"},
		{"emoji cut mid character", "🚀🚀🚀", 4, "🚀…"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := truncate(c.in, c.width)
			if got != c.want {
				t.Errorf("truncate(%q, %d) = %q, want %q", c.in, c.width, got, c.want)
			}
			if w := ansi.StringWidth(got); w > c.width {
				t.Errorf("width %d exceeds %d", w, c.width)
			}
		})
	}
}

// TestPadRight checks padding by display width for ASCII, CJK and emoji.
//
// Covers: EC-010
func TestPadRight(t *testing.T) {
	cases := []struct {
		in    string
		width int
		want  string
	}{
		{"ab", 4, "ab  "},
		{"日本", 6, "日本  "},
		{"🚀", 4, "🚀  "},
		{"abcd", 2, "abcd"},
		{"", 2, "  "},
	}
	for _, c := range cases {
		if got := padRight(c.in, c.width); got != c.want {
			t.Errorf("padRight(%q, %d) = %q, want %q", c.in, c.width, got, c.want)
		}
	}
}

// TestBannerFor checks the large banner, the narrow banner just below the
// lettering's width, and truncated description lines.
//
// Covers: EC-002
func TestBannerFor(t *testing.T) {
	w := bannerWidth()
	wide := bannerFor(w)
	if len(wide) != 8 || wide[0] != bannerLines[0] || wide[6] != bannerDescription[0][:w-1]+"…" {
		t.Errorf("wide banner = %q", wide)
	}
	narrow := bannerFor(w - 1)
	if len(narrow) != 3 || narrow[0] != "AICLI" {
		t.Fatalf("narrow banner = %q", narrow)
	}
	for _, l := range narrow {
		if ansi.StringWidth(l) > w-1 {
			t.Errorf("line %q wider than %d", l, w-1)
		}
	}
	tiny := bannerFor(3)
	if tiny[0] != "AI…" || tiny[1] != "Sc…" {
		t.Errorf("tiny banner = %q", tiny)
	}
	if got := bannerFor(60); got[6] != bannerDescription[0] {
		t.Errorf("description truncated at width 60: %q", got[6])
	}
}
