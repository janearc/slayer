package audition

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/janearc/slayer/cheeptune"
)

// sheetFileSuffix is what a sheet's file name ends in; scoreSuffix is what
// a score's does.
const (
	sheetFileSuffix = ".cuts.json"
	scoreSuffix     = ".json"
)

// namePattern is what a tune's name may be: lowercase words joined by
// hyphens, the library's own rule. The name becomes an element id and a
// file path, so nothing else is let through.
var namePattern = regexp.MustCompile(`^[a-z]+(-[a-z]+)*$`)

// validName refuses a name that is not lowercase words joined by hyphens.
func validName(name string) error {
	if !namePattern.MatchString(name) {
		return fmt.Errorf("%q is not a tune's name; a name is lowercase "+
			"words joined by hyphens, such as glass-hour", name)
	}
	return nil
}

// Tune is one tune of a library folder: its score, its sheet and the piece
// the two make. Read leaves the piece out; Load renders it.
type Tune struct {
	Name  string
	Score *cheeptune.Score
	Sheet *cheeptune.CutSheet
	Piece *cheeptune.Piece
}

// Names lists the tunes of a folder, in order: every NAME.json with its
// sheet NAME.cuts.json beside it. The votes record, a score with no sheet
// and a sheet with no score are not tunes and are skipped. A tune whose
// name is not lowercase words joined by hyphens is refused by name.
func Names(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	files := map[string]bool{}
	for _, e := range entries {
		files[e.Name()] = true
	}
	var names []string
	for file := range files {
		name, ok := strings.CutSuffix(file, scoreSuffix)
		if !ok || strings.HasSuffix(file, sheetFileSuffix) ||
			file == votesFile || !files[name+sheetFileSuffix] {
			continue
		}
		if err := validName(name); err != nil {
			return nil, fmt.Errorf("the folder %s: %s: %w", dir, file, err)
		}
		names = append(names, name)
	}
	sort.Strings(names)
	return names, nil
}

// Read reads a tune from a folder and checks its sheet against its score,
// and renders nothing: the tune it returns has no piece. It is all the
// page needs (a name, the sheet's words and parts, a length and a tempo),
// so a page can be written by a tree that does not have the tune's
// voicing. A sheet that disagrees with its score is refused by name.
func Read(dir, name string) (*Tune, error) {
	if err := validName(name); err != nil {
		return nil, err
	}
	path := filepath.Join(dir, name+scoreSuffix)
	score, err := cheeptune.ReadScore(path)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	sheet, err := cheeptune.ReadCuts(filepath.Join(dir, name+sheetFileSuffix),
		score)
	if err != nil {
		return nil, err
	}
	return &Tune{Name: name, Score: score, Sheet: sheet}, nil
}

// Load is Read and then the render: the tune with its piece, which a
// plate and a film need. A tune that clips, or whose sheet names a
// voicing this tree does not have, is refused by name.
func Load(dir, name string) (*Tune, error) {
	t, err := Read(dir, name)
	if err != nil {
		return nil, err
	}
	path := filepath.Join(dir, name+scoreSuffix)
	t.Piece, err = cheeptune.Open(path, t.Sheet.Voicing)
	if err != nil {
		return nil, fmt.Errorf("%s with %s: %w", name, t.Sheet.Voicing, err)
	}
	return t, nil
}

// rendered refuses a tune that was read and not rendered, for the things
// that draw or play its piece.
func (t *Tune) rendered() error {
	if t.Piece == nil {
		return fmt.Errorf("%s was read and not rendered, and a plate or "+
			"a film needs its sound; use Load", t.Name)
	}
	return nil
}
