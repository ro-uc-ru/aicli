package tui_test

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/exp/teatest"
	"github.com/spf13/cobra"

	"github.com/ro-uc-ru/aicli/internal/home"
	"github.com/ro-uc-ru/aicli/internal/tui"
)

const (
	// termW and termH are the terminal size of the flow tests.
	termW, termH = 80, 24
	// altScreenExit and altScreenEnter are the alternate-screen sequences.
	altScreenExit  = "\x1b[?1049l"
	altScreenEnter = "\x1b[?1049h"
	// mouseOn and mouseOff are the cell-motion mouse reporting sequences.
	mouseOn  = "\x1b[?1002h"
	mouseOff = "\x1b[?1002l"
)

// flowTree builds the test tree: bare (no short), db with migrate, long (a
// help of sixty body lines) and new. Cobra lists them in that order.
//
// In: none. Out: the root command. Side effects: none.
func flowTree() *cobra.Command {
	run := func(*cobra.Command, []string) {}
	root := &cobra.Command{Use: "aicli", Short: "root"}
	root.AddCommand(&cobra.Command{Use: "new", Short: "Create a project", Run: run})
	db := &cobra.Command{Use: "db", Short: "Database tools"}
	db.AddCommand(&cobra.Command{Use: "migrate", Short: "Run migrations", Run: run})
	root.AddCommand(db)
	var long []string
	for i := 1; i <= 60; i++ {
		long = append(long, fmt.Sprintf("body line %02d", i))
	}
	root.AddCommand(&cobra.Command{Use: "long", Short: "Long help", Long: strings.Join(long, "\n"), Run: run})
	root.AddCommand(&cobra.Command{Use: "bare", Run: run})
	return root
}

// key returns the key message of a named key or a single rune.
//
// In: s, a Bubble Tea key name ("down", "esc", "ctrl+c", ...) or one rune.
// Out: the message. Side effects: none.
func key(s string) tea.KeyMsg {
	named := map[string]tea.KeyType{
		"up": tea.KeyUp, "down": tea.KeyDown, "left": tea.KeyLeft, "right": tea.KeyRight,
		"home": tea.KeyHome, "end": tea.KeyEnd, "pgup": tea.KeyPgUp, "pgdown": tea.KeyPgDown,
		"enter": tea.KeyEnter, "esc": tea.KeyEsc, "backspace": tea.KeyBackspace,
		"tab": tea.KeyTab, "ctrl+c": tea.KeyCtrlC,
	}
	if k, ok := named[s]; ok {
		return tea.KeyMsg{Type: k}
	}
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
}

// keys turns key names into messages.
//
// In: names, key names as in key. Out: the messages. Side effects: none.
func keys(names ...string) []tea.Msg {
	var msgs []tea.Msg
	for _, n := range names {
		msgs = append(msgs, key(n))
	}
	return msgs
}

// click returns a left mouse press at a row.
//
// In: y, the screen row. Out: the message. Side effects: none.
func click(y int) tea.Msg {
	return tea.MouseMsg{X: 3, Y: y, Button: tea.MouseButtonLeft, Action: tea.MouseActionPress}
}

// wheel returns a mouse wheel press.
//
// In: b, the wheel button. Out: the message. Side effects: none.
func wheel(b tea.MouseButton) tea.Msg {
	return tea.MouseMsg{X: 3, Y: 12, Button: b, Action: tea.MouseActionPress}
}

// flowResult is what a flow run leaves behind.
type flowResult struct {
	// View is the last screen shown by the model.
	View string
	// Out is everything the program wrote to its output.
	Out string
}

// drive runs the home model in a teatest program, sends msgs in order and
// waits for it to finish.
//
// In: t, the test; entries; styled; w and h, the terminal size; msgs, the
// messages to send, the last of which must quit the program.
// Out: the final screen and the program output (teatest runs without the
// alternate screen and mouse reporting; those are checked through
// RunHome); the test fails if the program does not finish within 5 seconds.
// Side effects: runs a Bubble Tea program on in-memory streams.
func drive(t *testing.T, entries []tui.Entry, styled bool, w, h int, msgs ...tea.Msg) flowResult {
	t.Helper()
	tm := teatest.NewTestModel(t, tui.NewHomeModel(entries, styled, w, h),
		teatest.WithInitialTermSize(w, h))
	for _, m := range msgs {
		tm.Send(m)
		if opensEntry(m) {
			time.Sleep(50 * time.Millisecond) // the open request is a command that runs asynchronously
		}
	}
	final := tm.FinalModel(t, teatest.WithFinalTimeout(5*time.Second))
	out, err := io.ReadAll(tm.FinalOutput(t, teatest.WithFinalTimeout(5*time.Second)))
	if err != nil {
		t.Fatal(err)
	}
	return flowResult{View: final.View(), Out: string(out)}
}

