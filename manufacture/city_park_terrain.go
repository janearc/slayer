package manufacture

import (
	"math"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/noise"
)

// Park land is an authored two-kilometre maximum after the orbital's 250x
// scale. Its grid is four factory metres per cell or finer in each axis.
const parkTerrainStep = 4.0
const parkTerrainMaxHeight = 8.0
const parkTerrainBase = .0001

// City foliage assigns its material roles dynamically starting at 20.
// Reserve a distant range so distinct plant species cannot replace park paint.
const (
	parkMeadow   slot = 1000000
	parkWoodland slot = 1000001
	parkStone    slot = 1000002
	parkService  slot = 1000003
	parkPaving   slot = 1000004
)

// The paths keep their existing geometry and elevations. A zero-height band
// wider than a cell diagonal plus path half-width keeps every intersecting
// terrain triangle below the entire path, including between its vertices.
const parkPathFlat = 8.0
const parkPathFade = 28.0
const parkEdgeFade = 36.0

// parkTerrainLooks fills the palette with the looks of the park ground roles.
func parkTerrainLooks(palette looks) {
	palette[parkMeadow] = plain("park living meadow", .14, .25, .075)
	palette[parkWoodland] = plain("park woodland undergrowth", .055, .135, .035)
	palette[parkStone] = plain("park weathered soil and stone", .23, .20, .13)
	palette[parkService] = plain("weathered urban service paving", .19, .18, .16)
	palette[parkPaving] = plain("weathered urban stone paving", .30, .28, .24)
}

// parkTerrainGrid returns the cell counts along x and z, and the cell size in
// metres along each.
func parkTerrainGrid(park CityPark) (int, int, float64, float64) {
	w, d := park.Width.InMetres(), park.Depth.InMetres()
	nx, nz := int(math.Ceil(w/parkTerrainStep)), int(math.Ceil(d/parkTerrainStep))
	return nx, nz, w / float64(nx), d / float64(nz)
}

// parkTerrainSeed keeps terrain and cover distinct across separate parks
// without depending on generation order or an external random stream.
func parkTerrainSeed(park CityPark) uint64 {
	return math.Float64bits(park.At.X)*0x9e3779b97f4a7c15 ^ math.Float64bits(park.At.Z)*0xbf58476d1ce4e5b9
}

// parkTerrainVertex is deliberately a function of park and absolute position,
// so independent tree anchoring and mesh generation produce identical heights.
func parkTerrainVertex(park CityPark, paths [][]maths.Vec, x, z float64) float64 {
	w, d := park.Width.InMetres(), park.Depth.InMetres()
	dx, dz := x-park.At.X, z-park.At.Z
	// A small coherent centre contains the camera grove and its avenues.
	if math.Abs(dx) <= 12 && math.Abs(dz) <= 12 {
		return parkTerrainBase
	}
	edge := math.Min(w/2-math.Abs(dx), d/2-math.Abs(dz))
	if edge <= 0 {
		return parkTerrainBase
	}
	path := parkTerrainPathDistance(paths, x, z)
	if path <= parkPathFlat {
		return parkTerrainBase
	}
	fade := maths.Smooth(0, parkEdgeFade, edge) * maths.Smooth(parkPathFlat, parkPathFade, path)
	// Nested hills and low knolls offer depth without tilting nearby roads or
	// burial of the park's independently made paths. Bound remains explicit.
	seed := parkTerrainSeed(park)
	hills := .5 + .5*noise.Fractal(seed+4817, dx/155, 0, dz/155, 4)
	ridges := 1 - math.Abs(noise.Fractal(seed+4823, dx/58, 1, dz/58, 3))
	return parkTerrainBase + parkTerrainMaxHeight*math.Max(0, math.Min(1, (.18+.58*hills+.24*ridges)*fade))
}

// parkTerrainPathDistance reuses one path set across an entire terrain mesh.
func parkTerrainPathDistance(paths [][]maths.Vec, x, z float64) float64 {
	nearest := math.Inf(1)
	for _, path := range paths {
		for i := 1; i < len(path); i++ {
			a, b := path[i-1], path[i]
			dx, dz := b.X-a.X, b.Z-a.Z
			t := math.Max(0, math.Min(1, ((x-a.X)*dx+(z-a.Z)*dz)/(dx*dx+dz*dz)))
			px, pz := x-(a.X+t*dx), z-(a.Z+t*dz)
			nearest = math.Min(nearest, px*px+pz*pz)
		}
	}
	return math.Sqrt(nearest)
}

// CityParkTerrainHeight gives the exact triangulated park surface, in factory
// metres. It is intended for camera and tree placements on an opt-in park.
func CityParkTerrainHeight(park CityPark, at maths.Vec) float64 {
	if !positive(park.Width.InMetres()) || !positive(park.Depth.InMetres()) ||
		math.IsNaN(at.X) || math.IsNaN(at.Z) || math.IsInf(at.X, 0) || math.IsInf(at.Z, 0) {
		return 0
	}
	return parkTerrainHeight(park, at)
}

// CityParkLongPathAt returns the exact point on the relief-enabled park's
// 64-segment long path, with fraction clamped to [0,1]. Coordinates and Y are
// factory metres; the caller applies its city transform and world scale.
func CityParkLongPathAt(park CityPark, fraction float64) maths.Vec {
	if math.IsNaN(fraction) {
		fraction = 0
	}
	fraction = math.Max(0, math.Min(1, fraction))
	path := parkPaths(park)[1]
	along := fraction * float64(len(path)-1)
	index := min(int(along), len(path)-2)
	mix := along - float64(index)
	point := path[index].Scale(1 - mix).Add(path[index+1].Scale(mix))
	point.Y = .0011
	return point
}

