package placement

import (
	"fmt"
	"math"
	"time"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
)

type displayBuilder struct {
	skin            DisplayMesh
	vertices        map[DisplayVertex]uint32
	positions       map[maths.Vec]uint32
	vertexPositions []uint32
	edges           map[uint64]struct{}
	components      displayComponents
	// heightFields enables labeled height-field approximations for staged views.
	heightFields bool
	// material is the name of the material of the part being added.
	material string
}

// newDisplayBuilder returns an empty display builder with its vertex,
// position and edge tables made.
func newDisplayBuilder() *displayBuilder {
	return &displayBuilder{vertices: map[DisplayVertex]uint32{}, positions: map[maths.Vec]uint32{}, edges: map[uint64]struct{}{}}
}

// thing adds one part to the display mesh. Hidden parts are skipped. The
// colour is the material colour, or the glow when the colour is unset, and
// the shape is placed where the part stands at time zero.
func (builder *displayBuilder) thing(part thing.Thing) error {
	return builder.thingAt(part, 0)
}

// thingAt adds a staged part at its scene time.
func (builder *displayBuilder) thingAt(part thing.Thing, at time.Duration) error {
	if part.Hidden() {
		return nil
	}
	colour := maths.Light{}
	builder.material = ""
	if part.Material != nil {
		builder.material = part.Material.Name
		colour = part.Material.Colour
		if colour == (maths.Light{}) {
			colour = part.Material.Glow
		}
	}
	return builder.surface(part.Shape, part.At(at), colour)
}

// surface follows native GPU geometry extraction, with analytic display tessellation.
func (builder *displayBuilder) surface(solid shape.Shape, at maths.Subspace, colour maths.Light) error {
	switch made := solid.(type) {
	case *mesh.Mesh:
		for index := range made.FaceCount() {
			a, b, c := made.Corners(index)
			normals, smooth := made.CornerNormals(index)
			builder.triangle([3]maths.Vec{a, b, c}, normals, smooth, at, colour)
		}
	case *shape.Mesh:
		smooth := made.Smooth && len(made.Normals) > 0
		if smooth && len(made.Normals) != len(made.Vertices) {
			return fmt.Errorf("working mesh has invalid smooth normals")
		}
		for _, face := range made.Faces {
			var points, normals [3]maths.Vec
			for corner, index := range face {
				points[corner] = made.Vertices[index]
				if smooth {
					normals[corner] = made.Normals[index]
				}
			}
			builder.triangle(points, normals, smooth, at, colour)
		}
	case *shape.MeshGroup:
		for i := 0; i < made.MemberCount(); i++ {
			member := made.Member(i)
			if err := builder.surface(member.Mesh, member.Space.Within(at), colour); err != nil {
				return err
			}
		}
	case shape.Sphere:
		builder.sphere(made, at, colour)
	case shape.Tube:
		builder.tube(made, at, colour)
	case *shape.HeightField:
		if !builder.heightFields {
			return fmt.Errorf("native shape %T has no finite working display mesh", solid)
		}
		if err := builder.heightField(made, at, colour); err != nil {
			return err
		}
	default:
		return fmt.Errorf("native shape %T has no finite working display mesh", solid)
	}
	if _, group := solid.(*shape.MeshGroup); !group {
		hitter, ok := displayHitter(solid)
		if !ok || !shape.ValidPlacement(at) {
			return fmt.Errorf("native display component %T has no valid placed hitter", solid)
		}
		builder.components = append(builder.components, displayComponent{hitter: hitter, at: at, material: builder.material})
	}
	return nil
}

// triangle adds one triangle to the display mesh. Corners with the same
// position, normal and colour share one vertex, and each edge is recorded
// once. Without smooth normals, every corner takes the face normal. A
// triangle with no area is skipped.
func (builder *displayBuilder) triangle(points, normals [3]maths.Vec, smooth bool, at maths.Subspace, colour maths.Light) {
	normal := points[1].Sub(points[0]).Cross(points[2].Sub(points[0])).Unit()
	if normal.Len() == 0 {
		return
	}
	var indices [3]uint32
	for corner, point := range points {
		n := normal
		if smooth {
			n = normals[corner]
		}
		vertex := DisplayVertex{Position: at.Out(point), Normal: at.Orientation.Out(n).Unit(), Colour: colour}
		index, known := builder.vertices[vertex]
		if !known {
			index = uint32(len(builder.skin.Vertices))
			builder.skin.Vertices = append(builder.skin.Vertices, vertex)
			builder.vertexPositions = append(builder.vertexPositions, builder.positionID(vertex.Position))
			builder.vertices[vertex] = index
		}
		indices[corner] = index
	}
	builder.skin.Indices = append(builder.skin.Indices, indices[:]...)
	for edge := range 3 {
		a, b := indices[edge], indices[(edge+1)%3]
		pa, pb := builder.vertexPositions[a], builder.vertexPositions[b]
		if pa > pb {
			pa, pb = pb, pa
		}
		key := uint64(pa)<<32 | uint64(pb)
		if _, found := builder.edges[key]; !found {
			builder.edges[key] = struct{}{}
			builder.skin.Edges = append(builder.skin.Edges, a, b)
		}
	}
}

// positionID returns the index of position, giving it the next index the
// first time it is seen.
func (builder *displayBuilder) positionID(position maths.Vec) uint32 {
	index, exists := builder.positions[position]
	if !exists {
		index = uint32(len(builder.positions))
		builder.positions[position] = index
	}
	return index
}

