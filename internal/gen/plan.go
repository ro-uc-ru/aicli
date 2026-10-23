package gen

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
)

// Preview is what a plan would do to a project, as the confirmation shows it
// (FR-011).
type Preview struct {
	// Create holds the paths of the ActionCreate entries and of the
	// ActionShared entries that do not exist, in plan order.
	Create []string
	// Update holds the injection targets whose content would change, once
	// each, in the plan order of their first entry.
	Update []string
}

// CheckPlan checks a plan against a project without writing anything, in the
// order of FR-011: path validation (ValidateEntries), the injection targets
// (FR-010), the items already added (FR-003), and the existing paths of spec
// 004 FR-023 (including the ActionShared paths of EC-028). It then renders the
// injected blocks in memory to tell which targets would change.
//
// In: root, the project root directory; feature, the feature name used in
// messages; plan, the plan to check.
// Out: the Preview, or an error whose message is the plain text after
// "Error: ", with any continuation lines unindented: the error of
// ValidateEntries; "cannot add <feature>, <path> does not exist", "... is not
// a regular file", "... has no \"# aicli:<name>\" marker" or "... has more
// than one \"# aicli:<name>\" marker" for the first failing ActionInject entry;
// "cannot add <feature>, these <items> are already added:" with one
// "<value> (<item file>)" line per added item, in plan.Items order; "cannot
// add <feature>, these files already exist:" with one line per path, in plan
// order; a render error naming the path; or an error naming a path that
// cannot be checked.
// Side effects: none (it stats, resolves links and reads files only).
func CheckPlan(root, feature string, plan Plan) (Preview, error) {
	if err := ValidateEntries(root, feature, plan.Entries); err != nil {
		return Preview{}, err
	}
	realRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		return Preview{}, fmt.Errorf("cannot resolve project root %s: %w", root, err)
	}

	contents := map[string][]byte{}
	for _, e := range plan.Entries {
		if e.Action != ActionInject {
			continue
		}
		src, ok := contents[e.Path]
		if !ok {
			if src, err = readInjectTarget(root, feature, e.Path); err != nil {
				return Preview{}, err
			}
			contents[e.Path] = src
		}
		if _, _, err := Inject(src, e.Marker, nil); err != nil {
			return Preview{}, markerError(feature, e, err)
		}
	}

	var added []string
	for _, it := range plan.Items {
		exists, err := lexists(root, it.Path)
		if err != nil {
			return Preview{}, err
		}
		if exists {
			added = append(added, fmt.Sprintf("%s (%s)", it.Value, it.Path))
		}
	}
	if len(added) > 0 {
		return Preview{}, fmt.Errorf("cannot add %s, these %s are already added:\n%s", feature, plan.ItemNoun, strings.Join(added, "\n"))
	}

	var preview Preview
	var existing []string
	for _, e := range plan.Entries {
		switch e.Action {
		case ActionCreate:
			exists, err := lexists(root, e.Path)
			if err != nil {
				return Preview{}, err
			}
			if exists {
				existing = append(existing, e.Path)
			} else {
				preview.Create = append(preview.Create, e.Path)
			}
		case ActionShared:
			state, err := sharedState(realRoot, e.Path)
			if err != nil {
				return Preview{}, err
			}
			switch state {
			case sharedMissing:
				preview.Create = append(preview.Create, e.Path)
			case sharedInvalid:
				existing = append(existing, e.Path)
			}
		}
	}
	if len(existing) > 0 {
		return Preview{}, fmt.Errorf("cannot add %s, these files already exist:\n%s", feature, strings.Join(existing, "\n"))
	}

	changed, err := injectAll(plan.Entries, contents)
	if err != nil {
		return Preview{}, err
	}
	for _, p := range injectTargets(plan.Entries) {
		if changed[p] != nil {
			preview.Update = append(preview.Update, p)
		}
	}
	return preview, nil
}

// readInjectTarget reads an injection target, following symbolic links, and
// checks that it is an existing regular file (FR-010).
//
// In: root, the project root; feature, the feature name used in messages; p,
// the "/" path of the target relative to root.
// Out: the content, or "cannot add <feature>, <p> does not exist" (also for a
// broken link), "cannot add <feature>, <p> is not a regular file", or an
// error naming p when it cannot be read.
// Side effects: none (Stat and ReadFile only).
func readInjectTarget(root, feature, p string) ([]byte, error) {
	full := filepath.Join(root, filepath.FromSlash(p))
	info, err := os.Stat(full)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("cannot add %s, %s does not exist", feature, p)
		}
		return nil, fmt.Errorf("cannot check %q: %w", p, err)
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("cannot add %s, %s is not a regular file", feature, p)
	}
	src, err := os.ReadFile(full)
	if err != nil {
		return nil, fmt.Errorf("cannot read %q: %w", p, err)
	}
	return src, nil
}

