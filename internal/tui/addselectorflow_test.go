package tui_test

import (
	"context"
	"io"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/exp/teatest"

	"github.com/ro-uc-ru/aicli/internal/add"
	"github.com/ro-uc-ru/aicli/internal/featuretest"
	"github.com/ro-uc-ru/aicli/internal/tui"
)

// escSeq matches the terminal control sequences the program emits.
var escSeq = regexp.MustCompile("\x1b\\[[0-9;?]*[a-zA-Z]")

// listedFeatures returns the listed features of the T-011 test tree, in
// order: Main, NoInputs, NoShort, Wide.
//
// In: none. Out: the features. Side effects: none.
func listedFeatures() []tui.Feature {
	var listed []tui.Feature
	for _, f := range add.Features(featuretest.NewTree(featuretest.Valid()...)) {
		if f.Listed {
			listed = append(listed, f)
		}
	}
	return listed
}

// driveAdd runs the add model unstyled in a teatest program, sends msgs in
// order and waits for it to finish.
//
// In: t, the test; w and h, the terminal size; msgs, the messages to send,
// the last of which must quit the program. Out: the final screen and the
// final result; the test fails if the program does not finish in 5 seconds.
// Side effects: runs a Bubble Tea program on in-memory streams.
func driveAdd(t *testing.T, w, h int, msgs ...tea.Msg) (string, tui.AddResult) {
	t.Helper()
	return driveAddWith(t, listedFeatures(), w, h, msgs...)
}

// driveAddWith is driveAdd with the given features.
//
// In: t, the test; features; w and h, the size; msgs. Out: the final screen
// and result. Side effects: as driveAdd.
func driveAddWith(t *testing.T, features []tui.Feature, w, h int, msgs ...tea.Msg) (string, tui.AddResult) {
	t.Helper()
	return driveAddStyled(t, features, false, w, h, msgs...)
}

// driveAddStyled is driveAddWith with styling on or off.
//
// In: t, the test; features; styled; w and h, the size; msgs. Out: the final
// screen and result. Side effects: as driveAdd.
func driveAddStyled(t *testing.T, features []tui.Feature, styled bool, w, h int, msgs ...tea.Msg) (string, tui.AddResult) {
	t.Helper()
	tm := teatest.NewTestModel(t, tui.NewAddModel(features, styled, w, h), teatest.WithInitialTermSize(w, h))
	for _, m := range msgs {
		tm.Send(m)
		if k, ok := m.(tea.KeyMsg); opensEntry(m) || ok && k.Type == tea.KeyEsc {
			time.Sleep(50 * time.Millisecond) // Enter and Esc ask for a screen change through a command
		}
	}
	final := tm.FinalModel(t, teatest.WithFinalTimeout(5*time.Second)).(tui.AddModel)
	if _, err := io.ReadAll(tm.FinalOutput(t, teatest.WithFinalTimeout(5*time.Second))); err != nil {
		t.Fatal(err)
	}
	return final.View(), final.Result()
}

// selView runs msgs and then `q` and returns the last screen.
//
// In: t, the test; w and h, the size; msgs. Out: the screen. Side effects: as
// driveAdd.
func selView(t *testing.T, w, h int, msgs ...tea.Msg) string {
	t.Helper()
	v, _ := driveAdd(t, w, h, append(msgs, key("q"))...)
	return v
}

// TestAddSelectorScreen checks the header, the description, the entries in
// Cobra order with the pointer on the first, the footer, and the absence of
// the banner and of unlisted features.
//
// Covers: FR-008, EC-006
func TestAddSelectorScreen(t *testing.T) {
	v := selView(t, termW, termH)
	lines := strings.Split(v, "\n")
	if lines[0] != "aicli › add" || lines[1] != "Add an AI/LLM feature to this project." {
		t.Errorf("wrong header:\n%s", v)
	}
	order := []string{featuretest.Main, featuretest.NoInputs, featuretest.NoShort, featuretest.Wide}
	prev := -1
	for _, n := range order {
		row := rowOf(t, v, n)
		if row <= prev {
			t.Errorf("%s is out of order:\n%s", n, v)
		}
		prev = row
	}
	if !strings.HasPrefix(pointerLine(v), "> "+featuretest.Main) {
		t.Errorf("pointer not on the first entry:\n%s", v)
	}
	if !strings.Contains(v, "enter select") || !strings.Contains(v, "q quit") {
		t.Errorf("footer missing:\n%s", v)
	}
	for _, bad := range []string{"AICLI", "██", featuretest.Hidden, featuretest.Deprecated} {
		if strings.Contains(v, bad) {
			t.Errorf("screen contains %q:\n%s", bad, v)
		}
	}
}

