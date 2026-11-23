package manufacture

import (
	"fmt"
	"maps"
	"slices"

	"github.com/janearc/slayer/manufacture/cities"
)

// SignScripts are the scripts a recipe may ask the words of its signs to
// be in, the Latin first, which is the script of a sign that asks for
// none.
func SignScripts() []string { return cities.ScriptNames() }

// SignFaces are the keys, in sources.yaml, of the fonts the signs of a
// script are set in, which a shelf keeps as files. The Latin script is set
// in go's own fonts, which are no files, and has none.
func SignFaces(script string) ([]string, error) {
	return cities.ScriptFaces(cities.Script(script))
}

// WithScript is the row with the words of its signs in a script and, if
// carries names any, each of those trades carried once by every building
// the row makes, on a sign whatever the weights of the sign culture say.
//
// It is for the kinds that take their words from the sign culture: a tower
// that pins signs on, and the signs that stand on their own. It refuses a
// script, or a trade the script has no words for, that the culture does
// not have, and a kind that draws no words from the culture, which would
// have no use for a script. Until WithFaces says where the fonts are, the
// row can say a script but not draw it.
func WithScript(row Kind, script string, carries []string) (Kind, error) {
	lettering := cities.Lettering{Script: cities.Script(script),
		Carries: slices.Clone(carries)}
	if err := lettering.Check(); err != nil {
		return Kind{}, err
	}
	switch how := row.how.(type) {
	case towerRow:
		if how.spec.Signs == nil || !*how.spec.Signs {
			return Kind{}, fmt.Errorf("manufacture: a tower has signs to " +
				"put in a script only if it pins signs on")
		}
		how.spec.Lettering = lettering
		return Kind{name: row.name, how: how}, nil
	case signRow:
		if err := lettering.CheckFree(how.mount); err != nil {
			return Kind{}, err
		}
		how.lettering = lettering
		return Kind{name: row.name, how: how}, nil
	}
	return Kind{}, fmt.Errorf("manufacture: %s draws no words from the "+
		"sign culture, so it has no script or trades to carry", row.name)
}

// WithFaces is the row with the files of the fonts its script is set in,
// by the keys SignFaces gives. It refuses a kind that draws no signs.
func WithFaces(row Kind, faces map[string]string) (Kind, error) {
	switch how := row.how.(type) {
	case towerRow:
		how.spec.Lettering.Faces = maps.Clone(faces)
		return Kind{name: row.name, how: how}, nil
	case signRow:
		how.lettering.Faces = maps.Clone(faces)
		return Kind{name: row.name, how: how}, nil
	}
	return Kind{}, fmt.Errorf("manufacture: %s draws no signs, so it has "+
		"no fonts to set them in", row.name)
}
