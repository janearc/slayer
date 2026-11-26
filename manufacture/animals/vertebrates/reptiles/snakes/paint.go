package snakes

import (
	"math"

	"github.com/janearc/slayer/maths"
)

// snakePaint reads spatial markings in the static animal's local coordinates.
// Its profile is owned by the material and never mutates during rendering.
type snakePaint struct{ profile Profile }

func (paint snakePaint) At(at maths.Vec) maths.Light { return paint.Within(at, 0) }

func blend(a, b maths.Light, share float64) maths.Light {
	share = math.Max(0, math.Min(1, share))
	return a.Scale(1 - share).Add(b.Scale(share))
}

func smooth(a, b, x float64) float64 {
	u := math.Max(0, math.Min(1, (x-a)/(b-a)))
	return u * u * (3 - 2*u)
}

func periodic(x float64) float64 { return x - math.Floor(x) - .5 }

// Within softens borders by the requested local-space pixel footprint.
func (paint snakePaint) Within(at maths.Vec, footprint float64) maths.Light {
	p := paint.profile
	along := math.Max(0, math.Min(1, at.X/p.Length+.5))
	wide, high := section(p.Stations, along)
	// At the nose pole no circumferential angle is defined.
	if wide <= 0 || high <= 0 {
		return p.Pattern.Base
	}
	angle := math.Atan2(at.Z/(wide*p.Length), at.Y/(high*p.Length))
	turn := angle / math.Pi
	phase := along * p.Pattern.Repeat
	// Width is in pattern-cell units; the cap keeps a distant view bounded.
	feather := math.Min(.35, math.Max(.015, math.Max(0, footprint)*p.Pattern.Repeat/p.Length))
	marks := bodyMarks(p.Pattern.Kind, phase, turn, feather)
	colour := blend(p.Pattern.Base, p.Pattern.Edge, marks.border*p.Pattern.Contrast)
	colour = blend(colour, p.Pattern.Mark, marks.dark*p.Pattern.Contrast)
	colour = blend(colour, p.Pattern.Belly, marks.pale*p.Pattern.Contrast)
	// A pale underside remains legible even where a dorsal mark crosses it.
	belly := 1 - smooth(-.62, -.38, at.Y/(high*p.Length))
	colour = blend(colour, p.Pattern.Belly, belly)

	// Gaboon-style dark facial triangles lie under both eyes. They are a
	// fixed illustrative marking, not geometry or a claim of a measured shape.
	eyeAngle := math.Acos(p.Head.EyeHeight)
	if p.Pattern.Kind == "triangles" && facialTriangles(p.Head, along, angle) {
		colour = blend(colour, p.Pattern.Mark, p.Pattern.Contrast)
	}
	// Pythons have a series of labial pits. These dark marks indicate five
	// illustrative locations on each lip rather than a single loreal pit.
	if p.Head.Pit {
		start := math.Max(0, p.Head.EyeAt-.035)
		end := math.Min(.998, p.Head.EyeAt+.018)
		for i := 0; i < 5; i++ {
			pitAt := start + (end-start)*float64(i)/4
			d := math.Hypot((along-pitAt)/.0025, (math.Abs(angle)-eyeAngle-.45)/.065)
			colour = blend(colour, maths.RGB(.02, .016, .012), 1-smooth(.7, 1, d))
		}
	}
	return facePaint(p, along, angle, footprint, colour)
}

// facialTriangles places an anterior and a posterior triangular eye mark on
// either side of the Gaboon-style head; their locations are illustrative.
func facialTriangles(head Head, along, angle float64) bool {
	down := (math.Abs(angle) - math.Acos(head.EyeHeight)) / .48
	if down < 0 || down > 1 {
		return false
	}
	centres := [2]float64{math.Max(0, head.EyeAt-.038), math.Min(.995, head.EyeAt+.006)}
	for _, centre := range centres {
		if math.Abs((along-centre)/.013) < 1-down {
			return true
		}
	}
	return false
}
