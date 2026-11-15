package manufacture

import (
	"github.com/janearc/celestial/measure"
)

// the stage lights: the lasers and the strobe a forest show hides
// behind its trees. a laser is a small snout on a stand shining one
// line, or more than one, through a narrow opening; the strobe is a
// spotlight's cousin, wide, white and cold. like every row, their
// numbers are made up to sound right; see kinds.go.
//
// the wavelengths are the usual show modules' lines, from memory and
// not looked up: solid-state green at 532 nm, red diodes at 638,
// blue at 445, the argon line at 488 for cyan, and amber at 589,
// which is sodium's line. a magenta or a white head mixes lines from
// one mouth, as the real projectors do.
const (
	greenLine, redLine, blueLine = 532, 638, 445
	cyanLine, amberLine          = 488, 589
)

// nmLine is a wavelength in nanometres as a length.
func nmLine(nano float64) measure.Length {
	return measure.Metres(nano / nanometres)
}

// ray is one line a laser head shines: its wavelength, and its share
// of the head's power.
type ray struct {
	wave  measure.Length
	share float64
}

// beamhead is the row of a stage laser: how tall its stand, in
// metres, how far up its snout tips, in degrees, how strong it is,
// in watts, split over its lines, how thick its stand and how wide
// its opening, in metres, the snout it is made of, and its lines.
type beamhead struct {
	stand, lift, power span
	thick, bore        measure.Length
	head               horn
	around             int
	dressed            looks
	lines              []ray
}

// laserRow is a stage laser of some lines: a slim snout about a
// metre up, tipped level or a little skyward, its beam as narrow as
// its opening allows.
func laserRow(lines []ray) beamhead {
	return beamhead{stand: span{0.8, 1.2}, lift: span{0, 12},
		power: span{2, 8}, thick: measure.Metres(0.02),
		bore: measure.Metres(0.003),
		head: horn{long: measure.Metres(0.25),
			back:  measure.Metres(0.05),
			front: measure.Metres(0.05)},
		around:  10,
		dressed: looks{body: plain("black housing", 0.04, 0.04, 0.045)},
		lines:   lines}
}

// made is a stage laser from a seed: a snout on a stand, and each of
// its lines out of the same mouth, sharing one power.
func (row beamhead) made(from seed) (good, error) {
	stand := stalk{tall: row.stand.metres(from.draw(picking, 0, 0)),
		thick: row.thick}
	lamp, err := headed(stand, row.head,
		row.lift.degrees(from.draw(picking, 1, 0)), row.around,
		row.dressed)
	if err != nil {
		return good{}, err
	}
	power := row.power.watts(from.draw(picking, 2, 0))
	made := lamp.whole
	for _, line := range row.lines {
		colour, colourError := newLine(line.wave)
		through, boreError := newAperture(row.bore)
		if err := firstOf(colourError, boreError); err != nil {
			return good{}, err
		}
		shining, err := newLight(colour,
			measure.Watts(power.InWatts()*line.share), through)
		if err != nil {
			return good{}, err
		}
		made = made.lit(shining, lamp.mouth, lamp.way, row.head.front)
	}
	return made, nil
}

// firstOf is the first of the errors that is one.
func firstOf(these ...error) error {
	for _, err := range these {
		if err != nil {
			return err
		}
	}
	return nil
}

// strobeRow is the strobe: a spotlight's build with a short wide
// head, a cold white lamp, and a broad soft-edged wash. its flashing
// is not here; a light's timing belongs to the scene, not the
// manufactory.
func strobeRow() spot {
	return spot{stand: span{1.2, 1.8}, tip: span{0, 10},
		power: span{400, 1600}, half: span{35, 55},
		heat: measure.Kelvin(6500), soft: measure.Degrees(15),
		thick: measure.Metres(0.025),
		head: horn{long: measure.Metres(0.12),
			back:  measure.Metres(0.09),
			front: measure.Metres(0.11)},
		around:  12,
		dressed: looks{body: plain("black housing", 0.04, 0.04, 0.045)}}
}

// third is an even share of three lines.
const third = 1.0 / 3

// stageRows is the stage lights by name: five one-line lasers, the
// two mixed heads, and the strobe; and the scaffold they stand on,
// bare and as a show's tower with its gear (scaffold.go).
func stageRows() map[string]maker {
	one := func(nano float64) []ray {
		return []ray{{wave: nmLine(nano), share: 1}}
	}
	return map[string]maker{
		"laser-green": laserRow(one(greenLine)),
		"laser-red":   laserRow(one(redLine)),
		"laser-blue":  laserRow(one(blueLine)),
		"laser-cyan":  laserRow(one(cyanLine)),
		"laser-amber": laserRow(one(amberLine)),
		"laser-magenta": laserRow([]ray{
			{wave: nmLine(redLine), share: 0.5},
			{wave: nmLine(blueLine), share: 0.5}}),
		"laser-white": laserRow([]ray{
			{wave: nmLine(redLine), share: third},
			{wave: nmLine(greenLine), share: third},
			{wave: nmLine(blueLine), share: third}}),
		"strobe":         strobeRow(),
		"stage-scaffold": scaffoldRow(),
		"stage-tower":    stageTowerRow(),
	}
}
