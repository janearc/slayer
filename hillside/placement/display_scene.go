package placement

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"

	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/manufacture"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
	"github.com/janearc/slayer/set"
	"github.com/janearc/slayer/stage"
)

// Scene updates placements and camera without rebuilding already cached meshes.
func (cache *DisplayCache) Scene(read set.Set) (*DisplayScene, error) {
	if cache == nil {
		return nil, fmt.Errorf("working display cache is unavailable")
	}
	if !read.InSpace() || !read.HasSeed() || len(read.Unrendered()) != 0 {
		return nil, fmt.Errorf("working display needs supported native space placements and an explicit seed")
	}
	settings, has := read.Film()
	width, height, _ := settings.Size()
	if !has || width <= 0 || height <= 0 {
		return nil, fmt.Errorf("working display needs an authored film size")
	}
	if _, rig := settings.Rig(); rig {
		return nil, fmt.Errorf("working display does not support lens rigs")
	}
	places := map[string]maths.Vec{}
	for _, each := range read.Things() {
		places[each.Name()] = SpaceAt(each).Origin
	}
	shots, err := read.Shots(func(name string) (maths.Vec, bool) { at, found := places[name]; return at, found })
	if err != nil {
		return nil, err
	}
	if len(shots) == 0 {
		return nil, fmt.Errorf("working display needs a fixed camera")
	}
	wide := settings.Wide() * maths.Degree
	if shots[0].Lens().Wide > 0 {
		wide = shots[0].Lens().Wide
	}
	lens, err := film.PinholeOf(wide)
	if err != nil {
		return nil, err
	}
	scene := &DisplayScene{Eye: shots[0].Eye(), Focus: shots[0].Focus(), Lens: *lens, Width: width, Height: height}
	cache.mu.Lock()
	defer cache.mu.Unlock()
	if cache.meshes == nil {
		cache.meshes = map[displayKey]*DisplayMesh{}
	}
	if cache.worlds == nil {
		cache.worlds = map[string]*DisplayMesh{}
	}
	for _, each := range read.Things() {
		if each.Recipe() != "" || each.Alight() {
			return nil, fmt.Errorf("working display does not support recipes or burning placements")
		}
		key := displayKey{kind: each.Kind(), detail: each.Complexity(), sceneSeed: read.Seed(), objectSeed: each.Seed()}
		skin := cache.meshes[key]
		if skin == nil {
			skin, err = displayManufactured(read.Seed(), each)
			if err != nil {
				return nil, fmt.Errorf("%s: %w", each.Name(), err)
			}
			cache.meshes[key] = skin
		}
		scene.Objects = append(scene.Objects, DisplayObject{ID: "scene.set.things/" + each.Name(), Name: each.Name(), Mesh: skin, At: SpaceAt(each)})
	}
	if said, exists := read.World(); exists {
		world := thing.NewWorld()
		if err := stage.Add(world, said, maths.Vec{}); err != nil {
			return nil, err
		}
		if len(world.Things) > 0 {
			builder := newDisplayBuilder()
			for _, part := range world.Things {
				if err := builder.thing(*part); err != nil {
					return nil, err
				}
			}
			skin, err := builder.finish()
			if err != nil {
				return nil, err
			}
			raw, err := json.Marshal(skin)
			if err != nil {
				return nil, err
			}
			digest := sha256.Sum256(raw)
			key := hex.EncodeToString(digest[:])
			if kept := cache.worlds[key]; kept != nil {
				skin = kept
			} else {
				cache.worlds[key] = skin
			}
			scene.Objects = append(scene.Objects, DisplayObject{ID: "scene.world", Name: "world", Mesh: skin, At: maths.Local()})
		}
	}
	return scene, nil
}

// displayManufactured uses a native coarse representation, never dropped faces.
func displayManufactured(sceneSeed uint64, each set.Thing) (*DisplayMesh, error) {
	row, err := manufacture.KindNamed(each.Kind())
	if err != nil {
		return nil, err
	}
	request := Construction{Kind: row, SceneSeed: sceneSeed, ObjectSeed: int(each.Seed()), Complexity: "hull"}
	skin, coarseError := displayConstruction(request)
	if coarseError == nil {
		return skin, nil
	}
	if each.Complexity() == "hull" {
		return nil, coarseError
	}
	request.Complexity = each.Complexity()
	skin, authoredError := displayConstruction(request)
	if authoredError != nil {
		return nil, fmt.Errorf("coarse mesh: %v; authored %s mesh: %w", coarseError, request.Complexity, authoredError)
	}
	skin.Notice = "Authored detail · coarse mesh unavailable"
	skin.Diagnostic = coarseError.Error()
	return skin, nil
}

