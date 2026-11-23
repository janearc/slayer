package manufacture

import (
	"math"

	"github.com/janearc/slayer/maths"
)

// The mouth, the teeth, the blowhole, the eyes and the belly slits.
// Every number is chosen, in metres in rest space, from the proportions
// of a six metre cow; none is a measurement of an animal.
const (
	orcaMouthFront  = 3.03 // x where the mouth line meets the snout
	orcaMouthCorner = 2.12 // x of the corner of the gape, under the eye
	orcaMouthLow    = .33  // how far below the head's centre the line runs
	orcaMouthLift   = .05  // how far the line turns up at the corner
	orcaMouthWidth  = .02  // half width of the lip groove
	orcaMouthDepth  = .03  // depth of the lip groove
	orcaTeethFirst  = 2.22 // x of the rearmost tooth
	orcaTeethStep   = .065 // spacing of teeth along the jaw
	orcaTeethCount  = 12   // teeth a side, upper and lower in turn: 6 a jaw
	orcaToothRadius = .014 // radius of a tooth at its base
	orcaToothHeight = .034 // height of a tooth
	orcaToothRoot   = .018 // distance of a tooth's base from the mouth line
	orcaToothSink   = .012 // how far a tooth stands inside the skin surface
)

// orcaMouthLine is the height of the mouth line at x.
func orcaMouthLine(x float64) float64 {
	s := orcaTrunkProfile().at(x)
	return s.centre - orcaMouthLow*s.high +
		orcaMouthLift*orcaStep(2.5, orcaMouthCorner, x)
}

// orcaMouth is how deep the lip groove cuts at p: zero away from the
// mouth line, and fading out at the corner and at the snout.
func orcaMouth(p maths.Vec) float64 {
	if p.X < orcaMouthCorner-.1 || p.X > orcaMouthFront+.05 {
		return 0
	}
	s := orcaTrunkProfile().at(p.X)
	side := orcaStep(.45, .75, math.Abs(p.Z)/s.wide)
	ends := orcaStep(orcaMouthCorner-.05, orcaMouthCorner+.05, p.X) *
		orcaStep(orcaMouthFront+.01, orcaMouthFront-.05, p.X)
	gap := math.Abs(p.Y - orcaMouthLine(p.X))
	return orcaMouthDepth * side * ends * orcaSlit(gap, orcaMouthWidth)
}

// orcaTeeth is the distance from p to the nearest tooth: a cone set in
// the lip groove, the teeth of the upper jaw pointing down and those of
// the lower pointing up in turn, so each jaw a side has half of
// orcaTeethCount, six. That is fewer than the 10 to 12 of S3 in
// doc/model/bioform/orca/SOURCES.md; the bone's rows have twelve a
// jaw (orcaToothRows). The skin's count is kept as drawn.
func orcaTeeth(p maths.Vec) float64 {
	last := orcaTeethFirst + orcaTeethStep*(orcaTeethCount-1)
	if p.X < orcaTeethFirst-.04 || p.X > last+.04 {
		return orcaFar
	}
	s := orcaTrunkProfile().at(p.X)
	if math.Abs(p.Z)/s.wide < .6 {
		return orcaFar
	}
	index := math.Round((p.X - orcaTeethFirst) / orcaTeethStep)
	index = math.Min(math.Max(index, 0), orcaTeethCount-1)
	across := orcaTeethFirst + index*orcaTeethStep
	line := orcaMouthLine(across)
	sideways := math.Sqrt(math.Max(0, 1-(line-s.centre)*(line-s.centre)/
		(s.high*s.high))) * s.wide
	direction := 1.
	if int(index)%2 == 0 {
		direction = -1
	}
	base := line - direction*orcaToothRoot
	along := direction * (p.Y - base)
	radius := orcaToothRadius * (1 - along/orcaToothHeight)
	radial := math.Hypot(p.X-across, math.Abs(p.Z)-(sideways-orcaToothSink))
	return math.Max(radial-radius, math.Max(-along, along-orcaToothHeight))
}

