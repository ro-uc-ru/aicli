// Package gen renders the base project skeleton from the embedded templates.
// It is pure: it never touches the disk.
package gen

import (
	"bytes"
	"fmt"
	"io/fs"
	"path"
	"sort"
	"strings"
	"text/template"

	"github.com/ro-uc-ru/aicli/templates"
)

// root is the directory of the embedded filesystem that holds the skeleton.
const root = "skeleton"

// tmplSuffix marks the templates rendered with the project name.
const tmplSuffix = ".tmpl"

// Kind tells whether an Entry is a directory or a regular file.
type Kind int

const (
	// KindDir is a directory entry; its Content is nil.
	KindDir Kind = iota
	// KindFile is a regular file entry.
	KindFile
)

// Entry is one directory or file of the generated project.
type Entry struct {
	// Path is the slash-separated path relative to the project root.
	Path string
	// Content is the file content; nil for directories.
	Content []byte
	// Kind is KindDir or KindFile.
	Kind Kind
}

// data is the value passed to every template.
type data struct {
	// Name is the project name.
	Name string
}

// Render builds the entries of the skeleton for a project.
//
// In: name, the already validated project name.
// Out: the entries sorted by path (a directory always precedes its children),
// with the .tmpl suffix stripped and the templates rendered with name; an
// error with context if an embedded file cannot be read or rendered.
// Side effects: none.
func Render(name string) ([]Entry, error) {
	return render(templates.Skeleton, name)
}

// render is Render over an arbitrary filesystem laid out like templates.Skeleton.
//
// In: fsys, the filesystem holding the "skeleton" directory; name, the project name.
// Out: the sorted entries, or an error with context.
// Side effects: none.
func render(fsys fs.FS, name string) ([]Entry, error) {
	var entries []Entry
	err := fs.WalkDir(fsys, root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if p == root {
			return nil
		}
		rel := p[len(root)+1:] // p is inside root; embed paths use "/"
		if d.IsDir() {
			entries = append(entries, Entry{Path: rel, Kind: KindDir})
			return nil
		}
		raw, err := fs.ReadFile(fsys, p)
		if err != nil {
			return fmt.Errorf("read template %s: %w", p, err)
		}
		if path.Ext(rel) == tmplSuffix {
			raw, err = execute(rel, raw, name)
			if err != nil {
				return err
			}
			rel = strings.TrimSuffix(rel, tmplSuffix)
		}
		entries = append(entries, Entry{Path: rel, Content: raw, Kind: KindFile})
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("render skeleton: %w", err)
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Path < entries[j].Path })
	return entries, nil
}

// execute renders one template with the project name.
//
// In: rel, the template path (for error messages); src, its source; name, the project name.
// Out: the rendered bytes, or an error with context if parsing or execution fails.
// Side effects: none.
func execute(rel string, src []byte, name string) ([]byte, error) {
	t, err := template.New(rel).Option("missingkey=error").Parse(string(src))
	if err != nil {
		return nil, fmt.Errorf("parse template %s: %w", rel, err)
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, data{Name: name}); err != nil {
		return nil, fmt.Errorf("execute template %s: %w", rel, err)
	}
	return buf.Bytes(), nil
}
