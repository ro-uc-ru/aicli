# Plan: 005-add-llm-provider

- Status: approved

## Approach

The spec is met in two layers, in this order. First, the `add` mechanism of spec 004 is generalised: a multi-choice input, the direct form, the `shared` and `inject` actions, and item files, without naming any provider. Then `llm-provider` is declared as one more feature on top of that mechanism, with all its Python in embedded templates. The packages of spec 004 are reused; no new layer is added:

- `internal/feature`: the contract (`Input` gains multi-choice and item files), value validation and `Run`.
- `internal/gen`: the file mechanism moves from "a list of files to create" to "a plan of entries with an action" (`create`, `shared`, `inject`). It does the read-only check, the all-or-nothing apply and the per-path report.
- `internal/add`: decides when the direct form opens (FR-005).
- `internal/tui`: the checklist field, `(already added)`, the confirmation with two lists and the report with three verbs.
- `internal/llmprovider` (new): the feature plan, meaning which entries, in which order, which SDK and variables per provider, and the next steps. `cmd/addllmprovider` (new) only wraps it in a command.
- `templates/llmprovider/` (new): the generated Python files and the injected lines.

Key decisions:

- **The multi-choice value is "the remaining arguments"**. `ValidateValues` returns one flat list: the values of the ordinary inputs and, at the end, those of the multi-choice input. They are trimmed, free of duplicates and reordered to the declaration order of the choices (FR-004). The `Plan` function, the equivalent command and the form result (`AddResult.Values`) stay `[]string`. So `feature.Run` after `Add` and the direct call still share the same path as in spec 004.
- **The plan is a list of entries with an action**. `gen.FeatureFile` gains `Action` and `Marker`. `create` is the zero value of `Action`, so the features of spec 004 do not change. `Marker` applies only to `inject`, whose template renders the lines of the block. The feature also declares its items: the plural noun and, per value, the path of its item file. With that, `gen` applies FR-003 without knowing about providers.
- **One check function, shared by the confirmation and the run**. `gen.CheckPlan` is read-only. It applies, in the order of FR-011:
  1. Path validation (spec 004 FR-022), which now also resolves every `inject` target and every existing `shared` path.
  2. The `inject` targets (FR-010).
  3. Already added items (FR-003).
  4. Existing `create` paths, plus `shared` paths that are not regular files (spec 004 FR-023, EC-028).

  It returns a preview with the `Will create:` and `Will update:` paths. Injections are computed in memory to know whether they change anything. `gen.ApplyPlan` calls it again right before writing (EC-007).
- **Injection is computed entirely in memory before writing**:
  - For each target file, its original bytes are read and its entries are applied in plan order to the in-memory content.
  - Presence (FR-009) is looked up as consecutive trimmed lines in content that already holds the earlier blocks of the same run.
  - Blocks for the same marker go one after another right below the marker.
  - Indentation is the marker's plus each line's relative indentation; empty lines stay empty.
  - The line ending is the marker's. If the marker is the last line with no line ending, the first line ending of the file is used, or `\n` (EC-030).
  - The write happens in place on the resolved path (`O_WRONLY|O_TRUNC`, no creation), so the symbolic link, the inode and the permissions are kept (EC-029).
  - The original bytes are kept in the rollback list, which writes them back the same way (FR-011, EC-013).
- **Apply order is plan order**. `shared` and `create` entries are created with `O_EXCL` as in spec 004, and an existing `shared` path is skipped. Then each `inject` file is written once. The context is checked before each step. On a failure or a signal, the rollback undoes in reverse order: it restores the injected files and removes the created files and directories (EC-013, spec 004 EC-023).
- **The direct form is decided in `add.Prepare`**, which already walks the features before the tree runs (spec 004 EC-002). For each feature with inputs, it wraps its `RunE`. If there are no positional arguments, no changed flag and the mode is interactive, the wrapper opens the add screens with that feature alone, starting on the form and with `Esc` as quit. Otherwise it calls the original `RunE`, which is `feature.Run`. This way `internal/feature` does not import `internal/add` (no cycle) and the screen code is not duplicated.
- **The provider registry of the generated project discovers providers at runtime** (FR-016, EC-020, EC-027):
  - `app/ai/llm/__init__.py` walks the package modules with `pkgutil` and imports each one.
  - It keeps only the modules that expose a provider descriptor of the type defined in `base.py`: name, position, required variables, settings loader and client factory.
  - No provider module imports its SDK at load time; it imports it inside the method that builds or uses the client.
  - If two modules declare the same name, the one named `<name>.py` wins; if neither is named that, both are ignored. Providers are sorted by `(position, name)` (EC-031).
