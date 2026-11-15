package door

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// The bytes a note line is made of: a tab between the folder and the
// note, and a line break at its end. Neither may be in either part.
const (
	tab       = "\t"
	lineBreak = "\n"
)

// entry is one line of a team's notes file: a folder under the renders
// home and the note on it.
type entry struct{ folder, words string }

// newEntry is a note line, checked: a folder and words, neither empty
// and neither holding a tab or a line break, each refused by name.
func newEntry(folder, words string) (entry, error) {
	for _, part := range []struct{ name, text string }{
		{"folder", folder}, {"note", words}} {
		if part.text == "" {
			return entry{}, fmt.Errorf("the note's %s is empty", part.name)
		}
		if strings.ContainsAny(part.text, tab+lineBreak) {
			return entry{}, fmt.Errorf("the note's %s %q holds a tab or "+
				"a line break; a note is one line", part.name, part.text)
		}
	}
	return entry{folder: folder, words: words}, nil
}

// line is the entry as the notes file holds it.
func (note entry) line() string {
	return note.folder + tab + note.words + lineBreak
}

// into appends the entry to a notes file as one whole line with one
// write, as every other writer of that file does. A folder's earlier
// line is never rewritten: the renders page takes a folder's last line,
// and rewriting the file would lose a line another writer appended.
func (note entry) into(notes string) error {
	if err := os.MkdirAll(filepath.Dir(notes), folderMode); err != nil {
		return err
	}
	file, err := os.OpenFile(notes, os.O_APPEND|os.O_CREATE|os.O_RDWR,
		fileMode)
	if err != nil {
		return err
	}
	added := note.line()
	if !endsLine(file) {
		added = lineBreak + added
	}
	_, err = file.WriteString(added)
	return errors.Join(err, file.Close())
}

// endsLine is whether a file is empty or its last byte ends a line, so
// a line appended after it stands on its own.
func endsLine(file *os.File) bool {
	info, err := file.Stat()
	if err != nil || info.Size() == nothing {
		return true
	}
	last := make([]byte, one)
	if _, err := file.ReadAt(last, info.Size()-one); err != nil {
		return true
	}
	return string(last) == lineBreak
}

// copyFile copies a file to a path, beside itself and renamed into
// place. A file copied onto itself is left alone.
func copyFile(from, to string) error {
	if from == to {
		return nil
	}
	raw, err := os.ReadFile(from)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(to), folderMode); err != nil {
		return err
	}
	return replaced(to, raw)
}
