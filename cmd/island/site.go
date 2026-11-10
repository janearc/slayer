package main

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/emit"
	"github.com/janearc/slayer/manufacture"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
	"github.com/janearc/slayer/scene/footprint"
	"github.com/janearc/slayer/scene/frame"
	"github.com/janearc/slayer/scene/island"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
)

type placedObject struct {
	Name     string    `json:"name"`
	Kind     string    `json:"kind"`
	Detail   string    `json:"detail"`
	Position maths.Vec `json:"world-position-metres"`
	Ground   float64   `json:"radial-ground-height-metres"`
}

type siteRecord struct {
	ForestPreparation   time.Duration  `json:"forest-preparation-nanoseconds"`
	ForestSelection     time.Duration  `json:"forest-region-selection-nanoseconds"`
	ForestPlacement     time.Duration  `json:"forest-placement-nanoseconds"`
	BoundedRoots        int            `json:"candidates-culled-before-height-read"`
	GroundQueries       int            `json:"source-ground-placement-queries"`
	CanopyArea          float64        `json:"mean-vertical-specimen-area-square-metres"`
	CanopyCover         float64        `json:"approximate-overhead-canopy-cover"`
	CanopyAreaSamples   int            `json:"canopy-area-rays"`
	Objects             []placedObject `json:"objects"`
	Trees               int            `json:"drawn-trees"`
	Aggregated          int            `json:"aggregated-candidates"`
	RegionAggregated    bool           `json:"whole-region-aggregated"`
	RegionSkipped       string         `json:"whole-region-skipped-reason,omitempty"`
	ExaminedCells       int            `json:"examined-cells"`
	ExaminedRegions     int            `json:"examined-regions"`
	AggregateRegions    int            `json:"aggregate-regions"`
	GeneratedCandidates int            `json:"generated-candidates"`
	Prototypes          int            `json:"prototypes"`
	StoredFaces         int            `json:"prototype-stored-faces"`
	Instances           int            `json:"placed-material-groups"`
	Roots               []rootRecord   `json:"selected-root-records,omitempty"`
}

type prototypeKey struct {
	kind, detail string
	seed         uint64
}
type prototype struct {
	parts  []thing.Thing
	lights []manufacture.AttachedLight
	bounds maths.Box
	areas  map[int]float64
}

// prototypeStore owns bounded, camera-independent geometry across frames.
// Frame placements and lights never enter this cache.
type prototypeStore struct {
	values map[prototypeKey]prototype
	faces  int
}

type clearing struct{ east, south, radius float64 }

type siteBuilder struct {
	ctx       context.Context
	plan      *sitePlan
	frame     frame.Sphere
	heights   island.Heights
	view      footprint.Perspective
	relative  frame.Relative
	space     maths.Subspace
	world     *thing.World
	seconds   float64
	store     *prototypeStore
	clearings []clearing
	record    siteRecord
}

