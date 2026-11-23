package manufacture

import (
	"math"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/texture"
	"github.com/janearc/slayer/scene/thing"
)

// how a street looks as it gets wet. a wet surface is a dry one with a thin
// film of water over it: a dielectric of water's index, which mirrors a
// little looking straight down and nearly all looking along the surface,
// over a base that the water has darkened. the material already has all of
// that, as Gloss, which scales the Fresnel mirror, and Index. no number
// here is measured; each is picked for how a wet street looks, and says so.
const (
	// streetWaterIndex is water's index of refraction.
	streetWaterIndex = 1.333
	// streetFilmFirst and streetFilmFull are the wetness at which a
	// film begins to cover the road and at which it covers it whole.
	streetFilmFirst = 0.1
	streetFilmFull  = 0.7
	// streetAsphaltGloss is how much of the water's mirror the wet road
	// has: a film over rough asphalt breaks at the stones.
	streetAsphaltGloss = 0.9
	// streetAsphaltDry and streetAsphaltWet are how broad a sheen the
	// road's lamp highlights take dry and wet, from nought to one.
	streetAsphaltDry = 0.55
	streetAsphaltWet = 0.15
	// streetRipple is the width of the wrinkles a wet road's film lies in,
	// in metres, and streetRippleSlope how far they tilt what it
	// mirrors. picked: a mirrored panel is smeared, not broken. standing
	// water has none: it is flat, and mirrors sharply.
	streetRipple      = 0.2
	streetRippleSlope = 0.02
	// streetTrackAt is how far each wheel track lies from the street's
	// middle, in metres: the middle of each lane.
	streetTrackAt = 1.75
	// streetSeedBits keeps a seed under 2^24, which the GPU holds whole.
	streetSeedBits = 1<<24 - 1
	// streetFootwayGloss and streetKerbGloss are the share of the
	// water's mirror the stone has, wet: stone is rougher than a film.
	streetFootwayGloss = 0.3
	streetKerbGloss    = 0.2
	// streetPaintGloss and streetGrateGloss, the same for paint and iron.
	streetPaintGloss = 0.5
	streetGrateGloss = 0.35
	// streetSlab is the side of a paving slab and streetJoint the width
	// of the joint between slabs, in metres. streetGratePitch is the
	// distance from one bar of a grating to the next. picked.
	streetSlab       = 0.6
	streetJoint      = 0.012
	streetGratePitch = 0.06
)

// streetRamp is nought at or below least and one at or above most, and
// smooth between.
func streetRamp(value, least, most float64) float64 {
	share := math.Min(1, math.Max(0, (value-least)/(most-least)))
	return share * share * (3 - 2*share)
}

// film is how much of the street the water's film covers, from nought to
// one.
func (ground streetGround) film() float64 {
	return streetRamp(ground.wet, streetFilmFirst, streetFilmFull)
}

// looks is the look of each matter of the street, as wet as it is.
func (ground streetGround) looks(from seed) looks {
	film := ground.film()
	return looks{
		streetAsphalt: ground.road(from, film),
		streetKerb: ground.stone("street kerb", maths.RGB(0.2, 0.2, 0.19),
			streetKerbGloss*film),
		streetPavement: ground.footway(from, film),
		streetPaint: ground.stone("street paint", maths.RGB(0.5, 0.5, 0.45),
			streetPaintGloss*film),
		streetGrate: ground.iron(film),
		streetWater: ground.water(),
	}
}

// wrinkled is the film's wrinkles, which smear what a wet surface mirrors,
// when there is a film to be wrinkled, and none when it is dry.
func wrinkled(from seed, film, size, slope float64) *thing.Crinkle {
	if film <= 0 { // defined: a dry surface has no film
		return nil
	}
	return &thing.Crinkle{Size: size, Slope: slope,
		Seed: uint32(from.number & streetSeedBits)}
}

// road is the asphalt: a damp, mottled binder, and over it, as it wets,
// the film that mirrors.
func (ground streetGround) road(from seed, film float64) *thing.Material {
	return &thing.Material{Name: "street asphalt",
		Colour: maths.RGB(0.09, 0.09, 0.095),
		Texture: texture.Asphalt{Dry: maths.RGB(0.09, 0.09, 0.095),
			Chipping: maths.RGB(0.2, 0.19, 0.17), Wet: ground.wet,
			Tracks: streetTrackAt, Seed: from.number},
		Gloss: streetAsphaltGloss * film, Index: streetWaterIndex,
		Satin:   lerp(streetAsphaltDry, streetAsphaltWet, film),
		Crinkle: wrinkled(from, film, streetRipple, streetRippleSlope)}
}

// footway is the paving slabs, darker and a little mirroring when wet.
func (ground streetGround) footway(from seed, film float64) *thing.Material {
	return &thing.Material{Name: "street pavement",
		Colour: maths.RGB(0.28, 0.27, 0.25),
		Texture: texture.Slabs{Slab: maths.RGB(0.28, 0.27, 0.25),
			Size: streetSlab, Joint: streetJoint, Wet: film,
			Seed: from.number},
		Gloss: streetFootwayGloss * film, Index: streetWaterIndex,
		Satin:   streetAsphaltDry,
		Crinkle: wrinkled(from, film, streetRipple, streetRippleSlope)}
}

// stone is a plain painted or cut surface with a share of the water's
// mirror, wet.
func (ground streetGround) stone(name string, colour maths.Light,
	gloss float64,
) *thing.Material {
	return &thing.Material{Name: name, Colour: colour, Gloss: gloss,
		Index: streetWaterIndex, Satin: streetAsphaltDry}
}

// iron is the gratings' iron over the black of the pit below.
func (ground streetGround) iron(film float64) *thing.Material {
	return &thing.Material{Name: "street grate",
		Colour: maths.RGB(0.06, 0.06, 0.065),
		Texture: texture.Grate{Bar: maths.RGB(0.06, 0.06, 0.065),
			Pit: maths.RGB(0.004, 0.004, 0.005), Pitch: streetGratePitch},
		Gloss: streetGrateGloss * film, Index: streetWaterIndex,
		Satin: streetAsphaltDry}
}

// water is standing water: nothing under it that shows, so a dark base,
// and all of the water's mirror, over a surface as flat as water lies.
func (ground streetGround) water() *thing.Material {
	return &thing.Material{Name: "street water",
		Colour: maths.RGB(0.006, 0.006, 0.007), Gloss: 1,
		Index: streetWaterIndex}
}
