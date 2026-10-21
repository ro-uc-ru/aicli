# Design: `aicli add llm-provider`

Non-normative. The normative text is FR-002..FR-007, FR-010, FR-011, FR-013..FR-015, FR-018..FR-020, EC-003, EC-006..EC-010 and the golden files. Everything not shown here (selector, scrolling, truncation, buttons, error styling) is as in `designs/add-screen.md` of spec 004. Paths of generated Python files, version bounds and JSON values are illustrative.

## Colours (styled mode only)

Same ANSI-256 palette as spec 004; only the new elements are listed. In plain mode no colour is used: `> `, `[x]`, `[ ]` and `[✓]` carry all the meaning.

| Element | Style |
|---|---|
| Cursor `> ` of the checklist and the choice under it | 117, bold |
| `[x] ` marker and selected choice | 117 |
| `[ ] ` marker and unselected choice | 255 |
| Already added line `[✓] <choice> (already added)` (FR-007) | 245, whole line |
| `All providers are already added.` (EC-006) | 245 |
| Empty-selection error `select at least one provider` | same red as the `Error:` of ERR-5 |
| `Will update:` heading (FR-011) | 245, as `Will create:` |
| Word `create` in the report | green, as spec 004 FR-025 |
| Word `update` in the report | 117: a change to an existing file, not a new one |
| Word `skip` in the report | 245: nothing happened to that path |

## Form opened directly (FR-005, FR-006)

`aicli add llm-provider` in a terminal, with no arguments. There is a single field, so it always has the focus and there is no `tab`/`shift+tab`. The field title keeps the `> ` of spec 004 FR-011; the checklist cursor is a second `> ` inside the field, one column further in.

```
aicli › add › llm-provider
Add one or more LLM providers.

> provider
  LLM providers to add
  > [ ] openai
    [ ] claude
    [ ] gemini
    [ ] ollama

↑/↓ move · space toggle · enter continue · esc quit · ctrl+c quit
```

After `Space` on `openai`, `↓ ↓ ↓` and `Space` on `ollama`:

```
> provider
  LLM providers to add
    [x] openai
    [ ] claude
    [ ] gemini
  > [x] ollama
```

Opened from the selector of `aicli add`, the footer reads `… · esc back · ctrl+c quit` (EC-016).

`Enter` with nothing selected keeps the form and shows the error under the field (FR-006):

```
> provider
  LLM providers to add
  > [ ] openai
    [ ] claude
    [ ] gemini
    [ ] ollama
  select at least one provider
```

## Some providers already added (FR-007)

`openai` was added in an earlier run. Its line is dimmed and the cursor starts on the first choice that can still be added; `↑`/`↓` skip it.

```
> provider
  LLM providers to add
    [✓] openai (already added)
  > [ ] claude
    [ ] gemini
    [ ] ollama
```

## Every provider already added (EC-006)

No line can be selected, so there is no cursor; the message takes the place of the field error and `enter` disappears from the footer.

```
aicli › add › llm-provider
Add one or more LLM providers.

> provider
  LLM providers to add
    [✓] openai (already added)
    [✓] claude (already added)
    [✓] gemini (already added)
    [✓] ollama (already added)
  All providers are already added.

esc quit · ctrl+c quit
```

## Confirmation, first run (FR-004, FR-011)

The values are in choice order whatever order they were ticked in. Shared files that do not exist yet are under `Will create:`; the three skeleton files are under `Will update:`. On a short terminal both headings and their paths scroll together as one list (FR-011, spec 004 EC-010).

```
aicli › add › llm-provider
Add one or more LLM providers.

aicli add llm-provider openai ollama

Will create:
  app/ai/llm/__init__.py
  app/ai/llm/base.py
  app/services/llm.py
  app/schemas/llm.py
  app/api/routers/llm.py
  tests/test_llm_router.py
  app/ai/llm/openai.py
  tests/test_llm_openai.py
  app/ai/llm/ollama.py
  tests/test_llm_ollama.py

Will update:
  app/api/router.py
  pyproject.toml
  .env.example

  Add    [ Cancel ]

←/→/tab choose · enter confirm · y add · n cancel · esc back · ctrl+c quit
```

## Confirmation, later run (FR-011)

`openai` and `ollama` are already added; the user ticks `claude`. Shared files exist, so they are not listed; `app/api/router.py` already registers the router, so it is not listed either.

```
aicli add llm-provider claude

Will create:
  app/ai/llm/claude.py
  tests/test_llm_claude.py

Will update:
  pyproject.toml
  .env.example
```

## Confirmation that cannot proceed (EC-007, EC-010)

Same layout as the `Cannot add …` block of spec 004: the error replaces both lists and only `Cancel` remains.

Project generated before this spec (no marker in any file; the first `inject` entry in plan order is named, FR-010):

```
aicli add llm-provider openai

Cannot add llm-provider, app/api/router.py has no "# aicli:router-imports" marker

[ Cancel ]

enter cancel · n cancel · esc back · ctrl+c quit
```

Item file created by another process after the form opened:

```
aicli add llm-provider openai claude

Cannot add llm-provider, these providers are already added:
  openai (app/ai/llm/openai.py)

[ Cancel ]
```

## Report after `Add` (FR-011, FR-014)

First run, `openai ollama`. `app/api/router.py` is injected at two markers but reported once.