// compose fixes the site's chart once in the world. Changing the camera only
// changes visibility, surface representation and the final render translation.
func (plan plan) compose(ctx context.Context, world *thing.World, sphere frame.Sphere,
	heights island.Heights, view footprint.Perspective, origin frame.Relative, store *prototypeStore,
) (*siteRecord, error) {
	if plan.Site == nil {
		return nil, nil
	}
	point := plan.Site.At
	height, has, err := heights.Height(point.Longitude, point.Latitude)
	if err != nil {
		return nil, err
	}
	if !has {
		return nil, fmt.Errorf("island: site has no source height")
	}
	position, err := sphere.Position(frame.Location{Longitude: measure.Degrees(point.Longitude),
		Latitude: measure.Degrees(point.Latitude), Height: measure.Metres(height)})
	if err != nil {
		return nil, err
	}
	basis, err := sphere.Tangent(position)
	if err != nil {
		return nil, err
	}
	site := &siteBuilder{ctx: ctx, plan: plan.Site, frame: sphere, heights: heights, view: view,
		relative: origin, space: maths.NewSubspace(position, basis, 1),
		world: world, seconds: plan.Time, store: store}
	for _, object := range site.plan.Objects {
		if err := site.object(object); err != nil {
			return nil, err
		}
	}
	if forest := site.plan.Forest; forest != nil && forest.Density > 0 {
		diameter := math.Sqrt(8*forest.Radius*forest.Radius + (forest.Height+forest.Radius)*(forest.Height+forest.Radius))
		for _, object := range world.Things {
			if paint, ok := object.Material.Texture.(*altitudePaint); ok {
				paint.forest = &forestPaint{space: site.space, sphere: site.frame, edgeFade: forest.EdgeFade, heightFade: forest.HeightFade, bounds: forest.Bounds, minimum: forest.MinimumHeight,
					maximum: forest.MaximumHeight, fadeFrom: diameter / (2 * forest.Thresholds[0]), fadeTo: diameter / forest.Thresholds[0],
					colour: rgb(forest.AggregateColour), clearance: forest.Clearance, clearings: append([]clearing(nil), site.clearings...)}
				break
			}
		}
	}
	if err := site.forest(plan.Detail.MinimumHeight, plan.Detail.MaximumHeight); err != nil {
		return nil, err
	}
	if err := site.mist(); err != nil {
		return nil, err
	}
	if err := site.lasers(); err != nil {
		return nil, err
	}
	site.record.Prototypes = len(site.store.values)
	site.record.StoredFaces = site.store.faces
	return &site.record, nil
}

// ground maps fixed local east/south metre coordinates by a great-circle
// exponential chart on the authored reference sphere, then asks the source for
// radial height. The local population's density is defined in that chart.
func (site *siteBuilder) ground(east, south float64) (maths.Vec, float64, error) {
	if err := site.ctx.Err(); err != nil {
		return maths.Vec{}, 0, err
	}
	site.record.GroundQueries++
	up := site.chartUp(east, south)
	base := site.frame.Centre().Add(up.Scale(site.frame.Radius().InMetres()))
	location, err := site.frame.Location(base)
	if err != nil {
		return maths.Vec{}, 0, err
	}
	height, has, err := site.heights.Height(location.Longitude.InDegrees(), location.Latitude.InDegrees())
	if err != nil {
		return maths.Vec{}, 0, err
	}
	if !has {
		return maths.Vec{}, 0, fmt.Errorf("island: site point has no source height")
	}
	location.Height = measure.Metres(height)
	position, err := site.frame.Position(location)
	return position, height, err
}

// chartUp maps a fixed east/south chart point to the sphere's radial direction.
func (site *siteBuilder) chartUp(east, south float64) maths.Vec {
	distance := math.Hypot(east, south)
	up := site.space.Orientation.Y
	if distance == 0 {
		return up
	}
	angle := distance / site.frame.Radius().InMetres()
	tangent := site.space.Orientation.Out(maths.V(east/distance, 0, south/distance))
	return up.Scale(math.Cos(angle)).Add(tangent.Scale(math.Sin(angle)))
}

