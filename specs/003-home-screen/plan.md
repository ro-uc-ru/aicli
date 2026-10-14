# Plan: 003-home-screen

- Status: approved

## Approach

The root command becomes runnable: its `RunE` hands the executed root to a new package `internal/home`, which turns the Cobra tree into a plain list of entries and decides the mode; `internal/tui` draws it, either as static text or as a Bubble Tea program. `internal/tui` never sees Cobra types: it receives entries (path, short description, and a function that produces the command help on demand), so all Charm code stays in `internal/tui` (ARC-7) and all Cobra knowledge stays in `internal/home`.

Key decisions:
- **Entries are read from the tree on every run** (FR-004). `internal/home` walks `root.Commands()` depth-first, pre-order, in Cobra's order (FR-003). A command is listed if `IsAvailableCommand()` is true, it is not hidden, and it is not one of the three Cobra built-ins the spec's Definitions exclude (`help`, `completion`, `__complete`); an excluded command excludes its whole subtree, so `completion bash` is not listed (Definitions, QA-18). Those three Cobra built-in names are the only command names in the feature's code. Grouping commands are listed because `IsAvailableCommand` is true for a command with available subcommands.
- **Help is produced in-process and on demand** (FR-008, SEC-6). Opening an entry calls that command's own Cobra help function with its out and err writers pointed at buffers for the duration of the call, then restored. Before the call, the command's default help flag is initialised (Cobra adds `-h, --help` only to the command it executes), so the text equals `aicli <path> --help`. Anything Cobra writes to err while rendering (it reports template errors there) becomes an FR-016 error.
- **The list is drawn by our own small model; only the detail view uses `bubbles/viewport`.** A pure layout function takes the terminal size and the view and returns which parts fit (banner, two indicator lines, footer, list or help area height). It applies EC-003's hiding order and also gives the row-to-entry mapping that mouse clicks use.
- **Signals**: Bubble Tea runs with the run context and with its own signal handler turned off. `main.go` stays the only signal handler. When the context is cancelled, Bubble Tea restores the terminal and returns, and `internal/home` returns the context's cause (the existing `*scaffold.SignalError`). The existing `root.run` path then exits with 128+n and prints nothing (EC-014). `Ctrl+C` in raw mode reaches the model as a key (FR-010).
- **Exemption from spec 002**: the root's own name (`aicli`) is added to the single exempt list in `internal/project`. The existing top-level lookup already yields the root itself when the root is executed, so `checkProject` does not change.
- **Unknown arguments**: the root keeps Cobra's default argument validation (`Args` unset). For a root command, Cobra still answers `unknown command "x" for "aicli"` with suggestions, so EC-008's behaviour is unchanged.

Rejected alternatives:
- **`bubbles/list` for the list view**: its built-in filtering, pagination, status bar and item delegate would have to be switched off or worked around to get the exact `> ` prefix, the FR-020 indicator lines, the click-to-row mapping and EC-007's "unlisted keys do nothing". A list of a few dozen lines is simpler to own.
- **Running `aicli <path> --help` as a subprocess to get the help**: forbidden (SEC-6, FR-014).
- **Producing every help text at startup**: one broken help would stop the whole screen. Producing it when the entry is opened ties the failure to that entry (FR-016) and keeps startup cheap.
- **Building entries inside `internal/tui`**: puts Cobra in the rendering package and makes the TUI tests depend on a command tree. Plain entries keep the model testable with fake help texts.
- **Exempting the root structurally (`!c.HasParent()` in `checkProject`)**: that would create a second place that decides exemptions, against spec 002 FR-007 ("one place").
- **`Args: cobra.NoArgs` on the root**: it drops Cobra's "Did you mean this?" suggestions for unknown commands, which would change EC-008's existing behaviour.
- **Huh select for the list**: it has no click-to-open and no scrollable detail view.

## Components / Phases

**Phase 0: baseline before the change**
- Record the current plain `aicli --help` output as a golden file in `cmd/root/testdata/`. EC-008's test compares the new root help to this baseline. It must be captured before the root becomes runnable.

