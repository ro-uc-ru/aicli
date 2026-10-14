# Design: error output of the project root check

Non-normative. The normative text is FR-003, FR-006 and ERR-5, and the golden file of the plain output.

The styled form follows the convention of spec 001 (`✗ ` prefix, `Error:` red and bold, continuation lines dimmed); the plain form is the same text without `✗ ` and without ANSI.

## Not a project (FR-003)

### Styled (stderr is a TTY, `NO_COLOR` unset, `TERM` not `dumb`), rendered with Lipgloss

```
$ cd ~/work && aicli model Users        # no project here (EC-004)

✗ Error: not inside an aicli project
  Missing: pyproject.toml, alembic.ini, app/main.py, migrations/env.py
  Run this command from the root of a project created with `aicli new`.
(✗ and "Error:" red, bold; the two continuation lines dimmed)

$ cd ~/work/mi-api/app/services && aicli model Users        # EC-001

✗ Error: not inside an aicli project
  Missing: pyproject.toml, alembic.ini, app/main.py, migrations/env.py
  Run this command from the root of a project created with `aicli new`.

$ cd ~/work/mi-api && aicli model Users        # migrations/env.py deleted (EC-005)

✗ Error: not inside an aicli project
  Missing: migrations/env.py
  Run this command from the root of a project created with `aicli new`.
```

### Plain (stderr is not a TTY, or `NO_COLOR` set, or `TERM=dumb`): no ANSI, no `✗ `

```
$ aicli model Users 2>&1 | cat          # migrations/env.py deleted

Error: not inside an aicli project
  Missing: migrations/env.py
  Run this command from the root of a project created with `aicli new`.
```

## Inspection failure (FR-006)

Only the first error is reported, with markers taken in the order of the Definitions (EC-012); the message of FR-003 is never written together with it.

### Styled

```
$ aicli model Users        # app/ is chmod 000 (EC-007)

✗ Error: cannot inspect app/main.py: permission denied
(✗ and "Error:" red, bold; the rest in the default color)

$ aicli model Users        # pyproject.toml deleted AND app/ is chmod 000 (EC-012)

✗ Error: cannot inspect app/main.py: permission denied
(no "Missing:" line: the check could not be completed)

$ aicli model Users        # alembic.ini is a circular symbolic link (EC-011)

✗ Error: cannot inspect alembic.ini: too many levels of symbolic links

$ aicli model Users        # the working directory was removed (EC-008)

✗ Error: cannot determine the current directory: no such file or directory
```

### Plain

```
$ aicli model Users 2>&1 | cat          # app/ is chmod 000

Error: cannot inspect app/main.py: permission denied
```

## Not affected

```
$ cd /tmp && aicli new my-app           # exempt (FR-007): works in any directory
$ cd /tmp && aicli model --help         # --help is exempt for every command
$ cd /tmp && aicli modl Users           # unknown command (EC-014): Cobra's own error, no check
```

Styling depends on stderr being a TTY, not stdout: `aicli model Users > out.txt` still shows the error in color on the terminal. Stdout is always empty on failure.

The text of `<cause>` is the system's error text and may differ between platforms; only the permission denied case is pinned by an exact-match test.

Colors are indicative; they are chosen at implementation time.
