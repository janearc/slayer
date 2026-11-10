package main

import (
	"fmt"
	"math"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/biome"
)

// forestCandidate carries the source-height envelope already established for
// its selected region. It does not claim a sampled root position.
type forestCandidate struct {
	biome.Candidate
	minimum, maximum float64
}

// completeEnvelope measures every permitted variant/detail before using a
// tighter visibility bound. A hull alone cannot certify needle overhang.
func (site *siteBuilder) completeEnvelope() (maths.Box, error) {
	forest := site.plan.Forest
	radius, low, high := 0.0, 0.0, 0.0
	details := []string{"hull", "clumps", "quick", "demo"}
	if forest.GeometryMinimum == "crown" {
		details = append(details, "crown")
	}
	for _, member := range forest.Mixture {
		for variant := 0; variant < forest.Variety; variant++ {
			for _, detail := range details {
				value, err := site.load(prototypeKey{member.Kind, detail, forest.Seed + uint64(variant)})
				if err != nil {
					return maths.Box{}, err
				}
				if err := forest.contains(value.bounds); err != nil {
					return maths.Box{}, err
				}
				b := value.bounds
				radius = math.Max(radius, math.Hypot(math.Max(math.Abs(b.Min.X), math.Abs(b.Max.X)), math.Max(math.Abs(b.Min.Z), math.Abs(b.Max.Z))))
				low, high = math.Min(low, b.Min.Y), math.Max(high, b.Max.Y)
			}
		}
	}
	return maths.Box{Min: maths.V(-radius, low, -radius), Max: maths.V(radius, high, radius)}, nil
}

// visibleRoots traverses bounded region supports before enumerating cells.
// Far supports become authored canopy appearance without visiting their roots.
// Region assessments and actual cell work share the declared cell allowance.
func (site *siteBuilder) visibleRoots(region *biome.Region, bounds localBounds, minimum, maximum float64, envelope maths.Box, diameter float64) ([]forestCandidate, error) {
	forest := site.plan.Forest
	extent := math.Max(envelope.Min.Len(), envelope.Max.Len())
	queue := []localBounds{bounds}
	var roots []forestCandidate
	for len(queue) > 0 {
		if err := site.ctx.Err(); err != nil {
			return nil, err
		}
		last := len(queue) - 1
		current := queue[last]
		queue = queue[:last]
		if site.record.ExaminedRegions+site.record.ExaminedCells >= site.plan.Budget.Cells {
			return nil, fmt.Errorf("island: forest region/cell work budget exhausted")
		}
		site.record.ExaminedRegions++
		low, high, err := site.forestHeights(current, minimum, maximum)
		if err != nil {
			return nil, err
		}
		low, high = math.Max(low, forest.MinimumHeight), math.Min(high, forest.MaximumHeight)
		if low > high {
			continue
		}
		assessment, err := site.view.Assess(site.regionBounds(current, low, high, extent), measure.Metres(diameter))
		if err != nil {
			return nil, err
		}
		if !assessment.Visible {
			continue
		}
		if assessment.Pixels <= forest.Thresholds[0] {
			site.record.AggregateRegions++
			continue
		}
		width, depth := current.MaxEast-current.MinEast, current.MaxSouth-current.MinSouth
		if width > 16*forest.Cell || depth > 16*forest.Cell {
			x, z := (current.MinEast+current.MaxEast)/2, (current.MinSouth+current.MaxSouth)/2
			queue = append(queue, localBounds{current.MinEast, current.MinSouth, x, z}, localBounds{x, current.MinSouth, current.MaxEast, z}, localBounds{current.MinEast, z, x, current.MaxSouth}, localBounds{x, z, current.MaxEast, current.MaxSouth})
			continue
		}
		selected, err := region.SelectWithUsage(biome.Rect{MinEast: current.MinEast, MinSouth: current.MinSouth, MaxEast: current.MaxEast, MaxSouth: current.MaxSouth}, biome.Budget{Cells: site.plan.Budget.Cells - site.record.ExaminedCells - site.record.ExaminedRegions, Candidates: site.plan.Budget.Candidates - site.record.GeneratedCandidates})
		if err != nil {
			return nil, err
		}
		site.record.ExaminedCells += selected.Cells
		site.record.GeneratedCandidates += selected.Generated
		for _, root := range selected.Roots {
			roots = append(roots, forestCandidate{root, low, high})
		}
	}
	return roots, nil
}