**Phase 1: dependencies (`go.mod`)**
- Add `github.com/charmbracelet/bubbletea` (v1 line, compatible with the existing `lipgloss v1.1.0`) and `github.com/charmbracelet/bubbles` (viewport only), which the spec approves.
- Test only: `github.com/charmbracelet/x/exp/teatest`, which AGENTS.md lists in the stack.
- Promote `github.com/charmbracelet/x/ansi` from indirect to direct. It is already in the module graph through Lipgloss and provides width-aware truncation, wrapping and ANSI stripping.
- No other module is added. All four were approved by the team on 2026-10-14 (STK-4).

**Phase 2: `internal/project` (exemption)**
- Add the root's name to `exemptCommands`. Update the doc comment to say the entry stands for the root run with no arguments (FR-001).

**Phase 3: `internal/tui` (rendering and the interactive program)**
- *Mode helpers*: a new `IsInteractive(in, out, getenv)` returns true only when both are `*os.File` terminals and `TERM` is not `dumb`, using `golang.org/x/term` as `ShouldStyle` already does. The existing `ShouldStyle(out, getenv)` decides styling (FR-005, FR-012, FR-013, EC-009).
- *Entry type*: path, short description, and a help producer returning `(string, error)`. This is the only interface between `internal/home` and `internal/tui`.
- *Banner*: a constant with the six-line `AICLI` lettering from `designs/home-screen.md`, its display width, and the two fixed description lines (FR-002). Narrow form: the single line `AICLI` (EC-002).
- *Static renderer*: a pure function of the entries. It writes the banner, a blank line, then one line per entry: the path padded to the widest path by display width (EC-010), two spaces, the short description, or the path alone when there is none (EC-005). With no entries it writes `No commands available.` (EC-012). It is always plain and ignores width (FR-012).
- *Styles*: one constructor builds the styles of the design's colour table on a private renderer forced to ANSI256, as `RenderSuccess` does. When not styled, every style is a no-op, so the text is identical and only ANSI sequences differ (FR-013).
- *Layout*: a pure function `(width, height, view, content length) → visible parts and the list or help area height`. Parts are, top to bottom: banner (list view) or header (detail view), the indicator line above, the area, the indicator line below, and the footer. When the area would get fewer than one row, it hides the banner first, then both indicator lines, then the footer, then (detail view) the header (EC-003, QA-18). It also maps a screen row to an entry index or to "no entry" (FR-007, EC-004).
- *List view model*: pointer index (starts at 0) and scroll offset.
  - Keys: `↓`/`j`, `↑`/`k` move the pointer and stop at the ends; `Home`/`End` jump to the first or last entry; `Enter` opens the entry (FR-006, FR-007). `q` or `Ctrl+C` quits (FR-010). Any other key is ignored (EC-007).
  - After any keyboard move the offset is adjusted so the pointer is visible. The mouse wheel changes only the offset, clamped (FR-011).
  - A left click on an entry row moves the pointer there and opens the entry. Clicks elsewhere and all motion events are ignored (FR-007, EC-004; FR-018 is retired).
  - Each entry is drawn as `> ` or two spaces, then the padded path, then the description, truncated to the width with `…` by display width and never wrapped (EC-002, EC-010).
  - When styled: the parent prefix of a subcommand entry is dimmed, the selected row gets the highlight style, and all other rows get the normal style (FR-013).
  - With no entries it draws `No commands available.`; only `q` and `Ctrl+C` act, and the footer shows only `q quit` (EC-001).
- *Detail view model*: a header `aicli › <path>`, truncated by EC-002 (FR-019), and a `viewport` holding the help.
  - The help text is wrapped to the width by display width; lines are never truncated. Lines that fit are untouched, so stripping ANSI gives exactly the help when the terminal is wide enough (FR-008, EC-006).
  - When styled, top-level heading lines (a line at column 0 ending in `:`) and flag tokens are emphasised by adding ANSI sequences only.
  - The viewport's default key map is replaced by exactly `↑`/`k`, `↓`/`j`, `PgUp`, `PgDn`. The model handles `Home`/`End`. Viewport mouse handling is off (FR-009, EC-007).
  - Opening an entry always resets to the top with that entry's help (EC-011).
  - `Esc`, `←` and `Backspace` go back to the list with the pointer unchanged (FR-010).
