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
	object, err := makeSpecimen(entry.Kind, plan.Seed, detail)
	if err != nil {
		return nil, err
	}
	return plan.compose(entry, object)
}

// makeSpecimen shares checked construction between subjects and scale props.
func makeSpecimen(name string, seed uint64, detail string) (specimenSource, error) {
	row, lookupError := manufacture.KindNamed(name)
	if lookupError != nil {
		return nil, lookupError
	}
	object, constructionError := manufacture.MakeWithComplexity(row,
		manufacture.Sow(seed), detail)
	if constructionError != nil {
		return nil, constructionError
	}
	return object, nil
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
	bounds := object.Bounds()
	if plan.Presentation == "cove" {
		heading := maths.About(maths.YAxis, entry.Yaw*maths.Degree)
		var count int
		var err error
		bounds, count, err = addCove(world, bounds, heading.Out(entry.camera(plan.Camera)), "quick")
		if err != nil {
			return nil, err
		}
		faces += count
	}
	if faces > plan.FaceLimit {
		return nil, fmt.Errorf("%s needs %d stored faces; limit %d",
			entry.Kind, faces, plan.FaceLimit)
	}
	lit := entry.staged(plan)
	if err := lit.illuminate(world, object.Descriptions()); err != nil {
		return nil, err
	}
	display := lit
	display.Camera = entry.camera(plan.Camera)
	framed, frameError := display.view(world, bounds, faces,
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
	var err error
	direction, err = plan.orbit(direction, up)
	if err != nil {
		return nil, err
	}
	from := centre.Add(direction.Scale(radius * plan.Distance))
	if plan.Interior != nil {
		from, centre, up = plan.Interior.At, plan.Interior.Look, maths.YAxis
	}
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

// orbit turns a framed camera about its entry's up and local right axes.
func (plan plan) orbit(direction, up maths.Vec) (maths.Vec, error) {
	if plan.Orbit == (orbit{}) {
		return direction, nil
	}
	up = up.Unit()
	if plan.Orbit.Yaw != zeroLength {
		direction = maths.Rotation(up,
			plan.Orbit.Yaw*maths.Degree).Out(direction)
	}
	vertical := direction.Dot(up)
	horizontal := direction.Sub(up.Scale(vertical))
	elevation := math.Atan2(vertical, horizontal.Len())
	finalElevation := elevation + plan.Orbit.Pitch*maths.Degree
	limit := orbitLimit * maths.Degree
	if finalElevation > limit {
		finalElevation = limit
	} else if finalElevation < -limit {
		finalElevation = -limit
	}
	pitch := finalElevation - elevation
	if pitch != zeroLength {
		frame, err := maths.View(direction, maths.V(0, 0, 0), up)
		if err != nil {
			return maths.Vec{}, err
		}
		right := frame.Orientation.X
		direction = maths.Rotation(right.Neg(),
			pitch).Out(direction)
	}
	return direction, nil
}

// illuminate preserves attached spectra, pulse time and source exclusion.
func (plan plan) illuminate(world *thing.World,
	attached []manufacture.AttachedLight,
) error {
	world.Sky = plan.Sky
	if plan.Presentation == "space" {
		if plan.starField == nil {
			return fmt.Errorf("space presentation has no verified Gaia star field")
		}
		world.Sky = plan.starField
	}
	if err := plan.interiorSky(world); err != nil {
		return err
	}
	for _, source := range plan.Sources {
		lamp, err := emit.NewDistant(source.Toward, source.Strength)
		if err != nil {
			return err
		}
		world.Lamp(lamp)
	}
	for _, source := range plan.LocalLights {
		world.Lamp(emit.Lit{At: source.At, Strength: source.Strength})
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