- **HTTP error precedence without fighting FastAPI** (FR-020, EC-022). FastAPI validates the body before it calls the handler, so `POST /llm/{provider}/completion` takes `provider` as free text and the body as raw JSON. The route then:
  1. Asks the registry (`404`).
  2. Validates the body with the model of `app/schemas/llm.py` (`422`), raising the same `RequestValidationError` that the skeleton handles.
  3. Calls the use case and translates its service exceptions (`503`, `501`, `502`) into `HTTPException`.

  The body schema is published in OpenAPI through `openapi_extra`. The skeleton's `message_for` already returns `Service Unavailable`, `Not Implemented` and `Bad Gateway` (through `HTTPStatus`), so `app/errors.py` does not change.
- **The generated tests do not depend on the developer's environment** (FR-021, EC-024):
  - Each LLM test file has an `autouse` fixture. It deletes the variables prefixed `OPENAI_`, `ANTHROPIC_`, `GEMINI_`, `GOOGLE_` and `OLLAMA_`, and `chdir`s to `tmp_path` so that pydantic-settings does not read the relative `.env`.
  - SDKs are replaced in `sys.modules` by a fake module (to check what is passed to it, EC-026) or by `None` (to simulate that it is not installed, EC-018).
  - So the same tests pass with and without `uv sync`, with no skips.
- **The `app/api/router.py` markers stay out of import sorting**, with `# isort: off` / `# isort: on` around `# aicli:router-imports`, as in the design. So `ruff check --fix` does not move the injected lines and FR-009 finds them again (FR-015, EC-021).
- **SDK bounds (STK-4, approved by the spec)**: `openai>=3.36`, `anthropic>=2.0`, `google-genai>=2.32`. These are the latest releases on PyPI, checked on 2026-10-21. The templates are written and integration-tested against them, and older majors have different client internals (e.g. `openai` 3.x moved to `httpx2`). The APIs the templates use exist in those versions: `AsyncOpenAI(...).chat.completions.create`, `AsyncAnthropic(...).messages.create`, and `genai.Client(...).aio.models.generate_content`.
- **Credentials and endpoints passed explicitly (FR-017)**. Checked against the source of the versions above:
  - Base URLs: `https://api.openai.com/v1`, `https://api.anthropic.com`, and `https://generativelanguage.googleapis.com/` (through `http_options.base_url`, which also skips `GOOGLE_GEMINI_BASE_URL`). `ollama` uses `OLLAMA_BASE_URL` (default `http://localhost:11434/v1`) and the fixed key `ollama`.
  - `anthropic`: an explicit `api_key` stops the SDK from reading any credential from the environment, including `ANTHROPIC_AUTH_TOKEN`.
  - `google-genai`: `api_key` is explicit, so `GOOGLE_API_KEY`/`GEMINI_API_KEY` are not read, and `vertexai=False` is passed so `GOOGLE_GENAI_USE_VERTEXAI`/`GOOGLE_GENAI_USE_ENTERPRISE` cannot switch it to Vertex AI.
  - `openai` (used by `openai` and `ollama`): `OPENAI_CUSTOM_HEADERS` can carry an `Authorization` header, which would win over `api_key`. So both clients also pass `Authorization: Bearer <their key>` in `default_headers`, which makes the SDK drop the `Authorization` from the environment. `OPENAI_ADMIN_KEY` is read but only sent on admin endpoints, never on chat completions.

Rejected alternatives:

