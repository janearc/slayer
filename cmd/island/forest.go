package main

import (
	"fmt"
	"math"
	"time"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/biome"
)

type rootRecord struct {
	ID       biome.ID  `json:"id"`
	Kind     string    `json:"kind"`
	Variant  int       `json:"variant"`
	Position maths.Vec `json:"world-position-metres"`
	Detail   string    `json:"detail"`
}

// forest keeps population identity independent of camera and enumeration order.
// Distant roots are represented by authored landcover, with their construction
// omitted only when the conservative specimen envelope projects small enough.
func (site *siteBuilder) forest(minimum, maximum float64) error {
	forest := site.plan.Forest
	if forest == nil || forest.Density == 0 {
		return nil
	}
	started := time.Now()
	defer func() { site.record.ForestPreparation = time.Since(started) }()
	mixture := make([]biome.Species, len(forest.Mixture))
	for index, member := range forest.Mixture {
		mixture[index] = biome.Species{Recipe: member.Kind, Weight: member.Weight}
		for variant := 0; variant < forest.Variety; variant++ {
			value, err := site.load(prototypeKey{member.Kind, "hull", forest.Seed + uint64(variant)})
			if err != nil {
				return err
			}
			if err := forest.contains(value.bounds); err != nil {
				return err
			}
		}
	}
	bounds := forest.Bounds
	region, err := biome.New(biome.Definition{Name: forest.Name, Seed: forest.Seed,
		Bounds: biome.Rect{MinEast: bounds.MinEast, MinSouth: bounds.MinSouth, MaxEast: bounds.MaxEast, MaxSouth: bounds.MaxSouth},
		Cell:   measure.Metres(forest.Cell), PerHectare: forest.Density, Species: mixture})
	if err != nil {
		return err
	}
	box := site.regionBounds(bounds, minimum, maximum, forest.Height+forest.Radius)
	diameter := math.Sqrt(8*forest.Radius*forest.Radius + (forest.Height+forest.Radius)*(forest.Height+forest.Radius))
	assessment, err := site.view.Assess(box, measure.Metres(diameter))
	if err != nil {
		return err
	}
	if !assessment.Visible {
		site.record.RegionSkipped = "outside view"
		return nil
	}
	if err := site.canopyAppearance(); err != nil {
		return err
	}
	if assessment.Pixels <= forest.Thresholds[0] {
		site.record.RegionAggregated = true
		return nil
	}
	envelope := maths.Box{Min: maths.V(-forest.Radius, -forest.Radius, -forest.Radius), Max: maths.V(forest.Radius, forest.Height, forest.Radius)}
	var roots []forestCandidate
	selected := time.Now()
	if forest.Adaptive {
		envelope, err = site.completeEnvelope()
		if err != nil {
			return err
		}
		diameter = envelope.Max.Sub(envelope.Min).Len()
		roots, err = site.visibleRoots(region, bounds, minimum, maximum, envelope, diameter)
	} else {
		selection, selectionError := region.SelectWithUsage(region.Bounds(), biome.Budget{Cells: site.plan.Budget.Cells, Candidates: site.plan.Budget.Candidates})
		err = selectionError
		for _, root := range selection.Roots {
			roots = append(roots, forestCandidate{root, minimum, maximum})
		}
		site.record.ExaminedCells, site.record.GeneratedCandidates = selection.Cells, selection.Generated
	}
	if err != nil {
		return err
	}
	site.record.ForestSelection = time.Since(selected)
	placed := time.Now()
	defer func() { site.record.ForestPlacement = time.Since(placed) }()
	extent := math.Max(envelope.Min.Len(), envelope.Max.Len())
	for _, candidate := range roots {
		if err := site.ctx.Err(); err != nil {
			return err
		}
		edge := edgeWeight(forest.Bounds, forest.EdgeFade, candidate.East, candidate.South)
		if edge == 0 {
			continue
		}
		excluded := false
		for _, clearing := range site.clearings {
			if math.Hypot(candidate.East-clearing.east, candidate.South-clearing.south) < clearing.radius+forest.Radius+forest.Clearance {
				excluded = true
				break
			}
		}
		if excluded {
			continue
		}
		bound, err := site.view.Assess(site.candidateBounds(candidate, extent), measure.Metres(diameter))
		if err != nil {
			return err
		}
		if !bound.Visible || bound.Pixels <= forest.Thresholds[0] {
			site.record.BoundedRoots++
			continue
		}
		position, height, err := site.ground(candidate.East, candidate.South)
		if err != nil {
			return err
		}
		if height < forest.MinimumHeight || height > forest.MaximumHeight {
			continue
		}
		// The upper 53 bits give a stable binary64 share for density thinning.
		if math.Ldexp(float64(candidate.Seed>>11), -53) >= edge*heightWeight(forest.MinimumHeight, forest.MaximumHeight, forest.HeightFade, height) {
			continue
		}
		basis, err := site.frame.Tangent(position)
		if err != nil {
			return err
		}
		pose := maths.NewSubspace(position, basis.Then(maths.Rotation(maths.YAxis, candidate.Turn*2*math.Pi)), 1)
		view, err := site.view.Assess(transformedBounds(envelope, pose), measure.Metres(diameter))
		if err != nil {
			return err
		}
		if !view.Visible {
			continue
		}
		variant := int(candidate.Seed % uint64(forest.Variety))
		detail := forest.detail(view.Pixels)
		root := rootRecord{ID: candidate.ID, Kind: candidate.Recipe, Variant: variant, Position: position, Detail: detail}
		site.record.Roots = append(site.record.Roots, root)
		if detail == "aggregate" {
			site.record.Aggregated++
			continue
		}
		value, err := site.load(prototypeKey{candidate.Recipe, detail, forest.Seed + uint64(variant)})
		if err != nil {
			return err
		}
		if err := forest.contains(value.bounds); err != nil {
			return err
		}
		if err := site.add(value, pose); err != nil {
			return err
		}
		site.record.Trees++
	}
	return nil
}

