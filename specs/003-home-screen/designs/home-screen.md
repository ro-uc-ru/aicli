# Design: home screen

Non-normative. The normative text is FR-002, FR-003, FR-005..FR-013 and the golden file of the static output.

## Colours (FR-013, styled mode only)

ANSI-256 colours, reusing the ones of `internal/tui/success.go` so the home screen matches the rest of the CLI.

| Element | Style |
|---|---|
| Banner lettering | 117, bold |
| Banner description lines | 245 |
| Entries (normal) | 255 |
| Parent prefix of a subcommand entry (`db` in `db migrate`) | 245; under the pointer the whole entry takes the selected style |
| Entry under the pointer (`> ` prefix included) | foreground 117, bold, background 236 |
| Detail header (`aicli › new`, FR-019) | 117, bold |
| Detail headings (`Usage:`, `Flags:`, ...) | 117, bold |
| Flags in the detail view | 255 |
| Footer (key hints) | 245 |
| Scroll indicators (`↑ more`, `↓ more`, FR-020) | 245 |

## Interactive, list view (FR-005)

```
 █████╗ ██╗ ██████╗██╗     ██╗
██╔══██╗██║██╔════╝██║     ██║
███████║██║██║     ██║     ██║
██╔══██║██║██║     ██║     ██║
██║  ██║██║╚██████╗███████╗██║
╚═╝  ╚═╝╚═╝ ╚═════╝╚══════╝╚═╝
Scaffolding for FastAPI backends for AI/LLM apps.
Convention over configuration, driven by specs.

> new       Create a new aicli project
  generate  Generate code inside a project
  db        Database migrations

↑/↓ move · enter open · q quit
```
(the entry under the pointer, here `new`, drawn in the highlight colour and the others in the normal colour; the highlight follows the pointer when moved by keys or click (FR-013); banner in the accent colour; footer dimmed)

## Interactive, detail view (FR-008, FR-019)

```
aicli › new

Create a new aicli project

Usage:
  aicli new <name> [flags]

Flags:
  -h, --help   help for new

esc back · ↑/↓ scroll · q quit
```

## Static (FR-012, stdout is not a TTY)

```
 █████╗ ██╗ ██████╗██╗     ██╗
██╔══██╗██║██╔════╝██║     ██║
███████║██║██║     ██║     ██║
██╔══██║██║██║     ██║     ██║
██║  ██║██║╚██████╗███████╗██║
╚═╝  ╚═╝╚═╝ ╚═════╝╚══════╝╚═╝
Scaffolding for FastAPI backends for AI/LLM apps.
Convention over configuration, driven by specs.

new       Create a new aicli project
generate  Generate code inside a project
db        Database migrations
```
(the command names above are illustrative; the real list comes from the command tree)
