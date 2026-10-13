// Package scaffold validates project names and creates generated projects
// on disk.
package scaffold

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

// MaxNameLen is the maximum length of a project name, in bytes (EC-012).
const MaxNameLen = 255

// ErrMissingName is returned by ValidateName for an empty name (FR-012, EC-003).
var ErrMissingName = errors.New("project name is missing")

// ErrInvalidName is wrapped by every ValidateName rejection of a non-empty
// name (FR-013). Use errors.Is to detect it; the message gives the reason.
var ErrInvalidName = errors.New("invalid project name")

// reservedNames lists the dependencies declared in FR-007 in their PEP 503
// normalized form. A test keeps it equal to the embedded pyproject.toml.tmpl.
var reservedNames = map[string]bool{
	"fastapi":           true,
	"uvicorn":           true,
	"sqlalchemy":        true,
	"alembic":           true,
	"pydantic-settings": true,
	"pytest":            true,
	"httpx":             true,
	"ruff":              true,
}

// nameRe is the shape of a valid project name (EC-001, EC-002).
var nameRe = regexp.MustCompile(`^[a-z][a-z0-9_-]*$`)

// sepRe matches the runs of '-' and '_' that PEP 503 reads as one '-'.
var sepRe = regexp.MustCompile(`[-_]+`)

// normalize applies the PEP 503 name normalization.
// In: name, a project or dependency name. Out: the normalized name.
// No side effects.
func normalize(name string) string {
	return sepRe.ReplaceAllString(strings.ToLower(name), "-")
}

// ValidateName checks a project name given to `aicli new`.
// In: name, the raw name. Out: nil if valid; ErrMissingName if empty;
// otherwise an error wrapping ErrInvalidName whose message states the
// reason (too long, bad characters or start, reserved dependency name).
// No side effects.
func ValidateName(name string) error {
	if name == "" {
		return ErrMissingName
	}
	if len(name) > MaxNameLen {
		return fmt.Errorf("%w: longer than %d bytes", ErrInvalidName, MaxNameLen)
	}
	if !nameRe.MatchString(name) {
		return fmt.Errorf("%w: %q must start with a lowercase letter and contain only lowercase letters, digits, '-' and '_'", ErrInvalidName, name)
	}
	if reservedNames[normalize(name)] {
		return fmt.Errorf("%w: %q is the name of a dependency of the generated project", ErrInvalidName, name)
	}
	return nil
}