- **One value type per input (`[][]string`) instead of the flat list**: it would change `Plan`, `EquivalentCommand`, `AddResult` and every test feature of spec 004 for no gain, since only the last input may be multi-choice.
- **Temporary file and `rename` for injection**: atomic, but it replaces a symbolic link with a file, changes the inode and requires copying permissions. FR-009 and EC-029 ask for in-place modification.
- **Deciding the direct form in `feature.NewCommand` through a global hook in `internal/feature`**: that would be mutable global state across packages, while `add.Prepare` is already the place where feature commands are adjusted before the run.
- **Filling the registry by injection into `__init__.py`**: ruled out by the spec itself (EC-020: `__init__.py` is `shared` and may be created again with no providers in it).
- **One parametrised client template for all four providers**: the three SDKs have different APIs. One template per provider reads better and each gets its own golden. `openai` and `ollama` share an SDK but not their settings or key, so they are separate too.
- **Validating the body with FastAPI's typed parameter**: it would answer `422` before `404`, against the order of FR-020.

## Components / Phases

**Phase 1: `internal/feature` (contract and values, FR-001, FR-002, FR-004)**
- `Input` gains `Multi bool`, `ItemNoun string` (plural) and `ItemFiles map[string]string` (choice → item file path).
- `Declaration.NextSteps` becomes `func(values []string) []string`, because FR-014 depends on the values. The spec 004 test feature is adapted with a function that returns its fixed steps.
- `CheckContract` adds these rules, each with its own error:
  - a multi-choice input that is not the last input;
  - no choices, or empty or repeated choices;
  - a multi-choice input with a validator;
  - an item file of a choice that is not a `create` entry of the plan for every subset that includes that choice;
  - a `shared` entry whose rendered content differs between non-empty subsets. Each `shared` entry is rendered for all 2ⁿ−1 subsets and the bytes are compared.
- `NewCommand`: for a multi-choice input, `Use` shows `<input>...`.
- `ValidateValues`:
  - For the multi-choice input it takes all remaining arguments. "Too few" now means no value at all, and there is no "too many".
  - Each value is checked in the given order: empty, control characters, membership in the choices (spec 004 EC-009), and repetition after trimming (`<input> "<value>" is given more than once`, naming the first repeated value).
  - It returns the values in choice order.
- `Run`: gets the root, builds the `gen.Plan` (entries and items), calls `gen.ApplyPlan` and writes `tui.RenderAdded` with the report lines and `NextSteps(values)`.

**Phase 2: `internal/gen` (file actions, FR-003, FR-008..FR-011)**
- `FeatureFile` gains `Action` (`ActionCreate`, `ActionShared`, `ActionInject`) and `Marker`. New `Plan{Entries, ItemNoun, Items []Item{Value, Path}}`.
- Path validation: that of spec 004, with duplicates allowed only between `inject` entries for the same target. An `inject` target that is a link is resolved and must stay inside the root (EC-014).
- `CheckPlan(root, feature, plan) (Preview, error)`:
  - Read-only, with the check order of FR-011 and the fixed messages of FR-010 and FR-003. The FR-003 error lists `<value> (<item file>)` as continuation lines, in choice order.
  - The list of spec 004 FR-023 includes the `shared` paths that are not regular files, in plan order (EC-028).
  - `Preview` holds `Create` (the `create` entries plus the `shared` entries that do not exist) and `Update` (the targets that would change).
- Pure injection engine: it takes the original bytes and the blocks and returns the new bytes and whether they changed. It implements:
  - presence as consecutive trimmed lines;
  - insertion after the marker, with its indentation and line endings, and EC-030;
  - the unique-marker lookup (a line that, trimmed, is exactly `# aicli:<name>`).
- `ApplyPlan(ctx, fsys, root, feature, plan) ([]ReportLine, error)`:
  - It runs `CheckPlan` again, renders everything in memory (templates with `missingkey=error`), creates, injects, and rolls back if anything fails.
  - It returns one line per path in plan order (`create`, `skip`, `update`). A target injected by several entries appears once, at its first entry.
- `FeatureFS` gains `ReadFile` and `Stat`; the in-place write uses the existing `OpenFile` without `O_EXCL`.
- Rollback: injected files are restored byte for byte at the resolved path. Paths that could not be restored or removed are listed in the rollback error of spec 004.
- `CheckFeatureFiles` and `CreateFeatureFiles` become wrappers around `CheckPlan`/`ApplyPlan`, or are removed if nothing calls them any more. The spec 004 goldens do not change.

**Phase 3: `internal/add` (direct form and items, FR-005, FR-007)**
- `Features`: each `tui.Feature` also carries:
  - `Multi` and `ItemNoun`;
  - the `Added()` function, a read-only stat of the item files, called when the form opens;
  - `Check`, which now returns the `Preview` of `gen.CheckPlan`.
