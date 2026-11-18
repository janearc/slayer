package searching

import (
	"math"
	"sort"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
)

// summitSurface indexes the actual retained triangles, not another height
// approximation. Roads are clipped to these planes and lifted two centimetres;
// their interiors therefore cannot bridge over or cut through the ground.
type summitSurface struct {
	faces []summitFacet
	bins  map[[2]int][]int
	roads []summitRoadQuad
}
type summitFacet struct{ a, b, c maths.Vec }
type summitRoadQuad struct {
	polygon []maths.Vec
	a, b    maths.Vec
	half    float64
}

const summitCell = 16.0
const summitRoadLift = .02

// newSummitSurface indexes the retained faces that fall inside the summit area.
func newSummitSurface(surfaces []*mesh.Mesh) *summitSurface {
	s := &summitSurface{bins: map[[2]int][]int{}}
	for _, surface := range surfaces {
		for i := range surface.FaceCount() {
			a, b, c := surface.Corners(i)
			lowX, highX := math.Min(a.X, math.Min(b.X, c.X)), math.Max(a.X, math.Max(b.X, c.X))
			lowZ, highZ := math.Min(a.Z, math.Min(b.Z, c.Z)), math.Max(a.Z, math.Max(b.Z, c.Z))
			if lowX > 850 || highX < -850 || lowZ > 850 || highZ < -850 {
				continue
			}
			id := len(s.faces)
			s.faces = append(s.faces, summitFacet{a, b, c})
			for x := int(math.Floor(math.Max(-850, lowX) / summitCell)); x <= int(math.Floor(math.Min(850, highX)/summitCell)); x++ {
				for z := int(math.Floor(math.Max(-850, lowZ) / summitCell)); z <= int(math.Floor(math.Min(850, highZ)/summitCell)); z++ {
					key := [2]int{x, z}
					s.bins[key] = append(s.bins[key], id)
				}
			}
		}
	}
	return s
}

// crossXZ returns the cross product of a and b in the x, z plane.
func crossXZ(a, b maths.Vec) float64 { return a.X*b.Z - a.Z*b.X }

// height returns the facet height above x, z and whether x, z lies on it.
func (f summitFacet) height(x, z float64) (float64, bool) {
	p := maths.V(x-f.a.X, 0, z-f.a.Z)
	b, c := f.b.Sub(f.a), f.c.Sub(f.a)
	d := crossXZ(b, c)
	if math.Abs(d) < 1e-16 {
		return 0, false
	}
	u, v := crossXZ(p, c)/d, crossXZ(b, p)/d
	return f.a.Y + u*b.Y + v*c.Y, u >= -1e-8 && v >= -1e-8 && u+v <= 1+1e-8
}

// height returns the highest retained surface above x, z.
func (s *summitSurface) height(x, z float64) (float64, bool) {
	value, found := math.Inf(-1), false
	for _, id := range s.bins[[2]int{int(math.Floor(x / summitCell)), int(math.Floor(z / summitCell))}] {
		y, has := s.faces[id].height(x, z)
		if has && (!found || y > value) {
			value, found = y, true
		}
	}
	return value, found
}

// smoothSummitPath uses one tangent at each shared knot. Sampling a smooth
// centreline replaces the old sharp, separately oriented road rectangles.
func smoothSummitPath(path []maths.Vec) []maths.Vec {
	if len(path) < 2 {
		return nil
	}
	tangent := make([]maths.Vec, len(path))
	for i := range path {
		before, after := path[max(0, i-1)], path[min(len(path)-1, i+1)]
		t := after.Sub(before).Scale(.5)
		if i == 0 || i == len(path)-1 {
			t = after.Sub(before)
		}
		limit := math.Inf(1)
		if i > 0 {
			limit = math.Min(limit, path[i].Sub(before).Len())
		}
		if i < len(path)-1 {
			limit = math.Min(limit, after.Sub(path[i]).Len())
		}
		if t.Len() > .8*limit {
			t = t.Scale(.8 * limit / t.Len())
		}
		tangent[i] = t
	}
	var points []maths.Vec
	for i := 0; i < len(path)-1; i++ {
		count := max(2, int(math.Ceil(path[i+1].Sub(path[i]).Len()/1.5)))
		for j := range count {
			u := float64(j) / float64(count)
			u2, u3 := u*u, u*u*u
			point := path[i].Scale(2*u3 - 3*u2 + 1).Add(tangent[i].Scale(u3 - 2*u2 + u)).Add(path[i+1].Scale(-2*u3 + 3*u2)).Add(tangent[i+1].Scale(u3 - u2))
			point.Y = 0
			points = append(points, point)
		}
	}
	return append(points, path[len(path)-1])
}

