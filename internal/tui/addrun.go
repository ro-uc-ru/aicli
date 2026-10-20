package tui

import (
	"context"
	"fmt"
	"io"

	tea "github.com/charmbracelet/bubbletea"
)

// AddAction is how the add screens ended.
type AddAction string

const (
	// ActionQuit means the user left with `q`, `Esc` or `Ctrl+C`: nothing is
	// added and nothing is printed (FR-010).
	ActionQuit AddAction = "quit"
	// ActionCancel means the user pressed `Cancel` in the confirmation.
	ActionCancel AddAction = "cancel"
	// ActionAdd means the user pressed `Add` in the confirmation.
	ActionAdd AddAction = "add"
)

// AddResult is the outcome of the add screens.
type AddResult struct {
	// Action is how the screens ended.
	Action AddAction
	// Index is the index, in the features given to the screens, of the
	// selected feature; meaningful for ActionCancel and ActionAdd.
	Index int
	// Values are the trimmed input values, one per input; meaningful for
	// ActionAdd.
	Values []string
}

// addScreen identifies the screen of the add program that is shown.
type addScreen int

const (
	// screenSelector is the list of features.
	screenSelector addScreen = iota
	// screenForm is the form of the selected feature.
	screenForm
	// screenConfirm is the confirmation.
	screenConfirm
)

// AddModel is the Bubble Tea model of the whole add program: the selector,
// the form and the confirmation (FR-008, FR-010, FR-011, FR-018).
type AddModel struct {
	features []Feature
	styled   bool
	width    int
	height   int
	screen   addScreen
	selector listModel
	form     formModel
	confirm  confirmModel
	// current is the index of the selected feature.
	current int
	// kept holds, per feature index, the values typed in its form, kept for
	// the length of the run (FR-017).
	kept   map[int][]string
	result AddResult
}

// NewAddModel builds the add program model, showing the selector with the
// pointer on the first feature (FR-008).
//
// In: features, the features to list, in display order; styled, whether to
// add ANSI colour; width and height, the initial terminal size in cells.
// Out: the model; the terminal's own size replaces the initial one when
// Bubble Tea reports it. Its result is ActionQuit until the user adds or
// cancels.
// Side effects: none.
func NewAddModel(features []Feature, styled bool, width, height int) AddModel {
	return AddModel{
		features: features,
		styled:   styled,
		width:    width,
		height:   height,
		selector: newSelectorModel(features, styled, width, height),
		kept:     map[int][]string{},
		result:   AddResult{Action: ActionQuit},
	}
}

// Result returns how the program ended.
//
// In: none. Out: the result; ActionQuit if the user has not added or
// cancelled. Side effects: none.
func (m AddModel) Result() AddResult { return m.result }

// Init is part of the Bubble Tea model contract.
//
// In: none. Out: nil, no initial command. Side effects: none.
func (m AddModel) Init() tea.Cmd { return nil }

// Update routes a message to the active screen and moves between screens
// (FR-010, FR-015, FR-017, FR-018).
//
// In: msg, any Bubble Tea message.
// Out: the updated model and a command. A window-size message resizes every
// screen. The selector's choice opens the form, or the confirmation if the
// feature has no inputs; a finished form opens the confirmation; `Esc` in
// the form returns to the selector keeping the values; `Esc` in the
// confirmation returns to the form, or to the selector if the feature has no
// inputs; `Add` and `Cancel` set the result and quit.
// Side effects: calls the feature's Plan, Check and Command when the
// confirmation opens.
func (m AddModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.selector, _ = m.selector.Update(msg)
		m.form, _ = m.form.Update(msg)
		if m.screen == screenConfirm {
			m.confirm, _ = m.confirm.Update(msg)
		}
		return m, nil
	case selectMsg:
		return m.open(msg.Index)
	case formBackMsg:
		m.kept[m.current] = msg.Values
		m.screen = screenSelector
		return m, nil
	case formDoneMsg:
		m.kept[m.current] = m.form.values(false)
		var cmd tea.Cmd
		m.confirm, cmd = newConfirmModel(m.features[m.current], msg.Values, m.styled, m.width, m.height)
		m.screen = screenConfirm
		return m, cmd
	}
	switch m.screen {
	case screenForm:
		var cmd tea.Cmd
		m.form, cmd = m.form.Update(msg)
		return m, cmd
	case screenConfirm:
		return m.updateConfirm(msg)
	}
	var cmd tea.Cmd
	m.selector, cmd = m.selector.Update(msg)
	return m, cmd
}

