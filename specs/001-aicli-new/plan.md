# Plan: 001-aicli-new

- Status: approved

## Approach

The project is generated in a **hidden staging directory inside the working directory** (same filesystem) and published as `./<name>` with an **atomic no-replace rename** (`renameat2(RENAME_NOREPLACE)` on Linux, `renamex_np(RENAME_EXCL)` on macOS, via `golang.org/x/sys`, pre-approved by STK-6). This way `./<name>` only exists when the project is complete (FR-016, EC-009), and a destination that appears after the initial check is never replaced, not even an empty directory (EC-010). The staging name is a fixed prefix plus a random suffix of fixed length, so it does not depend on `<name>` (EC-012).

The skeleton comes from embedded templates (`embed`). Only the files that need `<name>` (`pyproject.toml`, `README.md`, `app/config.py`) are rendered with `text/template`; the rest are copied verbatim, which avoids clashes between `{{ }}` and the generated Python/Docker code. Output is deterministic: sorted traversal, nothing date-dependent or random in the content, and permissions set explicitly with `Chmod` (0644/0755), independent of the umask (FR-022).

Rejected alternatives:
- **Write directly into `./<name>` with `os.Mkdir` and roll back on failure**: simpler and needs no `x/sys`, but `./<name>` is visible half-built during generation and stays half-built on an uncatchable failure; contradicts FR-016/EC-009.
- **Staging + plain `os.Rename`**: on POSIX it replaces an empty destination directory, violating EC-010.
- **Detect a TTY with `os.ModeCharDevice`**: it would treat `/dev/null` as a TTY and emit ANSI; `golang.org/x/term` is used instead (STK-6).
- **Render every template with `text/template`**: forces escaping braces in the generated code for no benefit.

## Components / Phases

**Phase 0: module bootstrap** (there is no `go.mod` yet)
- `go.mod` with module `github.com/ro-uc-ru/aicli` and `go 1.25` pinned (STK-1; the installed version is 1.25.6). Dependencies: `cobra` v1.10.x and `lipgloss` v1.1.x (project stack; both compatible with 1.25), `golang.org/x/sys` and `golang.org/x/term` (STK-6), and `github.com/muesli/termenv` (already an indirect dependency of Lipgloss; imported directly only in `internal/tui` to force the color profile; approved under STK-4). The latest `x/sys` and `x/term` require Go 1.26, so earlier versions that build with 1.25 are pinned. No other dependency.
- `main.go`: calls `cmd/root` and exits with the code it returns; it prints nothing itself.
- Minimal `.golangci.yml` and the repository `.gitignore`.

**Phase 1: `templates/` (embedded templates)**
- `templates` package with `//go:embed all:skeleton` (the `all:` prefix includes dotfiles such as `.gitkeep`, `.env.example`, `.gitignore`).
- One file per entry of the spec's fixed layout; those that need `<name>` use the `.tmpl` suffix. Content per file:
  - `pyproject.toml`: `name`, `requires-python = ">=3.12"`, 5 runtime dependencies and a `dev` group (`pytest`, `httpx`, `ruff`), all with `>=`; no `[build-system]`; ruff configuration (`select` E, F, I) so that `ruff check .` passes (FR-007, FR-020, LIM-2).
  - `app/config.py`: `Settings` with exactly `APP_NAME`, `DATABASE_URL`, `LOG_LEVEL` (`Literal`), `env_file=".env"` and `extra="ignore"` so that other keys in the user's `.env` do not break startup (FR-005, EC-013).
  - `app/logging.py`: `logging.basicConfig`/dictConfig with a fixed format and the `LOG_LEVEL` level; never logs `DATABASE_URL` (FR-026).
  - `app/errors.py`: handlers for Starlette's `HTTPException`, `RequestValidationError` and `Exception`, all returning `ErrorModel`. A literal message table for 404/405/422/500 and `http.HTTPStatus(...).phrase` for the rest (with a fixed fallback if the code is not standard); the `Exception` handler logs only `type(exc).__name__`, with no traceback or message (FR-027, EC-014).
  - `app/main.py`: `FastAPI(title=settings.APP_NAME)`, sets up logging, registers the handlers and mounts `api/router.py` (FR-004).
  - `app/db.py`: declarative `Base`, engine from `Settings.DATABASE_URL`, `sessionmaker` and a session dependency (FR-006).
  - `app/api/router.py`, `app/api/routers/health.py`, `app/schemas/health.py` (`HealthModel.status: int`), `app/schemas/errors.py` (`ErrorModel`).
  - Empty `app/ai/__init__.py` and the other `__init__.py` files (FR-003, FR-019).
  - `alembic.ini` (`script_location = migrations`, `prepend_sys_path = .`, no `sqlalchemy.url`), `migrations/env.py` (URL from `Settings`, `target_metadata = Base.metadata`), `script.py.mako`, `versions/.gitkeep` (FR-006, FR-021).
  - `tests/conftest.py` (test client with `raise_server_exceptions=False` so the 500 can be asserted), `test_config.py`, `test_health.py`, `test_errors.py` (FR-008, EC-013, EC-014). No network, no LLM.
  - `.env.example`, `.gitignore`, `.dockerignore` with the exact content of FR-009/FR-025; `Dockerfile` (FR-017); `docker-compose.yml` = `services: {}` (FR-018); `README.md` (FR-010).

