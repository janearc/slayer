package primates

import (
	"math"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/noise"
)

// Chroma shifts red, golden and olive reflectance while preserving local
// photopic luminance. Melanin controls the principal lightness change.
type Chroma struct{ Red, Gold, Olive float64 }

func (c Chroma) valid() bool {
	return between(c.Red, -1, 1) && between(c.Gold, -1, 1) && between(c.Olive, -1, 1)
}

func (c Chroma) apply(base maths.Light) maths.Light {
	shift := maths.RGB(math.Exp(.20*c.Red+.068*c.Gold-.080*c.Olive), math.Exp(-.047*c.Red+.016*c.Gold+.085*c.Olive), math.Exp(.022*c.Red-.26*c.Gold-.12*c.Olive))
	changed := base.Times(shift)
	return changed.Scale(base.Luminance() / changed.Luminance())
}

type skin struct {
	spec Torso
	seed uint64
}

func (k skin) At(rest maths.Vec) maths.Light { return k.Within(rest, 0) }

func (k skin) Within(rest maths.Vec, footprint float64) maths.Light {
	return k.spec.optics(rest, k.seed, footprint).Colour
}

// SkinResponse separates a superficial, absorbing epidermis from the diffuse
// dermal return. Coefficients are illustrative RGB bands, not spectroscopy.
type SkinResponse struct {
	Colour                 maths.Light
	Roughness, DermalShare float64
	Radius                 [3]float64
}

// Areolar tubercles have irregular positions within the pigmented plateau.
// The same low, localized elevations drive the native mesh and local optics.
func areolarRelief(u, v, side float64) (float64, float64) {
	r := math.Hypot(u, v)
	a := math.Atan2(v, u)
	centre := .62 + .10*math.Sin(3*a+.48*side)
	tubercle := bell((r-centre)/.16) * math.Pow(.5+.5*math.Cos(8*a+.83*math.Sin(3*a+side)+.55*side), 7)
	papilla := math.Exp(-.62*math.Pow(r/.25, 4) - .38*math.Pow(r/.25, 2))
	return papilla, tubercle
}
func (s Torso) Optics(p maths.Vec, seed uint64) SkinResponse {
	return s.optics(p, seed, 0)
}

var skinDiffusionRadius = [3]float64{.0024, .00108, .00052}

func (s Torso) optics(p maths.Vec, seed uint64, footprint float64) SkinResponse {
	return s.opticsWithAbsorption(p, seed, footprint, skinAbsorption{}, BreastForm{})
}

func (s Torso) opticsWithAbsorption(p maths.Vec, seed uint64, footprint float64,
	absorption skinAbsorption, breasts BreastForm,
) SkinResponse {
	side, _, sign := s.side(p.X)
	tone := s.areolarTone.Right
	if sign > 0 {
		tone = s.areolarTone.Left
	}
	form := breasts.side(sign)
	x, y := s.landmarks(side)
	x += form.RootOutward
	u, v := form.areolarChart(side, p, x, y)
	angle := math.Atan2(v, u)
	micro := skinGrain(p, seed+1, 760, footprint)
	coarse := skinGrain(p, seed+4, 19, footprint)
	fine := skinGrain(p, seed+8, 78, footprint)
	margin := 1 + .050*math.Sin(3*angle+.9+.31*sign) +
		.039*math.Sin(5*angle-1.2-.24*sign) + .060*fine
	d := math.Hypot(u+.055*v, v) / margin
	papilla, tubercle := s.relief(side, u, v, sign)
	if s.LegacyContour {
		papilla, tubercle = areolarRelief(u, v, sign)
	}
	anterior := maths.Smooth(.008, .072, p.Z)
	if form.WithoutAreola {
		anterior = 0
	}
	// Keep a continuous plateau before the soft outer transition.
	// Fine fluctuations in the edge fade at a coarse pixel footprint.
	inner := .76 - .11*tone.EdgeFeather
	outer := 1.15 + .09*tone.EdgeFeather + .045*micro
	areola := (1 - maths.Smooth(inner, outer, d)) * anterior
	centre := .62*papilla + .38*maths.Smooth(.14, .70, papilla)
	papilla *= areola
	tubercle *= areola
	patch := areolarMasks{areola: areola, papilla: centre * areola, tubercles: tubercle}
	// Pigment follows tissue coordinates on both sides of the trunk. These
	// fields contain no light, camera direction, visibility or surface normal.
	ax := math.Abs(p.X)
	shoulder := bell((ax-.161*s.Width)/.031) * bell((p.Y-1.338)/.031)
	axilla := bell((ax-.148*s.Width)/.023) * bell((p.Y-1.285)/.041)
	clavicle := bell((p.Y-1.344-.014*math.Sin(19*ax))/.022) * bell((ax-.083*s.Width)/.072)
	chest := bell((ax-x)/.079) * bell((p.Y-y)/.075)
	melanin := .033 + .83*s.Melanin + .74*s.Melanin*s.Melanin +
		.033*coarse + .016*fine + .047*axilla + .018*shoulder
	blood := .16 + .42*s.Vascular + .044*coarse + .056*shoulder + .064*clavicle + .032*chest
	if absorption != (skinAbsorption{}) {
		melanin = max(0, melanin+absorption.melanin)
		blood = max(0, blood+absorption.blood)
	}
	if areola > 0 {
		// A separate millimetre-scale pigment field sits inside the taper.
		// It stays on the rest chart and carries no shading or relief.
		grain := skinGrain(p, seed+17, 230, footprint)
		localMelanin, localBlood := s.areolarAbsorption(tone, patch, sign, fine, grain)
		melanin = max(0, melanin+localMelanin)
		blood = max(0, blood+localBlood)
	}
	// Diffuse light traverses the pigmented layer on entry and return.
	dermis := maths.RGB(.81*math.Exp(-.11*blood), .65*math.Exp(-.55*blood), .54*math.Exp(-.38*blood))
	epidermis := maths.RGB(math.Exp(-.57*melanin), math.Exp(-.78*melanin), math.Exp(-.99*melanin))
	c := dermis.Times(epidermis).Times(epidermis).Scale(.99 + .014*micro)
	c = s.Undertone.apply(c)
	if regional := tone.chroma(patch); regional != (Chroma{}) {
		c = applyRegionalChroma(c, regional)
	}
	if s.paint != nil {
		c = s.paint.At(c, p, footprint)
	}
	rough := .70 + .07*fine - .068*areola - .040*papilla + .021*tubercle + .012*shoulder + .019*axilla
	return SkinResponse{Colour: c, Roughness: rough, DermalShare: .28 - .12*s.Melanin + .025*coarse, Radius: skinDiffusionRadius}
}

func skinGrain(point maths.Vec, seed uint64, rate, footprint float64) float64 {
	share := 1 - maths.Smooth(.32, 1, rate*footprint)
	if share == 0 {
		return 0
	}
	return share * noise.Gradient(seed, point.X*rate, point.Y*rate, point.Z*rate)
}
