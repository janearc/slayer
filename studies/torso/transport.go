package main

import (
	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/manufacture/animals/vertebrates/mammals/primates"
	"github.com/janearc/slayer/manufacture/geometry"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
	framebuffer "github.com/janearc/slayer/render/frame"
	"github.com/janearc/slayer/scene/thing"
	"math"
)

// materialStudy reconstructs incident diffuse light on the entire posed mesh.
// Only lookup of the final response uses camera visibility. Adjacent back-facing
// or off-screen facets still participate in the dermal diffusion neighbourhood.
func materialStudy(w *thing.World, eye maths.Subspace, lens film.Sensor, width, height int, s primates.Torso, key maths.Vec) (*framebuffer.Frame, error) {
	part := w.Things[0]
	skin := part.Shape.(*mesh.Mesh)
	points := skin.Points()
	faces := skin.Faces()
	normals := geometry.UnitNormals(points, faces)
	radiance := make([]maths.Light, len(points))
	sun := w.Lamps[0].(film.Sun)
	dirs := sun.Spread()
	for i, p := range points {
		n := normals[i]
		irr := maths.Grey(.32)
		square := math.Max(0, key.Unit().Dot(n))
		if square > 0 {
			visible := 0
			for _, dir := range dirs {
				ray := maths.Ray{At: p.Add(n.Scale(.00002)), Dir: dir}
				if _, hit := skin.Meet(ray, .000001, 3); !hit {
					visible++
				}
			}
			irr = irr.Add(sun.Strength.Scale(square * float64(visible) / (math.Pi * float64(len(dirs)))))
		}
		radiance[i] = irr
	}
	// The study's diffusion radii are homogeneous across this specimen. Use
	// the optical response as their single source, including the grafted arms.
	transfer := diffusionSteps(points, faces, radiance, s.Optics(maths.Vec{}, 431).Radius)
	// Embed the field in duplicate barycentric attributes on the same triangles;
	// the traced image keeps its native surface, specular and occlusion terms.
	field, err := mesh.NewWithRestPoints(points, transfer, faces, nil, false)
	must(err)
	dry := *part.Material
	dry.Satin = .77
	part.Material = &dry
	picture, err := traceFrame(w, eye, lens, width, height)
	must(err)
	oil := dry
	oil.Satin = .59
	part.Material = &oil
	moist, err := traceFrame(w, eye, lens, width, height)
	must(err)
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			ray := film.Look(eye, lens, x, y, width, height, maths.Band{})
			hit, ok := skin.Meet(ray, 0, 5)
			if !ok {
				continue
			}
			rest, _ := skin.RestPointAt(hit)
			d, _ := field.RestPointAt(hit)
			op := s.Optics(rest, 431)
			k := y*width + x
			pixel := &picture.Pix[k]
			wet := moist.Pix[k]
			mix := float32(math.Max(0, math.Min(1, (.77-op.Roughness)/.18)))
			pixel.R += mix*(wet.R-pixel.R) + float32(op.DermalShare*op.Colour.R*d.X)
			pixel.G += mix*(wet.G-pixel.G) + float32(op.DermalShare*op.Colour.G*d.Y)
			pixel.B += mix*(wet.B-pixel.B) + float32(op.DermalShare*op.Colour.B*d.Z)
		}
	}
	return picture, nil
}
