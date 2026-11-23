package session

import (
	"github.com/janearc/slayer/manufacture"
	"github.com/janearc/slayer/manufacture/cities"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/set"
)

// A building of a kind that knows slopes is not stocked and stood again
// and again at each place its stand gives: each is made for the ground
// under it, so that it stands on a plinth to the lowest of that ground.
// Anything else is stood as it always was.

// groundUnder is the terrain as a building standing at a place and turned
// about the vertical tells of it: the height of the ground at a point in
// the building's own frame, over the height of the place. It reads the
// terrain mesh itself, so a corner of a plinth is on the ground the
// picture draws.
func (stage *staging) groundUnder(place maths.Vec, turn float64) cities.Terrain {
	orientation := maths.About(maths.YAxis, turn)
	return func(x, z float64) float64 {
		at := place.Add(orientation.Out(maths.V(x, 0, z)))
		return stage.groundAt(at.X, at.Z) - place.Y
	}
}

// standSite stands one thing of a stand at a site. On terrain, a kind that
// knows slopes is made for the ground under the site; any other kind, and
// any thing on flat ground, is stood from the stock.
func (stage *staging) standSite(each set.Stand, sloped bool, pick,
	index uint64, place maths.Vec, facing *float64) (maths.Box, error) {
	if !sloped || each.Recipe() != "" || !manufacture.SlopeAware(each.Kind()) {
		return stage.putRecipeFacing(each.Kind(), each.Recipe(),
			each.Complexity(), pick, index, place, index < uint64(each.Alight()),
			"", facing)
	}
	turn := stage.placementTurn(each.Kind(), "", pick, index)
	if facing != nil {
		turn = *facing
	}
	row, err := manufacture.KindNamed(each.Kind())
	if err != nil {
		return maths.Box{}, err
	}
	row, err = stage.founded(each, row, stage.groundUnder(place, turn))
	if err != nil {
		return maths.Box{}, err
	}
	at := maths.Subspace{Origin: place,
		Orientation: maths.About(maths.YAxis, turn)}
	one, err := stage.grounded(row, pick, each.Complexity(), at)
	if err != nil {
		return maths.Box{}, err
	}
	return one.bounds, stage.add(one, each.Kind(), at, pick,
		index < uint64(each.Alight()))
}

// founded is a kind stood on the ground under a site. A stand that climbs
// has the floor of each bay founded at the ground outside its door, since
// the stand says which way the doors look; any other stands it as it
// always was, on the highest ground under it.
func (stage *staging) founded(each set.Stand, row manufacture.Kind,
	ground cities.Terrain) (manufacture.Kind, error) {
	if _, climbs := each.Climb(); climbs {
		return manufacture.Founded(row, ground, cities.FoundAtDoor)
	}
	return manufacture.Grounded(row, ground)
}

// grounded is a kind made for the ground under a site, at a complexity.
// Levelled, a kind drawn part by part is made for the place, and any
// other at the level the camera gives the box it fills at its coarsest.
func (stage *staging) grounded(row manufacture.Kind, pick uint64,
	detail string, at maths.Subspace) (kept, error) {
	if detail != manufacture.Levelled {
		return manufacturedKind(stage.given, row, int(pick), detail)
	}
	if manufacture.LevelsParts(row) {
		return stage.composite(row, pick, at)
	}
	leveller, ladder, err := stage.levellerOf(row)
	if err != nil {
		return kept{}, err
	}
	coarse, err := manufacturedKind(stage.given, row, int(pick),
		string(ladder[0].Level))
	if err != nil {
		return kept{}, err
	}
	level := levelIn(leveller, coarse.bounds, at)
	return manufacturedKind(stage.given, row, int(pick), string(level))
}

// villagey is whether a thing's kind, or its recipe's, is a village or a
// house of one, the kinds groundedThing may stand on the ground under them.
func (stage *staging) villagey(each set.Thing) bool {
	kind := each.Kind()
	if asked, found := stage.recipes.Recipe(each.Recipe()); found {
		kind = asked.Kind()
	}
	return kind == manufacture.VillageKind ||
		kind == manufacture.VillageHouseKind
}

// groundedThing stands a thing on terrain whose kind stands each of its
// houses on the ground under its own plot, a village, made for the ground
// under the place, and says whether it did. Any other thing is left to be
// stood as it always was.
func (stage *staging) groundedThing(each set.Thing, index uint64,
	place maths.Vec) (bool, error) {
	if !stage.villagey(each) {
		return false, nil
	}
	row, err := stage.construction(each.Kind(), each.Recipe())
	if err != nil || !manufacture.StandsOnTerrain(*row) {
		return false, nil
	}
	turn := stage.placementTurn(each.Kind(), each.Name(), each.Seed(), index)
	grounded, err := manufacture.Grounded(*row,
		stage.groundUnder(place, turn))
	if err != nil {
		return true, err
	}
	at := maths.Subspace{Origin: place,
		Orientation: maths.About(maths.YAxis, turn)}
	one, err := stage.grounded(grounded, each.Seed(), each.Complexity(), at)
	if err != nil {
		return true, err
	}
	stage.radii[each.Name()] = horizontalRadius(one.bounds)
	return true, stage.add(one, each.Kind(), at, each.Seed(), each.Alight())
}
