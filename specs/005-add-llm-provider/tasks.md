# Tasks: 005-add-llm-provider

- Status: approved

No Go dependency is added, so in every task `go mod tidy` must leave no diff. The generated-project packages `openai>=3.36`, `anthropic>=2.0` and `google-genai>=2.32` are the ones approved by the spec, with the bounds fixed in the plan. Every new test has a `Covers:` line. No generated file cites a spec ID.

## Phase 1: `internal/gen` (file actions)

- [ ] T-001 Add the plan types and the extended path validation:
  - `FeatureFile` gains `Action` (`ActionCreate` as the zero value, `ActionShared`, `ActionInject`) and `Marker`;
  - new `Plan{Entries, ItemNoun, Items []Item{Value, Path}}`;
  - path validation as in spec 004, with duplicate paths allowed only between `inject` entries for the same target;
  - an `inject` target that is a symbolic link is resolved and must lie inside the resolved root.

  Covers: FR-008, EC-014. Done when: table tests pass for two `inject` entries on one path (accepted), a `create` and an `inject` on one path (rejected), a `shared` duplicate (rejected), and an `inject` target linking outside the root (rejected, naming the path); every existing `internal/gen` test still passes.
- [ ] T-002 [P] Add the pure injection engine. It takes the original bytes, the marker name and the blocks, and returns the new bytes and whether they changed:
  - the marker is a line that, trimmed, is exactly `# aicli:<name>`; none and more than one are distinct errors;
  - a block is skipped when its lines already appear as consecutive trimmed lines, including in blocks inserted earlier in the same call;
  - otherwise the whole block is inserted, with blocks for one marker placed one after another right below it, in order;
  - each non-empty line gets the marker's leading whitespace plus its own relative indentation; empty lines stay empty;
  - the line ending is the marker's; a marker on the last line with no line ending gets the first line ending of the file, or `\n`;
  - no other byte changes.

  Covers: FR-009, EC-011, EC-012, EC-025, EC-030. Done when: a table test passes for an indented marker, a `\r\n` file, an entry already present, a line with a different version bound (inserted), several entries at one marker in plan order, a multi-line block with blank and repeated lines (inserted whole), the same block twice (inserted once), a marker on the last line with no line ending, no marker, and a repeated marker. After: T-001
- [ ] T-003 Add `CheckPlan(root, feature, plan) (Preview, error)`. It is read-only and checks in this order:
  1. path validation (T-001);
  2. `inject` targets: does not exist (including a broken link), not a regular file once resolved, no marker, repeated marker. The first failing entry in plan order gives the fixed FR-010 message;
  3. already added items: `cannot add <feature>, these <items> are already added:` with one `<value> (<item file>)` continuation line per added value, in choice order;
  4. the spec 004 FR-023 existence list: existing `create` paths, plus `shared` paths that exist but are neither a regular file nor a link to one inside the root, in plan order.

  `Preview.Create` holds the `create` entries and the missing `shared` entries. `Preview.Update` holds the targets whose injection would change them, computed with the T-002 engine.

  Covers: FR-003, FR-010, FR-011, EC-004, EC-005, EC-010, EC-028. Done when: unit tests pass for each FR-010 message, for an item file that is a regular file, a directory and a broken link, for mixed added and new values (only the added ones listed, even though their test files exist), for the fallback to spec 004 FR-023 when no value is added, for a `shared` path that is a directory and one that is a broken link, and for a `Preview` that leaves out existing `shared` files and injections that change nothing. The plain messages match golden files, and a before/after snapshot of the temp tree is identical. After: T-002
- [ ] T-004 Add `ReadFile` and `Stat` to `FeatureFS` and add `ApplyPlan(ctx, fsys, root, feature, plan) ([]ReportLine, error)`:
  - run `CheckPlan` again just before writing;
  - render every entry in memory (`missingkey=error`);
  - create `create` entries and missing `shared` entries with `O_EXCL`, and skip existing `shared` entries;
  - write each changed `inject` target once, in place at its resolved path (`O_WRONLY|O_TRUNC`), keeping the link and the permissions;
  - return one `create`, `skip` or `update` line per path in plan order. A target injected by several entries is reported once, at its first entry, and a target that does not change is `skip`.

  Covers: FR-008, FR-009, FR-011, EC-007, EC-009, EC-029. Done when: unit tests pass for:
  - a first run (all `create` plus `update`);
  - a later run (`skip` for the existing `shared` files);
  - a deleted `shared` file created again and an edited one left untouched;
  - an item file created after the check, which fails without replacing it;
  - a target reached through a link inside the root (the link stays a link, the file keeps its mode).

  After: T-003
