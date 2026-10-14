# 002-project-root-detection: Project root detection

- Status: approved
- Owner:  ro-uc-ru
- Date:   2026-10-14

## 1. Context

### Goal (required)
Every command that is not exempt runs only when the current working directory is the root of an aicli project, and fails early, clearly and without touching anything otherwise. `aicli new`, help and version are not affected.

### Why (required)
Every upcoming command (`generate`, `db`, `add`, `destroy`, ...) writes to fixed paths such as `app/models/`. Without a reliable project check, running one in the wrong directory pollutes an arbitrary folder (SEC-3, SEC-5). The check applies by default to every command, so a future command is protected without its spec having to remember it. The layout is fixed (ARC-6), so the project can be recognised without adding any file to the generated tree. This spec is the foundation those commands depend on.

### Optional
- Users / Beneficiaries: data scientists and AI engineers using `aicli` after `aicli new`; authors of future command specs, who get the check without writing it.
- Definitions:
  - Project marker: each of these four paths, relative to a directory: `pyproject.toml`, `alembic.ini`, `app/main.py`, `migrations/env.py`. All four are produced by spec 001 for every project.
  - Project root: a directory in which all four project markers exist, each as a regular file (a symbolic link counts if its target is a regular file).
  - Missing marker: a project marker that does not exist, whose path cannot exist because a parent component is not a directory, or that exists but is not a regular file (after following symbolic links).
  - Inspection failure: any other error when inspecting a project marker (e.g. permission denied, symbolic link loop, I/O error). It is never a missing marker.
  - Exempt command: a command that does not perform the check. Initially: `new`, `help`, `version`, and the commands Cobra registers by default, `completion` (with its subcommands) and the hidden `__complete`, with the flag forms (`-h`, `--help`, `-v`, `--version`). `version` and its flags are preventive: they are exempt even if not defined yet. Every other command is checked.
- Notes for the plan (non-normative):
  - The check is a single reusable function in `internal/`, applied globally from the root command (Cobra `PersistentPreRunE`) with the list of exempt commands, so that no command can skip it (ARC-3). It returns a distinguishable error type so callers can tell "not a project" from an inspection failure.
  - The wiring is verified end to end with a test command registered only in tests; it needs no declaration, since it is checked by default.
  - The current working directory is injectable in the function so tests can simulate EC-008 without deleting a real directory.
  - The list of exempt commands lives in one place in the code.
  - `aicli` with no arguments is not runnable today (it prints help), so it does not reach the check; the spec that makes it runnable must declare it exempt.
  - Each command spec uses the project root given by the check as the base of its paths (SEC-3).

## 2. Functional Requirements

- FR-001 (ubiquitous)    The CLI shall consider the current working directory a project root if and only if all four project markers exist in it as regular files; it shall look only at the current working directory and never at its parents or children.
- FR-002 (event-driven)  WHEN any command other than an exempt one is run, the CLI shall perform the check of FR-001 after Cobra has parsed its flags and arguments and before running any of the command's own logic, and shall run that logic only if the current working directory is a project root.
- FR-003 (unwanted)      IF the current working directory is not a project root and no inspection failure occurred (FR-006), THEN the CLI shall fail as in ERR-5 with the message below, where `<missing>` lists, comma-separated and in the order of the Definitions, the missing markers:
  ```
  Error: not inside an aicli project
    Missing: <missing>
    Run this command from the root of a project created with `aicli new`.
  ```
- FR-004 (unwanted)      IF the check of FR-001 fails for any reason, THEN the CLI shall create, modify or delete no file or directory, and spawn no external process.
- FR-005 (ubiquitous)    The check of FR-001 shall be read-only and shall only inspect whether each project marker exists and is a regular file; it shall never read the contents of any file nor execute any user code (SEC-6).
- FR-006 (unwanted)      The check shall resolve the current working directory before inspecting any project marker. IF the current working directory cannot be determined, or any project marker suffers an inspection failure, THEN the CLI shall fail as in ERR-5 and shall not write the message of FR-003, even if other markers are missing (ERR-1). It shall report only the first error, markers taken in the order of the Definitions, with one of these single-line messages, where `<path>` is the marker as written in the Definitions (relative, with `/`) and `<cause>` is the text of the system error:
  ```
  Error: cannot determine the current directory: <cause>
  Error: cannot inspect <path>: <cause>
  ```
- FR-007 (ubiquitous)    The CLI shall not perform the check of FR-001 for exempt commands, which shall work in any directory; `--help` and `help <command>` are exempt for every command. A spec that adds a command which needs an exemption shall declare it as exempt from the check of spec 002; the list of exempt commands shall live in one place in the code, and a test shall pin its exact content.
- FR-008 (ubiquitous)    Every project generated by `aicli new` shall be a project root as defined in FR-001 as soon as the command finishes, with no further step.
- FR-009 (ubiquitous)    On success, the check shall provide the project root as the absolute path of the current working directory as given by the operating system, without resolving symbolic links, for the command to use as the base of its paths.

## 3. Edge Cases

