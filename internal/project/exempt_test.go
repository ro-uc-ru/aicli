package project

import (
	"slices"
	"testing"
)

// TestExemptListContent pins the exact content of the exempt list.
//
// Covers: FR-007
func TestExemptListContent(t *testing.T) {
	want := []string{"new", "help", "version", "completion", "__complete"}
	if !slices.Equal(exemptCommands, want) {
		t.Fatalf("exemptCommands = %v, want %v", exemptCommands, want)
	}
}

// TestIsExempt checks that the predicate is true for every listed command
// and false for any other name.
//
// Covers: FR-007
func TestIsExempt(t *testing.T) {
	for _, n := range []string{"new", "help", "version", "completion", "__complete"} {
		if !IsExempt(n) {
			t.Errorf("IsExempt(%q) = false, want true", n)
		}
	}
	for _, n := range []string{"", "generate", "model", "New", "completions", "aicli"} {
		if IsExempt(n) {
			t.Errorf("IsExempt(%q) = true, want false", n)
		}
	}
}
