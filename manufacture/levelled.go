package manufacture

import (
	"fmt"

	"github.com/janearc/slayer/manufacture/cities"
	"github.com/janearc/slayer/maths"
)

// Levelled is the complexity a set gives a thing or a stand to say it is
// drawn as finely as the camera sees it: a thing near the eye at its
// finest, one far off as a hull. It is not a representation of its own,
// so Complexities does not name it.
const Levelled = "levelled"

// The ladders of the levels file, by what they level.
const (
	// treeLadder levels a tree, from its crown to its needles.
	treeLadder = "trees"
	// buildingLadder levels a building, a face, a storey and a window.
	buildingLadder = "buildings"
	// kindLadder levels any other kind that has its representations.
	kindLadder = "manufactory"
	// quickLadder is a kind that is only ever quick.
	quickLadder = "quick"
)

// levelledMaker is a kind drawn part by part, each part at its own level.
type levelledMaker interface {
	madeLevelled(from seed, pass cities.Pass,
		place maths.Subspace) (good, error)
}

// LevelsParts is whether a kind is drawn part by part when levelled: a
// building whose faces, storeys and windows are each levelled. Any other
// kind levelled is drawn whole, at the one level its place gives it.
func LevelsParts(row Kind) bool {
	_, parts := row.how.(levelledMaker)
	return parts
}

// MakeLevelled makes one of a kind drawn part by part, standing at a
// place in the world the pass's leveller sees. It refuses a kind that is
// not drawn part by part.
func MakeLevelled(row Kind, from seed, pass cities.Pass,
	place maths.Subspace) (good, error) {
	maker, parts := row.how.(levelledMaker)
	if !parts {
		return good{}, fmt.Errorf("manufacture: %s is not drawn part by "+
			"part", row.name)
	}
	return maker.madeLevelled(from, pass, place)
}

// LadderOf is the name of the ladder a kind is levelled by, and the
// ladder: trees by the trees', buildings by the buildings', a kind that
// is only ever quick by a ladder of that one rung, and any other by the
// manufactory's.
func LadderOf(row Kind) (string, []cities.Rung, error) {
	name := ladderName(row)
	if name == quickLadder {
		return name, []cities.Rung{{Level: "quick"}}, nil
	}
	ladder, err := cities.Ladder(name)
	return name, ladder, err
}

// ladderName is the name of the ladder of a kind's maker.
func ladderName(row Kind) string {
	switch row.how.(type) {
	case conifer, physicalSpruce:
		return treeLadder
	case towerRow, terraceRow, condominiumRow, villageHouseRow, villageRow:
		return buildingLadder
	}
	if _, levels := row.how.(interface {
		madeAt(seed, string) (good, error)
	}); levels {
		return kindLadder
	}
	return quickLadder
}

// constantPass is the pass that draws every part at one level.
func constantPass(detail string) cities.Pass {
	level := cities.Detail(detail)
	return cities.Pass{Leveller: cities.Constant(level), Finest: level}
}
