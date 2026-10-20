# Design: `aicli add`

Non-normative. The normative text is FR-005, FR-006, FR-007, FR-008, FR-011, FR-014, FR-015, FR-016, FR-018, FR-023, FR-025, FR-026, EC-003 and the golden files. The feature names (`feature-1`, ...), their descriptions and the inputs `input-1`, `input-2` are dummy placeholders: the real list comes from the command tree.

## Colours (styled mode only)

ANSI-256 colours, the same ones as the home screen (`designs/home-screen.md` of spec 003, reusing `internal/tui/success.go`), so `add` matches the rest of the CLI. In plain mode no colour is used and the pointer `> ` still identifies the selected entry.

| Element | Style |
|---|---|
| Selector header (`aicli › add`), form and confirmation header (`aicli › add › <feature>`) | 117, bold |
| Selector description line | 245 |
| Entries (normal) | 255 |
| Entry under the pointer (`> ` prefix included) | foreground 117, bold, background 236 |
| Scroll indicators (`↑ more`, `↓ more`) | 245 |
| Footer (key hints), in the selector, the form and the confirmation | 245 |
| Feature description (FR-012) | 245 |
| Form field title (input name), focused | 117, bold |
| Form field title, not focused | 255 |
| Form field description (input description) | 245 |
| Form input text | 255 |
| Form validation error (FR-013) | same red as the `Error:` of ERR-5 |
| Equivalent command (FR-018) | 255 |
| `Will create:` heading | 245 |
| Paths of the file list | 255 |
| `Cannot add …` block in the confirmation | same red as the `Error:` of ERR-5 |
| `Add` button | green (same as the `✓` of spec 001 FR-015); bold when focused |
| `Cancel` button | red (same as the `Error:` of ERR-5); bold when focused |
| Choice marker `(•) ` and selected choice | 117 |
| Other choices | 255 |
| Success line `✓ Added <feature>` (FR-025) | as `✓ Created project` of spec 001 (green, bold) |
| `Next steps:` and steps (FR-025) | as spec 001 FR-015 |
| Word `create` in the report (FR-025) | same green as the `✓` of spec 001 FR-015; the path plain |
| Error message (`Error:`, FR-026) | as spec 001 FR-023: `✗ ` prefix, red bold `Error:`, continuation lines dimmed (245) |

## Selector (interactive, FR-008)

```
aicli › add
Add an AI/LLM feature to this project.

> feature-1  Description of feature 1
  feature-2  Description of feature 2
  feature-3  Description of feature 3

↑/↓ move · enter select · q quit
```
(the entry under the pointer, here `feature-1`, drawn in the highlight colour and the others in the normal colour; the highlight follows the pointer when moved by keys or click; header in the accent colour; footer dimmed)

## Form after choosing `feature-1` (FR-011, FR-013, FR-015)

Still in the alternate screen. The short description of the feature (FR-012) is under the header, on one line; all fields are visible, focus on the first. The mouse is captured: a click on a field moves the focus to it (FR-015).

Here `input-1` is a choice input (FR-016) with the choices `openai`, `anthropic`, `ollama`, and `input-2` a text input with a validator (FR-014). The focused field has `> ` before its title (FR-011).

```
aicli › add › feature-1
Description of feature 1

> input-1
  Description of input 1
  (•) openai
  ( ) anthropic
  ( ) ollama

  input-2
  Description of input 2
  │

tab/enter next · ↑/↓ choose · shift+tab back · esc back · ctrl+c quit
```

On a text field the footer drops `↑/↓ choose`; on the last field it reads `enter continue · shift+tab back · esc back · ctrl+c quit`. A choice field without focus shows only its selected choice. A long value scrolls horizontally inside the input line (EC-012). An empty value, or one the validator rejects, shows its error under the field:

```
  input-1
  Description of input 1
  openai

> input-2
  Description of input 2
  │
  input-2 cannot be empty
```

After `Esc` back to the selector, opening `feature-1` again shows the values typed before (FR-017).

## Confirmation (FR-018)

Focus starts on `Cancel`; `y` adds directly. The mouse is released so the command can be selected and copied (FR-018).

```
aicli › add › feature-1
Description of feature 1

aicli add feature-1 openai 'gpt 5'

Will create:
  path/to/file-a
  path/to/file-b

  Add    [ Cancel ]

←/→/tab choose · enter confirm · y add · n cancel · esc back · ctrl+c quit
```
(`Add` green and `Cancel` red when styled, the focused one bold and in brackets; with styling off, the brackets alone show the focus)

If files of the plan already exist (FR-018), the confirmation shows the problem before anything runs and offers only `Cancel`:

```
aicli › add › feature-1
Description of feature 1

aicli add feature-1 openai 'gpt 5'

Cannot add feature-1, these files already exist:
  path/to/file-a

[ Cancel ]

enter cancel · n cancel · esc back · ctrl+c quit
```

After `Add`, the alternate screen is left and the feature runs as if the user had typed the command shown, and for example prints (FR-025):

```
create  path/to/file-a
create  path/to/file-b

✓ Added feature-1

Next steps:
  Set OPENAI_API_KEY in .env
```
(plain: `Added feature-1`, without `✓ `; the `Next steps:` block appears only if the feature declares steps)

After `Cancel` (or `n`), the alternate screen is left and one line is printed (FR-018):

```
Nothing added.
```

## Feature already added (FR-023)

```
✗ Error: cannot add feature-1, these files already exist:
  path/to/file-a
  path/to/file-b
```
(plain form without `✗ `)

## Direct call with a missing input (FR-006)

```
Error: feature-1 requires input-2
  Usage: aicli add feature-1 <input-1> <input-2> [flags]
```

## Static (FR-005, stdout is not a TTY)

```
aicli › add
Add an AI/LLM feature to this project.

feature-1  Description of feature 1
feature-2  Description of feature 2
feature-3  Description of feature 3

Run `aicli add <feature> --help` to see its inputs.
```

## Real tree today (EC-003)

Printed as plain text in both modes; the alternate screen is never entered.

```
aicli › add
Add an AI/LLM feature to this project.

No features available.
```

## Unknown feature (FR-007)

```
Error: unknown feature "featur-1"
  Did you mean "feature-1"?
  Run `aicli add` to see the available features.
```
