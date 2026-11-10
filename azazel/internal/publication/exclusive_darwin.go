package publication

import "golang.org/x/sys/unix"

// Install moves the source only if its destination has no entry.
// Unsupported exclusion is a refusal, with no replacing fallback.
func (localFilesystem) Install(request installSpec) error {
	return unix.RenamexNp(request.from, request.to, unix.RENAME_EXCL)
}

// The native adapter supplies the immutable publisher's complete boundary.
var _ immutableFilesystem = localFilesystem{}
