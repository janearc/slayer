package planet

import (
	"fmt"
	"math"
	"slices"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/noise"
	"github.com/janearc/slayer/occultation"
	"github.com/janearc/slayer/scene/gas"
)

// A Ring is a flat ring of particles about a planet's equator: its inner
// and outer radius in metres, its optical depth straight through it, the
// share of the light its particles turn aside that they scatter, and how
// far forward they throw it, as a henyey-greenstein lean.
type Ring struct {
	inner, outer, depth, lean float64
	albedo                    tint
	ringlets                  ringlets
	measured                  measured
	grains                    grains
	moonlets                  []moonlet
	glow                      glow
	from                      vantage
}

// NewRing is a grey ring of even depth, refused outside what a ring can
// be.
func NewRing(inner, outer, depth, albedo, lean float64) (Ring, error) {
	if !(inner > nothing) || !(outer > inner) || !(depth >= nothing) ||
		!(albedo >= nothing && albedo <= whole) ||
		!(lean > -whole && lean < whole) {
		return Ring{}, fmt.Errorf("a ring from %g to %g m, depth %g, "+
			"albedo %g, lean %g", inner, outer, depth, albedo, lean)
	}
	return Ring{inner: inner, outer: outer, depth: depth, lean: lean,
		albedo: tint{albedo, albedo}}, nil
}

// Tinted is the ring with its particles' albedo at 400 nm set apart from
// the one it was made with, which holds at 700 nm.
func (ring Ring) Tinted(blue float64) (Ring, error) {
	made, err := ring.albedo.blued(blue)
	ring.albedo = made
	return ring, err
}

// ringlets are the radial structure inside a ring: its depth rising and falling
// from place to place about its mean, by seeded gradient noise in layers, the
// coarsest so many metres across. strength zero is an even ring.
type ringlets struct {
	seed            uint64
	strength, width float64
}

// Ringleted is the ring with ringlets: its depth from place to place one and
// the strength times seeded noise of its mean, never less than zero, the
// coarsest ringlets so many metres across. modelled.
func (ring Ring) Ringleted(seed uint64, strength, width float64) (Ring,
	error) {
	if !(strength >= nothing) || !(width > nothing) {
		return ring, fmt.Errorf("ringlets of strength %g, %g m across",
			strength, width)
	}
	ring.ringlets = ringlets{seed: seed, strength: strength, width: width}
	return ring, nil
}

// ringletOctaves is how many layers of ringlets there are, each finer and
// weaker than the one before; its value is in picks.yaml.
var ringletOctaves = pickCount("ringlet-octaves")

// ringletFiner is how many times finer each layer of ringlets is than the one
// before, ringletWeaker how strong it is against that one's, and ringletAcross
// where across the ring's other two axes the noise is read; all three are in
// picks.yaml.
var (
	ringletFiner  = pick("ringlet-finer")
	ringletWeaker = pick("ringlet-weaker")
	ringletAcross = pick("ringlet-across")
)

// share is how much of the ring's mean depth stands at a reach.
func (made ringlets) share(reach float64) float64 {
	if made.strength == nothing {
		return whole
	}
	var sum float64
	size, strength := whole/made.width, float64(whole)
	for i := nothing; i < ringletOctaves; i++ {
		sum += strength * noise.Gradient(made.seed+uint64(i), size*reach,
			ringletAcross, ringletAcross)
		size, strength = size*ringletFiner, strength*ringletWeaker
	}
	return math.Max(nothing, whole+made.strength*sum)
}

// measured is a stretch of a ring's depth measured through another
// planet's rings, laid evenly across this one: the profile baked, the
// stretch's radii there, and how many of its metres one of this ring's
// spans.
type measured struct {
	baked          occultation.Baked
	from, to, wide float64
}

// Measured is the ring with its depth from place to place read from a
// profile measured through another planet's rings: the stretch from one
// radius to another, in metres, laid evenly across this ring, inner to
// outer.
//
// its depth is the stretch's as light sees it from far off, and its
// ringlets are the stretch's own, in place of any made up.
//
// a ray reads the profile as wide as it is, as the mean light kept straight
// through; at a slant that keeps a little less than the stretch's own would.
func (ring Ring) Measured(profile occultation.Profile, from,
	to float64) (Ring, error) {
	if !(from >= profile.Inner()) || !(to <= profile.Outer()) ||
		!(to > from) {
		return ring, fmt.Errorf("a stretch from %g to %g m of a profile "+
			"from %g to %g m", from, to, profile.Inner(), profile.Outer())
	}
	depth := profile.Seen((from+to)/double, to-from)
	if !(depth > nothing) {
		return ring, fmt.Errorf("the stretch from %g to %g m is empty", from,
			to)
	}
	ring.depth, ring.ringlets = depth, ringlets{}
	ring.measured = measured{baked: profile.Baked(), from: from, to: to,
		wide: (to - from) / (ring.outer - ring.inner)}
	return ring, nil
}

// share is how much of the ring's depth stands at a reach, seen by a ray
// so many metres wide there: the profile's depth at the place laid
// there, over the ring's. a ring not measured is even.
func (made measured) share(ring Ring, reach, finest float64) float64 {
	if made.wide == nothing {
		return whole
	}
	there := made.from + (reach-ring.inner)*made.wide
	seen := made.baked.Seen(float32(there), float32(finest*made.wide))
	return float64(seen) / ring.depth
}