// markerError turns a marker error of Inject into the message of FR-010.
//
// In: feature, the feature name; e, the ActionInject entry; err, the error
// returned by Inject.
// Out: the FR-010 error for ErrNoMarker or ErrRepeatedMarker; err otherwise.
// Side effects: none.
func markerError(feature string, e FeatureFile, err error) error {
	switch {
	case errors.Is(err, ErrNoMarker):
		return fmt.Errorf("cannot add %s, %s has no \"# aicli:%s\" marker", feature, e.Path, e.Marker)
	case errors.Is(err, ErrRepeatedMarker):
		return fmt.Errorf("cannot add %s, %s has more than one \"# aicli:%s\" marker", feature, e.Path, e.Marker)
	}
	return err
}

// lexists reports whether a path exists, of any kind, without following a
// final symbolic link.
//
// In: root, the project root; p, the "/" path relative to root.
// Out: true if it exists; an error naming p when it cannot be checked.
// Side effects: none (Lstat only).
func lexists(root, p string) (bool, error) {
	_, err := os.Lstat(filepath.Join(root, filepath.FromSlash(p)))
	switch {
	case err == nil:
		return true, nil
	case os.IsNotExist(err):
		return false, nil
	}
	return false, fmt.Errorf("cannot check %q: %w", p, err)
}

// sharedKind is the state of the path of an ActionShared entry.
type sharedKind int

const (
	// sharedMissing means the path does not exist, so it is created.
	sharedMissing sharedKind = iota
	// sharedPresent means it is a regular file, or a symbolic link to one
	// inside the root, so it is left untouched.
	sharedPresent
	// sharedInvalid means it exists as anything else (EC-028).
	sharedInvalid
)

// sharedState classifies the path of an ActionShared entry (FR-008, EC-028).
//
// In: realRoot, the project root after filepath.EvalSymlinks; p, the "/"
// path relative to the root.
// Out: its sharedKind, or an error naming p when it cannot be checked.
// Side effects: none (Lstat, EvalSymlinks and Stat only).
func sharedState(realRoot, p string) (sharedKind, error) {
	full := filepath.Join(realRoot, filepath.FromSlash(p))
	info, err := os.Lstat(full)
	if err != nil {
		if os.IsNotExist(err) {
			return sharedMissing, nil
		}
		return 0, fmt.Errorf("cannot check %q: %w", p, err)
	}
	if info.Mode().IsRegular() {
		return sharedPresent, nil
	}
	if info.Mode()&os.ModeSymlink == 0 {
		return sharedInvalid, nil
	}
	resolved, err := filepath.EvalSymlinks(full)
	if err != nil || !within(realRoot, resolved) {
		return sharedInvalid, nil
	}
	if info, err = os.Stat(resolved); err != nil || !info.Mode().IsRegular() {
		return sharedInvalid, nil
	}
	return sharedPresent, nil
}

// injectTargets lists the distinct ActionInject targets of a plan.
//
// In: entries, the plan entries in plan order.
// Out: each target path once, in the order of its first entry.
// Side effects: none.
func injectTargets(entries []FeatureFile) []string {
	var out []string
	seen := map[string]bool{}
	for _, e := range entries {
		if e.Action == ActionInject && !seen[e.Path] {
			seen[e.Path] = true
			out = append(out, e.Path)
		}
	}
	return out
}

// injectAll renders the ActionInject entries and applies them in memory. The
// blocks for one marker of one target are inserted together, in plan order,
// and the markers of a target are applied in the order of their first entry.
//
// In: entries, the plan entries in plan order; contents, the original content
// of every injection target, keyed by path (not modified).
// Out: the new content of each target that changes, keyed by path; targets
// that do not change are absent. Or a render error naming the path, or the
// error of Inject.
// Side effects: none.
func injectAll(entries []FeatureFile, contents map[string][]byte) (map[string][]byte, error) {
	type group struct {
		path, marker string
		blocks       []string
	}
	var groups []*group
	index := map[[2]string]*group{}
	for _, e := range entries {
		if e.Action != ActionInject {
			continue
		}
		b, err := renderFeatureFile(e)
		if err != nil {
			return nil, err
		}
		key := [2]string{e.Path, e.Marker}
		g := index[key]
		if g == nil {
			g = &group{path: e.Path, marker: e.Marker}
			index[key] = g
			groups = append(groups, g)
		}
		g.blocks = append(g.blocks, string(b))
	}
	changed := map[string][]byte{}
	for _, g := range groups {
		cur, ok := changed[g.path]
		if !ok {
			cur = contents[g.path]
		}
		out, did, err := Inject(cur, g.marker, g.blocks)
		if err != nil {
			return nil, err
		}
		if did {
			changed[g.path] = out
		}
	}
	return changed, nil
}

