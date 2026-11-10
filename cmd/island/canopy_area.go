package main

import (
	"context"
	"fmt"
	"math"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/shape"
)

// canopyAppearance estimates overhead cover from the actual seeded clumps.
// It averages species weights and variants, then uses a Poisson overlap
// approximation. It is neither a vegetation survey nor directional transport.
func (site *siteBuilder) canopyAppearance() error {
	forest := site.plan.Forest
	if forest.AreaSide == 0 {
		return nil
	}
	totalWeight, meanArea := 0.0, 0.0
	for _, member := range forest.Mixture {
		totalWeight += member.Weight
		for variant := 0; variant < forest.Variety; variant++ {
			key := prototypeKey{member.Kind, "clumps", forest.Seed + uint64(variant)}
			value, err := site.load(key)
			if err != nil {
				return err
			}
			if err := forest.contains(value.bounds); err != nil {
				return err
			}
			area, sampled := value.areas[forest.AreaSide]
			if !sampled {
				area, err = verticalArea(site.ctx, value, forest.AreaSide)
				if err != nil {
					return err
				}
				if value.areas == nil {
					value.areas = make(map[int]float64)
				}
				value.areas[forest.AreaSide] = area
				site.store.values[key] = value
				site.record.CanopyAreaSamples += forest.AreaSide * forest.AreaSide
			}
			meanArea += member.Weight * area / float64(forest.Variety)
		}
	}
	if totalWeight <= 0 || !finiteArea(meanArea) {
		return fmt.Errorf("island: invalid canopy mixture area")
	}
	meanArea /= totalWeight
	// Density is per hectare, an area of exactly ten thousand square metres.
	areaDensity := forest.Density * meanArea / 10000
	site.record.CanopyArea, site.record.CanopyCover = meanArea, -math.Expm1(-areaDensity)
	for _, object := range site.world.Things {
		if paint, ok := object.Material.Texture.(*altitudePaint); ok && paint.forest != nil {
			paint.forest.areaDensity, paint.forest.areaSampled = areaDensity, true
			break
		}
	}
	return nil
}

// verticalArea samples the specimen's horizontal bounding rectangle at fixed
// cell centres. A hit counts once, even when several materials overlap. This
// finite quadrature has aliasing error; the ray count is recorded, not hidden.
func verticalArea(ctx context.Context, value prototype, side int) (float64, error) {
	if side < 8 || side > 128 {
		return 0, fmt.Errorf("island: invalid canopy area sampling side")
	}
	low, high := value.bounds.Min, value.bounds.Max
	width, depth, height := high.X-low.X, high.Z-low.Z, high.Y-low.Y
	if width <= 0 || depth <= 0 || height <= 0 || !finiteArea(width*depth) {
		return 0, fmt.Errorf("island: invalid canopy sampling bounds")
	}
	hits := 0
	for row := 0; row < side; row++ {
		if err := ctx.Err(); err != nil {
			return 0, err
		}
		for column := 0; column < side; column++ {
			at := maths.V(low.X+width*(float64(column)+0.5)/float64(side), high.Y+height,
				low.Z+depth*(float64(row)+0.5)/float64(side))
			ray := maths.Ray{At: at, Dir: maths.V(0, -1, 0)}
			for _, part := range value.parts {
				hitter, ok := part.Shape.(shape.Hitter)
				if !ok {
					return 0, fmt.Errorf("island: canopy source needs exact mesh intersections")
				}
				space := part.At(0)
				scale := space.Scale
				if scale == 0 {
					scale = 1
				}
				if _, hit := hitter.Hit(space.RayIn(ray), 0, 3*height/scale); hit {
					hits++
					break
				}
			}
		}
	}
	return width * depth * float64(hits) / float64(side*side), nil
}

// finiteArea admits zero cover while refusing invalid or unbounded areas.
func finiteArea(value float64) bool { return value >= 0 && !math.IsInf(value, 0) && !math.IsNaN(value) }
