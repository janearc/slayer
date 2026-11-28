package main

import (
	"github.com/janearc/slayer/bioforms/pigment"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/noise"
	"math"
)

// complexion uses independent coarse vascular and superficial pigment fields.
// RGB absorption weights approximate the observed colour ordering; they are
// artistic parameters, without a spectral skin model or fitted concentrations.
var finePigment = pigment.Spots{Seed: 8321, Spacing: .00235, Radius: .00080}

func complexion(p maths.Vec, marks float64) maths.Light {
	base := skinWithMarks(p, 0)
	x, y, z := math.Abs(p.X), p.Y, p.Z
	front := maths.Smooth(.025, .062, z)
	cheek := gauss((x-.043)/.023) * gauss((y-1.526)/.019)
	nose := gauss(x/.013) * gauss((y-1.524)/.014)
	grain := func(seed uint64, k float64) float64 { return noise.Gradient(seed, p.X*k, p.Y*k, p.Z*k) }
	vessel := front * (.45*cheek + .39*nose + .25*gauss((x-.03)/.024)*gauss((y-1.542)/.006))
	vessel += .12*gauss((y-.46)/.06) + .14*gauss((x-.43)/.042)*gauss((y-1.29)/.05) + .18*gauss((x-.069)/.009)*gauss((y-1.534)/.026)
	mottling := .48*grain(53, 37) + .30*grain(35, 97) + .12*grain(54, 220)
	melanin := .23 + .25*grain(98, 18) + .26*grain(66, 70)
	base = base.Times(rgb(math.Exp(-.06*melanin), math.Exp(-.12*melanin-.17*vessel), math.Exp(-.20*melanin-.10*vessel)))
	base = base.Times(rgb(1+.08*mottling, 1+.04*mottling, 1-.03*mottling))
	zone := front * max(cheek, gauss((y-1.536)/.010)*gauss(x/.049))
	clusters := maths.Smooth(-.32, .42, grain(3, 52)+.48*grain(126, 115))
	flecks := marks * zone * (.14 + .86*clusters) * (.62*freckles.At(p) + .44*finePigment.At(p))
	return mix(base, rgb(.19, .07, .033), min(.89, flecks))
}

// oilFilm is the spatial blend of two traced microfacet responses. Broad
// cheeks and limbs stay mostly matte; nasal sidewalls and lips carry more.
func oilFilm(p maths.Vec) float64 {
	x, y := math.Abs(p.X), p.Y
	front := maths.Smooth(.034, .07, p.Z)
	nose := gauss(x/.016) * gauss((y-1.535)/.025)
	lip := gauss(x/.019) * gauss((y-1.488)/.006)
	forehead := gauss(x/.037) * gauss((y-1.588)/.018)
	lid := gauss((x-.032)/.015) * gauss((y-1.550)/.004)
	return .035 + front*min(.90, .64*nose+.36*lip+.21*forehead+.34*lid)
}
