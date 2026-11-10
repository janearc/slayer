package main

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"hash"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/manufacture"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
	"github.com/janearc/slayer/scene/shape"
)

func write(h hash.Hash, values any) {
	if err := binary.Write(h, binary.LittleEndian, values); err != nil {
		panic(err)
	}
}
func vector(h hash.Hash, point maths.Vec) { write(h, [3]float64{point.X, point.Y, point.Z}) }
func space(h hash.Hash, pose maths.Subspace) {
	vector(h, pose.Origin)
	write(h, pose.Scale)
	for _, axis := range []maths.Vec{maths.XAxis, maths.YAxis, maths.ZAxis} {
		vector(h, pose.Orientation.Out(axis))
	}
}
func geometry(h hash.Hash, solid shape.Shape, seen map[shape.Shape]int) {
	if index, exists := seen[solid]; exists {
		write(h, int64(index))
		return
	}
	index := len(seen)
	seen[solid] = index
	write(h, int64(-1-index))
	switch made := solid.(type) {
	case *mesh.Mesh:
		write(h, byte(1))
		write(h, int64(made.FaceCount()))
		for i := range made.FaceCount() {
			a, b, c := made.Corners(i)
			vector(h, a)
			vector(h, b)
			vector(h, c)
			normals, smooth := made.CornerNormals(i)
			write(h, smooth)
			if smooth {
				for _, normal := range normals {
					vector(h, normal)
				}
			}
		}
	case *shape.Mesh:
		write(h, byte(2))
		write(h, int64(len(made.Faces)))
		for _, face := range made.Faces {
			for _, corner := range face {
				vector(h, made.Vertices[corner])
			}
		}
		write(h, made.Smooth)
		write(h, int64(len(made.Normals)))
		for _, normal := range made.Normals {
			vector(h, normal)
		}
	case *shape.MeshGroup:
		members := made.Members()
		write(h, byte(3))
		write(h, int64(len(members)))
		for _, member := range members {
			space(h, member.Space)
			geometry(h, member.Mesh, seen)
		}
	default:
		panic(fmt.Sprintf("unrecorded manufactured shape %T", solid))
	}
}
func main() {
	spec := manufacture.Optical{Diameter: measure.Metres(24), Aperture: measure.Metres(8),
		Opening: measure.Degrees(35), Azimuth: measure.Degrees(180),
		Altitude: measure.Degrees(55), Segmented: false}
	row, err := manufacture.Observatory(spec)
	if err != nil {
		panic(err)
	}
	for _, level := range manufacture.Complexities() {
		made, err := manufacture.MakeWithComplexity(row, manufacture.Sow(5), level)
		if err != nil {
			panic(err)
		}
		things, err := made.Things(maths.Subspace{})
		if err != nil {
			panic(err)
		}
		h := sha256.New()
		bounds := made.Bounds()
		vector(h, bounds.Min)
		vector(h, bounds.Max)
		write(h, int64(len(things)))
		seen := map[shape.Shape]int{}
		for _, thing := range things {
			space(h, thing.At(0))
			geometry(h, thing.Shape, seen)
			material := thing.Material
			if material == nil {
				write(h, byte(0))
				continue
			}
			write(h, byte(1))
			write(h, int64(len(material.Name)))
			_, err = h.Write([]byte(material.Name))
			if err != nil {
				panic(err)
			}
			write(h, [8]float64{material.Colour.R, material.Colour.G, material.Colour.B,
				material.Glow.R, material.Glow.G, material.Glow.B, material.Gloss, material.Index})
		}

		fmt.Printf("%s %x\n", level, h.Sum(nil))
	}
}