// TestAddSelectorMoves checks the moves by keys, the wheel with the scroll
// indicators, and the resize.
//
// Covers: FR-009, EC-006
func TestAddSelectorMoves(t *testing.T) {
	cases := []struct {
		name string
		keys []string
		want string
	}{
		{"down", []string{"down"}, "> " + featuretest.NoInputs},
		{"j", []string{"j", "j"}, "> " + featuretest.NoShort},
		{"up stops at first", []string{"up"}, "> " + featuretest.Main},
		{"k", []string{"down", "down", "k"}, "> " + featuretest.NoInputs},
		{"end", []string{"end"}, "> " + featuretest.Wide},
		{"down stops at last", []string{"end", "down", "j"}, "> " + featuretest.Wide},
		{"home", []string{"end", "home"}, "> " + featuretest.Main},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := pointerLine(selView(t, termW, termH, keys(c.keys...)...)); !strings.HasPrefix(got, c.want) {
				t.Errorf("pointer line = %q, want prefix %q", got, c.want)
			}
		})
	}

	const h = 8 // not every entry fits
	initial := selView(t, termW, h)
	if !strings.Contains(initial, "↓ more") || strings.Contains(initial, "↑ more") {
		t.Fatalf("a short terminal needs only ↓ more:\n%s", initial)
	}
	down := selView(t, termW, h, wheel(tea.MouseButtonWheelDown))
	if !strings.Contains(down, "↑ more") || pointerLine(down) != "" {
		t.Errorf("wheel down should scroll away from the pointer:\n%s", down)
	}
	if up := selView(t, termW, h, wheel(tea.MouseButtonWheelDown), wheel(tea.MouseButtonWheelUp)); up != initial {
		t.Errorf("wheel up should return to the start:\n%s", up)
	}
	end := selView(t, termW, h, key("end"))
	if strings.Contains(end, "↓ more") || !strings.Contains(end, "↑ more") {
		t.Errorf("End: want only ↑ more:\n%s", end)
	}

	size := func(w, h int) tea.Msg { return tea.WindowSizeMsg{Width: w, Height: h} }
	grown := selView(t, termW, h, key("end"), size(termW, termH))
	if strings.Contains(grown, "more") || !strings.HasPrefix(pointerLine(grown), "> "+featuretest.Wide) {
		t.Errorf("resize lost the pointer or kept indicators:\n%s", grown)
	}
}

// openView runs msgs and then `Ctrl+C`, which quits from the form and the
// confirmation (`q` would be typed into a text field).
//
// In: t, the test; msgs. Out: the last screen. Side effects: as driveAdd.
func openView(t *testing.T, msgs ...tea.Msg) string {
	t.Helper()
	v, r := driveAdd(t, termW, termH, append(msgs, key("ctrl+c"))...)
	if r.Action != tui.ActionQuit {
		t.Errorf("result = %+v, want quit", r)
	}
	return v
}

// TestAddSelectorOpens checks that Enter and a click open the form of the
// entry, or the confirmation if it has no inputs, and that a click moves the
// pointer.
//
// Covers: FR-010
func TestAddSelectorOpens(t *testing.T) {
	form := "aicli › add › " + featuretest.Main
	if v := openView(t, key("enter")); !strings.Contains(v, form) || !strings.Contains(v, "input-1") {
		t.Errorf("Enter: want the form of %s:\n%s", featuretest.Main, v)
	}
	y := rowOf(t, selView(t, termW, termH), featuretest.Main)
	if v := openView(t, click(y)); !strings.Contains(v, form) {
		t.Errorf("click: want the form of %s:\n%s", featuretest.Main, v)
	}
	y = rowOf(t, selView(t, termW, termH), featuretest.NoInputs)
	v := openView(t, click(y))
	if !strings.Contains(v, "aicli › add › "+featuretest.NoInputs) || strings.Contains(v, "input-1") {
		t.Errorf("click on a feature with no inputs: want its confirmation:\n%s", v)
	}
	if v := openView(t, keys("down", "enter")...); !strings.Contains(v, "aicli › add › "+featuretest.NoInputs) {
		t.Errorf("Enter on the second entry:\n%s", v)
	}
}

