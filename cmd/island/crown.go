package main

import (
	"fmt"
	"math"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
)

// crown bins the component bounds of an existing clump specimen by height and
// quadrant. It never expands the triangles of an instanced branch.
// Each occupied bin becomes an octahedron with the original material. This is
// an authored distant silhouette: it fills small gaps, but keeps crown height,
// asymmetric lobes, bark colour and ordinary GPU shadow/reflection geometry.
func crown(source prototype) (prototype, int, error) {
	if source.bounds.Max.Y <= source.bounds.Min.Y {
		return prototype{}, 0, fmt.Errorf("island: crown needs a specimen with height")
	}
	var parts []thing.Thing
	bound := crownBin{bounds: source.bounds, used: true}
	faces := 0
	for _, part := range source.parts {
		var bins [32]crownBin // Eight height slices, four horizontal quadrants.
		visit := func(point maths.Vec) {
			height := (point.Y - source.bounds.Min.Y) / (source.bounds.Max.Y - source.bounds.Min.Y)
			index := min(7, max(0, int(8*height))) * 4
			if point.X >= 0 {
				index++
			}
			if point.Z >= 0 {
				index += 2
			}
			bins[index].include(point)
		}
		if err := crownPoints(part.Shape, part.At(0), visit); err != nil {
			return prototype{}, 0, err
		}
		var points []maths.Vec
		var triangles [][3]int
		for _, bin := range bins {
			bin.append(&points, &triangles)
		}
		if len(triangles) == 0 {
			return prototype{}, 0, fmt.Errorf("island: crown material has no spatial extent")
		}
		made, err := mesh.New(points, triangles, nil, false)
		if err != nil {
			return prototype{}, 0, err
		}
		parts = append(parts, thing.Thing{Shape: made, Material: part.Material})
		bound.include(made.Bounds().Min)
		bound.include(made.Bounds().Max)
		faces += made.FaceCount()
	}
	return prototype{parts: parts, bounds: bound.bounds}, faces, nil
}

// crownPoints visits the eight corners of each flat clump's bounds. These
// envelopes fill small branch gaps, intentionally, at distant viewing scales.
// Groups contain flat meshes, never other groups.
func crownPoints(solid shape.Shape, space maths.Subspace, visit func(maths.Vec)) error {
	switch value := solid.(type) {
	case *mesh.Mesh:
		box := value.Bounds()
		for _, x := range [2]float64{box.Min.X, box.Max.X} {
			for _, y := range [2]float64{box.Min.Y, box.Max.Y} {
				for _, z := range [2]float64{box.Min.Z, box.Max.Z} {
					visit(space.Out(maths.V(x, y, z)))
				}
			}
		}
	case *shape.MeshGroup:
		for _, member := range value.Members() {
			if err := crownPoints(member.Mesh, member.Space.Within(space), visit); err != nil {
				return err
			}
		}
	default:
		return fmt.Errorf("island: unsupported crown source %T", solid)
	}
	return nil
}

type crownBin struct {
	bounds maths.Box
	used   bool
}

// include widens an occupied bin without inventing bounds for an empty one.
func (bin *crownBin) include(point maths.Vec) {
	if !bin.used {
		bin.bounds, bin.used = maths.Box{Min: point, Max: point}, true
		return
	}
	low, high := bin.bounds.Min, bin.bounds.Max
	bin.bounds = maths.Box{Min: maths.V(math.Min(low.X, point.X), math.Min(low.Y, point.Y), math.Min(low.Z, point.Z)),
		Max: maths.V(math.Max(high.X, point.X), math.Max(high.Y, point.Y), math.Max(high.Z, point.Z))}
}

// append gives a bin eight outward faces. Planar bins have no volume and are
// omitted rather than inflated by an arbitrary world-scale epsilon.
func (bin crownBin) append(points *[]maths.Vec, triangles *[][3]int) {
	if !bin.used {
		return
	}
	centre := bin.bounds.Min.Add(bin.bounds.Max).Scale(0.5)
	radius := bin.bounds.Max.Sub(bin.bounds.Min).Scale(0.5)
	if radius.X <= 0 || radius.Y <= 0 || radius.Z <= 0 {
		return
	}
	first := len(*points)
	for _, offset := range []maths.Vec{maths.V(radius.X, 0, 0), maths.V(0, 0, -radius.Z),
		maths.V(-radius.X, 0, 0), maths.V(0, 0, radius.Z), maths.V(0, radius.Y, 0), maths.V(0, -radius.Y, 0)} {
		*points = append(*points, centre.Add(offset))
	}
	for side := 0; side < 4; side++ {
		next := (side + 1) % 4
		*triangles = append(*triangles, [3]int{first + side, first + next, first + 4},
			[3]int{first + next, first + side, first + 5})
	}
}
