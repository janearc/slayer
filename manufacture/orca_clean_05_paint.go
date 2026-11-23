package manufacture

import (
	"math"

	"github.com/janearc/slayer/maths"
)

// The reflectances of the animal's colours, linear, each chosen: black
// skin reflects a few parts in a hundred, white skin about seven in ten.
var (
	orcaBlack  = maths.RGB(.016, .020, .024)
	orcaWhite  = maths.RGB(.72, .74, .70)
	orcaSaddle = maths.RGB(.20, .21, .22)
	orcaPale   = maths.RGB(.34, .34, .35)
	orcaIris   = maths.RGB(.040, .025, .015)
	orcaPupil  = maths.RGB(.004, .004, .005)
	orcaGum    = maths.RGB(.050, .020, .020)
	orcaIvory  = maths.RGB(.78, .75, .66)
)

// The markings, chosen, in metres in rest space and in the trunk's own
// coordinates. The belly's edge is hard, as the animal's is; the saddle
// patch's is soft.
const (
	orcaBellyEdge   = -.34  // height of the belly's edge, in half heights
	orcaBellyLobe   = .62   // how far the white lobe climbs the flank
	orcaLobeStart   = .05   // x where the lobe begins to climb
	orcaLobePeak    = -1.05 // x where it has climbed fully
	orcaLobeEnd     = -1.28 // x where it ends, sharply
	orcaHardEdge    = .02   // half width of a hard edge, in half heights
	orcaPatchX      = 1.82  // x of the middle of the eye patch
	orcaPatchY      = .12   // height of the middle of the eye patch
	orcaPatchLong   = .21   // half the patch's length
	orcaPatchShort  = .065  // half the patch's breadth
	orcaPatchTilt   = .26   // tilt of the patch, rear end up, in radians
	orcaSaddleFront = .45   // x where the saddle patch begins, behind the fin
	orcaSaddleRear  = -.95  // x where the saddle patch has faded away
	orcaMarkPigment = .45   // share of the pigment a rake or scar has lost
)

// orcaMix is a blended with b, none of b at zero and all of it at one.
func orcaMix(a, b maths.Light, t float64) maths.Light {
	return maths.RGB(a.R+(b.R-a.R)*t, a.G+(b.G-a.G)*t, a.B+(b.B-a.B)*t)
}

// orca05Paint is the animal's colour at a point of its rest shape: the
// markings follow the body by where they lie on it, not by where it is.
type orca05Paint struct{}

// At is the colour at rest point p.
func (orca05Paint) At(p maths.Vec) maths.Light {
	colour := orcaBlack
	if orcaTrunk(p) < .03 {
		colour = orcaMix(colour, orcaWhite, orcaBelly(p))
		colour = orcaMix(colour, orcaSaddle, orcaSaddlePatch(p))
		colour = orcaMix(colour, orcaWhite, orcaEyePatch(p))
	} else {
		colour = orcaMix(colour, orcaWhite, orcaFlukeUnderside(p))
	}
	marks := math.Max(orcaRakeMark(p), orcaScarMark(p))
	colour = orcaMix(colour, orcaPale, orcaMarkPigment*marks)
	colour = orcaMix(colour, orcaGum, orcaStep(.01, .02, orcaMouth(p)))
	colour = orcaMix(colour, orcaPupil, orcaStep(.01, .025, orcaBlowhole(p)))
	colour = orcaMix(colour, orcaIvory, 1-orcaStep(.002, .006, orcaTeeth(p)))
	return orcaEyeColour(p, colour)
}

// orcaBelly is how far p lies in the white belly: one on the throat, the
// belly and the lobes that climb the flanks, zero on the black above.
func orcaBelly(p maths.Vec) float64 {
	s := orcaTrunkProfile().at(p.X)
	height := (p.Y - s.centre) / s.high
	climb := orcaStep(orcaLobeStart, orcaLobePeak, p.X) *
		orcaStep(orcaLobeEnd, orcaLobePeak, p.X)
	edge := orcaBellyEdge + orcaBellyLobe*climb
	return 1 - orcaStep(edge-orcaHardEdge, edge+orcaHardEdge, height)
}

// orcaSaddlePatch is how far p lies in the grey saddle behind the dorsal
// fin: soft at its lower edge and at its ends.
func orcaSaddlePatch(p maths.Vec) float64 {
	s := orcaTrunkProfile().at(p.X)
	height := (p.Y - s.centre) / s.high
	// the patch's lower edge climbs from low behind the fin to the back
	edge := .02 + .55*orcaStep(orcaSaddleFront-.2, orcaSaddleRear, p.X) +
		.05*math.Sin(5*p.X+1)
	slant := .6 * (1 - math.Min(height, 1)) // lower down, the front lies further back
	along := orcaStep(orcaSaddleFront-slant, orcaSaddleFront-slant-.25, p.X) *
		orcaStep(orcaSaddleRear, orcaSaddleRear+.35, p.X)
	return along * orcaStep(edge, edge+.3, height)
}

// orcaEyePatch is how far p lies in the white oval above and behind the
// eye, on either flank, with a hard edge.
func orcaEyePatch(p maths.Vec) float64 {
	if orcaFlank(p) == 0 {
		return 0
	}
	dx, dy := p.X-orcaPatchX, p.Y-orcaPatchY
	cosine, sine := math.Cos(orcaPatchTilt), math.Sin(orcaPatchTilt)
	along := (-dx*cosine + dy*sine) / orcaPatchLong
	across := (dx*sine + dy*cosine) / orcaPatchShort
	return orcaFlank(p) * (1 - orcaStep(.9, 1.0, math.Hypot(along, across)))
}

// orcaFlukeUnderside is how far p lies in the white underside of the
// flukes: white below the plane of the flukes, dark along the trailing
// edge, black above.
func orcaFlukeUnderside(p maths.Vec) float64 {
	if p.X > -2.2 || p.Y > orcaFlukeLevel {
		return 0
	}
	_, trail := orcaFlukeEdges(math.Abs(p.Z))
	return orcaStep(trail+.02, trail+.07, p.X)
}

// orcaEyeColour is colour, except inside the eye, where it is the iris
// and the pupil.
func orcaEyeColour(p maths.Vec, colour maths.Light) maths.Light {
	gap := maths.V(p.X, p.Y, math.Abs(p.Z)).Sub(orcaEyeCentre()).Len()
	colour = orcaMix(colour, orcaIris, 1-orcaStep(.024, .028, gap))
	return orcaMix(colour, orcaPupil, 1-orcaStep(.012, .016, gap))
}
