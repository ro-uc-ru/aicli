package gen

import (
	"bytes"
	"strings"
	"testing"
	"testing/fstest"
)

// layoutFiles is the exact file list of the spec's fixed layout.
var layoutFiles = []string{
	".dockerignore", ".env.example", ".gitignore", "Dockerfile", "README.md",
	"alembic.ini", "docker-compose.yml", "pyproject.toml",
	"app/__init__.py", "app/ai/__init__.py", "app/api/__init__.py",
	"app/api/router.py", "app/api/routers/__init__.py",
	"app/api/routers/health.py", "app/config.py", "app/db.py", "app/errors.py",
	"app/logging.py", "app/main.py", "app/models/__init__.py",
	"app/repositories/__init__.py", "app/schemas/__init__.py",
	"app/schemas/errors.py", "app/schemas/health.py",
	"app/services/__init__.py",
	"migrations/env.py", "migrations/script.py.mako", "migrations/versions/.gitkeep",
	"tests/__init__.py", "tests/conftest.py", "tests/test_config.py",
	"tests/test_errors.py", "tests/test_health.py",
}

// layoutDirs is the exact directory list of the spec's fixed layout.
var layoutDirs = []string{
	"app", "app/ai", "app/api", "app/api/routers", "app/models",
	"app/repositories", "app/schemas", "app/services", "migrations",
	"migrations/versions", "tests",
}

// TestRenderLayout checks that the rendered paths equal the fixed layout.
// In: t, the running test. Out: none. Side effects: none.
// Covers: FR-002, FR-011.
func TestRenderLayout(t *testing.T) {
	entries, err := Render("myapp")
	if err != nil {
		t.Fatal(err)
	}
	var files, dirs []string
	for i, e := range entries {
		if i > 0 && entries[i-1].Path >= e.Path {
			t.Errorf("entries not sorted at %q", e.Path)
		}
		if e.Kind == KindDir {
			dirs = append(dirs, e.Path)
		} else {
			files = append(files, e.Path)
		}
	}
	if got, want := strings.Join(files, "\n"), strings.Join(sorted(layoutFiles), "\n"); got != want {
		t.Errorf("files differ:\ngot:\n%s\nwant:\n%s", got, want)
	}
	if got, want := strings.Join(dirs, "\n"), strings.Join(sorted(layoutDirs), "\n"); got != want {
		t.Errorf("dirs differ:\ngot:\n%s\nwant:\n%s", got, want)
	}
	for _, f := range files {
		if f == ".env" || strings.HasSuffix(f, ".tmpl") {
			t.Errorf("unexpected file %q", f)
		}
	}
}

// TestConventionFoldersNotEmpty checks FR-003: every convention folder has a file.
// In: t, the running test. Out: none. Side effects: none.
// Covers: FR-003.
func TestConventionFoldersNotEmpty(t *testing.T) {
	entries, err := Render("myapp")
	if err != nil {
		t.Fatal(err)
	}
	for _, d := range []string{"models", "schemas", "repositories", "services", "ai", "api", "api/routers"} {
		if !hasFileUnder(entries, "app/"+d+"/") {
			t.Errorf("app/%s has no file", d)
		}
	}
	for _, d := range []string{"tests/", "migrations/"} {
		if !hasFileUnder(entries, d) {
			t.Errorf("%s has no file", d)
		}
	}
}

// TestRenderName checks that the project name appears in the templated files.
// In: t, the running test. Out: none. Side effects: none.
// Covers: FR-010.
func TestRenderName(t *testing.T) {
	entries, err := Render("myapp")
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range []string{"pyproject.toml", "app/config.py", "README.md"} {
		var found bool
		for _, e := range entries {
			if e.Path == p {
				found = true
				if !bytes.Contains(e.Content, []byte("myapp")) {
					t.Errorf("%s lacks the project name", p)
				}
				if bytes.Contains(e.Content, []byte("{{")) {
					t.Errorf("%s has an unrendered action", p)
				}
			}
		}
		if !found {
			t.Errorf("%s missing", p)
		}
	}
}

// TestRenderDeterministic checks that two renders are identical.
// In: t, the running test. Out: none. Side effects: none.
// Covers: FR-022.
func TestRenderDeterministic(t *testing.T) {
	a, _ := Render("myapp")
	b, _ := Render("myapp")
	if len(a) != len(b) {
		t.Fatal("different lengths")
	}
	for i := range a {
		if a[i].Path != b[i].Path || !bytes.Equal(a[i].Content, b[i].Content) {
			t.Errorf("entry %q differs", a[i].Path)
		}
	}
}

// TestRenderTemplateError checks that a bad template returns a contextual error.
// In: t, the running test. Out: none. Side effects: none.
// Covers: FR-016.
func TestRenderTemplateError(t *testing.T) {
	fsys := fstest.MapFS{"skeleton/x.tmpl": {Data: []byte("{{.Missing}}")}}
	if _, err := render(fsys, "a"); err == nil || !strings.Contains(err.Error(), "x.tmpl") {
		t.Errorf("want error naming x.tmpl, got %v", err)
	}
}

// sorted returns a sorted copy of s.
// In: s, the strings. Out: the sorted copy. Side effects: none.
func sorted(s []string) []string {
	out := append([]string(nil), s...)
	for i := range out {
		for j := i + 1; j < len(out); j++ {
			if out[j] < out[i] {
				out[i], out[j] = out[j], out[i]
			}
		}
	}
	return out
}

// hasFileUnder reports whether a file entry lives under the prefix.
// In: entries, the rendered entries; prefix, a path prefix ending in "/".
// Out: true if one file entry starts with prefix. Side effects: none.
func hasFileUnder(entries []Entry, prefix string) bool {
	for _, e := range entries {
		if e.Kind == KindFile && strings.HasPrefix(e.Path, prefix) {
			return true
		}
	}
	return false
}
