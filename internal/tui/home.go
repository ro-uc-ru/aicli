package tui

import (
	"io"
	"os"

	"golang.org/x/term"
)

// Entry is one command listed on the home screen. It is the only interface
// between internal/home and internal/tui.
type Entry struct {
	// Path is the full command path without the program name, e.g.
	// "new" or "db migrate".
	Path string
	// Short is the one-line description; empty when the command has none.
	Short string
	// Help produces the full help text of the command: it returns the text,
	// or an error if the help cannot be produced. It has no side effects
	// on the terminal.
	Help func() (string, error)
}

// IsInteractive reports whether the home screen may run as an interactive
// program.
//
// In: in, the input stream (typically os.Stdin); out, the output stream
// (typically os.Stdout); getenv, the environment lookup (typically
// os.Getenv).
// Out: true only if both in and out are *os.File terminals and TERM is not
// "dumb"; false otherwise, including when either is not an *os.File.
// Side effects: none beyond querying the terminal state of in and out.
func IsInteractive(in io.Reader, out io.Writer, getenv func(string) string) bool {
	fi, ok := in.(*os.File)
	if !ok || !term.IsTerminal(int(fi.Fd())) {
		return false
	}
	fo, ok := out.(*os.File)
	if !ok || !term.IsTerminal(int(fo.Fd())) {
		return false
	}
	return getenv("TERM") != "dumb"
}
