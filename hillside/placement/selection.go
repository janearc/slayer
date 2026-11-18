package placement

import (
	"fmt"
	"math"

	"github.com/janearc/slayer/manufacture"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
	"github.com/janearc/slayer/set"
	"github.com/janearc/slayer/stage"
)

type selectionSurface struct {
	name  string
	shape shape.Hitter
	at    maths.Subspace
}

type spaceGeometry struct{ surfaces []selectionSurface }

// SpaceGeometry constructs read-only native manufactured surfaces for selection.
// Explicit scene seeds keep CLI defaults outside this source-only contract.
func SpaceGeometry(read set.Set) (*spaceGeometry, error) {
	if !read.InSpace() || !read.HasSeed() || len(read.Unrendered()) != 0 {
		return nil, fmt.Errorf("selection requires supported space geometry and an explicit scene seed")
	}
	geometry := &spaceGeometry{}
	for _, each := range read.Things() {
		if each.Recipe() != "" || each.Alight() {
			return nil, fmt.Errorf("selection does not support recipes or burning placements")
		}
		row, err := manufacture.KindNamed(each.Kind())
		if err != nil {
			return nil, err
		}
		specimen, err := Make(Construction{Kind: row, SceneSeed: read.Seed(), ObjectSeed: int(each.Seed()), Complexity: each.Complexity()})
		if err != nil {
			return nil, err
		}
		things, err := specimen.Things(SpaceAt(each))
		if err != nil {
			return nil, err
		}
		if err := geometry.addSurfaces(each.Name(), things); err != nil {
			return nil, err
		}

	}
	if said, has := read.World(); has {
		world := thing.NewWorld()
		if err := stage.Add(world, said, maths.Vec{}); err != nil {
			return nil, err
		}
		for _, part := range world.Things {
			if err := geometry.addSurfaces("", []thing.Thing{*part}); err != nil {
				return nil, err
			}
		}
	}
	return geometry, nil
}

// addSurfaces keeps exact native geometry; unnamed surfaces block selection.
func (geometry *spaceGeometry) addSurfaces(name string, things []thing.Thing) error {
	for _, part := range things {
		if part.Hidden() {
			continue
		}
		hitter, ok := part.Shape.(shape.Hitter)
		if !ok {
			return fmt.Errorf("native shape for %s does not support exact intersection", name)
		}
		geometry.surfaces = append(geometry.surfaces, selectionSurface{name: name, shape: hitter, at: part.At(0)})
	}
	return nil
}

// Hit returns the nearest native manufactured placement name, or empty on miss.
func (geometry *spaceGeometry) Hit(ray maths.Ray) string {
	if geometry == nil {
		return ""
	}
	for _, value := range []float64{ray.At.X, ray.At.Y, ray.At.Z, ray.Dir.X, ray.Dir.Y, ray.Dir.Z} {
		if math.IsNaN(value) || math.IsInf(value, 0) {
			return ""
		}
	}
	if ray.Dir.Len() == 0 {
		return ""
	}
	ray.Dir = ray.Dir.Unit()
	nearest, name := math.Inf(1), ""
	for _, surface := range geometry.surfaces {
		distance, hit := surface.shape.Hit(surface.at.RayIn(ray), 0, nearest)
		if hit && distance < nearest {
			nearest, name = distance, surface.name
		}
	}
	return name
}