// contains refuses a specimen outside its declared visibility envelope.
func (forest forestPlan) contains(box maths.Box) error {
	if box.Min.X < -forest.Radius || box.Max.X > forest.Radius ||
		box.Min.Z < -forest.Radius || box.Max.Z > forest.Radius ||
		box.Min.Y < -forest.Radius || box.Max.Y > forest.Height {
		return fmt.Errorf("island: plant prototype exceeds its declared visibility envelope")
	}
	return nil
}

// detail selects a tier from the projected bound and applies the authored floor.
func (forest forestPlan) detail(pixels float64) string {
	for index, detail := range []string{"aggregate", "hull", "clumps", "quick"} {
		if pixels <= forest.Thresholds[index] {
			if detail == "hull" && forest.GeometryMinimum != "" {
				return forest.GeometryMinimum
			}
			return detail
		}
	}
	return "demo"
}

// regionBounds encloses the authored spherical chart and radial height range.
// Its conservative extent is independent of the visible candidate list.
func (site *siteBuilder) regionBounds(bounds localBounds, minimum, maximum, extent float64) maths.Box {
	radius := site.frame.Radius().InMetres()
	east, south := (bounds.MinEast+bounds.MaxEast)/2, (bounds.MinSouth+bounds.MaxSouth)/2
	angle := math.Hypot(bounds.MaxEast-bounds.MinEast, bounds.MaxSouth-bounds.MinSouth) / (2 * radius)
	centre := site.frame.Centre().Add(site.chartUp(east, south).Scale(radius + minimum/2 + maximum/2))
	margin := math.Nextafter((radius+maximum)*angle+(maximum-minimum)/2+extent, math.Inf(1))
	delta := maths.V(margin, margin, margin)
	return maths.Box{Min: centre.Sub(delta), Max: centre.Add(delta)}
}