// open shows the form of a feature, or its confirmation if it has no
// inputs.
//
// In: i, the index of the feature. Out: the model on the new screen and, for
// the confirmation, the disable-mouse command. Side effects: as for Update.
func (m AddModel) open(i int) (tea.Model, tea.Cmd) {
	m.current = i
	f := m.features[i]
	if len(f.Inputs) == 0 {
		var cmd tea.Cmd
		m.confirm, cmd = newConfirmModel(f, nil, m.styled, m.width, m.height)
		m.screen = screenConfirm
		return m, cmd
	}
	m.form = newFormModel(f, m.kept[i], m.styled, m.width, m.height)
	m.screen = screenForm
	return m, nil
}

// updateConfirm routes a message to the confirmation and acts on how it
// ended.
//
// In: msg, the message. Out: the model and command, as for Update. Side
// effects: none.
func (m AddModel) updateConfirm(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.confirm, cmd = m.confirm.Update(msg)
	switch m.confirm.Result() {
	case confirmAdd:
		m.result = AddResult{Action: ActionAdd, Index: m.current, Values: m.confirm.values}
		return m, tea.Quit
	case confirmCancel:
		m.result = AddResult{Action: ActionCancel, Index: m.current}
		return m, tea.Quit
	case confirmBack:
		m.screen = screenSelector
		if len(m.features[m.current].Inputs) > 0 {
			m.screen = screenForm
		}
	}
	return m, cmd
}

// View renders the active screen.
//
// In: none. Out: the screen text of the selector, the form or the
// confirmation. Side effects: none.
func (m AddModel) View() string {
	switch m.screen {
	case screenForm:
		return m.form.View()
	case screenConfirm:
		return m.confirm.View()
	}
	return m.selector.View()
}

// RunAdd runs the interactive add screens until the user leaves them.
//
// In: ctx, the run context, whose cancellation stops the screens; features,
// the features to list; in and out, the terminal input and output; styled,
// whether to add ANSI colour.
// Out: the result: ActionQuit when the user leaves with `q`, `Esc` or
// `Ctrl+C`; ActionCancel or ActionAdd, with the feature index and, for add,
// the trimmed values, from the confirmation. On error the result is the zero
// value and the error is context.Cause(ctx) if ctx is cancelled (EC-020),
// otherwise the failure to start or run the program, wrapped as `cannot
// start the add screen: <cause>` (FR-021). In every case the terminal has
// been restored (the alternate screen left) before returning.
// Side effects: takes over the terminal on in and out, using the alternate
// screen and cell-motion mouse reporting; installs no signal handler; calls
// the features' Plan, Check and Command, which write nothing (FR-020).
func RunAdd(ctx context.Context, features []Feature, in io.Reader, out io.Writer, styled bool) (AddResult, error) {
	model := NewAddModel(features, styled, defaultWidth, defaultHeight)
	p := tea.NewProgram(model,
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
		tea.WithInput(in),
		tea.WithOutput(out),
		tea.WithContext(ctx),
		tea.WithoutSignalHandler(),
	)
	final, err := p.Run()
	if ctx.Err() != nil {
		return AddResult{}, context.Cause(ctx)
	}
	if err != nil {
		return AddResult{}, fmt.Errorf("cannot start the add screen: %w", err)
	}
	if am, ok := final.(AddModel); ok {
		return am.result, nil
	}
	return AddResult{Action: ActionQuit}, nil
}
