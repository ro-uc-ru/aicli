package root

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

// TestAddStaticGoldenTestTree runs `aicli add` in static mode (buffers) in a
// project root with the T-011 tree registered and compares stdout with the
// golden file: the header, the description, one line per listed feature, the
// closing line, and no unlisted feature.
//
// In: t, the test; run with -update to regenerate the golden file.
// Out: none. Side effects: writes testdata/add-static-tree.golden with
// -update.
// Covers: FR-005
func TestAddStaticGoldenTestTree(t *testing.T) {
	code, out, errb := runAdd(t, projectDir(t), "add")
	if code != 0 || errb != "" {
		t.Fatalf("code=%d stderr=%q, want 0 and empty", code, errb)
	}
	compareGolden(t, "add-static-tree.golden", []byte(out))
	if !strings.HasSuffix(out, "Run `aicli add <feature> --help` to see its inputs.\n") || strings.ContainsRune(out, '\x1b') {
		t.Errorf("closing line missing or control sequence present:\n%q", out)
	}
}

// TestAddStaticGoldenRealTree runs `aicli add` in static mode in a project
// root with the real tree, which has the llm-provider feature, and compares
// stdout with the golden file: the listing and the closing line.
//
// In: t, the test; run with -update to regenerate the golden file.
// Out: none. Side effects: writes testdata/add-static-real.golden with
// -update.
// Covers: FR-005, FR-012
func TestAddStaticGoldenRealTree(t *testing.T) {
	cmd := newRoot(".", func() (string, error) { return projectDir(t), nil })
	var out, errb bytes.Buffer
	if code := run(context.Background(), cmd, []string{"add"}, &out, &errb); code != 0 || errb.Len() != 0 {
		t.Fatalf("code=%d stderr=%q, want 0 and empty", code, errb.String())
	}
	compareGolden(t, "add-static-real.golden", out.Bytes())
	if !strings.Contains(out.String(), "llm-provider  Add one or more LLM providers.\n") || strings.ContainsRune(out.String(), '\x1b') {
		t.Errorf("llm-provider line missing or control sequence present:\n%q", out.String())
	}
}