// ribbon adds a road of the given width along the smoothed path.
func (s *summitSurface) ribbon(path []maths.Vec, width float64) {
	points := smoothSummitPath(path)
	if len(points) < 2 {
		return
	}
	left, right := make([]maths.Vec, len(points)), make([]maths.Vec, len(points))
	for i, p := range points {
		t := points[min(i+1, len(points)-1)].Sub(points[max(0, i-1)])
		side := maths.V(-t.Z, 0, t.X).Unit().Scale(width / 2)
		left[i], right[i] = p.Add(side), p.Sub(side)
	}
	for i := 0; i < len(points)-1; i++ {
		polygon := roadClean([]maths.Vec{left[i], right[i], right[i+1], left[i+1]})
		if roadArea(polygon) < 0 {
			reverseRoad(polygon)
		}
		s.roads = append(s.roads, summitRoadQuad{polygon, points[i], points[i+1], width / 2})
	}
}

// nearRoad reports whether p lies within a road's half-width plus margin.
func (s *summitSurface) nearRoad(p maths.Vec, margin float64) bool {
	for _, road := range s.roads {
		d := road.b.Sub(road.a)
		t := math.Max(0, math.Min(1, p.Sub(road.a).Dot(d)/d.Dot(d)))
		if p.Sub(road.a.Add(d.Scale(t))).Len() < road.half+margin {
			return true
		}
	}
	return false
}

// roadArea returns the signed area of a polygon in the x, z plane.
func roadArea(p []maths.Vec) float64 {
	sum := 0.0
	for i := range p {
		sum += crossXZ(p[i], p[(i+1)%len(p)])
	}
	return sum / 2
}

// reverseRoad reverses the vertex order of a polygon in place.
func reverseRoad(p []maths.Vec) {
	for i, j := 0, len(p)-1; i < j; i, j = i+1, j-1 {
		p[i], p[j] = p[j], p[i]
	}
}

// roadClean drops repeated and collinear vertices, and returns nil when the
// polygon has no area left.
func roadClean(p []maths.Vec) []maths.Vec {
	var result []maths.Vec
	for _, v := range p {
		if len(result) == 0 || v.Sub(result[len(result)-1]).Len() > 1e-7 {
			result = append(result, v)
		}
	}
	if len(result) > 1 && result[0].Sub(result[len(result)-1]).Len() < 1e-7 {
		result = result[:len(result)-1]
	}
	for len(result) > 2 {
		changed := false
		for i := range result {
			a, b, c := result[(i+len(result)-1)%len(result)], result[i], result[(i+1)%len(result)]
			if math.Abs(crossXZ(b.Sub(a), c.Sub(b))) < 1e-9 {
				result = append(result[:i], result[i+1:]...)
				changed = true
				break
			}
		}
		if !changed {
			break
		}
	}
	if len(result) < 3 || math.Abs(roadAreaRaw(result)) < 1e-9 {
		return nil
	}
	return result
}

// roadAreaRaw returns the signed area of a polygon without cleaning it.
func roadAreaRaw(p []maths.Vec) float64 {
	sum := 0.0
	for i := range p {
		sum += crossXZ(p[i], p[(i+1)%len(p)])
	}
	return sum / 2
}

// roadHalf clips a convex polygon to either side of one directed edge.
func roadHalf(p []maths.Vec, a, b maths.Vec, inside bool) []maths.Vec {
	if len(p) == 0 {
		return nil
	}
	var result []maths.Vec
	sign := 1.0
	if !inside {
		sign = -1
	}
	edge := b.Sub(a)
	previous := p[len(p)-1]
	before := sign * crossXZ(edge, previous.Sub(a))
	for _, point := range p {
		after := sign * crossXZ(edge, point.Sub(a))
		if (before >= 0) != (after >= 0) {
			t := before / (before - after)
			result = append(result, previous.Add(point.Sub(previous).Scale(t)))
		}
		if after >= 0 {
			result = append(result, point)
		}
		previous, before = point, after
	}
	return roadClean(result)
}

// roadClip returns the part of p inside the convex polygon clip.
func roadClip(p, clip []maths.Vec) []maths.Vec {
	result := append([]maths.Vec(nil), p...)
	for i := range clip {
		result = roadHalf(result, clip[i], clip[(i+1)%len(clip)], true)
		if len(result) == 0 {
			break
		}
	}
	return result
}

// roadSubtract keeps disjoint pieces of p outside a convex already covered
// piece. Junctions are a union, not overlapping coplanar asphalt surfaces.
func roadSubtract(p, covered []maths.Vec) [][]maths.Vec {
	var outside [][]maths.Vec
	for i := range covered {
		a, b := covered[i], covered[(i+1)%len(covered)]
		if part := roadHalf(p, a, b, false); len(part) > 0 {
			outside = append(outside, part)
		}
		p = roadHalf(p, a, b, true)
		if len(p) == 0 {
			break
		}
	}
	return outside
}