// displayConstruction retains native validation at every representation boundary.
func displayConstruction(request Construction) (*DisplayMesh, error) {
	specimen, err := Make(request)
	if err != nil {
		return nil, err
	}
	parts, err := specimen.Things(maths.Local())
	if err != nil {
		return nil, err
	}
	builder := newDisplayBuilder()
	for _, part := range parts {
		if err := builder.thing(part); err != nil {
			return nil, err
		}
	}
	return builder.finish()
}

// Pick intersects current working placements, including unselectable world blockers.
func (scene *DisplayScene) Pick(x, y float64) string {
	if !displayFinite(x) || !displayFinite(y) || x < 0 || x > 1 || y < 0 || y > 1 {
		return ""
	}
	ray, err := scene.displayRay(x, y)
	if err != nil {
		return ""
	}
	nearest, id := math.Inf(1), ""
	for _, object := range scene.Objects {
		if object.Mesh == nil || object.Mesh.hitter == nil {
			continue
		}
		local := object.At.RayIn(ray)
		distance, hit := object.Mesh.hitter.Hit(local, 0, math.Inf(1))
		if !hit {
			continue
		}
		worldDistance := object.At.Out(local.Point(distance)).Sub(ray.At).Len()
		if worldDistance < nearest {
			nearest, id = worldDistance, object.ID
			if object.ID == "scene.world" {
				id = ""
			}
		}
	}
	return id
}

// Drag intersects the current object's horizontal plane; captured points may leave the image.
func (scene *DisplayScene) Drag(id string, x0, y0, x1, y1 float64) (float64, float64, error) {
	if scene == nil {
		return 0, 0, fmt.Errorf("working scene is unavailable")
	}
	for _, object := range scene.Objects {
		if object.ID != id || id == "scene.world" {
			continue
		}
		first, err := scene.displayPlane(x0, y0, object.At.Origin.Y)
		if err != nil {
			return 0, 0, err
		}
		last, err := scene.displayPlane(x1, y1, object.At.Origin.Y)
		if err != nil {
			return 0, 0, err
		}
		delta := last.Sub(first)
		if !displayFinite(delta.X) || !displayFinite(delta.Z) {
			return 0, 0, fmt.Errorf("working drag produced an invalid displacement")
		}
		return delta.X, -delta.Z, nil
	}
	return 0, 0, fmt.Errorf("working scene has no selected object")
}

// displayRay returns the eye ray through the pixel at image coordinates x and
// y, which are fractions of the image width and height. It fails for a nil
// scene, an empty image or a coordinate that is not finite.
func (scene *DisplayScene) displayRay(x, y float64) (maths.Ray, error) {
	if scene == nil || scene.Width <= 0 || scene.Height <= 0 || !displayFinite(x) || !displayFinite(y) {
		return maths.Ray{}, fmt.Errorf("working image coordinates are invalid")
	}
	// Native film.Ray works at integer pixel centres, including captured off-image pixels.
	px, py := math.Floor(x*float64(scene.Width)), math.Floor(y*float64(scene.Height))
	if math.Abs(px) > float64(math.MaxInt32) || math.Abs(py) > float64(math.MaxInt32) {
		return maths.Ray{}, fmt.Errorf("captured working coordinates exceed native raster bounds")
	}
	return film.Look(scene.Eye, scene.Lens, int(px), int(py), scene.Width, scene.Height, maths.Band{}), nil
}

// displayPlane returns the point where the eye ray through image coordinates
// x and y meets the horizontal plane at height. It fails when the ray is
// parallel to the plane or meets it behind the eye.
func (scene *DisplayScene) displayPlane(x, y, height float64) (maths.Vec, error) {
	ray, err := scene.displayRay(x, y)
	if err != nil {
		return maths.Vec{}, err
	}
	frame := maths.Subspace{Origin: maths.V(0, height, 0)}
	distance, hit := (shape.Plane{}).Hit(frame.RayIn(ray), 0, math.Inf(1))
	if !hit || !displayFinite(distance) {
		return maths.Vec{}, fmt.Errorf("working drag is parallel to or behind the object plane")
	}
	return ray.Point(distance), nil
}

// displayFinite reports whether value is neither NaN nor infinite.
func displayFinite(value float64) bool { return !math.IsNaN(value) && !math.IsInf(value, 0) }
