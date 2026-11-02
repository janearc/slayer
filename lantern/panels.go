package lantern

import (
	"math"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/noise"
	"github.com/janearc/slayer/mesh"
)

// orbit is one tilted ring the panels sit on: its pole, two directions
// across its plane, and where round it the first panel starts
type orbit struct {
	pole, across, onward maths.Vec
	start                float64
}

// orbitOf is the orbit with a number, of count: its pole one of count
// spread as evenly as a golden spiral spreads them over half the sphere,
// so the rings cross as little as rings can (a pole and its opposite
// make one ring).
//
// where round it the first panel starts is a hash, so no two rings are
// in step.
func orbitOf(number, count int) orbit {
	id := int64(number)
	height := 1 - (float64(number)+0.5)/float64(count)
	turn := float64(number) * math.Pi * (3 - math.Sqrt(5))
	flat := math.Sqrt(1 - height*height)
	pole := maths.V(flat*math.Cos(turn), height, flat*math.Sin(turn))
	reference := maths.V(0, 1, 0)
	if math.Abs(pole.Y) > 0.9 {
		reference = maths.V(1, 0, 0)
	}
	across := pole.Cross(reference).Unit()
	return orbit{pole: pole, across: across,
		onward: pole.Cross(across).Unit(),
		start:  2 * math.Pi * noise.Hash(13, id)}
}

// tumbleAxis is the axis a ring tumbles about: across its band, in the
// ring's own plane, turned to a side by a hash, so the rings tumble
// every which way
func (ring orbit) tumbleAxis(number int) maths.Vec {
	turn := 2 * math.Pi * noise.Hash(16, int64(number))
	return ring.across.Scale(math.Cos(turn)).
		Add(ring.onward.Scale(math.Sin(turn)))
}

// panel is one panel's four corners on an orbit, the panel numbered
// index of count, and the way out from the star at its middle.
//
// the panels tile the ring end to end with a tenth of each left open,
// so light comes between them; wide is how far each reaches toward the
// poles. each sits a little nearer or further than the rest.
func (ring orbit) panel(number, index, count int, shell,
	wide float64) ([4]maths.Vec, maths.Vec) {
	id := int64(number*100000 + index)
	angle := ring.start + 2*math.Pi*float64(index)/float64(count)
	radius := shell * (1 + 0.04*(noise.Hash(15, id)-0.5))
	out := ring.across.Scale(math.Cos(angle)).
		Add(ring.onward.Scale(math.Sin(angle)))
	centre := out.Scale(radius)
	long := 0.9 * 2 * math.Pi * radius / float64(count)
	along := ring.across.Scale(-math.Sin(angle)).
		Add(ring.onward.Scale(math.Cos(angle))).Scale(long / 2)
	up := ring.pole.Scale(wide / 2)
	return [4]maths.Vec{centre.Sub(along).Sub(up),
		centre.Add(along).Sub(up), centre.Add(along).Add(up),
		centre.Sub(along).Add(up)}, out
}

// facingIn is a square's two faces from its first corner, wound so each
// face's normal points in toward the star
func facingIn(corners []maths.Vec, first int, out maths.Vec) [][3]int {
	faces := [][3]int{{first, first + 1, first + 2},
		{first, first + 2, first + 3}}
	for index, face := range faces {
		one, two, three := corners[face[0]], corners[face[1]],
			corners[face[2]]
		if two.Sub(one).Cross(three.Sub(one)).Dot(out) > 0 {
			faces[index][1], faces[index][2] = face[2], face[1]
		}
	}
	return faces
}

// ringMesh is one orbit's panels as one mesh: spaced round it, each a
// small square side across facing the star, shell out from it.
func ringMesh(ring orbit, number, panels int, shell, side,
	wear float64) (*mesh.Mesh, error) {
	var corners []maths.Vec
	var faces [][3]int
	for index := range panels {
		square, out := ring.panel(number, index, panels, shell, side)
		first := len(corners)
		if wear <= 0 {
			corners = append(corners, square[:]...)
			faces = append(faces, facingIn(corners, first, out)...)
			continue
		}
		id := int64(number*100000 + index)
		more, cut := worn(square, out, id, wear)
		corners = append(corners, more...)
		for _, face := range facingInAll(more, cut, out) {
			faces = append(faces, [3]int{face[0] + first,
				face[1] + first, face[2] + first})
		}
	}
	return mesh.New(corners, faces, nil, false)
}

// facingInAll winds every face toward the star
func facingInAll(corners []maths.Vec, faces [][3]int,
	out maths.Vec) [][3]int {
	for index, face := range faces {
		one, two, three := corners[face[0]], corners[face[1]],
			corners[face[2]]
		if two.Sub(one).Cross(three.Sub(one)).Dot(out) > 0 {
			faces[index][1], faces[index][2] = face[2], face[1]
		}
	}
	return faces
}