**Phase 2: `internal/gen` (skeleton rendering)**
- Walks the `embed.FS` in lexical order, renders the `.tmpl` files with `<name>` and returns a sorted list of entries (relative path using `path`, content, kind). It is pure: it does not touch the disk.
- A function that writes the entries into a base directory: creates each directory with `Mkdir` + `Chmod 0755` and each file with `OpenFile` + `Chmod 0644` (paths via `path/filepath`). File creation goes through a minimal injectable interface so tests can force failures (FR-016).

**Phase 3: `internal/scaffold` (validation, staging, commit, rollback)**
- `ValidateName`: empty → missing-name error (FR-012, EC-003); then length ≤ 255 bytes, regex `^[a-z][a-z0-9_-]*$` and PEP 503 normalization (`[-_]+` → `-`) against the FR-007 dependency list (FR-013, EC-001, EC-002, EC-011, EC-012). Each rejection returns a distinct reason. A test checks that the reserved-name list matches the dependencies of the embedded `pyproject.toml`, so there is a single source of truth.
- `Run(ctx, baseDir, name)`: (1) validates; (2) `Lstat` of the destination: if it exists in any form (file, empty or non-empty directory, symlink, even a broken one) → existing-destination error (FR-014, EC-004, EC-008, EC-007); (3) `MkdirTemp` of the staging directory in `baseDir` (not writable → error, EC-005); (4) writes the skeleton; (5) checks `ctx`; (6) no-replace rename; if it fails because the destination appeared → FR-014 error (EC-010). A `defer` removes the staging directory on any error exit. `baseDir` is a parameter (not `os.Getwd`) so tests are parallel-safe.
- Signals: `cmd` creates the `ctx` with `signal.NotifyContext` (SIGINT, SIGTERM); `Run` checks it between steps and, if cancelled, cleans up the staging directory and returns a typed error carrying the signal. If the signal arrives after the rename and before printing, `Run` removes the destination it created itself, so EC-009 holds at any point before the success output.
- The no-replace rename lives in files with build tags (`_linux.go`, `_darwin.go`), the only exception to ARC-8 (STK-6). If the filesystem does not support the operation, it returns an error with context (exit 1, cleanup, no fallback).
- Internal hooks for tests: between the check and the rename, and between writes.

**Phase 4: `internal/tui` (success and error output)**
- `RenderSuccess(name, styled)`: the exact plain text of FR-015. Styled mode uses Lipgloss (first line with a `✓ ` prefix, highlighted commands, dimmed `(optional)`) with a forced color profile, so stripping ANSI and the prefix yields exactly the plain text.
- `RenderError(msg, styled)`: plain message `Error: <reason>` plus continuation lines indented by two spaces (FR-023). Styled: `✗ ` prefix, `Error:` in red bold, dimmed continuation lines; stripping ANSI and the prefix yields exactly the plain text.
- `ShouldStyle(w, env)`: true only if `w` is a TTY (`x/term`), `NO_COLOR` is unset or empty and `TERM != dumb`. It is evaluated on stdout for the success output and on stderr for errors.
- The only package that imports Lipgloss (ARC-7). It spawns no processes.

