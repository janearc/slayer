package medium

import (
	"math"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/noise"
)

// Clouds is a layer of cloud held against a body: droplets between Low and High
// above Radius, in patches laid by fractal noise over the sphere. Cover is how
// much is clouded, zero none and one all, but not in proportion: a half clouds
// about two thirds of the sky.
//
// A patch is Thick at its heart, thinning to its soft, broken edge. The
// thicker a patch the higher it reaches, a thin patch only part way up the
// layer and the thickest to its top; up through a patch it is whole to a
// third of the way, then thins to 0.15 of itself at its own top, where it
// stops. The layer is cut flat at Low, where a patch starts at its full
// thickness. Seed makes a sky's own weather. Nothing below Low or above High.
//
// Locked is a layer over a world with one face always toward its sun, which
// stands toward Toward, a vector in the medium's frame: its weather is laid
// about that substellar point and not by latitude (see CloudWeather).
type Clouds struct {
	Radius    float64
	Low, High float64
	Cover     float64
	Thick     float64
	Seed      uint64
	Locked    bool
	Toward    maths.Vec
	// Soft reads the fine billows without the storms' turn, an artistic
	// choice available independently of the body's orbit.
	Soft bool
	// Turn overrides a storm's angular turn at its heart when HasTurn is
	// true; otherwise the existing weather's authored default is used.
	Turn    float64
	HasTurn bool
}

// Bounds is the cube the layer's top just fits in.
func (clouds Clouds) Bounds() maths.Box {
	return cube(clouds.Radius + clouds.High)
}

// CloudPatches is how Density lays its patches, for a renderer that asks
// the same density itself: the noise's frequency over the unit direction
// and its octaves, how far either side of the cover's threshold an edge
// is soft, how far up the layer a patch is whole, and the share of itself
// it still is at the top. Billow is how much finer noise, Detail times
// the frequency with DetailOctaves, breaks the patches up, and Reach how
// far up the layer the thinnest patch reaches.
type CloudPatches struct {
	Frequency        float64
	Octaves          int
	Edge, Whole, Top float64
	Billow, Detail   float64
	DetailOctaves    int
	Reach            float64
}

// Patches is the numbers Density lays its patches by.
func (Clouds) Patches() CloudPatches {
	return CloudPatches{Frequency: chosen.cloudFrequency,
		Octaves: chosen.cloudOctaves, Edge: chosen.cloudEdge,
		Whole: chosen.cloudWhole, Top: chosen.cloudTop,
		Billow: chosen.cloudBillow, Detail: chosen.cloudDetail,
		DetailOctaves: chosen.cloudDetailOctaves, Reach: chosen.cloudReach}
}

// Span is where a ray crosses the layer's top.
func (clouds Clouds) Span(ray maths.Ray, near, far float64) (float64, float64,
	bool) {
	return sphere(clouds.Radius+clouds.High, ray, near, far)
}

// weathered is the noise and the cover at a direction from the body's
// centre. The cover is the layer's own, with the belts of its latitude and
// the storms' hearts added. The noise is read where the storms' turning
// and then the winds' sliding and stretching put it: patches about two
// dozen round at a cover of a half, drawn out along the latitude lines
// into wisps and wound into whorls; finer noise breaks their edges up.
func (clouds Clouds) weathered(direction maths.Vec) (float64, float64) {
	return clouds.weatheredBy(clouds.Weather(), direction)
}

// weatheredBy is weathered by a weather already made, which Weather makes
// afresh each time it is asked.
func (clouds Clouds) weatheredBy(weather CloudWeather,
	direction maths.Vec) (float64, float64) {
	latitude := math.Asin(math.Max(-unity, math.Min(unity, direction.Y)))
	turned, hearts := weather.Swirl(direction)
	read := weather.Wind(turned, latitude)
	frequency := chosen.cloudFrequency
	grain := halfway + noise.Fractal(clouds.Seed, read.X*frequency,
		read.Y*frequency, read.Z*frequency, chosen.cloudOctaves)
	grain += weather.Filaments(read, frequency)
	// the billow is read unstretched, so that edges break up and fray
	// where the wind would smear them
	fine := frequency * chosen.cloudDetail
	// in a locked sky the fine noise is read where the storms have not
	// turned it: wound with the coarse field, a whorl pulls it into
	// concentric lines at its edge, and the whorl is a hole ringed with
	// combing and not cloud spiralling
	finer := turned
	if weather.Locked || clouds.Soft {
		finer = direction
	}
	grain += chosen.cloudBillow * noise.Fractal(clouds.Seed+1,
		finer.X*fine, finer.Y*fine, finer.Z*fine, chosen.cloudDetailOctaves)
	cover, belt := weather.Cover(direction, latitude)
	if belt > nought {
		// a locked sky's terminator is broken by noise finer still
		grain += weather.Broken * belt * noise.Fractal(
			clouds.Seed+brokenStream, finer.X*fine, finer.Y*fine,
			finer.Z*fine, filamentOctaves)
	}
	return grain, clouds.Cover + cover + hearts
}

// Density is the patch here, if the point is in the layer.
func (clouds Clouds) Density(point maths.Vec) float64 {
	if clouds.High <= clouds.Low || clouds.Cover <= nought {
		return nought
	}
	above := point.Len() - clouds.Radius
	if above < clouds.Low || above > clouds.High {
		return nought
	}
	weather := clouds.Weather()
	grain, cover := clouds.weatheredBy(weather, point.Unit())
	threshold := unity - cover
	patch := maths.Smooth(threshold-chosen.cloudEdge,
		threshold+chosen.cloudEdge, grain)
	if patch <= nought {
		return nought
	}
	// puffy: whole to a third of the way up, thinning to the patch's own
	// top, which a thicker patch has higher
	height := (above - clouds.Low) / (clouds.High - clouds.Low)
	reach := math.Min(unity, (chosen.cloudReach+
		(unity-chosen.cloudReach)*patch)*weather.Top(point.Unit())*
		weather.Relief(point.Unit()))
	if height > reach {
		return nought
	}
	puff := unity - maths.Smooth(chosen.cloudWhole*reach, reach, height)
	// a turning sky's cloud keeps a share of its density to its top, which
	// is a step there; a locked sky's towers thin to nothing at their tops,
	// so that their tops and edges are soft and light comes through them
	floor := chosen.cloudTop
	if weather.Locked {
		floor = nought
	}
	return clouds.Thick * patch * math.Max(puff, floor)
}
