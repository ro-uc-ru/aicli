package tui_test

import (
	"regexp"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/ro-uc-ru/aicli/internal/home"
)

// sgrRE matches ANSI CSI escape sequences.
var sgrRE = regexp.MustCompile(`\x1b\[[0-9;]*[A-Za-z]`)

// styledView runs msgs and then `q` with styling on or off.
//
// In: t, the test; styled; msgs, the messages to send before quitting.
// Out: the final screen. Side effects: as drive.
func styledView(t *testing.T, styled bool, msgs ...tea.Msg) string {
	t.Helper()
	return drive(t, home.Entries(flowTree()), styled, termW, termH, append(msgs, key("q"))...).View
}

// entrySigs returns, for each listed entry path, the ANSI sequences of its row
// joined together, i.e. the style the row is drawn with.
//
// In: t, the test; v, a styled list screen. Out: the signature by path; the
// test fails if an entry row is missing. Side effects: none.
func entrySigs(t *testing.T, v string) map[string]string {
	t.Helper()
	sigs := map[string]string{}
	for _, l := range strings.Split(v, "\n") {
		plain := sgrRE.ReplaceAllString(l, "")
		for _, p := range []string{"bare", "db migrate", "db", "long", "new"} {
			if strings.HasPrefix(plain, "> "+p) || strings.HasPrefix(plain, "  "+p) {
				sigs[p] = strings.Join(sgrRE.FindAllString(l, -1), "")
				break
			}
		}
	}
	if len(sigs) != 5 {
		t.Fatalf("expected 5 entry rows, got %v in:\n%s", sigs, v)
	}
	return sigs
}

// TestFlowStyledSelection checks that, with styling on, the selected row has
// a different style from the others and that the style follows the pointer
// by key and by click.
//
// Covers: FR-013
func TestFlowStyledSelection(t *testing.T) {
	first := entrySigs(t, styledView(t, true))
	for p, s := range first {
		if p != "bare" && s == first["bare"] {
			t.Errorf("selected row %q has the same style as %q", "bare", p)
		}
	}
	last := entrySigs(t, styledView(t, true, key("end")))
	if last["new"] == first["new"] || last["new"] != first["bare"] {
		t.Errorf("highlight did not move to the last row by key")
	}
	if last["bare"] != first["long"] && last["bare"] != first["new"] {
		t.Errorf("previous row did not return to the normal style: %q", last["bare"])
	}
	down := entrySigs(t, styledView(t, true, key("down")))
	if down["db"] != first["bare"] || down["bare"] == first["bare"] || down["bare"] != last["bare"] {
		t.Errorf("highlight did not follow the pointer down")
	}
	y := rowOf(t, styledView(t, true), "Run migrations")
	clicked := entrySigs(t, styledView(t, true, click(y), key("esc")))
	if clicked["db migrate"] != first["bare"] || clicked["bare"] != last["bare"] {
		t.Errorf("highlight did not follow the click")
	}
}

// TestFlowUnstyled checks that, with styling off, there is no ANSI sequence
// and `> ` marks exactly the selected row, and that styling only adds ANSI
// to the text of the list and of the detail view.
//
// Covers: FR-013
func TestFlowUnstyled(t *testing.T) {
	scenes := map[string][]tea.Msg{
		"list":        nil,
		"moved":       keys("down", "down"),
		"detail":      keys("down", "down", "down", "enter"),
		"detail-leaf": keys("end", "enter"),
	}
	for name, msgs := range scenes {
		t.Run(name, func(t *testing.T) {
			plain := styledView(t, false, msgs...)
			if strings.Contains(plain, "\x1b") {
				t.Errorf("unstyled screen has escape sequences:\n%q", plain)
			}
			if name == "list" || name == "moved" {
				n := 0
				for _, l := range strings.Split(plain, "\n") {
					if strings.HasPrefix(l, "> ") {
						n++
					}
				}
				if n != 1 {
					t.Errorf("want exactly one `> ` row, got %d:\n%s", n, plain)
				}
			}
			styled := styledView(t, true, msgs...)
			if !strings.Contains(styled, "\x1b[") {
				t.Errorf("styled screen has no ANSI sequence")
			}
			if got := sgrRE.ReplaceAllString(styled, ""); got != plain {
				t.Errorf("text changed by styling:\nstyled:\n%s\nplain:\n%s", got, plain)
			}
		})
	}
}