- *Mouse capture*: the program starts with cell-motion mouse reporting. Entering the detail view issues Bubble Tea's disable-mouse command, and returning to the list issues the enable command again (FR-009).
- *Indicators and footer*: the indicator line above shows `↑ more` when the offset is above 0, and the line below shows `↓ more` when content is hidden below; otherwise the line is blank (FR-020). Footer: list `↑/↓ move · enter open · q quit`, detail `esc back · ↑/↓ scroll · q quit` (FR-017).
- *Resize*: on a window-size message the layout is recomputed. The list keeps its pointer and makes it visible. The detail view remembers its first visible line as a position in the unwrapped help (source line and wrap segment), re-wraps, restores that line, and clamps it so the end of the text is not above the bottom (FR-015).
- *Runner*: `RunHome(ctx, entries, in, out, styled)` builds the program with alternate screen, mouse cell motion, the given input and output, the context, and no signal handler.
  - If opening an entry fails, the model stores the error and quits; the runner returns it after Bubble Tea has left the alternate screen (FR-016).
  - A Bubble Tea start or run error is returned wrapped as `cannot start the home screen: <cause>` (FR-016, ERR-1).
  - A context cancellation returns `context.Cause(ctx)` (EC-014).
  - Quit by key returns nil (FR-010).
- An exported model constructor (`entries, styled, initial size`) exists so `teatest` can drive the model without the runner.

**Phase 4: `internal/home` (new package: tree → entries, mode choice)**
- *Entries from a root*: the depth-first walk and filter described in the Approach. Path = the command path without the root's name (FR-003, FR-004). The help producer:
  1. initialises the command's default help flag;
  2. points its out and err writers at buffers;
  3. calls its help function;
  4. restores the writers to unset, so they inherit from the root again (`run` sets writers only on the root);
  5. returns the text, or `cannot show help for <path>: <first line written to err>` (FR-008, FR-016).
- *Show(ctx, root, in, out, getenv)*:
  - If `tui.IsInteractive` is true, call `tui.RunHome` with `tui.ShouldStyle(out, getenv)`.
  - Otherwise, build the static text, check that the context is not cancelled (return its cause if it is), and write the text to `out` in one write (FR-012, EC-009, EC-014).
  - Write errors are returned with context (ERR-1).
- The package imports Cobra and `internal/tui` only. It opens no file, spawns nothing and uses no network (FR-014).

**Phase 5: `cmd/root` (wiring)**
- The root gets a `RunE` that only calls `home.Show` with the command's context, `InOrStdin`, `OutOrStdout` and `os.Getenv` (ARC-3). `Args` stays unset (EC-008). `NewRoot`, `Execute` and `run` keep their signatures. `main.go` does not change.
- Errors go through the existing `run` path: `Error: <cause>` on stderr, empty stdout, exit 1. A `*scaffold.SignalError` gives 128+n and no output (FR-016, EC-014, ERR-5).

**Phase 6: tests (each with its `Covers:` line)**
- `internal/home` unit:
  - The walk on test trees: nested subcommands in pre-order; a grouping command; hidden, deprecated, `help`, `completion` (with subcommands) and `__complete` excluded; an empty tree; a command without a short description; CJK and emoji names (FR-003, FR-004, EC-001, EC-005, EC-010).
  - The help producer equals the output of running `<path> --help` through Cobra on the same tree, for a leaf, a subcommand and a grouping command (FR-008).
  - A command with a broken help template gives the FR-016 error.
  - The writers are restored afterwards.
  - Static `Show` with buffers produces no ANSI or control bytes. A pre-cancelled context returns the cause and writes nothing (FR-012, EC-014).
- `internal/tui` unit:
  - The layout function over a table of sizes: hiding order, at least one row always left, row mapping (EC-003, EC-004, FR-020).
  - Truncation and padding by display width, including wide characters (EC-002, EC-010).
  - Narrow banner (EC-002).
  - Static renderer golden files with a fixed test entry list and with an empty list (FR-012, EC-012).