// Ringed is the body with rings about its equator, inner to outer. they
// may not overlap, and the first must stand clear of the body.
func (body Body) Ringed(rings []Ring) (Body, error) {
	edge := body.radius
	for i, ring := range rings {
		if ring.inner < edge {
			return Body{}, fmt.Errorf("ring %d starts at %g m, inside %g",
				i, ring.inner, edge)
		}
		edge = ring.outer
	}
	body.rings = rings
	return body, nil
}

// Rings is the body's rings, inner to outer.
func (body Body) Rings() []Ring { return slices.Clone(body.rings) }

// kept is how much of the light crossing a ring at a slant, the cosine
// of its angle to the ring's pole, gets through.
func (ring Ring) kept(slant float64) float64 {
	return math.Exp(-ring.depth / math.Abs(slant))
}

// shadowed is how much of the star's light reaches a point of the body
// through its rings, from a star one way, with the pole another, both
// unit ways: the ring the star's ray crosses the equator's plane in, if
// any, kept as its depth along the slant says.
func shadowed(rings []Ring, point, star, pole maths.Vec) float64 {
	slant := star.Dot(pole)
	if slant == nothing {
		return whole
	}
	along := -point.Dot(pole) / slant
	if !(along > nothing) {
		return whole
	}
	crossed := point.Add(star.Scale(along))
	reach := crossed.Len()
	plane := maths.Upright(pole)
	x, z := crossed.Dot(plane.X), crossed.Dot(plane.Z)
	for _, ring := range rings {
		if reach >= ring.inner && reach < ring.outer {
			return math.Exp(-ring.depth * ring.share(x, z) /
				math.Abs(slant))
		}
	}
	return whole
}

// A RingSlab is a ring as the tracer marches it: a slab in the ring's
// own space, the ring's plane flat across x and z and its pole up y, so
// many metres thick, holding the ring's optical depth spread through it.
//
// real rings are tens of metres thick; the slab is thicker so a march
// finds it, and thinner per metre to match, so its depth through is the
// ring's. it is seen only edge on.
type RingSlab struct {
	ring  Ring
	half  float64
	light gas.Gas
}

// greyRing is a ring's optics a metre: its albedo's share scattered and the
// rest absorbed, at a depth through the slab.
type greyRing struct {
	albedo  tint
	through float64
}

// at is the absorption and scattering a metre of the slab gives a
// wavelength in nanometres.
func (grey greyRing) at(nm float64) (absorb, scatter float64) {
	albedo := grey.albedo.at(nm)
	return (whole - albedo) * grey.through, albedo * grey.through
}

// NewRingSlab is a ring as a slab of a thickness in metres.
func NewRingSlab(ring Ring, thickness float64) (RingSlab, error) {
	if !(thickness > nothing) {
		return RingSlab{}, fmt.Errorf("a slab %g m thick", thickness)
	}
	grey := greyRing{albedo: ring.albedo, through: ring.depth / thickness}
	var light gas.Gas
	light.Name = "ring"
	light.Phase = ring.lean
	light.Visible = grey.at
	return RingSlab{ring: ring, half: thickness / double, light: light}, nil
}

// MarchesAtTheMiddle says a ring's march takes the middle of each step: a
// ring is a thin flat leg whose grains are hashed by radius, and a sample
// at a hashed step falls either side of it by the last bit of a float,
// which a float32 and a float64 do not share (2026-11-26).
func (slab RingSlab) MarchesAtTheMiddle() bool { return true }

// Density is the ring's share of its mean depth there, its ringlets', inside
// the slab, and zero outside it.
func (slab RingSlab) Density(point maths.Vec) float64 {
	return slab.ring.at(point, slab.half)
}

// at is the ring's share of its mean depth at a point of its own space, in a
// slab so many metres from its plane: its ringlets' inside it, zero outside.
func (ring Ring) at(point maths.Vec, half float64) float64 {
	reach := math.Hypot(point.X, point.Z)
	if math.Abs(point.Y) > half || reach < ring.inner ||
		reach >= ring.outer {
		return nothing
	}
	return ring.share(point.X, point.Z)
}

// share is how much of the ring's mean depth stands at a point of its
// plane, x and z in its own space: its ringlets', measured or made up,
// its grains' and what its moonlets leave, together. the slab, the glow
// and the shadow all read it, so all three see the same clumps and gaps.
func (ring Ring) share(x, z float64) float64 {
	finest, reach := ring.from.finest(x, z), math.Hypot(x, z)
	return ring.ringlets.share(reach) *
		ring.measured.share(ring, reach, finest) *
		ring.grains.share(x, z, finest) * ring.cleared(x, z)
}

// Bounds is the slab's box.
func (slab RingSlab) Bounds() maths.Box {
	return slabBox(slab.ring.outer, slab.half)
}

// Gas is the ring's optics a metre at unit density: grey, scattering its
// albedo's share and absorbing the rest, leaning as its particles do.
func (slab RingSlab) Gas() gas.Gas { return slab.light }
