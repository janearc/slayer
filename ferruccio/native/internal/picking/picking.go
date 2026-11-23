// Package picking selects native manufactured surfaces in an authored space scene.
package picking

import (
	"fmt"
	"math"

	"github.com/janearc/slayer/ferruccio/internal/draft"
	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/session/placement"
	"github.com/janearc/slayer/set"
)

// Scene is read-only selection geometry and its authored camera snapshot.
// Manufactured surfaces are selectable; native world surfaces only occlude them.
// Cache this snapshot for its preview source; construction belongs off the UI thread.
type Scene struct {
	hit           func(maths.Ray) string
	eye           maths.Subspace
	lens          film.Pinhole
	width, height int
	objects       map[string]draft.Object
	identities    map[string]string
}

// New prepares native geometry at the displayed image aspect without rendering.
// Selection supports explicit space placements, a fixed camera and film.wide.
func New(documentText string, aspect float64) (*Scene, error) {
	if !finite(aspect) || aspect <= 0 {
		return nil, fmt.Errorf("image aspect must be positive and finite")
	}
	document, err := draft.Open("selection.yaml", []byte(documentText))
	if err != nil {
		return nil, err
	}
	objects, err := document.Objects()
	if err != nil {
		return nil, err
	}
	for _, object := range objects {
		if object.Kind() == "import" {
			return nil, fmt.Errorf("selection supports native manufactured placements only")
		}
	}
	read, err := set.Read("selection.yaml", []byte(documentText))
	if err != nil {
		return nil, err
	}
	geometry, err := placement.SpaceGeometry(read)
	if err != nil {
		return nil, err
	}
	shots, err := read.Shots(func(string) (maths.Vec, bool) { return maths.Vec{}, false })
	if err != nil {
		return nil, err
	}
	if len(shots) == 0 {
		return nil, fmt.Errorf("selection needs a fixed authored camera")
	}
	settings, has := read.Film()
	width, height, _ := settings.Size()
	if !has || width <= 0 || height <= 0 {
		return nil, fmt.Errorf("selection needs an authored film size")
	}
	if _, rig := settings.Rig(); rig {
		return nil, fmt.Errorf("selection does not support lens rigs")
	}
	wide := settings.Wide() * maths.Degree
	if shots[0].Lens().Wide > 0 {
		wide = shots[0].Lens().Wide
	}
	lens, err := film.PinholeOf(wide)
	if err != nil {
		return nil, err
	}
	lens.Pixel = aspect * float64(height) / float64(width)
	if !finite(lens.Pixel) || lens.Pixel <= 0 {
		return nil, fmt.Errorf("image aspect cannot be represented")
	}
	scene := &Scene{hit: geometry.Hit, eye: shots[0].Eye(), lens: *lens, width: width, height: height,
		objects: map[string]draft.Object{}, identities: map[string]string{}}
	for _, object := range objects {
		scene.objects[object.ID()] = object
		scene.identities[object.Name()] = object.ID()
	}
	return scene, nil
}

// Pick returns a stable draft object identity, or empty when the image ray misses.
func (scene *Scene) Pick(x, y float64) string {
	ray, err := scene.ray(x, y)
	if err != nil || scene.hit == nil {
		return ""
	}
	return scene.identities[scene.hit(ray)]
}

// Drag returns metre east/north deltas along the selected object's horizontal plane.
func (scene *Scene) Drag(id string, x0, y0, x1, y1 float64) (float64, float64, error) {
	if scene == nil {
		return 0, 0, fmt.Errorf("selection scene is unavailable")
	}
	object, found := scene.objects[id]
	if !found {
		return 0, 0, fmt.Errorf("unknown scene object")
	}
	first, err := scene.planePoint(x0, y0, object.Position()[2])
	if err != nil {
		return 0, 0, err
	}
	last, err := scene.planePoint(x1, y1, object.Position()[2])
	if err != nil {
		return 0, 0, err
	}
	delta := last.Sub(first)
	if !finite(delta.X) || !finite(delta.Z) {
		return 0, 0, fmt.Errorf("drag produced a non-finite displacement")
	}
	return delta.X, -delta.Z, nil
}

// ray delegates image geometry and camera transforms to the native sensor.
func (scene *Scene) ray(x, y float64) (maths.Ray, error) {
	if scene == nil || scene.width <= 0 || scene.height <= 0 {
		return maths.Ray{}, fmt.Errorf("selection scene is unavailable")
	}
	if !finite(x) || !finite(y) || x < 0 || x > 1 || y < 0 || y > 1 {
		return maths.Ray{}, fmt.Errorf("image coordinates must be finite normalized values")
	}
	column, row := int(x*float64(scene.width)), int(y*float64(scene.height))
	if column == scene.width {
		column--
	}
	if row == scene.height {
		row--
	}
	return film.Look(scene.eye, scene.lens, column, row, scene.width, scene.height, maths.Band{}), nil
}

// planePoint uses the native horizontal plane intersection at object height.
func (scene *Scene) planePoint(x, y, height float64) (maths.Vec, error) {
	ray, err := scene.ray(x, y)
	if err != nil {
		return maths.Vec{}, err
	}
	plane := maths.Subspace{Origin: maths.V(0, height, 0)}
	distance, hit := (shape.Plane{}).Hit(plane.RayIn(ray), 0, math.Inf(1))
	if !hit || !finite(distance) {
		return maths.Vec{}, fmt.Errorf("drag ray is parallel to or behind the object plane")
	}
	return ray.Point(distance), nil
}

// finite validates interactive coordinates before entering native geometry.
func finite(value float64) bool { return !math.IsNaN(value) && !math.IsInf(value, 0) }
