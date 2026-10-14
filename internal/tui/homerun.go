package tui

import (
	"context"
	"fmt"
	"io"

	tea "github.com/charmbracelet/bubbletea"
)

const (
	// defaultWidth and defaultHeight are the size assumed until the terminal
	// reports its own.
	defaultWidth  = 80
	defaultHeight = 24
)

// HomeModel is the Bubble Tea model of the whole home screen: the list view
// and, once an entry is opened, its detail view.
type HomeModel struct {
	entries  []Entry
	list     listModel
	detail   detailModel
	inDetail bool
	err      error
}

// NewHomeModel builds the home screen model, showing the list with the
// pointer on the first entry (FR-005).
//
// In: entries, the commands to list, in display order; styled, whether to
// add ANSI colour; width and height, the initial terminal size in cells.
// Out: the model; the terminal's own size replaces the initial one when
// Bubble Tea reports it.
// Side effects: none.
func NewHomeModel(entries []Entry, styled bool, width, height int) HomeModel {
	return HomeModel{
		entries: entries,
		list:    newListModel(entries, styled, width, height),
		detail:  newDetailModel(styled, width, height),
	}
}

// Err returns the error that made the model quit, if any.
//
// In: none. Out: the error returned by the help producer of the entry that
// could not be opened, or nil. Side effects: none.
func (m HomeModel) Err() error { return m.err }

// Init is part of the Bubble Tea model contract.
//
// In: none. Out: nil, no initial command. Side effects: none.
func (m HomeModel) Init() tea.Cmd { return nil }

// Update routes a message to the active view (FR-007, FR-010, FR-015,
// FR-016).
//
// In: msg, any Bubble Tea message.
// Out: the updated model and a command. A window-size message resizes both
// views. Other messages go to the list, or to the detail view while it is
// shown, except mouse events there. When the list asks to open an entry,
// its help is produced and shown; if that fails the error is stored and the
// program quits. When the detail view closes, the list is shown again with
// its pointer unchanged.
// Side effects: calls the help producer of the opened entry.
func (m HomeModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.list, _ = m.list.Update(msg)
		m.detail, _ = m.detail.Update(msg)
		return m, nil
	case openMsg:
		return m.openEntry(msg.Index)
	case tea.MouseMsg:
		if m.inDetail {
			return m, nil
		}
	}
	if m.inDetail {
		var cmd tea.Cmd
		m.detail, cmd = m.detail.Update(msg)
		if m.detail.closed {
			m.inDetail = false
		}
		return m, cmd
	}
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

// openEntry shows the detail view of an entry.
//
// In: i, the index of the entry.
// Out: the model showing the entry's help and the command that turns mouse
// capture off; or, if the help cannot be produced, the model holding the
// error and tea.Quit.
// Side effects: calls the entry's help producer.
func (m HomeModel) openEntry(i int) (tea.Model, tea.Cmd) {
	e := m.entries[i]
	help := ""
	if e.Help != nil {
		var err error
		if help, err = e.Help(); err != nil {
			m.err = err
			return m, tea.Quit
		}
	}
	var cmd tea.Cmd
	m.detail, cmd = m.detail.open(e.Path, help)
	m.inDetail = true
	return m, cmd
}

// View renders the active view.
//
// In: none. Out: the screen text of the list, or of the detail view while it
// is shown. Side effects: none.
func (m HomeModel) View() string {
	if m.inDetail {
		return m.detail.View()
	}
	return m.list.View()
}

// RunHome runs the interactive home screen until the user quits.
//
// In: ctx, the run context, whose cancellation stops the screen; entries,
// the commands to list; in and out, the terminal input and output; styled,
// whether to add ANSI colour.
// Out: nil when the user quits with `q` or `Ctrl+C`; the error of a help
// producer that failed (FR-016); context.Cause(ctx) if ctx is cancelled
// (EC-014); otherwise the failure to start or run the program, wrapped as
// `cannot start the home screen: <cause>`. In every case the terminal has
// been restored (the alternate screen left) before returning.
// Side effects: takes over the terminal on in and out, using the alternate
// screen and cell-motion mouse reporting; installs no signal handler.
func RunHome(ctx context.Context, entries []Entry, in io.Reader, out io.Writer, styled bool) error {
	model := NewHomeModel(entries, styled, defaultWidth, defaultHeight)
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
		return context.Cause(ctx)
	}
	if err != nil {
		return fmt.Errorf("cannot start the home screen: %w", err)
	}
	if hm, ok := final.(HomeModel); ok && hm.err != nil {
		return hm.err
	}
	return nil
}
