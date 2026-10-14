# 003-home-screen: Home screen (`aicli` with no arguments)

- Status: approved
- Owner:  ro-uc-ru
- Date:   2026-10-14

## 1. Context

### Goal (required)
Running `aicli` with no arguments shows a home screen: a large `AICLI` banner with two lines describing the framework, and below it the list of every command available. In a terminal the user moves through the list with a pointer, opens a command with Enter or a click, and reads its full help; without a terminal the same information is printed as static text.

### Why (required)
Today `aicli` with no arguments prints Cobra's plain help, which gives a new user no identity and no guided way in. The list of commands will grow with every future spec, so the screen must discover commands at runtime instead of listing them by hand: a new command appears on the home screen with no change to this feature. The effort of this spec is the presentation (Lipgloss) rather than the command wiring.

### Optional
- Users / Beneficiaries: data scientists and AI engineers who run `aicli` to discover what it can do; authors of future command specs, whose command shows up with no extra work.
- Definitions:
  - Home screen: what `aicli` shows when run with no arguments and no flags.
  - Banner: the large `AICLI` lettering plus the two description lines.
  - Listed command: a command or subcommand of the command tree that is available (Cobra `IsAvailableCommand`), is not hidden and is not `help`, `completion` or `__complete`, and whose ancestors below the root are all listed commands (so the subcommands of an excluded command, e.g. `completion bash`, are not listed). A subcommand is its own entry, named by its full path without the program name (e.g. `db migrate`). A grouping command that is not runnable but has listed subcommands (e.g. `db`) is a listed command too; its detail view shows its own help.
  - Command help: the text `aicli <command path> --help` prints. It carries the documentation of the command, its usage, examples and flags; how each command documents its inputs and outputs is up to its own spec.
  - Interactive mode: stdin and stdout are TTYs and `TERM` is not `dumb`. Any other case is static mode.
  - Styled: `NO_COLOR` is unset or empty and `TERM` is not `dumb` (same rule as ERR-5, applied to stdout).
- Fixed banner text (English, like the rest of the CLI):
  - Line 1: `Scaffolding for FastAPI backends for AI/LLM apps.`
  - Line 2: `Convention over configuration, driven by specs.`
- Notes for the plan (non-normative):
  - Listed commands are read from the Cobra tree when the screen starts; no list of commands exists in this feature's code.
  - The command help is produced in-process by the command itself (Cobra help function), never by spawning `aicli` again (SEC-6).
  - Mouse support, the pointer, the banner and the scrollable detail live in `internal/tui` (ARC-7); `cmd/root` only wires the root command (ARC-3).
  - `charmbracelet/bubbles` (viewport, and list if useful) is approved by the team for this spec (STK-4, STK-2); no other new dependency.
  - `aicli` with no arguments must be declared exempt from the check of spec 002 (its FR-007); its exempt-commands test is updated accordingly.
  - Non-normative mockups: `designs/home-screen.md`.
  - The golden file of the static output with the real command tree changes whenever a command is added or removed; regenerating it with `-update` is part of every spec that adds a command and does not count as a change to this feature.

## 2. Functional Requirements

