package manufacture

import (
	"math"

	"github.com/janearc/slayer/maths"
)

// orcaFar is the distance a part reports for a point beyond its reach.
const orcaFar = 1.0

// orcaEdgeRadius rounds every fin edge, so no edge is thinner than the
// lattice can hold. It is chosen: about one lattice step.
const orcaEdgeRadius = .016

// orcaEdgeShape is the thickness profile across a fin's chord, zero at
// both edges, blunt at the leading one: u runs from the trailing edge
// (0) to the leading edge (1). Its largest value is one.
func orcaEdgeShape(u float64) float64 {
	const peak = 2.6 // makes the profile's maximum one, at u = 2/3
	return peak * u * math.Sqrt(1-u)
}

// orcaClamp01 limits x to the range zero to one.
func orcaClamp01(x float64) float64 { return math.Min(math.Max(x, 0), 1) }

// orcaStep is zero at from, one at to and eased between, in either
// order of the two.
func orcaStep(from, to, x float64) float64 {
	t := orcaClamp01((x - from) / (to - from))
	return t * t * (3 - 2*t)
}

// orcaSmoothMin is the smaller of a and b with the corner between them
// rounded over a width of k: this is what fillets a fin into the body.
func orcaSmoothMin(a, b, k float64) float64 {
	h := math.Max(k-math.Abs(a-b), 0) / k
	return math.Min(a, b) - h*h*k/4
}

// orcaRounded combines a distance across a part's outline and a distance
// across its thickness into one, with every edge rounded by radius.
func orcaRounded(outline, thickness, radius float64) float64 {
	a, b := outline+radius, thickness+radius
	return math.Hypot(math.Max(a, 0), math.Max(b, 0)) +
		math.Min(math.Max(a, b), 0) - radius
}

// orcaSlit is a groove's profile: one on its line, falling to zero at
// width away.
func orcaSlit(gap, width float64) float64 {
	t := math.Max(0, 1-(gap/width)*(gap/width))
	return t * t
}

// The dorsal fin of a cow: falcate, with the tip swept back over the
// trailing base. These numbers were chosen, in metres in rest space.
const (
	orcaFinFoot  = .40  // height where the profile begins, inside the back
	orcaFinTip   = 1.62 // height of the tip, a little under a metre clear
	orcaFinLead  = 1.05 // x of the leading edge at the foot
	orcaFinRake  = .80  // how far back the leading edge runs to the tip
	orcaFinCurve = 1.7  // power of the leading edge's sweep
	orcaFinTrail = .30  // x of the trailing edge at the foot
	orcaFinBow   = .22  // how far forward the trailing edge is hollowed
	orcaFinHalf  = .07  // half thickness at the foot
	orcaFinTaper = .6   // share of that thickness lost by the tip
	orcaFinChord = .06  // chord of the tip, which is blunt, not a cusp
)

// orcaDorsal is the distance from p to the dorsal fin: a flat profile
// with a blunt leading edge and a fine trailing one.
func orcaDorsal(p maths.Vec) float64 {
	if p.X < -.1 || p.X > 1.3 || p.Y < .3 || p.Y > orcaFinTip+.1 ||
		math.Abs(p.Z) > .2 {
		return orcaFar
	}
	span := orcaFinTip - orcaFinFoot
	t := orcaClamp01((p.Y - orcaFinFoot) / span)
	tipX := orcaFinLead - orcaFinRake - orcaFinChord
	lead := orcaFinLead - orcaFinRake*math.Pow(t, orcaFinCurve)
	trail := orcaFinTrail - (orcaFinTrail-tipX)*t +
		orcaFinBow*math.Sin(math.Pi*t)
	leadSlope := orcaFinRake * orcaFinCurve * math.Pow(t, orcaFinCurve-1) / span
	trailSlope := ((tipX - orcaFinTrail) +
		orcaFinBow*math.Pi*math.Cos(math.Pi*t)) / span
	outline := math.Max((p.X-lead)/math.Hypot(1, leadSlope),
		(trail-p.X)/math.Hypot(1, trailSlope))
	outline = math.Max(outline, p.Y-orcaFinTip)
	u := 0.
	if lead > trail {
		u = orcaClamp01((p.X - trail) / (lead - trail))
	}
	half := orcaFinHalf * (1 - orcaFinTaper*t) * orcaEdgeShape(u)
	return orcaRounded(outline, math.Abs(p.Z)-half, orcaEdgeRadius)
}

