//go:build darwin

package gpu

import (
	"errors"
	"math"
	"testing"

	"github.com/janearc/slayer/geom"
	"github.com/janearc/slayer/manufacture"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/noise"
	"github.com/janearc/slayer/metal"
	"github.com/janearc/slayer/scene/shape"
)

// probeConifer asks the kernel's conifer_hit, one ray a pixel
const probeConifer = `
kernel void probe_conifer(device const float4 *rays [[buffer(0)]],
                          device const Tree *trees [[buffer(1)]],
                          device float *out [[buffer(2)]],
                          constant uint2 &origin [[buffer(3)]],
                          uint2 at [[thread_position_in_grid]]) {
	float3 start = rays[at.x * 2].xyz, way = rays[at.x * 2 + 1].xyz;
	out[at.x] = conifer_hit(start, way, trees[0], 0, nowhere).distance;
}
`

// vec3 is manufacture's kind of point as a maths.Vec.
func vec3(at geom.Vec3) maths.Vec { return maths.V(at[0], at[1], at[2]) }

// meshOfConifer is manufacture's conifer as one shape.Mesh, in the
// conifer's own space
func meshOfConifer(t *testing.T,
	params manufacture.ConiferParams) *shape.Mesh {
	t.Helper()
	var points []maths.Vec
	var corners [][3]int
	for _, face := range manufacture.Conifer(geom.Vec3{}, params) {
		first := len(points)
		corners = append(corners, [3]int{first, first + 1, first + 2})
		for _, corner := range face.V {
			points = append(points, vec3(corner))
		}
	}
	mesh, err := shape.NewMesh(points, corners, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	return mesh
}

// treeBytes is a conifer's parameters as the kernel reads a Tree
func treeBytes(params manufacture.ConiferParams) []byte {
	var trees packer
	trees.row(treeStruct, values{
		"origin": {0, 0, 0, 1}, "x": {1, 0, 0}, "y": {0, 1, 0},
		"z": {0, 0, 1},
		"shape": {params.Height, params.BaseRadius, params.TaperRate,
			params.DepthScale},
		"lean": {params.BranchStart},
		"info": wholes(params.Depth, params.Sides, 2)})
	return trees.bytes
}

// aimedAtTheTree is rays from a sphere about the tree, at points in a
// box round it
func aimedAtTheTree(count int, params manufacture.ConiferParams) []maths.Ray {
	rays := make([]maths.Ray, 0, count)
	wide := params.BaseRadius * 2
	for index := range int64(count) {
		draw := func(which int64) float64 {
			return noise.Hash(9, index*8+which)
		}
		turn, lift := 2*math.Pi*draw(0), math.Acos(2*draw(1)-1)
		out := maths.V(math.Sin(lift)*math.Cos(turn),
			math.Sin(lift)*math.Sin(turn), math.Cos(lift))
		to := maths.V((draw(2)*2-1)*wide, (draw(3)*2-1)*wide,
			draw(4)*params.Height*1.4)
		from := out.Scale(4 * params.Height)
		ray := maths.Ray{At: from, Dir: to.Sub(from).Unit()}
		rays = append(rays, ray)
	}
	return rays
}

// againstMesh is how many rays hit the mesh, and how many the kernel sees
// differently: a hit it misses or a miss it hits, or a hit further off
// than a part in a thousand
func againstMesh(mesh *shape.Mesh, rays []maths.Ray, got []float32) (int, int) {
	hits, differ := 0, 0
	for index, ray := range rays {
		want, hit := mesh.Hit(ray, 0, 1e30)
		if hit {
			hits++
		}
		off := math.Abs(float64(got[index]) - want)
		if hit != (got[index] >= 0) || (hit && off > 1e-3*want) {
			differ++
		}
	}
	return hits, differ
}

// offTheMesh is the share of rays allowed to miss manufacture's mesh, one
// in this many: the kernel is in float and the mesh test in double, so
// a ray along an edge may fall either side. measured on an M3 Max at
// 30000 rays a depth, none did at any depth; this allows three.
const offTheMesh = 10000

// the kernel's conifer is manufacture's: fired at, it is hit where the
// mesh of manufacture.Conifer is hit, and at the same distance
func TestTheKernelsConiferIsManufacturesConifer(t *testing.T) {
	kernel, err := metal.Open(source()+probeConifer, "probe_conifer")
	if errors.Is(err, metal.ErrNoDevice) {
		t.Skip("no gpu here")
	}
	if err != nil {
		t.Fatal(err)
	}
	defer kernel.Close()
	for _, depth := range []int{0, 1, 3, 5, mostConiferDepth} {
		params := manufacture.DefaultConiferParams()
		params.Depth = depth
		mesh := meshOfConifer(t, params)
		rays := aimedAtTheTree(30000, params)
		got, err := kernel.Grid(len(rays), 1, 1, 1,
			[][]byte{rayBytes(rays), treeBytes(params)})
		if err != nil {
			t.Fatal(err)
		}
		hits, differ := againstMesh(mesh, rays, got)
		if hits < len(rays)/10 || hits > len(rays)*9/10 {
			t.Errorf("depth %d: %d of %d hit: it needs both", depth,
				hits, len(rays))
		}
		if differ > len(rays)/offTheMesh {
			t.Errorf("depth %d: %d of %d off the mesh", depth,
				differ, len(rays))
		}
	}
}

// probeConiferHit asks conifer_hit for the whole hit: how far, the
// material, and the way the face it met points
const probeConiferHit = `
kernel void probe_conifer_hit(device const float4 *rays [[buffer(0)]],
                              device const Tree *trees [[buffer(1)]],
                              device float *out [[buffer(2)]],
                              constant uint2 &origin [[buffer(3)]],
                              uint2 at [[thread_position_in_grid]]) {
	float3 start = rays[at.x * 2].xyz, way = rays[at.x * 2 + 1].xyz;
	Hit hit = conifer_hit(start, way, trees[0], 0, nowhere);
	device float *one = out + at.x * 5;
	one[0] = hit.distance;
	one[1] = float(hit.material);
	one[2] = hit.normal.x;
	one[3] = hit.normal.y;
	one[4] = hit.normal.z;
}
`

// nearestFace is the nearest face a ray meets, by trying every face in
// double precision, and the way that face points: the corners' cross,
// as the kernel makes it. a ray that meets nothing has a distance of -1.
func nearestFace(faces []geom.Triangle, ray maths.Ray) (float64, maths.Vec) {
	best, normal := -1.0, maths.Vec{}
	for _, face := range faces {
		corner := vec3(face.V[0])
		first := vec3(face.V[1]).Sub(corner)
		second := vec3(face.V[2]).Sub(corner)
		across := ray.Dir.Cross(second)
		det := first.Dot(across)
		if math.Abs(det) < 1e-12 {
			continue
		}
		from := ray.At.Sub(corner)
		u := from.Dot(across) / det
		up := from.Cross(first)
		v := ray.Dir.Dot(up) / det
		distance := second.Dot(up) / det
		if u < 0 || v < 0 || u+v > 1 || distance <= 0 ||
			(best >= 0 && distance >= best) {
			continue
		}
		best, normal = distance, first.Cross(second).Unit()
	}
	return best, normal
}

// the kernel's conifer says which way the face it met points, and what
// it is made of, as the nearest of manufacture's own faces does
func TestTheKernelsConiferHasTheNormalsOfItsFaces(t *testing.T) {
	kernel, err := metal.Open(source()+probeConiferHit, "probe_conifer_hit")
	if errors.Is(err, metal.ErrNoDevice) {
		t.Skip("no gpu here")
	}
	if err != nil {
		t.Fatal(err)
	}
	defer kernel.Close()
	params := manufacture.DefaultConiferParams()
	params.Depth = 3
	faces := manufacture.Conifer(geom.Vec3{}, params)
	rays := aimedAtTheTree(3000, params)
	got, err := kernel.Grid(len(rays), 1, 1, 5,
		[][]byte{rayBytes(rays), treeBytes(params)})
	if err != nil {
		t.Fatal(err)
	}
	checked, wrong := 0, 0
	for index, ray := range rays {
		distance, normal := nearestFace(faces, ray)
		seen := got[index*5 : index*5+5]
		if distance < 0 || off(seen[0], distance) > 1e-3*distance {
			continue
		}
		checked++
		if !points(seen[2:5], normal) || seen[1] != 2 {
			wrong++
		}
	}
	if checked < 300 {
		t.Fatalf("only %d hits where both agree: too few", checked)
	}
	if wrong > 0 {
		t.Errorf("%d of %d: wrong normal or material", wrong, checked)
	}
}

// off is how far a float is from a number
func off(got float32, want float64) float64 {
	return math.Abs(float64(got) - want)
}

// points is whether three floats are the direction, to within a hair
func points(got []float32, want maths.Vec) bool {
	across := float64(got[0])*want.X + float64(got[1])*want.Y +
		float64(got[2])*want.Z
	return across > 0.9999
}
