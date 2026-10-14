# Tasks: 003-home-screen

- Status: approved

Each new dependency (Bubble Tea, Bubbles, `x/ansi` as direct, `teatest`) is added by the first task that uses it. That task's "Done when" includes that `go mod tidy` leaves no diff. The team approved these dependencies on 2026-10-14.

## Phase 0: baseline before the change

- [x] T-001 Capture the current plain `aicli --help` output (root not yet runnable) as `cmd/root/testdata/root-help-before.golden`, through a test that can regenerate it with `-update`. Covers: EC-008. Done when: the golden file exists and has no `aicli [flags]` line, and the test comparing the current root help with it passes.

## Phase 2: `internal/project` (exemption)

- [x] T-002 Add the root's name (`aicli`) to the single exempt list and update its doc comment (it stands for the root run with no arguments). Update the exempt-list pin test and `TestIsExempt` (`aicli` moves from the negative cases to the positive ones). Covers: FR-001. Done when: `go test ./internal/project` passes with the updated pin.

## Phase 3: `internal/tui` (rendering and the interactive program)

- [x] T-003 [P] Add the home `Entry` type (path, short description, help producer returning `(string, error)`) and `IsInteractive(in, out, getenv)`, which is true only when both are `*os.File` terminals and `TERM` is not `dumb`. Covers: FR-005, EC-009. Done when: unit tests pass for in-memory buffers, pipes (`os.Pipe`) on one or both sides, and `TERM=dumb`, all giving false.
- [x] T-004 Add the banner (the six-line `AICLI` lettering of `designs/home-screen.md` and the two fixed description lines) and the static renderer: banner, blank line, one line per entry with the path padded to the widest path by display width, two spaces, then the short description or the path alone; `No commands available.` when there are no entries. Covers: FR-002, FR-003, FR-012, EC-005, EC-010, EC-012. Done when: golden tests in `internal/tui/testdata/` pass for a fixed entry list (with a nested path, an entry without a short description, and CJK and emoji paths) and for an empty list, and the output contains no ESC byte. After: T-003
- [x] T-005 [P] Add the home styles from the design's colour table: a private renderer forced to ANSI256 when styled, and no-op styles when not styled. Covers: FR-013. Done when: a unit test renders sample text in both modes; the unstyled output has no ESC byte; the styled output, stripped of ANSI, equals the unstyled one; the selected style differs from the normal one. After: T-003
- [x] T-006 [P] Add the pure layout function: `(width, height, view, content length)` → visible parts (banner or header, indicator lines, footer) and area height, with EC-003's hiding order (banner, indicator lines, footer, then the detail header), at least one row always left, and a screen row → entry index or "no entry" mapping. Covers: EC-003, FR-007, FR-019, FR-020, EC-004. Done when: a table test over sizes from 1 row up to the full layout passes for both views, including the row mapping with a scroll offset. After: T-003
- [x] T-007 [P] Add the width helpers on `x/ansi`: truncation with `…` by display width (never wrapping), the narrow banner (single `AICLI` line when the width is below the lettering's width), and padding by display width. Covers: EC-002, EC-010. Done when: unit tests pass for ASCII, CJK and emoji text at widths around the cut point, and for the narrow banner with truncated description lines; `go mod tidy` leaves no diff. After: T-004
- [x] T-008 Add the list view model on Bubble Tea:
  - pointer starts at 0 and scroll offset starts at 0;
  - `↓`/`j`, `↑`/`k` move and stop at the ends; `Home`/`End` jump; `Enter` opens; `q`/`Ctrl+C` quit;
  - other keys, motion events and clicks outside entries are ignored (FR-018 and EC-013 are retired);
  - a left click on an entry moves the pointer and opens it;
  - the wheel moves only the offset; the pointer is kept visible after every keyboard move;
  - rows are drawn with `> `, padding and truncation, with the highlight style on the selected row and the dimmed parent prefix;
  - `↑ more`/`↓ more` show on the reserved lines when content is hidden;
  - footer `↑/↓ move · enter open · q quit`;
  - the empty list shows `No commands available.`, footer `q quit`, and only `q`/`Ctrl+C` act.

  Covers: FR-005, FR-006, FR-007, FR-011, FR-013, FR-017, FR-020, EC-001, EC-004, EC-007. Done when: unit tests that send messages to `Update` and inspect `View` and the model state pass for every listed behaviour; `go mod tidy` leaves no diff. After: T-005, T-006, T-007
- [x] T-009 Add the detail view model with `bubbles/viewport`:
  - header `aicli › <path>`, truncated per EC-002;
  - help wrapped to the width by display width (lines that fit are untouched);
  - emphasis of heading lines and flag tokens when styled, adding ANSI only;
  - viewport key map limited to `↑`/`k`, `↓`/`j`, `PgUp`, `PgDn`, with `Home`/`End` handled by the model and viewport mouse off;
  - every open starts at the top with that entry's help;
  - `Esc`, `←` and `Backspace` return to the list with the pointer unchanged; `q`/`Ctrl+C` quit;
  - entering the detail view returns Bubble Tea's disable-mouse command, and returning to the list returns the enable command;
  - `↑ more`/`↓ more` indicators and footer `esc back · ↑/↓ scroll · q quit`.

  Covers: FR-008, FR-009, FR-010, FR-017, FR-019, FR-020, EC-002, EC-006, EC-007, EC-011. Done when: unit tests on `Update`/`View` pass for every listed behaviour, including that the returned commands produce the disable-mouse and enable-mouse messages; `go mod tidy` leaves no diff. After: T-008
- [x] T-010 Handle window-size messages in both views: recompute the layout; in the list, keep the pointer on the same entry and visible; in the detail view, re-wrap and keep the first visible line (source line and wrap segment), clamped so the end of the text is not above the bottom. Covers: FR-015. Done when: unit tests pass for shrinking and growing in both views, including a resize that forces the clamp. After: T-009
- [x] T-011 Add the runner `RunHome(ctx, entries, in, out, styled)` and the exported model constructor (entries, styled, initial size):
  - alternate screen, mouse cell motion, the given input and output, the context, and no Bubble Tea signal handler;
  - a help failure quits and is returned after the terminal is restored;
  - a start or run failure returns `cannot start the home screen: <cause>`;
  - a cancelled context returns `context.Cause(ctx)`;
  - quitting by key returns nil.

  Covers: FR-010, FR-016, EC-014. Done when: runner tests with pipe input and buffer output pass:
  - `q` returns nil and the output ends by leaving the alternate screen;
  - a context cancelled with a `*scaffold.SignalError` cause returns that cause and the output contains the alternate-screen exit;
  - a failing help producer returns its error after the alternate-screen exit;
  - a closed input file returns the `cannot start the home screen:` error.

  After: T-010

## Phase 4: `internal/home` (tree → entries, mode choice)

- [x] T-012 Create package `internal/home` with the entry builder:
  - depth-first pre-order walk of `root.Commands()` in Cobra's order;
  - a command is listed if it is available, not hidden and not `help`/`completion`/`__complete`, and an excluded command excludes its subtree;
  - path = command path without the root's name.

  The help producer initialises the command's default help flag, points its out and err writers at buffers, calls its help function, restores the writers to unset, and returns the text or `cannot show help for <path>: <first line written to err>`.

  Covers: FR-003, FR-004, FR-008, FR-016, EC-001, EC-005. Done when: unit tests on test trees pass for:
  - nesting order, a grouping command, and the exclusions (hidden, deprecated, `help`, `completion` with subcommands, `__complete`);
  - an empty tree and a command without a short description;
  - help equal to `<path> --help` run through Cobra on the same tree, for a leaf, a subcommand and a grouping command;
  - a broken help template giving the FR-016 error, and the writers restored afterwards.

  After: T-003
- [x] T-013 Add `Show(ctx, root, in, out, getenv)`:
  - interactive (`tui.IsInteractive`): call `tui.RunHome` with `tui.ShouldStyle(out, getenv)`;
  - otherwise: build the static text, return the context's cause if it is cancelled, and write the text in one write;
  - return write errors with context.

  Covers: FR-012, FR-014, EC-009, EC-014. Done when: unit tests pass for static `Show` with buffers (exact static text, no ESC byte), a pre-cancelled context (returns the cause, writes nothing), and a failing writer (error returned with context). After: T-004, T-011, T-012

## Phase 5: `cmd/root` (wiring)

- [x] T-014 Give the root a `RunE` that only calls `home.Show` with the command's context, `InOrStdin`, `OutOrStdout` and `os.Getenv`; leave `Args` unset. Replace `TestBareRootNoCheck`: `aicli` with no arguments and buffers, outside a project, exits 0, prints the static home screen and never consults the check. Covers: FR-001, EC-008. Done when: `go build ./...` succeeds, `main.go` is unchanged, the replaced test passes, and the existing `cmd/root` and `cmd/new` tests pass. After: T-002, T-013

## Phase 6: end-to-end, teatest and golden tests

- [x] T-015 Teatest flow in an external `internal/tui` test package, with entries built by `internal/home` from a test tree, unstyled:
  - initial pointer;
  - moves with the arrow keys, `j`/`k`, `Home` and `End`, stopping at the ends;
  - `Enter` and a click open the detail view with the right header;
  - scroll with every listed key, with `↑ more`/`↓ more` appearing and disappearing;
  - `Esc`, `←` and `Backspace` return with the pointer kept;
  - `q` and `Ctrl+C` quit from both views;
  - unlisted keys are ignored;
  - wheel scroll in the list and clicks outside entries;
  - the mouse-mode sequences show the mouse disabled in the detail view and enabled again in the list;
  - resize in both views; reopening an entry starts at the top;
  - the empty list;
  - a short terminal applies the EC-003 hiding order.

  Covers: FR-005, FR-006, FR-007, FR-009, FR-010, FR-011, FR-015, FR-017, FR-019, FR-020, EC-001, EC-003, EC-004, EC-006, EC-007, EC-011. Done when: every teatest case passes. After: T-011, T-012
- [x] T-016 [P] Teatest with styling forced on and off. When styled, the selected row is rendered with a different style from the others, and the style follows the pointer when it moves by key and by click. When unstyled, there is no colour sequence and `> ` marks the selected row. In both modes the entry text is the same. Covers: FR-013. Done when: the tests pass. After: T-015
- [x] T-017 [P] Test that the detail view's help area (without header, indicators and footer), stripped of ANSI, equals the entry's `--help` text when the terminal is at least as wide as its longest line, and that when narrower the only difference is wrapping. Use a test command with long lines, styled and unstyled. Covers: FR-008, EC-006. Done when: the tests pass. After: T-015
- [x] T-018 Golden tests in `cmd/root` of the static output with the real command tree and with the test tree including `probe`, regenerable with `-update`. Extend the built-binary test: `aicli` with no arguments and stdout piped exits 0 and prints the real-tree golden. Covers: FR-001, FR-004, FR-012, EC-009. Done when: the tests pass and `probe` appears in the test-tree golden with no change to the feature's code. After: T-014
- [x] T-019 [P] EC-008 tests:
  - the plain root help equals the T-001 baseline plus exactly the `aicli [flags]` Usage line;
  - `aicli unknowncmd` gives Cobra's unknown-command error (with suggestions when one applies);
  - an unknown flag on the root gives Cobra's flag error;
  - `aicli <command> --help` is unchanged;
  - none of them shows the home screen.

  Covers: EC-008. Done when: the tests pass. After: T-014
- [x] T-020 [P] FR-014 tests:
  - a before/after snapshot of the working directory is identical after a static run and after a teatest run;
  - a test parses the Go files of `internal/home` and the home files of `internal/tui` and fails if they import `os/exec`, `net` or `net/http`.

  Covers: FR-014. Done when: the tests pass. After: T-014, T-015

## Phase 7: quality and Definition of Done

- [x] T-021 Quality sweep:
  - `gofmt` and `golangci-lint run`;
  - a doc comment on every new function, method, type and constant;
  - a `Covers:` line on every new test;
  - no `cmd/` package and no package other than `internal/tui` imports Bubble Tea, Bubbles or Lipgloss (ARC-7);
  - `go.mod` adds only the approved dependencies.

  Covers: FR-001, FR-002, FR-003, FR-004, FR-005, FR-006, FR-007, FR-008, FR-009, FR-010, FR-011, FR-012, FR-013, FR-014, FR-015, FR-016, FR-017, FR-019, FR-020. Done when:
  - `gofmt -l .` prints nothing;
  - `golangci-lint run` and `go test ./...` pass;
  - `go mod tidy` leaves no diff;
  - a grep finds a `Covers:` line for every non-retired FR-001..FR-020 and EC-001..EC-014.

  After: T-015, T-016, T-017, T-018, T-019, T-020
- [x] T-022 Check every Definition of Done item in `spec.md` and tick the ones verified. Covers: all. Done when: every DoD item in `spec.md` is ticked, with its evidence (test name or command output) reported. After: T-021