- [ ] T-005 Extend the rollback of `ApplyPlan`:
  - on any failure or cancellation, injected files are written back byte for byte at their resolved path, and created files and directories are removed, in reverse order;
  - files that cannot be restored or removed are listed in the rollback error of spec 004;
  - the context is checked before each step, and on cancellation the rollback completes and `context.Cause(ctx)` is returned.

  Covers: FR-011, EC-013, EC-029. Done when: tests with a fake filesystem pass for a forced write failure on the second injected file, a failure after creations and injections (every byte and mode restored, the link still a link), a failing restore (listed in the rollback error, golden), and a context cancelled with a `*scaffold.SignalError` between two steps. After: T-004
- [ ] T-006 Make `CheckFeatureFiles` and `CreateFeatureFiles` thin wrappers over `CheckPlan` and `ApplyPlan`, or remove them if nothing calls them any more. Covers: FR-008. Done when: every spec 004 test and golden in `internal/gen`, `internal/feature`, `internal/add` and `cmd/root` passes unchanged. After: T-005

## Phase 2: `internal/feature` (contract and values)

- [ ] T-007 Extend the declaration:
  - `Input` gains `Multi bool`, `ItemNoun string` and `ItemFiles map[string]string`;
  - `Declaration.NextSteps` becomes `func(values []string) []string`, and the spec 004 test feature returns its fixed steps through it;
  - for a multi-choice input, `NewCommand` writes the input in `Use` as `<input>...`.

  Covers: FR-001, FR-002, FR-014. Done when: a test feature with a multi-choice input has `UseLine()` `aicli add <name> <input>... [flags]`; every spec 004 test and golden passes unchanged. After: T-006
- [ ] T-008 Extend `ValidateValues` for a multi-choice input:
  - it takes all remaining arguments; "too few" means no value at all (`<feature> requires <input>`) and there is no "too many";
  - each value, in the given order, is checked for empty, control characters, membership in the choices (exact match), and repetition after trimming (`<input> "<value>" is given more than once`, naming the first repeated value);
  - the values are returned in choice declaration order.

  Covers: FR-002, FR-004, EC-001, EC-002, EC-003. Done when: table tests pass for no value, `openai openai`, ` openai` and `openai`, `OpenAI`, `anthropic`, `chatgpt`, and the order of checks between them; `ollama openai` and `openai ollama` return the same list and the same `EquivalentCommand`; the plain messages match golden files. After: T-007
- [ ] T-009 Add the multi-choice rules to `CheckContract`, each with its own error naming the feature:
  - a multi-choice input that is not the last input;
  - a multi-choice input with no choices, or with empty or repeated choices;
  - a multi-choice input with a validator;
  - an item file that is not a `create` entry of the plan for every subset that includes its choice;
  - a `shared` entry whose rendered bytes differ between non-empty subsets of the values.

  Covers: FR-001. Done when: the T-011 rule-breaking features each give exactly their rule, and the valid multi-choice test feature gives no error. After: T-008, T-011
- [ ] T-010 [P] Change `tui.RenderAdded` to take report lines: verbs `create`, `skip` and `update`, each padded to 6 characters plus two spaces. When styled, `create` is green, `update` is 117 and `skip` is 245. Covers: FR-011. Done when: golden tests pass for a report with all three verbs, plain and styled; the styled output, stripped of ANSI and of `✓ `, equals the plain one; the spec 004 report goldens are unchanged. After: T-004
- [ ] T-011 Extend `internal/featuretest` with:
  - a multi-choice test feature on an in-memory template FS, with item files, two `shared` entries, `create` entries per value, and `inject` entries at two markers;
  - one test-only feature per multi-choice contract rule.

  Covers: FR-001. Done when: the features build, the valid one passes the spec 004 contract rules, and no non-test file imports the package. After: T-007
- [ ] T-012 Update `feature.Run`: it builds the `gen.Plan` from the declaration (entries, item noun and item files of the given values), calls `gen.ApplyPlan`, and writes `RenderAdded` with the report lines and `NextSteps(values)`. Covers: FR-004, FR-011, FR-014. Done when: unit tests with the T-011 feature in a temp project root pass: values in any order give the same files and the same report; the next steps depend on the values; an already added value creates and modifies nothing and writes nothing to `out`. After: T-008, T-010

## Phase 3: `internal/tui` (screens)

