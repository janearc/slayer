package settings

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Change is one key's new value in a file of game's format. An empty Value
// removes the key.
type Change struct{ Key, Value string }

// configFileMode is the mode of a file Update makes new: only its owner reads
// and writes it.
const configFileMode = 0o600

// configDirMode is the mode of a directory Update has to make for the file.
const configDirMode = 0o700

// Update sets the keys in changes in the file at path, which is in game's
// format: a line is a key, its words, and, for a tail, " :: " and the rest.
// It writes "key :: value" for each key it is given. Every other line stays
// as it was: comments, blank lines, their order and the keys it does not
// know.
//
// A key that is already there is replaced where it stands, and its repeats
// are dropped, with any indented lines that belonged to them. A key that is
// not there is added at the end. The file is made if it is missing, with its
// directory, and is replaced by renaming a new file beside it, so a reader
// sees the old file or the new one and never half of either.
//
// It knows nothing of the tool whose file it is, so it could live in the
// library that reads these files.
func Update(path string, changes []Change) error {
	for _, change := range changes {
		if err := checkChange(change); err != nil {
			return err
		}
	}
	raw, err := os.ReadFile(path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	lines := rewrite(splitLines(string(raw)), changes)
	text := strings.Join(lines, "\n")
	if len(lines) > 0 {
		text += "\n"
	}
	return replaceFile(path, []byte(text))
}

// checkChange refuses a key or value that could not be one line of a file.
func checkChange(change Change) error {
	if change.Key == "" || strings.ContainsAny(change.Key, " \t\r\n") {
		return fmt.Errorf("a key is one word, not %q", change.Key)
	}
	if strings.ContainsAny(change.Value, "\r\n") {
		return fmt.Errorf("the value of %s is more than one line", change.Key)
	}
	return nil
}

// splitLines is text as its lines, without their ends and with no empty
// last line for the end of the text.
func splitLines(text string) []string {
	if text == "" {
		return nil
	}
	return strings.Split(strings.TrimSuffix(text, "\n"), "\n")
}

// rewrite is lines with changes made as Update says.
func rewrite(lines []string, changes []Change) []string {
	owned := map[string]Change{}
	for _, change := range changes {
		owned[change.Key] = change
	}
	var result []string
	done := map[string]bool{}
	dropping := false
	for _, line := range lines {
		if dropping && strings.TrimLeft(line, " \t") != line {
			continue
		}
		dropping = false
		key := lineKey(line)
		change, ours := owned[key]
		if !ours {
			result = append(result, line)
			continue
		}
		dropping = true
		result = append(result, settingLine(change, done)...)
		done[key] = true
	}
	for _, change := range changes {
		result = append(result, settingLine(change, done)...)
		done[change.Key] = true
	}
	return result
}

// settingLine is the one line that sets change, or none when the key has
// been set already or the change removes it.
func settingLine(change Change, done map[string]bool) []string {
	if done[change.Key] || change.Value == "" {
		return nil
	}
	return []string{change.Key + " :: " + change.Value}
}

// lineKey is the key a line sets: its first word. A blank line, a comment
// and an indented line have none.
func lineKey(line string) string {
	if line == "" || line[0] == '#' || line[0] == ' ' || line[0] == '\t' {
		return ""
	}
	words := strings.Fields(line)
	if len(words) == 0 {
		return ""
	}
	return words[0]
}

// replaceFile puts content at path by writing a new file beside it and
// renaming it over. The file keeps the mode of the one it replaces.
func replaceFile(path string, content []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), configDirMode); err != nil {
		return err
	}
	mode := os.FileMode(configFileMode)
	if info, err := os.Stat(path); err == nil {
		mode = info.Mode().Perm()
	}
	staged, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+"-*")
	if err != nil {
		return err
	}
	_, err = staged.Write(content)
	if err == nil {
		err = staged.Sync()
	}
	if closeError := staged.Close(); err == nil {
		err = closeError
	}
	if err == nil {
		err = os.Chmod(staged.Name(), mode)
	}
	if err == nil {
		err = os.Rename(staged.Name(), path)
	}
	if err != nil {
		os.Remove(staged.Name())
	}
	return err
}
