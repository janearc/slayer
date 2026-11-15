package planet

import (
	"math"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/occultation"
)

// The readers in this file tell a renderer that asks a ring itself, as the
// gpu's kernel does, every number the ring's share is made from, so that it
// can make the same share from them. A term a ring does not have is not
// given: its share is one.

// Span is the ring's inner and outer radius, in metres.
func (ring Ring) Span() (inner, outer float64) { return ring.inner, ring.outer }

// Depth is the ring's mean optical depth straight through it.
func (ring Ring) Depth() float64 { return ring.depth }

// Eye is where the ring is seen from, a point of its own space, and how wide
// one ray from it is, in radians; a ray of zero is no eye.
func (ring Ring) Eye() (eye maths.Vec, ray float64) {
	return ring.from.eye, ring.from.ray
}

// Ringlet is a ring's radial structure, as the numbers its share is read from.
type Ringlet struct{ made ringlets }

// RingletTerm is the ring's ringlets, and whether it has any.
func (ring Ring) RingletTerm() (Ringlet, bool) {
	return Ringlet{made: ring.ringlets}, ring.ringlets.strength != 0
}

// Seed is the noise's seed; layer i's seed is this plus i.
func (term Ringlet) Seed() uint64 { return term.made.seed }

// Strength is how far the ringlets take the depth from its mean.
func (term Ringlet) Strength() float64 { return term.made.strength }

// Width is the coarsest ringlets' width in metres.
func (term Ringlet) Width() float64 { return term.made.width }

// Layers is how many layers of ringlets there are.
func (term Ringlet) Layers() int { return ringletOctaves }

// Finer is how many times finer each layer is than the one before.
func (term Ringlet) Finer() float64 { return ringletFiner }

// Weaker is how strong each layer is against the one before.
func (term Ringlet) Weaker() float64 { return ringletWeaker }

// Across is where on the noise's other two axes the layers are read.
func (term Ringlet) Across() float64 { return ringletAcross }

// Stretch is a stretch of a measured profile laid across a ring.
type Stretch struct{ made measured }

// StretchTerm is the ring's measured stretch, and whether it has one.
func (ring Ring) StretchTerm() (Stretch, bool) {
	return Stretch{made: ring.measured}, ring.measured.wide != 0
}

// Baked is the profile as a table of levels, finest first.
func (term Stretch) Baked() occultation.Baked { return term.made.baked }

// From is the radius, in metres of the profile, the stretch begins at.
func (term Stretch) From() float64 { return term.made.from }

// Wide is how many metres of the profile one metre of the ring spans.
func (term Stretch) Wide() float64 { return term.made.wide }

// Grain is how a ring's matter clumps across its plane, as the numbers its
// share is read from.
type Grain struct{ made grains }

// GrainTerm is the ring's grains, and whether it has any.
func (ring Ring) GrainTerm() (Grain, bool) {
	has := ring.grains.clumping != 0 && ring.grains.pattern != nil
	return Grain{made: ring.grains}, has
}

// Pattern is the name of the pattern the grains are drawn in.
func (term Grain) Pattern() string { return term.made.name }

// Seed is the pattern's seed; layer i's seed is this plus i.
func (term Grain) Seed() uint64 { return term.made.seed }

// Largest is the largest clumps' size in metres.
func (term Grain) Largest() float64 { return term.made.largest }

// Shear is how many times longer along the orbit than across a clump is.
func (term Grain) Shear() float64 { return term.made.shear }

// Slope is the power law's slope of the sizes of clumps.
func (term Grain) Slope() float64 { return term.made.slope }

// Threshold is the value of the pattern above which there is matter.
func (term Grain) Threshold() float64 { return term.made.threshold }

// Softness is the width of a clump's edge, before what the layers a ray
// cannot see add to it.
func (term Grain) Softness() float64 { return term.made.softness }

// Spread is one layer's pattern's standard deviation.
func (term Grain) Spread() float64 { return term.made.spread }