**Phase 5: `cmd/root` and `cmd/new`**
- `cmd/root`: Cobra root command with `SilenceUsage`/`SilenceErrors`; registers `cmd/new`; an `Execute` function that returns the exit code (1 by default; 128+n for the typed signal error, in which case it prints nothing). It prints errors to stderr with `tui.RenderError`. No logic.
- `cmd/new` (package named differently from `new` so it does not shadow the builtin): its own `Args` — 0 args or `""` → "missing name" (FR-012), more than 1 → usage error (FR-024); its `RunE` calls `internal/scaffold` and, only on success and with no signal, `internal/tui`, and writes to stdout. On failure stdout stays empty and stderr gets the error, styled or plain (FR-023, ERR-2). The FR-024 error adds the continuation line `  Usage: aicli new <name>`.

**Phase 6: tests, quality and final check**
- Unit and golden tests are written together with the code they verify (Phases 1 to 5); the list below is what they must cover as a whole.
- Unit: name validation, `gen`, `scaffold` (rollback with a failing write, destination as file / empty dir / symlink / broken symlink, double run, non-writable cwd, cancellation at each point, race of N goroutines with the same name → exactly one success, an empty destination appearing between the check and the rename), `tui` (success and error: plain, and styled with ANSI and prefix stripped = plain; `NO_COLOR`, `TERM=dumb`, no TTY) and `cmd/new` (stdout/stderr/exit code with and without a TTY, EC-006).
- Golden files under `internal/gen/testdata/golden/` for the whole generated tree with a fixed name (`myapp`) and for the plain success and error outputs (`go test ./... -update` regenerates them; includes the `Dockerfile`, FR-017). A test compares the generated tree with the exact list in the spec's layout, with no missing or extra files, with 0644/0755 permissions, and with two byte-identical runs even under a different umask (FR-022).
- Integration (`//go:build integration`): scaffolds into a temp dir and runs, with `os/exec` (argument list), `uv sync`, `uv run pytest`, `uv run ruff check .` and `uv run alembic upgrade head` (TST-3, TST-5, DoD). It is the only test that uses the network, solely to install dependencies (TST-6). It also covers the risk that `app/logging.py` shadows the standard `logging` module.
- Quality sweep: `gofmt`, `golangci-lint run`, a doc comment on every function, method, struct, interface and exported constant, `go mod tidy` with no diff, and a search confirming no `sh -c`, no hard-coded path separators and no network use outside the integration test (ARC-8, TST-4, TST-6).
- Final check of every item of the Definition of Done in `spec.md`.

## Coverage

| FR / EC | Covered by |
|---|---|
| FR-001 | Phase 5 (`cmd/new`) + Phase 3 (`Run`); `cmd/new` test and golden |
| FR-002 | Phase 1 (embedded tree) + Phase 6 (exact tree test) |
| FR-003 | Phase 1 (`__init__.py`/`.gitkeep` in every folder) |
| FR-004 | Phase 1 (`main.py`, `router.py`, `health.py`, `schemas/health.py`) + generated `test_health.py` |
| FR-005 | Phase 1 (`config.py`, `main.py`) + generated `test_config.py` |
| FR-006 | Phase 1 (`db.py`, `alembic.ini`, `migrations/`) + integration `alembic upgrade head` |
| FR-007 | Phase 1 (`pyproject.toml.tmpl`) + golden |
| FR-008 | Phase 1 (generated tests) + integration `uv run pytest` |
| FR-009 | Phase 1 (`.env.example`, `.gitignore`) + golden; Phase 3 never generates `.env` (tree test) |
| FR-010 | Phase 1 (`README.md.tmpl`) + golden |
| FR-011 | Phase 1 (no optional capability; tree and dependency tests) |
| FR-012 | Phase 5 (`Args`) + Phase 3 (`ValidateName`) |
| FR-013 | Phase 3 (`ValidateName`) |
| FR-014 | Phase 3 (`Lstat` of the destination) |
| FR-015 | Phase 4 (`RenderSuccess`, `ShouldStyle`) + plain-output golden |
| FR-016 | Phase 3 (staging + cleanup `defer`) + Phase 2 (injectable write) |
| FR-017 | Phase 1 (`Dockerfile`), verified by golden only |
| FR-018 | Phase 1 (`docker-compose.yml`) + golden |
| FR-019 | Phase 1 (empty `app/ai/__init__.py`) |
| FR-020 | Phase 1 (ruff config) + integration `ruff check .` |
| FR-021 | Phase 1 (`config.py`, `migrations/env.py`) + `test_config.py` + integration |
| FR-022 | Phase 2 (ordering, explicit `Chmod`, no dates) + determinism and umask test |
| FR-023 | Phase 4 (`RenderError`, `ShouldStyle` on stderr) + Phase 5 (`Execute`, empty stdout on failure) + plain-error golden |
| FR-024 | Phase 5 (`Args`) |
| FR-025 | Phase 1 (`.dockerignore`) + golden |
| FR-026 | Phase 1 (`app/logging.py`, `main.py`) + generated `test_config.py` |
| FR-027 | Phase 1 (`errors.py`, `schemas/errors.py`) + generated `test_errors.py` |
| EC-001 | Phase 3 (`ValidateName`) |
| EC-002 | Phase 3 (`ValidateName`) |
| EC-003 | Phase 5 (`Args`) + Phase 3 (`ValidateName`) |
| EC-004 | Phase 3 (`Lstat` of the destination) |
| EC-005 | Phase 3 (`MkdirTemp` fails in `baseDir`) |
| EC-006 | Phase 5 + Phase 6 (test with and without a TTY) |
| EC-007 | Phase 3 (`Lstat` of the destination) |
| EC-008 | Phase 3 (`Lstat`, not `Stat`) |
| EC-009 | Phase 3 (signals, cleanup of staging and of its own destination) + Phase 5 (exit code 128+n) |
| EC-010 | Phase 3 (no-replace rename) + race and hook tests |
| EC-011 | Phase 3 (PEP 503 normalization + reserved list) |
| EC-012 | Phase 3 (255 limit; fixed-length staging name) |
| EC-013 | Phase 1 (`LOG_LEVEL` as `Literal`) + generated `test_config.py` |
| EC-014 | Phase 1 (`errors.py`, `conftest.py`) + generated `test_errors.py` |

