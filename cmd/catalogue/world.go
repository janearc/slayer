package main

import (
	"fmt"
	"math"
	"time"

	"github.com/janearc/slayer/emit"
	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/manufacture"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
)

// halfExtent is defined by the midpoint and radius of an interval.
const halfExtent = 0.5

// specimenEpoch is defined as the stationary specimen's zero time.
const specimenEpoch time.Duration = 0

type specimen struct {
	world   *thing.World
	eye     *maths.Subspace
	lens    *film.Pinhole
	bounds  maths.Box
	faces   int
	cutaway bool
}

// specimenSource supplies geometry and local lamps for scene preparation.
type specimenSource interface {
	Things(maths.Subspace) ([]thing.Thing, error)
	Bounds() maths.Box
	Descriptions() []manufacture.AttachedLight
}

// construct releases procedural parts after placing their meshes and lamps.
func (plan plan) construct(entry entry, detail string) (*specimen, error) {
	row, lookupError := manufacture.KindNamed(entry.Kind)
	if lookupError != nil {
		return nil, lookupError
	}
	object, constructionError := manufacture.MakeWithComplexity(row,
		manufacture.Sow(plan.Seed), detail)
	if constructionError != nil {
		return nil, constructionError
	}
	return plan.compose(entry, object)
}

// compose retains meshes and lamp descriptions after the source is released.
func (plan plan) compose(entry entry,
	object specimenSource,
) (*specimen, error) {
	parts, placementError := object.Things(maths.Local())
	if placementError != nil {
		return nil, placementError
	}
	world := thing.NewWorld()
	faces := noEntries
	hidden := false
	for _, part := range parts {
		enclosure := part.Material.Name == "observatory enclosure"
		if plan.Cutaway && enclosure {
			hidden = true
			continue
		}
		faces += storedFaces(part.Shape)
		world.Add(part)
	}
	if faces > plan.FaceLimit {
		return nil, fmt.Errorf("%s needs %d stored faces; limit %d",
			entry.Kind, faces, plan.FaceLimit)
	}
	if err := plan.illuminate(world, object.Descriptions()); err != nil {
		return nil, err
	}
	display := plan
	display.Camera = entry.camera(plan.Camera)
	framed, frameError := display.view(world, object.Bounds(), faces,
		entry.Yaw, entry.vertical())
	if framed != nil {
		framed.cutaway = hidden
	}
	return framed, frameError
}

// view frames physical bounds, keeping a detail change from moving the camera.
func (plan plan) view(world *thing.World, bounds maths.Box,
	faces int, yaw float64, up maths.Vec,
) (*specimen, error) {
	radius := bounds.Max.Sub(bounds.Min).Len() * halfExtent
	if !(radius > zeroLength) || math.IsInf(radius, bothInfinitySigns) {
		return nil, fmt.Errorf("specimen has no finite physical extent")
	}
	centre := bounds.Min.Add(bounds.Max).Scale(halfExtent)
	heading := maths.About(maths.YAxis, yaw*maths.Degree)
	direction := heading.Out(plan.Camera).Unit()
	from := centre.Add(direction.Scale(radius * plan.Distance))
	eye, err := maths.View(from, centre, up)
	if err != nil {
		return nil, err
	}
	lens, err := film.PinholeOf(plan.Wide * maths.Degree)
	if err != nil {
		return nil, err
	}
	return &specimen{world: world, eye: eye, lens: lens,
		bounds: bounds, faces: faces}, nil
}

// illuminate preserves attached spectra, pulse time and source exclusion.
func (plan plan) illuminate(world *thing.World,
	attached []manufacture.AttachedLight,
) error {
	world.Sky = plan.Sky
	for _, source := range plan.Sources {
		lamp, err := emit.NewDistant(source.Toward, source.Strength)
		if err != nil {
			return err
		}
		world.Lamp(lamp)
	}
	for _, description := range attached {
		lamp, err := emit.Of(description.Light(), description.At(),
			description.Toward(), plan.LightExposure)
		if err != nil {
			return err
		}
		lamp.Beyond = description.Beyond().InMetres()
		world.Lamp(lamp.When(specimenEpoch))
	}
	return nil
}

// storedFaces counts prototypes once, as the mesh route stores them.
func storedFaces(solid shape.Shape) int {
	switch polygon := solid.(type) {
	case *mesh.Mesh:
		return polygon.FaceCount()
	case *shape.Mesh:
		return len(polygon.Faces)
	case *shape.MeshGroup:
		return polygon.StoredFaces()
	}
	return noEntries
}