// opensEntry reports whether a message asks the list to open an entry.
//
// In: m, a message. Out: true for Enter or a left click. Side effects: none.
func opensEntry(m tea.Msg) bool {
	switch m := m.(type) {
	case tea.KeyMsg:
		return m.Type == tea.KeyEnter
	case tea.MouseMsg:
		return m.Button == tea.MouseButtonLeft
	}
	return false
}

// view runs msgs and then `q`, returning the screen the msgs led to.
//
// In: t, the test; msgs; w and h, the size. Out: the final screen.
// Side effects: as drive.
func view(t *testing.T, w, h int, msgs ...tea.Msg) string {
	t.Helper()
	return drive(t, home.Entries(flowTree()), false, w, h, append(msgs, key("q"))...).View
}

// pointerLine returns the line marked with the pointer.
//
// In: v, a screen. Out: the line starting with `> `, or "". Side effects: none.
func pointerLine(v string) string {
	for _, l := range strings.Split(v, "\n") {
		if strings.HasPrefix(l, "> ") {
			return l
		}
	}
	return ""
}

// rowOf returns the screen row of the first line containing text.
//
// In: t, the test; v, a screen; text. Out: the row; the test fails if none.
// Side effects: none.
func rowOf(t *testing.T, v, text string) int {
	t.Helper()
	for i, l := range strings.Split(v, "\n") {
		if strings.Contains(l, text) {
			return i
		}
	}
	t.Fatalf("no line with %q in:\n%s", text, v)
	return -1
}

// TestFlowPointerMoves checks the initial pointer and every move key,
// including the stops at both ends.
//
// Covers: FR-005, FR-006
func TestFlowPointerMoves(t *testing.T) {
	cases := []struct {
		name string
		keys []string
		want string
	}{
		{"initial", nil, "> bare"},
		{"down", []string{"down"}, "> db"},
		{"j", []string{"j", "j"}, "> db migrate"},
		{"up stops at first", []string{"up"}, "> bare"},
		{"k", []string{"down", "down", "k"}, "> db  "},
		{"end", []string{"end"}, "> new"},
		{"down stops at last", []string{"end", "down", "j"}, "> new"},
		{"home", []string{"end", "home"}, "> bare"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := pointerLine(view(t, termW, termH, keys(c.keys...)...))
			if !strings.HasPrefix(got, c.want) {
				t.Errorf("pointer line = %q, want prefix %q", got, c.want)
			}
		})
	}
}

// TestFlowOpenByEnterAndClick checks that Enter and a click open the detail
// view with the right header and help, and that a click moves the pointer.
//
// Covers: FR-007, FR-019, EC-011
func TestFlowOpenByEnterAndClick(t *testing.T) {
	v := view(t, termW, termH, keys("down", "down", "enter")...)
	if !strings.Contains(v, "aicli › db migrate") || !strings.Contains(v, "Run migrations") {
		t.Errorf("Enter: wrong detail view:\n%s", v)
	}
	list := view(t, termW, termH)
	y := rowOf(t, list, "db migrate")
	v = view(t, termW, termH, click(y))
	if !strings.Contains(v, "aicli › db migrate") || !strings.Contains(v, "Run migrations") {
		t.Errorf("click: wrong detail view:\n%s", v)
	}
	back := view(t, termW, termH, click(y), key("esc"))
	if !strings.HasPrefix(pointerLine(back), "> db migrate") {
		t.Errorf("click did not move the pointer: %q", pointerLine(back))
	}
	if other := view(t, termW, termH, keys("end", "enter")...); !strings.Contains(other, "aicli › new") ||
		strings.Contains(other, "Run migrations") {
		t.Errorf("opening another entry shows the wrong help:\n%s", other)
	}
}

// helpLine returns the first help line of a detail screen, the one after
// the header, its blank line and the indicator line.
//
// In: v, a detail screen. Out: that line. Side effects: none.
func helpLine(v string) string { return strings.Split(v, "\n")[3] }

