package trace

import (
	"fmt"

	"github.com/janearc/slayer/bioforms/diffusion"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
)

type subsurfaceLight struct {
	// Diffused unit-albedo light and the weights it replaces at the hit.
	// Both fields are indexed by the posed mesh's vertices.
	light, share []maths.Vec
}

// prepareSubsurface evaluates lamp and sky irradiance in scene space using
// the ordinary CPU shadow queries. Only the diffusion distances use mesh
// space, so translating the mesh does not change the distance calculation.
func (scene *staged) prepareSubsurface() error {
	for i := range scene.crowd.things {
		placed := &scene.crowd.things[i]
		material := placed.thing.Material
		if material == nil || !material.Subsurface.Enabled() {
			continue
		}
		skin, meshed := placed.thing.Shape.(*mesh.Mesh)
		if !meshed || !skin.Smooth() || placed.scale <= 0 || !finite(placed.scale) ||
			material.Through != 0 || material.Metal != nil {
			return fmt.Errorf("trace: subsurface material %q needs a smooth, opaque dielectric mesh with positive scale", material.Name)
		}
		if scene.oceanSun != nil {
			return fmt.Errorf("trace: subsurface material %q cannot use the ocean mean-plane source", material.Name)
		}
		points := skin.Points()
		normals := skin.VertexNormals()
		incident := make([]maths.Light, len(points))
		white := maths.Grey(1)
		for vertex, point := range points {
			normal := placed.at.Orientation.Out(normals[vertex]).Unit()
			if normal == (maths.Vec{}) {
				continue
			}
			world := placed.at.Out(point)
			incident[vertex] = scene.lamplight(world, normal, white).
				Add(scene.skylight(world, normal, white))
		}
		profile := material.Subsurface.Profile()
		probes, err := diffusion.SurfaceProbes(diffusion.Surface{
			Points: points, Faces: skin.Faces(), Incident: incident, Radius: profile.Radius,
		})
		if err != nil {
			return fmt.Errorf("trace: subsurface material %q: %w", material.Name, err)
		}
		field := subsurfaceLight{
			light: make([]maths.Vec, len(probes)),
			share: make([]maths.Vec, len(probes)),
		}
		for vertex, probe := range probes {
			field.light[vertex] = probe.Nearby.Scale(profile.Share)
			field.share[vertex] = probe.Share.Scale(profile.Share)
		}
		placed.subsurface = field
	}
	return nil
}

// subsurfaceAt interpolates the irradiance field only on the oriented front
// of a mesh. Its reverse side has no through-thickness transport.
func (met hit) subsurfaceAt(ray maths.Ray) (nearby, weight maths.Light) {
	field := met.thing.subsurface
	if field.light == nil || !met.on.meshed {
		return nearby, weight
	}
	skin := met.thing.thing.Shape.(*mesh.Mesh)
	a, b, c := skin.Corners(met.on.met.Face())
	direction := met.thing.at.Orientation.In(ray.Dir)
	if b.Sub(a).Cross(c.Sub(a)).Dot(direction) >= 0 {
		return nearby, weight
	}
	light, _ := skin.VertexPointAt(met.on.met, field.light)
	share, _ := skin.VertexPointAt(met.on.met, field.share)
	return maths.RGB(light.X, light.Y, light.Z), maths.RGB(share.X, share.Y, share.Z)
}