- `Prepare`: on top of what it already does, it wraps the `RunE` of each feature with inputs, as described in the Approach. The direct branch calls `runScreens` with the feature alone in "direct" mode and handles the result as `runInteractive` does: `quit` writes nothing, `cancel` writes `Nothing added.`, and `add` calls `feature.Run`.
- Static mode and calls with arguments or flags do not change (spec 004 FR-006).

**Phase 4: `internal/tui` (screens and report, FR-005..FR-007, FR-011)**
- `FeatureInput` gains `Multi`, `ItemNoun` and `Added []bool`, filled from `Feature.Added()` when the form opens.
- New `multiChoiceField`:
  - With the focus it shows `[x] `/`[ ] `/`[✓] … (already added)` and the `> ` cursor. The cursor starts on the first choice that can be added, or is not shown if there is none.
  - `↑`/`↓`/`k`/`j` skip already added lines and stop at the ends.
  - `Space` toggles the selection, including deselecting (EC-015). A click moves the cursor and toggles.
  - Without the focus it shows the selection joined with `, `.
  - It scrolls internally as `choiceField` does. Its value is the selected choices in declaration order.
- `formModel`:
  - The multi-choice input is validated only on `Enter` (`select at least one <input>`).
  - If every choice is already added, it shows `All <items> are already added.`, `Enter` does nothing and the footer does not list it (EC-006).
  - The footer includes `space toggle`.
  - In direct mode, `Esc` returns `quit` and the footer reads `esc quit`. From the selector, `Esc` goes back and the footer reads `esc back` (EC-016).
- `AddModel`/`RunAdd` accept a start screen (the selector, or the form of one feature) and pass on the direct mode. `Esc` in the confirmation still returns to the form.
- Confirmation:
  - The path list becomes a list of rows: `Will create:`, its paths, a blank row, `Will update:`, its paths. Each heading appears only when it has paths.
  - `computeConfirmLayout` receives the total row count, so the scrolling and indicators of spec 004 EC-010 apply to the combined list.
  - A `Check` error replaces both lists, as it already does.
- `RenderAdded(feature, lines []ReportLine, steps, styled)`: the verb is padded to 6 characters plus two spaces. When styled, `create` is green, `update` is 117 and `skip` is 245, as in the design table. The spec 004 goldens do not change because they only have `create`.
- New styles from the design: cursor 117 bold, selected 117, unselected 255, already added 245, `Will update:` 245. All are no-ops when not styled.

**Phase 5: the `llm-provider` feature (FR-012..FR-014)**
- `internal/llmprovider`:
  - `Declaration()` has the name, the short description, the `provider` input (multi-choice `openai, claude, gemini, ollama`, noun `providers`, item `app/ai/llm/<provider>.py`), `Plan` and `NextSteps`.
  - Internal per-provider tables: the SDK (`openai`, `anthropic`, `google-genai`, and `openai` for `ollama`), the required variables, and the optional commented line of `ollama`.
  - `Plan` emits exactly the order of FR-013: six `shared` entries, two `create` entries per provider, `router.py` at its two markers, one entry per distinct SDK in order of first appearance, and one entry per `.env.example` line.
- `cmd/addllmprovider`: only `NewCommand()`, which returns `feature.NewCommand(llmprovider.Declaration())` (ARC-3). `cmd/root` registers it under `add` (ARC-4).
- `templates/llmprovider/`, embedded as `templates.LLMProvider`:
  - `shared/`. All these files refer to each other by module (`schemas.<Model>`):
    - `app/ai/llm/__init__.py`: the registry.
    - `app/ai/llm/base.py`: `Message`, the `CompletionClient` protocol, the provider descriptor, provider errors and `# aicli:capabilities`.
    - `app/services/llm.py`: `health`, `complete`, service exceptions and `# aicli:services`.
    - `app/schemas/llm.py`: request, response and health models, and `# aicli:schemas`.
    - `app/api/routers/llm.py`: `GET /llm/health`, `POST /llm/{provider}/completion` and `# aicli:routes`.
    - `tests/test_llm_router.py`: a fake client that implements `CompletionClient` and one that does not. It tests every status code and their precedence, `/llm/health`, an ignored foreign module, a user module `mistral`, duplicate names and position ties, and a re-created `__init__.py`.
  - `provider/<provider>.py`, one per provider: prefixed settings with no model default; empty or whitespace-only counts as missing, and an optional variable then takes its default; an async client that imports its SDK lazily, with `# aicli:methods` inside the class.
  - `provider/test_llm_<provider>.py`, one per provider: tests the settings, the registration, the kwargs passed to the fake SDK (EC-026), `502` with the SDK missing (EC-018), and environment isolation.
  - `inject/`: `router_import`, `router_include`, `dependency` (data: the requirement) and `env` (data: the line).
