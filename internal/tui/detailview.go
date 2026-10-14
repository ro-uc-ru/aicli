package tui

import (
	"regexp"
	"strings"

	bkey "github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
)

// footerDetail is the key-hint footer of the detail view (FR-017).
const footerDetail = "esc back · ↑/↓ scroll · q quit"

// flagRe matches a flag token (`-h`, `--help`) at the start of a word.
var flagRe = regexp.MustCompile(`(^|\s)(-{1,2}[A-Za-z][A-Za-z0-9-]*)`)

// detailModel is the Bubble Tea model of the home screen's detail view,
// which shows the help of one command in a scrollable area.
type detailModel struct {
	styles homeStyles
	width  int
	height int
	path   string
	help   string
	lines  []string
	vp     viewport.Model
	closed bool
}

// newDetailModel builds an empty detail view model.
//
// In: styled, whether to add ANSI emphasis; width and height, the terminal
// size in cells.
// Out: the model, showing no help yet. Its viewport accepts only `↑`/`k`,
// `↓`/`j`, `PgUp` and `PgDn`, and ignores the mouse (FR-009, EC-007).
// Side effects: none.
func newDetailModel(styled bool, width, height int) detailModel {
	vp := viewport.New(width, 1)
	vp.MouseWheelEnabled = false
	vp.KeyMap = viewport.KeyMap{
		Up:       bkey.NewBinding(bkey.WithKeys("up", "k")),
		Down:     bkey.NewBinding(bkey.WithKeys("down", "j")),
		PageUp:   bkey.NewBinding(bkey.WithKeys("pgup")),
		PageDown: bkey.NewBinding(bkey.WithKeys("pgdown")),
	}
	m := detailModel{styles: newHomeStyles(styled), width: width, height: height, vp: vp}
	return m.relayout()
}

// open shows the help of one entry, from its first line (EC-011).
//
// In: path, the command path (without the program name); help, the full help
// text.
// Out: the model showing that help scrolled to the top, and the command that
// turns mouse capture off so that the terminal can select text (FR-009).
// Side effects: none.
func (m detailModel) open(path, help string) (detailModel, tea.Cmd) {
	m.path, m.help, m.closed = path, help, false
	m = m.relayout()
	m.vp.GotoTop()
	return m, tea.DisableMouse
}

// wrapSource wraps help to a width by display width, keeping the source
// lines apart.
//
// In: help, the text; width, the terminal width in cells.
// Out: for each source line, its wrapped lines; lines that fit are
// untouched, longer ones are split at the width without dropping any
// character. Side effects: none.
func wrapSource(help string, width int) [][]string {
	help = strings.TrimSuffix(help, "\n")
	var out [][]string
	for _, l := range strings.Split(help, "\n") {
		if width < 1 {
			out = append(out, []string{l})
			continue
		}
		out = append(out, strings.Split(ansi.Hardwrap(l, width, true), "\n"))
	}
	return out
}

// wrapHelp wraps help to a width by display width.
//
// In: help, the text; width, the terminal width in cells.
// Out: the wrapped lines, as flattened from wrapSource. Side effects: none.
func wrapHelp(help string, width int) []string {
	var out []string
	for _, seg := range wrapSource(help, width) {
		out = append(out, seg...)
	}
	return out
}

// firstLine locates the first visible line in the source text.
//
// In: none. Out: the index of its source line and its wrap segment within
// that line, for the current width. Side effects: none.
func (m detailModel) firstLine() (source, segment int) {
	n := m.vp.YOffset
	for i, seg := range wrapSource(m.help, m.width) {
		if n < len(seg) {
			return i, n
		}
		n -= len(seg)
	}
	return 0, 0
}

// resize applies a new terminal size (FR-015).
//
// In: width and height, the new size in cells.
// Out: the model re-wrapped and re-laid out, whose first visible line is
// the same source line and wrap segment as before (the last segment if the
// line now wraps into fewer), moved up only as far as needed so that the
// end of the text is not above the bottom edge.
// Side effects: none.
func (m detailModel) resize(width, height int) detailModel {
	source, segment := m.firstLine()
	m.width, m.height = width, height
	m = m.relayout()
	idx := 0
	for i, seg := range wrapSource(m.help, width) {
		if i == source {
			idx += min(segment, len(seg)-1)
			break
		}
		idx += len(seg)
	}
	m.vp.SetYOffset(idx)
	return m
}