// TestFlowDetailScroll checks every scroll key and the `↑ more` and `↓ more`
// indicators appearing and disappearing.
//
// Covers: FR-009, FR-017, FR-019, FR-020, EC-006
func TestFlowDetailScroll(t *testing.T) {
	open := keys("down", "down", "down", "enter") // pointer on long
	top := view(t, termW, termH, open...)
	if !strings.Contains(top, "aicli › long") || !strings.Contains(top, "↓ more") || strings.Contains(top, "↑ more") {
		t.Fatalf("top of a long help needs only ↓ more:\n%s", top)
	}
	if !strings.Contains(top, "esc back") {
		t.Errorf("detail footer missing:\n%s", top)
	}
	for _, k := range []string{"down", "j", "pgdown"} {
		v := view(t, termW, termH, append(open, key(k))...)
		if helpLine(v) == helpLine(top) || !strings.Contains(v, "↑ more") {
			t.Errorf("%s did not scroll down:\n%s", k, v)
		}
		v2 := view(t, termW, termH, append(open, keys(k, k, "up")...)...)
		v3 := view(t, termW, termH, append(open, keys(k, k, "k")...)...)
		v4 := view(t, termW, termH, append(open, keys(k, k, "pgup")...)...)
		if k == "pgdown" {
			if helpLine(v2) != helpLine(top) && !strings.Contains(v4, "↓ more") {
				t.Errorf("pgup did not scroll back")
			}
			continue
		}
		if v2 != v3 {
			t.Errorf("up and k differ after %s", k)
		}
	}
	bottom := view(t, termW, termH, append(open, key("end"))...)
	if strings.Contains(bottom, "↓ more") || !strings.Contains(bottom, "↑ more") || !strings.Contains(bottom, "--help") {
		t.Errorf("End: want ↑ more only and the end of the help:\n%s", bottom)
	}
	if again := view(t, termW, termH, append(open, keys("end", "home")...)...); again != top {
		t.Errorf("Home did not return to the top:\n%s", again)
	}
}

// TestFlowBack checks that Esc, ← and Backspace return to the list with the
// pointer kept.
//
// Covers: FR-010
func TestFlowBack(t *testing.T) {
	for _, k := range []string{"esc", "left", "backspace"} {
		v := view(t, termW, termH, append(keys("down", "down", "enter"), key(k))...)
		if strings.Contains(v, "aicli ›") || !strings.HasPrefix(pointerLine(v), "> db migrate") {
			t.Errorf("%s: want the list with the pointer on db migrate:\n%s", k, v)
		}
	}
}

// syncBuffer is a bytes.Buffer safe for use by the program and the test.
type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

// Write appends p. In: p, the bytes. Out: len(p) and nil. Side effects: locks.
func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

// String returns what has been written. In: none. Out: the text. Side effects: locks.
func (b *syncBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

// step is one input of a RunHome session: bytes to type, and a condition on
// the output so far that must hold before the next step (nil: none).
type step struct {
	send  string
	until func(out string) bool
}

// has returns a condition that holds when the output contains text n times.
//
// In: text; n, the count. Out: the condition. Side effects: none.
func has(text string, n int) func(string) bool {
	return func(out string) bool { return strings.Count(out, text) >= n }
}

// runReal runs tui.RunHome on a pipe and in-memory output, typing the steps
// in order, and returns the whole output once the program has quit.
//
// In: t, the test; entries; ready, a text of the first screen to wait for;
// steps, the inputs, the last of which quits.
// Out: the output of the program; the test fails on a non-nil error from
// RunHome or if a wait or the end takes more than 5 seconds.
// Side effects: runs a Bubble Tea program on a pipe.
func runReal(t *testing.T, entries []tui.Entry, ready string, steps ...step) string {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = r.Close(); _ = w.Close() })
	out := &syncBuffer{}
	done := make(chan error, 1)
	go func() { done <- tui.RunHome(context.Background(), entries, r, out, false) }()
	waitFor := func(cond func(string) bool) {
		deadline := time.Now().Add(5 * time.Second)
		for !cond(out.String()) {
			if time.Now().After(deadline) {
				t.Fatalf("output never reached the awaited state: %q", out.String())
			}
			time.Sleep(10 * time.Millisecond)
		}
	}
	waitFor(has(ready, 1))
	for _, s := range steps {
		if _, err := w.WriteString(s.send); err != nil {
			t.Fatal(err)
		}
		if s.until != nil {
			waitFor(s.until)
		}
	}
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("RunHome: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("RunHome did not return")
	}
	return out.String()
}

