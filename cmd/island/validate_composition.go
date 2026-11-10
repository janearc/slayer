package main

import (
	"fmt"
	"math"

	"github.com/janearc/slayer/manufacture"
)

// validLocalBounds requires a nonempty east/south rectangle.
func validLocalBounds(bounds localBounds) bool {
	return bounds.MinEast < bounds.MaxEast && bounds.MinSouth < bounds.MaxSouth
}

// checkComposition holds ordering and cross-field constraints after the
// bundled schema checks scalar types, finite values and resource ceilings.
func (plan plan) checkComposition() error {
	for index, band := range plan.Landcover {
		if index > 0 && band.Below <= plan.Landcover[index-1].Below {
			return fmt.Errorf("island: landcover height boundaries must increase")
		}
		if index > 0 && 2*plan.LandcoverTransition > band.Below-plan.Landcover[index-1].Below {
			return fmt.Errorf("island: landcover transitions overlap")
		}
	}
	if plan.LandcoverTransition > 0 && len(plan.Landcover) < 2 {
		return fmt.Errorf("island: a landcover transition needs adjacent bands")
	}
	if plan.Site == nil {
		return nil
	}
	site := plan.Site
	names := make(map[string]bool)
	for _, object := range site.Objects {
		if _, err := manufacture.KindNamed(object.Kind); err != nil {
			return err
		}
		if names[object.Name] {
			return fmt.Errorf("island: duplicate site object %q", object.Name)
		}
		names[object.Name] = true
	}
	if forest := site.Forest; forest != nil {
		for _, member := range forest.Mixture {
			if _, err := manufacture.KindNamed(member.Kind); err != nil {
				return err
			}
		}
		if !validLocalBounds(forest.Bounds) || forest.MinimumHeight > forest.MaximumHeight {
			return fmt.Errorf("island: invalid forest footprint or elevation band")
		}
		if forest.EdgeFade > math.Min(forest.Bounds.MaxEast-forest.Bounds.MinEast, forest.Bounds.MaxSouth-forest.Bounds.MinSouth)/2 {
			return fmt.Errorf("island: forest edge fade exceeds its smaller footprint radius")
		}
		if 2*forest.HeightFade > forest.MaximumHeight-forest.MinimumHeight {
			return fmt.Errorf("island: forest elevation fades overlap")
		}
		for index, threshold := range forest.Thresholds {
			if index > 0 && threshold <= forest.Thresholds[index-1] {
				return fmt.Errorf("island: tree-detail thresholds must increase")
			}
		}
	}
	for _, mist := range site.Atmosphere {
		if !validLocalBounds(mist.Bounds) {
			return fmt.Errorf("island: invalid mist footprint")
		}
		if mist.EdgeFade > math.Min(mist.Bounds.MaxEast-mist.Bounds.MinEast, mist.Bounds.MaxSouth-mist.Bounds.MinSouth)/2 {
			return fmt.Errorf("island: mist edge fade exceeds its smaller footprint radius")
		}
	}
	return nil
}