- FR-001 (event-driven)  WHEN `aicli` is run with no arguments and no flags, the CLI shall show the home screen and exit with code 0, and the home screen shall be exempt from the project root check of spec 002.
- FR-002 (ubiquitous)    The home screen shall show the banner at the top: the word `AICLI` in large lettering followed by the two description lines of the fixed banner text.
- FR-003 (ubiquitous)    The home screen shall show, below the banner, one entry per listed command, in depth-first pre-order (each command followed by its listed subcommands, siblings in the order Cobra gives them), each with the command path and its short description.
- FR-004 (ubiquitous)    The list of entries shall be built from the command tree each time the home screen is shown, so that a command added to or removed from the tree is added to or removed from the list with no other change; no command shall be named in the code of this feature.
- FR-005 (state-driven)  WHILE in interactive mode, the CLI shall run the home screen as an interactive screen in the terminal's alternate screen, with a pointer marking exactly one entry, shown as a `> ` prefix on that entry (when styled, FR-013 also gives it a distinct colour); the pointer shall start on the first entry.
- FR-006 (event-driven)  WHEN, in interactive mode, the user presses `↓` or `j` (`↑` or `k`), the pointer shall move to the next (previous) entry; at the last (first) entry it shall stay where it is. WHEN the user presses `Home` or `End`, it shall move to the first or last entry.
- FR-007 (event-driven)  WHEN, in interactive mode, the user presses `Enter`, or clicks with the mouse on an entry, the CLI shall open the detail view of that entry (a click also moves the pointer to it).
- FR-008 (ubiquitous)    The detail view shall show the command help of the selected entry, with the same text as `aicli <command path> --help`: WHILE the terminal is at least as wide as the longest line of that help, stripping any ANSI sequences from the help area of the detail view (the detail view without its header, FR-019, its scroll indicators, FR-020, and its footer, FR-017) shall give exactly that text; WHILE it is narrower, the only other change allowed is the line wrapping of EC-006. When styled, headings and flags may be emphasized with Lipgloss; styling shall only add ANSI sequences.
- FR-009 (state-driven)  WHILE the detail view is shown and its text is taller than the space available to the help area, the CLI shall let the user scroll it with `↑`/`↓`, `j`/`k`, `PgUp`/`PgDn` and `Home`/`End`. WHILE the detail view is shown, the CLI shall not capture the mouse, so that the terminal's own text selection lets the user copy the help; mouse capture shall be enabled again on returning to the list.
- FR-010 (event-driven)  WHEN, in the detail view, the user presses `Esc`, `←` or `Backspace`, the CLI shall return to the list with the pointer on the same entry as before. WHEN the user presses `q` or `Ctrl+C` in either view, the CLI shall exit with code 0 and restore the terminal: it leaves the alternate screen, and nothing of the home screen remains printed. `Ctrl+C` here is the key read by the interactive screen, not a signal (see EC-014).
- FR-011 (state-driven)  WHILE in interactive mode, the mouse wheel in the list shall scroll the list when it is taller than the space available to it; the pointer shall always remain visible after a keyboard move.
- FR-012 (state-driven)  WHILE in static mode, the CLI shall print the banner and the list to stdout once, as plain text with no pointer, no ANSI sequences and no mouse or terminal control sequences, and exit with code 0 (ERR-3). Each entry shall be one line: the command path padded to a common width, then the short description. In static mode the terminal width is ignored: the banner is always the large lettering, and no line is truncated or wrapped (EC-002 and EC-003 apply only in interactive mode).
- FR-013 (state-driven)  WHILE in interactive mode and styled, the CLI shall style the screen with Lipgloss (banner, pointer, selected entry, detail header, scroll indicators, key-hint footer), and the entry under the pointer shall be drawn in a colour (foreground and/or background) different from the other entries; WHEN the pointer moves to another entry, by keyboard or mouse, the previous entry shall return to the normal colour and the new one shall take the highlight colour; WHILE not styled, the screen shall use no ANSI colour sequences and the pointer `> ` shall still identify the selected entry. Styling shall not change the text of the entries.
- FR-014 (ubiquitous)    The home screen shall not create, modify or delete any file, shall not spawn any external process and shall not use the network (SEC-4, SEC-6).
- FR-015 (event-driven)  WHEN the terminal is resized while the interactive screen is shown, the CLI shall re-lay it out for the new size: in the list, the pointer shall stay on the same entry and remain visible; in the detail view, the first visible line shall stay the same, unless it must move up so that the end of the text is not above the bottom edge.
- FR-016 (unwanted)      IF the home screen cannot be started or the command help of an entry cannot be produced, THEN the CLI shall first restore the terminal (leaving the alternate screen) and then fail as in ERR-5, with the first line `Error: <cause>`; nothing of the home screen shall remain on stdout.
- FR-017 (ubiquitous)    The footer of the interactive screen shall list the keys valid in the current view (list: move, open, quit; detail: scroll, back, quit).
- FR-018 (event-driven)  WHEN, in the list view in interactive mode, the mouse moves over an entry, the CLI shall move the pointer to that entry (and so its highlight colour, FR-013) without opening it; moving over anything that is not an entry shall leave the pointer where it is. (retired: hover changed the selection by accident while using the keyboard; the pointer moves only by keyboard and click, QA-14)
- FR-019 (state-driven)  WHILE the detail view is shown, the CLI shall show above the help area a header line `aicli › <command path>` (e.g. `aicli › db migrate`) that stays fixed while the help scrolls.
- FR-020 (state-driven)  WHILE in interactive mode, the CLI shall reserve one line directly above and one directly below the list and the help area; the line above shall show `↑ more` when content is hidden above the visible part, and the line below `↓ more` when content is hidden below it; otherwise the line shall be blank.

