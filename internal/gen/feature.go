package gen

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

// FeatureFile is one file that a feature creates in an existing project
// through the shared feature-file mechanism (FR-022..FR-025).
type FeatureFile struct {
	// Path is the target, relative to the project root, with "/" as separator.
	Path string
	// Templates is the filesystem that holds the template (embedded, ARC-5).
	Templates fs.FS
	// Template is the template name inside Templates.
	Template string
	// Data is the value passed to the template; it holds at least the input values.
	Data any
	// Action is what the entry does to Path; the zero value is ActionCreate.
	Action FileAction
	// Marker is the marker name, without the "# aicli:" prefix, after which an
	// ActionInject entry inserts its lines. It is unused by the other actions.
	Marker string
}

// FileAction is the action of one plan entry on its path (FR-008).
type FileAction int

const (
	// ActionCreate creates a new file and fails if the path exists (spec 004).
	ActionCreate FileAction = iota
	// ActionShared creates the file if it is missing and leaves an existing
	// regular file untouched.
	ActionShared
	// ActionInject inserts the rendered lines after a marker of an existing file.
	ActionInject
)

// Item is one value of a multi-choice input together with the item file that
// records whether that value has already been added.
type Item struct {
	// Value is the choice, as declared by the feature.
	Value string
	// Path is the item file, relative to the project root, with "/" separators.
	Path string
}

// Plan is everything a feature run does to a project: its entries in plan
// order, plus the items of the given values used to detect values already added.
type Plan struct {
	// Entries are the file actions, in plan order.
	Entries []FeatureFile
	// ItemNoun is the plural noun naming the items in messages (e.g. "providers");
	// empty when the feature has no multi-choice input.
	ItemNoun string
	// Items are the item files of the given values, in choice order.
	Items []Item
}

// ValidateFeatureFiles checks every target path before anything is written
// (FR-022, EC-021, EC-022, SEC-3). It reads the filesystem but never writes.
//
// In: root, the project root directory; feature, the feature name used in
// messages; paths, the target paths relative to root, in plan order.
// Out: nil if every path is valid; otherwise an error whose message is the
// plain text after "Error: " and names the offending path. An empty list
// gives "<feature> declares no files".
// Side effects: none (it only resolves symbolic links and stats ancestors).
func ValidateFeatureFiles(root, feature string, paths []string) error {
	return ValidateEntries(root, feature, createEntries(paths))
}

// ValidateEntries checks the path of every plan entry before anything is
// written, as ValidateFeatureFiles does, with the file actions of FR-008: the
// same path may appear in several entries only if all of them are
// ActionInject, and an ActionInject target that is a symbolic link must
// resolve inside the project root (EC-014).
//
// In: root, the project root directory; feature, the feature name used in
// messages; entries, the plan entries in plan order.
// Out: nil if every path is valid; otherwise an error whose message is the
// plain text after "Error: " and names the offending path. An empty list
// gives "<feature> declares no files".
// Side effects: none (it only resolves symbolic links and stats paths).
func ValidateEntries(root, feature string, entries []FeatureFile) error {
	if len(entries) == 0 {
		return fmt.Errorf("%s declares no files", feature)
	}
	realRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		return fmt.Errorf("cannot resolve project root %s: %w", root, err)
	}
	seen := make(map[string]FileAction, len(entries))
	for _, e := range entries {
		p := e.Path
		if err := checkSyntax(p); err != nil {
			return err
		}
		if prev, ok := seen[p]; ok {
			if prev != ActionInject || e.Action != ActionInject {
				return fmt.Errorf("invalid path %q: listed more than once", p)
			}
			continue
		}
		seen[p] = e.Action
		if err := checkAncestors(realRoot, p); err != nil {
			return err
		}
		if e.Action == ActionInject {
			if err := checkInjectTarget(realRoot, p); err != nil {
				return err
			}
		}
	}
	return nil
}

// checkInjectTarget checks that an injection target reached through a
// symbolic link resolves inside the project root (EC-014). A missing target,
// a broken link and a target that is not a link pass: whether the target
// exists and is a regular file is checked later (FR-010).
//
// In: realRoot, the project root after filepath.EvalSymlinks; p, a path that
// passed checkSyntax and checkAncestors.
// Out: nil, or an error naming p when the link cannot be resolved or leads
// outside the root.
// Side effects: none (Lstat and EvalSymlinks only).
func checkInjectTarget(realRoot, p string) error {
	target := filepath.Join(realRoot, filepath.FromSlash(p))
	info, err := os.Lstat(target)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("invalid path %q: %w", p, err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		return nil
	}
	resolved, err := filepath.EvalSymlinks(target)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("invalid path %q: cannot resolve: %w", p, err)
	}
	if !within(realRoot, resolved) {
		return fmt.Errorf("invalid path %q: resolves outside the project", p)
	}
	return nil
}

