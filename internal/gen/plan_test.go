package gen

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"testing/fstest"
)

// planTemplates is the template source of the CheckPlan tests.
var planTemplates = fstest.MapFS{
	"file.tmpl": {Data: []byte("x\n")},
	"dep.tmpl":  {Data: []byte("\"{{.}}\",\n")},
	"env.tmpl":  {Data: []byte("{{.}}=\n")},
}

// planEntry builds an ActionCreate or ActionShared entry of the CheckPlan tests.
// In: p, the path; a, the action. Out: the entry. Side effects: none.
func planEntry(p string, a FileAction) FeatureFile {
	return FeatureFile{Path: p, Templates: planTemplates, Template: "file.tmpl", Action: a}
}

// planInject builds an ActionInject entry of the CheckPlan tests.
// In: p, the target; marker, the marker name; tmpl, the template; data, its
// value. Out: the entry. Side effects: none.
func planInject(p, marker, tmpl, data string) FeatureFile {
	return FeatureFile{Path: p, Templates: planTemplates, Template: tmpl, Data: data, Action: ActionInject, Marker: marker}
}

// writePlanFile writes a file under root, creating its parents.
// In: t, the running test; root, the root; p, the "/" path; content, the
// bytes. Out: none. Side effects: writes the file; fails the test on error.
func writePlanFile(t *testing.T, root, p, content string) {
	t.Helper()
	full := filepath.Join(root, filepath.FromSlash(p))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// symlinkPlan creates a symbolic link under root, creating its parents.
// In: t, the running test; root, the root; target, the link target; p, the
// "/" path of the link. Out: none. Side effects: creates the link.
func symlinkPlan(t *testing.T, root, target, p string) {
	t.Helper()
	full := filepath.Join(root, filepath.FromSlash(p))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, full); err != nil {
		t.Fatal(err)
	}
}

// snapshotTree records every path under root with its mode and, for regular
// files, its content, or its target for links, without following links.
// In: t, the running test; root, the directory. Out: path -> description.
// Side effects: fails the test on read errors.
func snapshotTree(t *testing.T, root string) map[string]string {
	t.Helper()
	out := map[string]string{}
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		info, err := os.Lstat(p)
		if err != nil {
			return err
		}
		desc := info.Mode().String()
		switch {
		case info.Mode().IsRegular():
			b, err := os.ReadFile(p)
			if err != nil {
				return err
			}
			desc += " " + string(b)
		case info.Mode()&os.ModeSymlink != 0:
			target, err := os.Readlink(p)
			if err != nil {
				return err
			}
			desc += " -> " + target
		}
		out[p] = desc
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return out
}

// baseProject writes the injection targets shared by the CheckPlan tests.
// In: t, the running test; root, the project root. Out: none.
// Side effects: writes pyproject.toml and .env.example.
func baseProject(t *testing.T, root string) {
	t.Helper()
	writePlanFile(t, root, "pyproject.toml", "dependencies = [\n    \"fastapi\",\n    # aicli:dependencies\n]\n")
	writePlanFile(t, root, ".env.example", "APP=1\n# aicli:env\n")
}