// candidateBounds encloses a root's radial height interval plus the complete
// rotated specimen envelope. It can reject a root before a source-height read;
// roots retained by this bound still take the ordinary exact placement path.
func (site *siteBuilder) candidateBounds(root forestCandidate, extent float64) maths.Box {
	up := site.chartUp(root.East, root.South)
	middle := root.minimum/2 + root.maximum/2
	centre := site.frame.Centre().Add(up.Scale(site.frame.Radius().InMetres() + middle))
	half := (root.maximum - root.minimum) / 2
	delta := maths.V(math.Abs(up.X)*half+extent, math.Abs(up.Y)*half+extent, math.Abs(up.Z)*half+extent)
	low, high := centre.Sub(delta), centre.Add(delta)
	return maths.Box{Min: maths.V(math.Nextafter(low.X, math.Inf(-1)), math.Nextafter(low.Y, math.Inf(-1)), math.Nextafter(low.Z, math.Inf(-1))),
		Max: maths.V(math.Nextafter(high.X, math.Inf(1)), math.Nextafter(high.Y, math.Inf(1)), math.Nextafter(high.Z, math.Inf(1)))}
}

// forestHeights queries a source-coordinate cap enclosing the chart rectangle.
// The sphere's exponential chart contracts transverse distances at these small
// regional extents; half the chart diagonal bounds travel from its centre.
// Unknown bounds retain the caller's global envelope instead of inventing data.
func (site *siteBuilder) forestHeights(bounds localBounds, minimum, maximum float64) (float64, float64, error) {
	bounded, ok := site.heights.(interface {
		HeightRange(float64, float64, float64, float64) (float64, float64, bool, error)
	})
	if !ok {
		return minimum, maximum, nil
	}
	radius := site.frame.Radius().InMetres()
	up := site.chartUp((bounds.MinEast+bounds.MaxEast)/2, (bounds.MinSouth+bounds.MaxSouth)/2)
	location, err := site.frame.Location(site.frame.Centre().Add(up.Scale(radius)))
	if err != nil {
		return 0, 0, err
	}
	angle := math.Hypot(bounds.MaxEast-bounds.MinEast, bounds.MaxSouth-bounds.MinSouth) / (2 * radius)
	lat, lon := location.Latitude.InRadians(), location.Longitude.InRadians()
	if math.Abs(lat)+angle >= math.Pi/2 {
		return minimum, maximum, nil
	}
	longitude := math.Asin(math.Min(1, math.Sin(angle)/math.Cos(lat)))
	if lon-longitude < -math.Pi || lon+longitude > math.Pi {
		return minimum, maximum, nil
	}
	low, high, has, err := bounded.HeightRange((lon-longitude)/maths.Degree, (lat-angle)/maths.Degree, (lon+longitude)/maths.Degree, (lat+angle)/maths.Degree)
	if err != nil {
		return 0, 0, err
	}
	if !has {
		return minimum, maximum, nil
	}
	if math.IsNaN(low) || math.IsNaN(high) || math.IsInf(low, 0) || math.IsInf(high, 0) || low > high {
		return 0, 0, fmt.Errorf("island: invalid forest source height bounds")
	}
	if low < minimum || high > maximum {
		return 0, 0, fmt.Errorf("island: forest source exceeds declared global height envelope")
	}
	return low, high, nil
}

// edgeWeight tapers density inside the region's half-open horizontal bounds.
func edgeWeight(bounds localBounds, fade, east, south float64) float64 {
	if east < bounds.MinEast || east >= bounds.MaxEast || south < bounds.MinSouth || south >= bounds.MaxSouth {
		return 0
	}
	if fade == 0 {
		return 1
	}
	distance := math.Min(math.Min(east-bounds.MinEast, bounds.MaxEast-east), math.Min(south-bounds.MinSouth, bounds.MaxSouth-south))
	return maths.Smooth(0, fade, distance)
}

// heightWeight thins both ends of an authored elevation band without moving it.
func heightWeight(minimum, maximum, fade, height float64) float64 {
	if height < minimum || height > maximum {
		return 0
	}
	if fade == 0 {
		return 1
	}
	return maths.Smooth(minimum, minimum+fade, height) * (1 - maths.Smooth(maximum-fade, maximum, height))
}