// The pectoral flipper: a paddle, broad and rounded, set low on the
// flank and directed down, out and a little back. Chosen, in metres.
const (
	orcaFlipperRootX = 1.55 // x of the root, buried in the flank
	orcaFlipperRootY = -.22 // height of the root
	orcaFlipperRootZ = .30  // distance of the root from the midline
	orcaFlipperAt    = .29  // where along the span the paddle's middle lies
	orcaFlipperReach = .68  // half the paddle's length along the span
	orcaFlipperChord = .27  // half the paddle's greatest width
	orcaFlipperSweep = .10  // how far the paddle's middle trails per metre
	orcaFlipperHalf  = .055 // half thickness at the middle
)

var (
	// orcaFlipperSpan runs from the root to the tip.
	orcaFlipperSpan = maths.V(-.45, -.40, 1).Unit()
	// orcaFlipperChordway lies across the paddle, forward.
	orcaFlipperChordway = maths.V(1, 0, 0).Sub(
		orcaFlipperSpan.Scale(orcaFlipperSpan.X)).Unit()
	// orcaFlipperNormal stands through the paddle's thickness.
	orcaFlipperNormal = orcaFlipperSpan.Cross(orcaFlipperChordway).Unit()
)

// orcaFlipper is the distance from p to the pair of flippers; the left
// is the mirror of the right.
func orcaFlipper(p maths.Vec) float64 {
	q := maths.V(p.X, p.Y, math.Abs(p.Z)).Sub(
		maths.V(orcaFlipperRootX, orcaFlipperRootY, orcaFlipperRootZ))
	along := q.Dot(orcaFlipperSpan)
	across := q.Dot(orcaFlipperChordway)
	through := q.Dot(orcaFlipperNormal)
	if along < -.5 || along > 1.1 || math.Abs(across) > .45 ||
		math.Abs(through) > .2 {
		return orcaFar
	}
	a := (along - orcaFlipperAt) / orcaFlipperReach
	b := (across + orcaFlipperSweep*along) / orcaFlipperChord
	rim := a*a + b*b
	outline := (math.Sqrt(rim) - 1) * orcaFlipperChord
	half := orcaFlipperHalf * math.Sqrt(math.Max(0, 1-rim))
	return orcaRounded(outline, math.Abs(through)-half, orcaEdgeRadius)
}

// The flukes: swept back to fine tips, a median notch, thickest at the
// root. Chosen, in metres in rest space.
const (
	orcaFlukeLead  = -2.30 // x of the leading edge at the midline
	orcaFlukeRake  = .78   // how far back the leading edge runs to the tip
	orcaFlukePower = 1.35  // power of the leading edge's sweep
	orcaFlukeTrail = -2.72 // x of the trailing edge at the midline, un-notched
	orcaFlukeDrop  = .41   // how far back the trailing edge runs to the tip
	orcaFlukeSpan  = 1.0   // half the span, tip to midline
	orcaFlukeNotch = .11   // depth of the median notch
	orcaFlukeCleft = .16   // half width of the notch at the trailing edge
	orcaFlukeHalf  = .06   // half thickness at the root
	orcaFlukeTaper = .5    // share of that thickness lost by the tip
	orcaFlukeLevel = -.01  // height of the flukes' plane
)

