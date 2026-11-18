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
		if parent := filepath.Dir(directory); parent == directory {
			return nil
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
