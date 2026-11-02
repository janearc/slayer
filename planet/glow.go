package planet

import (
	"fmt"
	"math"

	"github.com/janearc/celestial/bands"
	"github.com/janearc/libtheme-css/spaces/srgb"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/observer"
)

// Xrays are what a planet gives off in x-rays from a band about its
// equator: their power in watts, and how far the band reaches north and
// south of the equator, in degrees.
type Xrays struct{ power, band float64 }

// NewXrays is x-rays, refused at no power or with no band.
func NewXrays(power, band float64) (Xrays, error) {
	if !(power >= 0) || !(band > 0 && band < 90) {
		return Xrays{}, fmt.Errorf("x-rays of %g W from a band %g degrees "+
			"wide", power, band)
	}
	return Xrays{power: power, band: band}, nil
}

// on is the x-rays falling on the ring plane at a reach, in watts a
// square metre, both faces together, and the sine of the angle they
// come down at.
//
// modelled: the band's half above the plane shines on the plane's upper
// face as a ring of light at the band's edge, its height over the plane,
// falling off as a line source's does; the lower half alike below.
func (xrays Xrays) on(radius, reach float64) (power, sine float64) {
	height := radius * math.Sin(xrays.band*maths.Degree)
	out := reach - radius
	if !(out > 0) {
		return 0, 0
	}
	square := height*height + out*out
	return xrays.power * height / (2 * math.Pi * math.Pi * reach * square),
		height / math.Sqrt(square)
}

// the phosphors a ring may hold, by name: the share of absorbed x-ray
// power each gives back as light, and the middle and width of what it
// gives, in nanometres.
//
// csi:tl's yield is worked from the lbl scintillator library's 55,000
// to 61,000 photons a mev and its 560 nm peak, about 12%; zns:ag's 18%
// is quoted, by saleh, lynn and mccloy 2017, not yet read; its 450 nm
// peak is eljen's sheet. both widths are picked. ice gives back none.
var phosphors = map[string]struct{ yield, peak, width float64 }{
	"zns-ag": {0.18, 450, 80},
	"csi-tl": {0.12, 560, 120},
}

// glow is a ring's phosphors: the share of the x-ray power it absorbs
// that it gives back as light, and that light's colour, a unit of power
// spread over libtheme's bands.
type glow struct {
	yield  float64
	colour bands.Light
}

// Glowing is the ring holding phosphors by share of its mass; what is
// not named is ice, which gives back no light.
func (ring Ring) Glowing(shares map[string]float64) (Ring, error) {
	lines, yield := map[float64]float64{}, 0.0
	for name, share := range shares {
		made, known := phosphors[name]
		if !known || !(share >= 0 && share <= 1) {
			return ring, fmt.Errorf("a ring of %g %s", share, name)
		}
		yield += share * made.yield
		spread(lines, share*made.yield, made.peak, made.width)
	}
	if yield == 0 {
		return ring, nil
	}
	for nm := range lines {
		lines[nm] /= yield
	}
	ring.glow = glow{yield: yield, colour: bands.Lines(lines)}
	return ring, nil
}

// spread adds a bell of power, so wide at half its height, about a
// middle, nanometre by nanometre.
func spread(lines map[float64]float64, power, middle, width float64) {
	spread := width / (2 * math.Sqrt(2*math.Ln2))
	whole := power / (spread * math.Sqrt(2*math.Pi))
	for nm := middle - 3*width; nm <= middle+3*width; nm++ {
		apart := (nm - middle) / spread
		lines[nm] += whole * math.Exp(-apart*apart/2)
	}
}

// RingGlow is a ring's phosphors as the tracer marches them: a slab that
// gives off light and takes almost none away. its density is the ring's
// times a faintness, so the light it stops is far below anything the
// picture holds, and its glow is told by the shine that goes with it.
type RingGlow struct {
	ring        Ring
	half, faint float64
}

// faintness is the glow slab's density against the ring's: picked, so
// small its slab stops none of the light behind it.
const faintness = 1e-12

// NewRingGlow is a ring's glow as a slab of a thickness in metres, and
// the shine that says its light, for a planet of a radius under its
// x-rays, at an exposure.
func NewRingGlow(ring Ring, thickness, radius float64, xrays Xrays,
	exposure float64) (RingGlow, *GlowShine, error) {
	if !(thickness > 0) || !(radius > 0) {
		return RingGlow{}, nil, fmt.Errorf("a glow %g m thick about a "+
			"planet of %g m", thickness, radius)
	}
	slab := RingGlow{ring: ring, half: thickness / 2, faint: faintness}
	shine := &GlowShine{ring: ring, thickness: thickness, radius: radius,
		xrays: xrays, colour: seenAs(ring.glow.colour, exposure)}
	return slab, shine, nil
}

// Density is the ring's, its ringlets', times the faintness.
func (slab RingGlow) Density(p maths.Vec) float64 {
	return slab.faint * slab.ring.at(p, slab.half)
}

// Bounds is the slab's box.
func (slab RingGlow) Bounds() maths.Box {
	far := slab.ring.outer
	return maths.Box{Min: maths.V(-far, -slab.half, -far),
		Max: maths.V(far, slab.half, far)}
}

// GlowShine is a ring's phosphors' light at each point of its glow slab,
// a metre of it at unit density.
type GlowShine struct {
	ring              Ring
	thickness, radius float64
	xrays             Xrays
	colour            maths.Light
}

// At is the glow a metre of the slab gives off a steradian at a point of
// the ring's own space, over its density there: the x-rays falling
// there, the share the ring's depth there stops along their slant, given
// back by its phosphors, alike every way and spread through the slab.
func (shine *GlowShine) At(p maths.Vec) maths.Light {
	reach := math.Hypot(p.X, p.Z)
	share := shine.ring.share(p.X, p.Z)
	power, sine := shine.xrays.on(shine.radius, reach)
	if !(share > 0) || !(sine > 0) || shine.ring.glow.yield == 0 {
		return maths.Light{}
	}
	stopped := -math.Expm1(-shine.ring.depth * share / sine)
	given := power * stopped * shine.ring.glow.yield /
		(4 * math.Pi * shine.thickness)
	return shine.colour.Scale(given / (faintness * share))
}

// seenAs is light in libtheme's bands as the tracer's three channels, at
// an exposure.
func seenAs(light bands.Light, exposure float64) maths.Light {
	red, green, blue := srgb.Light(observer.Swatch(light.Scale(exposure)))
	return maths.RGB(math.Max(0, red), math.Max(0, green),
		math.Max(0, blue))
}
