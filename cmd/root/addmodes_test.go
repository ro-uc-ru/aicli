package root

import (
	"strings"
	"testing"

	"github.com/ro-uc-ru/aicli/internal/featuretest"
)

// TestAddLLMProviderStaticNoArguments runs `aicli add llm-provider` with no
// arguments in static mode (no TTY) in a scaffolded project: it fails with
// the message and usage line of FR-002, shows no form, creates nothing and
// leaves stdout empty.
//
// In: t, the test. Out: none. Side effects: writes in a temp directory.
// Covers: FR-005, EC-001
func TestAddLLMProviderStaticNoArguments(t *testing.T) {
	dir := scaffolded(t)
	before := snapshot(t, dir)
	code, out, errb := runReal(t, dir, "add", "llm-provider")
	if code != 1 || out != "" {
		t.Fatalf("code=%d stdout=%q, want 1 and empty", code, out)
	}
	want := "Error: llm-provider requires provider\n  Usage: aicli add llm-provider <provider>... [flags]\n"
	if errb != want {
		t.Fatalf("stderr = %q, want %q", errb, want)
	}
	wantUnchanged(t, dir, before)
}

// TestAddNoInputsFeatureRunsWithoutScreens runs a feature that declares no
// inputs with no arguments: it runs at once and reports its file, with no
// form or confirmation on stdout. The interactive-mode half of this rule needs
// a terminal and is covered by TestPrepareDirectSkips in internal/add.
//
// In: t, the test. Out: none. Side effects: writes in a temp directory.
// Covers: FR-005
func TestAddNoInputsFeatureRunsWithoutScreens(t *testing.T) {
	dir := projectDir(t)
	code, out, errb := runAdd(t, dir, "add", featuretest.NoInputs)
	if code != 0 || errb != "" {
		t.Fatalf("code=%d stderr=%q, want 0 and empty", code, errb)
	}
	if !strings.Contains(out, "Added "+featuretest.NoInputs) || ansiSeq.MatchString(out) || strings.Contains(out, "Will create") {
		t.Fatalf("stdout = %q", out)
	}
}