// load retains only renderable geometry, bounds and copied light descriptions;
// procedural construction arrays are released after this call.
func (site *siteBuilder) load(key prototypeKey) (prototype, error) {
	if value, found := site.store.values[key]; found {
		return value, nil
	}
	if key.detail == "crown" {
		source, err := site.load(prototypeKey{key.kind, "clumps", key.seed})
		if err != nil {
			return prototype{}, err
		}
		value, faces, err := crown(source)
		if err != nil {
			return prototype{}, err
		}
		if faces > site.plan.Budget.Faces-site.store.faces {
			return prototype{}, fmt.Errorf("island: crown exceeds retained prototype face budget")
		}
		site.store.faces += faces
		site.store.values[key] = value
		return value, nil
	}
	kind, err := manufacture.KindNamed(key.kind)
	if err != nil {
		return prototype{}, err
	}
	made, err := manufacture.MakeWithComplexity(kind, manufacture.Sow(key.seed), key.detail)
	if err != nil {
		return prototype{}, err
	}
	parts, err := made.Things(maths.Local())
	if err != nil {
		return prototype{}, err
	}
	faces := 0
	for _, part := range parts {
		switch value := part.Shape.(type) {
		case *mesh.Mesh:
			faces += value.FaceCount()
		case *shape.MeshGroup:
			faces += value.StoredFaces()
		default:
			return prototype{}, fmt.Errorf("island: unsupported manufactured geometry %T", value)
		}
	}
	if faces > site.plan.Budget.Faces-site.store.faces {
		return prototype{}, fmt.Errorf("island: %s %s seed %d needs %d stored faces; %d of %d already used", key.kind, key.detail, key.seed, faces, site.store.faces, site.plan.Budget.Faces)
	}
	value := prototype{parts: parts, lights: made.Descriptions(), bounds: made.Bounds()}
	site.store.faces += faces
	site.store.values[key] = value
	return value, nil
}

// add places shared geometry and its transformed lights within the frame budget.
func (site *siteBuilder) add(value prototype, pose maths.Subspace) error {
	if len(value.parts) > site.plan.Budget.Instances-site.record.Instances {
		return fmt.Errorf("island: site instance budget exhausted")
	}
	local, err := site.relative.Placement(pose)
	if err != nil {
		return err
	}
	for _, part := range value.parts {
		part.Motion = thing.Still(local)
		site.world.Add(part)
	}
	site.record.Instances += len(value.parts)
	lights, err := manufacture.PlaceLights(value.lights, local)
	if err != nil {
		return err
	}
	for _, light := range lights {
		point, err := emit.Of(light.Light(), light.At(), light.Toward(), site.plan.LightExposure)
		if err != nil {
			return err
		}
		point.Beyond = light.Beyond().InMetres()
		site.world.Lamp(point.When(time.Duration(site.seconds * float64(time.Second))))
	}
	return nil
}

// object grounds an authored recipe and reserves its horizontal clearing.
func (site *siteBuilder) object(object objectPlan) error {
	value, err := site.load(prototypeKey{object.Kind, object.Detail, object.Seed})
	if err != nil {
		return err
	}
	position, height, err := site.ground(object.East, object.South)
	if err != nil {
		return err
	}
	basis, err := site.frame.Tangent(position)
	if err != nil {
		return err
	}
	position = position.Add(basis.Y.Scale(object.Above))
	pose := maths.NewSubspace(position, basis.Then(maths.Rotation(maths.YAxis, -object.Heading*maths.Degree)), 1)
	if err := site.add(value, pose); err != nil {
		return err
	}
	radius := math.Hypot(math.Max(math.Abs(value.bounds.Min.X), math.Abs(value.bounds.Max.X)),
		math.Max(math.Abs(value.bounds.Min.Z), math.Abs(value.bounds.Max.Z)))
	site.clearings = append(site.clearings, clearing{object.East, object.South, radius})
	site.record.Objects = append(site.record.Objects, placedObject{object.Name, object.Kind, object.Detail, position, height})
	return nil
}

// transformedBounds encloses all eight posed corners in world-axis bounds.
func transformedBounds(box maths.Box, pose maths.Subspace) maths.Box {
	first := pose.Out(box.Min)
	result := maths.Box{Min: first, Max: first}
	for _, x := range []float64{box.Min.X, box.Max.X} {
		for _, y := range []float64{box.Min.Y, box.Max.Y} {
			for _, z := range []float64{box.Min.Z, box.Max.Z} {
				point := pose.Out(maths.V(x, y, z))
				result.Min = maths.V(math.Min(result.Min.X, point.X), math.Min(result.Min.Y, point.Y), math.Min(result.Min.Z, point.Z))
				result.Max = maths.V(math.Max(result.Max.X, point.X), math.Max(result.Max.Y, point.Y), math.Max(result.Max.Z, point.Z))
			}
		}
	}
	return result
}