// Verb is the action reported for one path of a successful run (FR-011).
type Verb string

const (
	// VerbCreate reports a created ActionCreate or ActionShared file.
	VerbCreate Verb = "create"
	// VerbSkip reports an ActionShared file that already existed, or an
	// injection target where no entry inserted anything.
	VerbSkip Verb = "skip"
	// VerbUpdate reports an injection target that was modified.
	VerbUpdate Verb = "update"
)

// ReportLine is one line of the report of a successful run.
type ReportLine struct {
	// Verb is what happened to the path.
	Verb Verb
	// Path is the path relative to the project root, with "/" separators.
	Path string
}

// ApplyPlan applies a plan to an existing project, all or nothing (FR-008,
// FR-009, FR-011, SEC-5). It runs CheckPlan again, renders every entry in
// memory, then, in plan order, creates the ActionCreate entries and the
// missing ActionShared entries (never replacing a file), skips the existing
// ActionShared entries, and writes each changed injection target once, in
// place at the path its symbolic links resolve to, so a link stays a link
// and the file keeps its mode. On any failure or cancellation it rolls back
// in reverse order: injection targets get back their original bytes and mode,
// and the paths this run created are removed (EC-013).
//
// In: ctx, the run context, checked before each step; fsys, the filesystem
// to write through; root, the project root; feature, the feature name used in
// messages; plan, the plan to apply.
// Out: one ReportLine per path in plan order (an injection target once, at its
// first entry), or an error: the one of CheckPlan; a render error naming the
// path; an error naming an injection target that cannot be read or written; a
// create/write error with the continuation lines of rollbackPlan; or, when
// ctx is cancelled, context.Cause(ctx) once the rollback is complete.
// Side effects: creates directories (DirMode) and files (FileMode) under
// root and rewrites injection targets; on failure restores the targets and
// removes the created paths again.
func ApplyPlan(ctx context.Context, fsys FeatureFS, root, feature string, plan Plan) ([]ReportLine, error) {
	if _, err := CheckPlan(root, feature, plan); err != nil {
		return nil, err
	}
	realRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		return nil, fmt.Errorf("cannot resolve project root %s: %w", root, err)
	}

	rendered := make([][]byte, len(plan.Entries))
	resolved := map[string]string{}
	contents := map[string][]byte{}
	for i, e := range plan.Entries {
		if e.Action != ActionInject {
			b, err := renderFeatureFile(e)
			if err != nil {
				return nil, err
			}
			rendered[i] = b
			continue
		}
		if _, ok := resolved[e.Path]; ok {
			continue
		}
		target, err := filepath.EvalSymlinks(filepath.Join(root, filepath.FromSlash(e.Path)))
		if err != nil {
			return nil, fmt.Errorf("cannot resolve %q: %w", e.Path, err)
		}
		src, err := fsys.ReadFile(target)
		if err != nil {
			return nil, fmt.Errorf("cannot read %q: %w", e.Path, err)
		}
		resolved[e.Path] = target
		contents[e.Path] = src
	}
	changed, err := injectAll(plan.Entries, contents)
	if err != nil {
		return nil, err
	}

	var report []ReportLine
	var journal []planStep
	reported := map[string]bool{}
	for i, e := range plan.Entries {
		if reported[e.Path] {
			continue
		}
		reported[e.Path] = true
		if ctx.Err() != nil {
			return nil, rollbackPlan(ctx, fsys, journal, context.Cause(ctx))
		}
		verb := VerbCreate
		switch e.Action {
		case ActionShared:
			state, err := sharedState(realRoot, e.Path)
			if err != nil {
				return nil, rollbackPlan(ctx, fsys, journal, err)
			}
			if state != sharedMissing {
				verb = VerbSkip
				break
			}
			fallthrough
		case ActionCreate:
			var created []createdPath
			err := createOne(ctx, fsys, root, e.Path, rendered[i], &created)
			for _, c := range created {
				journal = append(journal, planStep{path: c.path, rel: c.rel})
			}
			if err != nil {
				return nil, rollbackPlan(ctx, fsys, journal, err)
			}
		case ActionInject:
			out, ok := changed[e.Path]
			if !ok {
				verb = VerbSkip
				break
			}
			target := resolved[e.Path]
			info, err := fsys.Stat(target)
			if err != nil {
				return nil, rollbackPlan(ctx, fsys, journal, fmt.Errorf("cannot check %q: %w", e.Path, err))
			}
			if !info.Mode().IsRegular() {
				return nil, rollbackPlan(ctx, fsys, journal, fmt.Errorf("cannot update %q: not a regular file", e.Path))
			}
			journal = append(journal, planStep{path: target, rel: e.Path, restore: true, orig: contents[e.Path], mode: info.Mode() & modeBits})
			if err := rewriteFile(fsys, target, e.Path, out); err != nil {
				return nil, rollbackPlan(ctx, fsys, journal, err)
			}
			verb = VerbUpdate
		}
		report = append(report, ReportLine{Verb: verb, Path: e.Path})
	}
	return report, nil
}