- [ ] T-013 [P] Add the new styles from the design's colour table: checklist cursor 117 bold, selected 117, unselected 255, already added 245 (whole line), `All … already added.` 245, and `Will update:` 245. All are no-ops when not styled. Covers: FR-006, FR-007, FR-011. Done when: a unit test renders each style in both modes; the unstyled output has no ESC byte, and the styled output stripped of ANSI equals the unstyled one. After: T-012
- [ ] T-014 Add `multiChoiceField`:
  - while focused, every choice on its own line in declaration order with `[x] ` or `[ ] `, and already added choices as `[✓] <choice> (already added)`;
  - the cursor `> ` starts on the first choice that is not already added, and is not shown if every choice is already added;
  - `↑`/`↓`/`k`/`j` move the cursor, skip already added lines and stop at the ends;
  - `Space` toggles the choice under the cursor; a click moves the cursor there and toggles it; both do nothing on an already added line;
  - while not focused, the selected choices joined with `, `, or nothing;
  - the choices scroll inside the field, keeping the cursor visible;
  - the value is the selected choices in declaration order.

  Covers: FR-006, FR-007, EC-005, EC-015. Done when: unit tests on `Update`/`View` pass for each behaviour, including deselecting with `Space`, a click on an already added line, a list taller than its space, and selections made in a different order (value still in declaration order). After: T-013
- [ ] T-015 Extend `FeatureInput` (`Multi`, `ItemNoun`, `Added []bool`) and the form model:
  - a multi-choice field is validated only on `Enter`, showing `select at least one <input>`; `Shift+Tab` and clicks leave it without validating;
  - when every choice is already added, it shows `All <items> are already added.` in place of the error, `Enter` does nothing, and the footer does not list `enter`;
  - the footer lists `space toggle`;
  - in direct mode `Esc` quits and the footer reads `esc quit`; from the selector `Esc` goes back and reads `esc back`.

  Covers: FR-005, FR-006, EC-006, EC-016. Done when: unit tests on `Update`/`View` pass for each behaviour, and the multi-choice value reaches the confirmation as one value per selected choice. After: T-014
- [ ] T-016 Let `NewAddModel`/`RunAdd` start either on the selector or on the form of one feature in direct mode, and fill `Added` from `Feature.Added()` when the form opens. `Esc` in the confirmation still returns to the form. Covers: FR-005, FR-007, EC-016. Done when: runner tests pass: started in direct mode, `Esc` on the form returns `quit` after leaving the alternate screen; started on the selector, `Esc` on the form returns to the selector; `Added` is read once, when the form opens. After: T-015
- [ ] T-017 Change the confirmation to show two lists:
  - the confirmation layout takes the total row count of `Will create:`, its paths, a blank row, `Will update:` and its paths, so that both lists scroll together with the spec 004 indicators and keys;
  - each heading appears only when it has paths;
  - `Feature.Check` returns the `Preview`, and a check error replaces both lists with only `Cancel`, as before.

  Covers: FR-011, EC-007, EC-010. Done when: a layout table test passes for both lists visible, only one list, and both scrolling on a short terminal; unit tests pass for the first-run and later-run lists of the design and for the blocked confirmation with a missing marker and with an item added after the form opened. After: T-016

## Phase 4: `internal/add` (direct form and items)

- [ ] T-018 Extend `add.Features`: each `tui.Feature` carries `Multi`, `ItemNoun`, an `Added()` function that only stats the item files, and a `Check` that returns the `Preview` of `gen.CheckPlan` with the project root. Covers: FR-007, FR-011, EC-005. Done when: unit tests on the T-011 tree pass: `Added()` reports a regular file, a directory and a broken link as added; `Check` returns the expected `Preview`; a before/after snapshot of the temp tree is identical. After: T-017
- [ ] T-019 Extend `add.Prepare`: it wraps the `RunE` of each feature with inputs. With no positional argument, no changed flag and interactive mode, the wrapper runs the add screens with that feature alone in direct mode and handles the result as `runInteractive` does (`quit` prints nothing, `cancel` prints `Nothing added.`, `add` calls `feature.Run`). Otherwise it calls the original `RunE`. Covers: FR-005, EC-001, EC-016. Done when: unit tests with a stubbed runner pass: interactive with no arguments opens the direct form; static mode with no arguments fails as in spec 004 FR-006 with `Error: <feature> requires <input>`; arguments or a flag skip the form; a feature with no inputs runs with no form and no confirmation, also in interactive mode; calling `Prepare` twice wraps once. After: T-018