// finish returns the finished display mesh, with each native component as a
// separate hit target. It fails when no triangle was added.
func (builder *displayBuilder) finish() (*DisplayMesh, error) {
	if len(builder.skin.Indices) == 0 {
		return nil, fmt.Errorf("native working display contains no surfaces")
	}
	builder.skin.hitter = builder.components
	return &builder.skin, nil
}

// displayComponents preserves native validated surfaces as separate hit targets.
// Touching parts share GPU vertex storage without becoming a welded solid.
type displayComponent struct {
	hitter   shape.Hitter
	at       maths.Subspace
	author   thing.AuthorRef
	authored bool
	// material is the name of the material the part was made with, which
	// may be empty.
	material string
}

type displayComponents []displayComponent

// Hit returns the distance along ray to the nearest component hit before far,
// and whether any component was hit. Without a hit it returns far unchanged.
func (parts displayComponents) Hit(ray maths.Ray, near, far float64) (float64, bool) {
	far, _, found := parts.hitPart(ray, near, far)
	return far, found
}

// hitPart is Hit and also gives the material name of the component that was
// hit first, empty when it has none or nothing was hit.
func (parts displayComponents) hitPart(ray maths.Ray, near, far float64) (float64, string, bool) {
	distance, index, found := parts.hitComponent(ray, near, far)
	if !found {
		return distance, "", false
	}
	return distance, parts[index].material, true
}

// hitComponent returns the nearest surface's index, including unknown ones.
func (parts displayComponents) hitComponent(ray maths.Ray, near, far float64) (float64, int, bool) {
	found, index := false, 0
	for i, part := range parts {
		scale := part.at.Scale
		if scale == 0 {
			scale = 1
		}
		distance, hit := part.hitter.Hit(part.at.RayIn(ray), near/scale, far/scale)
		if hit {
			far, found, index = distance*scale, true, i
		}
	}
	return far, index, found
}

// These are display-only tessellation budgets, not changes to the analytic shape.
const displaySegments = 16
const displayBands = 8

// sphere adds a display tessellation of sphere, placed at at. The poles are
// single points. The sphere is cut into displayBands latitude bands and
// displaySegments longitude columns.
func (builder *displayBuilder) sphere(sphere shape.Sphere, at maths.Subspace, colour maths.Light) {
	point := func(row, column int) maths.Vec {
		if row == 0 {
			return maths.V(0, sphere.Radius, 0)
		}
		if row == displayBands {
			return maths.V(0, -sphere.Radius, 0)
		}
		latitude := math.Pi * float64(row) / displayBands
		longitude := 2 * math.Pi * float64(column%displaySegments) / displaySegments
		return maths.V(math.Sin(latitude)*math.Cos(longitude), math.Cos(latitude), math.Sin(latitude)*math.Sin(longitude)).Scale(sphere.Radius)
	}
	for row := 0; row < displayBands; row++ {
		for column := 0; column < displaySegments; column++ {
			a, b, c, d := point(row, column), point(row+1, column), point(row+1, column+1), point(row, column+1)
			if row > 0 {
				builder.sphereFace(sphere, [3]maths.Vec{a, c, d}, at, colour)
			}
			if row < displayBands-1 {
				builder.sphereFace(sphere, [3]maths.Vec{a, b, c}, at, colour)
			}
		}
	}
}

// sphereFace adds one triangle of a sphere tessellation. The corner normals
// come from the sphere, and the winding is flipped where needed so the
// triangle faces outward.
func (builder *displayBuilder) sphereFace(sphere shape.Sphere, points [3]maths.Vec, at maths.Subspace, colour maths.Light) {
	normal := points[1].Sub(points[0]).Cross(points[2].Sub(points[0]))
	if normal.Dot(points[0]) < 0 {
		points[1], points[2] = points[2], points[1]
	}
	normals := [3]maths.Vec{sphere.Normal(points[0]), sphere.Normal(points[1]), sphere.Normal(points[2])}
	builder.triangle(points, normals, true, at, colour)
}

// tube adds the display tessellation of tube, placed at at: its outer and
// inner walls, and its top and bottom rings, each split into triangles.
func (builder *displayBuilder) tube(tube shape.Tube, at maths.Subspace, colour maths.Light) {
	point := func(radius, height float64, column int) maths.Vec {
		angle := 2 * math.Pi * float64(column%displaySegments) / displaySegments
		return maths.V(radius*math.Cos(angle), height, radius*math.Sin(angle))
	}
	for column := 0; column < displaySegments; column++ {
		inner, outer := tube.Radius-tube.Thick/2, tube.Radius+tube.Thick/2
		low, high := -tube.Height/2, tube.Height/2
		a, b, c, d := point(outer, low, column), point(outer, low, column+1), point(outer, high, column+1), point(outer, high, column)
		e, f, g, h := point(inner, low, column), point(inner, low, column+1), point(inner, high, column+1), point(inner, high, column)
		for _, quad := range [][4]maths.Vec{{a, d, c, b}, {e, f, g, h}, {d, h, g, c}, {a, b, f, e}} {
			builder.triangle([3]maths.Vec{quad[0], quad[1], quad[2]}, [3]maths.Vec{}, false, at, colour)
			builder.triangle([3]maths.Vec{quad[0], quad[2], quad[3]}, [3]maths.Vec{}, false, at, colour)
		}
	}
}
