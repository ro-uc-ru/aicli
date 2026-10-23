package gen

import (
	"bytes"
	"errors"
	"strings"
)

// ErrNoMarker is returned by Inject when the target has no marker line of the
// given name (FR-010).
var ErrNoMarker = errors.New("no marker")

// ErrRepeatedMarker is returned by Inject when the target has more than one
// marker line of the given name (FR-010).
var ErrRepeatedMarker = errors.New("more than one marker")

// line is one line of a file split by splitLines.
type line struct {
	// text is the line without its line ending.
	text string
	// eol is the line ending: "\n", "\r\n", or "" for a last line without one.
	eol string
}

// Inject inserts blocks of lines right after the marker line of a file, as
// FR-009 describes. It is pure: it works on bytes only.
//
// A marker line is one that, trimmed, is exactly "# aicli:<marker>". A block
// is skipped when its trimmed lines already appear as consecutive trimmed
// lines of the file, including blocks inserted earlier in the same call;
// otherwise the whole block is inserted below the marker, after the blocks
// this call already inserted there. Each non-empty inserted line gets the
// marker's leading whitespace before its own text; an empty line stays empty.
// Inserted lines end with the marker's line ending; a marker without one gets
// the first line ending of the file, or "\n" (EC-030). No other byte changes.
//
// In: src, the original file content; marker, the marker name without the
// "# aicli:" prefix; blocks, the rendered blocks in plan order, each one or
// more lines (a single trailing line ending is ignored).
// Out: the new content and true if any block was inserted; src itself and
// false if none was; or ErrNoMarker / ErrRepeatedMarker, with nil and false.
// Side effects: none.
func Inject(src []byte, marker string, blocks []string) ([]byte, bool, error) {
	lines := splitLines(string(src))
	want := "# aicli:" + marker
	at := -1
	for i, l := range lines {
		if strings.TrimSpace(l.text) != want {
			continue
		}
		if at >= 0 {
			return nil, false, ErrRepeatedMarker
		}
		at = i
	}
	if at < 0 {
		return nil, false, ErrNoMarker
	}

	eol := lines[at].eol
	if eol == "" {
		eol = firstEOL(lines)
	}
	text := lines[at].text
	indent := text[:len(text)-len(strings.TrimLeft(text, " \t"))]

	next := at + 1
	changed := false
	for _, b := range blocks {
		blk := splitBlock(b)
		if len(blk) == 0 || contains(lines, blk) {
			continue
		}
		ins := make([]line, len(blk))
		for i, t := range blk {
			if t != "" {
				t = indent + t
			}
			ins[i] = line{text: t, eol: eol}
		}
		lines = append(lines[:next], append(ins, lines[next:]...)...)
		next += len(ins)
		changed = true
	}
	if !changed {
		return src, false, nil
	}
	lines[at].eol = eol

	var out bytes.Buffer
	out.Grow(len(src))
	for _, l := range lines {
		out.WriteString(l.text)
		out.WriteString(l.eol)
	}
	return out.Bytes(), true, nil
}

// splitLines splits s into lines, keeping each line ending apart from its text.
//
// In: s, the file content.
// Out: the lines in order; a final line without an ending has eol "", and an
// empty s, or a trailing line ending, adds no empty last line.
// Side effects: none.
func splitLines(s string) []line {
	var out []line
	for s != "" {
		i := strings.IndexByte(s, '\n')
		if i < 0 {
			out = append(out, line{text: s})
			break
		}
		l := line{text: s[:i], eol: "\n"}
		if strings.HasSuffix(l.text, "\r") {
			l.text, l.eol = l.text[:len(l.text)-1], "\r\n"
		}
		out = append(out, l)
		s = s[i+1:]
	}
	return out
}

// splitBlock splits one rendered block into the texts of its lines.
//
// In: b, the rendered block; one trailing line ending is ignored.
// Out: the line texts without endings, in order; empty for an empty block.
// Side effects: none.
func splitBlock(b string) []string {
	ls := splitLines(b)
	out := make([]string, len(ls))
	for i, l := range ls {
		out[i] = l.text
	}
	return out
}

// firstEOL returns the line ending of the first line that has one.
//
// In: lines, the lines of a file.
// Out: that ending, or "\n" if no line has one.
// Side effects: none.
func firstEOL(lines []line) string {
	for _, l := range lines {
		if l.eol != "" {
			return l.eol
		}
	}
	return "\n"
}

// contains reports whether blk appears in lines as consecutive lines, each
// pair compared after trimming leading and trailing whitespace (FR-009).
//
// In: lines, the current lines of the file; blk, the line texts of a block.
// Out: true if the block is already present.
// Side effects: none.
func contains(lines []line, blk []string) bool {
	for i := 0; i+len(blk) <= len(lines); i++ {
		match := true
		for j, t := range blk {
			if strings.TrimSpace(lines[i+j].text) != strings.TrimSpace(t) {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}
