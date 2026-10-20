# Tasks: 004-add-command

- Status: approve
No dependency is added. Bubble Tea, Bubbles (`textinput`), Lipgloss, `x/ansi` and `teatest` are already in `go.mod`, so in every task `go mod tidy` must leave no diff. Every new test has a `Covers:` line.

## Phase 1: `internal/gen` (shared file mechanism)

- [x] T-001 Add the `FeatureFile` type (relative `/` path, template `fs.FS` and name, data) and the path validation:
  - non-empty, relative, `/` separator, no `\`, no empty, `.` or `..` component (so no leading, trailing or doubled `/`), no duplicate path;
  - the project root and the deepest existing ancestor of each target are resolved with `filepath.EvalSymlinks`, and the target must lie inside the resolved root;
  - an existing ancestor that is not a directory fails;
  - an empty list fails with `<feature> declares no files`;
  - each error names the path.

  Covers: FR-022, EC-021, EC-022. Done when: a table test passes for every rejected form above, for a symlink that leads outside the root, and for a parent that is a regular file; a test passes on a root reached through a symbolic link (macOS `/var` → `/private/var`, an explicit link on Linux) and accepts a path inside it.
- [x] T-002 Add the existence check (`Lstat` on every target, listing every existing path in list order under `cannot add <feature>, these files already exist:`) and `CheckFeatureFiles(root, feature, paths)`, which runs validation and existence together without writing. Covers: FR-023, FR-018. Done when: unit tests pass for an existing file, a directory and a broken link (all reported, in list order) and for the validation errors of T-001; a before/after snapshot of the temp tree is identical. After: T-001
- [x] T-003 Add the mechanism's filesystem interface (`Lstat`, `Mkdir`, `OpenFile`, `Chmod`, `Remove`) with an OS implementation, and `CreateFeatureFiles(ctx, fsys, root, feature, files)`:
  - run `CheckFeatureFiles`;
  - render every file in memory with `text/template` and `missingkey=error`; a render failure names the path and the cause and creates nothing;
  - create missing parent directories (`DirMode`) and each file with `O_CREATE|O_EXCL` (`FileMode`), recording what was created;
  - return the created paths in list order.

  The existing `FS` and `Write` stay unchanged. Covers: FR-024, EC-018. Done when: golden tests of the created tree pass (contents and modes); a template that fails to render leaves the tree unchanged; a file created by a hook after the check makes the run fail and leaves no file of this run; `internal/gen` tests for `aicli new` still pass. After: T-002
- [x] T-004 Add the all-or-nothing rollback and cancellation to `CreateFeatureFiles`:
  - on any create or write failure, remove what this run created in reverse order, removing directories only if empty;
  - paths that cannot be removed are listed after a `could not remove:` continuation line, in creation order;
  - the context is checked before each creation, and on cancellation the rollback completes and `context.Cause(ctx)` is returned.

  Covers: FR-024, EC-023. Done when: tests with a fake filesystem pass for: a forced write failure (nothing of this run left); a parent directory kept because something else is inside it; a failing `Remove` that gives the `could not remove:` lines (golden); a context cancelled with a `*scaffold.SignalError` between two files (the rollback is complete and that cause is returned). After: T-003

## Phase 2: `internal/feature` (contract, helper, run)

- [x] T-005 Create `internal/feature` with the `Input` type (name, description, choices, validator `func(string) string`) and the `Declaration` type (name, short, long, inputs, `Plan func([]string) ([]gen.FeatureFile, error)`, next steps). Add `NewCommand(decl)`, which builds a command with `Use` `<name> <input-1> …`, accepts any number of arguments and registers the declaration in a registry keyed by the command. Add `Lookup(cmd)`, which reads it back. Covers: FR-003, FR-006. Done when: unit tests pass: `Lookup` finds the declaration of a built command and finds none for a plain Cobra command; under a test `add`, `UseLine()` is `aicli add <name> <input-1> <input-2> [flags]` once the help flag is initialised. After: T-001
- [x] T-006 Add `CheckContract(add)`. It covers the direct runnable subcommands of `add` except `help`, and returns one error per broken rule, naming the feature and the rule: aliases; empty or duplicate input name; empty choice list; empty or duplicate choice; choices together with a validator; no declaration or no plan. Covers: FR-003. Done when: a unit test passes for one test-only feature per rule, each giving exactly its rule, and for a valid feature, which gives no error. After: T-005
- [x] T-007 Add `ValidateValues(cmd, decl, args)`. It trims every argument, then checks in this order:
  - too few arguments (`<feature> requires <input>`, naming the first missing input);
  - too many arguments (`<feature> takes <n> argument(s), got <m>`);
  - then, input by input in declaration order: empty, control characters (C0/C1, `DEL`), not one of the choices (exact match, listing every choice), and the validator, called only after the first two checks pass.

  Each error's second line is `Usage: ` followed by the usage line. Covers: FR-006, FR-014, FR-016, EC-009, EC-013. Done when: table tests pass for each failure and for the order between them, including choices in a different letter case; a test validator records that it never receives an empty value or a value with control characters; the plain messages match golden files. After: T-005
- [x] T-008 [P] Add `EquivalentCommand(name, values)`. A value is shown as is when it is non-empty ASCII made of letters, digits and `_ @ % + = : , . / -`; otherwise it is single-quoted, with `'` written as `'\''`. `--` goes before the first value when any value starts with `-`. Covers: FR-018, EC-015, EC-016. Done when: a table test passes for plain values, spaces, quotes, an empty value, non-ASCII text, shell metacharacters and a leading `-`; for each case, splitting the output with a POSIX-shell word-splitting helper written in the test gives back the original values. After: T-005
- [x] T-009 [P] Add `tui.RenderAdded(feature, paths, steps, styled)`. It prints `create  <path>` per file, a blank line and `Added <feature>`; when there are steps, it adds a blank line, `Next steps:` and each step indented by two spaces. When styled it uses the design's colours and the `✓ ` prefix. Covers: FR-025. Done when: golden tests pass for plain output with and without steps, and the styled output, stripped of ANSI and of `✓ `, equals the plain one. After: T-005
- [x] T-010 Add `Run(ctx, cmd, values, out, getenv)`:
  1. validate the values;
  2. compute the plan; a failure gives `<message>` and creates nothing;
  3. call `gen.CreateFeatureFiles` with the root from `project.RootFrom`;
  4. write `RenderAdded` in one write, styled by `tui.ShouldStyle(out, getenv)`.

  The `RunE` built by `NewCommand` only calls `Run` with `os.Getenv`. Covers: FR-006, FR-019, FR-025. Done when: unit tests with a temp project root in the context pass: valid values create the files and print the plain report; invalid values and a failing plan create nothing and write nothing to `out`. After: T-004, T-007, T-009
- [x] T-011 Create `internal/featuretest`, a package imported only by tests. It provides:
  - the test feature: a choice input, a text input with a validator, two feature files from an in-memory template FS, next steps, and a plan that fails for one choice;
  - one test-only feature per contract rule;
  - a hidden feature, a deprecated feature, a feature with no inputs, a feature without a short description, and a feature with a wide-character name;
  - helpers that register a chosen set under a given `add` command.

  Covers: FR-002, FR-003, EC-002, EC-006, EC-007, EC-008. Done when: `CheckContract` on a tree with the valid features gives no error, and on each rule-breaking feature gives exactly its rule; no non-test file imports the package (checked by a test that parses the imports of the module's non-test files). After: T-006, T-010

## Phase 3: `internal/add` (the `add` command's logic)

- [x] T-012 Add the `tui.Feature` type: name, short description, inputs (name, description, choices, validator), `Plan` returning the paths, and `Check` returning the read-only check error. Add `add.Features(add)`, which builds them from the tree on every call, in Cobra order, marking a feature as listed when it is available and not hidden. `Check` calls `gen.CheckFeatureFiles` with the project root. Covers: FR-002, EC-002, EC-007. Done when: unit tests on the T-011 tree pass: all features in order, hidden and deprecated ones not listed, a feature without a short description has an empty one, and a feature added to the tree after a first call appears on the next call with no code change. After: T-011
- [x] T-013 Add `add.Prepare(root)`: for each feature under `add` with Cobra's `Deprecated` set, clear it and set `Hidden`. Covers: EC-002. Done when: after `Prepare`, running the deprecated test feature directly through Cobra with valid values prints only the FR-025 report (no deprecation notice), and the feature is still not listed. After: T-012
- [x] T-014 [P] Add `tui.RenderAddStatic(features)`. It prints the header, the description line, a blank line, one line per listed feature (name padded to the widest by display width, two spaces, the short description or the name alone), a blank line and the closing line. With no listed feature, the output is the EC-003 text instead. Both are always plain. Covers: FR-005, EC-003, EC-006, EC-007. Done when: golden tests pass for the test features (including the wide-character and no-description ones) and for an empty list; no output contains an ESC byte. After: T-012
- [x] T-015 Add `add.Show(ctx, addCmd, args, in, out, getenv)` for every path except the interactive one:
  - an argument that is not the name of any feature returns the FR-007 error, with the `Did you mean` line only when `SuggestionsFor` finds listed features;
  - with no listed feature, it writes the EC-003 text in both modes;
  - in static mode (`tui.IsInteractive` false), it returns the context's cause if the context is cancelled, otherwise writes `RenderAddStatic` in one write.

  Covers: FR-005, FR-007, EC-003, EC-004, EC-005, EC-020. Done when: unit tests pass for:
  - an unknown name with and without a close listed feature (golden);
  - an unknown name in a tree whose only features are unlisted (no `Did you mean`);
  - buffers giving the static output;
  - a pipe on one side and a TTY stand-in on the other giving static mode;
  - a pre-cancelled context writing nothing and returning the cause.

  After: T-013, T-014

## Phase 4: `internal/tui` (screens)

- [x] T-016 Turn the home `listModel` into a list component configured with its top block, its footer, its quit keys and the action for `Enter`/click. The home screen keeps its exact behaviour. Covers: FR-009. Done when: every existing `internal/tui`, `internal/home` and `cmd/root` test and golden passes with no change to its expected output. After: T-012
- [x] T-017 [P] Add the add-screen styles from the design's colour table on the private ANSI256 renderer, all no-ops when not styled. Covers: FR-008, FR-011, FR-016, FR-018. Done when: a unit test renders each style in both modes; the unstyled output has no ESC byte; the styled output, stripped of ANSI, equals the unstyled one; the `Add` and `Cancel` styles differ, focused or not. After: T-012
- [x] T-018 Add the selector:
  - a layout view kind whose top block is the header, the description line and a blank line, with the EC-010 hiding order (description line, indicators, footer, header);
  - a selector model on the list component: `aicli › add` header, entries with the feature name padded by display width and the short description, footer `↑/↓ move · enter select · q quit`;
  - `Enter`/click selects; `q`, `Esc` and `Ctrl+C` quit; no banner.

  Covers: FR-008, FR-009, FR-010, EC-006, EC-010, EC-011. Done when: a layout table test over heights from 1 row up passes; unit tests on `Update`/`View` pass for the header, entries, pointer start, select by key and by click, and the three quit keys. After: T-016, T-017
- [x] T-019 Add the form layout, a pure function of the size, the description, the field heights and the focused field. It has the header, the description, the two indicator lines (`↑ more`, `↓ more`, otherwise blank), the fields and the footer. Fields scroll as whole blocks, keeping the focused field and its error visible. The EC-010 hiding order applies. Covers: FR-011, EC-010. Done when: a table test passes for a tall terminal (all fields visible), short terminals (scrolling, focus visible with its error) and the hiding order down to one row. After: T-018
- [x] T-020 Add the text field on `bubbles/textinput`:
  - title with `> ` or two spaces, description below, input line that scrolls horizontally;
  - C0/C1 controls and `DEL` are removed from typed and pasted runes before they reach `textinput`;
  - on validation the value is trimmed, an empty value gives `<input> cannot be empty`, and otherwise the validator's message is shown under the field;
  - only the keys listed in EC-011 act.

  Covers: FR-011, FR-013, FR-014, EC-011, EC-012, EC-013. Done when: unit tests pass for a value wider than the field (cursor visible, full value kept), a paste with newlines, tabs and `DEL` (discarded, the rest kept, no submit), the empty error, the validator error, and an unlisted key. After: T-019
- [x] T-021 Add the choice field:
  - while focused, every choice on its own line with `(•) ` or `( ) `; `↑`/`↓`/`k`/`j` move the selection and stop at the ends; the selection starts on the first choice;
  - while not focused, only the selected choice;
  - when the field is taller than its space, the choices scroll inside it and the selection stays visible;
  - a click on an option selects it.

  Covers: FR-015, FR-016, EC-010, EC-011. Done when: unit tests on `Update`/`View` pass for each behaviour, including a list of choices taller than the space. After: T-019
- [x] T-022 Add the form model:
  - `aicli › add › <feature>` header, the short description line (none and no blank line when it is empty), fields in declaration order with the focus on the first;
  - `Tab`/`Enter` validate and move to the next field; `Shift+Tab` and clicks move the focus without validating; `Enter` on the last field validates all fields and either asks for the confirmation or moves the focus to the first invalid field;
  - `Esc` returns to the selector on the same feature and keeps the values; reopening the same feature shows them with the focus on the first field; another feature starts with its own values;
  - per-field footers as in the design; clicks outside fields do nothing.

  Covers: FR-011, FR-012, FR-013, FR-015, FR-017, EC-008, EC-011, EC-014. Done when: unit tests on `Update`/`View` pass for each behaviour above, including that `Esc` and `Ctrl+C` never produce a run request. After: T-020, T-021
- [x] T-023 Add the confirmation layout, a pure function of the size, the description, the wrapped command and the path count. It covers the file-list heading, the two indicator lines, the scrolling paths with `↑/↓ scroll` added to the footer, the buttons, and the EC-010 hiding order (file list block, description, indicators, footer, header). As a last step, the command scrolls above the buttons, starting at its first line. Covers: FR-018, EC-010. Done when: a table test passes for: everything visible; paths scrolling; the file list hidden; and a wrapped command taller than the terminal, keeping the buttons and one command line. After: T-019
- [x] T-024 Add the confirmation model:
  - header, description line, the equivalent command (from `feature.EquivalentCommand`, wrapped and never truncated), `Will create:` and the plan paths indented by two spaces;
  - a failing plan or check replaces the list with the error text (no `Error: `, capitalised, red when styled) and shows only `Cancel`, where `y`, `←`, `→` and `Tab` do nothing;
  - buttons `[ <label> ]` and `  <label>  `, with the focus starting on `Cancel`; `←`/`→`/`Tab` move the focus; `Enter` presses the focused button; `y` is `Add`; `n` is `Cancel`;
  - `Esc` returns to the form with its values, or to the selector when the feature has no inputs;
  - entering the confirmation returns the disable-mouse command, and `Esc` returns the enable command;
  - the plan and the check are computed again each time the confirmation is shown.

  Covers: FR-012, FR-018, EC-010, EC-011, EC-017. Done when: unit tests pass for each behaviour, including the blocked confirmation for an existing file and for a failing plan, an edited value that shows a new command after `Esc` and `Enter`, and the mouse commands. After: T-008, T-022, T-023
- [x] T-025 Add the program model `AddModel`, which joins the selector, the form and the confirmation (a feature with no inputs goes straight to the confirmation), with an exported constructor. Add `RunAdd(ctx, features, in, out, styled)`, which returns `Result{Action, Index, Values}`:
  - it uses the alternate screen, cell-motion mouse, the given input and output, the context and no signal handler;
  - quitting returns `quit`; `Cancel` returns `cancel`; `Add` returns `add` with the trimmed values;
  - a start or run failure returns `cannot start the add screen: <cause>` after the terminal is restored;
  - a cancelled context returns `context.Cause(ctx)`.

  Covers: FR-010, FR-019, FR-021, EC-014, EC-020. Done when: runner tests with pipe input and buffer output pass:
  - `q` returns `quit` and the output ends by leaving the alternate screen;
  - a context cancelled with a `*scaffold.SignalError` returns that cause after leaving the alternate screen;
  - a closed input file returns the `cannot start the add screen:` error;
  - a scripted select, fill and `y` returns `add` with the values.

  After: T-018, T-024

## Phase 5: wiring (`internal/add`, `cmd/add`, `cmd/root`)

- [x] T-026 Complete `add.Show` with the interactive path. It runs `tui.RunAdd` with `tui.ShouldStyle(out, getenv)`:
  - `quit` returns nil and prints nothing;
  - `cancel` writes `Nothing added.` plain to stdout;
  - `add` calls `feature.Run` on the selected command, with the add command's context and stdout, after the program has returned.

  Covers: FR-010, FR-018, FR-019, EC-015, EC-019. Done when: unit tests with a stubbed runner pass: for each action, the stdout content and the files created are checked; a value starting with `-` reaches the feature as a value; a feature failure returns its error with stdout empty. After: T-015, T-025
- [x] T-027 Create `cmd/add` with `NewCommand()`: `Use` `add`, short `Add an AI/LLM feature to this project.`, any number of arguments, and a `RunE` that only calls `add.Show` with the command's context, `InOrStdin`, `OutOrStdout` and `os.Getenv`. Register it in `cmd/root`, and make `run` call `add.Prepare` before executing. The exempt list is unchanged. Covers: FR-001, FR-002. Done when: `go build ./...` succeeds; `aicli add` inside a test project root prints the EC-003 text; `cmd/add` imports no other `cmd/` package and names no feature; the existing `cmd/root` and `cmd/new` tests pass, except the real-tree home golden, which T-028 regenerates. After: T-026

## Phase 6: end-to-end, teatest and golden tests

- [x] T-028 In `cmd/root`, regenerate the real-tree home golden with `-update` (it now shows `add`), and add a home static test with the T-011 tree that shows `add <feature>` for listed features only. Covers: FR-004, EC-002. Done when: both tests pass, and the diff of the regenerated golden is only the `add` line. After: T-027
- [x] T-029 [P] Project-root tests in `cmd/root`. Outside a project root, `aicli add`, `aicli add <feature>` (with valid values) and `aicli add <unknown>` each fail with spec 002's message, create nothing and show no screen. `aicli add --help` and `aicli help add` work outside a project. The exempt-list pin test still pins the same list. Covers: FR-001, EC-001. Done when: the tests pass. After: T-027
- [x] T-030 [P] End-to-end direct-call tests through `run` with the T-011 tree in a temp project root:
  - valid values create the files and print the report;
  - golden files for the plain errors of FR-006 (each check, including control characters, choices and validator), of FR-007 (with and without `Did you mean`), of FR-023 and of a failing plan;
  - each styled error, stripped of ANSI and of `✗ `, equals the plain one, with stdout empty and exit code 1;
  - a hidden feature and a deprecated feature run directly with no notice.

  Covers: FR-006, FR-007, FR-012, FR-023, FR-025, FR-026, EC-002, EC-009, EC-013, EC-016. Done when: the tests pass. After: T-027
- [x] T-031 Teatest flow for the selector, with the T-011 features, unstyled:
  - the header and entries in order, pointer on the first;
  - moves by keys and wheel, scroll indicators, resize;
  - `Enter` and a click open the form;
  - `q`, `Esc` and `Ctrl+C` restore the terminal and print nothing;
  - unlisted keys and clicks outside entries do nothing;
  - no banner.

  Covers: FR-008, FR-009, FR-010, EC-006, EC-011. Done when: every case passes. After: T-026
- [x] T-032 Teatest flow for the form:
  - fields in declaration order with names and descriptions;
  - `Tab`/`Enter`/`Shift+Tab` and clicks on every part of a field;
  - empty values and validator errors;
  - a choice field with keys and click;
  - horizontal scroll of a long value; pasted control characters discarded;
  - scrolling on a short terminal;
  - `Esc` back to the selector, and reopening the feature shows the same values;
  - `Ctrl+C` after filling fields creates nothing;
  - the mouse is captured.

  Covers: FR-011, FR-013, FR-014, FR-015, FR-016, FR-017, EC-010, EC-012, EC-013, EC-014. Done when: every case passes. After: T-026
- [x] T-033 Teatest flow for the confirmation and the run:
  - the equivalent command and `Will create:`;
  - focus on `Cancel`; `←`/`→`/`Tab`, `Enter`, `y`, `n`;
  - `Cancel` prints `Nothing added.` after the terminal is restored;
  - `Add` creates the files and prints the report after the terminal is restored;
  - `Esc` back to the form, editing a value, and the new command shown;
  - a feature with no inputs goes straight to the confirmation, and `Esc` returns to the selector;
  - the confirmation is blocked by an existing file and by a failing plan;
  - a file created between the confirmation and `Add` makes the run fail with the FR-023 error, nothing replaced, and the terminal restored;
  - the mouse is released, and captured again after `Esc`.

  Covers: FR-012, FR-018, FR-019, FR-020, EC-008, EC-015, EC-017, EC-018, EC-019. Done when: every case passes. After: T-032
- [x] T-034 [P] Teatest with styling off and on:
  - off: `> `, `(•) ` and `[ … ]` mark the focus with no colour sequence;
  - on: the highlighted entry and the focused title change with focus, and `Add` and `Cancel` use two different colours, focused or not.

  Covers: FR-009, FR-011, FR-016, FR-018. Done when: the tests pass. After: T-033
- [x] T-035 [P] Layout teatests (EC-010) on small terminals: a long file list scrolling under its heading, a choice field taller than the space, and a wrapped equivalent command taller than the terminal that scrolls above the buttons; truncation with `…` of the header, descriptions, choices, errors and paths, never wrapped. Covers: EC-010. Done when: the tests pass. After: T-033
- [x] T-036 [P] EC-003 and static goldens:
  - the static output golden with the T-011 tree, including the closing line, and with the real tree (no features);
  - a teatest showing that with no listed features the interactive mode prints the same plain text and never enters the alternate screen.

  Covers: FR-005, EC-003, EC-005. Done when: the tests pass. After: T-027
- [x] T-037 [P] Signal tests:
  - a test-only binary, built from a `testdata` main package that registers the test feature with a filesystem that blocks between files, receives SIGINT during creation; it exits 130, prints nothing and leaves no file of the run;
  - a context cancelled with a `*scaffold.SignalError` in static mode and while the screens are shown gives 128+n through `run` with nothing printed.

  Covers: EC-020, EC-023. Done when: the tests pass on macOS and Linux. After: T-027
- [x] T-038 [P] FR-020 tests:
  - a working-directory snapshot is unchanged after a teatest run that quits from the confirmation;
  - a test parses the Go files of `internal/add` and of the add screens in `internal/tui` and fails on imports of `os/exec`, `net` or `net/http`.

  Covers: FR-020. Done when: the tests pass. After: T-033
- [x] T-039 [P] Contract test on the real tree: `CheckContract` on the `add` built by `NewRoot` gives no error. Covers: FR-003. Done when: the test passes. After: T-027

## Phase 7: quality and Definition of Done

- [x] T-040 Quality sweep:
  - run `gofmt` and `golangci-lint run`;
  - a doc comment on every new function, method, type and constant;
  - a `Covers:` line on every new test;
  - only `internal/tui` imports Bubble Tea, Bubbles or Lipgloss (ARC-7);
  - no `cmd/` package imports another `cmd/` package except `cmd/root` (ARC-4);
  - `go.mod` is unchanged.

  Covers: FR-001, FR-002, FR-003, FR-004, FR-005, FR-006, FR-007, FR-008, FR-009, FR-010, FR-011, FR-012, FR-013, FR-014, FR-015, FR-016, FR-017, FR-018, FR-019, FR-020, FR-021, FR-022, FR-023, FR-024, FR-025, FR-026. Done when: `gofmt -l .` prints nothing; `golangci-lint run` and `go test ./...` pass; `go mod tidy` leaves no diff; a grep finds a `Covers:` line for every FR-001..FR-026 and EC-001..EC-023. After: T-028, T-029, T-030, T-031, T-032, T-033, T-034, T-035, T-036, T-037, T-038, T-039
- [x] T-041 Check every Definition of Done item in `spec.md` and tick the ones verified. Covers: all. Done when: every DoD item in `spec.md` is ticked, and the evidence for each (test name or command output) is reported. After: T-040
