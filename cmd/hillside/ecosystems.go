package main

import (
	"fmt"
	"math"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/biome"
	"github.com/janearc/slayer/set"
)

// ecosystems queries fixed source regions through the current terrain window.
// Candidate identity is crop-independent; ground placement and filtering are
// not.
func (stage *staging) ecosystems(read set.Set) error {
	if len(read.Ecosystems()) == 0 {
		return nil
	}
	budget, err := biome.DefaultBudget()
	if err != nil {
		return err
	}
	across, down, step := stage.ground.Size()
	query := biome.Rect{MinEast: stage.west, MinSouth: -stage.north,
		MaxEast:  stage.west + float64(across-1)*step,
		MaxSouth: -stage.north + float64(down-1)*step}
	for _, request := range read.Ecosystems() {
		if err := stage.ecosystem(request, query, &budget); err != nil {
			return fmt.Errorf("ecosystem %s: %w", request.Name(), err)
		}
	}
	return nil
}

// ecosystem places a mixture without creating a mesh for each population
// member.
func (stage *staging) ecosystem(request set.Ecosystem, query biome.Rect,
	budget *biome.Budget,
) error {
	population, err := request.Population()
	if err != nil {
		return err
	}
	selection, err := population.SelectWithUsage(query, *budget)
	if err != nil {
		return err
	}
	budget.Cells -= selection.Cells
	budget.Candidates -= selection.Generated
	var canopies []groveTree
	for _, candidate := range selection.Roots {
		canopy, placed, err := stage.ecosystemCandidate(request, candidate)
		if err != nil {
			return err
		}
		if placed {
			canopies = append(canopies, canopy)
		}
	}
	weather, fade := request.Atmosphere()
	return stage.populationMist(weather, fade, canopies)
}

// ecosystemCandidate samples actual triangulated ground before habitat tests.
func (stage *staging) ecosystemCandidate(request set.Ecosystem,
	candidate biome.Candidate,
) (groveTree, bool, error) {
	x, z := candidate.East-stage.west, stage.north+candidate.South
	place := maths.V(x, stage.groundAt(x, z), z)
	above, below := request.Band()
	height := place.Y + stage.lowest
	if height <= above || height >= below {
		return groveTree{}, false, nil
	}
	row, found := stage.recipes.Recipe(candidate.Recipe)
	if !found {
		return groveTree{}, false, fmt.Errorf("unknown mixture recipe %q", candidate.Recipe)
	}
	seed := stir(request.Seed(), int(candidate.Seed%uint64(request.Variety())))
	one, err := stage.ecosystemStock(row.Kind(), candidate.Recipe, request.Complexity(), seed)
	if err != nil {
		return groveTree{}, false, err
	}
	allowed, err := stage.ecosystemClearance(request, place, one.bounds)
	if err != nil || !allowed {
		return groveTree{}, false, err
	}
	pose := maths.NewSubspace(place, maths.Rotation(maths.YAxis, candidate.Turn*2*math.Pi), 1)
	if err := stage.add(one, row.Kind(), pose, candidate.Seed, false); err != nil {
		return groveTree{}, false, err
	}
	stage.note(request.Name(), row.Kind(), place)
	return groveTree{at: place, bounds: one.bounds}, true, nil
}

// ecosystemStock shares a finite variety per recipe instead of per-root
// geometry.
func (stage *staging) ecosystemStock(kind, recipe, detail string, seed uint64) (kept, error) {
	key := stocked{name: "recipe:" + recipe, pick: int(seed), detail: detail}
	if known, found := stage.have[key]; found {
		return known, nil
	}
	made, err := stage.constructed(kind, recipe, detail, seed)
	if err != nil {
		return kept{}, err
	}
	if made.stored > stage.given.faces-stage.made.kept {
		return kept{}, fmt.Errorf("%w: ecosystem geometry exceeds -faces", errTooMany)
	}
	stage.have[key] = *made
	stage.made.kept += made.stored
	return *made, nil
}

// ecosystemClearance includes both the excluded object and candidate canopy.
func (stage *staging) ecosystemClearance(request set.Ecosystem,
	place maths.Vec, bounds maths.Box,
) (bool, error) {
	name, clearance := request.Exclude()
	if name == "" {
		return true, nil
	}
	centre, found := stage.places[name]
	if !found {
		return false, fmt.Errorf("exclusion %s has no placed position", name)
	}
	radius := stage.radii[name] + horizontalRadius(bounds) + clearance
	return math.Hypot(place.X-centre.X, place.Z-centre.Z) >= radius, nil
}
