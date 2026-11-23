package session

import (
	"fmt"

	"github.com/janearc/slayer/manufacture"
)

// constructKey is what a construction is worked out from, besides what
// the stage holds fixed: its recipes, its sources list and its set.
type constructKey struct {
	kind, recipe string
}

// constructedRow is a construction's answer, its refusal kept as it was
// said.
type constructedRow struct {
	row *manufacture.Kind
	err error
}

// construction is the row of a kind or recipe, worked out by the first
// ask for the pair and kept: the stage's recipes, sources list and set
// do not change while it stands, and nothing else goes into the answer.
// A refusal is kept as well, so it is said in the same words each time.
// Only the stage's loop asks; the workers are given rows.
func (stage *staging) construction(kind, recipe string,
) (*manufacture.Kind, error) {
	key := constructKey{kind, recipe}
	if known, found := stage.constructions[key]; found {
		return known.row, known.err
	}
	row, err := stage.constructionWorked(kind, recipe)
	if stage.constructions == nil {
		stage.constructions = map[constructKey]constructedRow{}
	}
	stage.constructions[key] = constructedRow{row, err}
	return row, err
}

// constructionWorked is construction's work: the kind named, or the
// recipe constructed and set in the fonts its script asks for. It runs
// only on a stock miss. Static recipes retain their meshes.
func (stage *staging) constructionWorked(kind, recipe string,
) (*manufacture.Kind, error) {
	if recipe == "" {
		row, err := manufacture.KindNamed(kind)
		if err != nil {
			return nil, err
		}
		return &row, nil
	}
	row, found := stage.recipes.Recipe(recipe)
	if !found {
		return nil, fmt.Errorf("unknown construction recipe %q", recipe)
	}
	made, err := row.Construction()
	if err != nil {
		return nil, err
	}
	return stage.lettered(made, row.Script())
}

// constructed returns no stock on a construction refusal.
func (stage *staging) constructed(kind, recipe, detail string,
	seed uint64,
) (*kept, error) {
	return stage.constructedTaller(kind, recipe, detail, seed, 1, false)
}

// constructedTaller is constructed grown so many times taller, and its
// branches branched when asked: a whorled tree grown, not magnified, for
// improv/moonlake's -close and -giants. once as tall and unbranched is
// the kind as it is; anything but a whorled tree is refused.
func (stage *staging) constructedTaller(kind, recipe, detail string,
	seed uint64, times float64, branched bool,
) (*kept, error) {
	row, err := stage.construction(kind, recipe)
	if err != nil {
		return nil, err
	}
	return madeTaller(stage.given, *row, detail, seed, times, branched)
}

// madeTaller is constructedTaller from a row already constructed. It
// touches nothing of the stage but what it is given, so many may be made
// at once.
func madeTaller(given asked, row manufacture.Kind, detail string,
	seed uint64, times float64, branched bool,
) (*kept, error) {
	if times != 1 {
		grown, err := manufacture.Taller(row, times)
		if err != nil {
			return nil, err
		}
		row = grown
	}
	if branched {
		full, err := manufacture.Branched(row)
		if err != nil {
			return nil, err
		}
		row = full
	}
	one, err := manufacturedKind(given, row, int(seed), detail)
	if err != nil {
		return nil, err
	}
	return &one, nil
}
