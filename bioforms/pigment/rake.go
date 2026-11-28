package pigment

import (
	"fmt"
	"math"

	"github.com/janearc/slayer/maths"
)

// ScarRake describes a group of roughly parallel healed scratch marks.
// Across sets the spacing direction. Along is projected perpendicular to
// Across to orient the marks. Depth is the half-depth of the projection.
// Length and Radius are the largest mark length and half-width, respectively.
// All distances use the same units as Centre in the specimen's rest chart.
type ScarRake struct {
	Centre, Across, Along          maths.Vec
	Seed                           uint64
	Count                          int
	Length, Spacing, Radius, Depth float64
}

type scarRakeMask struct {
	centre, across, along, normal maths.Vec
	width, length, radius, depth  float64
	marks                         []Mask
}

// NewScarRake generates a seeded group of curved, separated strokes on a
// local plane. Choose a depth that reaches the intended curved surface
// while excluding nearby skin. The mask is ready for a Scar paint layer.
func NewScarRake(spec ScarRake) (Mask, error) {
	across := spec.Across.Unit()
	normal := across.Cross(spec.Along.Unit())
	if !finite(spec.Centre) || !finite(across) || !(normal.Len() > 1e-8) ||
		spec.Count < 1 || !positive(spec.Length) || !positive(spec.Spacing) ||
		!positive(spec.Radius) || !positive(spec.Depth) {
		return nil, fmt.Errorf("pigment: scar rake needs a finite centre, independent axes, and positive count and dimensions")
	}
	normal = normal.Unit()
	rake := scarRakeMask{
		centre: spec.Centre, across: across, along: normal.Cross(across),
		normal: normal, radius: spec.Radius, depth: spec.Depth,
		marks: make([]Mask, 0, spec.Count),
	}
	h := spec.Seed
	draw := func() float64 {
		h = scramble(h + 1)
		return fraction(h)
	}
	for i := range spec.Count {
		a := (float64(i) - float64(spec.Count-1)/2 + .24*(draw()-.5)) * spec.Spacing
		b := .22 * (draw() - .5) * spec.Length
		halfLength := (.34 + .16*draw()) * spec.Length
		drift := .35 * (draw() - .5) * spec.Spacing
		bow := (.12 + .24*draw()) * spec.Spacing
		radius := (.58 + .42*draw()) * spec.Radius
		var points [7]maths.Vec
		for k := range points {
			t := 2*float64(k)/float64(len(points)-1) - 1
			x := a + drift*t + bow*(1-t*t)
			y := b + halfLength*t
			points[k] = maths.V(x, y, 0)
			rake.width = max(rake.width, math.Abs(x))
			rake.length = max(rake.length, math.Abs(y))
		}
		mark, err := newStroke(Stroke{Points: points[:], Radius: radius, Feather: .7}, .76)
		if err != nil {
			return nil, err
		}
		rake.marks = append(rake.marks, mark)
	}
	return rake, nil
}

func (rake scarRakeMask) Coverage(point maths.Vec, footprint float64) float64 {
	delta := point.Sub(rake.centre)
	depth := 1 - maths.Smooth(.65, 1, math.Abs(delta.Dot(rake.normal))/rake.depth)
	if !(depth > 0) {
		return 0
	}
	x, y := delta.Dot(rake.across), delta.Dot(rake.along)
	padding := rake.radius + max(0, footprint/2)
	if math.Abs(x) > rake.width+padding || math.Abs(y) > rake.length+padding {
		return 0
	}
	local := maths.V(x, y, 0)
	var coverage float64
	for _, mark := range rake.marks {
		coverage = max(coverage, mark.Coverage(local, footprint))
	}
	return depth * coverage
}
