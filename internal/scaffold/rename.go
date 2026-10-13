package scaffold

import "errors"

// ErrDestinationExists is returned by renameNoReplace when the destination
// already exists; the destination is left untouched (EC-010, FR-014).
var ErrDestinationExists = errors.New("destination already exists")

// ErrRenameUnsupported is returned by renameNoReplace when the filesystem or
// kernel does not support an atomic no-replace rename. There is no fallback.
var ErrRenameUnsupported = errors.New("atomic no-replace rename not supported")
