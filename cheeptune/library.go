package cheeptune

import (
	"embed"
	"fmt"
	"io/fs"
	"sort"
	"strings"
)

// The library is the tunes made for videos: a score and a cut sheet each,
// in library/, compiled in beside the built-ins so a job does not need a
// checkout to name one. A library tune is named with an "@" like a built-in
// score, "@dawn-patrol", and its sheet names the voicing it was written for.
// A voicing may live in the library too, for the tunes written for it.
//
//	p, sheet, err := cheeptune.OpenTune("dawn-patrol")
//
// Built-in names win: a library tune may not take the name of a built-in,
// and a test holds that.

//go:embed library/*.json
var library embed.FS

// cutsSuffix is what a cut sheet's file name ends in, before .json.
const cutsSuffix = ".cuts"

// Library lists the names of the tunes in the library, in order.
func Library() []string {
	entries, err := fs.ReadDir(library, "library")
	if err != nil {
		return nil
	}
	var names []string
	for _, e := range entries {
		name := strings.TrimSuffix(e.Name(), ".json")
		if strings.HasSuffix(name, cutsSuffix) || libraryVoicing(name) {
			continue
		}
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// libraryVoicing is whether a library file is a voicing. A tune written
// for a sound no built-in voicing makes keeps its voicing beside it in the
// library; a sheet names it with an "@" as it names a built-in, and it is
// not a tune.
func libraryVoicing(name string) bool {
	b, err := libraryDoc(name)
	if err != nil {
		return false
	}
	kind, _ := Kind(b)
	return kind == "voicing"
}

// LibraryVoicings lists the voicings kept in the library, each named with
// its "@", in order.
func LibraryVoicings() []string {
	entries, err := fs.ReadDir(library, "library")
	if err != nil {
		return nil
	}
	var names []string
	for _, e := range entries {
		name := strings.TrimSuffix(e.Name(), ".json")
		if libraryVoicing(name) {
			names = append(names, "@"+name)
		}
	}
	sort.Strings(names)
	return names
}

// libraryDoc is a library file by its name without the extension.
func libraryDoc(name string) ([]byte, error) {
	return library.ReadFile("library/" + name + ".json")
}

// OpenTune renders a library tune with the voicing its sheet names, and
// returns the sheet with it. An unknown name is refused with the names that
// exist, and a sheet whose voicing does not exist with the tune and the
// voicing.
func OpenTune(name string) (*Piece, *CutSheet, error) {
	return openTune(name, libraryDoc)
}

// openTune is OpenTune reading the tune's documents with read.
func openTune(name string, read func(string) ([]byte, error)) (*Piece,
	*CutSheet, error) {
	name = strings.TrimPrefix(name, "@")
	sb, err := read(name)
	if err != nil {
		return nil, nil, fmt.Errorf("no library tune %q; there is %s", name,
			strings.Join(Library(), " "))
	}
	s, err := ParseScore(sb)
	if err != nil {
		return nil, nil, fmt.Errorf("%s: %w", name, err)
	}
	cb, err := read(name + cutsSuffix)
	if err != nil {
		return nil, nil, fmt.Errorf("library tune %q has no cut sheet", name)
	}
	sheet, err := ParseCuts(cb, s)
	if err != nil {
		return nil, nil, fmt.Errorf("%s: %w", name+cutsSuffix, err)
	}
	p, err := Open("@"+name, sheet.Voicing)
	if err != nil {
		return nil, nil, fmt.Errorf("library tune %q with voicing %q: %w",
			name, sheet.Voicing, err)
	}
	return p, sheet, nil
}