## Phase 5: the `llm-provider` feature (Go side and skeleton)

- [ ] T-020 Add `internal/llmprovider` with `Declaration()`:
  - name `llm-provider`, short `Add one or more LLM providers.`;
  - the multi-choice input `provider` (`LLM providers to add`), choices `openai, claude, gemini, ollama`, noun `providers`, item files `app/ai/llm/<provider>.py`;
  - `Plan` in the order of FR-013: six `shared` entries; `app/ai/llm/<provider>.py` and `tests/test_llm_<provider>.py` per provider; `app/api/router.py` at `# aicli:router-imports` and `# aicli:routers`; `pyproject.toml` at `# aicli:dependencies`, one entry per distinct SDK in order of first appearance; `.env.example` at `# aicli:env`, one entry per line, with `# OLLAMA_BASE_URL=http://localhost:11434/v1` after `OLLAMA_MODEL=`;
  - `NextSteps`: `Run uv sync to install the provider SDKs`, then `Set <VAR>, <VAR> in .env` per provider in choice order.

  Covers: FR-012, FR-013, FR-014, EC-008. Done when: unit tests pass for the entry order with one provider and with all four, `openai ollama` having the `openai` SDK once, and the next steps of `openai ollama`. After: T-012
- [ ] T-021 Add the injected-line templates in `templates/llmprovider/inject/` (`router_import`, `router_include`, `dependency`, `env`) and the `templates.LLMProvider` embed. Covers: FR-008, FR-013. Done when: golden tests of each rendered block pass, with the requirements `openai>=3.36`, `anthropic>=2.0` and `google-genai>=2.32`. After: T-020
- [ ] T-022 Add the skeleton markers: `# aicli:dependencies` as the last element of the `dependencies` array of `pyproject.toml.tmpl`; `# aicli:env` as the last line of `.env.example`; in `app/api/router.py`, `# isort: off` / `# aicli:router-imports` / `# isort: on` after the imports and `# aicli:routers` at the end. Regenerate the spec 001 skeleton goldens with `-update`. Covers: FR-015, EC-010. Done when: the skeleton golden diff shows only the marker lines, the spec 001 tree test passes unchanged (its FR-002), and a project generated before this change fails `CheckPlan` with `app/api/router.py has no "# aicli:router-imports" marker`. After: T-021
- [ ] T-023 Create `cmd/addllmprovider` with `NewCommand()`, which only returns `feature.NewCommand(llmprovider.Declaration())`. Register it under `add` in `cmd/root`. Regenerate the home and real-tree static `aicli add` goldens with `-update`. Covers: FR-012. Done when: `go build ./...` succeeds; `aicli add` statically lists `llm-provider  Add one or more LLM providers.`; the golden diffs show only that line; `cmd/addllmprovider` imports no other `cmd/` package; `CheckContract` on the real tree gives no error. After: T-022

## Phase 6: generated Python templates

Each template is rendered into a golden file (TST-2). Behaviour is checked by the generated tests (T-028, T-029) through the integration test (T-036).

- [ ] T-024 Add `shared/app/ai/llm/base.py`: the `Message` type (role `user` or `assistant`, text content), the async `CompletionClient` protocol (messages, optional system prompt, maximum tokens, returns the text), the provider descriptor (name, position, required variables, settings loader, client factory), provider errors, and `# aicli:capabilities`. No SDK import. Covers: FR-016, FR-022. Done when: its golden passes and contains `# aicli:capabilities` exactly once. After: T-023
- [ ] T-025 Add `shared/app/ai/llm/__init__.py`: the registry. It walks the package modules with `pkgutil`, keeps only those that expose a provider descriptor, uses only the `<name>.py` module when two declare the same name (ignores both if neither is), sorts by `(position, name)`, and for each provider returns its name, whether it is configured (a whitespace-only value counts as missing), its missing variables in declaration order, and whether its client implements a capability; it builds clients on demand. Covers: FR-016, EC-017, EC-020, EC-027, EC-031. Done when: its golden passes and imports no provider SDK. After: T-024
- [ ] T-026 Add `shared/app/schemas/llm.py`, `shared/app/services/llm.py` and `shared/app/api/routers/llm.py`, referring to each other by module:
  - schemas: completion request (`messages` non-empty, `system` optional, `max_tokens` 1..32768 default 1024), completion response (`provider`, `model`, `content`), health response, and `# aicli:schemas`;
  - service: `health()` and `complete()`, raising service errors for not found, not configured, not implemented and provider failure; on failure it logs only the exception type name; `# aicli:services`;
  - router: `GET /llm/health`, and `POST /llm/{provider}/completion`, which takes the provider as free text and the raw JSON body, answers `404`, then validates the body (`422`), then calls the service and maps `503`, `501` and `502` to `HTTPException`; the body schema is in `openapi_extra`; `# aicli:routes`.

  Covers: FR-018, FR-019, FR-020, FR-022, EC-019, EC-022. Done when: the three goldens pass and each contains its marker exactly once. After: T-025
