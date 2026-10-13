package scaffold

import (
	"errors"
	"fmt"

	"golang.org/x/sys/unix"
)

// renameNoReplace atomically renames oldPath to newPath, failing instead of
// replacing newPath if it exists, even as an empty directory (renamex_np with
// RENAME_EXCL).
// In: oldPath, the existing source; newPath, the destination.
// Out: nil on success; ErrDestinationExists if newPath exists; an error
// wrapping ErrRenameUnsupported if the filesystem lacks support; otherwise
// the underlying error with context.
// Side effects: moves oldPath to newPath.
func renameNoReplace(oldPath, newPath string) error {
	err := unix.RenamexNp(oldPath, newPath, unix.RENAME_EXCL)
	switch {
	case err == nil:
		return nil
	case errors.Is(err, unix.EEXIST), errors.Is(err, unix.ENOTEMPTY):
		return fmt.Errorf("rename %q to %q: %w", oldPath, newPath, ErrDestinationExists)
	case errors.Is(err, unix.ENOTSUP), errors.Is(err, unix.ENOSYS), errors.Is(err, unix.EINVAL):
		return fmt.Errorf("rename %q to %q: %w: %v", oldPath, newPath, ErrRenameUnsupported, err)
	default:
		return fmt.Errorf("rename %q to %q: %w", oldPath, newPath, err)
	}
}
