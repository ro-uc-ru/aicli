package templates

import "flag"

// _ registers the -update flag so that `go test ./... -update` is accepted by
// this package, which has no golden files of its own.
var _ = flag.Bool("update", false, "regenerate golden files (unused here)")
