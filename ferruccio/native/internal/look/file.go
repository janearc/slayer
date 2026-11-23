package look

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Suffix ends the name of a looks file: neon-street.yaml has
// neon-street.looks.txt. It is not a yaml name, so tools that read every
// set in a folder do not take a looks file for a set.
const Suffix = ".looks.txt"

// PathFor is the looks file that sits beside the set at path.
func PathFor(setPath string) string {
	base := strings.TrimSuffix(filepath.Base(setPath), filepath.Ext(setPath))
	return filepath.Join(filepath.Dir(setPath), base+Suffix)
}

// Place is where a look of the set at setPath is written: beside the set,
// or, for an unsaved example that has no place of its own, in the evidence
// directory. It never names the set file itself.
func Place(setPath, evidence string, unsaved bool) (string, error) {
	if setPath == "" {
		return "", errors.New("open or save a YAML file before writing a look")
	}
	if !unsaved {
		return PathFor(setPath), nil
	}
	if evidence == "" {
		return "", errors.New("this example has no place of its own and no evidence directory is set: save it, or give --evidence")
	}
	return filepath.Join(evidence, filepath.Base(PathFor(setPath))), nil
}

// Append adds block to the end of the file at path, making it if need be.
// Earlier looks are kept. A file that did not end in a newline gets one
// first, so the block starts on a line of its own.
func Append(path, block string) error {
	existing, err := os.ReadFile(path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if len(existing) > 0 && existing[len(existing)-1] != '\n' {
		block = "\n" + block
	}
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0o644)
	if err != nil {
		return err
	}
	if _, err := file.WriteString(block); err != nil {
		file.Close()
		return err
	}
	return file.Close()
}

// Blocks splits the text of a looks file into its looks, oldest first. Each
// begins with its divider line. Text before the first divider is not a look.
func Blocks(text string) []string {
	var blocks []string
	var current []string
	for _, line := range strings.SplitAfter(text, "\n") {
		if strings.TrimRight(line, "\r\n") == Divider {
			if current != nil {
				blocks = append(blocks, strings.Join(current, ""))
			}
			current = []string{}
		}
		if current != nil {
			current = append(current, line)
		}
	}
	if current != nil {
		blocks = append(blocks, strings.Join(current, ""))
	}
	return blocks
}

// Read is the looks in the file at path, oldest first. A file that is not
// there is an error that says so, naming it.
func Read(path string) ([]string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("no looks: %w", err)
	}
	blocks := Blocks(string(raw))
	if len(blocks) == 0 {
		return nil, fmt.Errorf("no looks in %s", path)
	}
	return blocks, nil
}