- EC-001 IF the current working directory is inside a project but is not its root (e.g. `app/services`), THEN the CLI shall treat it as not a project root and fail as in FR-003; no parent directory is consulted. (FR-001)
- EC-002 IF a project marker exists but is a directory (e.g. `app/main.py/`), THEN the CLI shall treat that marker as missing. (FR-001)
- EC-003 IF a project marker is a symbolic link whose target is a regular file, THEN the CLI shall treat it as present; IF it is a broken symbolic link or points to a directory, THEN it shall treat it as missing. (FR-001)
- EC-004 IF the current working directory is empty, THEN the CLI shall fail as in FR-003 with all four project markers in `Missing`. (FR-003)
- EC-005 IF only some project markers exist (e.g. `migrations/env.py` was deleted), THEN the CLI shall fail as in FR-003 listing exactly the missing ones. (FR-003)
- EC-006 IF the current working directory is a project root that is nested inside another project root, THEN the CLI shall treat it as a project root without consulting the outer one. (FR-001)
- EC-007 IF a parent directory of a marker (e.g. `app/` or `migrations/`) is not searchable, THEN the CLI shall fail as in FR-006 and not as in FR-003. (FR-006)
- EC-008 IF the current working directory cannot be determined, THEN the CLI shall fail as in FR-006 with the first message of FR-006, without inspecting any marker and without claiming the directory is not a project. (FR-006)
- EC-009 IF an extra file, folder or marker content differs from the generated one (e.g. edited `main.py`, added directories), THEN the CLI shall still treat the directory as a project root, since only the existence of the four project markers is checked. (FR-001, FR-005)
- EC-010 IF a parent component of a marker is a regular file (e.g. `app` is a file, so `app/main.py` cannot exist), THEN the CLI shall list that marker as missing and fail as in FR-003. (FR-001, FR-003)
- EC-011 IF a project marker is a circular symbolic link, THEN the CLI shall fail as in FR-006 and not as in FR-003. (FR-006)
- EC-012 IF one marker is missing and another suffers an inspection failure (e.g. `pyproject.toml` was deleted and `app/` is not searchable), THEN the CLI shall fail as in FR-006 naming the first inspection failure, and shall not write the message of FR-003. (FR-006)
- EC-013 IF a project marker exists but is neither a regular file nor a symbolic link to one (e.g. a FIFO), THEN the CLI shall treat it as missing. (FR-001)
- EC-014 IF the command given is unknown, THEN the CLI shall fail with Cobra's own error for unknown commands and shall not perform the check. (FR-002)

## 4. Scope

### In scope
- The definition of project root and the check against the current working directory.
- The rule that every command is checked by default, and the list of exempt commands.
- The fixed error message and exit code when the check fails.
- The guarantee that generated projects pass the check.

### Out of scope
- Searching parent directories (Rails-style): chosen against for simplicity and predictability; a future spec may add it.
- A `.aicli` marker file or any change to the generated tree: would amend spec 001 (FR-002); layout detection needs none.
- A command or flag to print the detected root: no consumer yet; adds public surface.
- A flag to point at another directory (`--path`, `--project`): no convention requires it yet.
- Validating the contents or version of the layout (e.g. parsing `pyproject.toml`): would need a TOML dependency (STK-4) and is not needed to protect the paths.
- Repairing or diagnosing a broken project, and checking `uv`, Python or git: a future `doctor` spec.
- The commands that rely on this check (`generate`, `db`, `add`, `destroy`, ...): each has its own spec.
- Case sensitivity of marker names follows the filesystem; it is not normalised.

- [x] FR-001..FR-009 covered by tests
- [x] EC-001..EC-014 covered by tests
- [x] FR-002, FR-003, FR-004 and FR-007 verified end to end with a test command registered only in tests and checked by default
- [x] A directory generated by `aicli new` passes the check in a test, so a layout change in spec 001 breaks it (FR-008)
- [x] The fixed message of FR-003 is covered by a golden or exact-match test, in plain form
- [x] The message of FR-006 for permission denied is covered by an exact-match test, in plain form; the other causes check the prefix and the path
- [x] A test pins the exact list of exempt commands (FR-007)
- [x] The check is verified to write nothing to disk in the failure case
- [x] `go test ./...` and `golangci-lint run` pass (TST-4)
 Changelog
- 2026-10-14 QA-1: every command is checked by default; exempt commands replace "project-requiring" (Definitions, FR-002, FR-007); test command registered only in tests (DoD); global wiring in the plan notes.
- 2026-10-14 QA-2: Definitions of missing marker and inspection failure; FR-006 rewritten; EC-010, EC-011.
- 2026-10-14 QA-3: `--help` and `help <command>` exempt for every command (FR-007); check runs after Cobra parsing (FR-002).
- 2026-10-14 QA-4: FR-006 takes priority over FR-003 and reports only the first error; EC-012.
- 2026-10-14 QA-5: single-line message templates of FR-006.
- 2026-10-14 QA-6: FR-003 and FR-006 cite ERR-5 (added to the constitution).
- 2026-10-14 QA-7: FR-006 resolves the working directory first; EC-008 rewritten; injectable directory in the plan notes.
- 2026-10-14 QA-8: FR-002 rewritten; FR-009 added.
- 2026-10-14 QA-9: EC-009 now cites FR-001 and FR-005.
- 2026-10-14 QA-10: EC-006 reworded.
- 2026-10-14 QA-11: EC-013 added; case sensitivity added to Out of scope.
- 2026-10-14 QA-12: FR-003 trimmed to the message, delegating exit code, stdout and stderr to ERR-5.
- 2026-10-14 EC-014 added (unknown command), from the discussion of QA-1.
- 2026-10-14 QA-13: `completion` and `__complete` (Cobra defaults) added to the exempt commands (Definitions).
- 2026-10-14 QA-14: EC-010 now cites FR-001 and FR-003.
- 2026-10-14 QA-15: `version` and its flags declared preventive (Definitions).
- 2026-10-14 Cleanup before approval: the retired EC-009 removed and edge cases renumbered consecutively (EC-001..EC-014); the IDs in this changelog follow the new numbering.
