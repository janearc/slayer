package manufacture

import (
	"math"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/noise"
	"github.com/janearc/slayer/scene/texture"
	"github.com/janearc/slayer/scene/thing"
)

// The pelage is the coat as one material: sealMarks's spots, rings and
// blotches on a dark back and light belly, hair streaks along the flow,
// a bare dark nose pad, and a wet coat's darkening and sheen. Every
// number here is chosen.
const (
	// sealHaloLight is how strongly a ring's light middle lightens the
	// coat.
	sealHaloLight = .55
	// sealStreakAlong and sealStreakAcross are how many streak cells lie
	// to the metre along and across the flow, and sealStreakStrength how
	// far streaks darken and lighten the coat.
	sealStreakAlong, sealStreakAcross, sealStreakStrength = 60., 700., .22
	// sealStreakNear and sealStreakFar are the pixel footprints, in
	// metres, between which streaks fade out so they never glitter.
	sealStreakNear, sealStreakFar = .0006, .003
	// sealWetDarken is how much of its colour a soaked coat loses.
	sealWetDarken = .77
	// sealUnderskin is the share of the coat's colour the skin under the
	// hair keeps.
	sealUnderskin = .5
)

var (
	// sealNoseColour is the bare nose pad.
	sealNoseColour = maths.RGB(.014, .012, .011)
	// sealHaloColour is the light the halo ring round a spot adds.
	sealHaloColour = maths.RGB(.30, .28, .24)
	// sealWetFloor is the colour a soaked coat keeps from the water on
	// its hair.
	sealWetFloor = maths.RGB(.021, .019, .016)
)

// sealPelage is the texture of a seal's coat, read at rest coordinates.
type sealPelage struct {
	coat sealCoat
	flow sealFlow
	wet  float64
}

// Pelage is the seal's coat as one texture over its rest skin, dry at
// look.Wet of nought and soaked at one.
func (rig *SealRig) Pelage(look SealLook) texture.Texture {
	return sealPelage{coat: sealCoat{seed: look.Seed}, flow: rig.flowField(), wet: look.Wet}
}

// At is the pelage's colour at a point, with no footprint.
func (pelage sealPelage) At(point maths.Vec) maths.Light {
	return pelage.Within(point, 0)
}

// Within is the pelage's colour at a point over a footprint in metres: the
// ground, a halo and the marks cut over it, the marks fading on the belly.
func (pelage sealPelage) Within(point maths.Vec, footprint float64) maths.Light {
	base, lightSide := pelage.coat.ground(point)
	mark, lift := sealMarks(pelage.coat.seed, point, footprint)
	mark *= 1 - sealBellyMarkFade*lightSide
	dry := base.Add(sealHaloColour.Scale(sealHaloLight * lift))
	dry = dry.Scale(1 - mark).Add(sealMarkColour.Scale(mark))
	dry = dry.Scale(1 + pelage.streak(point, footprint))
	coat := dry.Scale(1 - sealWetDarken*pelage.wet).
		Add(sealWetFloor.Scale(pelage.wet))
	nose := sealNose(point)
	return coat.Scale(1 - nose).Add(sealNoseColour.Scale(nose))
}

// streak is the lightening or darkening of hair streaks at a point:
// noise stretched along the coat's flow and thin across it.
func (pelage sealPelage) streak(point maths.Vec, footprint float64) float64 {
	flow := sealFrame(pelage.flow.at(point), maths.Up)
	value := noise.Gradient(pelage.coat.seed+9, point.Dot(flow.Z)*sealStreakAlong,
		point.Dot(flow.X)*sealStreakAcross, point.Dot(flow.Y)*sealStreakAcross)
	return sealStreakStrength * value *
		(1 - maths.Smooth(sealStreakNear, sealStreakFar, footprint))
}

// sealNose is how far a rest point lies in the bare nose pad, from none
// to one. The pad is an oval at the front of the muzzle.
func sealNose(point maths.Vec) float64 {
	width := .028 + .024*maths.Smooth(.023, .077, point.Y)
	pad := 1 - maths.Smooth(.76, 1.02, math.Hypot((point.Y-.059)/.038, point.Z/width))
	return pad * maths.Smooth(.779, .804, point.X)
}

// sealDimmed is a texture at a share of its colour.
type sealDimmed struct {
	inner texture.Texture
	share float64
}

// At is the dimmed texture's colour at a point, with no footprint.
func (dim sealDimmed) At(point maths.Vec) maths.Light { return dim.Within(point, 0) }

// Within is the inner texture sampled over a footprint and scaled by the
// dimmer's share.
func (dim sealDimmed) Within(point maths.Vec, footprint float64) maths.Light {
	return texture.Sample(dim.inner, point, footprint).Scale(dim.share)
}

// Materials are the seal's skin and hair materials for a look: the skin
// under the hair is dimmer, a wet coat glosses and a dry one does not.
func (rig *SealRig) Materials(look SealLook) (skin, hair *thing.Material) {
	pelage := rig.Pelage(look)
	gloss := .05 + .37*look.Wet
	satin := .18 + .15*look.Wet
	skin = &thing.Material{Name: "harbor seal skin", Texture: sealDimmed{pelage, sealUnderskin},
		Gloss: gloss, Satin: satin, Index: 1.43}
	hair = &thing.Material{Name: "harbor seal hair", Texture: pelage,
		Gloss: gloss * .7, Satin: satin, Index: 1.43}
	return skin, hair
}
