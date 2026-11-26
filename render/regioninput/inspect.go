package regioninput

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// Inspect hashes explicitly named files in an existing snapshot. It rejects
// symlinks within the snapshot and detects ordinary file replacement while
// reading. The caller must keep the snapshot stable and enumerate its complete
// closure; this function cannot prove either condition.
func Inspect(root string, names []string) ([]Entry, error) {
	if !filepath.IsAbs(root) || len(names) == 0 {
		return nil, fmt.Errorf("regioninput: absolute snapshot and input names are required")
	}
	actualRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		return nil, err
	}
	seen := make(map[string]bool, len(names))
	entries := make([]Entry, 0, len(names))
	for _, name := range names {
		if !filepath.IsLocal(name) || filepath.Clean(name) != name || name == "." || seen[name] {
			return nil, fmt.Errorf("regioninput: invalid or duplicate name %q", name)
		}
		seen[name] = true
		path := filepath.Join(actualRoot, name)
		resolved, err := filepath.EvalSymlinks(path)
		if err != nil {
			return nil, err
		}
		if resolved != path {
			return nil, fmt.Errorf("regioninput: symlink in snapshot path %q", name)
		}
		before, err := os.Lstat(path)
		if err != nil {
			return nil, err
		}
		if !before.Mode().IsRegular() {
			return nil, fmt.Errorf("regioninput: input %q is not a regular file", name)
		}
		file, err := os.Open(path)
		if err != nil {
			return nil, err
		}
		opened, err := file.Stat()
		if err != nil {
			file.Close()
			return nil, err
		}
		if !opened.Mode().IsRegular() || !os.SameFile(before, opened) {
			file.Close()
			return nil, fmt.Errorf("regioninput: input %q changed", name)
		}
		hash := sha256.New()
		n, readErr := io.Copy(hash, file)
		after, statErr := file.Stat()
		closeErr := file.Close()
		if readErr != nil {
			return nil, readErr
		}
		if statErr != nil {
			return nil, statErr
		}
		if closeErr != nil {
			return nil, closeErr
		}
		if n != opened.Size() || after.Size() != opened.Size() || !after.ModTime().Equal(opened.ModTime()) {
			return nil, fmt.Errorf("regioninput: input %q changed while read", name)
		}
		var digest [32]byte
		copy(digest[:], hash.Sum(nil))
		entries = append(entries, Entry{Name: name, Bytes: n, SHA256: digest})
	}
	return entries, nil
}
