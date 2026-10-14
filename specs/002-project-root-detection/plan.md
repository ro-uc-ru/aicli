# Plan: 002-project-root-detection

- Status: approved

## Approach

The check is a single function in a new package `internal/project`. It resolves the working directory through an injectable `getwd` (production: `os.Getwd`, which returns the OS path without resolving symbolic links, FR-009), then inspects the four markers **in the order of the Definitions** with `os.Stat` (follows symbolic links, reads no contents). Each result is classified as present (regular file), missing (does not exist, a parent component is not a directory, or not a regular file), or inspection failure (any other error). The first inspection failure stops the check and wins over any missing marker (FR-006, EC-012); otherwise, if any marker is missing, it returns a typed "not a project" error listing them; otherwise it returns the root. Three distinct error types let callers tell the cases apart.

The check is wired once, globally, as the root command's `PersistentPreRunE`, with `cobra.EnableTraverseRunHooks` turned on so that a future command defining its own persistent hook cannot shadow the root's (ARC-3, "no command can skip it"). Cobra already handles `--help`/`-h`, `--version`, unknown commands and argument validation before persistent hooks run, which gives the order FR-002 asks for and EC-014 for free. The hook skips the check when the executed command's top-level ancestor (the child of root) is in the exempt list; otherwise it runs the check and, on success, stores the root in the command's context for its `RunE` to read (FR-009).

