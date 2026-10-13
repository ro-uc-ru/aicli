package scaffold

import (
	"errors"
	"io/fs"
	"regexp"
	"strings"
	"testing"

	"github.com/ro-uc-ru/aicli/templates"
)

// TestValidateName runs the table of valid and invalid names.
// In: t, the running test. Out: none. Side effects: none.
// Covers: FR-013, EC-001, EC-002, EC-011, EC-012.
func TestValidateName(t *testing.T) {
	tests := []struct {
		name    string
		in      string
		missing bool
		invalid bool
	}{
		{"valid", "my-app_1", false, false},
		{"single letter", "a", false, false},
		{"255 bytes", "a" + strings.Repeat("b", 254), false, false},
		{"empty", "", true, false},
		{"parent dir", "../x", false, true},
		{"separator", "a/b", false, true},
		{"leading digit", "1app", false, true},
		{"leading hyphen", "-app", false, true},
		{"leading dot", ".app", false, true},
		{"uppercase", "App", false, true},
		{"space", "my app", false, true},
		{"non-ascii", "café", false, true},
		{"256 bytes", "a" + strings.Repeat("b", 255), false, true},
		{"fastapi", "fastapi", false, true},
		{"underscore variant", "pydantic_settings", false, true},
		{"double underscore", "pydantic__settings", false, true},
		{"dev dependency", "pytest", false, true},
		{"not reserved", "fastapi2", false, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateName(tc.in)
			switch {
			case tc.missing:
				if !errors.Is(err, ErrMissingName) {
					t.Fatalf("got %v, want ErrMissingName", err)
				}
			case tc.invalid:
				if !errors.Is(err, ErrInvalidName) {
					t.Fatalf("got %v, want ErrInvalidName", err)
				}
			case err != nil:
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

// TestValidateNameReasonsDiffer checks that distinct rejections give
// distinct messages.
// In: t, the running test. Out: none. Side effects: none.
// Covers: FR-012, FR-013.
func TestValidateNameReasonsDiffer(t *testing.T) {
	msgs := map[string]bool{}
	for _, n := range []string{strings.Repeat("a", 256), "Bad", "fastapi"} {
		msgs[ValidateName(n).Error()] = true
	}
	if len(msgs) != 3 {
		t.Fatalf("reasons not distinct: %v", msgs)
	}
}

// TestReservedMatchesPyproject checks that reservedNames equals the
// dependencies declared in the embedded pyproject.toml.tmpl.
// In: t, the running test. Out: none. Side effects: none.
// Covers: FR-007, EC-011.
func TestReservedMatchesPyproject(t *testing.T) {
	b, err := fs.ReadFile(templates.Skeleton, "skeleton/pyproject.toml.tmpl")
	if err != nil {
		t.Fatal(err)
	}
	dep := regexp.MustCompile(`(?m)^\s+"([A-Za-z0-9._-]+)>=`)
	got := map[string]bool{}
	for _, m := range dep.FindAllStringSubmatch(string(b), -1) {
		got[normalize(m[1])] = true
	}
	if len(got) != len(reservedNames) {
		t.Fatalf("pyproject deps %v, reserved %v", got, reservedNames)
	}
	for n := range got {
		if !reservedNames[n] {
			t.Errorf("dependency %q missing from reservedNames", n)
		}
	}
}
