// Package llmprovider declares the `aicli add llm-provider` feature: the
// providers it can add, the files it creates and the lines it injects into a
// generated project (spec 005, FR-012..FR-014).
package llmprovider

import (
	"io/fs"
	"strings"

	"github.com/ro-uc-ru/aicli/internal/feature"
	"github.com/ro-uc-ru/aicli/internal/gen"
	"github.com/ro-uc-ru/aicli/templates"
)

// Name is the command name of the feature.
const Name = "llm-provider"

// Short is the one-line description of the feature.
const Short = "Add one or more LLM providers."

// InputName is the name of the multi-choice input.
const InputName = "provider"

// providerInfo is what the feature knows about one provider.
type providerInfo struct {
	// name is the provider name, as a choice of the input.
	name string
	// sdk is the Python distribution of the SDK the provider uses.
	sdk string
	// requirement is the sdk with its minimum version.
	requirement string
	// vars are the required environment variables, in declaration order.
	vars []string
	// optional is the commented optional line for .env.example, or "".
	optional string
}

// providers lists the providers in choice order (FR-012).
var providers = []providerInfo{
	{name: "openai", sdk: "openai", requirement: "openai>=3.36", vars: []string{"OPENAI_API_KEY", "OPENAI_MODEL"}},
	{name: "claude", sdk: "anthropic", requirement: "anthropic>=2.0", vars: []string{"ANTHROPIC_API_KEY", "ANTHROPIC_MODEL"}},
	{name: "gemini", sdk: "google-genai", requirement: "google-genai>=2.32", vars: []string{"GEMINI_API_KEY", "GEMINI_MODEL"}},
	{name: "ollama", sdk: "openai", requirement: "openai>=3.36", vars: []string{"OLLAMA_MODEL"}, optional: "# OLLAMA_BASE_URL=http://localhost:11434/v1"},
}

// templateFS is the filesystem that holds the templates of the feature, the
// "llmprovider" directory of templates.LLMProvider (ARC-5).
var templateFS = mustSub(templates.LLMProvider, "llmprovider")

// mustSub returns the sub-tree of fsys at dir.
//
// In: fsys, an embedded filesystem; dir, a directory of it. Out: the sub-tree.
// Side effects: panics if dir is not a valid directory name, which cannot
// happen for the constant used here.
func mustSub(fsys fs.FS, dir string) fs.FS {
	sub, err := fs.Sub(fsys, dir)
	if err != nil {
		panic(err)
	}
	return sub
}

// Data is the value passed to the template of an entry.
type Data struct {
	// Provider is the provider name, for the per-provider entries.
	Provider string
	// SDK is the SDK distribution, for the dependency entries.
	SDK string
	// Requirement is the SDK with its minimum version (e.g. "openai>=3.36"),
	// for the dependency entries.
	Requirement string
	// Line is the full line, for the .env.example entries.
	Line string
}

// itemFile returns the item file of a provider.
//
// In: name, a provider name. Out: "app/ai/llm/<name>.py". No side effects.
func itemFile(name string) string { return "app/ai/llm/" + name + ".py" }

// Declaration returns the declaration of the llm-provider feature.
//
// In: none.
// Out: the declaration: name, short description, the multi-choice input
// "provider" over openai, claude, gemini and ollama with item noun
// "providers" and item files app/ai/llm/<provider>.py, Plan and NextSteps.
// No side effects.
func Declaration() feature.Declaration {
	choices := make([]string, len(providers))
	files := make(map[string]string, len(providers))
	for i, p := range providers {
		choices[i] = p.name
		files[p.name] = itemFile(p.name)
	}
	return feature.Declaration{
		Name:  Name,
		Short: Short,
		Inputs: []feature.Input{{
			Name:        InputName,
			Description: "LLM providers to add",
			Choices:     choices,
			Multi:       true,
			ItemNoun:    "providers",
			ItemFiles:   files,
		}},
		Plan:      Plan,
		NextSteps: NextSteps,
	}
}

// selected returns the infos of the given provider names, in choice order.
//
// In: values, provider names (any order). Out: the matching infos in choice
// order; unknown names are ignored. No side effects.
func selected(values []string) []providerInfo {
	var out []providerInfo
	for _, p := range providers {
		for _, v := range values {
			if v == p.name {
				out = append(out, p)
				break
			}
		}
	}
	return out
}

// Plan computes the entries of the feature in the order of FR-013.
//
// In: values, the chosen providers (any order).
// Out: six shared entries; per provider, its module and its test; the two
// injections into app/api/router.py; one pyproject.toml injection per distinct
// SDK in order of first appearance; one .env.example injection per line. The
// error is always nil.
// No side effects.
func Plan(values []string) ([]gen.FeatureFile, error) {
	sel := selected(values)
	var files []gen.FeatureFile
	for _, p := range []string{
		"app/ai/llm/__init__.py", "app/ai/llm/base.py", "app/services/llm.py",
		"app/schemas/llm.py", "app/api/routers/llm.py", "tests/test_llm_router.py",
	} {
		files = append(files, gen.FeatureFile{Path: p, Templates: templateFS, Template: "shared/" + p + ".tmpl", Action: gen.ActionShared})
	}
	for _, p := range sel {
		d := Data{Provider: p.name}
		files = append(files,
			gen.FeatureFile{Path: itemFile(p.name), Templates: templateFS, Template: "provider/" + p.name + ".py.tmpl", Data: d},
			gen.FeatureFile{Path: "tests/test_llm_" + p.name + ".py", Templates: templateFS, Template: "provider/test_" + p.name + ".py.tmpl", Data: d})
	}
	files = append(files,
		gen.FeatureFile{Path: "app/api/router.py", Templates: templateFS, Template: "inject/router_import.tmpl", Action: gen.ActionInject, Marker: "router-imports"},
		gen.FeatureFile{Path: "app/api/router.py", Templates: templateFS, Template: "inject/router_include.tmpl", Action: gen.ActionInject, Marker: "routers"})
	seen := map[string]bool{}
	for _, p := range sel {
		if seen[p.sdk] {
			continue
		}
		seen[p.sdk] = true
		files = append(files, gen.FeatureFile{Path: "pyproject.toml", Templates: templateFS, Template: "inject/dependency.tmpl", Data: Data{SDK: p.sdk, Requirement: p.requirement}, Action: gen.ActionInject, Marker: "dependencies"})
	}
	for _, p := range sel {
		lines := make([]string, 0, len(p.vars)+1)
		for _, v := range p.vars {
			lines = append(lines, v+"=")
		}
		if p.optional != "" {
			lines = append(lines, p.optional)
		}
		for _, l := range lines {
			files = append(files, gen.FeatureFile{Path: ".env.example", Templates: templateFS, Template: "inject/env.tmpl", Data: Data{Line: l}, Action: gen.ActionInject, Marker: "env"})
		}
	}
	return files, nil
}

// NextSteps returns the instructions printed after the files are created
// (FR-014).
//
// In: values, the chosen providers (any order).
// Out: "Run uv sync to install the provider SDKs", then, per provider in
// choice order, "Set <VAR>, <VAR> in .env" with its required variables.
// No side effects.
func NextSteps(values []string) []string {
	steps := []string{"Run uv sync to install the provider SDKs"}
	for _, p := range selected(values) {
		steps = append(steps, "Set "+strings.Join(p.vars, ", ")+" in .env")
	}
	return steps
}
