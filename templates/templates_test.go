package templates

import (
	"io/fs"
	"regexp"
	"strings"
	"testing"
)

// read returns the content of a skeleton file.
// In: t, the running test; name, path relative to the skeleton root.
// Out: the file content as a string. Side effects: fails the test if the
// file cannot be read.
func read(t *testing.T, name string) string {
	t.Helper()
	b, err := fs.ReadFile(Skeleton, "skeleton/"+name)
	if err != nil {
		t.Fatalf("read %s: %v", name, err)
	}
	return string(b)
}

// lines splits content into a set of its trimmed lines.
// In: s, the file content. Out: set of lines. No side effects.
func lines(s string) map[string]bool {
	set := map[string]bool{}
	for _, l := range strings.Split(s, "\n") {
		set[strings.TrimSpace(l)] = true
	}
	return set
}

// TestProjectFilesExist checks that every project-level file is embedded.
// In: t, the running test. Out: none. Side effects: none.
// Covers: FR-002.
func TestProjectFilesExist(t *testing.T) {
	for _, n := range []string{
		".env.example", ".gitignore", ".dockerignore", "Dockerfile",
		"docker-compose.yml", "README.md.tmpl", "pyproject.toml.tmpl",
	} {
		if _, err := fs.Stat(Skeleton, "skeleton/"+n); err != nil {
			t.Errorf("missing %s: %v", n, err)
		}
	}
}

// TestIgnoreFiles checks the FR-009/FR-025 entries of .gitignore and
// .dockerignore and that .gitignore never excludes .env.example or uv.lock.
// In: t, the running test. Out: none. Side effects: none.
// Covers: FR-009, FR-025.
func TestIgnoreFiles(t *testing.T) {
	common := []string{
		".env", "*.db", ".venv/", "__pycache__/", "*.py[cod]", "*.egg-info/",
		"build/", "dist/", ".pytest_cache/", ".ruff_cache/", ".coverage",
		"htmlcov/", ".idea/", ".vscode/", ".DS_Store", ".agents/", ".claude/",
		".mcp.json",
	}
	git := lines(read(t, ".gitignore"))
	docker := lines(read(t, ".dockerignore"))
	for _, e := range common {
		if !git[e] {
			t.Errorf(".gitignore lacks %q", e)
		}
		if !docker[e] {
			t.Errorf(".dockerignore lacks %q", e)
		}
	}
	if !docker[".git"] {
		t.Error(".dockerignore lacks .git")
	}
	for _, bad := range []string{".env.example", "uv.lock", ".env*", ".env.*"} {
		if git[bad] {
			t.Errorf(".gitignore must not exclude %q", bad)
		}
	}
	if docker["uv.lock"] {
		t.Error(".dockerignore must not exclude uv.lock")
	}
}

// TestComposeIsEmpty checks docker-compose.yml is exactly "services: {}".
// In: t, the running test. Out: none. Side effects: none.
// Covers: FR-018.
func TestComposeIsEmpty(t *testing.T) {
	if got := strings.TrimSpace(read(t, "docker-compose.yml")); got != "services: {}" {
		t.Errorf("docker-compose.yml = %q", got)
	}
}

// TestPyproject checks pyproject.toml.tmpl lists only the FR-007 dependencies,
// all with a lower bound.
// In: t, the running test. Out: none. Side effects: none.
// Covers: FR-007, FR-011, FR-020.
func TestPyproject(t *testing.T) {
	src := read(t, "pyproject.toml.tmpl")
	if !strings.Contains(src, `requires-python = ">=3.12"`) {
		t.Error("requires-python missing")
	}
	var deps []string
	for _, m := range regexp.MustCompile(`(?m)^\s+"([^"]+)",$`).FindAllStringSubmatch(src, -1) {
		deps = append(deps, m[1])
	}
	want := []string{
		"fastapi", "uvicorn", "sqlalchemy", "alembic", "pydantic-settings",
		"pytest", "httpx", "ruff",
	}
	if len(deps) != len(want) {
		t.Fatalf("got %d dependencies %v, want %d", len(deps), deps, len(want))
	}
	for i, d := range deps {
		if !strings.HasPrefix(d, want[i]+">=") {
			t.Errorf("dependency %q: want %s with >=", d, want[i])
		}
	}
}

