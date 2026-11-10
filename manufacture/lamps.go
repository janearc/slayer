package manufacture

import (
	"cmp"
	"math"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
)

// the kinds that carry a light: a campfire, a spotlight, a maser. each
// is things gathered into one, with a light's description put on it.
// like every row, their numbers are made up to sound right; see
// kinds.go.

// fire is the row of a campfire: how many stones ring it and how far out, in
// metres, what stones they are,
//
// how many sticks lean over it and what sticks, where the sticks stand as a
// share of the ring and how far they lean in, how hot it burns, in kelvin, how
// strongly, in watts, how high above the ground, how far out from its middle
// its light starts, past the embers it sits in, and how deep it flickers.
type fire struct {
	stones, ring, sticks, heat, power span
	stone                             stone
	stick                             stick
	stand, flicker                    float64
	lean                              measure.Angle
	high, wide                        measure.Length
}

// fireRow is the campfire.
func fireRow() fire {
	wood := looks{body: plain("charred wood", 0.12, 0.08, 0.06)}
	return fire{stones: span{7, 10}, ring: span{0.35, 0.5},
		sticks: span{4, 6}, heat: span{1100, 1500},
		power: span{2000, 6000}, stone: stoneRows()["cobble"],
		stick: stick{long: span{0.5, 0.75}, thick: span{0.025, 0.04},
			taper: 0.6, around: 6, dressed: wood},
		stand: 0.55, flicker: 0.35, lean: measure.Degrees(55),
		high: measure.Metres(0.2), wide: measure.Metres(0.06)}
}

// how the stones of a ring stray, differ in size and are twisted: a
// little, a little, and any way round. made up.
const (
	stonesStray, stonesUneven, stonesTwist = 0.3, 0.3, 1
	sticksStray, sticksUneven, sticksTwist = 0.2, 0.2, 1
)

// rings is the two rings a fire is laid in: its stones, and inside
// them its sticks, leaning in.
func (row fire) rings(from seed) (ring, ring, error) {
	out := row.ring.metres(from.draw(picking, 0, 0))
	stones, err := newRing(row.stones.count(from.draw(picking, 1, 0)),
		out, measure.Degrees(0), stonesStray, stonesUneven, stonesTwist)
	if err != nil {
		return ring{}, ring{}, err
	}
	sticks, err := newRing(row.sticks.count(from.draw(picking, 2, 0)),
		out.Times(row.stand), row.lean, sticksStray, sticksUneven,
		sticksTwist)
	return stones, sticks, err
}

// flame is the fire's light: a glow every way, flickering.
func (row fire) flame(from seed) (light, error) {
	glow, err := newThermal(row.heat.kelvin(from.draw(picking, 3, 0)))
	if err != nil {
		return light{}, err
	}
	burning, err := newLight(glow,
		row.power.watts(from.draw(picking, 4, 0)), everyway{})
	if err != nil {
		return light{}, err
	}
	return burning.flickering(from.sprout(flaring, 0), row.flicker)
}

// made is a campfire from a seed: stones in a ring, sticks leaned in
// over the middle, and a fire that glows every way and flickers.
func (row fire) made(from seed) (good, error) {
	stones, sticks, err := row.rings(from)
	if err != nil {
		return good{}, err
	}
	ringed, err := around(stones, row.stone, from.sprout(stoning, 0))
	if err != nil {
		return good{}, err
	}
	leaned, err := around(sticks, row.stick, from.sprout(sticking, 0))
	if err != nil {
		return good{}, err
	}
	flame, err := row.flame(from)
	if err != nil {
		return good{}, err
	}
	laid, err := gather(ringed, leaned)
	if err != nil {
		return good{}, err
	}
	return laid.lit(flame, maths.V(0, inScene(row.high), 0), maths.Up,
		row.wide), nil
}

// around is one of a maker's things at every place of a ring, each
// from a sprout of the seed, gathered into one thing of one matter.
func around(where ring, of maker, from seed) (good, error) {
	var all good
	for index := range where.across(0) {
		there := where.place(0, index, from)
		one, err := of.made(from.sprout(index, 0))
		if err != nil {
			return good{}, err
		}
		placed, err := one.placed(there.space)
		if err != nil {
			return good{}, err
		}
		all.parts = append(all.parts, placed.parts...)
		all.dressed = one.dressed
	}
	return all, nil
}

// stick is the row of a stick for a fire: how long and how thick at
// the foot, in metres, how much of that it keeps to its tip, and how
// finely it is cut. a short tapering tube, standing.
type stick struct {
	long, thick span
	taper       float64
	around      int
	dressed     looks
}

// made is a stick from a seed.
func (row stick) made(from seed) (good, error) {
	girth := row.thick.metres(from.draw(picking, 1, 0))
	shape, err := newTube(row.long.metres(from.draw(picking, 0, 0)),
		girth, girth.Times(row.taper))
	if err != nil {
		return good{}, err
	}
	return shaped(shape, row.around, row.dressed)
}

// stalk is the stand a head sits on: how tall and how thick.
type stalk struct{ tall, thick measure.Length }

// horn is a head that widens toward its front: how long, and its
// radius at the back and at the front.
type horn struct{ long, back, front measure.Length }

// mounted is a head on a stand, and where a light in it goes: the
// middle of the head's front, and the way the head points.
type mounted struct {
	whole      good
	mouth, way maths.Vec
}

