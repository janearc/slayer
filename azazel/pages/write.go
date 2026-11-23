package pages

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// written is how a published file may be read: by anyone on the machine.
const written fs.FileMode = 0o644 // defined: what frames-index.sh gives a page

// partEnding ends every temporary name, so that no temporary file ever
// ends as a picture or a film does and the gallery never counts one.
const partEnding = ".part" // picked: no gallery or page ending

// copyEnding ends the name of a set file's copy in a render folder.
const copyEnding = "-yaml.txt" // defined: latest.sh's copy of a set file

// placeBeside makes an empty temporary file in the folder of path, for a
// writer or a tool to fill and then rename over path.
func placeBeside(path string) (string, error) {
	pattern := "." + filepath.Base(path) + ".*" + partEnding
	file, err := os.CreateTemp(filepath.Dir(path), pattern)
	if err != nil {
		return "", err
	}
	return file.Name(), file.Close()
}

// writeBeside writes a file whole: into a temporary file beside it, synced,
// then renamed over its name and the folder synced, so no reader ever sees
// part of one and a power cut cannot leave an empty one.
func writeBeside(path string, data []byte, mode fs.FileMode) error {
	pattern := "." + filepath.Base(path) + ".*" + partEnding
	file, err := os.CreateTemp(filepath.Dir(path), pattern)
	if err != nil {
		return err
	}
	_, err = file.Write(data)
	err = errors.Join(err, file.Chmod(mode), file.Sync(), file.Close())
	if err == nil {
		err = renameDurably(file.Name(), path)
	}
	if err != nil {
		fileName := file.Name()
		return errors.Join(err, os.Remove(fileName))
	}
	return nil
}

// renameDurably renames temporary over path with both on storage: the
// file is synced, then the folder, then the rename is made and the folder
// synced again.
func renameDurably(temporary, path string) error {
	if err := syncPath(temporary); err != nil {
		return err
	}
	folder := filepath.Dir(path)
	if err := syncPath(folder); err != nil {
		return err
	}
	if err := os.Rename(temporary, path); err != nil {
		return err
	}
	return syncPath(folder)
}

// syncPath flushes the file or folder at path to storage.
func syncPath(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	return errors.Join(file.Sync(), file.Close())
}

// regularFile says whether path names a file, following a link, as the
// shell's -f test does.
func regularFile(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.Mode().IsRegular()
}

// removeLeftovers removes the temporary files that a write of this package
// left in dir when it was killed: hidden names that end as partEnding does,
// of a page, a film, a record, a line or a set file's copy. A temporary of
// another tool is never touched.
func removeLeftovers(dir string) error {
	entries, err := os.ReadDir(dir)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	var failures []error
	for _, entry := range entries {
		if entry.Type().IsRegular() && leftover(entry.Name()) {
			entryPath := filepath.Join(dir, entry.Name())
			failures = append(failures, os.Remove(entryPath))
		}
	}
	return errors.Join(failures...)
}

// leftover says whether name is a temporary that this package makes.
func leftover(name string) bool {
	if !strings.HasPrefix(name, ".") || !strings.HasSuffix(name, partEnding) {
		return false
	}
	for _, base := range []string{pageName, latestName, progressName,
		partialName, partialRecord} {
		if strings.HasPrefix(name, "."+base+".") {
			return true
		}
	}
	return strings.Contains(name, copyEnding+".")
}
