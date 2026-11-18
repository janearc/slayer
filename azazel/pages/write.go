package pages

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
)

// written is how a published file may be read: by anyone on the machine.
const written fs.FileMode = 0o644 // defined: what frames-index.sh gives a page

// partEnding ends every temporary name, so that no temporary file ever
// ends as a picture or a film does and the gallery never counts one.
const partEnding = ".part" // picked: no gallery or page ending

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

// writeBeside writes a file whole: into a temporary file beside it, then
// renamed over its name, so no reader ever sees part of one.
func writeBeside(path string, data []byte, mode fs.FileMode) error {
	pattern := "." + filepath.Base(path) + ".*" + partEnding
	file, err := os.CreateTemp(filepath.Dir(path), pattern)
	if err != nil {
		return err
	}
	_, err = file.Write(data)
	err = errors.Join(err, file.Chmod(mode), file.Close())
	if err == nil {
		err = os.Rename(file.Name(), path)
	}
	if err != nil {
		return errors.Join(err, os.Remove(file.Name()))
	}
	return nil
}

// regularFile says whether path names a file, following a link, as the
// shell's -f test does.
func regularFile(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.Mode().IsRegular()
}