```
create  app/ai/llm/__init__.py
create  app/ai/llm/base.py
create  app/services/llm.py
create  app/schemas/llm.py
create  app/api/routers/llm.py
create  tests/test_llm_router.py
create  app/ai/llm/openai.py
create  tests/test_llm_openai.py
create  app/ai/llm/ollama.py
create  tests/test_llm_ollama.py
update  app/api/router.py
update  pyproject.toml
update  .env.example

✓ Added llm-provider

Next steps:
  Run uv sync to install the provider SDKs
  Set OPENAI_API_KEY, OPENAI_MODEL in .env
  Set OLLAMA_MODEL in .env
```
(plain: `Added llm-provider`, without `✓ `)

Later run, `claude`:

```
skip    app/ai/llm/__init__.py
skip    app/ai/llm/base.py
skip    app/services/llm.py
skip    app/schemas/llm.py
skip    app/api/routers/llm.py
skip    tests/test_llm_router.py
create  app/ai/llm/claude.py
create  tests/test_llm_claude.py
skip    app/api/router.py
update  pyproject.toml
update  .env.example

✓ Added llm-provider

Next steps:
  Run uv sync to install the provider SDKs
  Set ANTHROPIC_API_KEY, ANTHROPIC_MODEL in .env
```

`ollama` added when `openai` already is (EC-008): the `openai` SDK is already in `pyproject.toml`, so that line reads `skip    pyproject.toml`.

## Errors of a direct call (FR-002, FR-003, FR-010, EC-001..EC-004)

Styled as spec 004 (`✗ ` prefix, red bold `Error:`, dimmed continuation lines); shown here plain.

```
$ aicli add llm-provider            # static mode
Error: llm-provider requires provider
  Usage: aicli add llm-provider <provider>... [flags]

$ aicli add llm-provider openai claude openai
Error: provider "openai" is given more than once
  Usage: aicli add llm-provider <provider>... [flags]

$ aicli add llm-provider anthropic
Error: provider must be one of: openai, claude, gemini, ollama
  Usage: aicli add llm-provider <provider>... [flags]

$ aicli add llm-provider openai claude    # openai already added
Error: cannot add llm-provider, these providers are already added:
  openai (app/ai/llm/openai.py)

$ aicli add llm-provider openai           # skeleton without markers
Error: cannot add llm-provider, app/api/router.py has no "# aicli:router-imports" marker
```

## `aicli add` with the real tree (spec 004 FR-005, FR-008)

Static:

```
aicli › add
Add an AI/LLM feature to this project.

llm-provider  Add one or more LLM providers.

Run `aicli add <feature> --help` to see its inputs.
```

Interactive selector:

```
aicli › add
Add an AI/LLM feature to this project.

> llm-provider  Add one or more LLM providers.

↑/↓ move · enter select · q quit
```

## Injected files (FR-009, FR-013, FR-015)

After `aicli add llm-provider openai ollama`. Markers stay in place; injected lines go right after them with the marker's indentation. A later run inserts its lines right after the marker too, so they land above the lines of earlier runs.

`pyproject.toml`:

```toml
dependencies = [
    "fastapi>=0.115",
    "uvicorn>=0.32",
    "sqlalchemy>=2.0",
    "alembic>=1.13",
    "pydantic-settings>=2.6",
    # aicli:dependencies
    "openai>=…",
]
```

`.env.example`:

```
APP_NAME=app
DATABASE_URL=sqlite:///./app.db
LOG_LEVEL=INFO
# aicli:env
OPENAI_API_KEY=
OPENAI_MODEL=
OLLAMA_MODEL=
# OLLAMA_BASE_URL=http://localhost:11434/v1
```

`app/api/router.py` (one possible layout; the plan decides how the imports stay out of sorting):

```python
"""Aggregates the routers of the app."""

from fastapi import APIRouter

from app.api.routers import health

# isort: off
# aicli:router-imports
from app.api.routers import llm
# isort: on

router = APIRouter()
router.include_router(health.router)
# aicli:routers
router.include_router(llm.router)
```

## Generated API (FR-018..FR-020)

`GET /llm/health` with `openai` configured and `ollama` missing its model:

```json
{
  "status": "degraded",
  "providers": [
    {"name": "openai", "configured": true, "missing": []},
    {"name": "ollama", "configured": false, "missing": ["OLLAMA_MODEL"]}
  ]
}
```

`POST /llm/openai/completion`:

```json
{
  "messages": [{"role": "user", "content": "Summarise SDD in one line."}],
  "system": "Be brief.",
  "max_tokens": 256
}
```

`200`:

```json
{"provider": "openai", "model": "<OPENAI_MODEL>", "content": "Write the spec first, then the code."}
```

Errors, always the skeleton's `ErrorModel` (first match wins, in this order):

| Case | Status | Body |
|---|---|---|
| `{provider}` not added, e.g. `/llm/mistral/completion` | 404 | `{"status": 404, "message": "Not Found"}` |
| Invalid body | 422 | `{"status": 422, "message": "Unprocessable Entity"}` |
| Added, not configured | 503 | `{"status": 503, "message": "Service Unavailable"}` |
| Client lacks the capability | 501 | `{"status": 501, "message": "Not Implemented"}` |
| Provider call fails, or SDK not installed | 502 | `{"status": 502, "message": "Bad Gateway"}` |
