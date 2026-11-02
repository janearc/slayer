package manufacture

import (
	"errors"
	"testing"

	"github.com/janearc/celestial/measure"
)

// a row is data, and data can be wrong. whatever a wrong row asks
// for, the door that makes the piece refuses it, and the refusal
// comes back from the row as the error it was: nothing is made of a
// row with a piece missing. the design promises every refusal tested
// by its error; hippo's review found the rows' own were not.
func TestAWrongRowIsRefusedByThePieceItGotWrong(t *testing.T) {
	none, below := span{0, 0}, span{-1, -1}
	cobble := stoneRows()["cobble"]
	wrong := map[string]struct {
		row  maker
		want error
	}{}
	add := func(name string, row maker, want error) {
		wrong[name] = struct {
			row  maker
			want error
		}{row, want}
	}

	stone := cobble
	stone.long = below
	add("a stone of a length below nought", stone, errLump)
	stone = cobble
	stone.around = 2
	add("a stone cut two round", stone, errCut)

	spruce := spruceRow()
	spruce.lowest = none
	add("a spruce with no branch in a whorl", spruce, errWhorls)
	spruce = spruceRow()
	spruce.hangs.reach = 0
	add("a spruce whose branches reach nowhere", spruce, errHabit)
	spruce = spruceRow()
	spruce.slender = none
	add("a spruce with no girth", spruce, errTubeFoot)
	norway := norwayRow()
	norway.lowest = none
	add("a norway spruce with no branch in a whorl", norway, errWhorls)
	norway = norwayRow()
	norway.grows.shoots.per = none
	add("a norway spruce with no branchlets", norway, errWhorls)
	norway = norwayRow()
	norway.slender = none
	add("a norway spruce with no girth", norway, errTubeFoot)
	norway = norwayRow()
	norway.grows.trunk.sides = 2
	add("a norway spruce whose trunk is cut two round", norway, errCut)
	norway = norwayRow()
	norway.grows.bark.sides = 2
	add("a norway spruce whose bark is cut two round", norway, errCut)
	norway = norwayRow()
	norway.grows.clad = sleeveFoliage{joints: lengths(0.05, 0.07),
		sides: 2}
	add("a norway spruce whose branches are clad two round", norway,
		errCut)
	norway = norwayRow()
	norway.grows.shoots.wear = sleeveFoliage{joints: lengths(0.12, 0.2),
		sides: 2}
	add("a norway spruce whose branchlets wear two round", norway, errCut)

	fern := fernRow()
	fern.fronds = none
	add("a fern of no fronds", fern, errRing)
	fern = fernRow()
	fern.leaflets = none
	add("a fern of no leaflets", fern, errWhorls)
	fern = fernRow()
	fern.long = below
	add("a fern of fronds below nought", fern, errTubeLength)

	can := canRow()
	can.tall = measure.Metres(0)
	add("a can of no height", can, errTubeLength)
	can = canRow()
	can.around = 2
	add("a can cut two round", can, errCut)

	log := logRow()
	log.stubs = none
	add("a log of no stubs", log, errWhorls)
	log = logRow()
	log.long = below
	add("a log of a length below nought", log, errTubeLength)
	log = logRow()
	log.around = 2
	add("a log cut two round", log, errCut)
	log = logRow()
	log.stubAround = 2
	add("a log whose stubs are cut two round", log, errCut)

	stick := fireRow().stick
	stick.long = below
	add("a stick of a length below nought", stick, errTubeLength)
	stick = fireRow().stick
	stick.around = 2
	add("a stick cut two round", stick, errCut)

	fire := fireRow()
	fire.stones = none
	add("a fire of no stones", fire, errRing)
	fire = fireRow()
	fire.sticks = none
	add("a fire of no sticks", fire, errRing)
	fire = fireRow()
	fire.stone.long = below
	add("a fire of stones below nought", fire, errLump)
	fire = fireRow()
	fire.stick.around = 2
	add("a fire of sticks cut two round", fire, errCut)
	fire = fireRow()
	fire.stick.dressed = nil
	add("a fire of sticks of no look", fire, errLook)
	fire = fireRow()
	fire.heat = none
	add("a fire of no heat", fire, errLight)
	fire = fireRow()
	fire.power = none
	add("a fire of no power", fire, errLight)
	fire = fireRow()
	fire.flicker = 2
	add("a fire flickering past dark", fire, errLight)

	spot := spotRow()
	spot.stand = below
	add("a spotlight on a stand below nought", spot, errTubeLength)
	spot = spotRow()
	spot.head.back = measure.Metres(0)
	add("a spotlight with no back to its head", spot, errTubeFoot)
	spot = spotRow()
	spot.around = 2
	add("a spotlight cut two round", spot, errCut)
	spot = spotRow()
	spot.heat = measure.Kelvin(0)
	add("a spotlight of no heat", spot, errLight)
	spot = spotRow()
	spot.half = span{-5, -5}
	add("a spotlight with a beam below nought", spot, errLight)
	spot = spotRow()
	spot.power = none
	add("a spotlight of no power", spot, errLight)

	maser := maserRow()
	maser.mount = below
	add("a maser on a mount below nought", maser, errTubeLength)
	maser = maserRow()
	maser.back = 0
	add("a maser with no back to its dish", maser, errTubeFoot)
	maser = maserRow()
	maser.around = 2
	add("a maser cut two round", maser, errCut)
	maser = maserRow()
	maser.wave = measure.Metres(0)
	add("a maser of no line", maser, errLight)
	maser = maserRow()
	maser.power = none
	add("a maser of no power", maser, errLight)

	for name, bad := range wrong {
		made, err := bad.row.made(sow(1))
		if !errors.Is(err, bad.want) {
			t.Errorf("%s: %v, not %v", name, err, bad.want)
		}
		if len(made.parts) != 0 || len(made.lights) != 0 {
			t.Errorf("%s: refused, and made %d parts all the same",
				name, len(made.parts))
		}
	}
	// and the rows as they are make things, so the table above is of
	// wrong rows and not of a door that refuses everything
	for name, row := range table() {
		if _, err := row.made(sow(1)); err != nil {
			t.Errorf("%s as it is: %v", name, err)
		}
	}
}