// Filled is the share of the plane the grains leave holding matter.
func (term Grain) Filled() float64 { return filled(term.made.clumping) }

// Widths is how many standard deviations of the hidden layers wide the
// edge of a clump is taken over.
func (term Grain) Widths() float64 { return hiddenWidths }

// Weights is how strongly each layer of clumps shows against the largest,
// largest first.
func (term Grain) Weights() []float64 {
	var found []float64
	size := term.made.largest
	for range term.Layers() {
		found = append(found, term.made.weight(size))
		size /= 2
	}
	return found
}

// Layers is how many layers of clumps are drawn, a layer for each doubling of
// size from the largest down to the smallest, no more than the most.
func (term Grain) Layers() int {
	count := 0
	for size := term.made.largest; count < mostLayers &&
		size >= term.made.smallest; size /= 2 {
		count++
	}
	return count
}

// Moonlet is a body in a ring, as the numbers the clearing it makes is read
// from.
type Moonlet struct{ made moonlet }

// MoonletTerms is the ring's moonlets, none if it has none.
func (ring Ring) MoonletTerms() []Moonlet {
	var found []Moonlet
	for _, each := range ring.moonlets {
		found = append(found, Moonlet{made: each})
	}
	return found
}

// Reach is how far out the moonlet orbits, in metres.
func (term Moonlet) Reach() float64 { return term.made.reach }

// Angle is where round its orbit the moonlet is, in radians.
func (term Moonlet) Angle() float64 { return term.made.angle }

// Gap is how far either side of its orbit the moonlet clears, in metres.
func (term Moonlet) Gap() float64 { return gapReach * term.made.hill }

// Length is how far along its orbit a propeller's lobes reach, in metres.
func (term Moonlet) Length() float64 { return wake * term.made.hill }

// Edge is how wide a clearing's soft edge is, in the widths of its gap.
func (term Moonlet) Edge() float64 { return edgeWidth }

// Opens is whether the moonlet clears its whole orbit.
func (term Moonlet) Opens() bool { return term.made.opens }

// Ring is the slab's ring.
func (slab RingSlab) Ring() Ring { return slab.ring }

// Half is half the slab's thickness, in metres.
func (slab RingSlab) Half() float64 { return slab.half }

// Ring is the glow slab's ring.
func (slab RingGlow) Ring() Ring { return slab.ring }

// Half is half the glow slab's thickness, in metres.
func (slab RingGlow) Half() float64 { return slab.half }

// Faint is the glow slab's density against the ring's.
func (slab RingGlow) Faint() float64 { return slab.faint }

// Shine is the shine made with the slab, which says its light.
func (slab RingGlow) Shine() *GlowShine { return slab.shine }

// Yield is the share of the x-ray power the ring absorbs that it gives back
// as light, zero for a ring with no phosphors.
func (shine *GlowShine) Yield() float64 { return shine.ring.glow.yield }

// Colour is the light a unit of the glow's power is seen as.
func (shine *GlowShine) Colour() maths.Light { return shine.colour }

// Radius is the radius in metres of the planet the x-rays come from.
func (shine *GlowShine) Radius() float64 { return shine.radius }

// Lift is how high over the ring's plane the x-rays' band stands, in metres.
func (shine *GlowShine) Lift() float64 {
	return shine.radius * math.Sin(shine.xrays.band*maths.Degree)
}

// spreading is what an x-ray's power is spread over: a sphere's four pi
// steradians of the glow's light, and the two pi squared of the band's ring
// of light's own spreading, true by definition.
const spreading = 2 * math.Pi * math.Pi * 4 * math.Pi

// Source is the x-ray power, times the band's lift over the plane and the
// ring's yield, over the spreading and the slab's thickness: what the glow's
// light at a reach is, times the share the ring stops, over the reach and the
// square of the lift and the way out from the planet.
func (shine *GlowShine) Source() float64 {
	return shine.xrays.power * shine.Lift() * shine.ring.glow.yield /
		(spreading * shine.thickness)
}
