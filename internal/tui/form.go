package tui

import (
	"slices"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

const (
	// footerFormNext and footerFormLast are the key hints of a field that
	// is not the last and of the last field; the choice variants add
	// `↑/↓ choose` (FR-011).
	footerFormNext = "tab/enter next · shift+tab back · esc back · ctrl+c quit"
	footerFormLast = "enter continue · shift+tab back · esc back · ctrl+c quit"
	// hintChoose is the key hint a choice field adds to its footer.
	hintChoose = "↑/↓ choose · "
)

// formDoneMsg asks the program to show the confirmation of the feature
// (FR-015, FR-018).
type formDoneMsg struct {
	// Values holds one value per input, in order: text values trimmed,
	// choice values as selected.
	Values []string
}

// formBackMsg asks the program to return to the selector, keeping the
// values for the next time the form of this feature opens (FR-017).
type formBackMsg struct {
	// Values holds one value per input, in order, as typed.
	Values []string
}

// formField is one field of the form: a text field or a choice field.
type formField struct {
	isChoice bool
	text     textField
	choice   choiceField
}

// focus gives the field the focus.
//
// In: none. Out: the focused field. Side effects: none.
func (f formField) focus() formField {
	if f.isChoice {
		f.choice = f.choice.Focus()
	} else {
		f.text = f.text.Focus()
	}
	return f
}

// blur takes the focus away from the field.
//
// In: none. Out: the unfocused field. Side effects: none.
func (f formField) blur() formField {
	if f.isChoice {
		f.choice = f.choice.Blur()
	} else {
		f.text = f.text.Blur()
	}
	return f
}

// value returns the value as typed or selected.
//
// In: none. Out: the raw value. Side effects: none.
func (f formField) value() string {
	if f.isChoice {
		return f.choice.Value()
	}
	return f.text.Value()
}

// view renders the field.
//
// In: space, the rows available to it. Out: its lines. Side effects: none.
func (f formField) view(space int) []string {
	if f.isChoice {
		return f.choice.View(space)
	}
	return f.text.View()
}

// height returns the rows the field needs.
//
// In: none. Out: the height of its lines when drawn in full. Side effects:
// none.
func (f formField) height() int {
	if f.isChoice {
		return f.choice.Height()
	}
	return len(f.text.View())
}

// formModel is the model of the form screen of one feature (FR-011).
type formModel struct {
	feature Feature
	fields  []formField
	focus   int
	styles  addStyles
	width   int
	height  int
}

// newFormModel builds the form of a feature.
//
// In: feature, the feature, whose Inputs are not empty; values, the values
// kept from an earlier visit (one per input) or nil; styled, whether to add
// ANSI colour; width and height, the initial terminal size in cells.
// Out: the model with one field per input in declaration order, filled from
// values (empty, or the first choice, where none is given) and the focus on
// the first field (FR-017).
// Side effects: none.
func newFormModel(feature Feature, values []string, styled bool, width, height int) formModel {
	m := formModel{feature: feature, styles: newAddStyles(styled), width: width, height: height}
	for i, in := range feature.Inputs {
		f := formField{isChoice: len(in.Choices) > 0}
		if f.isChoice {
			f.choice = newChoiceField(in, styled, width)
		} else {
			f.text = newTextField(in, styled, width)
		}
		if i < len(values) {
			if f.isChoice {
				f.choice = f.choice.SetValue(values[i])
			} else {
				f.text = f.text.SetValue(values[i])
			}
		}
		m.fields = append(m.fields, f)
	}
	if len(m.fields) > 0 {
		m.fields[0] = m.fields[0].focus()
	}
	return m
}

// Init is part of the Bubble Tea model contract.
//
// In: none. Out: nil, no initial command. Side effects: none.
func (m formModel) Init() tea.Cmd { return nil }

// layout returns the current screen layout of the form.
//
// In: none. Out: the layout for the size, the description and the field
// heights. Side effects: none.
func (m formModel) layout() formLayout {
	heights := make([]int, len(m.fields))
	for i, f := range m.fields {
		heights[i] = f.height()
	}
	return computeFormLayout(m.height, m.feature.Short != "", heights, m.focus)
}

// values returns the field values.
//
// In: trim, whether to trim the text values. Out: one value per field.
// Side effects: none.
func (m formModel) values(trim bool) []string {
	out := make([]string, len(m.fields))
	for i, f := range m.fields {
		out[i] = f.value()
		if trim && !f.isChoice {
			out[i] = strings.TrimSpace(out[i])
		}
	}
	return out
}

// validate checks one field.
//
// In: i, the field index. Out: the model with the field's error set, and
// whether the field is valid. Choice fields are always valid. Side effects:
// calls the input's validator.
func (m formModel) validate(i int) (formModel, bool) {
	if m.fields[i].isChoice {
		return m, true
	}
	var msg string
	m.fields[i].text, msg = m.fields[i].text.Validate()
	return m, msg == ""
}

// moveFocus moves the focus to another field.
//
// In: i, the index of the field to focus, in range. Out: the model with the
// focus on it. Side effects: none.
func (m formModel) moveFocus(i int) formModel {
	m.fields[m.focus] = m.fields[m.focus].blur()
	m.focus = i
	m.fields[i] = m.fields[i].focus()
	return m
}

// Update handles one message (FR-013, FR-015, FR-017, EC-011, EC-014).
//
// In: msg, a key, mouse or window-size message; any other is ignored.
// Out: the updated model and a command. `Tab` and `Enter` validate the
// focused field and move to the next one, or stay on it with its error if
// it is invalid; `Shift+Tab` moves back without validating; `Enter` on the
// last field validates every field and either produces formDoneMsg or moves
// the focus to the first invalid field; `Esc` produces formBackMsg; `Ctrl+C`
// quits and produces no formDoneMsg. A left click on a field moves the
// focus to it without validating, and on an option of a choice field also
// selects it; clicks elsewhere, the wheel and other keys do nothing.
// Side effects: calls validators.
func (m formModel) Update(msg tea.Msg) (formModel, tea.Cmd) {
	m.fields = slices.Clone(m.fields)
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		for i := range m.fields {
			if m.fields[i].isChoice {
				m.fields[i].choice = m.fields[i].choice.resize(msg.Width)
			} else {
				m.fields[i].text = m.fields[i].text.resize(msg.Width)
			}
		}
	case tea.KeyMsg:
		return m.updateKey(msg)
	case tea.MouseMsg:
		return m.updateMouse(msg), nil
	}
	return m, nil
}