// TestAppCore checks the app/ core templates exist and that config.py.tmpl
// defines exactly the three settings with their defaults and env_file.
// In: t, the running test. Out: none. Side effects: none.
// Covers: FR-005, FR-021, FR-026.
func TestAppCore(t *testing.T) {
	for _, n := range []string{
		"__init__.py", "config.py.tmpl", "logging.py", "db.py", "main.py",
	} {
		if _, err := fs.Stat(Skeleton, "skeleton/app/"+n); err != nil {
			t.Errorf("missing app/%s: %v", n, err)
		}
	}
	cfg := read(t, "app/config.py.tmpl")
	var names []string
	for _, m := range regexp.MustCompile(`(?m)^    ([A-Z_]+)\s*:`).FindAllStringSubmatch(cfg, -1) {
		names = append(names, m[1])
	}
	if got := strings.Join(names, ","); got != "APP_NAME,DATABASE_URL,LOG_LEVEL" {
		t.Errorf("settings = %s", got)
	}
	for _, want := range []string{
		`DATABASE_URL: str = "sqlite:///./app.db"`,
		`LOG_LEVEL: Literal["DEBUG", "INFO", "WARNING", "ERROR", "CRITICAL"]`,
		`env_file=".env"`,
		`APP_NAME: str = "{{.Name}}"`,
	} {
		if !strings.Contains(cfg, want) {
			t.Errorf("config.py.tmpl lacks %s", want)
		}
	}
}

// TestAppAPIAndErrors checks the API, schemas, errors and empty-package
// templates exist, app/ai/__init__.py is empty and errors.py holds the
// literal messages of 404, 405, 422 and 500.
// In: t, the running test. Out: none. Side effects: none.
// Covers: FR-004, FR-019, FR-027, EC-014.
func TestAppAPIAndErrors(t *testing.T) {
	for _, n := range []string{
		"api/__init__.py", "api/router.py", "api/routers/__init__.py",
		"api/routers/health.py", "schemas/__init__.py", "schemas/health.py",
		"schemas/errors.py", "errors.py", "models/__init__.py",
		"repositories/__init__.py", "services/__init__.py", "ai/__init__.py",
	} {
		if _, err := fs.Stat(Skeleton, "skeleton/app/"+n); err != nil {
			t.Errorf("missing app/%s: %v", n, err)
		}
	}
	if got := read(t, "app/ai/__init__.py"); got != "" {
		t.Errorf("app/ai/__init__.py not empty: %q", got)
	}
	errs := read(t, "app/errors.py")
	for _, m := range []string{
		"Not Found", "Method Not Allowed", "Unprocessable Entity",
		"Internal Server Error",
	} {
		if !strings.Contains(errs, `"`+m+`"`) {
			t.Errorf("errors.py lacks message %q", m)
		}
	}
}

// TestAlembic checks the Alembic templates exist, alembic.ini points at
// migrations without a sqlalchemy.url, and env.py reads the URL from Settings.
// In: t, the running test. Out: none. Side effects: none.
// Covers: FR-006, FR-021.
func TestAlembic(t *testing.T) {
	for _, n := range []string{
		"alembic.ini", "migrations/env.py", "migrations/script.py.mako",
		"migrations/versions/.gitkeep",
	} {
		if _, err := fs.Stat(Skeleton, "skeleton/"+n); err != nil {
			t.Errorf("missing %s: %v", n, err)
		}
	}
	ini := read(t, "alembic.ini")
	if !strings.Contains(ini, "script_location = migrations") {
		t.Error("alembic.ini lacks script_location = migrations")
	}
	if strings.Contains(ini, "sqlalchemy.url") {
		t.Error("alembic.ini must not set sqlalchemy.url")
	}
	env := read(t, "migrations/env.py")
	for _, want := range []string{
		"from app.config import settings", "settings.DATABASE_URL",
		"target_metadata = Base.metadata",
	} {
		if !strings.Contains(env, want) {
			t.Errorf("env.py lacks %q", want)
		}
	}
}

// TestGeneratedTests checks the generated pytest files exist and cover the
// default DATABASE_URL, the app title, an invalid LOG_LEVEL, the logging setup
// and the 404 and 500 ErrorModel responses.
// In: t, the running test. Out: none. Side effects: none.
// Covers: FR-005, FR-008, FR-026, FR-027, EC-013, EC-014.
func TestGeneratedTests(t *testing.T) {
	for _, n := range []string{
		"__init__.py", "conftest.py", "test_config.py", "test_health.py",
		"test_errors.py",
	} {
		if _, err := fs.Stat(Skeleton, "skeleton/tests/"+n); err != nil {
			t.Errorf("missing tests/%s: %v", n, err)
		}
	}
	if !strings.Contains(read(t, "tests/conftest.py"), "raise_server_exceptions=False") {
		t.Error("conftest.py lacks raise_server_exceptions=False")
	}
	cfg := read(t, "tests/test_config.py")
	for _, want := range []string{
		`"sqlite:///./app.db"`, "info", "title", `LOG_LEVEL="LOUD"`,
		"configure_logging",
	} {
		if !strings.Contains(cfg, want) {
			t.Errorf("test_config.py lacks %q", want)
		}
	}
	errs := read(t, "tests/test_errors.py")
	for _, want := range []string{
		`"status": 404`, `"message": "Not Found"`, `"status": 500`,
		`"message": "Internal Server Error"`,
	} {
		if !strings.Contains(errs, want) {
			t.Errorf("test_errors.py lacks %q", want)
		}
	}
}
