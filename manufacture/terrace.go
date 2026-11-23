package manufacture

import (
	"github.com/janearc/slayer/manufacture/cities"
	"github.com/janearc/slayer/maths"
)

// terraceRow is a kind of row of houses: a terrace drawn by package
// cities from one culture, and made a good here, which is where a
// material becomes a slot. A set names the culture by the kind it asks
// for, terrace-groningen, terrace-portofino or terrace-midwest, because
// the kind and the seed are all a set hands the manufactory.
type terraceRow struct {
	culture  string
	ground   cities.Terrain
	founding cities.Founding
}

// made is the terrace at the quick detail level.
func (row terraceRow) made(from seed) (good, error) {
	return row.madeAt(from, "quick")
}

// madeAt is the terrace at the named detail level. It returns errGood
// when the seed is not made.
func (row terraceRow) madeAt(from seed, detail string) (good, error) {
	return row.madeLevelled(from, constantPass(detail), maths.Subspace{})
}

// madeLevelled is the terrace drawn by a pass, standing at a place in the
// world the pass sees: each bay, face, storey and window at the level the
// pass gives it. The row is drawn as it is at the pass's finest level.
func (row terraceRow) madeLevelled(from seed, pass cities.Pass,
	place maths.Subspace) (good, error) {
	if !from.made() {
		return good{}, errGood
	}
	design, err := cities.TerraceFounded(row.culture,
		cities.Seed(from.number), row.ground, pass.Finest, row.founding)
	if err != nil {
		return good{}, err
	}
	drawn, err := design.Levelled(pass, place)
	if err != nil {
		return good{}, err
	}
	return goodOfRendered(drawn)
}

// condominiumRow is the kind condominium: two to four buildings of
// stacked flats round a shared court, drawn by package cities and made a
// good here as the terraces are.
type condominiumRow struct{ ground cities.Terrain }

// made is the condominium at the quick detail level.
func (row condominiumRow) made(from seed) (good, error) {
	return row.madeAt(from, "quick")
}

// madeAt is the condominium at the named detail level. It returns
// errGood when the seed is not made.
func (row condominiumRow) madeAt(from seed, detail string) (good, error) {
	if !from.made() {
		return good{}, errGood
	}
	design, err := cities.CondominiumAt(cities.Seed(from.number), row.ground,
		cities.Detail(detail))
	if err != nil {
		return good{}, err
	}
	drawn, err := design.Rendered(cities.Detail(detail))
	if err != nil {
		return good{}, err
	}
	made, err := goodOfRendered(drawn)
	if err != nil {
		return good{}, err
	}
	return withMounts(made, design.Mounts, from, detail)
}
