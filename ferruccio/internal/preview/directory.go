package preview

import (
	"fmt"
	"path/filepath"
	"strings"
	"unicode"
)

// directoryPrefix validates a basename before Prepare creates any artifact.
func (request Request) directoryPrefix() (string, error) {
	prefix := request.DirectoryPrefix
	if prefix == "" {
		return "ferruccio-render-", nil
	}
	if prefix == "." || prefix == ".." || strings.TrimSpace(prefix) == "" ||
		filepath.Base(prefix) != prefix || filepath.VolumeName(prefix) != "" ||
		strings.ContainsAny(prefix, "/\\") {
		return "", fmt.Errorf("preview directory prefix must be one nonempty basename")
	}
	for _, character := range prefix {
		if unicode.IsControl(character) {
			return "", fmt.Errorf("preview directory prefix cannot contain control characters")
		}
	}
	return prefix, nil
}
