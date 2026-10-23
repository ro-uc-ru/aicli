package addllmprovider_test

import (
	"os/exec"
	"strings"
	"testing"

	"github.com/ro-uc-ru/aicli/cmd/addllmprovider"
	"github.com/ro-uc-ru/aicli/internal/feature"
)

// TestCommand checks the command name, its usage line and that its
// declaration is registered. The contract check on the real tree is
// TestRealAddTreeContract in cmd/root.
//
// Covers: FR-012
func TestCommand(t *testing.T) {
	cmd := addllmprovider.NewCommand()
	if cmd.Name() != "llm-provider" || cmd.Use != "llm-provider <provider>..." {
		t.Fatalf("use: %q", cmd.Use)
	}
	if _, ok := feature.Lookup(cmd); !ok {
		t.Fatal("declaration not registered")
	}
}

// TestImportsNoCmd checks that the package imports no other cmd/ package.
//
// Covers: FR-012
func TestImportsNoCmd(t *testing.T) {
	out, err := exec.Command("go", "list", "-deps", ".").Output()
	if err != nil {
		t.Fatal(err)
	}
	for _, l := range strings.Fields(string(out)) {
		if strings.Contains(l, "/cmd/") && !strings.HasSuffix(l, "/cmd/addllmprovider") {
			t.Errorf("imports %s", l)
		}
	}
}