- [ ] T-027 Add the four provider templates `provider/<provider>.py`:
  - settings through pydantic-settings with the provider's prefix and the root `.env`, no model default; an empty or whitespace-only optional variable takes its default;
  - a provider descriptor with the provider's name, position and required variables;
  - an async client implementing `CompletionClient` that imports its SDK only when it is built or used;
  - key, model and base URL passed explicitly: `https://api.openai.com/v1`, `https://api.anthropic.com`, `https://generativelanguage.googleapis.com/` (through `http_options.base_url`, with `vertexai=False` for `gemini`), and `OLLAMA_BASE_URL` (default `http://localhost:11434/v1`) with the fixed key `ollama`;
  - the `openai` and `ollama` clients also pass `Authorization: Bearer <their key>` in `default_headers`;
  - `# aicli:methods` inside the client class.

  Covers: FR-016, FR-017, FR-022, EC-023, EC-026. Done when: the four goldens pass, each with `# aicli:methods` exactly once and no top-level SDK import. After: T-024
- [ ] T-028 Add `shared/tests/test_llm_router.py`:
  - an `autouse` fixture that deletes every `OPENAI_`, `ANTHROPIC_`, `GEMINI_`, `GOOGLE_` and `OLLAMA_` variable and `chdir`s to `tmp_path`;
  - a fake client that implements `CompletionClient` and one that does not;
  - tests for `/llm/health` (`ok` with no provider, `degraded`, order, `missing` without values), `200` on completion, `404`, `422`, `503`, `501`, `502` and their precedence, whitespace-only values, a foreign module ignored, a user `mistral` module found at its position, duplicate names and tied positions, and a re-created `__init__.py`.

  No test makes a network call or needs an SDK installed. Covers: FR-018, FR-019, FR-020, FR-021, EC-017, EC-019, EC-020, EC-022, EC-024, EC-027, EC-031. Done when: its golden passes. After: T-026
- [ ] T-029 Add the four `provider/test_llm_<provider>.py` templates, with the same isolation fixture:
  - settings and missing variables, including the `ollama` default for an empty `OLLAMA_BASE_URL`;
  - the provider is registered at its position;
  - with a fake SDK in `sys.modules`, the client passes the key, model and base URL from its own settings, never from `OPENAI_API_KEY`, `GOOGLE_API_KEY` or `OPENAI_CUSTOM_HEADERS`, and the `ollama` client is built with `OPENAI_API_KEY` unset;
  - with the SDK set to `None` in `sys.modules`, completion answers `502`, the log has only the exception type, and `/llm/health` still works.

  Covers: FR-017, FR-021, EC-018, EC-023, EC-024, EC-026. Done when: the four goldens pass. After: T-027, T-028

## Phase 7: tests