// parkTerrainHeight interpolates precisely the same diagonal triangles used
// by parkTerrain, so a placement can stand on the mesh between grid vertices.
func parkTerrainHeight(park CityPark, at maths.Vec) float64 {
	nx, nz, sx, sz := parkTerrainGrid(park)
	x0, z0 := park.At.X-park.Width.InMetres()/2, park.At.Z-park.Depth.InMetres()/2
	u, v := (at.X-x0)/sx, (at.Z-z0)/sz
	if u < 0 || v < 0 || u > float64(nx) || v > float64(nz) {
		return 0
	}
	if u == 0 || v == 0 || u == float64(nx) || v == float64(nz) {
		return parkTerrainBase
	}
	i, j := int(math.Floor(u)), int(math.Floor(v))
	u -= float64(i)
	v -= float64(j)
	x, z := x0+float64(i)*sx, z0+float64(j)*sz
	paths := parkPaths(park)
	a := parkTerrainVertex(park, paths, x, z)
	b := parkTerrainVertex(park, paths, x+sx, z)
	c := parkTerrainVertex(park, paths, x, z+sz)
	d := parkTerrainVertex(park, paths, x+sx, z+sz)
	if u+v <= 1 {
		return a + (b-a)*u + (c-a)*v
	}
	return d + (c-d)*(1-u) + (b-d)*(1-v)
}

// parkTerrainRole picks the ground role at x and z from fractal noise:
// woodland, stone, or meadow.
func parkTerrainRole(park CityPark, x, z float64) slot {
	dx, dz := x-park.At.X, z-park.At.Z
	cover := noise.Fractal(parkTerrainSeed(park)+4831, dx/95, 2, dz/95, 4)
	if cover > .16 {
		return parkWoodland
	}
	if cover < -.24 {
		return parkStone
	}
	return parkMeadow
}

// parkTerrain adds a physical surface in material batches, not a translucent
// overlay: its original path and planted instances remain individual geometry.
func parkTerrain(batch *cityBatch, park CityPark) {
	parkTerrainWithTrees(batch, park, nil, false)
}

// parkUrbanTerrain keeps the same triangulated relief and every tree and path,
// but finishes clear portions as city service surfaces around the new infill.
func parkUrbanTerrain(batch *cityBatch, park CityPark, trees []CityTree) {
	parkTerrainWithTrees(batch, park, trees, true)
}

// parkUrbanTerrainRole picks the ground role of a cell in an urban park. A
// cell clear of the grove and the trees, and farther than half a cell from a
// path, is service paving or stone paving.
func parkUrbanTerrainRole(park CityPark, paths [][]maths.Vec, buckets map[[2]int][]maths.Box, x, z, sx, sz float64) slot {
	centre := maths.V(x+sx/2, 0, z+sz/2)
	role := parkTerrainRole(park, centre.X, centre.Z)
	cell := maths.Box{Min: maths.V(x, 0, z), Max: maths.V(x+sx, 0, z+sz)}
	grove := maths.Box{Min: park.At.Add(maths.V(-4.2, 0, -3.2)), Max: park.At.Add(maths.V(4.2, 0, 3.2))}
	if !boxesOverlapXZ(cell, grove) && !parkInfillMeetsTrees(cell, buckets) && parkTerrainPathDistance(paths, centre.X, centre.Z) > math.Hypot(sx, sz)/2+.05 {
		role = parkService
		if noise.Fractal(parkTerrainSeed(park)+4851, centre.X/29, 0, centre.Z/29, 3) > .12 {
			role = parkPaving
		}
	}
	return role
}

// parkTerrainWithTrees adds the park ground to batch as a mesh with one cell
// per grid square. Urban parks take their roles from the urban rules.
func parkTerrainWithTrees(batch *cityBatch, park CityPark, trees []CityTree, urban bool) {
	nx, nz, sx, sz := parkTerrainGrid(park)
	x0, z0 := park.At.X-park.Width.InMetres()/2, park.At.Z-park.Depth.InMetres()/2
	paths := parkPaths(park)
	buckets := parkInfillTreeBuckets(trees)
	previous := make([]float64, nx+1)
	for i := 0; i <= nx; i++ {
		previous[i] = parkTerrainVertex(park, paths, x0+float64(i)*sx, z0)
	}
	for j := 0; j < nz; j++ {
		z := z0 + float64(j)*sz
		next := make([]float64, nx+1)
		for i := 0; i <= nx; i++ {
			next[i] = parkTerrainVertex(park, paths, x0+float64(i)*sx, z+sz)
		}
		rows := map[slot]surface{}
		for i := 0; i < nx; i++ {
			x := x0 + float64(i)*sx
			a := maths.V(x, previous[i], z)
			b := maths.V(x+sx, previous[i+1], z)
			c := maths.V(x, next[i], z+sz)
			d := maths.V(x+sx, next[i+1], z+sz)
			role := parkTerrainRole(park, x+sx/2, z+sz/2)
			if urban {
				role = parkUrbanTerrainRole(park, paths, buckets, x, z, sx, sz)
			}
			rows[role] = append(rows[role], face{a, c, b}, face{b, c, d})
		}
		for _, role := range []slot{parkMeadow, parkWoodland, parkStone, parkService, parkPaving} {
			if len(rows[role]) > 0 {
				batch.skin(role, rows[role])
			}
		}
		previous = next
	}
}
