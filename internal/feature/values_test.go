package feature

import (
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// update regenerates the golden files instead of comparing against them.
var update = flag.Bool("update", false, "regenerate golden files")

// valuesFixture builds a test add tree with a feature of three inputs: free
// text with a recording validator, a choice input, and a plain text input.
func valuesFixture(t *testing.T, seen *[]string) (*cobra.Command, *Declaration) {
	t.Helper()
	cmd := NewCommand(Declaration{Name: "feature-1", Inputs: []Input{
		{Name: "input-1", Validator: func(v string) string {
			*seen = append(*seen, v)
			if v == "bad" {
				return "input-1 is bad"
			}
			return ""
		}},
		{Name: "input-2", Choices: []string{"red", "green"}},
		{Name: "input-3"},
	}})
	root := &cobra.Command{Use: "aicli"}
	add := &cobra.Command{Use: "add"}
	root.AddCommand(add)
	add.AddCommand(cmd)
	d, _ := Lookup(cmd)
	return cmd, d
}

// TestValidateValues checks each failure, the order between them, trimming,
// choices in another letter case, and that the validator never receives an
// empty value or one with control characters. The plain messages are compared
// with golden files.
//
// Covers: FR-006, FR-014, FR-016, EC-009, EC-013
func TestValidateValues(t *testing.T) {
	tests := []struct {
		name string
		args []string
		ok   bool
	}{
		{"too-few-none", nil, false},
		{"too-few-second", []string{"a"}, false},
		{"too-many", []string{"a", "red", "c", "d"}, false},
		{"too-few-before-empty", []string{" "}, false},
		{"empty", []string{"  ", "red", "c"}, false},
		{"control", []string{"a\nb", "red", "c"}, false},
		{"control-c1", []string{"a", "red", "c\u0085d"}, false},
		{"control-del", []string{"a", "red", "c\x7f"}, false},
		{"empty-before-control-order", []string{"a", "", "c\n"}, false},
		{"control-before-choice", []string{"a", "x\ty", "c"}, false},
		{"choice-case", []string{"a", "Red", "c"}, false},
		{"choice-unknown", []string{"a", "blue", "c"}, false},
		{"declaration-order-validator-first", []string{"bad", "blue", ""}, false},
		{"validator", []string{"bad", "red", "c"}, false},
		{"valid-trimmed", []string{"  a ", "\tgreen ", " c"}, true},
	}
	var got strings.Builder
	for _, tc := range tests {
		var seen []string
		cmd, d := valuesFixture(t, &seen)
		vals, err := ValidateValues(cmd, d, tc.args)
		if (err == nil) != tc.ok {
			t.Fatalf("%s: err = %v, want ok=%v", tc.name, err, tc.ok)
		}
		if tc.ok {
			if strings.Join(vals, "|") != "a|green|c" {
				t.Fatalf("%s: values = %q", tc.name, vals)
			}
			continue
		}
		for _, v := range seen {
			if v == "" || strings.IndexFunc(v, func(r rune) bool { return r < 0x20 || (r >= 0x7f && r < 0xa0) }) >= 0 {
				t.Fatalf("%s: validator received %q", tc.name, v)
			}
		}
		got.WriteString("== " + tc.name + "\n" + err.Error() + "\n")
	}
	path := filepath.Join("testdata", "values.golden")
	if *update {
		if err := os.MkdirAll("testdata", 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(got.String()), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if got.String() != string(want) {
		t.Fatalf("messages differ from %s:\n%s", path, got.String())
	}
}