// TestFlowQuit checks, through RunHome, that `q` and `Ctrl+C` quit from both
// views and that the output ends by leaving the alternate screen.
//
// Covers: FR-010
func TestFlowQuit(t *testing.T) {
	entries := home.Entries(flowTree())
	cases := map[string][]step{
		"list q":        {{send: "q"}},
		"list ctrl+c":   {{send: "\x03"}},
		"detail q":      {{send: "\r", until: has("aicli ›", 1)}, {send: "q"}},
		"detail ctrl+c": {{send: "\r", until: has("aicli ›", 1)}, {send: "\x03"}},
	}
	for name, steps := range cases {
		t.Run(name, func(t *testing.T) {
			out := runReal(t, entries, "> bare", steps...)
			if !strings.Contains(out, altScreenEnter) || strings.LastIndex(out, altScreenExit) < strings.LastIndex(out, altScreenEnter) {
				t.Errorf("alternate screen not left at the end: %q", out)
			}
		})
	}
}

// TestFlowUnlistedKeys checks that keys outside the lists change nothing.
//
// Covers: EC-007
func TestFlowUnlistedKeys(t *testing.T) {
	junk := keys("x", "tab", "right", "pgdown", "pgup", "a", " ")
	if got, want := view(t, termW, termH, junk...), view(t, termW, termH); got != want {
		t.Errorf("list changed by unlisted keys:\n%s", got)
	}
	open := keys("down", "down", "down", "enter")
	junk = keys("x", "tab", "right", "a", " ", "enter")
	if got, want := view(t, termW, termH, append(open, junk...)...), view(t, termW, termH, open...); got != want {
		t.Errorf("detail changed by unlisted keys:\n%s", got)
	}
}

// TestFlowWheelAndOutsideClicks checks the wheel scrolling a short list
// without moving the pointer, and clicks outside the entries doing nothing.
//
// Covers: FR-011, EC-004
func TestFlowWheelAndOutsideClicks(t *testing.T) {
	const h = 14 // two rows for the list
	initial := view(t, termW, h)
	down := view(t, termW, h, wheel(tea.MouseButtonWheelDown))
	if !strings.Contains(down, "↑ more") || pointerLine(down) != "" {
		t.Errorf("wheel down should scroll the list away from the pointer:\n%s", down)
	}
	if up := view(t, termW, h, wheel(tea.MouseButtonWheelDown), wheel(tea.MouseButtonWheelUp)); up != initial {
		t.Errorf("wheel up should return to the start:\n%s", up)
	}
	if many := view(t, termW, h, wheel(tea.MouseButtonWheelDown), wheel(tea.MouseButtonWheelDown),
		wheel(tea.MouseButtonWheelDown), wheel(tea.MouseButtonWheelDown), wheel(tea.MouseButtonWheelDown),
		wheel(tea.MouseButtonWheelDown)); strings.Contains(many, "↓ more") {
		t.Errorf("wheel must stop at the end:\n%s", many)
	}
	if kb := view(t, termW, h, wheel(tea.MouseButtonWheelDown), key("down")); !strings.Contains(kb, "> db") {
		t.Errorf("a key move must bring the pointer into view:\n%s", kb)
	}
	for _, y := range []int{0, 5, 9, termH - 1} {
		if got := view(t, termW, termH, click(y)); got != view(t, termW, termH) {
			t.Errorf("click at row %d changed the screen:\n%s", y, got)
		}
	}
}

// TestFlowMouseModes checks through RunHome's output that mouse reporting is
// turned off in the detail view and on again in the list.
//
// Covers: FR-009
func TestFlowMouseModes(t *testing.T) {
	entries := home.Entries(flowTree())
	list := runReal(t, entries, "> bare", step{send: "q"})
	if n := strings.Count(list, mouseOn); n != 1 {
		t.Fatalf("list only: want one mouse enable, got %d in %q", n, list)
	}
	detail := runReal(t, entries, "> bare", step{send: "\r", until: has("aicli ›", 1)}, step{send: "q"})
	if n := strings.Count(detail, mouseOn); n != 1 {
		t.Errorf("detail: want no second enable, got %d in %q", n, detail)
	}
	if strings.Index(detail, mouseOff) > strings.Index(detail, "aicli ›") {
		t.Errorf("detail: mouse still on when the detail view was drawn: %q", detail)
	}
	back := runReal(t, entries, "> bare", step{send: "\r", until: has("aicli ›", 1)}, step{send: "\x1b[D", until: has(mouseOn, 2)}, step{send: "q"})
	first := strings.Index(back, mouseOn)
	off := strings.Index(back, mouseOff)
	last := strings.LastIndex(back, mouseOn)
	if first >= off || off >= last {
		t.Errorf("back: want enable, disable, enable in order, got %q", back)
	}
}

