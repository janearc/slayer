package manufacture

import (
	"math"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
)

// sealMold gathers the small closed pieces of one material, such as
// corneas, lips and claws, into one mesh with smooth shading.
type sealMold struct {
	faces   surface
	normals [][3]maths.Vec
}

// mesh is the gathered pieces as one mesh, or nil when nothing was added.
func (mold *sealMold) mesh() (*mesh.Mesh, error) {
	if len(mold.faces) == 0 {
		return nil, nil
	}
	return mold.faces.shaded(mold.normals)
}

// sealFrame is a right-handed frame whose Z axis is normal, and whose X
// axis is along as near as it can lie across that normal.
func sealFrame(normal, along maths.Vec) maths.Orientation {
	z := normal.Unit()
	x := along.Sub(z.Scale(along.Dot(z)))
	if x.Len() < sealFrameLeast {
		x = z.Cross(maths.Up)
	}
	if x.Len() < sealFrameLeast {
		x = z.Cross(maths.V(1, 0, 0))
	}
	x = x.Unit()
	return maths.Orientation{X: x, Y: z.Cross(x), Z: z}
}

// sealFrameLeast is the shortest an along direction may be, once its part
// along the normal is taken away, before another is chosen.
const sealFrameLeast = 1e-6

// ellipsoid adds a closed ellipsoid with the given centre, turned by
// turn, whose half-lengths along the turned axes are radii. Fine is the
// cut of the ball it is made from.
func (mold *sealMold) ellipsoid(centre maths.Vec, turn maths.Orientation,
	radii maths.Vec, fine int,
) {
	for _, one := range ball(cut{around: fine}) {
		var placed face
		var shading [3]maths.Vec
		for corner, point := range one {
			placed[corner] = centre.Add(turn.Out(maths.V(
				point.X*radii.X, point.Y*radii.Y, point.Z*radii.Z)))
			shading[corner] = turn.Out(maths.V(
				point.X/radii.X, point.Y/radii.Y, point.Z/radii.Z)).Unit()
		}
		mold.faces = append(mold.faces, placed)
		mold.normals = append(mold.normals, shading)
	}
}

// Cuts of a lip: how many steps run round its outline and round its tube.
const (
	sealLipAround = 18
	sealLipTube   = 6
)

// lip adds a closed ring, a torus whose centre line is an ellipse with
// half-lengths across and tall in the local X and Y axes, with a tube of
// the given radius. Local Z is the way the ring faces.
func (mold *sealMold) lip(centre maths.Vec, turn maths.Orientation,
	across, tall, tube float64,
) {
	point := func(step, round int) (maths.Vec, maths.Vec) {
		angle := 2 * math.Pi * float64(step) / sealLipAround
		phase := 2 * math.Pi * float64(round) / sealLipTube
		sine, cosine := math.Sincos(angle)
		out := maths.V(cosine, sine, 0)
		normal := out.Scale(math.Cos(phase)).Add(maths.V(0, 0, math.Sin(phase)))
		local := maths.V(across*cosine, tall*sine, 0).Add(normal.Scale(tube))
		return centre.Add(turn.Out(local)), turn.Out(normal)
	}
	for step := range sealLipAround {
		for round := range sealLipTube {
			var corners [4]maths.Vec
			var shading [4]maths.Vec
			for index, offset := range [4][2]int{{0, 0}, {1, 0}, {1, 1}, {0, 1}} {
				corners[index], shading[index] = point(step+offset[0], round+offset[1])
			}
			mold.quad(corners, shading)
		}
	}
}

// quad adds a quadrilateral as two triangles, with its corner normals.
func (mold *sealMold) quad(corners, shading [4]maths.Vec) {
	for _, three := range [2][3]int{{0, 1, 2}, {0, 2, 3}} {
		one := face{corners[three[0]], corners[three[1]], corners[three[2]]}
		if !one.hasArea() {
			continue
		}
		mold.faces = append(mold.faces, one)
		mold.normals = append(mold.normals,
			[3]maths.Vec{shading[three[0]], shading[three[1]], shading[three[2]]})
	}
}

// strand is a tube three-sided, so that a strand costs few faces.
func (mold *sealMold) strand(path []maths.Vec, radius []float64) {
	const sides = 3
	mold.tube(path, radius, sides)
}

// tube adds a closed tube along a polyline, with a radius at each point
// and the given number of sides. The first point may lie inside the skin
// it grows from.
func (mold *sealMold) tube(path []maths.Vec, radius []float64, sides int) {
	rings := make([][]maths.Vec, len(path))
	for index, centre := range path {
		tangent := path[min(index+1, len(path)-1)].Sub(path[max(index-1, 0)]).Unit()
		turn := sealFrame(tangent, maths.Up)
		ring := make([]maths.Vec, sides)
		for side := range ring {
			sine, cosine := math.Sincos(2 * math.Pi * float64(side) / float64(sides))
			ring[side] = centre.Add(turn.Out(maths.V(cosine, sine, 0).Scale(radius[index])))
		}
		rings[index] = ring
	}
	skin := alfaLoft{rings: rings}.skin(cut{})
	mold.faces = append(mold.faces, skin...)
	mold.normals = append(mold.normals, creased(skin)...)
}
