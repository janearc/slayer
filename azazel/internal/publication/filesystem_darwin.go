package publication

import (
	"fmt"
	"os"
	"path/filepath"
)

// localFilesystem delegates operations on explicitly selected local paths.
type localFilesystem struct{}

// CreateTemp transfers one same-parent file and name, or only an error.
func (localFilesystem) CreateTemp(request tempSpec) openedTemp {
	if !filepath.IsAbs(request.parent) ||
		request.parent != filepath.Clean(request.parent) {
		return openedTemp{err: fmt.Errorf(
			"temporary parent needs a clean absolute path: %w",
			os.ErrInvalid,
		)}
	}
	file, err := os.CreateTemp(request.parent, request.pattern)
	if err != nil {
		return openedTemp{err: err}
	}
	return openedTemp{file: localTemporary{file: file}, name: file.Name()}
}

// Rename delegates the exact entry replacement and retains its error.
func (localFilesystem) Rename(request renameSpec) error {
	return os.Rename(request.from, request.to)
}

// Remove delegates one named entry's removal and retains its error.
func (localFilesystem) Remove(request pathSpec) error {
	return os.Remove(request.path)
}

// The native adapter supplies the publisher's complete filesystem boundary.
var _ filesystem = localFilesystem{}
