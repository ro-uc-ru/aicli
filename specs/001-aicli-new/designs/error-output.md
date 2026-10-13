# Design: error output of `aicli new`

Non-normative. The normative text is FR-023 and the golden file of the plain output.

## Styled (stderr is a TTY, `NO_COLOR` unset, `TERM` not `dumb`), rendered with Lipgloss

```
$ aicli new Mi-App

✗ Error: invalid name "Mi-App": must start with a lowercase letter and contain only lowercase letters, digits, "-" and "_"
(✗ and "Error:" red, bold; the reason in the default color)

$ aicli new myapp          # ./myapp already exists

✗ Error: destination "myapp" already exists

$ aicli new

✗ Error: missing project name
  Usage: aicli new <name>   (continuation line dimmed)
```

## Plain (stderr is not a TTY, or `NO_COLOR` set, or `TERM=dumb`): no ANSI, no `✗ `

```
$ aicli new Mi-App 2>&1 | cat

Error: invalid name "Mi-App": must start with a lowercase letter and contain only lowercase letters, digits, "-" and "_"
```

Styling depends on stderr being a TTY, not stdout: `aicli new x > out.txt` still shows the error in color on the terminal. Stdout is always empty on failure.

Colors are indicative; they are chosen at implementation time.
