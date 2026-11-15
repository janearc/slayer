package main

import (
	"context"
	"fmt"
	"math"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
)

type retainedCrownFit struct {
	Kind string        `json:"kind"`
	Seed uint64        `json:"seed"`
	Fit  *crownAreaFit `json:"fit"`
}

type crownAreaFit struct {
	Side   int            `json:"samples-per-axis"`
	Bounds maths.Box      `json:"specimen-probe-bounds-metres"`
	Parts  []crownPartFit `json:"materials"`
}

type crownPartFit struct {
	Material string  `json:"material"`
	Source   float64 `json:"clump-area-square-metres"`
	Before   float64 `json:"crown-area-before-square-metres"`
	After    float64 `json:"crown-area-after-square-metres"`
	Scale    float64 `json:"horizontal-scale"`
	Limited  string  `json:"limitation,omitempty"`
}

// fitCrownArea reduces excessive opaque area without choosing materials by
// name or colour. Horizontal scaling multiplies exact vertical projected area
// by scale squared. Sampled ratios only approximate those continuous areas.
func fitCrownArea(ctx context.Context, source, proxy prototype, side int) (prototype, error) {
	if len(source.parts) != len(proxy.parts) || len(proxy.parts) == 0 {
		return prototype{}, fmt.Errorf("island: crown material correspondence is missing")
	}
	domain := proxy.bounds // crown construction already encloses the source.
	if domain.Min.X > 0 || domain.Max.X < 0 || domain.Min.Z > 0 || domain.Max.Z < 0 {
		return prototype{}, fmt.Errorf("island: crown area fitting needs the root axis inside its envelope")
	}
	wanted, err := projectedPartAreas(ctx, source, domain, side)
	if err != nil {
		return prototype{}, err
	}
	before, err := projectedPartAreas(ctx, proxy, domain, side)
	if err != nil {
		return prototype{}, err
	}
	fit := &crownAreaFit{Side: side, Bounds: domain}
	parts := make([]thing.Thing, len(proxy.parts))
	for index, part := range proxy.parts {
		if part.Material == nil || part.Material != source.parts[index].Material {
			return prototype{}, fmt.Errorf("island: crown material correspondence changed")
		}
		entry := crownPartFit{Material: part.Material.Name, Source: wanted[index], Before: before[index], Scale: 1}
		switch {
		case wanted[index] == 0 || before[index] == 0:
			entry.Limited = "zero sampled area is unresolved; original crown retained"
		case wanted[index] > before[index]:
			entry.Limited = "target needs expansion; original crown retained within its envelope"
		default:
			entry.Scale = math.Sqrt(wanted[index] / before[index])
		}
		parts[index], err = shrinkCrownPart(part, entry.Scale)
		if err != nil {
			return prototype{}, err
		}
		fit.Parts = append(fit.Parts, entry)
	}
	proxy.parts = parts
	after, err := projectedPartAreas(ctx, proxy, domain, side)
	if err != nil {
		return prototype{}, err
	}
	for index, area := range after {
		fit.Parts[index].After = area
	}
	proxy.fit = fit
	return proxy, nil
}

// shrinkCrownPart scales the small crown mesh about its root axis, preserving
// height, material and topology. It never widens the existing envelope.
func shrinkCrownPart(part thing.Thing, scale float64) (thing.Thing, error) {
	if !finiteArea(scale) || scale <= 0 || scale > 1 {
		return thing.Thing{}, fmt.Errorf("island: invalid crown area scale")
	}
	if scale == 1 {
		return part, nil
	}
	source, ok := part.Shape.(*mesh.Mesh)
	if !ok {
		return thing.Thing{}, fmt.Errorf("island: crown fitting needs a flat crown mesh")
	}
	points := source.Points()
	for index := range points {
		points[index].X *= scale
		points[index].Z *= scale
	}
	made, err := mesh.New(points, source.Faces(), nil, false)
	if err != nil {
		return thing.Thing{}, err
	}
	part.Shape = made
	return part, nil
}

// projectedPartAreas probes every material independently on one fixed grid.
// Overlapping materials each count their own hit; these areas cannot be added
// to obtain union cover or interpreted as visible material fractions.
func projectedPartAreas(ctx context.Context, value prototype, domain maths.Box, side int) ([]float64, error) {
	width, depth, height := domain.Max.X-domain.Min.X, domain.Max.Z-domain.Min.Z, domain.Max.Y-domain.Min.Y
	if side < 8 || side > 128 || width <= 0 || depth <= 0 || height <= 0 || !finiteArea(width*depth) {
		return nil, fmt.Errorf("island: invalid crown material probe grid")
	}
	hits := make([]int, len(value.parts))
	for row := 0; row < side; row++ {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		for column := 0; column < side; column++ {
			ray := maths.Ray{At: maths.V(domain.Min.X+width*(float64(column)+0.5)/float64(side), domain.Max.Y+height,
				domain.Min.Z+depth*(float64(row)+0.5)/float64(side)), Dir: maths.V(0, -1, 0)}
			for index, part := range value.parts {
				hitter, ok := part.Shape.(shape.Hitter)
				if !ok {
					return nil, fmt.Errorf("island: crown source needs exact mesh intersections")
				}
				space := part.At(0)
				scale := space.Scale
				if scale == 0 {
					scale = 1
				}
				if _, hit := hitter.Hit(space.RayIn(ray), 0, 3*height/scale); hit {
					hits[index]++
				}
			}
		}
	}
	areas := make([]float64, len(hits))
	for index, count := range hits {
		areas[index] = width * depth * float64(count) / float64(side*side)
	}
	return areas, nil
}
