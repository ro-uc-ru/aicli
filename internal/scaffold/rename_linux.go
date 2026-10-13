package scaffold

import (
	"errors"
	"fmt"

	"golang.org/x/sys/unix"
)

// renameNoReplace atomically renames oldPath to newPath, failing instead of
// replacing newPath if it exists, even as an empty directory (renameat2 with
// RENAME_NOREPLACE).
// In: oldPath, the existing source; newPath, the destination.
// Out: nil on success; ErrDestinationExists if newPath exists; an error
// wrapping ErrRenameUnsupported if the filesystem or kernel lacks support;
// otherwise the underlying error with context.
// Side effects: moves oldPath to newPath.
func renameNoReplace(oldPath, newPath string) error {
	err := unix.Renameat2(unix.AT_FDCWD, oldPath, unix.AT_FDCWD, newPath, unix.RENAME_NOREPLACE)
	switch {
	case err == nil:
		return nil
	case errors.Is(err, unix.EEXIST), errors.Is(err, unix.ENOTEMPTY):
		return fmt.Errorf("rename %q to %q: %w", oldPath, newPath, ErrDestinationExists)
	case errors.Is(err, unix.ENOSYS), errors.Is(err, unix.EINVAL), errors.Is(err, unix.ENOTSUP):
		return fmt.Errorf("rename %q to %q: %w: %v", oldPath, newPath, ErrRenameUnsupported, err)
	default:
		return fmt.Errorf("rename %q to %q: %w", oldPath, newPath, err)
	}
}
