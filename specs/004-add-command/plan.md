# Plan: 004-add-command

- Status: approved

## Approach

A new flat command folder `cmd/add` holds only the Cobra `add` command. Its logic is split across three `internal/` packages, following the split of spec 003 (Cobra knowledge outside `internal/tui`, Charm code only inside it):

- `internal/feature`: the feature contract. It has the helper a future feature uses to build its command, the lookup from a command to its declaration, the contract check, argument validation, the equivalent command, and the run of a feature.
- `internal/add`: what `aicli add` does. It discovers the listed features, chooses static or interactive mode, handles unknown names, and runs the chosen feature after `Add`.
- `internal/gen`: the shared all-or-nothing file mechanism, next to the existing skeleton writer.
- `internal/tui`: draws the selector, the form, the confirmation, the static list and the success report. It receives plain structs, never Cobra types.

Key decisions:
- **A feature is declared through `feature.NewCommand(decl)`**. The declaration holds the name, short and long description, inputs (name, description, optional choices or validator), the plan function and the next steps. The helper builds the Cobra command: `Use` is `<name> <input-1> …`, so Cobra's usage line matches FR-006, and `Args` accepts any count. It also records the declaration in a package registry keyed by the command pointer. `feature.Lookup(cmd)` returns it. A command under `add` that was not built by the helper has no declaration, so the contract check reports "declares no feature plan" (FR-003).
- **Argument checks run inside `RunE`, not in Cobra's `Args`**. Cobra validates `Args` before the persistent pre-run hooks, so a missing argument would be reported before the project root check. That would break FR-001's order. The `add` command takes any arguments for the same reason, and FR-007 is decided in its `RunE` (EC-001).
- **Only one function runs a feature**: `feature.Run(ctx, cmd, values, stdout)`. It checks the values (FR-006 order), computes the plan, calls the shared mechanism and prints the report. The direct call (`RunE`) and the run after `Add` (FR-019) both call it. After `Add`, `internal/add` calls it in-process on the selected feature command, with the trimmed values as positional arguments, the add command's context (which already carries the project root) and its stdout. The flags keep their defaults because they were never parsed, which is exactly what typing the equivalent command does.
- **One Bubble Tea program for the selector, the form and the confirmation**. It returns a result: quit, cancel, or add with a feature index and its values. Printing `Nothing added.` and running the feature happen after the program has returned, so the terminal is always restored first (FR-018, FR-019, EC-019). Bubble Tea runs with the run context and no signal handler, as the home screen does. `main.go` stays the only signal handler (EC-020).
- **The selector reuses the home list**. The home `listModel` becomes a list component configured with its top block (banner or the `aicli › add` header and description), its footer, and the action for `Enter`/click. The selector uses the same `> ` pointer, keys, wheel, indicators, highlight and resize code (FR-009, ARC-7). The layout function gets a third view kind for the selector's top block.
- **The form uses `bubbles/textinput` inside our own field model**. `textinput` already scrolls a long value horizontally (EC-012). Its sanitizer turns tabs and newlines into spaces, but EC-013 asks to discard them, so the form strips C0/C1 controls and `DEL` from key and paste runes before handing them to the field. Choice fields are our own small model (FR-016).
- **Exclusive creation with `O_CREATE|O_EXCL` and `Mkdir`**, with a rollback list. Each created file and directory is recorded in creation order. On any failure, files are removed in reverse order and directories only if empty, through an injectable filesystem interface so tests can force a failed write, a file that appears after the check, and a rollback that fails (FR-024). The context is checked before each creation. On cancellation the rollback runs to the end and the context's cause, a `*scaffold.SignalError`, is returned, so the existing `root.run` path exits with 128+n and prints nothing (EC-023).
- **Read-only preview for the confirmation**. The mechanism exposes its path validation and existence checks (FR-022, FR-023) as a separate function that stats but never writes. The confirmation calls it through a plain function value it receives from `internal/add` (FR-018, FR-020).
- **Deprecated features print no notice (EC-002)**. Cobra prints its deprecation notice before any hook runs, so `cmd/root.run`, the single execute path that tests also use, calls `add.Prepare(root)` once before executing. For each feature under `add` that has Cobra's `Deprecated` set, `Prepare` clears it and sets `Hidden`. The feature stays unavailable and unlisted, still runs directly, and prints no notice.
- **Suggestions** for FR-007 use the `add` command's own `SuggestionsFor`. Cobra's rule only offers available, non-hidden commands, which is the listed set. With no listed feature there is no suggestion (EC-004).
- **Errors are plain messages whose continuation lines have no indentation**. The existing `tui.RenderError` path in `root.run` adds the two-space indent and the styled form (FR-026), as it does for `new`. The `could not remove:` line and each path left behind become continuation lines.