- [ ] T-030 [P] Golden test of the injected files: after `openai ollama` and then `claude`, the rendered `pyproject.toml`, `.env.example` and `app/api/router.py` of a scaffolded project match their goldens; a test checks that every file of FR-022 has its marker exactly once in a project with all four providers, and once in a project with only markers and no injected lines. Covers: FR-013, FR-015, FR-022, EC-008, EC-012. Done when: the tests pass. After: T-029
- [ ] T-031 [P] End-to-end error tests through `run` in a scaffolded project, with golden files for the plain errors: missing value, repeated value, value outside the choices (FR-002); already added, and the fallback to spec 004 FR-023 (FR-003); missing target, target that is not a regular file, missing marker, repeated marker (FR-010). Each styled error, stripped of ANSI and of `✗ `, equals the plain one, with stdout empty and exit code 1. Covers: FR-002, FR-003, FR-010, EC-001, EC-002, EC-003, EC-004, EC-010, EC-014. Done when: the tests pass. After: T-029
- [ ] T-032 [P] End-to-end report tests through `run`, with plain and styled goldens for: a first run with `openai ollama`; a later run with `claude` (`skip` and `update` lines); `ollama` after `openai` (`skip    pyproject.toml`); and a run where injection changes nothing. Covers: FR-004, FR-011, FR-014, EC-008, EC-009, EC-011. Done when: the tests pass, and `aicli add llm-provider ollama openai` gives the same output as `aicli add llm-provider openai ollama`. After: T-029
- [ ] T-033 [P] FR-005 mode tests in `cmd/root`: in static mode, `aicli add llm-provider` with no arguments fails with `Error: llm-provider requires provider` and the usage line, creating nothing; a feature with no inputs called with no arguments runs without a form or a confirmation, in static and interactive mode. Covers: FR-005, EC-001. Done when: the tests pass. After: T-029
- [ ] T-034 Teatest flows for `llm-provider`, unstyled unless stated:
  - the direct form of `aicli add llm-provider` (header, description, checklist, footer of the design);
  - keys, `Space`, click and the `> `/`[x]`/`[ ]` markers with styling off;
  - `(already added)` lines skipped by the cursor, and every provider already added (no cursor, message, no `enter` in the footer);
  - `Enter` with nothing selected shows `select at least one provider`;
  - `Esc` in the direct form restores the terminal, prints nothing and creates nothing; `Esc` in a form opened from the selector returns to it;
  - the confirmation with `Will create:` and `Will update:` for the first and later runs of the design, scrolling together on a short terminal;
  - an item file created after the form opened blocks the confirmation with the FR-003 error;
  - `Add` creates the files and prints the report after the terminal is restored;
  - styled: the colours of the design table.

  Covers: FR-005, FR-006, FR-007, FR-011, EC-006, EC-007, EC-015, EC-016. Done when: every case passes. After: T-029
- [ ] T-035 [P] Unit and contract test sweep: `CheckContract` on the real tree and on each multi-choice rule-breaking feature; a test renders each `shared` entry of `llm-provider` for all 15 non-empty sets of providers and checks they are byte-identical. Covers: FR-001, FR-012. Done when: the tests pass. After: T-029
- [ ] T-036 Integration test (`integration` tag, in `cmd/root`). Its doc comment carries the `Covers:` line of the generated-app requirements. Steps:
  1. Scaffold a project and run `uv sync`.
  2. Run `aicli add llm-provider openai`.
  3. Run pytest, `ruff check` and `ruff format --check` with `uv run --no-sync`.
  4. Run `uv sync`, then pytest, `ruff check` and `ruff format --check` again.
  5. Run `ruff check --fix` and `ruff format`.
  6. Run `aicli add llm-provider claude gemini ollama`, and check that `app/api/router.py` has the `llm` import and `include_router` lines once each.
  7. Repeat steps 3 and 4.

  It runs with a `.env` holding fake provider values and with provider variables exported, and the generated tests still pass and make no network call. Covers: FR-015, FR-016, FR-017, FR-018, FR-019, FR-020, FR-021, FR-022, EC-012, EC-017, EC-018, EC-019, EC-020, EC-021, EC-022, EC-023, EC-024, EC-026, EC-027, EC-031. Done when: `go test -tags integration ./cmd/root -run LLMProvider` passes on macOS and Linux. After: T-030, T-031, T-032, T-033, T-034, T-035

## Phase 8: quality and Definition of Done

- [ ] T-037 Quality sweep:
  - run `gofmt` and `golangci-lint run`;
  - a doc comment on every new function, method, type and constant;
  - a `Covers:` line on every new test;
  - only `internal/tui` imports Bubble Tea, Bubbles or Lipgloss (ARC-7);
  - no `cmd/` package imports another `cmd/` package except `cmd/root` (ARC-4);
  - no generated file cites a spec ID;
  - `go.mod` is unchanged.

  Covers: FR-001, FR-002, FR-003, FR-004, FR-005, FR-006, FR-007, FR-008, FR-009, FR-010, FR-011, FR-012, FR-013, FR-014, FR-015, FR-016, FR-017, FR-018, FR-019, FR-020, FR-021, FR-022. Done when: `gofmt -l .` prints nothing; `golangci-lint run`, `go test ./...` and `go test -tags integration ./...` pass; `go mod tidy` leaves no diff; a grep finds a `Covers:` line for every FR-001..FR-022 and EC-001..EC-031. After: T-036
- [ ] T-038 Check every Definition of Done item in `spec.md` and tick the ones verified. Covers: all. Done when: every DoD item in `spec.md` is ticked, and the evidence for each (test name or command output) is reported. After: T-037
