package session

import (
	"errors"
	"fmt"

	"github.com/janearc/slayer/manufacture"
	"github.com/janearc/slayer/manufacture/cities"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/strategies"
	"github.com/janearc/slayer/maths/strategies/scrooge"
	"github.com/janearc/slayer/set"
)

// A thing or a stand whose complexity is levelled is drawn as finely as
// the set's camera sees it. Each thing of a stand is made at the one
// level its place and its size give it, from the stock as any other; a
// named building that is drawn part by part is made for where it stands,
// each face, storey, window and band of storeys at its own level.

// errNoSight is the refusal of a levelled thing in a set with no camera.
var errNoSight = errors.New("a levelled thing needs the set's camera, " +
	"and the set has none")

// seen is the set's camera as a leveller sees it: the camera the job
// names, or the first, through its lens, on the film's pixels.
func (stage *staging) seen() (strategies.Eye, error) {
	if stage.sight != nil {
		return *stage.sight, nil
	}
	shot, err := stage.levellingShot()
	if err != nil {
		return strategies.Eye{}, fmt.Errorf("the leveller: %w", err)
	}
	eye := strategies.Eye{Place: shot.Eye(),
		Wide:   stage.given.lensOf(shot).wide * maths.Degree,
		Across: stage.given.width, Down: stage.given.height}
	stage.sight = &eye
	return eye, nil
}

// levellingShot is the camera the job names, or the set's first.
func (stage *staging) levellingShot() (set.Shot, error) {
	if stage.given.camera != "" {
		return namedShot(stage.recipes, stage.given.camera, stage.where)
	}
	shots, err := stage.recipes.Shots(stage.where)
	if err != nil {
		return set.Shot{}, err
	}
	if len(shots) == 0 {
		return set.Shot{}, errNoSight
	}
	return shots[0], nil
}

// levellerOf is the leveller of a kind's ladder from the set's camera,
// made once for each ladder, and the ladder.
func (stage *staging) levellerOf(row manufacture.Kind) (cities.Leveller,
	[]cities.Rung, error) {
	name, ladder, err := manufacture.LadderOf(row)
	if err != nil {
		return nil, nil, err
	}
	if made, known := stage.levellers[name]; known {
		return made, ladder, nil
	}
	eye, err := stage.seen()
	if err != nil {
		return nil, nil, err
	}
	made, err := cities.NewLeveller(eye, ladder)
	if err != nil {
		return nil, nil, err
	}
	if stage.levellers == nil {
		stage.levellers = map[string]cities.Leveller{}
	}
	stage.levellers[name] = made
	return made, ladder, nil
}

// levelledStock is a thing at its complexity from the stock; levelled,
// at the level the camera gives it where it stands, or, a named thing of
// a kind drawn part by part, made for its place.
func (stage *staging) levelledStock(kind, recipe, detail string,
	seed uint64, name string, at maths.Subspace) (kept, error) {
	if detail != manufacture.Levelled {
		if one, sparing, err := stage.sparingly(kind, recipe, detail, seed,
			name, at); sparing || err != nil {
			return one, err
		}
		return stage.stockOf(kind, recipe, detail, seed, name)
	}
	row, err := stage.construction(kind, recipe)
	if err != nil {
		return kept{}, err
	}
	if name != "" && manufacture.LevelsParts(*row) {
		return stage.composite(*row, seed, at)
	}
	level, err := stage.levelAt(*row, kind, recipe, seed, name, at)
	if err != nil {
		return kept{}, err
	}
	if manufacture.LevelsParts(*row) {
		return stage.stockWhole(*row, kind, recipe, level, seed)
	}
	return stage.stockOf(kind, recipe, level, seed, name)
}

// stockWhole is a kind drawn part by part, made for its ladder's finest
// and drawn whole at one level, from the stock: a far tower of a stand
// drawn bare keeps its lit windows, as one made for the hull would not.
func (stage *staging) stockWhole(row manufacture.Kind, kind, recipe,
	level string, seed uint64) (kept, error) {
	_, ladder, err := stage.levellerOf(row)
	if err != nil {
		return kept{}, err
	}
	finest := ladder[len(ladder)-1].Level
	key := stocked{name: stockIdentity(kind, recipe, 1, false) +
		", made for " + string(finest), pick: int(seed), detail: level}
	if one, known := stage.have[key]; known {
		return one, nil
	}
	pass := cities.Pass{Leveller: cities.Constant(cities.Detail(level)),
		Finest: finest}
	one, err := manufacturedLevelled(stage.given, row, int(seed), pass,
		maths.Subspace{})
	if err != nil {
		return kept{}, err
	}
	stage.have[key] = one
	stage.made.kept += one.stored
	return one, nil
}

