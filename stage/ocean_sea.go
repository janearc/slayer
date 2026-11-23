package stage

import (
	"fmt"

	"github.com/janearc/slayer/arid"
	"github.com/janearc/slayer/manufacture"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/oceantex"
	"github.com/janearc/slayer/scene/thing"
	"github.com/janearc/slayer/set"
)

// SeaOf is the oceantex sea a set's ocean says: its wind, fetch, depth
// and direction, and its swell where it says one. It refuses a swell that
// oceantex refuses.
func SeaOf(surface set.WaveSurface) (oceantex.Sea, error) {
	wind, fetch, depth, toward := surface.Sea()
	sea := oceantex.Sea{Wind: wind, Fetch: fetch, Depth: depth,
		Toward: toward}
	said, has := surface.Swell()
	if !has {
		return sea, nil
	}
	swell, err := oceantex.NewSwell(said.HeightMetres(), said.PeriodSeconds(),
		said.TowardRadians())
	if err != nil {
		return oceantex.Sea{}, fmt.Errorf("swell: %w", err)
	}
	sea.Swell = swell
	return sea, nil
}

// openSea is the manufactory's sea with no edge: the wave bands of the
// set's sea at its time and seed, summed at every point of the plane,
// under the material the set names. Its mean level is the ocean's own
// `middle` height.
//
// It carries the mean-plane direct sun where the set says so, and no other
// optical approximation, which each need a square or a plane to stand on;
// the set refuses their keys beside an ocean that says no extent.
//
// The ocean's extent, which the sun's path reads, is the longest band's
// side, the distance after which the sea repeats; staging puts it in.
func openSea(said set.World, surface set.WaveSurface, shift maths.Vec,
	scale float64) (manufacture.Sea, error) {
	model, err := SeaOf(surface)
	if err != nil {
		return manufacture.Sea{}, fmt.Errorf("ocean: %w", err)
	}
	return manufacture.Sea{Model: model, Seconds: surface.Seconds(),
		Seed: surface.Seed(), At: seaAt(surface, shift, scale),
		Optics: thing.OceanSurface{MetresPerUnit: said.Unit(),
			DirectSunMeanPlane: surface.DirectSunMode() == "mean-plane",
			SunLampIndex:       -1, FillLampIndex: -1}}, nil
}

// seaGlazed is the material the sea wears: glazed, with a cox-munk glint
// taken from the slope the sea's own bands leave unresolved, not from a
// sheet's spacing, of which a sea with no edge has none. A set that says
// no glint gets Cox and Munk's; only an artistic glint, a look chosen by
// eye, is taken as authored.
func seaGlazed(worn *thing.Material, surface set.WaveSurface,
	sea *arid.Sea) (*thing.Material, error) {
	if surface.Glint().Model() == set.GlintArtistic {
		return glazed(worn, surface)
	}
	model, err := SeaOf(surface)
	if err != nil {
		return nil, err
	}
	budget, err := sea.GlintBudget(model)
	if err != nil {
		return nil, err
	}
	own := *worn
	own.SlopeVariance = budget.Unresolved
	return &own, nil
}
