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
	if !(power >= nothing) || !(band > nothing && band < poleLatitude) {
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
	if !(out > nothing) {
		return nothing, nothing
	}
	square := height*height + out*out
	return xrays.power * height / (circle * math.Pi * reach * square),
		height / math.Sqrt(square)
}

// A phosphor is what a ring may hold that glows under x-rays: the share
// of absorbed x-ray power it gives back as light, and the middle and width
// of what it gives, in nanometres.
type phosphor struct{ yield, peak, width float64 }

// phosphors are the phosphors a ring may hold, by name; their numbers are
// in picks.yaml. ice gives back none.
//
// csi:tl's yield is worked from the lbl scintillator library's 55,000
// to 61,000 photons a mev and its 560 nm peak; zns:ag's is quoted, by
// saleh, lynn and mccloy 2017; its 450 nm peak is eljen's sheet. the
// widths are chosen.
var phosphors = map[string]phosphor{
	"zns-ag": {pick("zns-ag-yield"), pick("zns-ag-peak"),
		pick("zns-ag-width")},
	"csi-tl": {pick("csi-tl-yield"), pick("csi-tl-peak"),
		pick("csi-tl-width")},
}

// glow is a ring's phosphors: the share of the x-ray power it absorbs
// that it gives back as light, and that light's colour, a unit of power
// spread over libtheme's bands.
type glow struct {
	yield  float64
	colour bands.Light
}

// Glowing is the ring holding phosphors by share of its mass; what is
// not named is ice, which gives back no light. Shares that sum to more
// than one are refused.
func (ring Ring) Glowing(shares map[string]float64) (Ring, error) {
	lines := map[float64]float64{}
	var yield, total float64
	for name, share := range shares {
		made, known := phosphors[name]
		if !known || !(share >= nothing && share <= whole) {
			return ring, fmt.Errorf("a ring of %g %s", share, name)
		}
		total += share
		yield += share * made.yield
		spread(lines, share*made.yield, made.peak, made.width)
	}
	if total > whole+shareTolerance {
		return ring, fmt.Errorf("a ring of phosphors by %g of its mass",
			total)
	}
	if yield == nothing {
		return ring, nil
	}
	for nm := range lines {
		lines[nm] /= yield
	}
	ring.glow = glow{yield: yield, colour: bands.Lines(lines)}
	return ring, nil
}

// bellWidths is how many widths either side of its middle a phosphor's bell
// is spread over, from picks.yaml.
var bellWidths = pick("bell-widths")

// spread adds a bell of power, so wide at half its height, about a
// middle, nanometre by nanometre.
func spread(lines map[float64]float64, power, middle, width float64) {
	spread := width / (double * math.Sqrt(double*math.Ln2))
	peak := power / (spread * math.Sqrt(circle))
	extent := bellWidths * width
	for nm := middle - extent; nm <= middle+extent; nm++ {
		apart := (nm - middle) / spread
		lines[nm] += peak * math.Exp(-apart*apart/double)
	}
}

// RingGlow is a ring's phosphors as the tracer marches them: a slab that
// gives off light and takes almost none away. its density is the ring's
// times a faintness, so the light it stops is far below anything the
// picture holds, and its glow is told by the shine that goes with it.
type RingGlow struct {
	ring        Ring
	half, faint float64
	shine       *GlowShine
}

// faintness is the glow slab's density against the ring's, so small its
// slab stops none of the light behind it; its value is in picks.yaml.
var faintness = pick("glow-faintness")

// NewRingGlow is a ring's glow as a slab of a thickness in metres, and
// the shine that says its light, for a planet of a radius under its
// x-rays, at an exposure.
func NewRingGlow(ring Ring, thickness, radius float64, xrays Xrays,
	exposure float64) (RingGlow, *GlowShine, error) {
	if !(thickness > nothing) || !(radius > nothing) {
		return RingGlow{}, nil, fmt.Errorf("a glow %g m thick about a "+
			"planet of %g m", thickness, radius)
	}
	shine := &GlowShine{ring: ring, thickness: thickness, radius: radius,
		xrays: xrays, colour: seenAs(ring.glow.colour, exposure)}
	slab := RingGlow{ring: ring, half: thickness / double, faint: faintness,
		shine: shine}
	return slab, shine, nil
}

// MarchesAtTheMiddle says a ring's glow, as a ring, is marched at the
// middle of each step; see RingSlab.
func (slab RingGlow) MarchesAtTheMiddle() bool { return true }

// Density is the ring's, its ringlets', times the faintness.
func (slab RingGlow) Density(point maths.Vec) float64 {
	return slab.faint * slab.ring.at(point, slab.half)
}

// Bounds is the slab's box.
func (slab RingGlow) Bounds() maths.Box {
	return slabBox(slab.ring.outer, slab.half)
}

// slabBox is the box of a slab of a ring: out to a reach either way across
// its plane, and so far above and below it.
func slabBox(far, half float64) maths.Box {
	var box maths.Box
	box.Min = maths.V(-far, -half, -far)
	box.Max = maths.V(far, half, far)
	return box
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
func (shine *GlowShine) At(point maths.Vec) maths.Light {
	reach := math.Hypot(point.X, point.Z)
	share := shine.ring.share(point.X, point.Z)
	power, sine := shine.xrays.on(shine.radius, reach)
	if !(share > nothing) || !(sine > nothing) ||
		shine.ring.glow.yield == nothing {
		return noLight
	}
	stopped := -math.Expm1(-shine.ring.depth * share / sine)
	given := power * stopped * shine.ring.glow.yield /
		(sphere * shine.thickness)
	return shine.colour.Scale(given / (faintness * share))
}

// seenAs is light in libtheme's bands as the tracer's three channels, at
// an exposure.
func seenAs(light bands.Light, exposure float64) maths.Light {
	seen := observer.Swatch(light.Scale(exposure))
	red, green, blue := srgb.Light(seen)
	return maths.RGB(math.Max(nothing, red), math.Max(nothing, green),
		math.Max(nothing, blue))
}
