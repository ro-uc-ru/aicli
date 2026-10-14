package project

import (
	"context"
	"testing"
)

// TestRootContext checks that a stored root reads back unchanged and that a
// context without one reports its absence.
//
// Covers: FR-009
func TestRootContext(t *testing.T) {
	ctx := WithRoot(context.Background(), "/work/my-api")
	got, ok := RootFrom(ctx)
	if !ok || got != "/work/my-api" {
		t.Fatalf("RootFrom = %q, %v; want %q, true", got, ok, "/work/my-api")
	}

	if got, ok := RootFrom(context.Background()); ok || got != "" {
		t.Fatalf("RootFrom(empty) = %q, %v; want \"\", false", got, ok)
	}
}