// modeBits are the mode bits that a restored injection target gets back.
const modeBits = os.ModePerm | os.ModeSetuid | os.ModeSetgid | os.ModeSticky

// planStep is one change an ApplyPlan run made, recorded as soon as it starts
// so that the rollback can undo it.
type planStep struct {
	// path is the full path on the filesystem (resolved for an injection).
	path string
	// rel is the path relative to the project root, with "/" separators.
	rel string
	// restore is true for an injection target, false for a created path.
	restore bool
	// orig is the original content of an injection target.
	orig []byte
	// mode is the original mode of an injection target.
	mode os.FileMode
}

// rollbackPlan undoes the steps of a failed ApplyPlan run in reverse order:
// it writes every injection target back with its original bytes and mode,
// and removes every created file and directory (a directory only when empty).
// It always completes, even when ctx is cancelled (EC-013).
//
// In: ctx, the run context; fsys, the filesystem; journal, the steps in the
// order they started; cause, the error that stopped the run.
// Out: context.Cause(ctx) if ctx is cancelled; otherwise cause, with the
// continuation lines "could not remove:" and "could not restore:", each
// followed by one line per path (indented by two spaces) in step order, when
// some path could not be removed or restored.
// Side effects: rewrites injection targets, removes files and empty
// directories.
func rollbackPlan(ctx context.Context, fsys FeatureFS, journal []planStep, cause error) error {
	var left, unrestored []string
	for i := len(journal) - 1; i >= 0; i-- {
		st := journal[i]
		if st.restore {
			if err := restoreFile(fsys, st); err != nil {
				unrestored = append([]string{st.rel}, unrestored...)
			}
			continue
		}
		err := fsys.Remove(st.path)
		if err == nil || errors.Is(err, syscall.ENOTEMPTY) || errors.Is(err, syscall.EEXIST) {
			continue
		}
		left = append([]string{st.rel}, left...)
	}
	if ctx.Err() != nil {
		return context.Cause(ctx)
	}
	msg := ""
	if len(left) > 0 {
		msg += "\ncould not remove:\n  " + strings.Join(left, "\n  ")
	}
	if len(unrestored) > 0 {
		msg += "\ncould not restore:\n  " + strings.Join(unrestored, "\n  ")
	}
	if msg == "" {
		return cause
	}
	return fmt.Errorf("%w%s", cause, msg)
}

// restoreFile writes an injection target back with its original content and
// mode.
//
// In: fsys, the filesystem; st, the recorded injection step.
// Out: nil, or the first error.
// Side effects: rewrites st.path and sets its mode.
func restoreFile(fsys FeatureFS, st planStep) error {
	if err := rewriteFile(fsys, st.path, st.rel, st.orig); err != nil {
		return err
	}
	return fsys.Chmod(st.path, st.mode)
}

// rewriteFile replaces the content of an existing file in place, which keeps
// its mode, its inode and any symbolic link pointing to it.
//
// In: fsys, the filesystem; target, the resolved full path; rel, the "/" path
// for messages; content, the new bytes.
// Out: nil, or an error naming rel.
// Side effects: truncates and rewrites target.
func rewriteFile(fsys FeatureFS, target, rel string, content []byte) (err error) {
	f, err := fsys.OpenFile(target, os.O_WRONLY|os.O_TRUNC, 0)
	if err != nil {
		return fmt.Errorf("update file %s: %w", rel, err)
	}
	defer func() {
		if cerr := f.Close(); err == nil && cerr != nil {
			err = fmt.Errorf("close file %s: %w", rel, cerr)
		}
	}()
	if _, err = f.Write(content); err != nil {
		return fmt.Errorf("update file %s: %w", rel, err)
	}
	return nil
}