Rejected alternatives:
- **Opt-in call per command (each `RunE` calls the check)**: what the spec explicitly replaced in QA-1; a forgotten call leaves a command unprotected.
- **Check in `root.Execute` before `cmd.ExecuteContext`, using `cmd.Find`**: runs before Cobra parses flags and arguments (breaks FR-002's order) and would need to replicate Cobra's handling of `--help`, `help`, unknown commands and completion.
- **Root `PersistentPreRunE` without `EnableTraverseRunHooks`**: Cobra runs only the nearest persistent hook, so a future command with its own would silently bypass the check.
- **Wrapping every registered command's `RunE` while building the tree**: misses commands added after construction (including the test-only command) and duplicates what Cobra's hook already offers.
- **Classifying "parent component is not a directory" (EC-010) by comparing the error with `syscall.ENOTDIR`**: a single call, but it ties the classification to a system error number (ARC-8); `Stat`-ing the parent components with `os` only is chosen instead. The extra calls happen only on the error path.
- **Passing the root through a package-level variable**: hidden global state; the command context is per-run and test-safe.

## Components / Phases

**Phase 1: `internal/project` (new package: the check)**
- The marker list as one ordered constant set, written relative with `/` exactly as in the Definitions; joined to the working directory with `filepath.FromSlash` + `filepath.Join` (ARC-8).
- The check function: takes the injectable `getwd`; returns the root (absolute path exactly as `getwd` gave it) or one of three error types:
  - a working-directory error carrying the cause → message `cannot determine the current directory: <cause>`, where `<cause>` is unwrapped like the inspection one (on Linux `os.Getwd` returns an `*os.SyscallError` whose text starts with `getwd: `); no marker is inspected (EC-008);
  - an inspection error carrying the marker path and the cause → `cannot inspect <path>: <cause>`, where `<cause>` is the underlying system error text (unwrapped from `*fs.PathError` or `*os.SyscallError` by one shared helper, so it reads e.g. `permission denied`);
  - a "not a project" error carrying the ordered missing markers → the three-line text of FR-003, with continuation lines unindented, so that `tui.RenderError` adds the `Error: ` prefix and the two-space indentation (as `cmd/new` already does for FR-024).
- Classification per marker: `Stat` ok and regular → present; `Stat` ok and not regular (directory, FIFO, …) → missing (EC-002, EC-003, EC-013); `fs.ErrNotExist` (includes broken symbolic links) → missing (EC-003, EC-004, EC-005); any other error → for markers with parent components (`app/main.py`, `migrations/env.py`), `Stat` each parent component from the working directory down: if one exists and is not a directory, the marker is missing (EC-010); otherwise (permission denied, symbolic link loop, I/O) it is an inspection failure carrying the marker's original error, and the check stops (EC-007, EC-011, EC-012).
- The exempt-command list (`new`, `help`, `version`, `completion`, `__complete`) as one unexported list plus an exported predicate that takes a top-level command name; the only place the list exists (FR-007).
- Context helpers: one to store the detected root in a `context.Context` and one to read it back, for command specs to use as the base of their paths (FR-009).
- The package only stats paths: it opens, writes, creates or removes nothing and spawns no process (FR-004, FR-005).

**Phase 2: `cmd/root` (wiring)**
- `NewRoot` gains the root `PersistentPreRunE`; `cobra.EnableTraverseRunHooks` is turned on once, in the package `init()`, so that building a root has no global side effect. The hook finds the executed command's top-level ancestor, asks `internal/project` whether it is exempt and, if not, runs the check and sets the root on the command's context. No logic beyond that call (ARC-3).
- `getwd` is injectable: an unexported constructor takes it; `NewRoot` keeps its current signature and passes `os.Getwd`, so `main.go` and `cmd/new` do not change.
- Errors from the hook flow through the existing `Execute` path: `tui.RenderError` on stderr, styled only when stderr allows it, empty stdout, exit 1 (ERR-5). No change to `internal/tui`: `RenderError` already produces the styled form of `designs/error-output.md` (✗, red bold `Error:`, dimmed continuation lines).

**Phase 3: tests**
- Unit (`internal/project`), each case in its own temp dir built by the test: full project; empty dir; each subset of markers; marker as directory; symlink to file / to directory / broken / circular; FIFO; `app` as a regular file; `app/` with mode 000 (skipped as root); one missing plus one unreadable (EC-012, order); nested project; subdirectory of a project; edited marker contents and extra files; `getwd` failing (no marker inspected, checked with a `getwd` stub and a directory whose markers would otherwise fail); returned root equals `getwd`'s value unchanged when that path goes through a symbolic link (FR-009).
- FR-008: a test renders and writes the real skeleton with `internal/gen` into a temp dir and runs the check on the generated project directory; a layout change in spec 001 that drops a marker breaks it.
- Exempt list: a test pins its exact content (FR-007).
- End to end (`cmd/root`): a test command registered only in the test, after `NewRoot`, whose `RunE` records that it ran and the root it read from the context; one variant also defines its own `PersistentPreRunE` to prove the check still runs. Cases: outside a project → exit 1, empty stdout, golden of the plain FR-003 stderr (`cmd/root/testdata/`, regenerated with `-update`), `RunE` never ran, and a before/after snapshot (paths, modes, sizes, mtimes) of the working directory proves nothing was written (FR-004); permission denied → exact plain message of FR-006; circular symlink and `getwd` failure → prefix and path; inside a project → `RunE` ran with the right root.
- Exempt commands outside a project: `new <name>`, `help`, `help <test command>`, `<test command> --help` and `-h`, `completion bash`, `__complete` all succeed without the check; an unknown command gets Cobra's error (EC-014).
- Every test carries its `Covers:` line.

**Phase 4: quality and final check**
- `gofmt`, `golangci-lint run`, doc comments on every new function, type and constant, no new dependency (`context` is standard library), `go test ./...` green.
- Tick the Definition of Done items in `spec.md` that the final task verifies.

## Coverage

| FR / EC | Covered by |
|---|---|
| FR-001 | Phase 1 (check, cwd only, four markers) + Phase 3 unit |
| FR-002 | Phase 2 (root `PersistentPreRunE`, traverse hooks) + Phase 3 end to end |
| FR-003 | Phase 1 ("not a project" error, ordered list) + Phase 2 (`RenderError`, exit 1) + golden |
| FR-004 | Phase 1 (stat only) + Phase 2 (hook fails before `RunE`) + Phase 3 snapshot test |
| FR-005 | Phase 1 (`Stat` only, no reads, no processes) + Phase 3 (EC-009 test) |
| FR-006 | Phase 1 (cwd first, first inspection failure wins, cause unwrapped) + Phase 3 exact-match test |
| FR-007 | Phase 1 (single exempt list + predicate) + Phase 2 (top-level ancestor) + Phase 3 pin and exempt tests |
| FR-008 | Phase 3 (skeleton from `internal/gen` passes the check) |
| FR-009 | Phase 1 (root = `getwd` value, no symlink resolution; context helpers) + Phase 2 (context set) + Phase 3 |
| EC-001 | Phase 1 (no parent lookup) + Phase 3 |
| EC-002 | Phase 1 (non-regular → missing) + Phase 3 |
| EC-003 | Phase 1 (`Stat` follows links; broken → not exist) + Phase 3 |
| EC-004 | Phase 1 + Phase 3 (all four listed) |
| EC-005 | Phase 1 (ordered missing list) + Phase 3 |
| EC-006 | Phase 1 (cwd only) + Phase 3 |
| EC-007 | Phase 1 (permission denied → inspection failure) + Phase 3 |
| EC-008 | Phase 1 (cwd error before any marker) + Phase 2 (injectable `getwd`) + Phase 3 |
| EC-009 | Phase 1 (existence only) + Phase 3 |
| EC-010 | Phase 1 (parent component `Stat`: not a directory → missing) + Phase 3 |
| EC-011 | Phase 1 (loop → inspection failure) + Phase 3 |
| EC-012 | Phase 1 (inspection failure wins, first in order) + Phase 3 |
| EC-013 | Phase 1 (FIFO → missing) + Phase 3 |
| EC-014 | Phase 2 (Cobra resolves the command before hooks) + Phase 3 |

## Constitution check

- ARC-1: spec 002 is approved before any code.
- ARC-2 / ARC-4: no new command folder; only `cmd/root` changes, and it imports only `internal/` packages besides the commands it registers.
- ARC-3: `cmd/root` only wires the hook; the check, the exempt list and the root-in-context helpers live in `internal/project`.
- ARC-6: the project is recognised from the fixed layout; nothing is added to the generated tree.
- ARC-7: `internal/project` imports no Charm library; styling stays in `internal/tui`.
- ARC-8: marker paths built with `path/filepath`; filesystem access and error classification only through `os`/`io/fs` (no system error numbers).
- STK-2 / STK-4: Cobra for the hook; no new dependency.
- SEC-3: commands get a validated root to use as the base of their paths.
- SEC-5 / SEC-6: the check writes nothing, reads no file contents and executes nothing.
- ERR-1: every inspection error is returned with the marker and the cause; none is swallowed or turned into "missing".
- ERR-2 / ERR-5: failures go through `root.Execute`: plain `Error: …` with two-space continuation lines, styled only on a TTY without `NO_COLOR`/`TERM=dumb`, empty stdout, exit 1.
- ERR-3: the check is non-interactive.
- TST-1 / TST-2 / TST-4: unit, end-to-end and golden tests; `go test ./...` and `golangci-lint run` pass.
- TST-6: no network, no LLM.
- LIM-1: behaviour verified on macOS and Linux (permission tests skipped when running as root).
