package llmprovider_test

import (
	"bytes"
	"testing"

	"github.com/ro-uc-ru/aicli/internal/feature"
	"github.com/ro-uc-ru/aicli/internal/featuretest"
	"github.com/ro-uc-ru/aicli/internal/gen"
	"github.com/ro-uc-ru/aicli/internal/llmprovider"
)

// TestDeclarationFollowsContract runs the feature contract check, with its
// multi-choice rules, on a tree holding the llm-provider feature: it must
// give no error. The rule-breaking multi-choice features are checked in
// internal/featuretest and the real tree in cmd/root.
//
// Covers: FR-001, FR-012
func TestDeclarationFollowsContract(t *testing.T) {
	add := featuretest.NewTree(feature.NewCommand(llmprovider.Declaration()))
	if errs := feature.CheckContract(add); len(errs) != 0 {
		t.Fatalf("contract errors: %v", errs)
	}
}

// TestSharedEntriesIndependentOfProviders renders each shared entry of the
// plan for all 15 non-empty sets of providers, in choice order, and checks
// that the bytes are identical for every set.
//
// Covers: FR-001, FR-012
func TestSharedEntriesIndependentOfProviders(t *testing.T) {
	choices := []string{"openai", "claude", "gemini", "ollama"}
	var want map[string][]byte
	sets := 0
	for mask := 1; mask < 1<<len(choices); mask++ {
		var values []string
		for i, c := range choices {
			if mask&(1<<i) != 0 {
				values = append(values, c)
			}
		}
		entries, err := llmprovider.Plan(values)
		if err != nil {
			t.Fatal(err)
		}
		got := map[string][]byte{}
		for _, e := range entries {
			if e.Action != gen.ActionShared {
				continue
			}
			b, err := gen.RenderEntry(e)
			if err != nil {
				t.Fatal(err)
			}
			got[e.Path] = b
		}
		sets++
		if want == nil {
			want = got
			if len(want) != 6 {
				t.Fatalf("got %d shared entries, want 6", len(want))
			}
			continue
		}
		for p, b := range want {
			if !bytes.Equal(b, got[p]) {
				t.Errorf("%s differs for %v", p, values)
			}
		}
	}
	if sets != 15 {
		t.Fatalf("checked %d sets, want 15", sets)
	}
}