// TestAddSelectorQuitKeys checks, through the real program, that `q`, `Esc`
// and `Ctrl+C` leave the alternate screen, release the mouse, print nothing
// afterwards and return the quit action.
//
// Covers: FR-010
func TestAddSelectorQuitKeys(t *testing.T) {
	for name, send := range map[string]string{"q": "q", "esc": "\x1b", "ctrl+c": "\x03"} {
		t.Run(name, func(t *testing.T) {
			r, w, err := os.Pipe()
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = r.Close(); _ = w.Close() })
			out := &syncBuffer{}
			type res struct {
				r   tui.AddResult
				err error
			}
			done := make(chan res, 1)
			go func() {
				ar, err := tui.RunAdd(context.Background(), listedFeatures(), r, out, false)
				done <- res{ar, err}
			}()
			deadline := time.Now().Add(5 * time.Second)
			for !strings.Contains(out.String(), "> "+featuretest.Main) {
				if time.Now().After(deadline) {
					t.Fatalf("selector never shown: %q", out.String())
				}
				time.Sleep(10 * time.Millisecond)
			}
			if _, err := w.WriteString(send); err != nil {
				t.Fatal(err)
			}
			select {
			case got := <-done:
				if got.err != nil || got.r.Action != tui.ActionQuit {
					t.Fatalf("result = %+v, %v; want quit", got.r, got.err)
				}
			case <-time.After(5 * time.Second):
				t.Fatal("RunAdd did not return")
			}
			s := out.String()
			if !strings.Contains(s, altScreenEnter) || !strings.Contains(s, mouseOn) {
				t.Errorf("alternate screen or mouse not enabled: %q", s)
			}
			exit := strings.LastIndex(s, altScreenExit)
			if exit < strings.LastIndex(s, altScreenEnter) {
				t.Fatalf("alternate screen not left: %q", s)
			}
			if tail := escSeq.ReplaceAllString(s[exit+len(altScreenExit):], ""); strings.TrimSpace(tail) != "" {
				t.Errorf("text printed after leaving the alternate screen: %q", tail)
			}
			if !strings.Contains(s, mouseOff) {
				t.Errorf("mouse not released: %q", s)
			}
		})
	}
}

// TestAddSelectorQuitModel checks that the three quit keys end the model
// with the quit action and no feature chosen.
//
// Covers: FR-010
func TestAddSelectorQuitModel(t *testing.T) {
	for _, k := range []string{"q", "esc", "ctrl+c"} {
		if _, r := driveAdd(t, termW, termH, key(k)); r.Action != tui.ActionQuit || r.Values != nil {
			t.Errorf("%s: result = %+v, want quit", k, r)
		}
	}
}

// TestAddSelectorUnlisted checks that unlisted keys and clicks outside the
// entries change nothing.
//
// Covers: EC-011
func TestAddSelectorUnlisted(t *testing.T) {
	base := selView(t, termW, termH)
	junk := keys("x", "tab", "right", "left", "pgdown", "pgup", "a", " ", "backspace")
	if got := selView(t, termW, termH, junk...); got != base {
		t.Errorf("unlisted keys changed the selector:\n%s", got)
	}
	for _, y := range []int{0, 1, 2, 3, termH - 1} {
		if got := selView(t, termW, termH, click(y)); got != base {
			t.Errorf("click at row %d changed the selector:\n%s", y, got)
		}
	}
	right := tea.MouseMsg{X: 3, Y: rowOf(t, base, featuretest.Main), Button: tea.MouseButtonRight, Action: tea.MouseActionPress}
	if got := selView(t, termW, termH, right); got != base {
		t.Errorf("right click changed the selector:\n%s", got)
	}
}
