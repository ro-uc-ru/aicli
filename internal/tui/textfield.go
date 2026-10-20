package tui

import (
	"strings"
	"unicode"

	"github.com/charmbracelet/bubbles/cursor"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// fieldIndent is the indent, in cells, of the description, the input line
// and the error under a field title (FR-011).
const fieldIndent = 2

// textField is a free-text field of the add form: a title, the input
// description, an input line on bubbles/textinput and, after a failed
// validation, an error (FR-011, FR-013).
type textField struct {
	input   FeatureInput
	ti      textinput.Model
	styles  addStyles
	focused bool
	err     string
	width   int
}

// newTextField builds the text field of one input.
//
// In: in, the input (name, description, validator); styled, whether to add
// ANSI colour; width, the terminal width in cells.
// Out: an unfocused field with an empty value and no error.
// Side effects: none.
func newTextField(in FeatureInput, styled bool, width int) textField {
	ti := textinput.New()
	ti.Prompt = ""
	ti.CharLimit = 0
	ti.Cursor.SetMode(cursor.CursorStatic)
	f := textField{input: in, ti: ti, styles: newAddStyles(styled)}
	f.ti.TextStyle = f.styles.Normal
	return f.resize(width)
}

// resize sets the terminal width the field is drawn for.
//
// In: width, the terminal width in cells. Out: the field whose input line
// scrolls horizontally inside width minus the indent, keeping one cell for
// the cursor (EC-012). Side effects: none.
func (f textField) resize(width int) textField {
	f.width = width
	f.ti.Width = max(width-fieldIndent-1, 1)
	return f
}

// Focus gives the field the focus.
//
// In: none. Out: the focused field. Side effects: none.
func (f textField) Focus() textField {
	f.focused = true
	f.ti.Focus()
	return f
}

// Blur takes the focus away from the field.
//
// In: none. Out: the unfocused field. Side effects: none.
func (f textField) Blur() textField {
	f.focused = false
	f.ti.Blur()
	return f
}

// Value returns the text typed so far.
//
// In: none. Out: the full value, untrimmed, never truncated (EC-012).
// Side effects: none.
func (f textField) Value() string { return f.ti.Value() }

// SetValue replaces the text of the field and clears its error.
//
// In: v, the new value. Out: the field with v as value, the cursor at its
// end. Side effects: none.
func (f textField) SetValue(v string) textField {
	f.ti.SetValue(v)
	f.err = ""
	return f
}

// stripControls removes C0 and C1 control characters and DEL.
//
// In: rs, the runes typed or pasted. Out: the runes that are not control
// characters, in order (EC-013). Side effects: none.
func stripControls(rs []rune) []rune {
	out := make([]rune, 0, len(rs))
	for _, r := range rs {
		if !unicode.IsControl(r) {
			out = append(out, r)
		}
	}
	return out
}

// Update handles one key press while the field has the focus (EC-011,
// EC-013).
//
// In: msg, the key. Out: the updated field. Only printable characters (typed
// or pasted, with control characters removed), Backspace, Delete, `←`/`→`,
// `Home` and `End` act; every other key, including Alt combinations, is
// ignored. A paste that holds only control characters changes nothing, and
// no pasted newline ever ends the field.
// Side effects: none.
func (f textField) Update(msg tea.KeyMsg) textField {
	switch msg.Type {
	case tea.KeyRunes, tea.KeySpace:
		if msg.Alt {
			return f
		}
		rs := stripControls(msg.Runes)
		if len(rs) == 0 {
			return f
		}
		msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: rs, Paste: msg.Paste}
	case tea.KeyBackspace, tea.KeyDelete, tea.KeyLeft, tea.KeyRight, tea.KeyHome, tea.KeyEnd:
		if msg.Alt {
			return f
		}
	default:
		return f
	}
	f.ti, _ = f.ti.Update(msg)
	return f
}

// Validate checks the value (FR-013, FR-014).
//
// In: none. Out: the field with its error set, and the message: `<input>
// cannot be empty` if the trimmed value is empty, otherwise the validator's
// message if it returns one, otherwise "" and no error. The validator is
// called only with a non-empty trimmed value. The value itself is not
// changed.
// Side effects: calls the input's validator.
func (f textField) Validate() (textField, string) {
	v := strings.TrimSpace(f.ti.Value())
	switch {
	case v == "":
		f.err = f.input.Name + " cannot be empty"
	case f.input.Validator != nil:
		f.err = f.input.Validator(v)
	default:
		f.err = ""
	}
	return f, f.err
}

// View renders the field.
//
// In: none. Out: its lines: the title (`> ` before it when focused, two
// spaces otherwise), the description, the input line and, if set, the
// error, the last three indented by two spaces. Title, description and
// error are truncated with `…` to the width, never wrapped; the input line
// scrolls horizontally so the cursor is visible. The title is highlighted
// when focused and styled.
// Side effects: none.
func (f textField) View() []string {
	indent := strings.Repeat(" ", fieldIndent)
	title, style := indent+f.input.Name, f.styles.Normal
	if f.focused {
		title, style = "> "+f.input.Name, f.styles.FieldFocused
	}
	lines := []string{
		style.Render(truncate(title, f.width)),
		f.styles.Dim.Render(truncate(indent+f.input.Description, f.width)),
		indent + f.ti.View(),
	}
	if f.err != "" {
		lines = append(lines, f.styles.Error.Render(truncate(indent+f.err, f.width)))
	}
	return lines
}
