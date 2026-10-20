# aicli

A Rails-style CLI that scaffolds and evolves convention-over-configuration **FastAPI** backends for AI/LLM apps, so data scientists and AI engineers get a consistent project structure instead of ad-hoc model-serving code.

`aicli` is spec-driven (SDD) and built with Go, [Cobra](https://github.com/spf13/cobra) and the [Charm](https://charm.sh) suite (Bubble Tea, Huh, Lipgloss).

## Features

- `aicli new`: scaffold a new FastAPI project with a fixed, conventional layout.
- `aicli add`: add optional AI/LLM capabilities (LLM provider, vector store, queues, observability, ...) to an existing aicli project. Each capability is a subcommand of `add`.
- Home screen: running `aicli` in a terminal shows an interactive menu.
- Interactive when there is a TTY, fully flag-driven (non-interactive) when there is not.

## What gets generated

Generated projects use:

| Area | Tooling |
|------|---------|
| Language | Python (`requires-python = ">=3.12"`) |
| Web framework | FastAPI, served with uvicorn |
| Database | SQLAlchemy (base ORM) + Alembic migrations |
| Configuration | Pydantic + pydantic-settings (environment variables) |
| Packaging | uv |
| Tests / lint | pytest, httpx (test client), ruff |

Every generated project:

- follows **one fixed layout**; there are no flags or config files to change it;
- ships `.env.example` (never `.env`) and a `.gitignore` that excludes `.env`;
- includes passing pytest tests out of the box;
- does not depend on `aicli` at runtime.

Optional capabilities are **opt-in** features or flags and are never part of the default scaffold.

## Install and build

Requires the Go version pinned in `go.mod`. Supported platforms: macOS and Linux (Windows is a future goal).

```bash
go mod download                  # install dependencies
go build -o bin/aicli .          # build the binary
go run .                         # run locally
```

## Usage

```bash
aicli            # home screen (interactive menu)
aicli new        # scaffold a new project
aicli add        # pick a feature to add to the current project
```

Run any command with `--help` for its flags. Errors are written to stderr and the CLI exits non-zero.

## Project structure

```
main.go            binary entry point; calls cmd/root
cmd/root           Cobra root command; registers every command package
cmd/<name>         one flat folder per command (e.g. cmd/new, cmd/add); no nesting
internal/tui       interactive prompts and styled output (the only place that imports Charm)
internal/gen       scaffolding and code generators
templates/         embedded templates for generated Python/FastAPI projects
specs/             one spec per feature (the source of truth)
docs/              project documentation, including constitution.md
```

Command packages contain only the Cobra command, flags and wiring; logic lives in `internal/`.

## Development

```bash
go test ./...                       # all tests
go test ./internal/gen -run TestX   # one test / package
go test -tags integration ./...     # integration tests (scaffold + run the generated project)
go test ./... -update               # regenerate golden files (review the diff)
golangci-lint run                   # lint
gofmt -w .                          # format
```

Integration tests require `uv` and a Python interpreter on `PATH`; they scaffold projects into a temp dir and run the generated project's own pytest and lint. Apart from installing the generated project's dependencies, tests never use the network and never call a real LLM provider.

### Spec-driven workflow

- Read [`docs/constitution.md`](docs/constitution.md) and the active spec in `specs/` before any work.
- Every feature is a spec. No code is written for a feature that has no spec.
- Every feature ships with tests, and it is not done until they pass. Each test's doc comment lists the requirements it covers on a line such as `Covers: FR-013, EC-002`, so spec coverage can be checked with `grep`.
- Generators are covered by golden-file tests.
- `specs/` is read-only during implementation, apart from ticking tasks and Definition-of-Done items.
- Code must pass `gofmt` and `golangci-lint`, and every function, method, struct, interface and exported constant has a doc comment describing its purpose, inputs, outputs/errors and side effects.
- No new dependencies (Go modules or generated-project packages) without team approval.

Current specs:

- `001-aicli-new`: the `aicli new` command
- `002-project-root-detection`: project root detection for next steps
- `003-home-screen`: home menu
- `004-add-command`: the `aicli add` command and the shared feature mechanism

## Constitution highlights

The [constitution](docs/constitution.md) overrides any spec or plan. Key principles:

- **Stack:** Go only; Cobra for commands, Charm only for the TUI; generated projects target Python/FastAPI.
- **Architecture:** flat `cmd/` tree; generated Python comes from embedded templates; code never depends on the host OS (`path/filepath`, `os/exec` with argument lists, never a shell).
- **Security:** no secrets in the repo; project names and paths are validated and generators never write outside the target directory; existing files are never overwritten without explicit confirmation; no telemetry without opt-in; the CLI never executes the user's Python code (the only external process is `uv`).
- **Errors:** errors carry context and go to stderr with exit code 1 and an `Error: ` prefix; output is plain when not on a TTY, `NO_COLOR` is set or `TERM=dumb`; secrets never appear in output.
- **Scope:** in scope are scaffolding and conventions for FastAPI backends for AI/LLM apps (provider selection, prompts, tools); out of scope are deploying or hosting generated apps and targets other than Python/FastAPI.

## License

No license has been specified yet.
