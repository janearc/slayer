package manufacture

import "github.com/janearc/celestial/measure"

// pyre is the row of a bonfire's wood: how many logs lean together, how
// far out their feet stand, in metres, how far each leans in from
// upright, and what logs. it carries no light of its own. a set lights
// it with alight, and its flames are the fire physics', from its size.
type pyre struct {
	logs, ring span
	lean       measure.Angle
	log        stick
}

// pyreRow is the bonfire, a very big one, big enough to throw
// its light into the trees about a clearing. made up.
//
// eleven to fifteen logs of 6 to 8 m and 30 to 45 cm through, their feet
// 2.4 to 3 m out, each leaning 21 degrees in, so their tops cross over
// the middle about 7 m up and its flames reach about 10.
func pyreRow() pyre {
	return pyre{logs: span{11, 15}, ring: span{2.4, 3},
		lean: measure.Degrees(21),
		log: stick{long: span{6, 8}, thick: span{0.30, 0.45},
			taper: 0.85, around: 8,
			dressed: looks{body: plain("bark", 0.24, 0.16, 0.1)}}}
}

// made is a bonfire from a seed: its logs leaned together over the
// middle.
func (row pyre) made(from seed) (good, error) {
	logs, err := newRing(row.logs.count(from.draw(picking, 0, 0)),
		row.ring.metres(from.draw(picking, 1, 0)), row.lean, sticksStray,
		sticksUneven, sticksTwist)
	if err != nil {
		return good{}, err
	}
	return around(logs, row.log, from.sprout(sticking, 0))
}
