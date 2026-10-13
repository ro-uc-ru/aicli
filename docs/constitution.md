# Constitution

- Status: approved

> Basis of the project. Every spec, plan and task MUST comply with it.
> It overrides any spec/plan. Once approved it is NOT changed during normal work;
> any change needs team approval.
> Each statement is VERIFIABLE: a reviewer can look at the repo and say
> "complies" or "does not".
> Cite statements by ID (e.g. STK-1). IDs are never renumbered. A removed
> statement keeps its line with "(retired: <reason>)" appended.

## 1. Stack (STK)
- STK-1 The CLI is written only in Go, with the version pinned in go.mod.
- STK-2 The command tree is built with Cobra; interactive UI uses only Charm libraries (Bubble Tea, Huh, Lipgloss; others only if a spec requires them).
- STK-3 Generated projects target Python, FastAPI, SQLAlchemy, Alembic, Pydantic with pydantic-settings, uv and pytest by default, plus uvicorn (server), httpx (test client) and ruff (lint) as minimal base tooling. Any further dependency (database drivers, LLM providers, etc.) is decided by the spec that needs it, under STK-4.
- STK-4 No new dependency (Go module or generated-project package) without team approval.
- STK-5 Optional capabilities (LLM provider, vector store, queues, observability) are opt-in features or flags, never part of the default scaffold.
- STK-6 `golang.org/x/sys` and `golang.org/x/term` are pre-approved Go dependencies, to be added only when a spec needs them (atomic no-replace rename; TTY detection). Platform-specific code that uses them lives in files with build tags for the platforms in LIM-1, as the only exception to ARC-8.

## 2. Architecture and Boundaries (ARC)
- ARC-1 Every feature has a spec in `specs/` before any code is written for it.
- ARC-2 Each command or subcommand is one folder directly under `cmd/`; nested folders under `cmd/` do not exist.
- ARC-3 `cmd/<name>` packages contain only the Cobra command, flags and wiring; logic lives in `internal/`.
- ARC-4 No `cmd/` package imports another `cmd/` package, except `cmd/root`.
- ARC-5 Generated Python code comes from templates in `templates/` embedded with `embed`, not from string building in Go code.
- ARC-6 All generated projects share one fixed layout; no flag or config changes it.
- ARC-7 Only `internal/tui` imports Bubble Tea, Huh or Lipgloss.
- ARC-8 Code never depends on the host OS: OS paths are built only with `path/filepath` (paths inside `embed.FS` use `path`), files are handled only through the `os`/`io/fs` APIs, and external processes are spawned with `os/exec` using an argument list, never through a shell (`sh -c`).

## 3. Testing (TST)
- TST-1 Every feature ships with tests, and it is not done until they pass.
- TST-2 Every generator has golden-file tests.
- TST-3 Integration tests scaffold a project into a temp dir and run its own pytest and lint.
- TST-4 `go test ./...` and `golangci-lint run` pass before merge.
- TST-5 Every generated project includes passing pytest tests out of the box.
- TST-6 No test calls a real LLM provider. No test uses the network, except integration tests (build tag `integration`), and only to install the generated project's dependencies (e.g. `uv sync`).

## 4. Security (SEC)
- SEC-1 No secrets in the repo; generated projects read configuration from environment variables through pydantic-settings.
- SEC-2 Generated projects ship `.env.example`, never `.env`, and their `.gitignore` excludes `.env`.
- SEC-3 Project names and paths given to the CLI are validated at the command boundary; generators never write outside the target directory.
- SEC-4 The CLI sends no telemetry or data to any external service without explicit opt-in.
- SEC-5 Generators never overwrite an existing file without explicit confirmation or flag.
- SEC-6 The CLI never imports, evaluates or executes the user's Python code itself; the only external process it spawns is `uv`.

## 5. Errors and Logging (ERR)
- ERR-1 Errors are returned with context or logged, never swallowed.
- ERR-2 On failure the CLI exits non-zero and writes the error to stderr.
- ERR-3 Every command works non-interactively (flags only) when there is no TTY.
- ERR-4 Logs and output never contain API keys or secrets.

## 6. Scope (SCP)
- SCP-1 In scope: scaffolding and generating convention-based FastAPI backends for AI/LLM apps.
- SCP-2 In scope: opinionated conventions for LLM-specific parts (provider selection, prompts, tools).
- SCP-3 Out of scope: deploying or hosting the generated apps.
- SCP-4 Out of scope: targets other than Python and FastAPI.
- SCP-5 Generated projects do not depend on aicli at runtime.

## 7. Limits (LIM)
- LIM-1 Supported platforms for the CLI: macOS and Linux. Windows is a future goal and is not supported yet.
- LIM-2 Generated projects declare `requires-python = ">=3.12"`.
