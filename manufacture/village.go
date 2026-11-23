package manufacture

import (
	"fmt"

	"github.com/janearc/slayer/manufacture/cities"
	"github.com/janearc/slayer/maths"
)

// The housing of an urban village, drawn by package cities and made a
// good here, where a material becomes a slot. A village-house is one
// house of it, standing alone on a lane; a village is a parcel packed with
// them, alleys between and lanes every so many plots, with cables over
// both. Each house is levelled as a building is, and each stands on the
// ground under its own plot.

// The names of the kinds of an urban village.
const (
	// VillageHouseKind is one house of a village.
	VillageHouseKind = "village-house"
	// VillageKind is a parcel packed with houses.
	VillageKind = "village"
)

// VillageSpec is what a set says of a village: the width of its parcel
// along east and its depth along north, in metres, nought for the
// culture's own.
type VillageSpec struct {
	Width, Depth float64
}

// villageHouseRow is the kind village-house, stood on a ground if it is
// given one.
type villageHouseRow struct{ ground cities.Terrain }

// villageRow is the kind village, with the parcel a set asks for, stood
// on a ground if it is given one.
type villageRow struct {
	spec   VillageSpec
	ground cities.Terrain
}

// Village is the kind village with the parcel a set asks for. It refuses
// a parcel below nought, and draws the village of seed nought to refuse a
// parcel that holds no plot.
func Village(spec VillageSpec) (Kind, error) {
	if spec.Width < 0 || spec.Depth < 0 {
		return Kind{}, fmt.Errorf("manufacture: a village's parcel is not "+
			"below nought, not %v by %v m", spec.Width, spec.Depth)
	}
	row := villageRow{spec: spec}
	if _, err := row.drawn(sow(0), "hull"); err != nil {
		return Kind{}, err
	}
	return Kind{name: VillageKind, how: row}, nil
}

// made is the house at the quick detail level.
func (row villageHouseRow) made(from seed) (good, error) {
	return row.madeAt(from, "quick")
}

// madeAt is the house at the named detail level. It returns errGood when
// the seed is not made.
func (row villageHouseRow) madeAt(from seed, detail string) (good, error) {
	return row.madeLevelled(from, constantPass(detail), maths.Subspace{})
}

// madeLevelled is the house drawn by a pass, standing at a place in the
// world the pass sees, made at the pass's finest level.
func (row villageHouseRow) madeLevelled(from seed, pass cities.Pass,
	place maths.Subspace) (good, error) {
	if !from.made() {
		return good{}, errGood
	}
	design, err := cities.VillageHouseAt(cities.Seed(from.number),
		row.ground, pass.Finest)
	if err != nil {
		return good{}, err
	}
	return levelledGood(design, pass, place)
}

// made is the village at the quick detail level.
func (row villageRow) made(from seed) (good, error) {
	return row.madeAt(from, "quick")
}

// madeAt is the village at the named detail level. It returns errGood
// when the seed is not made.
func (row villageRow) madeAt(from seed, detail string) (good, error) {
	return row.madeLevelled(from, constantPass(detail), maths.Subspace{})
}

// madeLevelled is the village drawn by a pass, each house at the level
// the pass gives it where it stands, made at the pass's finest level.
func (row villageRow) madeLevelled(from seed, pass cities.Pass,
	place maths.Subspace) (good, error) {
	if !from.made() {
		return good{}, errGood
	}
	village, err := row.drawn(from, pass.Finest)
	if err != nil {
		return good{}, err
	}
	return levelledGood(village.Design(), pass, place)
}

// drawn is the village of a seed made at a level, before it is a good.
func (row villageRow) drawn(from seed, level cities.Detail) (cities.Village,
	error) {
	return cities.NewVillage(cities.VillageAsk{Width: row.spec.Width,
		Depth: row.spec.Depth, Level: level}, cities.Seed(from.number),
		row.ground)
}

// levelledGood is a design drawn by a pass at a place, made a good.
func levelledGood(design cities.Design, pass cities.Pass,
	place maths.Subspace) (good, error) {
	drawn, err := design.Levelled(pass, place)
	if err != nil {
		return good{}, err
	}
	return goodOfRendered(drawn)
}

// StandsOnTerrain is whether a thing of a kind is stood on terrain each
// of its houses on the ground under its own plot, and not as one flat
// thing: a village and a village house are. The terraces and the
// condominium are stood so only in a stand, as they always were.
func StandsOnTerrain(row Kind) bool {
	switch row.how.(type) {
	case villageHouseRow, villageRow:
		return true
	}
	return false
}