// relayout re-wraps the help and resizes the viewport to the layout.
//
// In: none. Out: the model whose lines, viewport content and viewport height
// match the current size. Side effects: none.
func (m detailModel) relayout() detailModel {
	m.lines = wrapHelp(m.help, m.width)
	m.vp.Width = m.width
	m.vp.Height = computeLayout(m.width, m.height, viewDetail, len(m.lines)).AreaHeight
	m.vp.SetContent(strings.Join(m.lines, "\n"))
	return m
}

// Init is part of the Bubble Tea model contract.
//
// In: none. Out: nil, no initial command. Side effects: none.
func (m detailModel) Init() tea.Cmd { return nil }

// Update handles one message (FR-009, FR-010, EC-007).
//
// In: msg, a key or window-size message; any other message, including
// mouse events, is ignored. A window-size message re-lays out the view
// (FR-015).
// Out: the updated model and a command. `↑`/`k`, `↓`/`j`, `PgUp` and `PgDn`
// scroll, `Home` and `End` jump to the top or the bottom. `Esc`, `←` and
// `Backspace` mark the view closed and return the command that turns mouse
// capture on again. `q` and `Ctrl+C` return tea.Quit. Any other key does
// nothing.
// Side effects: none.
func (m detailModel) Update(msg tea.Msg) (detailModel, tea.Cmd) {
	if size, ok := msg.(tea.WindowSizeMsg); ok {
		return m.resize(size.Width, size.Height), nil
	}
	k, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}
	switch k.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "esc", "left", "backspace":
		m.closed = true
		return m, tea.EnableMouseCellMotion
	case "home":
		m.vp.GotoTop()
	case "end":
		m.vp.GotoBottom()
	case "up", "k", "down", "j", "pgup", "pgdown":
		var cmd tea.Cmd
		m.vp, cmd = m.vp.Update(k)
		return m, cmd
	}
	return m, nil
}

// View renders the detail view (FR-008, FR-017, FR-019, FR-020).
//
// In: none.
// Out: the screen text, without a trailing newline: the header
// `aicli › <path>` truncated to the width and a blank line, the indicator
// line above, the visible help lines, the indicator line below and the
// footer. Parts that do not fit are hidden as in computeLayout. Without
// styling, the help lines are exactly the wrapped help text.
// Side effects: none.
func (m detailModel) View() string {
	l := computeLayout(m.width, m.height, viewDetail, len(m.lines))
	var out []string
	if l.Header {
		out = append(out, m.styles.Header.Render(truncate("aicli › "+m.path, m.width)), "")
	}
	if l.Indicators {
		out = append(out, m.indicator(!m.vp.AtTop(), moreAbove))
	}
	top := m.vp.YOffset
	for _, line := range m.lines[top:min(top+l.AreaHeight, len(m.lines))] {
		out = append(out, m.emphasize(line))
	}
	if l.Indicators {
		out = append(out, m.indicator(!m.vp.AtBottom(), moreBelow))
	}
	if l.Footer {
		out = append(out, m.styles.Footer.Render(truncate(footerDetail, m.width)))
	}
	return strings.Join(out, "\n")
}

// indicator renders one scroll-indicator line.
//
// In: show, whether content is hidden on that side; text, the indicator.
// Out: the styled text, or an empty line. Side effects: none.
func (m detailModel) indicator(show bool, text string) string {
	if !show {
		return ""
	}
	return m.styles.Indicator.Render(truncate(text, m.width))
}

// emphasize adds the heading and flag styles to a help line.
//
// In: line, one wrapped help line.
// Out: the line with ANSI added around a heading (column 0, ending in `:`)
// or around each flag token; the text itself is unchanged. Unstyled models
// return the line as is. Side effects: none.
func (m detailModel) emphasize(line string) string {
	if line == "" || strings.HasPrefix(line, " ") && !flagRe.MatchString(line) {
		return line
	}
	if !strings.HasPrefix(line, " ") && strings.HasSuffix(line, ":") {
		return m.styles.Heading.Render(line)
	}
	return flagRe.ReplaceAllStringFunc(line, func(s string) string {
		flag := strings.TrimLeft(s, " \t")
		return s[:len(s)-len(flag)] + m.styles.Flag.Render(flag)
	})
}