- `internal/tui` teatest (external test package, with entries built by `internal/home` from a test tree so the flow runs end to end):
  - Initial pointer.
  - Moving with the arrow keys, `j`/`k`, `Home` and `End`, stopping at the ends.
  - `Enter` and a click open the detail view, and the header is right.
  - Scrolling with every listed key; `↑ more`/`↓ more` appear and disappear.
  - `Esc`, `←` and `Backspace` return with the pointer kept.
  - `q` and `Ctrl+C` quit from both views and the output ends by leaving the alternate screen.
  - Unlisted keys are ignored.
  - Wheel scroll in the list.
  - Clicks outside entries.
  - Mouse disabled in the detail view and re-enabled in the list (checked through the mouse-mode escape sequences in the output).
  - Resize in both views.
  - Reopening an entry starts at the top.
  - Empty list.
  - With styling forced on, the selected row's style differs from the others and follows the pointer by key and by click. With styling off, there is no colour sequence and `> ` marks the row.
  - The detail view, without its header, indicators and footer and stripped of ANSI, equals the `--help` text at a sufficient width. Wrapping is the only change when narrower.
  - Covers FR-005, FR-006, FR-007, FR-008, FR-009, FR-010, FR-011, FR-013, FR-015, FR-017, FR-019, FR-020, EC-001, EC-003, EC-004, EC-006, EC-007, EC-011.
- `internal/tui` runner:
  - A cancelled context with a signal cause returns that cause after leaving the alternate screen (EC-014).
  - A help failure while the program runs returns the error after the terminal is restored (FR-016).
  - A forced start failure (for example a closed input file) returns `cannot start the home screen: …` (FR-016).
- `cmd/root`:
  - `aicli` with no arguments, buffers as stdin and stdout (static), outside a project: exit 0, the check not consulted. This replaces `TestBareRootNoCheck` (FR-001).
  - The exempt-list pin and `IsExempt` test are updated to include the root's name.
  - Static output golden for the real tree and for the test tree with the `probe` command (FR-012, FR-004).
  - EC-008: the root help equals the Phase 0 baseline plus the `aicli [flags]` Usage line. `aicli unknowncmd`, a flag, and `<cmd> --help` are unchanged.
  - The working directory snapshot is unchanged after a run (FR-014).
  - A test parses the Go files of `internal/home` and the home files of `internal/tui` and fails on imports of `os/exec` or `net` (FR-014).
  - The built-binary test runs `aicli` with no arguments and stdout piped: exit 0 and the real-tree golden (FR-001, EC-009).

**Phase 7: quality and final check**
- Run `gofmt` and `golangci-lint run`.
- Doc comments on every new function, type and constant.
- `go test ./...` passes.
- Tick the Definition of Done items in `spec.md` that the final task verifies.

## Coverage