- No generated file cites a spec ID.

**Phase 6: skeleton markers (FR-015)**
- `templates/skeleton/pyproject.toml.tmpl`: `# aicli:dependencies` as the last element of the `dependencies` array, with the array's indentation.
- `templates/skeleton/.env.example`: `# aicli:env` as the last line.
- `templates/skeleton/app/api/router.py`: the `# isort: off` / `# aicli:router-imports` / `# isort: on` block after the imports, and `# aicli:routers` at the end, as in the design.
- The goldens of the skeleton (spec 001), the home screen (spec 003) and the static `aicli add` with the real tree (spec 004) are regenerated with `-update`, and the diff is reviewed.

**Phase 7: tests (each with its `Covers:` line)**
- `internal/feature`:
  - The new contract rules, on test features that break each one and on the real tree.
  - Goldens of the FR-002 errors: missing value, repeated value, value outside the choices, and the usage line with `<provider>...`.
  - Value order and equivalent command (FR-004).
- `internal/featuretest`: a test feature with a multi-choice input, items, `shared` and `inject` entries on an in-memory FS, plus the variants that break each rule.
- `internal/gen`:
  - Injection engine table: indented marker, `\r\n`, entries already present, several entries at one marker, a block with blank and repeated lines, a marker on the last line with no line ending, and a duplicate block in the same run.
  - FR-010 errors (does not exist, broken link, directory, no marker, repeated marker) and FR-003 errors, with the fallback to spec 004 FR-023.
  - A `shared` path that is a directory or a broken link (EC-028).
  - A target reached through a link inside the root (link and permissions kept, also after rollback) and one outside it.
  - Byte-for-byte restore after a forced write failure and after a cancelled context.
  - Goldens of the plain and styled report: first run with two providers, a later run with a third, and an injection that changes nothing.
- `internal/add`: the direct branch versus the normal branch of `Prepare`. The normal branch covers static mode, arguments, flags, and a feature with no inputs in interactive mode, which runs with no form.
- `internal/tui` (teatest):
  - the direct form;
  - the checklist with keys, `Space`, clicks and markers with styling off;
  - `(already added)`, the empty-selection error, and every provider already added;
  - `Esc` quitting versus `Esc` returning to the selector;
  - the confirmation with `Will create:` and `Will update:` and scrolling of the combined list;
  - the styled colours.
- `internal/llmprovider` and goldens:
  - The generated files per provider, and the injected `pyproject.toml`, `.env.example` and `app/api/router.py`.
  - A test that every FR-022 file has its marker exactly once.
  - A test that the plan follows the FR-013 order, with one SDK only once for `openai ollama` (EC-008).
- `cmd/root`: end-to-end errors (styled and plain, stdout empty) and the direct run with its report.
- Integration (`integration` tag, in `cmd/root`). Its doc comment carries the `Covers:` line of the generated-app requirements. Steps:
  1. Scaffold a project and run `uv sync`.
  2. Run `add llm-provider openai`.
  3. Run pytest, `ruff check` and `ruff format --check` with `uv run --no-sync`.
  4. Run `uv sync`, then pytest and ruff again.
  5. Run `ruff check --fix` and `ruff format`.
  6. Run `add llm-provider claude gemini ollama` and check that no line is injected again.
  7. Repeat steps 3 and 4 for both states.

**Phase 8: quality and close**
- `gofmt`, `golangci-lint run` and doc comments on everything new.
- `go test ./...` and `go test -tags integration ./...` pass.
- Tick the DoD items of `spec.md` that the final task verifies.

## Coverage