// The blowhole: a crescent slit on the midline of the head's top, its
// horns forward, an opening whose inside is painted dark. It lies over
// the back of the braincase, where the nasal passage rises from the
// bony nares at the skull's vertex and leans back; the place is chosen
// from the skull of orca_clean_05_bones.go.
const (
	orcaBlowX     = 1.92 // x of the slit's middle
	orcaBlowHalf  = .10  // half the slit's length across the head
	orcaBlowHorn  = .04  // how far the horns lie ahead of the middle
	orcaBlowWidth = .014 // half width of the slit, a little over a lattice step
	orcaBlowDepth = .05  // depth of the slit
)

// orcaBlowhole is how deep the blowhole slit cuts at p.
func orcaBlowhole(p maths.Vec) float64 {
	across := math.Abs(p.Z)
	if p.X < orcaBlowX-.05 || p.X > orcaBlowX+.1 || across > orcaBlowHalf+.01 {
		return 0
	}
	s := orcaTrunkProfile().at(p.X)
	top := orcaStep(s.centre+s.high-.10, s.centre+s.high-.05, p.Y)
	ends := orcaStep(orcaBlowHalf+.01, orcaBlowHalf-.03, across)
	bend := across / orcaBlowHalf
	gap := math.Abs(p.X - (orcaBlowX + orcaBlowHorn*bend*bend))
	return orcaBlowDepth * top * ends * orcaSlit(gap, orcaBlowWidth)
}

// The eye: small, low on the head, a little ahead of the eye patch.
const (
	orcaEyeX      = 2.07 // x of the eye
	orcaEyeY      = -.03 // height of the eye
	orcaEyeRadius = .034 // radius of the hollow the eye sits in
	orcaEyeDepth  = .018 // depth of that hollow
)

// orcaEyeCentre is where the eye lies on the skin, on the right side;
// the left is the mirror.
func orcaEyeCentre() maths.Vec {
	s := orcaTrunkProfile().at(orcaEyeX)
	rise := (orcaEyeY - s.centre) / s.high
	return maths.V(orcaEyeX, orcaEyeY, s.wide*math.Sqrt(1-rise*rise))
}

// orcaEye is how deep the hollow round the eye cuts at p.
func orcaEye(p maths.Vec) float64 {
	centre := orcaEyeCentre()
	gap := maths.V(p.X, p.Y, math.Abs(p.Z)).Sub(centre).Len()
	if gap > orcaEyeRadius {
		return 0
	}
	t := 1 - (gap/orcaEyeRadius)*(gap/orcaEyeRadius)
	return orcaEyeDepth * t * t
}

// The belly slits of a cow: the genital slit on the midline with a
// mammary slit each side of the end of it, and the anus behind.
const (
	orcaSlitWidth = .018 // half width of each ventral slit
	orcaSlitDepth = .016 // depth of each ventral slit
)

// orcaVentral is how deep the belly slits and the anal dimple cut at p.
func orcaVentral(p maths.Vec) float64 {
	if p.X < -.95 || p.X > -.1 || math.Abs(p.Z) > .15 {
		return 0
	}
	s := orcaTrunkProfile().at(p.X)
	belly := orcaStep(.80, .92, (s.centre-p.Y)/s.high)
	if belly == 0 {
		return 0
	}
	across := math.Abs(p.Z)
	genital := orcaSlit(across, orcaSlitWidth) *
		orcaStep(-.52, -.47, p.X) * orcaStep(-.18, -.23, p.X)
	mammary := orcaSlit(math.Abs(across-.07), orcaSlitWidth) *
		orcaStep(-.77, -.73, p.X) * orcaStep(-.58, -.62, p.X)
	anus := orcaSlit(math.Hypot(p.X+.85, across), 2*orcaSlitWidth)
	return orcaSlitDepth * belly * math.Max(genital, math.Max(mammary, anus))
}

// orcaCarve is how far everything cut into the skin moves the surface
// inward at p, in metres.
func orcaCarve(p maths.Vec) float64 {
	return orcaMouth(p) + orcaBlowhole(p) + orcaEye(p) + orcaVentral(p)
}
