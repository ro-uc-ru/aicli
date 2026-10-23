//go:build integration

package root

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// fakeProviderEnv holds fake values for every variable of the four providers.
var fakeProviderEnv = []string{
	"OPENAI_API_KEY=fake-openai-key", "OPENAI_MODEL=fake-openai-model",
	"ANTHROPIC_API_KEY=fake-anthropic-key", "ANTHROPIC_MODEL=fake-anthropic-model",
	"GEMINI_API_KEY=fake-gemini-key", "GEMINI_MODEL=fake-gemini-model",
	"OLLAMA_MODEL=fake-ollama-model",
}

// deadProxy is a proxy address that refuses connections, so that any network
// call the generated tests made would fail.
const deadProxy = "http://127.0.0.1:9"

// uvRun runs `uv <args...>` in dir through os/exec with an argument list.
//
// In: t, the test; dir, the working directory; env, extra KEY=VALUE entries
// added to the environment of uv; args, the uv arguments.
// Out: the combined output. The test fails if uv exits non-zero.
// Side effects: spawns uv; `uv sync` uses the network to install the
// generated project's dependencies (TST-6).
func uvRun(t *testing.T, dir string, env []string, args ...string) string {
	t.Helper()
	cmd := exec.Command("uv", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), env...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("uv %s: %v\n%s", strings.Join(args, " "), err, out)
	}
	return string(out)
}

// checkGenerated runs the generated project's pytest, `ruff check` and
// `ruff format --check` with `uv run --no-sync`, first with the project's own
// `.env` of fake values and then with the provider variables exported as well.
// Any network call made by pytest fails, because the proxy variables point to
// a dead address.
//
// In: t, the test; dir, the project root. Out: none; the test fails on the
// first failing command. Side effects: spawns uv; writes caches in dir.
func checkGenerated(t *testing.T, dir string) {
	t.Helper()
	proxy := []string{"HTTP_PROXY=" + deadProxy, "HTTPS_PROXY=" + deadProxy, "ALL_PROXY=" + deadProxy, "NO_PROXY="}
	for _, env := range [][]string{proxy, append(append([]string(nil), proxy...), fakeProviderEnv...)} {
		uvRun(t, dir, env, "run", "--no-sync", "pytest", "-q")
		uvRun(t, dir, env, "run", "--no-sync", "ruff", "check")
		uvRun(t, dir, env, "run", "--no-sync", "ruff", "format", "--check")
	}
}

// readProject returns the content of a project file.
//
// In: t, the test; dir, the project root; rel, the "/" path. Out: the content.
// Side effects: none.
func readProject(t *testing.T, dir, rel string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(rel)))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// TestIntegrationLLMProvider scaffolds a project, runs `aicli add
// llm-provider openai`, checks the generated tests and lint before and after
// `uv sync`, applies `ruff check --fix` and `ruff format` (the injected files
// must not change), adds `claude gemini ollama`, checks that the router has
// the llm import and include_router lines once each, and checks again. It runs
// with a `.env` of fake provider values and with the variables exported.
//
// In: t, the test. Out: none. Side effects: writes in a temp directory; spawns
// uv; uses the network only for `uv sync` (TST-6).
// Covers: FR-015, FR-016, FR-017, FR-018, FR-019, FR-020, FR-021, FR-022, EC-012, EC-017, EC-018, EC-019, EC-020, EC-021, EC-022, EC-023, EC-024, EC-026, EC-027, EC-031
func TestIntegrationLLMProvider(t *testing.T) {
	if _, err := exec.LookPath("uv"); err != nil {
		t.Skip("uv is not on PATH")
	}
	dir := scaffolded(t)
	uvRun(t, dir, nil, "sync")
	env := strings.Join(fakeProviderEnv, "\n") + "\n"
	if err := os.WriteFile(filepath.Join(dir, ".env"), []byte(env), 0o600); err != nil {
		t.Fatal(err)
	}

	if code, _, errb := runReal(t, dir, "add", "llm-provider", "openai"); code != 0 {
		t.Fatalf("add openai: %s", errb)
	}
	checkGenerated(t, dir)
	uvRun(t, dir, nil, "sync")
	checkGenerated(t, dir)

	injected := []string{"app/api/router.py", "pyproject.toml", ".env.example"}
	before := map[string]string{}
	for _, f := range injected {
		before[f] = readProject(t, dir, f)
	}
	uvRun(t, dir, nil, "run", "--no-sync", "ruff", "check", "--fix")
	uvRun(t, dir, nil, "run", "--no-sync", "ruff", "format")
	for _, f := range injected {
		if got := readProject(t, dir, f); got != before[f] {
			t.Errorf("%s changed after ruff --fix and format:\n%s", f, got)
		}
	}

	if code, _, errb := runReal(t, dir, "add", "llm-provider", "claude", "gemini", "ollama"); code != 0 {
		t.Fatalf("add claude gemini ollama: %s", errb)
	}
	router := readProject(t, dir, "app/api/router.py")
	for _, line := range []string{"from app.api.routers import llm\n", "router.include_router(llm.router)\n"} {
		if n := strings.Count(router, line); n != 1 {
			t.Errorf("router.py has %q %d times, want 1:\n%s", strings.TrimSpace(line), n, router)
		}
	}
	checkGenerated(t, dir)
	uvRun(t, dir, nil, "sync")
	checkGenerated(t, dir)
}
