package root

import (
	"testing"

	"github.com/ro-uc-ru/aicli/internal/feature"
)

// TestRealAddTreeContract runs the feature contract check on the `add`
// command of the real tree built by NewRoot: it must give no error.
//
// In: t, the test. Out: none. Side effects: none.
// Covers: FR-003
func TestRealAddTreeContract(t *testing.T) {
	add, _, err := NewRoot(".").Find([]string{"add"})
	if err != nil || add.Name() != "add" {
		t.Fatalf("add command not found: %v", err)
	}
	if errs := feature.CheckContract(add); len(errs) != 0 {
		t.Fatalf("contract errors: %v", errs)
	}
}
