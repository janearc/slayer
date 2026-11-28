package main

import (
	"github.com/janearc/slayer/bioforms/diffusion"
	"github.com/janearc/slayer/maths"
)

func diffusionSteps(points []maths.Vec, faces [][3]int, incident []maths.Light, radii [3]float64) []maths.Vec {
	change, err := diffusion.SurfaceDelta(diffusion.Surface{
		Points: points, Faces: faces, Incident: incident, Radius: radii,
	})
	must(err)
	return change
}