// updateKey handles a key press in the form.
//
// In: msg, the key. Out: the updated model and command, as for Update.
// Side effects: calls validators.
func (m formModel) updateKey(msg tea.KeyMsg) (formModel, tea.Cmd) {
	last := len(m.fields) - 1
	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit
	case "esc":
		v := m.values(false)
		return m, func() tea.Msg { return formBackMsg{Values: v} }
	case "shift+tab":
		if m.focus > 0 {
			m = m.moveFocus(m.focus - 1)
		}
		return m, nil
	case "tab", "enter":
		if m.focus < last {
			var ok bool
			if m, ok = m.validate(m.focus); ok {
				m = m.moveFocus(m.focus + 1)
			}
			return m, nil
		}
		if msg.String() == "tab" {
			return m, nil
		}
		return m.finish()
	}
	if m.focus < 0 || m.focus > last {
		return m, nil
	}
	if f := &m.fields[m.focus]; f.isChoice {
		f.choice = f.choice.Update(msg)
	} else {
		f.text = f.text.Update(msg)
	}
	return m, nil
}

// finish validates every field after `Enter` on the last one.
//
// In: none. Out: the model; with all fields valid, a command producing
// formDoneMsg; otherwise the focus on the first invalid field and no
// command. Side effects: calls validators.
func (m formModel) finish() (formModel, tea.Cmd) {
	first := -1
	for i := range m.fields {
		var ok bool
		if m, ok = m.validate(i); !ok && first < 0 {
			first = i
		}
	}
	if first >= 0 {
		return m.moveFocus(first), nil
	}
	v := m.values(true)
	return m, func() tea.Msg { return formDoneMsg{Values: v} }
}

// updateMouse handles a mouse event in the form.
//
// In: msg, the event. Out: the updated model. Only a left press on a field
// acts (FR-015, EC-011). Side effects: none.
func (m formModel) updateMouse(msg tea.MouseMsg) formModel {
	if msg.Action != tea.MouseActionPress || msg.Button != tea.MouseButtonLeft {
		return m
	}
	l := m.layout()
	if msg.Y < l.AreaTop || msg.Y >= l.AreaTop+l.AreaHeight {
		return m
	}
	row := l.AreaTop
	for i := l.First; i < l.Last; i++ {
		h := len(m.fields[i].view(l.AreaHeight))
		if msg.Y < row+h {
			if i != m.focus {
				m = m.moveFocus(i)
			}
			if f := &m.fields[i]; f.isChoice {
				f.choice = f.choice.Click(msg.Y-row, l.AreaHeight)
			}
			return m
		}
		row += h + 1
		if msg.Y < row {
			return m
		}
	}
	return m
}

// footer returns the key hints for the focused field.
//
// In: none. Out: the footer text (FR-011). Side effects: none.
func (m formModel) footer() string {
	base := footerFormNext
	if m.focus == len(m.fields)-1 {
		base = footerFormLast
	}
	if m.focus < len(m.fields) && m.fields[m.focus].isChoice {
		i := strings.Index(base, "shift+tab")
		base = base[:i] + hintChoose + base[i:]
	}
	return base
}

// View renders the form (FR-011, FR-012, EC-008, EC-010).
//
// In: none.
// Out: the screen text, without a trailing newline: the header
// `aicli › add › <feature>`, the feature description (none, and no blank
// line, when empty), the indicator line above, the visible fields one blank
// line apart, the indicator line below and the footer, each part only if the
// layout shows it. Lines are truncated with `…` to the width, never wrapped.
// Side effects: none.
func (m formModel) View() string {
	l := m.layout()
	var lines []string
	if l.Header {
		lines = append(lines, m.styles.Header.Render(truncate("aicli › add › "+m.feature.Name, m.width)))
	}
	if l.Description {
		lines = append(lines, m.styles.Dim.Render(truncate(m.feature.Short, m.width)))
	}
	above, below := l.More(len(m.fields))
	if l.Indicators {
		lines = append(lines, m.indicator(above, moreAbove))
	}
	for i := l.First; i < l.Last; i++ {
		if i > l.First {
			lines = append(lines, "")
		}
		lines = append(lines, m.fields[i].view(l.AreaHeight)...)
	}
	if l.Indicators {
		lines = append(lines, m.indicator(below, moreBelow))
	}
	if l.Footer {
		lines = append(lines, m.styles.Dim.Render(truncate(m.footer(), m.width)))
	}
	return strings.Join(lines, "\n")
}

// indicator renders one scroll-indicator line.
//
// In: show, whether fields are hidden on that side; text, the indicator.
// Out: the styled text, or an empty line. Side effects: none.
func (m formModel) indicator(show bool, text string) string {
	if !show {
		return ""
	}
	return m.styles.Dim.Render(truncate(text, m.width))
}