// orcaFlukeEdges is the x of the flukes' leading and trailing edges at
// distance sigma from the midline.
func orcaFlukeEdges(sigma float64) (lead, trail float64) {
	reach := math.Min(sigma, orcaFlukeSpan)
	lead = orcaFlukeLead - orcaFlukeRake*math.Pow(reach, orcaFlukePower)
	cleft := math.Max(0, 1-sigma/orcaFlukeCleft)
	trail = orcaFlukeTrail - orcaFlukeDrop*reach + orcaFlukeNotch*cleft*cleft
	return lead, trail
}

// orcaFlukes is the distance from p to the flukes, which lie in one
// horizontal plane and are the same each side of the midline.
func orcaFlukes(p maths.Vec) float64 {
	sigma := math.Abs(p.Z)
	if p.X > -2.2 || p.X < -3.2 || sigma > orcaFlukeSpan+.1 ||
		math.Abs(p.Y-orcaFlukeLevel) > .2 {
		return orcaFar
	}
	reach := math.Min(sigma, orcaFlukeSpan)
	lead, trail := orcaFlukeEdges(sigma)
	leadSlope := orcaFlukeRake * orcaFlukePower * math.Pow(reach, orcaFlukePower-1)
	outline := math.Max((p.X-lead)/math.Hypot(1, leadSlope),
		(trail-p.X)/math.Hypot(1, orcaFlukeDrop))
	outline = math.Max(outline, sigma-orcaFlukeSpan)
	u := 0.
	if lead > trail {
		u = orcaClamp01((p.X - trail) / (lead - trail))
	}
	half := orcaFlukeHalf * (1 - orcaFlukeTaper*reach) * orcaEdgeShape(u)
	level := math.Abs(p.Y-orcaFlukeLevel) - half
	return orcaRounded(outline, level, orcaEdgeRadius)
}

// The keels along the tail stock, a ridge above and another below.
const (
	orcaKeelHeight = .06   // how far the ridge stands clear of the skin
	orcaKeelHalf   = .022  // half its thickness
	orcaKeelStart  = -1.70 // x where the ridge begins to rise
	orcaKeelFull   = -2.10 // x where it has risen fully
	orcaKeelEnd    = -2.62 // x where the trunk ends
)

// orcaKeel is the distance from p to the dorsal keel (side one) or the
// ventral keel (side minus one), each a thin ridge on the tail stock.
func orcaKeel(p maths.Vec, side float64) float64 {
	if p.X < orcaKeelEnd || p.X > orcaKeelStart || math.Abs(p.Z) > .1 {
		return orcaFar
	}
	s := orcaTrunkProfile().at(p.X)
	rise := orcaKeelHeight * orcaStep(orcaKeelStart, orcaKeelFull, p.X)
	depth := side * (p.Y - s.centre)
	outline := math.Max(depth-(s.high+rise), (s.high-.05)-depth)
	return orcaRounded(outline, math.Abs(p.Z)-orcaKeelHalf, orcaEdgeRadius)
}

// orcaField is the signed distance, near the surface, from p to the
// whole animal in rest space: the trunk, the fins and flukes joined to
// it with filleted roots, the grooves cut into it, and the teeth.
func orcaField(p maths.Vec) float64 {
	const (
		rootBlend = .12 // fillet width at the dorsal fin and flukes
		limbBlend = .10 // fillet width at the flipper
		keelBlend = .05 // fillet width at the keels
	)
	d := orcaTrunk(p)
	d = orcaSmoothMin(d, orcaDorsal(p), rootBlend)
	d = orcaSmoothMin(d, orcaFlipper(p), limbBlend)
	d = orcaSmoothMin(d, orcaFlukes(p), rootBlend)
	d = orcaSmoothMin(d, orcaKeel(p, 1), keelBlend)
	d = orcaSmoothMin(d, orcaKeel(p, -1), keelBlend)
	d += orcaCarve(p)
	return math.Min(d, orcaTeeth(p))
}
