//go:build darwin

package gpu

import (
	"encoding/binary"
	"errors"
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/noise"
	"github.com/janearc/slayer/metal"
	"github.com/janearc/slayer/scene/shape"
)

// probeCone asks the kernel's cone_hit, one ray a pixel: a ray is its
// start and its way, and the cone's height, radius and sides come first
const probeCone = `
kernel void probe_cone(device const float4 *rays [[buffer(0)]],
                       device const float4 *cone [[buffer(1)]],
                       device float *out [[buffer(2)]],
                       constant uint2 &origin [[buffer(3)]],
                       uint2 at [[thread_position_in_grid]]) {
	float3 start = rays[at.x * 2].xyz, way = rays[at.x * 2 + 1].xyz;
	out[at.x] = cone_hit(start, way, cone[0].x, cone[0].y, int(cone[0].z),
	                     0, nowhere).distance;
}
`

// coneFaces is the faces of a cone as manufacture.Cone makes them: for
// each side, the base ring's edge and the apex over the middle
func coneFaces(height, radius float64, sides int) [][3]maths.Vec {
	apex := maths.V(0, 0, height)
	ring := func(side int) maths.Vec {
		turn := 2 * math.Pi * float64(side) / float64(sides)
		return maths.V(math.Cos(turn)*radius, math.Sin(turn)*radius, 0)
	}
	faces := make([][3]maths.Vec, 0, sides)
	for side := range sides {
		corners := [3]maths.Vec{ring(side), ring(side + 1), apex}
		faces = append(faces, corners)
	}
	return faces
}

// aimedAtTheCone is count rays from a sphere about the cone, each at a
// point in a box a little wider than the cone: some hit, some do not
func aimedAtTheCone(count int, height, radius float64) []maths.Ray {
	rays := make([]maths.Ray, 0, count)
	for index := range int64(count) {
		draw := func(which int64) float64 {
			return noise.Hash(5, index*8+which)
		}
		turn, lift := 2*math.Pi*draw(0), math.Acos(2*draw(1)-1)
		out := maths.V(math.Sin(lift)*math.Cos(turn),
			math.Sin(lift)*math.Sin(turn), math.Cos(lift))
		wide := radius * 1.3
		to := maths.V((draw(2)*2-1)*wide, (draw(3)*2-1)*wide,
			draw(4)*height*1.1)
		from := out.Scale(4 * height)
		ray := maths.Ray{At: from, Dir: to.Sub(from).Unit()}
		rays = append(rays, ray)
	}
	return rays
}

// rayBytes is the rays as the probe reads them
func rayBytes(rays []maths.Ray) []byte {
	var bytes []byte
	for _, ray := range rays {
		start := append(point(ray.At), 0)
		way := append(point(ray.Dir), 0)
		bytes = append(bytes, floatBytes(start...)...)
		bytes = append(bytes, floatBytes(way...)...)
	}
	return bytes
}

// floatBytes is numbers as the probe reads them
func floatBytes(numbers ...float64) []byte {
	var bytes []byte
	for _, number := range numbers {
		bytes = binary.LittleEndian.AppendUint32(bytes,
			math.Float32bits(float32(number)))
	}
	return bytes
}

// the kernel's cone is a shape.Mesh of the same faces: the same rays hit
// it or miss it, and at the same distance. a ray along an edge may fall
// either side, so a few in a thousand may differ.
func TestTheKernelsConeIsTheMeshOfItsFaces(t *testing.T) {
	height, radius, sides := 10.0, 2.0, 8
	faces := coneFaces(height, radius, sides)
	var points []maths.Vec
	var corners [][3]int
	for _, face := range faces {
		corners = append(corners, [3]int{len(points), len(points) + 1,
			len(points) + 2})
		points = append(points, face[:]...)
	}
	mesh, err := shape.NewMesh(points, corners, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	kernel, err := metal.Open(source()+probeCone, "probe_cone")
	if errors.Is(err, metal.ErrNoDevice) {
		t.Skip("no gpu here")
	}
	if err != nil {
		t.Fatal(err)
	}
	defer kernel.Close()
	rays := aimedAtTheCone(20000, height, radius)
	got, err := kernel.Grid(len(rays), 1, 1, 1, [][]byte{rayBytes(rays),
		floatBytes(height, radius, float64(sides), 0)})
	if err != nil {
		t.Fatal(err)
	}
	hits, differ := 0, 0
	for index, ray := range rays {
		want, hit := mesh.Hit(ray, 0, 1e30)
		if hit {
			hits++
		}
		far := hit && math.Abs(float64(got[index])-want) > 1e-3*want
		if hit != (got[index] >= 0) || far {
			differ++
		}
	}
	if hits < 3000 || hits > len(rays)-3000 {
		t.Errorf("%d of %d rays hit: it needs hits and misses", hits,
			len(rays))
	}
	if differ > len(rays)/500 {
		t.Errorf("%d of %d rays differ from the mesh's", differ,
			len(rays))
	}
}
