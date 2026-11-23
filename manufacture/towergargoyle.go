package manufacture

import (
	"github.com/janearc/slayer/manufacture/cities"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/strategies/scrooge"
)

// concreteSkin is the name of the tower skin that is cast concrete; the
// gargoyles of a tower of that skin are concrete too, and those of every
// other skin are granite.
const concreteSkin = "concrete"

// gargoyleStoneOf is the stone the gargoyles of a tower of a skin are cut
// from: concrete for a concrete tower, granite for the rest.
func gargoyleStoneOf(skin string) string {
	if skin == concreteSkin {
		return concreteSkin
	}
	return "granite"
}

// gargoyled is a tower with the gargoyles under its ledges, each on its
// corbel set into the face, if its recipe pins them on; any other tower
// is as it was. The gargoyles are the cities package's mounts of the
// tower, which are the places of kinds of this package, made from seeds
// drawn from the tower's.
//
// When scrooge is asked for and the pass carries an eye, the mounts are
// made only as fine as the picture can show, standing where the tower
// stands at place: none for a tower the eye does not see.
func (row towerRow) gargoyled(made good, tower *cities.Tower, from seed,
	pass cities.Pass, place maths.Subspace) (good, error) {
	if row.spec.Gargoyles == nil || !*row.spec.Gargoyles {
		return made, nil
	}
	var sight *mountSight
	if scrooge.Asked() && pass.Eye != nil {
		sight = &mountSight{eye: *pass.Eye, place: place}
		if !sight.eye.Sees(transformed(made.Bounds(), place)) {
			return made, nil
		}
	}
	stone := gargoyleStoneOf(row.spec.Skin)
	mounts, err := tower.GargoyleMounts(stone)
	if err != nil {
		return good{}, err
	}
	cornice := gargoyleCorniceRow{stone: gargoyleStones()[stone]}
	return cornice.withGargoyles(made, mounts, from, string(pass.Finest),
		sight)
}
