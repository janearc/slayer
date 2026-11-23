package preview

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

const sourceManifestLimit = 8 * 1024 * 1024

// CopySourceManifest preserves the nearest original sources.yaml when a caller
// copies an input into a new private directory. It copies metadata only; source
// datasets continue to belong to the separately configured external shelf.
// An existing destination must contain identical bytes and is never overwritten.
func CopySourceManifest(sourceFile, destinationDirectory string) error {
	absolute, err := filepath.Abs(sourceFile)
	if err != nil {
		return err
	}
	top := manifestSearchTop(filepath.Dir(absolute))
	for directory := filepath.Dir(absolute); ; directory = filepath.Dir(directory) {
		manifest := filepath.Join(directory, "sources.yaml")
		info, err := os.Lstat(manifest)
		if err == nil {
			if !info.Mode().IsRegular() {
				return fmt.Errorf("source manifest %q must be a regular file", manifest)
			}
			raw, err := readSourceManifest(manifest)
			if err != nil {
				return err
			}
			if err := retainSubmissionFile(filepath.Join(destinationDirectory, "sources.yaml"), raw); err != nil {
				return fmt.Errorf("preserve source manifest: %w", err)
			}
			return nil
		}
		if !os.IsNotExist(err) {
			return err
		}
		if parent := filepath.Dir(directory); parent == directory || directory == top {
			return nil
		}
	}
}

// manifestSearchTop is the highest folder whose sources.yaml may belong to a
// set: the root of the repository that holds it. A set outside any repository
// has no tree beyond its own folder, so the search never climbs past it to
// an unrelated manifest.
func manifestSearchTop(start string) string {
	for directory := start; ; directory = filepath.Dir(directory) {
		if _, err := os.Lstat(filepath.Join(directory, ".git")); err == nil {
			return directory
		}
		if parent := filepath.Dir(directory); parent == directory {
			return start
		}
	}
}

// readSourceManifest bounds metadata reads without loading any source datasets.
func readSourceManifest(path string) ([]byte, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	raw, err := io.ReadAll(io.LimitReader(file, sourceManifestLimit+1))
	if err != nil {
		return nil, err
	}
	if len(raw) > sourceManifestLimit {
		return nil, fmt.Errorf("source manifest exceeds 8 MiB: %q", path)
	}
	return raw, nil
}
