package project

import "context"

// rootKey is the context key under which the detected project root is stored.
type rootKey struct{}

// WithRoot stores the detected project root in a context.
//
// In: ctx, the parent context; root, the project root as given by Check.
// Out: a child context carrying root.
// No side effects.
func WithRoot(ctx context.Context, root string) context.Context {
	return context.WithValue(ctx, rootKey{}, root)
}

// RootFrom reads the project root stored by WithRoot.
//
// In: ctx, a context.
// Out: the stored root and true; or "" and false if ctx carries none
// (e.g. for an exempt command, where the check did not run).
// No side effects.
func RootFrom(ctx context.Context) (string, bool) {
	root, ok := ctx.Value(rootKey{}).(string)
	return root, ok
}
