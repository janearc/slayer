package show

import "math"

// Rig is the laser fans' level, 0 off to 1 full, as its fade stands.
func (made Moment) Rig() float64 { return made.now.rig.at(made.seconds) }

// Lasers is how bright the fans are this frame: the rig's level struck
// by the pulse, the brighter bank's.
func (made Moment) Lasers() float64 {
	return made.Rig() * math.Max(made.gate, made.gateRight)
}

// Strobe is whether a strobe fires this frame, white or magenta. The
// lattice, which lightshow drives on its own clock, is not counted.
func (made Moment) Strobe() bool { return made.white == 1 || made.magenta == 1 }

// Floods is the brighter flood's flash this frame, 0 to 1.
func (made Moment) Floods() float64 { return math.Max(made.floodLeft, made.floodRight) }

// Searchlights is how bright the searchlights are this frame: their
// level struck by the beat.
func (made Moment) Searchlights() float64 { return made.now.search * made.searchGate }

// Moon is the moon's disc this frame, 0 to 1, its pulse counted; nought
// while it stands behind the rim. The scene's dimmer is not counted:
// the disc is its own cue.
func (made Moment) Moon() float64 {
	if made.now.moonAlt.at(made.seconds) <= rimMoon {
		return 0
	}
	return made.moon
}

// master is the moonlit scene's dimmer, 0 to 1.
func (made Moment) master() float64 { return made.now.master.at(made.seconds) }

// Overhead is the ship's beam, 0 to 1 of daylight.
func (made Moment) Overhead() float64 { return made.now.overhead.at(made.seconds) }

// Colour is the beams' colour this beat, from the palette, as linear
// srgb from 0 to 1.
func (made Moment) Colour() [3]float64 { return made.colour }

// Palettes is every palette colour the sheet's cues name, each once, in
// the order they first appear.
func (of *Sheet) Palettes() []string {
	var names []string
	seen := map[string]bool{}
	for _, each := range of.cues {
		for _, name := range each.line.Palette {
			if !seen[name] {
				seen[name] = true
				names = append(names, name)
			}
		}
	}
	return names
}

// PaletteColour is a palette name as linear srgb from 0 to 1, and
// whether lightshow knows it.
func PaletteColour(name string) ([3]float64, bool) {
	colour, ok := paletteColours[name]
	return colour, ok
}