// checkSyntax applies the purely lexical rules of FR-022 to one path.
//
// In: p, a target path relative to the project root.
// Out: nil, or an error naming p and the broken rule.
// Side effects: none.
func checkSyntax(p string) error {
	switch {
	case p == "":
		return fmt.Errorf("invalid path %q: empty", p)
	case strings.Contains(p, `\`):
		return fmt.Errorf("invalid path %q: must use '/' as separator", p)
	case strings.HasPrefix(p, "/"):
		return fmt.Errorf("invalid path %q: must be relative", p)
	}
	for _, c := range strings.Split(p, "/") {
		switch c {
		case "":
			return fmt.Errorf("invalid path %q: empty component", p)
		case ".", "..":
			return fmt.Errorf("invalid path %q: component %q not allowed", p, c)
		}
	}
	return nil
}

// checkAncestors walks the parent directories of p under the resolved root,
// following symbolic links, and checks that every existing one is a directory
// inside the root (EC-022). The walk stops at the first missing component.
//
// In: realRoot, the project root after filepath.EvalSymlinks; p, a path that
// passed checkSyntax.
// Out: nil, or an error naming p when an existing ancestor is not a
// directory, cannot be resolved, or leads outside the root.
// Side effects: none (Lstat, EvalSymlinks and Stat only).
func checkAncestors(realRoot, p string) error {
	comps := strings.Split(p, "/")
	cur := realRoot
	for _, c := range comps[:len(comps)-1] {
		next := filepath.Join(cur, c)
		if _, err := os.Lstat(next); err != nil {
			if os.IsNotExist(err) {
				return nil
			}
			return fmt.Errorf("invalid path %q: %w", p, err)
		}
		resolved, err := filepath.EvalSymlinks(next)
		if err != nil {
			return fmt.Errorf("invalid path %q: cannot resolve %s: %w", p, c, err)
		}
		if !within(realRoot, resolved) {
			return fmt.Errorf("invalid path %q: resolves outside the project", p)
		}
		info, err := os.Stat(resolved)
		if err != nil {
			return fmt.Errorf("invalid path %q: %w", p, err)
		}
		if !info.IsDir() {
			return fmt.Errorf("invalid path %q: %s is not a directory", p, c)
		}
		cur = resolved
	}
	return nil
}

// within reports whether target is root itself or lies below it.
//
// In: root and target, both absolute and symlink-free.
// Out: true if target is inside root.
// Side effects: none.
func within(root, target string) bool {
	rel, err := filepath.Rel(root, target)
	if err != nil {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

// CheckFeatureFiles validates the target paths and checks that none of them
// exists yet, without writing anything (FR-022, FR-023, SEC-5). It is
// CheckPlan on a plan of ActionCreate entries only.
//
// In: root, the project root directory; feature, the feature name used in
// messages; paths, the target paths relative to root, in plan order.
// Out: nil if every path is valid and free; otherwise the error of
// ValidateFeatureFiles, or an error whose message is
// "cannot add <feature>, these files already exist:" followed by one line per
// existing path, indented by two spaces, in list order.
// Side effects: none (Lstat only).
func CheckFeatureFiles(root, feature string, paths []string) error {
	_, err := CheckPlan(root, feature, Plan{Entries: createEntries(paths)})
	return err
}

// createEntries turns target paths into ActionCreate entries with no template.
//
// In: paths, the target paths in plan order.
// Out: one entry per path, in the same order.
// Side effects: none.
func createEntries(paths []string) []FeatureFile {
	entries := make([]FeatureFile, len(paths))
	for i, p := range paths {
		entries[i] = FeatureFile{Path: p}
	}
	return entries
}

// FeatureFS is the filesystem interface of CreateFeatureFiles and ApplyPlan,
// injectable so tests can force failures or race the existence check.
type FeatureFS interface {
	// Lstat describes a path without following a final symbolic link.
	// In: name, the path. Out: its info, or an error (os.IsNotExist if absent).
	// Side effects: none.
	Lstat(name string) (os.FileInfo, error)
	// Mkdir creates one directory.
	// In: name, the path; perm, the requested mode.
	// Out: an error if it cannot be created or already exists.
	// Side effects: creates the directory.
	Mkdir(name string, perm os.FileMode) error
	// OpenFile opens a file; with os.O_EXCL it fails if the path exists.
	// In: name, the path; flag, the os.O_* flags; perm, the requested mode.
	// Out: a writable handle, or an error.
	// Side effects: may create the file.
	OpenFile(name string, flag int, perm os.FileMode) (io.WriteCloser, error)
	// Chmod sets the mode of a path exactly, regardless of the umask.
	// In: name, the path; mode, the mode. Out: an error on failure.
	// Side effects: changes the permissions of name.
	Chmod(name string, mode os.FileMode) error
	// Remove deletes a file or an empty directory.
	// In: name, the path. Out: an error on failure.
	// Side effects: deletes name.
	Remove(name string) error
	// ReadFile reads a whole file, following symbolic links.
	// In: name, the path. Out: its content, or an error.
	// Side effects: none.
	ReadFile(name string) ([]byte, error)
	// Stat describes a path, following symbolic links.
	// In: name, the path. Out: its info, or an error (os.IsNotExist if absent).
	// Side effects: none.
	Stat(name string) (os.FileInfo, error)
}

// OSFeatureFS is the FeatureFS backed by the real filesystem.
type OSFeatureFS struct{}

// Lstat implements FeatureFS with os.Lstat.
// In: name, the path. Out: the os.Lstat result. Side effects: none.
func (OSFeatureFS) Lstat(name string) (os.FileInfo, error) { return os.Lstat(name) }

// Mkdir implements FeatureFS with os.Mkdir.
// In: name, the path; perm, the mode. Out: the os.Mkdir error.
// Side effects: creates the directory.
func (OSFeatureFS) Mkdir(name string, perm os.FileMode) error { return os.Mkdir(name, perm) }

// OpenFile implements FeatureFS with os.OpenFile.
// In: name, the path; flag, the os.O_* flags; perm, the mode.
// Out: the handle, or the os.OpenFile error. Side effects: may create the file.
func (OSFeatureFS) OpenFile(name string, flag int, perm os.FileMode) (io.WriteCloser, error) {
	return os.OpenFile(name, flag, perm)
}

// Chmod implements FeatureFS with os.Chmod.
// In: name, the path; mode, the mode. Out: the os.Chmod error.
// Side effects: changes the permissions of name.
func (OSFeatureFS) Chmod(name string, mode os.FileMode) error { return os.Chmod(name, mode) }

// Remove implements FeatureFS with os.Remove.
// In: name, the path. Out: the os.Remove error. Side effects: deletes name.
func (OSFeatureFS) Remove(name string) error { return os.Remove(name) }

// ReadFile implements FeatureFS with os.ReadFile.
// In: name, the path. Out: the os.ReadFile result. Side effects: none.
func (OSFeatureFS) ReadFile(name string) ([]byte, error) { return os.ReadFile(name) }

// Stat implements FeatureFS with os.Stat.
// In: name, the path. Out: the os.Stat result. Side effects: none.
func (OSFeatureFS) Stat(name string) (os.FileInfo, error) { return os.Stat(name) }

// createdPath is one path that createOne created.
type createdPath struct {
	// path is the full path on the filesystem.
	path string
	// rel is the path relative to the project root, with "/" separators.
	rel string
}

// CreateFeatureFiles creates the files of a feature in an existing project,
// all or nothing (FR-024, EC-023, SEC-5): it checks them, renders all of them
// in memory, then creates the missing parent directories and each file, never
// replacing one. On a create or write failure, or on cancellation, it removes
// what this run created. It is ApplyPlan on a plan of ActionCreate entries
// only; any Action or Marker of files is ignored.
//
// In: ctx, the run context, checked before each creation; fsys, the
// filesystem to write through; root, the project root; feature, the feature
// name used in messages; files, the feature files in plan order.
// Out: the created file paths relative to root, in list order, or an error:
// the one of CheckFeatureFiles; a render error naming the path and the cause;
// a create/write error naming the path, followed, if the rollback left paths
// behind, by the continuation lines "  could not remove:" and one line per
// path (indented by four spaces) in creation order; or, when ctx is
// cancelled, context.Cause(ctx) once the rollback is complete.
// Side effects: creates directories (DirMode) and files (FileMode) under
// root; on failure removes them again in reverse order, directories only
// when empty, and nothing else.
func CreateFeatureFiles(ctx context.Context, fsys FeatureFS, root, feature string, files []FeatureFile) ([]string, error) {
	entries := make([]FeatureFile, len(files))
	for i, f := range files {
		f.Action, f.Marker = ActionCreate, ""
		entries[i] = f
	}
	report, err := ApplyPlan(ctx, fsys, root, feature, Plan{Entries: entries})
	if err != nil {
		return nil, err
	}
	paths := make([]string, len(report))
	for i, l := range report {
		paths[i] = l.Path
	}
	return paths, nil
}

// createOne creates the missing parents and then the file of one feature file.
//
// In: ctx, checked before each creation; fsys, the filesystem; root, the
// project root; rel, the "/" path of the file; content, the bytes to write;
// created, the list that records every path created, appended in place.
// Out: nil, or the context cause or an error naming the path.
// Side effects: creates directories and a file, recording each as soon as it
// exists.
func createOne(ctx context.Context, fsys FeatureFS, root, rel string, content []byte, created *[]createdPath) error {
	comps := strings.Split(rel, "/")
	cur := root
	for i, c := range comps[:len(comps)-1] {
		cur = filepath.Join(cur, c)
		if _, err := fsys.Lstat(cur); err == nil {
			continue
		} else if !os.IsNotExist(err) {
			return fmt.Errorf("check directory %s: %w", cur, err)
		}
		if ctx.Err() != nil {
			return context.Cause(ctx)
		}
		if err := fsys.Mkdir(cur, DirMode); err != nil {
			return fmt.Errorf("create directory %s: %w", cur, err)
		}
		*created = append(*created, createdPath{path: cur, rel: strings.Join(comps[:i+1], "/")})
		if err := fsys.Chmod(cur, DirMode); err != nil {
			return fmt.Errorf("set mode of directory %s: %w", cur, err)
		}
	}
	if ctx.Err() != nil {
		return context.Cause(ctx)
	}
	return writeFeatureFile(fsys, filepath.Join(root, filepath.FromSlash(rel)), rel, content, created)
}

// RenderEntry renders one plan entry in memory, as ApplyPlan does.
//
// In: f, the entry. Out: the rendered bytes, or an error naming f.Path and
// the cause. Side effects: none.
func RenderEntry(f FeatureFile) ([]byte, error) { return renderFeatureFile(f) }

// renderFeatureFile renders one feature file in memory.
//
// In: f, the feature file. Out: the rendered bytes, or an error naming
// f.Path and the cause. Side effects: none.
func renderFeatureFile(f FeatureFile) ([]byte, error) {
	src, err := fs.ReadFile(f.Templates, f.Template)
	if err != nil {
		return nil, fmt.Errorf("render %s: %w", f.Path, err)
	}
	t, err := template.New(f.Template).Option("missingkey=error").Parse(string(src))
	if err != nil {
		return nil, fmt.Errorf("render %s: %w", f.Path, err)
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, f.Data); err != nil {
		return nil, fmt.Errorf("render %s: %w", f.Path, err)
	}
	return buf.Bytes(), nil
}

// writeFeatureFile creates one file only if it does not exist.
//
// In: fsys, the filesystem; target, the full path; rel, the "/" path for
// messages; content, the bytes to write; created, the list that records the
// file once it exists, appended in place.
// Out: nil, or an error naming rel. Side effects: creates the file with
// O_EXCL, sets FileMode exactly and records it in created.
func writeFeatureFile(fsys FeatureFS, target, rel string, content []byte, created *[]createdPath) (err error) {
	f, err := fsys.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_EXCL, FileMode)
	if err != nil {
		return fmt.Errorf("create file %s: %w", rel, err)
	}
	*created = append(*created, createdPath{path: target, rel: rel})
	defer func() {
		if cerr := f.Close(); err == nil && cerr != nil {
			err = fmt.Errorf("close file %s: %w", rel, cerr)
		}
	}()
	if _, err = f.Write(content); err != nil {
		return fmt.Errorf("write file %s: %w", rel, err)
	}
	if err = fsys.Chmod(target, FileMode); err != nil {
		return fmt.Errorf("set mode of file %s: %w", rel, err)
	}
	return nil
}