// TestCheckPlanErrors checks every failure of CheckPlan, its check order and
// its plain message against testdata/checkplan.golden, and that the tree is
// unchanged.
//
// Covers: FR-003, FR-010, FR-011, EC-004, EC-005, EC-010, EC-028
func TestCheckPlanErrors(t *testing.T) {
	providers := func(values ...string) Plan {
		p := Plan{ItemNoun: "providers"}
		for _, v := range values {
			item := "app/ai/llm/" + v + ".py"
			p.Entries = append(p.Entries, planEntry(item, ActionCreate), planEntry("tests/test_llm_"+v+".py", ActionCreate))
			p.Items = append(p.Items, Item{Value: v, Path: item})
		}
		p.Entries = append(p.Entries,
			planEntry("app/ai/llm/__init__.py", ActionShared),
			planInject("pyproject.toml", "dependencies", "dep.tmpl", "sdk>=1"),
			planInject(".env.example", "env", "env.tmpl", "KEY"))
		return p
	}
	tests := []struct {
		name  string
		setup func(t *testing.T, root string)
		plan  Plan
	}{
		{
			name:  "target-missing",
			setup: func(t *testing.T, root string) { writePlanFile(t, root, "pyproject.toml", "# aicli:dependencies\n") },
			plan:  providers("openai"),
		},
		{
			name: "target-broken-link",
			setup: func(t *testing.T, root string) {
				writePlanFile(t, root, ".env.example", "# aicli:env\n")
				symlinkPlan(t, root, filepath.Join(root, "gone.toml"), "pyproject.toml")
			},
			plan: providers("openai"),
		},
		{
			name: "target-directory",
			setup: func(t *testing.T, root string) {
				baseProject(t, root)
				if err := os.Remove(filepath.Join(root, ".env.example")); err != nil {
					t.Fatal(err)
				}
				if err := os.Mkdir(filepath.Join(root, ".env.example"), 0o755); err != nil {
					t.Fatal(err)
				}
			},
			plan: providers("openai"),
		},
		{
			name: "no-marker",
			setup: func(t *testing.T, root string) {
				baseProject(t, root)
				writePlanFile(t, root, ".env.example", "APP=1\n")
			},
			plan: providers("openai"),
		},
		{
			name: "repeated-marker",
			setup: func(t *testing.T, root string) {
				baseProject(t, root)
				writePlanFile(t, root, "pyproject.toml", "# aicli:dependencies\n  # aicli:dependencies\n")
			},
			plan: providers("openai"),
		},
		{
			name: "marker-before-items",
			setup: func(t *testing.T, root string) {
				baseProject(t, root)
				writePlanFile(t, root, ".env.example", "")
				writePlanFile(t, root, "app/ai/llm/openai.py", "")
			},
			plan: providers("openai"),
		},
		{
			name: "added-file-dir-link",
			setup: func(t *testing.T, root string) {
				baseProject(t, root)
				writePlanFile(t, root, "app/ai/llm/openai.py", "")
				if err := os.MkdirAll(filepath.Join(root, "app/ai/llm/claude.py"), 0o755); err != nil {
					t.Fatal(err)
				}
				symlinkPlan(t, root, filepath.Join(root, "nowhere"), "app/ai/llm/gemini.py")
			},
			plan: providers("openai", "claude", "gemini", "ollama"),
		},
		{
			name: "added-mixed",
			setup: func(t *testing.T, root string) {
				baseProject(t, root)
				writePlanFile(t, root, "app/ai/llm/openai.py", "")
				writePlanFile(t, root, "tests/test_llm_openai.py", "")
				writePlanFile(t, root, "tests/test_llm_claude.py", "")
			},
			plan: providers("openai", "claude"),
		},
		{
			name: "existing-fallback",
			setup: func(t *testing.T, root string) {
				baseProject(t, root)
				writePlanFile(t, root, "tests/test_llm_claude.py", "")
				writePlanFile(t, root, "tests/test_llm_openai.py", "")
			},
			plan: providers("openai", "claude"),
		},
		{
			name: "shared-directory",
			setup: func(t *testing.T, root string) {
				baseProject(t, root)
				writePlanFile(t, root, "tests/test_llm_openai.py", "")
				if err := os.MkdirAll(filepath.Join(root, "app/ai/llm/__init__.py"), 0o755); err != nil {
					t.Fatal(err)
				}
			},
			plan: providers("openai"),
		},
		{
			name: "shared-broken-link",
			setup: func(t *testing.T, root string) {
				baseProject(t, root)
				symlinkPlan(t, root, filepath.Join(root, "nowhere"), "app/ai/llm/__init__.py")
			},
			plan: providers("openai"),
		},
		{
			name: "shared-link-outside",
			setup: func(t *testing.T, root string) {
				baseProject(t, root)
				outside := filepath.Join(t.TempDir(), "init.py")
				if err := os.WriteFile(outside, nil, 0o644); err != nil {
					t.Fatal(err)
				}
				symlinkPlan(t, root, outside, "app/ai/llm/__init__.py")
			},
			plan: providers("openai"),
		},
	}
	var got strings.Builder
	for _, tc := range tests {
		root := t.TempDir()
		tc.setup(t, root)
		before := snapshotTree(t, root)
		_, err := CheckPlan(root, "llm-provider", tc.plan)
		if err == nil {
			t.Fatalf("%s: got nil error", tc.name)
		}
		if after := snapshotTree(t, root); !reflect.DeepEqual(before, after) {
			t.Errorf("%s: tree changed:\n%v\n->\n%v", tc.name, before, after)
		}
		got.WriteString("== " + tc.name + "\n" + err.Error() + "\n")
	}
	path := filepath.Join("testdata", "checkplan.golden")
	if *update {
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

// TestCheckPlanPreview checks the Preview: missing shared files are listed,
// existing ones (also through an inside link) are not, and only the targets
// an injection would change are listed, once each.
//
// Covers: FR-011, EC-010
func TestCheckPlanPreview(t *testing.T) {
	root := t.TempDir()
	baseProject(t, root)
	writePlanFile(t, root, ".env.example", "APP=1\n# aicli:env\nKEY=\n")
	writePlanFile(t, root, "app/api/routers/llm.py", "user edit\n")
	writePlanFile(t, root, "real_init.py", "")
	symlinkPlan(t, root, filepath.Join(root, "real_init.py"), "app/ai/llm/__init__.py")
	plan := Plan{
		ItemNoun: "providers",
		Items:    []Item{{Value: "openai", Path: "app/ai/llm/openai.py"}},
		Entries: []FeatureFile{
			planEntry("app/ai/llm/openai.py", ActionCreate),
			planEntry("app/ai/llm/__init__.py", ActionShared),
			planEntry("app/ai/llm/base.py", ActionShared),
			planEntry("app/api/routers/llm.py", ActionShared),
			planInject(".env.example", "env", "env.tmpl", "KEY"),
			planInject("pyproject.toml", "dependencies", "dep.tmpl", "openai>=3.36"),
			planInject("pyproject.toml", "dependencies", "dep.tmpl", "openai>=3.36"),
			planEntry("tests/test_llm_openai.py", ActionCreate),
		},
	}
	before := snapshotTree(t, root)
	got, err := CheckPlan(root, "llm-provider", plan)
	if err != nil {
		t.Fatal(err)
	}
	want := Preview{
		Create: []string{"app/ai/llm/openai.py", "app/ai/llm/base.py", "tests/test_llm_openai.py"},
		Update: []string{"pyproject.toml"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v, want %+v", got, want)
	}
	if after := snapshotTree(t, root); !reflect.DeepEqual(before, after) {
		t.Errorf("tree changed:\n%v\n->\n%v", before, after)
	}
}

// providerPlan builds a plan shaped like the LLM provider feature for the
// ApplyPlan tests: per value an item file and a test file, two shared files,
// one dependency per value and one variable per value.
// In: values, the provider names in choice order. Out: the plan.
// Side effects: none.
func providerPlan(values ...string) Plan {
	p := Plan{ItemNoun: "providers"}
	for _, v := range values {
		item := "app/ai/llm/" + v + ".py"
		p.Entries = append(p.Entries, planEntry(item, ActionCreate))
		p.Items = append(p.Items, Item{Value: v, Path: item})
	}
	p.Entries = append(p.Entries,
		planEntry("app/ai/llm/__init__.py", ActionShared),
		planEntry("app/api/routers/llm.py", ActionShared))
	for _, v := range values {
		p.Entries = append(p.Entries, planInject("pyproject.toml", "dependencies", "dep.tmpl", "sdk>=1"))
		p.Entries = append(p.Entries, planInject(".env.example", "env", "env.tmpl", strings.ToUpper(v)+"_KEY"))
	}
	for _, v := range values {
		p.Entries = append(p.Entries, planEntry("tests/test_llm_"+v+".py", ActionCreate))
	}
	return p
}

// readPlanFile reads a file under root.
// In: t, the running test; root, the root; p, the "/" path. Out: its content.
// Side effects: fails the test on read errors.
func readPlanFile(t *testing.T, root, p string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(p)))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// TestApplyPlanRuns checks a first run, then a later run with another value
// and a shared file deleted and another edited by the user.
//
// Covers: FR-008, FR-009, FR-011, EC-009
func TestApplyPlanRuns(t *testing.T) {
	root := t.TempDir()
	baseProject(t, root)
	ctx := context.Background()

	got, err := ApplyPlan(ctx, OSFeatureFS{}, root, "llm-provider", providerPlan("openai"))
	if err != nil {
		t.Fatal(err)
	}
	want := []ReportLine{
		{VerbCreate, "app/ai/llm/openai.py"},
		{VerbCreate, "app/ai/llm/__init__.py"},
		{VerbCreate, "app/api/routers/llm.py"},
		{VerbUpdate, "pyproject.toml"},
		{VerbUpdate, ".env.example"},
		{VerbCreate, "tests/test_llm_openai.py"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("first run: got %v, want %v", got, want)
	}
	if s := readPlanFile(t, root, "pyproject.toml"); s != "dependencies = [\n    \"fastapi\",\n    # aicli:dependencies\n    \"sdk>=1\",\n]\n" {
		t.Errorf("pyproject.toml: %q", s)
	}
	if s := readPlanFile(t, root, ".env.example"); s != "APP=1\n# aicli:env\nOPENAI_KEY=\n" {
		t.Errorf(".env.example: %q", s)
	}

	if err := os.Remove(filepath.Join(root, "app/ai/llm/__init__.py")); err != nil {
		t.Fatal(err)
	}
	writePlanFile(t, root, "app/api/routers/llm.py", "user edit\n")
	got, err = ApplyPlan(ctx, OSFeatureFS{}, root, "llm-provider", providerPlan("claude"))
	if err != nil {
		t.Fatal(err)
	}
	want = []ReportLine{
		{VerbCreate, "app/ai/llm/claude.py"},
		{VerbCreate, "app/ai/llm/__init__.py"},
		{VerbSkip, "app/api/routers/llm.py"},
		{VerbSkip, "pyproject.toml"},
		{VerbUpdate, ".env.example"},
		{VerbCreate, "tests/test_llm_claude.py"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("later run: got %v, want %v", got, want)
	}
	if s := readPlanFile(t, root, "app/api/routers/llm.py"); s != "user edit\n" {
		t.Errorf("edited shared file replaced: %q", s)
	}
	if s := readPlanFile(t, root, "app/ai/llm/__init__.py"); s != "x\n" {
		t.Errorf("deleted shared file not created again: %q", s)
	}
	if s := readPlanFile(t, root, ".env.example"); s != "APP=1\n# aicli:env\nCLAUDE_KEY=\nOPENAI_KEY=\n" {
		t.Errorf(".env.example: %q", s)
	}
}

// TestApplyPlanItemRace checks that an item file created after the check is
// never replaced, the run fails and nothing of this run is left.
//
// Covers: FR-008, EC-007
func TestApplyPlanItemRace(t *testing.T) {
	root := t.TempDir()
	baseProject(t, root)
	fsys := &racyFS{hook: filepath.Join(root, "app", "ai", "llm", "openai.py")}
	if err := os.MkdirAll(filepath.Join(root, "app", "ai", "llm"), 0o755); err != nil {
		t.Fatal(err)
	}
	before := snapshotTree(t, root)
	if _, err := ApplyPlan(context.Background(), fsys, root, "llm-provider", providerPlan("openai")); err == nil {
		t.Fatal("expected an error")
	}
	if s := readPlanFile(t, root, "app/ai/llm/openai.py"); s != "theirs" {
		t.Errorf("item file replaced: %q", s)
	}
	after := snapshotTree(t, root)
	delete(after, filepath.Join(root, "app", "ai", "llm", "openai.py"))
	if !reflect.DeepEqual(before, after) {
		t.Errorf("tree changed:\n%v\n->\n%v", before, after)
	}
}

// TestApplyPlanInjectThroughLink checks that a target reached through a link
// inside the root is modified at the file it resolves to, the link stays a
// link and the file keeps its mode.
//
// Covers: FR-009, EC-029
func TestApplyPlanInjectThroughLink(t *testing.T) {
	root := t.TempDir()
	writePlanFile(t, root, ".env.example", "# aicli:env\n")
	writePlanFile(t, root, "config/pyproject.toml", "deps = [\n  # aicli:dependencies\n]\n")
	real := filepath.Join(root, "config", "pyproject.toml")
	if err := os.Chmod(real, 0o600); err != nil {
		t.Fatal(err)
	}
	symlinkPlan(t, root, filepath.Join("config", "pyproject.toml"), "pyproject.toml")
	plan := Plan{Entries: []FeatureFile{planInject("pyproject.toml", "dependencies", "dep.tmpl", "sdk>=1")}}
	got, err := ApplyPlan(context.Background(), OSFeatureFS{}, root, "llm-provider", plan)
	if err != nil {
		t.Fatal(err)
	}
	if want := []ReportLine{{VerbUpdate, "pyproject.toml"}}; !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	info, err := os.Lstat(filepath.Join(root, "pyproject.toml"))
	if err != nil || info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("link not kept: %v %v", info, err)
	}
	info, err = os.Stat(real)
	if err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("mode not kept: %v %v", info, err)
	}
	if s := readPlanFile(t, root, "config/pyproject.toml"); s != "deps = [\n  # aicli:dependencies\n  \"sdk>=1\",\n]\n" {
		t.Errorf("content: %q", s)
	}
}