## 3. Edge Cases

- EC-001 IF no command is listed (e.g. a tree with only `help`, `completion` and hidden commands), THEN the home screen shall show the banner and the line `No commands available.` in place of the list, and in interactive mode only `q` and `Ctrl+C` shall have effect; `Enter` and clicks shall do nothing. (FR-003)
- EC-002 IF, in interactive mode, the terminal is narrower than the large `AICLI` lettering, THEN the banner shall show `AICLI` as a single plain line followed by the two description lines; if a description line, entry or detail header (FR-019) is wider than the terminal, it shall be truncated with `…` and never wrapped. (FR-002, FR-003, FR-019)
- EC-003 IF, in interactive mode, the terminal is too short to show the banner and the whole list, THEN the list shall scroll within the remaining space and the banner shall stay at the top; IF there is no room even for one entry (in the detail view, one line of help), THEN elements shall be hidden in this order until it fits: first the banner, then the scroll-indicator lines (FR-020), then the footer (FR-017), then, in the detail view, the detail header (FR-019); at least one entry, or one line of help, shall always remain visible. (FR-009, FR-011, FR-019, FR-020)
- EC-004 WHEN the user clicks outside every entry (banner, empty space, footer), THEN nothing shall happen. (FR-007)
- EC-005 IF a command has no short description, THEN its entry shall show the path alone. (FR-003)
- EC-006 IF the command help is longer than the space available to the help area or wider than the terminal, THEN the detail view shall be scrollable vertically and long lines shall be wrapped to the terminal width, not truncated. (FR-009)
- EC-007 IF the user presses a key that is not listed for the current view, THEN nothing shall happen. (FR-006, FR-010)
- EC-008 IF `aicli` is run with arguments or flags (e.g. `aicli new`, `aicli --help`, `aicli -v`, `aicli unknowncmd`), THEN the home screen shall not be shown and the existing behaviour (Cobra help, version, the command, or Cobra's unknown-command error) shall be unchanged, except that the Usage block of the root help gains the line `aicli [flags]` because the root becomes runnable. (FR-001)
- EC-009 IF stdin is a TTY but stdout is not (or the reverse), THEN the CLI shall use static mode. (FR-012)
- EC-010 IF a command path or short description contains characters wider than one cell (e.g. CJK, emoji), THEN alignment and truncation shall use the displayed width, not the byte or rune count. (FR-003, FR-012)
- EC-011 IF the user opens the same entry twice or opens one entry after another, THEN each detail view shall start at the top and show its own help. (FR-008)
- EC-012 WHEN in the static output no listed command exists, THEN it shall print the banner and `No commands available.` and exit with code 0. (FR-012, EC-001)
- EC-013 IF no mouse motion events are received (e.g. the terminal does not report them), THEN the list shall behave as with keyboard and clicks only: the pointer is moved and highlighted by the keyboard (FR-006) and by clicks (FR-007), and nothing else changes. (FR-018) (retired: FR-018 retired, QA-14)
- EC-014 IF the process receives a signal (e.g. SIGINT or SIGTERM sent with `kill`, or `Ctrl+C` in static mode), THEN the CLI shall restore the terminal if the interactive screen was shown and exit as in ERR-5, with 128 plus the signal number and printing nothing. (FR-010, FR-012)

## 4. Scope

### In scope
- The home screen for `aicli` without arguments: banner, dynamic list, pointer, detail view with the command help, mouse and keyboard.
- The static, plain-text form for non-interactive use.
- Declaring the no-argument root exempt from the project root check of spec 002.

### Out of scope
- Running a command from the home screen: the screen is for reading help only, so that no command is triggered by accident.
- Search or filtering of the list: the list is short today; a future spec may add it.
- Prompting for arguments or flags of a command: each command spec owns its own prompts.
- A convention for documenting a command's inputs and outputs: the detail shows whatever the command's help holds; each command spec decides its content.
- Themes, user-configurable colours or banner text: fixed by convention (no configuration knobs).
- Windows support: LIM-1.
- Changing the existing behaviour of `aicli --help`, `aicli help` and `aicli <command> --help`, other than the `aicli [flags]` Usage line of the root help (EC-008).

## 5. Definition of Done
- [x] FR-001..FR-020 covered by tests (except retired FR-018)
- [x] EC-001..EC-014 covered by tests (except retired EC-013)
- [x] The home screen is verified end to end with a test command registered only in tests, which appears in the list and in its detail view with no change to this feature's code (FR-004)
- [x] The static output of FR-012 is covered by a golden file, in plain form, with the real command tree and with the test command tree
- [x] The interactive flow (move, Enter, scroll, Esc, q, click, resize, detail header, scroll indicators) is covered by teatest
- [x] A test checks that the plain root help equals the previous one except for the `aicli [flags]` Usage line (EC-008)
- [x] A test with styling forced on checks that the entry under the pointer is rendered with a different style than the others, and that the style follows the pointer when it moves by key and by click (FR-013)
- [x] A test checks that the detail view, stripped of ANSI, equals the command's `--help` text (FR-008)
- [x] The exempt-commands test of spec 002 is updated and pins the no-argument root (FR-001)
- [x] `go test ./...` and `golangci-lint run` pass (TST-4)

## 6. Changelog

- 2026-10-14 FR-013 now requires the entry under the pointer to change colour as the pointer moves; FR-018 (mouse hover moves the pointer) and EC-013 added, plus a DoD line. Status stays draft.
- 2026-10-14 QA-1: EC-008 and scope accept the `aicli [flags]` Usage line added by a runnable root; DoD line added.
- 2026-10-14 QA-2: FR-008 equality holds when the terminal is wide enough; narrower, only EC-006 wrapping may differ.
- 2026-10-14 QA-3: the interactive screen uses the alternate screen (FR-005); FR-010 and FR-016 restore it before exit, leaving nothing printed.
- 2026-10-14 QA-4: FR-010 `Ctrl+C` is a key in interactive mode; added EC-014 (signals follow ERR-5).
- 2026-10-14 QA-5: the detail view releases the mouse so help text can be copied; mouse-wheel scroll removed from FR-009; EC-013 reworded to be testable.
- 2026-10-14 QA-6: FR-003 lists entries in depth-first pre-order; grouping commands are listed (Definitions).
- 2026-10-14 QA-7: plan note on regenerating the real-tree golden file in future command specs.
- 2026-10-14 QA-8: static mode ignores the terminal width (FR-012); EC-002 and EC-003 apply only in interactive mode.
- 2026-10-14 QA-9: `q` quits from both the list and the detail view (FR-010).
- 2026-10-14 QA-10: FR-015 states the observable result of a resize (pointer on the same entry and visible; detail top line kept unless clamped).
- 2026-10-14 QA-11: FR-008 equality applies only to the help area, not to the header, scroll indicators or footer.
- 2026-10-14 QA-12: added FR-019, a fixed `aicli › <command path>` header in the detail view.
- 2026-10-14 QA-13: added FR-020, `↑ more` / `↓ more` scroll indicators on reserved lines.
- 2026-10-14 QA-14: FR-018 (hover moves the pointer) and EC-013 retired; DoD updated.
- 2026-10-14 QA-15: FR-009, FR-011 and EC-006 measure against the space available, not the whole terminal.
- 2026-10-14 QA-16: EC-003 fixes the hiding order on short terminals (banner, scroll indicators, footer); one entry or help line always visible.
- 2026-10-14 QA-17: EC-002 truncation also applies to the detail header (FR-019).
- 2026-10-14 QA-18 (from planning, approved by the owner): the detail header is hidden last in EC-003's order; Definitions state that subcommands of an excluded command are not listed.