Rejected alternatives:
- **Re-executing the root with the equivalent command's arguments after `Add`**: this would run the project check twice, re-parse flags on an already executed tree, and nest `Execute` inside `Execute`. A direct call to the same `feature.Run` gives the same result without that.
- **Storing inputs in Cobra `Annotations`**: annotations are strings, so validators and the plan function cannot be stored there. A registry keyed by command keeps the declaration typed.
- **Staging plus an atomic no-replace rename, as `aicli new` does**: feature files go into an existing tree, often several directories, and a single rename cannot place them. A per-file no-replace rename gives no more than `O_EXCL`, so `golang.org/x/sys` is not needed here (STK-6 stays unused).
- **Discarding the feature command's own output writer to hide the deprecation notice**: this would also discard `aicli add <feature> --help`.
- **Huh for the form**: the spec rules it out (one Bubble Tea program, mouse capture rules, exact markers).
- **Cobra `Args` validators**: they run before the project root check (see above).

## Components / Phases

**Phase 1: `internal/gen` (shared file mechanism)**
- New `FeatureFile` type: a path relative to the project root, written with `/`; a template source (an `fs.FS` and the template name inside it); and the template data.
- *Path validation (FR-022, EC-021)*: a path must be non-empty, relative and use `/`. It must have no `\`, no empty, `.` or `..` component, and no duplicates. The project root is resolved with `filepath.EvalSymlinks`. For each target, the deepest existing ancestor is resolved and must lie inside the resolved root. An existing ancestor that is not a directory fails (EC-022). An empty list fails with `<feature> declares no files`.
- *Existence check (FR-023)*: `Lstat` every target, so a broken link counts as existing. The error lists every existing path in list order, under the first line `cannot add <feature>, these files already exist:`.
- `CheckFeatureFiles(root, feature, paths)`: the validation and existence checks together, read-only. The confirmation uses it.
- `CreateFeatureFiles(ctx, fsys, root, feature, files)`:
  1. Run `CheckFeatureFiles`.
  2. Render every file in memory with `text/template` (`missingkey=error`); a failure names the path and the cause (FR-024, ARC-5).
  3. Create the missing parent directories (`DirMode`) and each file with `O_EXCL` (`FileMode`), recording what it creates.
  4. On failure or cancellation, roll back in reverse order. Paths the rollback could not remove are added as `could not remove:` lines.
  5. Return the created paths in list order.
- A filesystem interface for this mechanism (`Lstat`, `Mkdir`, `OpenFile`, `Chmod`, `Remove`) with an OS implementation. The existing `FS` and `Write` of `aicli new` do not change.

**Phase 2: `internal/feature` (contract, helper, run)**
- Types:
  - `Input`: name, description, choices, validator `func(string) string`, which returns an empty string on success.
  - `Declaration`: name, short, long, inputs, `Plan func(values []string) ([]gen.FeatureFile, error)` and next steps.
- `NewCommand(decl)`: builds the command as described in the Approach and registers the declaration. `Lookup(cmd)` reads it back.
- `CheckContract(add)`: walks the direct runnable subcommands of `add`, skipping `help`. It returns one error per broken rule, each naming the feature and the rule: aliases; empty or duplicate input names; an empty choice list, or empty or duplicate choices; both choices and a validator; no declaration or no plan (FR-003).
- `ValidateValues(cmd, decl, args)`: trims the arguments, then applies FR-006's order: too few (first missing input), too many, then per input empty, control characters, outside the choices (exact match, EC-009) and validator. Each message's second line is `Usage: ` + `cmd.UseLine()`, after the default help flag has been initialised so the line ends in `[flags]`.
- `EquivalentCommand(name, values)`: applies the quoting rule of the Definitions, and puts `--` before the first value when any value starts with `-` (EC-015, EC-016).
- `Run(ctx, cmd, values, out, getenv)`:
  1. Validate the values.
  2. Compute the plan; a failure gives `Error: <message>` and nothing is created.
  3. Call `gen.CreateFeatureFiles` with the root from `project.RootFrom`.
  4. Write `tui.RenderAdded(feature, paths, nextSteps, styled)` in one write, where styled is `tui.ShouldStyle(out, getenv)` (FR-025).
  The command's `RunE` only calls `Run` with `os.Getenv`.

**Phase 3: `internal/add` (the `add` command's logic)**
- `Features(add)`: the features in Cobra order, with a flag that marks the listed ones (available and not hidden). Each becomes a `tui.Feature`: name, short description, inputs (name, description, choices, validator), a `Plan` function that returns only the paths, and a `Check` function that calls `gen.CheckFeatureFiles` with the project root.
- `Prepare(root)`: the deprecation step from the Approach.
- `Show(ctx, addCmd, args, in, out, getenv)`:
  - With an argument that is not a feature, return the FR-007 error, with the `Did you mean` line only when `SuggestionsFor` finds a listed feature.
  - With no listed feature, write the EC-003 text in both modes.
  - In static mode, write `tui.RenderAddStatic` in one write, after checking the context (FR-005, EC-005, EC-020).
  - Otherwise, run `tui.RunAdd`. For `cancel`, write `Nothing added.`. For `add`, call `feature.Run` on the selected command with the add command's context and stdout. For `quit`, return nil.

**Phase 4: `internal/tui` (screens and output)**
- *List component*: the refactor from the Approach. The home screen keeps its exact behaviour, and its existing tests and goldens must stay green.
- *Layout*:
  - Selector view kind: header, description line and blank line as the top block, which EC-010 hides in this order: description, indicators, footer, header.
  - Form layout: header, description, indicator lines, footer; fields scroll as whole blocks and the focused field stays visible with its error.
  - Confirmation layout: header, description, the wrapped equivalent command, the file list (heading, two indicator lines and the scrolling paths), the buttons and the footer. Its hiding order follows EC-010: file list block first, then the description, indicators, footer and header. In the end the command scrolls above the buttons.
  - Each layout is a pure function of size and content, tested with tables.
- *Form model*:
  - Fields are text fields (`textinput`, control runes filtered out) or choice fields (`(•) `/`( ) `, `↑`/`↓`/`k`/`j`, the full list only while focused, scrolling inside the field when it is too tall).
  - `> ` marks the focused title.
  - `Tab`/`Enter` validate the field being left. `Shift+Tab` and clicks move the focus without validating. `Enter` on the last field validates all fields and moves the focus to the first invalid one. Errors use the `<input> cannot be empty` text and the validator's message.
  - Values are kept per feature for the length of the run (FR-017).
  - Per-field footers as in the design. Unlisted keys and clicks outside fields do nothing (EC-011).
- *Confirmation model*:
  - Shows the equivalent command (wrapped, never truncated), then `Will create:` with the plan paths, or the plan/check error rewritten without `Error: ` and with a capital first letter, in red.
  - `[ Add ]`/`[ Cancel ]` buttons with the focus on `Cancel`. Keys: `←`/`→`/`Tab`, `Enter`, `y`, `n`, `Esc` (back to the form, or to the selector when the feature has no inputs). When blocked, only `Cancel` is shown and `y`, `←`, `→` and `Tab` do nothing.
  - Entering the confirmation issues the disable-mouse command, and going back issues the enable command (FR-018).
- *Program*: `RunAdd(ctx, features, in, out, styled)` returns `Result{Action, Index, Values}`. It uses the same program options as `RunHome`. A start or run error is returned as `cannot start the add screen: <cause>` (FR-021, ERR-1), and a cancelled context returns `context.Cause(ctx)`. An exported model constructor exists for `teatest`.
- *Static and report renderers*: `RenderAddStatic(features)` and the EC-003 text, always plain, with names padded by display width (EC-006, EC-007). `RenderAdded(feature, paths, steps, styled)` follows FR-025 and the design's colours. Styled output only adds ANSI and the `✓ ` prefix.
- *Styles*: selector, form and confirmation styles from the design table, on the private ANSI256 renderer, all no-ops when not styled.

**Phase 5: `cmd/add` and `cmd/root` (wiring)**
- `cmd/add`: `NewCommand()` returns `add` with its `Use`, the fixed short description, any argument count, and a `RunE` that only calls `add.Show` with the command's context, `InOrStdin`, `OutOrStdout` and `os.Getenv` (ARC-3). No feature is named.
- `cmd/root`: registers `addcmd.NewCommand()`, and `run` calls `add.Prepare` before executing. `add` is not added to the exempt list (FR-001). Future feature folders are registered under `add` here, since ARC-4 forbids `cmd/add` from importing them.

**Phase 6: test support and tests (each with its `Covers:` line)**
- `internal/featuretest`, a package imported only by tests, builds the test feature: a choice input, a text input with a validator, two feature files from an in-memory template FS, next steps, and a plan that fails for one chosen value. It also builds test-only features that each break one contract rule, plus a hidden one, a deprecated one, one with no inputs, one without a short description, and one with a wide-character name.
- `internal/gen`:
  - Path validation table: empty, absolute, `\`, empty/`.`/`..` components, leading, trailing or doubled `/`, duplicates, a parent that is a file, and a symlink that leads outside the root.
  - A root reached through a symlink (`/var` → `/private/var` on macOS, an explicit link on Linux) is accepted.
  - The existence error, including a directory and a broken link.
  - A template render failure creates nothing.
  - Rollback after a forced write failure and after a file that appears after the check; a non-empty directory is kept.
  - The `could not remove:` lines.
  - A cancelled context mid-creation completes the rollback and returns the signal cause.
  - Golden files for the created files.
- `internal/feature`:
  - Contract check on the test-feature tree and on each rule-breaking feature.
  - The FR-006 errors in order, with goldens, including control characters, choices (different case) and validator.
  - The equivalent command quoting table.
  - Report goldens, plain and styled with and without next steps, where stripping ANSI and `✓ ` gives the plain text.
  - A failing plan creates nothing.
- `internal/add`: listed/unlisted features, suggestions, EC-003 and EC-004, a direct run of a hidden or deprecated feature after `Prepare` (no notice), static output goldens (test tree, and the real tree with no features), and a pre-cancelled context.
- `internal/tui`:
  - Layout tables for all three screens, including a file list, a choice field and a wrapped command taller than the terminal.
  - Truncation by display width.
  - `teatest` for the full interactive list of the Definition of Done: selector keys, click, wheel, resize and quit; form navigation, clicks, empty values, validator errors, choice keys and clicks, horizontal scroll, pasted controls discarded, `Esc` keeping values; confirmation `y`/`n`/`Enter`/`Esc`, mouse released and re-captured, blocked confirmation with only `Cancel`, markers with styling off, and distinct `Add`/`Cancel` colours with styling on.
  - The EC-003 flow, which never enters the alternate screen.
  - A runner start failure, and a cancelled context.
- `cmd/root`:
  - The real-tree home golden is regenerated with `-update` and shows `add`.
  - A home test with the test feature shows `add <feature>`.
  - Outside a project root, `add`, `add <feature>` and `add <unknown>` give spec 002's message (EC-001); `add --help` and `help add` work anywhere.
  - The exempt-list pin is unchanged.
  - End to end through `run`: a direct call that creates files and prints the report, the FR-007 errors with and without `Did you mean`, the FR-023 error, and styled errors with stdout empty (FR-026).
  - A built-binary test sends SIGINT during creation (through a test hook that blocks between files) and checks the exit code is 130 and that nothing is left behind (EC-023).
  - A test parses the Go files of `internal/add` and of the add screens and fails on imports of `os/exec` or `net` (FR-020).

**Phase 7: quality and final check**
- `gofmt` and `golangci-lint run`.
- Doc comments on every new function, type and constant.
- `go test ./...` passes.
- Tick the Definition of Done items in `spec.md` that the final task verifies.

## Coverage

| FR / EC | Covered by |
|---|---|
| FR-001 | Phase 5 (`add` not exempt) + Approach (checks in `RunE`, after the pre-run check) + Phase 6 `cmd/root` |
| FR-002 | Phase 3 (`Features` read from the tree on every run) + Phase 6 test feature listed with no code change |
| FR-003 | Phase 2 (`CheckContract`, registry) + Phase 6 rule-breaking features and real tree |
| FR-004 | Existing home walk (no change) + Phase 6 regenerated golden and test-tree home test |
| FR-005 | Phase 3 (static mode, single write) + Phase 4 (`RenderAddStatic`) + Phase 6 goldens |
| FR-006 | Phase 2 (`ValidateValues`, `Run`) + Phase 6 goldens |
| FR-007 | Phase 3 (unknown name, `SuggestionsFor`) + Phase 6 `internal/add` and `cmd/root` |
| FR-008 | Phase 4 (selector top block, list component) + Phase 3 (mode choice) + Phase 6 teatest |
| FR-009 | Phase 4 (list component shared with the home screen, `Esc` quits) + Phase 6 teatest |
| FR-010 | Phase 4 (selection, quit keys, `Result` quit) + Phase 3 (quit prints nothing) + Phase 6 teatest |
| FR-011 | Phase 4 (form model, `> ` marker, indicator lines) + Phase 6 teatest |
| FR-012 | Phase 4 (description line in form and confirmation only) + Phase 6 teatest and direct-call golden |
| FR-013 | Phase 4 (trim, empty error, validator error) + Phase 6 teatest |
| FR-014 | Phase 2 (validator after empty and control checks) + Phase 4 (on leaving, on last `Enter`) + Phase 6 |
| FR-015 | Phase 4 (focus keys, mouse focus) + Phase 6 teatest |
| FR-016 | Phase 4 (choice field) + Phase 2 (choices on a direct call) + Phase 6 |
| FR-017 | Phase 4 (values kept per feature in memory) + Phase 6 teatest |
| FR-018 | Phase 4 (confirmation model, mouse released) + Phase 1 (`CheckFeatureFiles`) + Phase 3 (`Nothing added.`) + Phase 6 |
| FR-019 | Phase 3 (in-process `feature.Run` after the program returns) + Phase 6 end-to-end |
| FR-020 | Approach (read-only preview) + Phase 6 import test and working-directory snapshot |
| FR-021 | Phase 4 (`RunAdd` start error after restore) + Phase 6 runner test |
| FR-022 | Phase 1 (path validation, symlink resolution, empty list) + Phase 6 table |
| FR-023 | Phase 1 (existence check, `Lstat`) + Phase 6 goldens |
| FR-024 | Phase 1 (render first, `O_EXCL`, rollback, `could not remove:`) + Phase 6 forced failures |
| FR-025 | Phase 2 (`Run` writes the report) + Phase 4 (`RenderAdded`) + Phase 6 goldens |
| FR-026 | Approach (plain messages via `tui.RenderError` in `root.run`) + Phase 6 styled vs plain |
| EC-001 | Approach (`add` accepts any args, FR-007 in `RunE`) + Phase 6 `cmd/root` |
| EC-002 | Phase 3 (`Prepare`, listed filter) + Phase 6 hidden and deprecated features |
| EC-003 | Phase 3 (plain text in both modes) + Phase 4 (renderer) + Phase 6 golden and teatest |
| EC-004 | Phase 3 (no suggestion without listed features) + Phase 6 |
| EC-005 | Phase 3 (`tui.IsInteractive` needs both TTYs) + Phase 6 |
| EC-006 | Phase 4 (display-width padding and truncation) + Phase 6 wide-name feature |
| EC-007 | Phase 4 (name alone) + Phase 6 |
| EC-008 | Phase 4 (no description line) + Phase 6 teatest |
| EC-009 | Phase 2 (exact match, choices listed) + Phase 6 golden |
| EC-010 | Phase 4 (three layout functions, hiding order, command wrap and scroll) + Phase 6 layout tables and teatest |
| EC-011 | Phase 4 (unlisted keys and clicks ignored) + Phase 6 teatest |
| EC-012 | Phase 4 (`textinput` horizontal scroll) + Phase 6 teatest |
| EC-013 | Phase 4 (control runes discarded) + Phase 2 (direct-call error) + Phase 6 |
| EC-014 | Phase 4 (`Esc`/`Ctrl+C` return quit, no run) + Phase 6 teatest |
| EC-015 | Phase 2 (`--` in the equivalent command) + Phase 3 (values passed as positional, not parsed) + Phase 6 |
| EC-016 | Phase 2 (quoting) + Phase 6 table and end-to-end |
| EC-017 | Phase 4 (confirmation rebuilt from current values) + Phase 6 teatest |
| EC-018 | Phase 1 (checks repeated at run, `O_EXCL`) + Phase 6 file created between confirmation and `Add` |
| EC-019 | Phase 3 (feature runs after the terminal is restored) + Phase 6 teatest/end-to-end |
| EC-020 | Phase 4 (context, no Bubble Tea signal handler) + Phase 3 (static checks the context) + existing `SignalError` path + Phase 6 |
| EC-021 | Phase 1 (duplicate paths) + Phase 6 |
| EC-022 | Phase 1 (parent is a file, symlink outside root) + Phase 6 |
| EC-023 | Phase 1 (context checked per creation, full rollback, cause returned) + Phase 6 built-binary signal test |

## Constitution check

- ARC-1: spec 004 is approved; this plan covers only it and ships no real feature.
- ARC-2: one new flat folder, `cmd/add`; future features get their own flat folders.
- ARC-3: `cmd/add` only wires `RunE` to `internal/add`; the logic lives in `internal/add`, `internal/feature` and `internal/gen`.
- ARC-4: `cmd/add` imports no other `cmd/` package; features are registered under `add` by `cmd/root`.
- ARC-5: feature files are rendered from embedded templates with `text/template`; the test feature uses an in-memory FS.
- ARC-7: Bubble Tea, Bubbles and Lipgloss stay in `internal/tui`; the other packages pass it plain structs and functions.
- ARC-8: paths are built with `path/filepath`, and template names inside an FS with `path`; files go through `os`; nothing is spawned.
- STK-2 / STK-4: Cobra plus Bubble Tea, Bubbles (`textinput`) and Lipgloss, which are already direct dependencies; Huh is not used and no module is added.
- STK-5: `add` ships no capability; features are opt-in subcommands.
- STK-6: not needed; `O_EXCL` gives no-replace creation without `golang.org/x/sys`.
- SEC-3: target paths are validated before any write, and symlinks cannot lead outside the project root.
- SEC-4 / SEC-6: no network and no subprocess; the feature runs in-process (FR-020).
- SEC-5: no file is ever replaced; existing paths stop the run (FR-023), and creation is exclusive.
- ERR-1: render, write and rollback failures are returned with their path and cause.
- ERR-3: static mode and the direct call need no TTY or prompt.
- ERR-5: plain `Error:` messages with continuation lines through `root.run`; signals give 128+n and print nothing.
- TST-1 / TST-2 / TST-4: unit, teatest, golden and end-to-end tests; `go test ./...` and `golangci-lint run` pass.
- TST-6: no test uses the network or an LLM.
- LIM-1: macOS and Linux only; the symlinked-root test covers both.