// TestFlowResize checks resizing in both views.
//
// Covers: FR-015
func TestFlowResize(t *testing.T) {
	size := func(w, h int) tea.Msg { return tea.WindowSizeMsg{Width: w, Height: h} }
	v := view(t, termW, termH, key("end"), size(termW, 14))
	if !strings.HasPrefix(pointerLine(v), "> new") {
		t.Errorf("list resize lost the pointer:\n%s", v)
	}
	open := keys("down", "down", "down", "enter", "j", "j", "j", "j", "j")
	before := helpLine(view(t, termW, termH, open...))
	if after := helpLine(view(t, termW, termH, append(open, size(100, termH))...)); after != before {
		t.Errorf("detail resize moved the first line: %q -> %q", before, after)
	}
	bottom := view(t, termW, 10, keys("down", "down", "down", "enter", "end")...)
	grown := view(t, termW, 10, append(keys("down", "down", "down", "enter", "end"), size(termW, 40))...)
	if !strings.Contains(bottom, "--help") || !strings.Contains(grown, "--help") || strings.Contains(grown, "↓ more") {
		t.Errorf("growing the terminal must keep the end of the help at the bottom:\n%s", grown)
	}
}

// TestFlowReopenStartsAtTop checks that reopening an entry starts at the top.
//
// Covers: EC-011
func TestFlowReopenStartsAtTop(t *testing.T) {
	open := keys("down", "down", "down", "enter")
	top := view(t, termW, termH, open...)
	again := view(t, termW, termH, append(open, keys("j", "j", "j", "end", "esc", "enter")...)...)
	if again != top {
		t.Errorf("reopened entry did not start at the top:\n%s", again)
	}
}

// TestFlowEmptyList checks the empty list: the message, the footer, and only
// `q` and `Ctrl+C` having effect.
//
// Covers: EC-001
func TestFlowEmptyList(t *testing.T) {
	base := drive(t, nil, false, termW, termH, key("q")).View
	if !strings.Contains(base, "No commands available.") || pointerLine(base) != "" {
		t.Fatalf("empty list screen:\n%s", base)
	}
	if !strings.Contains(base, "q quit") || strings.Contains(base, "enter open") {
		t.Errorf("empty footer should show only q quit:\n%s", base)
	}
	msgs := append(keys("enter", "down", "j", "end", "x"), click(rowOf(t, base, "No commands")), click(10))
	if got := drive(t, nil, false, termW, termH, append(msgs, key("q"))...).View; got != base {
		t.Errorf("empty list changed:\n%s", got)
	}
	if out := runReal(t, nil, "No commands available.", step{send: "\x03"}); !strings.Contains(out, altScreenExit) {
		t.Errorf("Ctrl+C did not quit the empty list")
	}
}

// TestFlowShortTerminal checks the EC-003 hiding order in both views.
//
// Covers: EC-003, FR-020
func TestFlowShortTerminal(t *testing.T) {
	banner := "AICLI"
	if v := view(t, termW, 30); !strings.Contains(v, "██") && !strings.Contains(v, banner) {
		t.Fatalf("a tall terminal shows the banner:\n%s", v)
	}
	l12 := view(t, termW, 12)
	if strings.Contains(l12, "Rails-style") || strings.Contains(l12, "██") || !strings.Contains(l12, "q quit") ||
		!strings.HasPrefix(pointerLine(l12), "> bare") {
		t.Errorf("height 12: banner goes first, the rest stays:\n%s", l12)
	}
	l2 := view(t, termW, 2)
	if lines := strings.Split(l2, "\n"); len(lines) != 2 || !strings.HasPrefix(lines[0], "> bare") ||
		!strings.Contains(lines[1], "q quit") {
		t.Errorf("height 2: one entry and the footer:\n%s", l2)
	}
	l1 := view(t, termW, 1)
	if strings.Contains(l1, "\n") || !strings.HasPrefix(l1, "> bare") {
		t.Errorf("height 1: only one entry:\n%s", l1)
	}
	open := keys("down", "down", "down", "enter")
	d3 := view(t, termW, 3, open...)
	if !strings.Contains(d3, "aicli › long") || strings.Contains(d3, "esc back") || strings.Contains(d3, "more") {
		t.Errorf("detail height 3: header and one line, no footer or indicators:\n%s", d3)
	}
	d1 := view(t, termW, 1, open...)
	if strings.Contains(d1, "\n") || strings.Contains(d1, "aicli ›") || d1 == "" {
		t.Errorf("detail height 1: one line of help only:\n%s", d1)
	}
}