// headed is a stand with a head on it: a thin upright tube carrying a tube that
// widens toward its front, tipped so far up from level, or down, less than
// zero.
func headed(stand stalk, head horn, tilt measure.Angle, around int,
	dressed looks) (mounted, error) {
	post, err := newTube(stand.tall, stand.thick, stand.thick)
	if err != nil {
		return mounted{}, err
	}
	bell, err := newTube(head.long, head.back, head.front)
	if err != nil {
		return mounted{}, err
	}
	var pieces [2]good
	for index, shape := range []tube{post, bell} {
		made, err := shaped(shape, around, dressed)
		if err != nil {
			return mounted{}, err
		}
		pieces[index] = made
	}
	stood, turned := pieces[0], pieces[1]
	top := maths.V(0, inScene(stand.tall), 0)
	up := tilt.InRadians()
	way := maths.V(math.Cos(up), math.Sin(up), 0)
	// the way is a unit long whatever the tilt, so it is a way to go:
	// the error is spaceAlong's, for a run that has none
	space, spaceError := spaceAlong(top, way)
	placed, placementError := turned.placed(space)
	if err := cmp.Or(spaceError, placementError); err != nil {
		return mounted{}, err
	}
	stood.parts = append(stood.parts, placed.parts...)
	return mounted{whole: stood,
		mouth: top.Add(way.Scale(inScene(head.long))), way: way}, nil
}

// spot is the row of a spotlight: how tall its stand, in metres, how
// far its head tips down, in degrees, how strong its lamp, in watts,
// how wide its beam, in degrees, how hot its lamp, the soft edge of
// its beam, and the stand and the head it is made of.
type spot struct {
	stand, tip, power, half span
	heat                    measure.Temperature
	soft                    measure.Angle
	thick                   measure.Length
	head                    horn
	around                  int
	dressed                 looks
}

// spotRow is the spotlight.
func spotRow() spot {
	return spot{stand: span{1, 1.6}, tip: span{10, 25},
		power: span{500, 2000}, half: span{12, 25},
		heat: measure.Kelvin(3200), soft: measure.Degrees(6),
		thick: measure.Metres(0.02),
		head: horn{long: measure.Metres(0.3),
			back:  measure.Metres(0.07),
			front: measure.Metres(0.12)},
		around: 12,
		dressed: looks{body: &thing.Material{Name: "black iron",
			Colour: maths.Grey(0.08), Gloss: 0.2}}}
}

// made is a spotlight from a seed.
func (row spot) made(from seed) (good, error) {
	down := row.tip.degrees(from.draw(picking, 1, 0)).InRadians()
	stand := stalk{tall: row.stand.metres(from.draw(picking, 0, 0)),
		thick: row.thick}
	lamp, err := headed(stand, row.head, measure.Radians(-down),
		row.around, row.dressed)
	if err != nil {
		return good{}, err
	}
	glow, err := newThermal(row.heat)
	if err != nil {
		return good{}, err
	}
	beam, err := newCone(row.half.degrees(from.draw(picking, 2, 0)),
		row.soft)
	if err != nil {
		return good{}, err
	}
	shining, err := newLight(glow,
		row.power.watts(from.draw(picking, 3, 0)), beam)
	if err != nil {
		return good{}, err
	}
	return lamp.whole.lit(shining, lamp.mouth, lamp.way,
		row.head.front), nil
}

// maser is the row of a maser: how tall its mount and how far across
// its dish, in metres, how high it points, in degrees, how strong it
// is, in watts, how thick its mount, how deep its dish and how wide
// its back as shares of its width, and the line it shines in.
type maser struct {
	mount, dish, lift, power span
	thick                    measure.Length
	deep, back               float64
	wave                     measure.Length
	around                   int
	dressed                  looks
}

// hydrogen's line, from memory and not looked up: 1420.4 MHz, which
// at the speed of light, exact by the metre's own definition, is 21
// cm.
const (
	hydrogenHertz = 1420.4e6
	lightSpeed    = 299792458
)

// maserRow is the maser.
func maserRow() maser {
	return maser{mount: span{1.5, 2.5}, dish: span{2, 4},
		lift: span{30, 70}, power: span{0.5, 5},
		thick: measure.Metres(0.06), deep: 0.25, back: 0.1,
		wave:    measure.Metres(lightSpeed / hydrogenHertz),
		around:  16,
		dressed: looks{body: plain("white paint", 0.85, 0.85, 0.82)}}
}

// made is a maser from a seed: a dish on a mount, and its line out of
// the dish as narrow as the dish allows.
func (row maser) made(from seed) (good, error) {
	across := row.dish.metres(from.draw(picking, 1, 0))
	stand := stalk{tall: row.mount.metres(from.draw(picking, 0, 0)),
		thick: row.thick}
	dish := horn{long: across.Times(row.deep),
		back: across.Times(row.back), front: across.Times(midway)}
	lamp, headError := headed(stand, dish,
		row.lift.degrees(from.draw(picking, 2, 0)), row.around,
		row.dressed)
	line, lineError := newLine(row.wave)
	through, apertureError := newAperture(across)
	if err := cmp.Or(headError, lineError, apertureError); err != nil {
		return good{}, err
	}
	shining, err := newLight(line,
		row.power.watts(from.draw(picking, 3, 0)), through)
	if err != nil {
		return good{}, err
	}
	return lamp.whole.lit(shining, lamp.mouth, lamp.way, dish.front), nil
}
