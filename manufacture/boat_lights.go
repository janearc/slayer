package manufacture

import (
	"fmt"
	"math"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/maths"
)

// The lights of a boat. Under way it shows a red light on its port side,
// a green one to starboard and a white one at its stern, and, a power
// vessel, a white one at its masthead; at anchor, one white light all
// round. Each is a real lamp, a cone of the arc the data gives pointed
// the way the light looks, with a lens that glows where the light is, and
// a screen behind the lens so that it is seen only from where it shines.

// boatMounts is where a boat's lights stand, in its own space, which
// its fittings decide: the side lights, the stern lamp, the top light
// on the mast, whether it is a power vessel, and the working lights.
type boatMounts struct {
	port, starboard, stern, top maths.Vec
	power                       bool
	working                     []boatWorking
}

// boatWorking is a working light: the lamp by its name in the data, the
// lens its glow is seen in, where it stands and the way it looks.
type boatWorking struct {
	lamp       string
	lens       slot
	at, toward maths.Vec
}

// boatShown is a lamp the boat shows: its name, which a scene may
// address it by, the lamp by its name in the data, where and which way.
type boatShown struct {
	name, lamp string
	at, toward maths.Vec
}

// The lens and its screen, in metres: the lens is a cube; a screen is a
// board of this thickness, this tall and this long, standing this far
// inboard of it and this far aft of it. All chosen, small and just
// enough to hide a lens from the side it does not shine to.
const (
	lensSize    = 0.16
	screenThick = 0.03
	screenTall  = 0.3
	screenLong  = 0.5
	screenWide  = 0.2
	screenInset = 0.09
	screenAft   = 0.1
)

// lens is a lens of a glow at a place, a cube.
func (craft *boat) lens(at maths.Vec, material slot) {
	craft.block(at.X, at.Y-lensSize*spaceHalf, at.Z, lensSize, lensSize,
		lensSize, material)
}

// screen is a board of the lamp's housing at a place, so wide across,
// tall, and so long along the boat.
func (craft *boat) screen(across, up, along, wide, long float64) {
	craft.block(across, up-screenTall*spaceHalf, along, wide, screenTall, long,
		boatSteel)
}

// sidelight is a side light's lens, with a board inboard of it and one
// aft of it, so that it is not seen from the other side or from astern.
func (craft *boat) sidelight(at maths.Vec, turn float64, material slot) {
	craft.lens(at, material)
	craft.screen(at.X-turn*screenInset, at.Y, at.Z+screenAft, screenThick,
		screenLong)
	craft.screen(at.X-turn*screenThick, at.Y, at.Z+screenAft, screenWide,
		screenThick)
}

// navigation puts the housings and lenses of the lights the boat shows
// into its assembly: the side lights, stern lamp and top light of a boat
// under way, the all-round light of one at anchor, and the working lights.
func (craft *boat) navigation() {
	mounts := craft.mounts
	if craft.row.anchored {
		craft.lens(mounts.top, boatWhiteLens)
	} else {
		craft.sidelight(mounts.port, -spaceOne, boatRedLens)
		craft.sidelight(mounts.starboard, spaceOne, boatGreenLens)
		craft.lens(mounts.stern, boatWhiteLens)
		craft.screen(mounts.stern.X, mounts.stern.Y, mounts.stern.Z-screenAft,
			screenWide, screenThick)
		if mounts.power {
			craft.lens(mounts.top, boatWhiteLens)
			craft.screen(mounts.top.X, mounts.top.Y, mounts.top.Z+screenAft,
				screenWide, screenThick)
		}
	}
	for _, work := range mounts.working {
		craft.lens(work.at, work.lens)
	}
}

// sideward is the way a side light looks: the middle of its arc, which
// is half the arc from dead ahead, to its own side.
func (craft *boat) sideward(lamp string, turn float64) maths.Vec {
	half := boatRead.Lamps[lamp].Arc * spaceHalf * maths.Degree
	return maths.V(turn*math.Sin(half), 0, -math.Cos(half))
}

// shown is every lamp the boat shows, by its state.
func (craft *boat) shown() []boatShown {
	mounts := craft.mounts
	var shown []boatShown
	if craft.row.anchored {
		shown = append(shown, boatShown{"anchor light", "anchor", mounts.top,
			maths.Up})
	} else {
		shown = append(shown,
			boatShown{"port light", "port", mounts.port,
				craft.sideward("port", -spaceOne)},
			boatShown{"starboard light", "starboard", mounts.starboard,
				craft.sideward("starboard", spaceOne)},
			boatShown{"stern light", "stern", mounts.stern,
				maths.V(0, 0, spaceOne)})
		if mounts.power {
			shown = append(shown, boatShown{"masthead light", "masthead",
				mounts.top, maths.V(0, 0, -spaceOne)})
		}
	}
	for index, work := range mounts.working {
		shown = append(shown, boatShown{fmt.Sprintf("%s light %d",
			work.lamp, index+spaceOne), work.lamp, work.at, work.toward})
	}
	return shown
}

// lit is the boat with its lamps: each light a lamp of the data, at its
// place, looking its way, shining from past its own housing.
func (craft *boat) lit(made good) (good, error) {
	for _, one := range craft.shown() {
		lamp, err := boatRead.lamp(one.lamp)
		if err != nil {
			return good{}, err
		}
		made = made.litAs(one.name, lamp, one.at, one.toward,
			measure.Metres(boatRead.Lamps[one.lamp].Beyond))
	}
	return made, nil
}

// lamp is the light the data gives a lamp: of its tint or its heat, its
// luminance at a metre in the kit's relative light, kept to a cone of its
// arc unless it shines all round.
func (sizes boatSizes) lamp(name string) (light, error) {
	data := sizes.Lamps[name]
	colour, scale, err := data.spectrum()
	if err != nil {
		return light{}, err
	}
	var over spread = everyway{}
	if data.Arc < boatTurn {
		over, err = newCone(measure.Degrees(data.Arc*spaceHalf),
			measure.Degrees(data.Soft))
		if err != nil {
			return light{}, err
		}
	}
	made, err := newLight(colour, measure.Watts(data.Luminance*scale), over)
	made.luminance = data.Luminance
	return made, err
}

// spectrum is what a lamp's light is made of, and how many watts of it
// make a unit of luminance: a tint's own, or one for a heat's.
func (data boatLamp) spectrum() (spectrum, float64, error) {
	if len(data.Tint) == alfaChannels {
		of, err := newTint(maths.RGB(data.Tint[0], data.Tint[1], data.Tint[2]))
		if err != nil {
			return spectrum{}, 0, err
		}
		return of, of.tint.watts, nil
	}
	of, err := newThermal(measure.Kelvin(data.Kelvin))
	return of, spaceOne, err
}
