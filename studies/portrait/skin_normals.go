package main

import (
	"github.com/janearc/slayer/manufacture/geometry"
	"github.com/janearc/slayer/maths"
	"math"
)

// Relax the shading field independently of position and occlusion. Creases
// narrower than a nostril rim, eyelid or lip are protected by the rest mask.
func skinNormals(points, rest []maths.Vec, faces [][3]int) []maths.Vec {
	initial := geometry.UnitNormals(points, faces)
	adjacent := make([][]int, len(points))
	for _, f := range faces {
		for i, a := range f {
			adjacent[a] = append(adjacent[a], f[(i+1)%3], f[(i+2)%3])
		}
	}
	field := initial
	for iteration := 0; iteration < 12; iteration++ {
		next := make([]maths.Vec, len(points))
		for i, old := range field {
			sum := old
			for _, j := range adjacent[i] {
				w := maths.Smooth(.55, .94, initial[i].Dot(initial[j]))
				sum = sum.Add(field[j].Scale(w))
			}
			next[i] = sum.Unit()
		}
		field = next
	}
	for i, p := range rest {
		x := math.Abs(p.X)
		feature := math.Max(gauss((x-.031)/.026)*gauss((p.Y-1.554)/.008), gauss(x/.028)*gauss((p.Y-1.487)/.010))
		feature = math.Max(feature, gauss(x/.021)*gauss((p.Y-1.520)/.005))
		w := .95 * (1 - feature)
		field[i] = initial[i].Scale(1 - w).Add(field[i].Scale(w)).Unit()
	}
	return field
}
