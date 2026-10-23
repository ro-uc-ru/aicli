package tui

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// listsFeature returns a feature whose Check returns the given preview and
// error.
func listsFeature(p Preview, err error) Feature {
	return Feature{
		Name: "llm-provider", Short: "Add one or more LLM providers.", Listed: true,
		Inputs:  []FeatureInput{{Name: "provider"}},
		Command: func(v []string) string { return "aicli add llm-provider " + strings.Join(v, " ") },
		Check:   func([]string) (Preview, error) { return p, err },
	}
}

// TestConfirmPreviewRows checks the rows of the confirmation: both lists
// with a blank row between them (first run), only the second list or only
// the first (later run), and no rows at all for an empty preview.
//
// Covers: FR-011
func TestConfirmPreviewRows(t *testing.T) {
	cases := []struct {
		name string
		p    Preview
		want []string
	}{
		{"first run", Preview{Create: []string{"a.py", "b.py"}, Update: []string{"router.py", "pyproject.toml"}},
			[]string{"Will create:", "  a.py", "  b.py", "", "Will update:", "  router.py", "  pyproject.toml"}},
		{"later run", Preview{Create: []string{"claude.py"}, Update: []string{"pyproject.toml"}},
			[]string{"Will create:", "  claude.py", "", "Will update:", "  pyproject.toml"}},
		{"only create", Preview{Create: []string{"a.py"}}, []string{"Will create:", "  a.py"}},
		{"only update", Preview{Update: []string{"a.py"}}, []string{"Will update:", "  a.py"}},
		{"nothing", Preview{}, nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			m, _ := newConfirmModel(listsFeature(c.p, nil), []string{"claude"}, false, 80, 30)
			if !reflect.DeepEqual(m.rest, c.want) {
				t.Errorf("rows = %q, want %q", m.rest, c.want)
			}
		})
	}
}

// TestConfirmFirstAndLaterRun checks the whole screen of the first-run and
// later-run confirmations of the design, plain and styled.
//
// Covers: FR-011
func TestConfirmFirstAndLaterRun(t *testing.T) {
	m, _ := newConfirmModel(listsFeature(Preview{
		Create: []string{"app/ai/llm/claude.py", "tests/test_llm_claude.py"},
		Update: []string{"pyproject.toml", ".env.example"},
	}, nil), []string{"claude"}, false, 80, 30)
	want := strings.Join([]string{
		"aicli › add › llm-provider",
		"Add one or more LLM providers.",
		"",
		"aicli add llm-provider claude",
		"",
		"Will create:",
		"  app/ai/llm/claude.py",
		"  tests/test_llm_claude.py",
		"",
		"Will update:",
		"  pyproject.toml",
		"  .env.example",
		"",
		"  Add    [ Cancel ]",
		"←/→/tab choose · enter confirm · y add · n cancel · esc back · ctrl+c quit",
	}, "\n")
	if got := m.View(); got != want {
		t.Errorf("view:\n%s\nwant:\n%s", got, want)
	}
	styled, _ := newConfirmModel(listsFeature(Preview{Create: []string{"a"}, Update: []string{"b"}}, nil), []string{"claude"}, true, 80, 30)
	if !strings.Contains(styled.View(), "\x1b") {
		t.Error("styled view has no ESC")
	}
}

// TestConfirmBothListsScroll checks that on a short terminal both lists
// scroll together as one list, headings included, and that every row can be
// reached.
//
// Covers: FR-011, EC-010
func TestConfirmBothListsScroll(t *testing.T) {
	f := listsFeature(Preview{Create: []string{"c1", "c2"}, Update: []string{"u1", "u2"}}, nil)
	f.Short = ""
	m, _ := newConfirmModel(f, []string{"x"}, false, 80, 8)
	// header, cmd, up, 2 rows, down, buttons, footer: rows are
	// Will create:, c1, c2, blank, Will update:, u1, u2.
	if v := m.View(); !strings.Contains(v, "Will create:") || !strings.Contains(v, "  c1") || strings.Contains(v, "c2") || !strings.Contains(v, "↓ more") {
		t.Errorf("top:\n%s", v)
	}
	for i := 0; i < 20; i++ {
		m, _ = pressC(m, tea.KeyMsg{Type: tea.KeyDown})
	}
	if v := m.View(); !strings.Contains(v, "  u2") || !strings.Contains(v, "  u1") || strings.Contains(v, "Will create:") || strings.Contains(v, "↓ more") || !strings.Contains(v, "↑ more") {
		t.Errorf("bottom:\n%s", v)
	}
}

// TestConfirmBlockedByCheck checks that a Check error replaces both lists
// with only `Cancel`: a missing marker and an item added after the form
// opened.
//
// Covers: EC-007, EC-010
func TestConfirmBlockedByCheck(t *testing.T) {
	cases := map[string]struct {
		err  string
		want []string
	}{
		"missing marker": {
			"cannot add llm-provider, app/api/router.py has no \"# aicli:router-imports\" marker",
			[]string{"Cannot add llm-provider, app/api/router.py has no \"# aicli:router-imports\" marker"},
		},
		"added after the form opened": {
			"cannot add llm-provider, these providers are already added:\n  openai (app/ai/llm/openai.py)",
			[]string{"Cannot add llm-provider, these providers are already added:", "  openai (app/ai/llm/openai.py)"},
		},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			f := listsFeature(Preview{Create: []string{"never.py"}}, errors.New("Error: "+c.err))
			m, _ := newConfirmModel(f, []string{"openai"}, false, 80, 30)
			if !m.blocked || !reflect.DeepEqual(m.rest, c.want) {
				t.Fatalf("blocked %v, rows %q", m.blocked, m.rest)
			}
			v := m.View()
			if strings.Contains(v, "Will create:") || strings.Contains(v, "never.py") || strings.Contains(v, "Add    [") || !strings.Contains(v, "[ Cancel ]") {
				t.Errorf("view:\n%s", v)
			}
			if !strings.Contains(m.footer(m.layout()), "enter cancel") {
				t.Errorf("footer = %q", m.footer(m.layout()))
			}
		})
	}
}