// roadMesh builds the road surface from the ground triangles it clips.
func (s *summitSurface) roadMesh() (*shape.Mesh, error) {
	hits := map[int][]int{}
	for q, road := range s.roads {
		lowX, lowZ, highX, highZ := math.Inf(1), math.Inf(1), math.Inf(-1), math.Inf(-1)
		for _, p := range road.polygon {
			lowX = math.Min(lowX, p.X)
			lowZ = math.Min(lowZ, p.Z)
			highX = math.Max(highX, p.X)
			highZ = math.Max(highZ, p.Z)
		}
		seen := map[int]bool{}
		for x := int(math.Floor(lowX / summitCell)); x <= int(math.Floor(highX/summitCell)); x++ {
			for z := int(math.Floor(lowZ / summitCell)); z <= int(math.Floor(highZ/summitCell)); z++ {
				for _, id := range s.bins[[2]int{x, z}] {
					if !seen[id] {
						seen[id] = true
						hits[id] = append(hits[id], q)
					}
				}
			}
		}
	}
	ids := make([]int, 0, len(hits))
	for id := range hits {
		ids = append(ids, id)
	}
	sort.Ints(ids)
	var vertices []maths.Vec
	var faces [][3]int
	for _, id := range ids {
		f := s.faces[id]
		triangle := []maths.Vec{f.a, f.b, f.c}
		if roadArea(triangle) < 0 {
			reverseRoad(triangle)
		}
		var covered [][]maths.Vec
		for _, q := range hits[id] {
			p := roadClip(triangle, s.roads[q].polygon)
			if len(p) == 0 {
				continue
			}
			pieces := [][]maths.Vec{p}
			for _, existing := range covered {
				var next [][]maths.Vec
				for _, piece := range pieces {
					next = append(next, roadSubtract(piece, existing)...)
				}
				pieces = next
				if len(pieces) == 0 {
					break
				}
			}
			covered = append(covered, pieces...)
		}
		for _, p := range covered {
			first := len(vertices)
			for _, v := range p {
				v.X = math.Round(v.X*1e7) / 1e7
				v.Z = math.Round(v.Z*1e7) / 1e7
				v.Y, _ = f.height(v.X, v.Z)
				v.Y = math.Round(v.Y*1e7)/1e7 + summitRoadLift
				vertices = append(vertices, v)
			}
			for i := 1; i < len(p)-1; i++ {
				if math.Abs(crossXZ(vertices[first+i].Sub(vertices[first]), vertices[first+i+1].Sub(vertices[first]))) > 1e-9 {
					faces = append(faces, [3]int{first, first + i + 1, first + i})
				}
			}
		}
	}
	return shape.NewMesh(vertices, faces, nil, false)
}

// fitSummitRoads adds the summit roads, their markers and their reflectors.
func (s *heardStage) fitSummitRoads() error {
	for i, path := range maunaKeaRoads {
		s.summitGround.ribbon(path, []float64{8, 7, 8, 8, 8}[i])
	}
	s.summitGround.ribbon([]maths.Vec{maths.V(-95, 0, 12), maths.V(-75, 0, 21), maths.V(-55, 0, 33), maths.V(-40, 0, 42)}, 6)
	road, err := s.summitGround.roadMesh()
	if err != nil {
		return err
	}
	s.world.Add(thing.Thing{Shape: road, Material: &thing.Material{Name: "smooth access roads fitted to retained Maunakea terrain", Colour: maths.RGB(.12, .115, .11)}})
	var posts, reflectors siteBlocks
	for i, strip := range s.summitGround.roads {
		if i%14 != 0 {
			continue
		}
		tangent := strip.b.Sub(strip.a).Unit()
		side := maths.V(-tangent.Z, 0, tangent.X)
		for _, sign := range []float64{-1, 1} {
			p := strip.a.Add(side.Scale(sign * (strip.half + .45)))
			y, has := s.summitGround.height(p.X, p.Z)
			if !has {
				continue
			}
			posts.box(maths.V(p.X-.045, y, p.Z-.045), maths.V(.09, .7, .09))
			reflectors.box(maths.V(p.X-.06, y+.53, p.Z-.06), maths.V(.12, .12, .12))
		}
	}
	if err := posts.place(s, &thing.Material{Name: "terrain-mounted summit road markers", Colour: maths.RGB(.45, .44, .4)}); err != nil {
		return err
	}
	if err := reflectors.place(s, &thing.Material{Name: "summit roadside reflectors", Colour: maths.RGB(.76, .71, .6)}); err != nil {
		return err
	}
	return nil
}