| FR / EC | Covered by |
|---|---|
| FR-001 | Phase 5 (root `RunE`) + Phase 2 (exempt list) + Phase 6 `cmd/root` |
| FR-002 | Phase 3 (banner) + Phase 6 golden and teatest |
| FR-003 | Phase 4 (depth-first walk, filter, path) + Phase 3 (entry rendering) + Phase 6 |
| FR-004 | Phase 4 (tree read on every run, no names besides the Cobra built-ins) + Phase 6 `probe` E2E and golden |
| FR-005 | Phase 3 (`IsInteractive`, alternate screen, pointer at 0) + Phase 6 teatest |
| FR-006 | Phase 3 (list keys, clamping) + Phase 6 teatest |
| FR-007 | Phase 3 (`Enter`, click → row mapping) + Phase 6 teatest |
| FR-008 | Phase 4 (in-process help, help flag initialised) + Phase 3 (wrap only, styling adds ANSI only) + Phase 6 equality tests |
| FR-009 | Phase 3 (restricted viewport keys, mouse off in detail, on in list) + Phase 6 teatest |
| FR-010 | Phase 3 (back keys, `q`/`Ctrl+C` in both views, alternate screen left) + Phase 6 teatest |
| FR-011 | Phase 3 (wheel scrolls offset, pointer kept visible after keys) + Phase 6 teatest |
| FR-012 | Phase 3 (static renderer) + Phase 4 (mode choice, single write) + Phase 6 golden files |
| FR-013 | Phase 3 (styles, no-op when not styled) + Phase 6 styled/unstyled teatest |
| FR-014 | Phase 4 (no files, processes or network) + Phase 6 snapshot and import test |
| FR-015 | Phase 3 (resize: pointer visible; detail first line kept, clamped) + Phase 6 teatest |
| FR-016 | Phase 3 (runner returns error after restore) + Phase 4 (help error) + Phase 5 (`run` path) + Phase 6 |
| FR-017 | Phase 3 (per-view footer) + Phase 6 teatest |
| FR-018 | retired (QA-14); Phase 3 ignores motion events |
| FR-019 | Phase 3 (fixed detail header) + Phase 6 teatest |
| FR-020 | Phase 3 (reserved indicator lines) + Phase 3 layout + Phase 6 |
| EC-001 | Phase 4 (empty walk, subtree exclusion) + Phase 3 (empty list view, only `q`/`Ctrl+C`) + Phase 6 |
| EC-002 | Phase 3 (narrow banner, truncation with `…` of descriptions, entries, header) + Phase 6 unit |
| EC-003 | Phase 3 (layout hiding order, detail header last) + Phase 6 layout table |
| EC-004 | Phase 3 (row mapping → no entry) + Phase 6 teatest |
| EC-005 | Phase 3 (path alone) + Phase 6 |
| EC-006 | Phase 3 (wrap to width, scrollable viewport) + Phase 6 teatest |
| EC-007 | Phase 3 (unlisted keys ignored, restricted viewport key map) + Phase 6 teatest |
| EC-008 | Phase 5 (`Args` unset, `RunE` only for no arguments) + Phase 0 baseline + Phase 6 |
| EC-009 | Phase 3 (`IsInteractive` needs both TTYs) + Phase 6 (buffers → static; built binary with pipe) |
| EC-010 | Phase 3 (display-width padding and truncation) + Phase 6 unit |
| EC-011 | Phase 3 (reset to top on open) + Phase 6 teatest |
| EC-012 | Phase 3 (static `No commands available.`) + Phase 6 golden |
| EC-013 | retired (QA-14) |
| EC-014 | Phase 3 (context, no Bubble Tea signal handler, returns cause) + Phase 4 (static checks context) + Phase 5 (existing `SignalError` path) + Phase 6 |

## Constitution check

- ARC-1: spec 003 is approved; this plan covers only it.
- ARC-2 / ARC-4: no new command folder; only `cmd/root` changes.
- ARC-3: `cmd/root` only wires `RunE` to `internal/home`. The tree walk, help capture and mode choice live in `internal/home`; drawing lives in `internal/tui`.
- ARC-7: Bubble Tea, Bubbles and Lipgloss are imported only by `internal/tui` and its tests. `internal/home` and `cmd/root` exchange plain entries with it.
- ARC-8: no OS-specific code; TTY detection reuses `golang.org/x/term`; no paths are built and nothing is spawned.
- STK-2: Cobra for the command, Charm for the UI; Bubbles is required by this spec.
- STK-4: Bubble Tea, Bubbles, `teatest` (test only) and `x/ansi` (promoted from indirect to direct) were approved by the team on 2026-10-14; nothing else is added.
- STK-6: `golang.org/x/term` is already a dependency and is reused for `IsInteractive`.
- SEC-4 / SEC-6: no network, no telemetry, no subprocess; help is rendered by Cobra in-process (FR-014).
- ERR-1: help and start failures are returned with context, never swallowed.
- ERR-2 / ERR-5: failures go through `root.run` (`Error: …` on stderr, empty stdout, exit 1). Signals give 128+n and no output.
- ERR-3: static mode prints everything without a TTY and needs no input.
- TST-1 / TST-2 / TST-4: unit, teatest, end-to-end and golden tests; `go test ./...` and `golangci-lint run` pass.
- TST-6: no test uses the network or an LLM.
- LIM-1: behaviour targeted and tested on macOS and Linux only.
