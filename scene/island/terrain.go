// Package island prepares sourced geographic terrain in one spherical world.
// A view selects geometry; source coverage and the world's frame stay fixed.
package island

import (
	"context"
	"fmt"
	"math"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
	"github.com/janearc/slayer/scene/footprint"
	"github.com/janearc/slayer/scene/frame"
	"github.com/janearc/slayer/terrain"
)

// Heights returns radial heights above the scene's declared reference sphere.
// Its owner performs or explicitly names the source/datum approximation. An
// absent sample is not ocean: preparation refuses incomplete source coverage.
type Heights interface {
	Height(longitude, latitude float64) (float64, bool, error)
}

// Region is a non-wrapping longitude/latitude rectangle in degrees.
type Region struct{ West, South, East, North float64 }

// Detail supplies authored work limits and a projected cell-span target.
// CellPixels controls sampling density, not a certified omitted-relief error.
// Height bounds must enclose the complete represented source, including peaks
// between selected samples. Points bounds retained sampled vertex records.
type Detail struct {
	Patches, Evaluations, Depth, Side, Points int
	CellPixels, MinimumHeight, MaximumHeight  float64
	// NormalLongitude/Latitude are positive source-coordinate stencil steps
	// in degrees. Both zero preserves flat shading. A full source halo is
	// required; these steps are authored sampling choices, not error bounds.
	NormalLongitude, NormalLatitude float64
}

// Terrain retains an immutable frame, source and region across camera views.
// The source may own bounded caches; calls to Prepare must remain serial.
type Terrain struct {
	frame  frame.Sphere
	from   Heights
	region Region
	detail Detail
	root   terrain.Patch
}

// New checks source, frame, region and all storage/work counts before sampling.
// Side-1 is a power of two so every coarse edge knot also belongs to a finer
// patch lattice. No radius, terrain height or sampling target is inferred.
func New(sphere frame.Sphere, from Heights, region Region, detail Detail) (*Terrain, error) {
	if from == nil || !finite(sphere.Radius().InMetres()) || sphere.Radius().InMetres() <= 0 ||
		!finite(region.West) || !finite(region.East) || !finite(region.South) || !finite(region.North) ||
		region.West < -180 || region.East > 180 || region.South < -90 || region.North > 90 ||
		region.West >= region.East || region.South >= region.North {
		return nil, fmt.Errorf("island: need a spherical frame, height source and non-wrapping region")
	}
	if detail.Side < 3 || detail.Side > 65 || (detail.Side-1)&(detail.Side-2) != 0 ||
		detail.Depth < 0 || detail.Depth > 20 || detail.Patches < 1 || detail.Evaluations < 1 ||
		detail.Points < detail.Side || detail.Side > detail.Points/detail.Side ||
		!finite(detail.CellPixels) || detail.CellPixels <= 0 ||
		!finite(detail.MinimumHeight) || !finite(detail.MaximumHeight) ||
		!finite(detail.NormalLongitude) || !finite(detail.NormalLatitude) ||
		detail.NormalLongitude < 0 || detail.NormalLatitude < 0 ||
		(detail.NormalLongitude == 0) != (detail.NormalLatitude == 0) ||
		detail.NormalLongitude > 180 || detail.NormalLatitude > 90 ||
		detail.MinimumHeight > detail.MaximumHeight ||
		sphere.Radius().InMetres()+detail.MinimumHeight <= 0 {
		return nil, fmt.Errorf("island: invalid terrain detail or height envelope")
	}
	span := (detail.Side - 1) << detail.Depth
	return &Terrain{frame: sphere, from: from, region: region, detail: detail,
		root: terrain.Patch{MinColumn: 0, MinRow: 0, MaxColumn: span, MaxRow: span}}, nil
}

// Prepared holds complete view-specific meshes, already relative to Origin.
// Limited reports an unmet cell-span target, including near-plane uncertainty.
// The count is sampled source positions, before mesh constructor storage.
type Prepared struct {
	Surfaces            []*mesh.Mesh
	Patches             []terrain.Patch
	Origin              maths.Vec
	Limited             bool
	Samples             int
	NormalStencils      int
	NormalHeightQueries int
}

// Prepare selects source patches using the actual pinhole view, then matches
// shared edges in world space before subtracting one render origin. A source
// failure or budget refusal returns no partially usable Prepared value.
func (ground *Terrain) Prepare(ctx context.Context, view footprint.Perspective,
	origin frame.Relative,
) (Prepared, error) {
	if ground == nil || ground.from == nil || ctx == nil {
		return Prepared{}, fmt.Errorf("island: no prepared terrain source")
	}
	// A stitched fine edge may follow a coarse neighbour's distant height.
	// The complete root box encloses every source point and every convex
	// world-space edge join. Until per-leaf stitched bounds are available,
	// retain coarse off-screen coverage rather than culling by native heights
	// which do not enclose that leaf's final stitched representation.
	rootBounds, _, err := ground.bounds(ground.root, false)
	if err != nil {
		return Prepared{}, err
	}
	visible, err := view.Assess(rootBounds, measure.Metres(0))
	if err != nil {
		return Prepared{}, err
	}
	if !visible.Visible {
		return Prepared{Origin: origin.Origin()}, nil
	}
	selection, err := terrain.SelectPatches(ground.root, terrain.SelectionBudget{
		Patches: ground.detail.Patches, Evaluations: ground.detail.Evaluations,
		Depth: ground.detail.Depth, Target: ground.detail.CellPixels,
	}, func(patch terrain.Patch) (terrain.PatchView, error) {
		if err := ctx.Err(); err != nil {
			return terrain.PatchView{}, err
		}
		box, spacing, err := ground.bounds(patch, true)
		if err != nil {
			return terrain.PatchView{}, err
		}
		assessment, err := view.Assess(box, measure.Metres(spacing))
		return terrain.PatchView{Visible: true, Score: assessment.Pixels}, err
	})
	if err != nil {
		return Prepared{}, err
	}
	vertices := vertexBook{terrain: ground, points: make(map[lattice]maths.Vec), edges: newEdgeIndex(selection.Patches)}
	if ground.detail.NormalLongitude > 0 {
		vertices.normals = make(map[lattice]maths.Vec)
	}
	result := Prepared{Patches: selection.Patches, Origin: origin.Origin(), Limited: selection.Limited}
	for _, patch := range selection.Patches {
		if err := ctx.Err(); err != nil {
			return Prepared{}, err
		}
		surface, err := vertices.surface(patch, origin)
		if err != nil {
			return Prepared{}, err
		}
		result.Surfaces = append(result.Surfaces, surface)
	}
	result.Samples = len(vertices.points)
	result.NormalStencils = len(vertices.normals)
	result.NormalHeightQueries = vertices.normalQueries
	return result, nil
}

