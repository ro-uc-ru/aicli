# Design: success output of `aicli new`

Non-normative. The normative text is FR-015 and the golden file of the plain output.

## Styled (stdout is a TTY, `NO_COLOR` unset), rendered with Lipgloss

```
$ aicli new myapp

✓ Created project myapp        (green, bold)

Next steps:                    (muted, bold)
  cd myapp                     (commands in light blue)
  cp .env.example .env   (optional)   ("(optional)" dimmed)
  uv sync
  uv run uvicorn app.main:app --reload
```

## Plain (not a TTY, or `NO_COLOR` set): no ANSI, no `✓ `

```
$ aicli new myapp | cat

Created project myapp

Next steps:
  cd myapp
  cp .env.example .env   (optional)
  uv sync
  uv run uvicorn app.main:app --reload
```

Colors are indicative; they are chosen at implementation time.
