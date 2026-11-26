package placement

import (
	"math"

	"github.com/janearc/slayer/maths"
)

// displayNear is how close to the eye, in metres, a thing still counts as in
// front of it when its bounds are projected. It is the near plane the working
// view draws with.
const displayNear = .05

// defaultWide is the field of view, in radians, a lens with none says.
const defaultWide = 60 * maths.Degree

// SetPoint is point as a set writes it: east, north and up, in metres. The
// display keeps east in x, up in y and south in z.
func SetPoint(point maths.Vec) [3]float64 {
	return [3]float64{point.X, -point.Z, point.Y}
}

// DisplayHit is what an eye ray through the picture first meets.
type DisplayHit struct {
	// ID and Name are the object's identity and its name in the set. The
	// world's blockers are "scene.world" and "world".
	ID, Name string
	// Point is where the ray meets it, in the display's frame; SetPoint
	// gives it as east, north and up.
	Point maths.Vec
	// Distance is how far that is from the eye, in metres.
	Distance float64
	// Part is the name of the material of the part that was hit. It is empty
	// when the part has none, and the display carries no other part name.
	Part string
}

// PickAt is what Pick finds, with where. x and y are fractions of the image
// width and height. It reports false when the ray misses everything or the
// coordinates are not in the image. It names the world's blockers, which Pick
// leaves out.
func (scene *DisplayScene) PickAt(x, y float64) (DisplayHit, bool) {
	if !displayFinite(x) || !displayFinite(y) || x < 0 || x > 1 || y < 0 || y > 1 {
		return DisplayHit{}, false
	}
	ray, err := scene.displayRay(x, y)
	if err != nil {
		return DisplayHit{}, false
	}
	nearest, found := math.Inf(1), DisplayHit{}
	for _, object := range scene.Objects {
		if object.Mesh == nil || object.Mesh.hitter == nil {
			continue
		}
		local := object.At.RayIn(ray)
		distance, hit := object.Mesh.hitter.Hit(local, 0, math.Inf(1))
		if !hit {
			continue
		}
		point := object.At.Out(local.Point(distance))
		if worldDistance := point.Sub(ray.At).Len(); worldDistance < nearest {
			nearest = worldDistance
			found = DisplayHit{ID: object.ID, Name: object.Name, Point: point, Distance: worldDistance, Part: object.Mesh.partAt(local)}
		}
	}
	if hit, ok := scene.batchHit(ray, nearest); ok {
		found = hit
	}
	return found, found.ID != ""
}

// partAt is the material name of the part a ray in the mesh's own frame hits
// first, or empty when it hits none or the mesh holds no part names.
func (mesh *DisplayMesh) partAt(ray maths.Ray) string {
	parts, ok := mesh.hitter.(displayComponents)
	if !ok {
		return ""
	}
	_, material, _ := parts.hitPart(ray, 0, math.Inf(1))
	return material
}

// Bounds is the box, in the mesh's own frame, that holds every vertex. It
// reports false for a mesh with none.
func (mesh *DisplayMesh) Bounds() (low, high maths.Vec, ok bool) {
	if mesh == nil || len(mesh.Vertices) == 0 {
		return maths.Vec{}, maths.Vec{}, false
	}
	low, high = mesh.Vertices[0].Position, mesh.Vertices[0].Position
	for _, vertex := range mesh.Vertices {
		at := vertex.Position
		low = maths.V(math.Min(low.X, at.X), math.Min(low.Y, at.Y), math.Min(low.Z, at.Z))
		high = maths.V(math.Max(high.X, at.X), math.Max(high.Y, at.Y), math.Max(high.Z, at.Z))
	}
	return low, high, true
}

// ObjectView is how one object stands to the camera. Both numbers come from
// the object's bounding box, not from its surface: Distance to the nearest
// point of the box and Share the part of the picture, from zero to one, that
// the box's outline on the picture covers. A thing hidden behind another
// still has its share.
type ObjectView struct {
	ID, Name string
	Distance float64
	Share    float64
}

// InView lists the objects whose bounds fall on the picture, nearest first.
// It measures with the camera and lens as they now are.
func (scene *DisplayScene) InView() []ObjectView {
	if scene == nil || scene.Width <= 0 || scene.Height <= 0 {
		return nil
	}
	var seen []ObjectView
	for _, object := range scene.Objects {
		low, high, ok := object.Mesh.Bounds()
		if !ok {
			continue
		}
		share := scene.boxShare(object.At, low, high)
		if share <= 0 {
			continue
		}
		seen = append(seen, ObjectView{ID: object.ID, Name: object.Name, Share: share, Distance: scene.boxDistance(object.At, low, high)})
	}
	sortViews(seen)
	return seen
}