// levelAt is the level the camera gives a thing standing at a place, by
// the box it fills at its ladder's coarsest, which is measured once for
// each kind and seed and not kept as a thing.
func (stage *staging) levelAt(row manufacture.Kind, kind, recipe string,
	seed uint64, name string, at maths.Subspace) (string, error) {
	leveller, ladder, err := stage.levellerOf(row)
	if err != nil {
		return "", err
	}
	coarsest := string(ladder[0].Level)
	key := stocked{name: kind + "/" + recipe + "/" + name, pick: int(seed),
		detail: coarsest}
	box, known := stage.sized[key]
	if !known {
		one, err := stage.constructed(kind, recipe, coarsest, seed)
		if err != nil {
			return "", err
		}
		box = one.bounds
		if stage.sized == nil {
			stage.sized = map[stocked]maths.Box{}
		}
		stage.sized[key] = box
	}
	return string(levelIn(leveller, box, at)), nil
}

// levelIn is the level a leveller gives a box of a thing that stands at
// a place.
func levelIn(leveller cities.Leveller, box maths.Box,
	at maths.Subspace) cities.Detail {
	middle := at.Out(box.Min.Add(box.Max).Scale(0.5))
	return leveller(maths.Subspace{Origin: middle},
		box.Max.Sub(box.Min).Len())
}

// composite is a kind drawn part by part, made for a place: each part at
// the level the camera gives it, the whole built at its ladder's finest.
func (stage *staging) composite(row manufacture.Kind, seed uint64,
	at maths.Subspace) (kept, error) {
	_, ladder, err := stage.levellerOf(row)
	if err != nil {
		return kept{}, err
	}
	return stage.compositeUpTo(row, seed, at, len(ladder)-1)
}

// compositeUpTo is composite with no part drawn finer than a rung of the
// kind's ladder, the whole built for that rung.
func (stage *staging) compositeUpTo(row manufacture.Kind, seed uint64,
	at maths.Subspace, rung int) (kept, error) {
	leveller, ladder, err := stage.levellerOf(row)
	if err != nil {
		return kept{}, err
	}
	eye, err := stage.seen()
	if err != nil {
		return kept{}, err
	}
	one, err := composing{row: row, leveller: leveller, ladder: ladder,
		eye: eye}.made(stage.given, seed, at, rung)
	if err != nil {
		return kept{}, err
	}
	stage.made.kept += one.stored
	return one, nil
}

// composing is a kind drawn part by part, with what drawing it takes of
// the stage: its ladder, the leveller of the ladder from the camera, and
// the camera. Making one touches nothing of the stage but what it holds
// and what it is given, so many may be made at once.
type composing struct {
	row      manufacture.Kind
	leveller cities.Leveller
	ladder   []cities.Rung
	eye      strategies.Eye
}

// made is composing's kind drawn part by part, for a place, no part finer
// than a rung of its ladder and the whole built for that rung.
func (with composing) made(given asked, seed uint64, at maths.Subspace,
	rung int) (kept, error) {
	finest := with.ladder[rung].Level
	held := func(at maths.Subspace, extent float64) cities.Detail {
		level := with.leveller(at, extent)
		for _, coarser := range with.ladder[:rung] {
			if level == coarser.Level {
				return level
			}
		}
		return finest
	}
	eye := with.eye
	pass := cities.Pass{Leveller: held, Sees: eye.Sees, Finest: finest,
		Eye: &eye}
	return manufacturedLevelled(given, with.row, int(seed), pass, at)
}

// sparingly is a named thing of a kind drawn part by part whose set names
// one complexity for the whole of it, made instead part by part where
// scrooge is asked: each face, storey and mount as the camera sees it,
// and none finer than the set's complexity. It says whether it made the
// thing; a thing it does not make is the stock's, whole, as before.
//
// A thing made ahead of the loop, for these very arguments, is taken as it
// was made.
func (stage *staging) sparingly(kind, recipe, detail string, seed uint64,
	name string, at maths.Subspace) (kept, bool, error) {
	if !scrooge.Asked() || name == "" {
		return kept{}, false, nil
	}
	key := warmKey{kind: kind, recipe: recipe, detail: detail, seed: seed,
		name: name, at: at}
	if one, taken := stage.warm.composite(key); taken {
		stage.made.kept += one.stored
		return one, true, nil
	}
	row, rung, sparing := stage.sparingRung(kind, recipe, detail)
	if !sparing {
		return kept{}, false, nil
	}
	one, err := stage.compositeUpTo(*row, seed, at, rung)
	return one, true, err
}

// sparingRung is a kind's row, and, where the kind is drawn part by part
// and the complexity is a rung of its ladder, which rung and true. The
// row is nil only where the kind cannot be constructed.
func (stage *staging) sparingRung(kind, recipe, detail string) (
	*manufacture.Kind, int, bool) {
	row, err := stage.construction(kind, recipe)
	if err != nil {
		return nil, 0, false
	}
	if !manufacture.LevelsParts(*row) {
		return row, 0, false
	}
	_, ladder, err := manufacture.LadderOf(*row)
	if err != nil {
		return row, 0, false
	}
	for rung, each := range ladder {
		if string(each.Level) == detail {
			return row, rung, true
		}
	}
	return row, 0, false
}