// location is anchored to the full region, never a selected patch's origin.
func (ground *Terrain) location(column, row float64) (longitude, latitude float64) {
	span := float64(ground.root.MaxColumn)
	return ground.region.West + column/span*(ground.region.East-ground.region.West),
		ground.region.North - row/span*(ground.region.North-ground.region.South)
}

// bounds encloses the spherical rectangle by a centre and a path-length ball.
// Latitude then longitude provides an upper bound on spherical travel from
// the rectangle's middle. The supplied complete height envelope is separate
// from the cell-span criterion; no sparse DEM estimate is called an error bound.
func (ground *Terrain) bounds(patch terrain.Patch, indexed bool) (maths.Box, float64, error) {
	west, north := ground.location(float64(patch.MinColumn), float64(patch.MinRow))
	east, south := ground.location(float64(patch.MaxColumn), float64(patch.MaxRow))
	minimum, maximum := ground.detail.MinimumHeight, ground.detail.MaximumHeight
	if bounded, ok := ground.from.(interface {
		HeightRange(float64, float64, float64, float64) (float64, float64, bool, error)
	}); indexed && ok {
		low, high, has, err := bounded.HeightRange(west, south, east, north)
		if err != nil {
			return maths.Box{}, 0, err
		}
		if has {
			if !finite(low) || !finite(high) || low < minimum || high > maximum || low > high {
				return maths.Box{}, 0, fmt.Errorf("island: native source bounds exceed the declared height envelope")
			}
			minimum, maximum = low, high
		}
	}
	height := minimum/2 + maximum/2
	centre, err := ground.frame.Position(frame.Location{
		Longitude: measure.Degrees(west/2 + east/2), Latitude: measure.Degrees(north/2 + south/2),
		Height: measure.Metres(height),
	})
	if err != nil {
		return maths.Box{}, 0, err
	}
	radius := ground.frame.Radius().InMetres() + maximum
	arc := ((east - west) + (north - south)) * maths.Degree
	margin := math.Nextafter(radius*arc/2+(maximum-minimum)/2, math.Inf(1))
	spacing := radius * arc / float64(ground.detail.Side-1)
	delta := maths.V(margin, margin, margin)
	box := maths.Box{Min: centre.Sub(delta), Max: centre.Add(delta)}
	if !finite(margin) || !finite(spacing) || !point(box.Min) || !point(box.Max) {
		return maths.Box{}, 0, fmt.Errorf("island: source bounds exceed the numeric range")
	}
	return box, spacing, nil
}

type lattice struct{ column, row int }

type vertexBook struct {
	terrain       *Terrain
	points        map[lattice]maths.Vec
	edges         edgeIndex
	normals       map[lattice]maths.Vec
	normalQueries int
}

// at gives every shared native selection position one source query per view.
func (book *vertexBook) at(key lattice) (maths.Vec, error) {
	if value, found := book.points[key]; found {
		return value, nil
	}
	if len(book.points) >= book.terrain.detail.Points {
		return maths.Vec{}, fmt.Errorf("island: source point budget exhausted")
	}
	lon, lat := book.terrain.location(float64(key.column), float64(key.row))
	height, has, err := book.terrain.from.Height(lon, lat)
	if err != nil {
		return maths.Vec{}, err
	}
	if !has {
		return maths.Vec{}, fmt.Errorf("island: no source height at %.9f longitude, %.9f latitude", lon, lat)
	}
	if !finite(height) || height < book.terrain.detail.MinimumHeight || height > book.terrain.detail.MaximumHeight {
		return maths.Vec{}, fmt.Errorf("island: source height lies outside its declared envelope")
	}
	value, err := book.terrain.frame.Position(frame.Location{
		Longitude: measure.Degrees(lon), Latitude: measure.Degrees(lat), Height: measure.Metres(height),
	})
	if err == nil {
		book.points[key] = value
	}
	return value, err
}

// finite rejects invalid scalar inputs before geometry and allocation.
func finite(value float64) bool { return !math.IsNaN(value) && !math.IsInf(value, 0) }

// point refuses a world position with any non-finite coordinate.
func point(value maths.Vec) bool { return finite(value.X) && finite(value.Y) && finite(value.Z) }
