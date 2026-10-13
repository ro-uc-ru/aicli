# aicli

## Purpose
A Rails-style CLI that scaffolds and evolves convention-over-configuration FastAPI backends for AI/LLM apps, so data scientists and AI engineers get a consistent project structure instead of ad-hoc model-serving code. Spec-driven (SDD) command-line tool built with Go, Cobra and the Charm suite.

## Rules
- Read docs/constitution.md and the active spec in specs/ before any work.
- Every feature is a spec. Do not write code for a feature that has no spec in specs/.
- Follow standard Go style: code must pass gofmt and golangci-lint.
- Optional capabilities (LLM provider, vector store, queues, observability, etc.) are opt-in features or flags. Never make them part of the default scaffold.
- Generated projects follow one fixed layout. Do not add configuration knobs where a convention exists.
- Document every function, method, struct, interface and exported constant with a doc comment that states its purpose, each parameter (in), each return value and error (out), and side effects, as in a C API reference.
- Every feature ships with tests, and a feature is not done until they pass.
- Keep the command tree flat: one folder per command or subcommand directly under cmd/, no nested folders, no unrelated commands sharing a package. Logic lives in internal/, not in cmd/.
- Do not add dependencies without asking.

## Stack
- Language:   Go (version pinned in go.mod)
- Framework:  Cobra (CLI), Charm: Bubble Tea, Huh, Lipgloss (TUI/prompts/styling; Charm Log or other Charm libs only if a spec requires them)
- DB:         none (the CLI is stateless; it generates projects that use SQLAlchemy)
- Tests:      go test (unit + golden files), teatest (TUI), integration tests behind the `integration` build tag
- Lint:       golangci-lint, gofmt

### Generated project target
The projects aicli generates use: Python, FastAPI, SQLAlchemy (base ORM), Alembic, Pydantic + pydantic-settings, uv (package manager), pytest. The LLM provider and any extras are selected per feature.

### Modules and dependencies
- main.go        <- binary entry point; calls cmd/root. Depends on: cmd/root
- cmd/root       <- Cobra root command; registers every command package under cmd/. Depends on: cobra
- cmd/<name>     <- one flat folder per command or subcommand (e.g. cmd/new, cmd/generate, cmd/generatemodel, cmd/dbmigrate). Nesting is forbidden: never cmd/a/b/c. Each folder owns its command, flags and tests. Depends on: cobra, internal/*
- internal/tui   <- interactive prompts and styled output. Depends on: bubbletea, huh, lipgloss
- internal/gen   <- scaffolding and code generators. Depends on: text/template, embed
- templates/     <- embedded templates for generated Python/FastAPI projects. Depends on: embed
- specs/         <- one spec per feature (SDD source of truth)
- docs/          <- project documentation: constitution.md (approved principles and hard limits)

## Commands
```bash
go mod download                   # install dependencies
go run .                          # run locally
go build -o bin/aicli .          # build binary
go test ./...                     # all tests
go test ./internal/gen -run TestX # one test / package
go test -tags integration ./...   # integration tests (scaffold + run generated project)
go test ./... -update             # regenerate golden files (review the diff)
golangci-lint run                 # lint
gofmt -w .                        # format
```

### Setup notes
- `go test -tags integration ./...`: requires `uv` and a Python interpreter on PATH; the tests scaffold projects into a temp dir and run the generated project's own pytest and lint.
