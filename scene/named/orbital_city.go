package named

import (
	"fmt"
	"math"
	"time"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/emit"
	"github.com/janearc/slayer/manufacture"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/texture"
	"github.com/janearc/slayer/scene/thing"
)

// The established 5,120 m city is magnified to 1,280 km across, making
// individual inhabited towers visible beside the larger orbital structures.
// The offset keeps its rectangle away from the focal bridge's footprint.
const orbitalCityScale = 250.0

// Ten metres lifts the road slightly above the ring while keeping the city
// within the same flattened site; the paving sits five metres below the road.
var orbitalCityOffset = maths.V(0, 10, -1400000)

// surfaceCity wraps the full clustered ten-thousand-building city, including
// parks, highways, full tree prototypes and occupied night windows. Building
// chunks conform to the ring; repeated tree prototypes remain instanced and
// stand rigidly in each location's tangent frame instead of being duplicated.
// The recipe's six independently shadowed street/park fixtures remain actual
// lamps. Their callback must be composed into Built.tick by the scene owner.
//
// The existing GPU rejects a flood in a scene with more than eight lamps; the
// full orbital has nine. This is a renderer limitation, not a reason to drop
// fixtures or silently change illumination. The CPU supports the full scene.
func (one hoop) surfaceCity(world *thing.World, centre maths.Vec, seed uint64, day time.Duration, urbanDensity ...bool) (func(time.Duration), error) {
	spec := manufacture.DefaultClusteredCity()
	spec.Night = true
	trees := manufacture.DefaultCityTrees()
	if len(urbanDensity) > 0 && urbanDensity[0] {
		spec.ParkRelief = true
		spec.ParkPathWidth = measure.Metres(4 / orbitalCityScale)
		trees.SizeScale = 1 / orbitalCityScale
		trees.ParkEnrichment = true
		trees.UrbanRows = true
	}
	spec.Trees = &trees
	kind, err := manufacture.ClusteredCity(spec)
	if err != nil {
		return nil, err
	}
	made, err := manufacture.Make(kind, manufacture.Sow(seed))
	if err != nil {
		return nil, err
	}
	parts, err := made.Things(maths.Local())
	if err != nil {
		return nil, err
	}
	for _, part := range parts {
		group, ok := part.Shape.(*shape.MeshGroup)
		if !ok {
			return nil, fmt.Errorf("orbital city needs material mesh groups, got %T", part.Shape)
		}
		wrapped, err := one.cityGroup(group, centre)
		if err != nil {
			return nil, err
		}
		part.Shape = wrapped
		part.Motion = one.turning(day)
		material := *part.Material
		if material.Texture != nil {
			material.Texture = orbitalCityPaint{material.Texture, one, centre}
		}
		if material.Shine != nil {
			material.Shine = orbitalCityPaint{material.Shine, one, centre}
		}
		part.Material = &material
		world.Add(part)
	}
	return one.cityLights(world, centre, day, made.Descriptions())
}

// cityLights preserves each factory fixture as an independent GPU-recognized
// emitter. The location and direction follow the ring at every frame time.
func (one hoop) cityLights(world *thing.World, centre maths.Vec, day time.Duration, descriptions []manufacture.AttachedLight) (func(time.Duration), error) {
	points := make([]emit.Point, 0, len(descriptions))
	for _, description := range descriptions {
		point, err := emit.Of(description.Light(), maths.Vec{}, maths.YAxis, orbitalCityScale*orbitalCityScale*1e-12)
		if err != nil {
			return nil, err
		}
		point.Beyond = description.Beyond().InMetres() * orbitalCityScale * 1e-6
		points = append(points, point)
	}
	first := len(world.Lamps)
	for range points {
		world.Lamp(emit.Lit{})
	}
	advance := func(at time.Duration) {
		turn := one.turning(day).At(at)
		for i, description := range descriptions {
			local := description.At().Scale(orbitalCityScale).Add(orbitalCityOffset)
			anchor := one.surfacePoint(centre, local)
			frame := one.surfaceFrame(anchor)
			point := points[i]
			point.At = turn.Out(anchor)
			point.Toward = turn.Orientation.Out(frame.Orientation.Out(description.Toward())).Unit()
			world.Lamps[first+i] = point.When(at)
		}
	}
	advance(0)
	return advance, nil
}

// cityGroup curves unique city chunks while retaining repeated foliage meshes.
func (one hoop) cityGroup(group *shape.MeshGroup, centre maths.Vec) (*shape.MeshGroup, error) {
	counts := make(map[shape.Shape]int)
	for i := 0; i < group.MemberCount(); i++ {
		counts[group.Member(i).Mesh]++
	}
	members := make([]shape.MeshMember, 0, group.MemberCount())
	for i := 0; i < group.MemberCount(); i++ {
		member := group.Member(i)
		if counts[member.Mesh] > 1 {
			member.Space = one.cityInstance(centre, member.Space)
		} else {
			bent, err := one.cityChunk(centre, member)
			if err != nil {
				return nil, err
			}
			member = shape.MeshMember{Mesh: bent, Space: maths.Local()}
		}
		members = append(members, member)
	}
	return shape.NewMeshGroup(members)
}

// cityChunk bends every vertex of a distinct material chunk to the native floor.
func (one hoop) cityChunk(centre maths.Vec, member shape.MeshMember) (*mesh.Mesh, error) {
	local, ok := member.Mesh.(*mesh.Mesh)
	if !ok {
		return nil, fmt.Errorf("orbital city needs native flat mesh, got %T", member.Mesh)
	}
	points := local.Points()
	for i, point := range points {
		city := member.Space.Out(point).Scale(orbitalCityScale).Add(orbitalCityOffset)
		points[i] = one.surfacePoint(centre, city)
	}
	return mesh.New(points, local.Faces(), nil, false)
}

// cityInstance keeps a shared plant rigid, anchored and upright on the ring.
func (one hoop) cityInstance(centre maths.Vec, local maths.Subspace) maths.Subspace {
	place := local.Origin.Scale(orbitalCityScale).Add(orbitalCityOffset)
	anchor := one.surfacePoint(centre, place)
	frame := one.surfaceFrame(anchor)
	frame.Origin = anchor
	frame.Scale = orbitalCityScale * 1e-6
	local.Origin = maths.Vec{}
	return local.Within(frame)
}

// orbitalCityPaint converts a ring-local hit back into the factory's metre
// coordinates, so native foliage and other material textures follow the city.
type orbitalCityPaint struct {
	paint  texture.Texture
	ring   hoop
	centre maths.Vec
}

// At returns the light at a world point, mapped into the city's local metres.
func (paint orbitalCityPaint) At(point maths.Vec) maths.Light {
	angle := math.Atan2(point.X, point.Z)
	delta := math.Remainder(angle-math.Atan2(paint.centre.X, paint.centre.Z), 2*math.Pi)
	metres := maths.V(delta*paint.ring.radius*1e6, (paint.ring.surfaceRadius(angle)-math.Hypot(point.X, point.Z))*1e6, (point.Y-paint.centre.Y)*1e6)
	return paint.paint.At(metres.Sub(orbitalCityOffset).Scale(1 / orbitalCityScale))
}
