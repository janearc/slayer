package placement

import (
	"math"

	"github.com/janearc/slayer/maths"
)

// batchHit intersects the displayed triangles and preserves the staged
// authored group. Unknown and non-editable groups remain occluding world.
func (scene *DisplayScene) batchHit(ray maths.Ray, maximum float64) (DisplayHit, bool) {
	length := ray.Dir.Len()
	if length == 0 || math.IsInf(length, 0) || math.IsNaN(length) {
		return DisplayHit{}, false
	}
	ray.Dir = ray.Dir.Scale(1 / length)
	nearest := maximum
	var found DisplayHit
	bounds := make(map[*DisplayMesh]maths.Box)
	for _, batch := range scene.Batches {
		if batch.Mesh == nil {
			continue
		}
		if batch.HasBounds {
			if _, _, hit := batch.Bounds.Hit(ray, 0, nearest); !hit {
				continue
			}
		}
		box, known := bounds[batch.Mesh]
		if !known {
			low, high, ok := batch.Mesh.Bounds()
			if !ok {
				continue
			}
			box = maths.Box{Min: low, Max: high}
			bounds[batch.Mesh] = box
		}
		for index := 0; index < len(batch.Places); {
			if batch.HasBounds && len(batch.pickBlocks) > 0 && index%PlacementBlockSize == 0 {
				_, _, blockHit := batch.pickBlocks[index/PlacementBlockSize].Hit(ray, 0, nearest)
				if !blockHit {
					index += PlacementBlockSize
					continue
				}
			}
			place := batch.Places[index]
			index++
			local := place.RayIn(ray)
			if _, _, hit := box.Hit(local, 0, math.Inf(1)); !hit {
				continue
			}
			for i := 0; i+2 < len(batch.Mesh.Indices); i += 3 {
				a, b, c := batch.Mesh.Indices[i], batch.Mesh.Indices[i+1], batch.Mesh.Indices[i+2]
				if uint64(a) >= uint64(len(batch.Mesh.Vertices)) || uint64(b) >= uint64(len(batch.Mesh.Vertices)) || uint64(c) >= uint64(len(batch.Mesh.Vertices)) {
					continue
				}
				distance, hit := triangleHit(local, batch.Mesh.Vertices[a].Position, batch.Mesh.Vertices[b].Position, batch.Mesh.Vertices[c].Position)
				if !hit {
					continue
				}
				point := place.Out(local.Point(distance))
				worldDistance := point.Sub(ray.At).Len()
				if worldDistance >= nearest {
					continue
				}
				nearest = worldDistance
				found = DisplayHit{ID: "scene.world", Name: "world", Point: point, Distance: worldDistance}
				if batch.Authored && (batch.Author.Collection == "scene.set.things" || batch.Author.Collection == "scene.set.stands") && batch.Mesh.Notice == "" {
					found.ID, found.Name = batch.Author.Collection+"/"+batch.Author.Name, batch.Author.Name
				}
			}
		}
	}
	return found, found.ID != ""
}

// triangleHit intersects both faces of one displayed triangle.
func triangleHit(ray maths.Ray, a, b, c maths.Vec) (float64, bool) {
	e1, e2 := b.Sub(a), c.Sub(a)
	p := ray.Dir.Cross(e2)
	det := e1.Dot(p)
	threshold := 1e-24 * e1.Dot(e1) * e2.Dot(e2) * ray.Dir.Dot(ray.Dir)
	if det*det <= threshold || math.IsNaN(det) || math.IsInf(det, 0) {
		return 0, false
	}
	inv := 1 / det
	delta := ray.At.Sub(a)
	u := delta.Dot(p) * inv
	if u < 0 || u > 1 {
		return 0, false
	}
	q := delta.Cross(e1)
	v := ray.Dir.Dot(q) * inv
	if v < 0 || u+v > 1 {
		return 0, false
	}
	distance := e2.Dot(q) * inv
	return distance, distance >= 0 && !math.IsInf(distance, 0) && !math.IsNaN(distance)
}