## Constitution check

- STK-1: Go module with the version pinned in `go.mod` (Phase 0).
- STK-2: Cobra for the command tree; Lipgloss for styling (Phases 4 and 5).
- STK-3: the generated `pyproject.toml` declares only the FR-007 dependencies.
- STK-4: no dependency is added beyond Cobra, Lipgloss, `x/sys`, `x/term` and `termenv` (approved).
- STK-5 / SCP-1: the base skeleton includes no optional capability (FR-011).
- STK-6: `x/sys` only in the build-tagged rename files; `x/term` only for TTY detection.
- ARC-1: spec 001 is approved before any code is written.
- ARC-2: `cmd/root` and `cmd/new`, flat folders under `cmd/`.
- ARC-3: `cmd/new` only holds the command, flags and wiring; logic lives in `internal/scaffold`, `internal/gen` and `internal/tui`.
- ARC-4: `cmd/new` imports no other `cmd/` package; only `cmd/root` imports them.
- ARC-5: all generated Python comes from the embedded `templates/`; Go does not build code from strings.
- ARC-6: a single fixed layout, no flag or configuration changes it.
- ARC-7: Lipgloss is imported only in `internal/tui`, for both the success and the error styling.
- ARC-8: paths with `path/filepath` (and `path` inside the `embed.FS`), files with `os`/`io/fs`, `uv` in integration tests via `os/exec` with an argument list, no shell; the build-tagged system calls are the STK-6 exception.
- TST-1 / TST-4: the feature ships with its tests; `go test ./...` and `golangci-lint run` must pass.
- TST-2: golden files for the skeleton and for the plain success and error outputs.
- TST-3 / TST-5: integration with the generated project's pytest and ruff; the project ships passing tests.
- TST-6: no test uses the network except the integration one (dependency install); none calls an LLM provider.
- SEC-1 / SEC-2: configuration through environment variables via pydantic-settings; `.env.example` is generated, never `.env`, and `.gitignore` excludes `.env`.
- SEC-3: the name is validated at the command boundary; the generator only writes into the staging directory and `./<name>`.
- SEC-4: no network call or telemetry in the CLI.
- SEC-5: nothing is ever overwritten; an existing destination is always an error (FR-014).
- SEC-6: the CLI does not execute the user's Python code or spawn processes (only `uv` in integration tests).
- ERR-1 / ERR-2: errors returned with context, `Error: …` on stderr (styled only on a TTY) and a non-zero exit.
- ERR-3: the command works with arguments only; there is no interactive mode.
- ERR-4: neither the CLI nor the generated code logs secrets (`DATABASE_URL` included).
- SCP-5: the generated project does not depend on aicli at runtime.
- LIM-1: macOS and Linux only (build-tagged rename for both).
- LIM-2: `requires-python = ">=3.12"` in the generated `pyproject.toml`.