| FR / EC | Covered by |
|---|---|
| FR-001 | Phase 1 (multi-choice `Input`, `CheckContract` rules, `shared` rendered per subset) + Phase 7 contract |
| FR-002 | Phase 1 (`ValidateValues`, `Use` with `<input>...`) + Phase 7 goldens |
| FR-003 | Phase 2 (`CheckPlan`, items, check order) + Phase 7 `internal/gen` |
| FR-004 | Approach (sorted flat list) + Phase 1 (`ValidateValues`) + Phase 4 (checklist value) + Phase 7 |
| FR-005 | Approach + Phase 3 (`Prepare` wraps `RunE`) + Phase 4 (start screen, `Esc` quits) + Phase 7 |
| FR-006 | Phase 4 (`multiChoiceField`, validation only on `Enter`) + Phase 7 teatest |
| FR-007 | Phase 3 (read-only `Added()`) + Phase 4 (`[✓]` lines, skipped by the cursor) + Phase 7 teatest |
| FR-008 | Phase 2 (`Action`, `shared`, path validation, duplicates only for `inject`) + Phase 5 (lines from templates) |
| FR-009 | Approach (in-memory injection, in-place write) + Phase 2 (injection engine) + Phase 7 table |
| FR-010 | Phase 2 (`CheckPlan`, target and marker checks) + Phase 7 goldens |
| FR-011 | Phase 2 (`Preview`, `ReportLine`, injection rollback) + Phase 4 (two lists, `RenderAdded`) + Phase 7 |
| FR-012 | Phase 5 (`Declaration`, `cmd/addllmprovider`, registration in `cmd/root`) |
| FR-013 | Phase 5 (`Plan` and entry order) + Phase 7 order test and goldens |
| FR-014 | Phase 1 (`NextSteps` as a function) + Phase 5 + Phase 7 report goldens |
| FR-015 | Phase 6 (markers, `isort: off/on`) + Phase 7 integration (`ruff format --check`, `--fix` between runs) |
| FR-016 | Approach (registry by discovery, lazy SDK import) + Phase 5 (`__init__.py`, `base.py`) |
| FR-017 | Approach (bounds, explicit base URL, key and `Authorization` header, `vertexai=False`) + Phase 5 (provider templates) |
| FR-018 | Phase 5 (`services/llm.health`, `routers/llm.py`) + generated tests |
| FR-019 | Approach (the route only translates HTTP) + Phase 5 (`complete` use case, schemas) + generated tests |
| FR-020 | Approach (precedence 404→422→503→501→502, raw body) + Phase 5 + generated tests |
| FR-021 | Approach (isolation fixture, fake or missing SDK) + Phase 5 (generated tests) + Phase 7 integration |
| FR-022 | Phase 5 (markers in each template) + Phase 7 (unique-marker test, integration) |
| EC-001 | Phase 1 ("too few") + Phase 3 (no direct branch in static mode) + Phase 7 golden |
| EC-002 | Phase 1 (repetition after trimming) + Phase 7 golden |
| EC-003 | Phase 1 (exact match with the choices) + Phase 7 golden |
| EC-004 | Phase 2 (FR-003 wins over spec 004 FR-023) + Phase 7 |
| EC-005 | Phases 2 and 3 (`Lstat` of the item file) + Phase 7 |
| EC-006 | Phase 4 (no cursor, message, no `Enter`) + Phase 7 teatest |
| EC-007 | Approach (`ApplyPlan` checks again) + Phase 7 |
| EC-008 | Phase 5 (one SDK once in the plan) + Phase 2 (presence) + Phase 7 report goldens |
| EC-009 | Phase 2 (`shared` created or skipped) + Phase 7 |
| EC-010 | Phase 2 (FR-010) + Phase 4 (error in the confirmation) + Phase 7 |
| EC-011 | Phase 2 (presence by trimmed line) + Phase 7 table |
| EC-012 | Phase 2 (marker indentation and line ending) + Phase 7 table + integration (valid TOML) |
| EC-013 | Approach (reverse-order rollback, signal) + Phase 2 + Phase 7 forced failure and cancelled context |
| EC-014 | Phase 2 (resolved target inside the root) + Phase 7 |
| EC-015 | Phase 4 (`Space` deselects, internal scrolling) + Phase 7 teatest |
| EC-016 | Phase 4 (`Esc` by origin) + Phase 3 + Phase 7 teatest |
| EC-017 | Phase 5 (whitespace only counts as missing) + generated tests |
| EC-018 | Approach (lazy SDK, `502`, missing SDK in tests) + generated tests |
| EC-019 | Phase 5 (schema validation) + generated tests |
| EC-020 | Approach (discovery) + generated tests (re-created `__init__.py`) |
| EC-021 | Approach (`isort: off/on`) + Phase 7 integration |
| EC-022 | Approach (precedence) + generated tests |
| EC-023 | Phase 5 (empty optional takes its default) + generated tests |
| EC-024 | Approach (isolation fixture) + generated tests + integration |
| EC-025 | Phase 2 (whole block or nothing) + Phase 7 table |
| EC-026 | Approach (explicit kwargs, fixed `ollama` key) + per-provider generated tests |
| EC-027 | Approach (descriptor required) + generated tests (`utils.py`, `mistral`) |
| EC-028 | Phase 2 (non-regular `shared` in the spec 004 FR-023 list) + Phase 7 |
| EC-029 | Approach (in-place write at the resolved path) + Phase 2 + Phase 7 |
| EC-030 | Phase 2 (line ending after a final marker) + Phase 7 table |
| EC-031 | Approach (duplicate and order rule) + generated tests |