// sortViews puts the nearest first, and the larger share first for equal
// distances, with a simple insertion sort that keeps equal names in order.
func sortViews(seen []ObjectView) {
	for index := 1; index < len(seen); index++ {
		for back := index; back > 0 && viewBefore(seen[back], seen[back-1]); back-- {
			seen[back], seen[back-1] = seen[back-1], seen[back]
		}
	}
}

// viewBefore reports whether a stands before b: nearer, or as near and larger.
func viewBefore(a, b ObjectView) bool {
	if a.Distance != b.Distance {
		return a.Distance < b.Distance
	}
	return a.Share > b.Share
}

// boxDistance is how far the eye is from the nearest point of the box that
// low and high bound in the frame at.
func (scene *DisplayScene) boxDistance(at maths.Subspace, low, high maths.Vec) float64 {
	eye := at.In(scene.Eye.Origin)
	nearest := maths.V(math.Max(low.X, math.Min(eye.X, high.X)), math.Max(low.Y, math.Min(eye.Y, high.Y)), math.Max(low.Z, math.Min(eye.Z, high.Z)))
	return at.Out(nearest).Sub(scene.Eye.Origin).Len()
}

// boxShare is the part of the picture, from zero to one, covered by the
// outline of the box that low and high bound in the frame at. Edges are cut
// at the near plane first, so a box the camera stands in or beside still has
// an outline. It is zero when no part of the box is in front of the camera.
func (scene *DisplayScene) boxShare(at maths.Subspace, low, high maths.Vec) float64 {
	corners := scene.eyeCorners(at, low, high)
	left, top, right, bottom := math.Inf(1), math.Inf(1), math.Inf(-1), math.Inf(-1)
	for index := range corners {
		for bit := 1; bit < 8; bit <<= 1 {
			if index&bit != 0 {
				continue
			}
			first, second, in := clipNear(corners[index], corners[index|bit])
			if !in {
				continue
			}
			for _, end := range []maths.Vec{first, second} {
				x, y := scene.project(end)
				left, right = math.Min(left, x), math.Max(right, x)
				top, bottom = math.Min(top, y), math.Max(bottom, y)
			}
		}
	}
	width := math.Min(right, 1) - math.Max(left, 0)
	height := math.Min(bottom, 1) - math.Max(top, 0)
	if !(width > 0 && height > 0) {
		return 0
	}
	return width * height
}

// eyeCorners are the eight corners of the box that low and high bound in the
// frame at, in the eye's frame. Bit 0 of a corner's index picks x, bit 1 y
// and bit 2 z, low when clear and high when set.
func (scene *DisplayScene) eyeCorners(at maths.Subspace, low, high maths.Vec) [8]maths.Vec {
	var corners [8]maths.Vec
	for index := range corners {
		corner := low
		if index&1 != 0 {
			corner.X = high.X
		}
		if index&2 != 0 {
			corner.Y = high.Y
		}
		if index&4 != 0 {
			corner.Z = high.Z
		}
		corners[index] = scene.Eye.In(at.Out(corner))
	}
	return corners
}

// clipNear cuts the segment from a to b, in the eye's frame, to the part in
// front of the near plane, where the eye looks along minus z. It reports
// false when none of it is.
func clipNear(a, b maths.Vec) (maths.Vec, maths.Vec, bool) {
	depthA, depthB := -a.Z-displayNear, -b.Z-displayNear
	switch {
	case depthA < 0 && depthB < 0:
		return a, b, false
	case depthA < 0:
		a = a.Add(b.Sub(a).Scale(depthA / (depthA - depthB)))
	case depthB < 0:
		b = a.Add(b.Sub(a).Scale(depthA / (depthA - depthB)))
	}
	return a, b, true
}

// project is where a point in the eye's frame, in front of the camera, falls
// on the picture: x from the left and y from the top, as fractions. It is
// the inverse of the eye ray for a pixel.
func (scene *DisplayScene) project(point maths.Vec) (float64, float64) {
	wide := scene.Lens.Wide
	if !(wide > 0 && wide < math.Pi) {
		wide = defaultWide
	}
	pixel := scene.Lens.Pixel
	if !(pixel > 0) {
		pixel = 1
	}
	half := math.Tan(wide / 2)
	aspect := float64(scene.Width) * pixel / float64(scene.Height)
	across, above := point.X/-point.Z, point.Y/-point.Z
	return (across/half + 1) / 2, (1 - above*aspect/half) / 2
}
