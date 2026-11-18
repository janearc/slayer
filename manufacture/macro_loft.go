package manufacture

import (
	"math"

	"github.com/janearc/slayer/maths"
)

// macroLoft is a closed annular rectangular loft with a real open core.
// Straight loft segments interpolate the explicitly tapered, twisted rings.
type macroLoft struct {
	width, depth, innerWidth, innerDepth, height float64
	taper, twist, phase                          float64
	segments                                     int
}

// corners gives an exact horizontal ring, with local y proportional to t.
func (loft macroLoft) corners(t float64, inner bool) [4]maths.Vec {
	width, depth := loft.width, loft.depth
	if inner {
		width, depth = loft.innerWidth, loft.innerDepth
	}
	scale := 1 - loft.taper*t
	turn := loft.phase + loft.twist*t
	c, s := math.Cos(turn), math.Sin(turn)
	var points [4]maths.Vec
	for i, p := range [][2]float64{{-1, -1}, {1, -1}, {1, 1}, {-1, 1}} {
		x, z := p[0]*width*scale/2, p[1]*depth*scale/2
		points[i] = maths.V(c*x-s*z, t*loft.height, s*x+c*z)
	}
	return points
}

// skin joins outward walls, reversed inner walls and annular end caps.
func (loft macroLoft) skin(cut) surface {
	var skin surface
	for level := range loft.segments {
		t0, t1 := float64(level)/float64(loft.segments), float64(level+1)/float64(loft.segments)
		for _, inner := range []bool{false, true} {
			low, high := loft.corners(t0, inner), loft.corners(t1, inner)
			for i, a := range low {
				j := (i + 1) % 4
				b, c, d := low[j], high[i], high[j]
				pair := surface{{a, c, b}, {b, c, d}}
				if inner {
					for f := range pair {
						pair[f][1], pair[f][2] = pair[f][2], pair[f][1]
					}
				}
				skin = append(skin, pair...)
			}
		}
	}
	for _, t := range []float64{0, 1} {
		outer, inner := loft.corners(t, false), loft.corners(t, true)
		for i, a := range outer {
			j := (i + 1) % 4
			pair := surface{{a, inner[i], inner[j]}, {a, inner[j], outer[j]}}
			if t == 0 {
				for f := range pair {
					pair[f][1], pair[f][2] = pair[f][2], pair[f][1]
				}
			}
			skin = append(skin, pair...)
		}
	}
	return skin
}
