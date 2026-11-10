package oceantex

import (
	"math"
	"sort"

	"github.com/janearc/libtheme-css/primitives/swatch"
	"github.com/janearc/libtheme-css/spaces/srgb"
)

var (
	waterFirst    = num("water.first")
	waterStep     = num("water.step")
	absorbed      = nums("pope-fry.absorption")
	scatterAt500  = num("morel.scattering.at-500")
	scatterRef    = num("morel.scattering.reference")
	scatterPower  = num("morel.scattering.power")
	backwardShare = num("water.backward.share")
	gordonLinear  = num("gordon.linear")
	gordonSquare  = num("gordon.square")
	surfaceGain   = num("lee.surface.gain")
	surfaceLoss   = num("lee.surface.loss")

	pigmentShape   = nums("prieur.shape")
	particleScale  = num("case1.particle.scale")
	particlePower  = num("case1.particle.power")
	cdomShare      = num("case1.cdom.share")
	cdomSlope      = num("case1.cdom.slope")
	cdomRef        = num("case1.cdom.reference")
	scatterScale   = num("case1.scatter.scale")
	scatterGrowth  = num("case1.scatter.power")
	scatterAt      = num("case1.scatter.reference")
	scatterColour  = num("case1.scatter.colour")
	backwardScale  = num("ulloa.scale")
	backwardOffset = num("ulloa.offset")
	backwardSlope  = num("ulloa.slope")
)

// table is a row of a ten-nanometre table at a wavelength, on the straight
// line between the two nearest rows. Outside the table it is the nearest
// row.
func table(rows []float64, nm float64) float64 {
	at := (nm - waterFirst) / waterStep
	if at <= 0 {
		return rows[0]
	}
	last := len(rows) - 1
	if at >= float64(last) {
		return rows[last]
	}
	below := int(at)
	part := at - float64(below)
	return rows[below] + (rows[below+1]-rows[below])*part
}

// Absorption is pure water's absorption per metre at a wavelength in
// nanometres [PF97].
func Absorption(nm float64) float64 { return table(absorbed, nm) }

// Scattering is how much light pure seawater scatters, per metre [M74].
func Scattering(nm float64) float64 {
	return scatterAt500 * math.Pow(nm/scatterRef, scatterPower)
}

// Reflectance is the remote-sensing reflectance of deep seawater just
// above the surface, per steradian: the light leaving the water straight
// up, as a fraction of the light falling on it. chlorophyll is in
// milligrams a cubic metre, and zero is water with nothing in it.
//
// Absorption is water's [PF97], plus phytoplankton pigment and dissolved
// organic matter by the classic case 1 model: 0.06 A Chl^0.65, and a fifth
// of that at 440 nm falling off toward the red [PS81, MM01]. Scattering
// backward is half of what the water's molecules scatter [M74], plus the
// particles' 0.30 Chl^0.62 (550/nm) [GM83] times the share of it that goes
// backward [U94].
//
// Below the surface the reflectance is 0.0949 u + 0.0794 u^2, where u is
// backscattering over absorption plus backscattering [G88]. Crossing the
// surface it becomes 0.52 r / (1 - 1.7 r) [L02].
func Reflectance(nm, chlorophyll float64) float64 {
	absorb := Absorption(nm)
	back := Scattering(nm) * backwardShare
	if chlorophyll > 0 {
		pigment := particleScale * math.Pow(chlorophyll, particlePower)
		absorb += pigment * table(pigmentShape, nm)
		absorb += cdomShare * pigment * table(pigmentShape, cdomRef) *
			math.Exp(-cdomSlope*(nm-cdomRef))
		particles := scatterScale * math.Pow(chlorophyll, scatterGrowth) *
			math.Pow(scatterAt/nm, scatterColour)
		// GLUE: the share measured at 550 nm, used at every wavelength,
		// and never let fall below nothing.
		share := backwardScale *
			(backwardOffset - backwardSlope*math.Log10(chlorophyll))
		back += particles * math.Max(share, 0)
	}
	u := back / (absorb + back)
	below := gordonLinear*u + gordonSquare*u*u
	return surfaceGain * below / (1 - surfaceLoss*below)
}

// lit is what a surface looks like under a lamp: the lamp's spectrum,
// times the part of it the surface sends back at each wavelength, weighed
// by libtheme's observer, and measured against a white: a second spectrum
// whose own luminance is called 1. With the lamp as its own white, a
// surface that sends everything back has luminance 1.
//
// The sum is taken in wavelength order so the same question has the same
// answer, as libtheme's own sums are.
func lit(
	lamp, white map[int]float64, part func(nm int) float64,
) swatch.Swatch {
	order := make([]int, 0, len(lamp))
	for nm := range lamp {
		order = append(order, nm)
	}
	sort.Ints(order)
	var x, y, z float64
	for _, nm := range order {
		mx, my, mz := swatch.Monochrome(nm).XYZ()
		sent := lamp[nm] * part(nm)
		x, y, z = x+sent*mx, y+sent*my, z+sent*mz
	}
	order = order[:0]
	for nm := range white {
		order = append(order, nm)
	}
	sort.Ints(order)
	var whole float64
	for _, nm := range order {
		_, my, _ := swatch.Monochrome(nm).XYZ()
		whole += white[nm] * my
	}
	if whole == 0 {
		return swatch.NoLight()
	}
	return swatch.FromXYZ(x/whole, y/whole, z/whole)
}

// Lamp is the colour of a lamp's own light, with luminance 1.
func Lamp(lamp map[int]float64) swatch.Swatch {
	return lit(lamp, lamp, func(int) float64 { return 1 })
}

// Water is the colour of deep seawater under a lamp, seen from straight
// above with the surface's own reflections left out: pi times
// Reflectance, so that a perfect white matte surface would be the lamp's
// colour at luminance 1. chlorophyll is in milligrams a cubic metre.
//
// GLUE: outside 400 to 700 nm the water is taken to send nothing back.
func Water(lamp map[int]float64, chlorophyll float64) swatch.Swatch {
	return lit(lamp, lamp, waterPart(chlorophyll))
}

// waterPart is the part of the light the water sends back at a
// wavelength, as a matte surface would: pi times its reflectance.
func waterPart(chlorophyll float64) func(nm int) float64 {
	last := waterFirst + waterStep*float64(len(absorbed)-1)
	return func(nm int) float64 {
		if float64(nm) < waterFirst || float64(nm) > last {
			return 0
		}
		return math.Pi * Reflectance(float64(nm), chlorophyll)
	}
}

// light is a colour as amounts of the three sRGB lamps, linear, unclipped:
// what libtheme's srgb.Light returns, kept together so it can be added.
type light [3]float64

// lightOf is a swatch as light.
func lightOf(s swatch.Swatch) light {
	r, g, b := srgb.Light(s)
	return light{r, g, b}
}

// times is the light k times as strong.
func (l light) times(k float64) light {
	return light{l[0] * k, l[1] * k, l[2] * k}
}

// plus is both lights together.
func (l light) plus(o light) light {
	return light{l[0] + o[0], l[1] + o[1], l[2] + o[2]}
}
