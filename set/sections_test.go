package set_test

import (
	_ "embed"
	"strings"
	"testing"

	"github.com/janearc/puzzlemouth/yamlfile"
	"github.com/janearc/slayer/set"
)

//go:embed testdata/film-sections.yaml
var filmSectionsRaw []byte

// filmSections is testdata/film-sections.yaml: the lines a film is given
// and the sections and drift it is expected to say.
type filmSections struct {
	Sections []struct {
		Name, Lines string
		Over, Size  int
		Keep        bool
	}
	Drifts []struct {
		Name, Lines string
		Drift       float64
	}
	Edges []struct {
		Name, Lines string
		Over, Size  int
		Keep        bool
		Drift       float64
	}
	Refused []struct{ Name, Lines string }
}

// filmSectionsData reads testdata/film-sections.yaml.
func filmSectionsData(t *testing.T) filmSections {
	t.Helper()
	var data filmSections
	if err := yamlfile.Read("film-sections.yaml", filmSectionsRaw,
		&data); err != nil {
		t.Fatal(err)
	}
	return data
}

// onePlace is how many places of the set take the added lines. It is
// defined as one: the set has one film block.
const onePlace = 1

// filmed is Kilauea Iki's set with lines added to its film block, and the
// film it makes; nil where the set is refused.
func filmed(t *testing.T, lines string) (*set.Film, error) {
	t.Helper()
	written := strings.Replace(kilaueaIki(t), "film:\n", "film:\n"+lines,
		onePlace)
	if !strings.Contains(written, "film:\n") {
		t.Fatal("the set has no film block to add to")
	}
	made, err := set.Read("kilauea-iki.yaml", []byte(written))
	if err != nil {
		return nil, err
	}
	film, _ := made.Film()
	return &film, nil
}

// a film that says nothing of sections draws frames over 320 pixels wide
// in sections 64 pixels square and keeps the ones that did not change; a
// film that says them is as it says.
func TestAFilmSaysItsSections(t *testing.T) {
	for _, row := range filmSectionsData(t).Sections {
		film, err := filmed(t, row.Lines)
		if err != nil {
			t.Fatal(err)
		}
		over, size, keep := film.Sections()
		if over != row.Over || size != row.Size || keep != row.Keep {
			t.Errorf("%s: sections are over %d, size %d, keep %v, not over "+
				"%d, size %d, keep %v", row.Name, over, size, keep,
				row.Over, row.Size, row.Keep)
		}
	}
}

// sections with a size of nothing, a key the reader does not know, or a
// keep that is not true or false are refused.
func TestSectionsThatCannotBeAreRefused(t *testing.T) {
	for _, lines := range []string{
		"    sections: {size: 0}\n", "    sections: {drift: 2}\n",
		"    sections: {deviation: 1}\n",
		"    sections: {keep: sometimes}\n", "    sections: {over: -1}\n"} {
		if _, err := filmed(t, lines); err == nil {
			t.Errorf("a film with %q was read", strings.TrimSpace(lines))
		}
	}
}

// a film's drift is zero unless it says, and then what it says.
func TestAFilmSaysItsDrift(t *testing.T) {
	for _, row := range filmSectionsData(t).Drifts {
		film, err := filmed(t, row.Lines)
		if err != nil {
			t.Fatal(err)
		}
		if got := film.Drift(); got != row.Drift {
			t.Errorf("%s: with %q the drift is %v, not %v", row.Name,
				row.Lines, got, row.Drift)
		}
	}
}

// the least and most a film's sections take are taken as they are said,
// and the drift with them: one pixel and a hundred thousand, none of the
// steady lamps' light and the whole of it.
func TestSectionsAtTheirBoundsAreRead(t *testing.T) {
	for _, row := range filmSectionsData(t).Edges {
		film, err := filmed(t, row.Lines)
		if err != nil {
			t.Errorf("%s: %v", row.Name, err)
			continue
		}
		over, size, keep := film.Sections()
		if over != row.Over || size != row.Size || keep != row.Keep ||
			film.Drift() != row.Drift {
			t.Errorf("%s: over %d, size %d, keep %v, drift %v, not over %d, "+
				"size %d, keep %v, drift %v", row.Name, over, size, keep,
				film.Drift(), row.Over, row.Size, row.Keep, row.Drift)
		}
	}
}

// a width or side that is not a whole number of pixels from one to a
// hundred thousand, a drift outside zero to one, and a number that is
// not a number, are refused, each by its own line.
func TestSectionsPastTheirBoundsAreRefused(t *testing.T) {
	for _, row := range filmSectionsData(t).Refused {
		if _, err := filmed(t, row.Lines); err == nil {
			t.Errorf("%s: a film with %q was read", row.Name,
				strings.TrimSpace(row.Lines))
		}
	}
}