## Constitution check

- ARC-1: spec 005 is approved, and the plan implements only what it says.
- ARC-2: one new flat folder, `cmd/addllmprovider`.
- ARC-3: `cmd/addllmprovider` only builds the command; the logic lives in `internal/llmprovider`, `internal/feature`, `internal/gen` and `internal/add`.
- ARC-4: `cmd/addllmprovider` does not import `cmd/add`; `cmd/root` registers it under `add`.
- ARC-5: generated files and injected lines come from templates embedded in `templates/llmprovider/`, never from strings built in Go.
- ARC-6: the generated layout is the fixed one of the spec; there are no flags or configuration.
- ARC-7: Bubble Tea and Lipgloss stay in `internal/tui`; `internal/add` passes it plain structs and functions.
- ARC-8: paths are built with `path/filepath` (and `path` inside `embed.FS`) and files are handled with `os`. No process is spawned except `uv` in the integration test.
- STK-4: no new Go module. The generated-project packages (`openai`, `anthropic`, `google-genai`) are the ones the spec approves, with the bounds fixed in the Approach.
- STK-5: providers arrive only with `aicli add llm-provider`; the skeleton gains only marker comments.
- SEC-1 / ERR-4: keys are read from the environment with pydantic-settings and passed explicitly to the SDK. They are never logged (only the exception type name) and never returned by `/llm/health`.
- SEC-2: `.env.example` only receives names with empty values.
- SEC-3: every path, including `inject` targets and their links, is validated inside the root before anything is written.
- SEC-5: lines are only inserted after a marker, with a report (`update`) and a byte-for-byte rollback. `create` uses `O_EXCL`, and an existing `shared` file is not touched.
- SEC-6: the CLI does not import or run Python; the registry runs in the generated app, not in aicli.
- ERR-1 / ERR-2 / ERR-5: errors carry context, and plain `Error:` messages with continuation lines go through the existing `root.run` path.
- ERR-3: the direct call with arguments and static mode need no TTY.
- TST-1 / TST-2 / TST-4: unit, teatest, golden and end-to-end tests; `go test ./...` and `golangci-lint run` pass.
- TST-3 / TST-5: the integration test runs the generated project's pytest and ruff, with and without the SDKs.
- TST-6: the generated tests use fake clients and SDKs; only `uv sync` uses the network, in integration.
- SCP-2 / SCP-5: provider selection is in scope, and the generated project does not depend on aicli.
- LIM-1: the symbolic link and permission tests run on macOS and Linux.
- LIM-2: `requires-python` does not change.

## Known limitations

- `AsyncOpenAI` reads `OPENAI_ORG_ID` and `OPENAI_PROJECT_ID` from the environment when they are not passed, and the `ollama` client would send them as headers to `OLLAMA_BASE_URL`. They are identifiers, not credentials or endpoints, so FR-017 does not cover them and the plan leaves them alone. Blocking them would need a spec change.
