package biome_test

import (
	"math"
	"reflect"
	"testing"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/scene/biome"
)

// definition is a valid one-species region that a test then spoils.
func definition() biome.Definition {
	return biome.Definition{Name: "gaps", Seed: 7,
		Bounds: biome.Rect{MaxEast: 200, MaxSouth: 200}, Cell: measure.Metres(100),
		PerHectare: 1, Species: []biome.Species{{Recipe: "oak", Weight: 1}}}
}

// TestDefaultBudgetReadsThePolicyFile checks the work limits against the
// values the policy file states.
func TestDefaultBudgetReadsThePolicyFile(t *testing.T) {
	got, err := biome.DefaultBudget()
	if err != nil {
		t.Fatal(err)
	}
	if want := (biome.Budget{Cells: 65536, Candidates: 65536}); got != want {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

// TestNewRefusesBadDefinitions checks each way a definition is refused.
func TestNewRefusesBadDefinitions(t *testing.T) {
	cases := map[string]func(*biome.Definition){
		"blank name":       func(d *biome.Definition) { d.Name = "  " },
		"empty bounds":     func(d *biome.Definition) { d.Bounds.MaxEast = d.Bounds.MinEast },
		"infinite bounds":  func(d *biome.Definition) { d.Bounds.MaxSouth = math.Inf(1) },
		"zero cell":        func(d *biome.Definition) { d.Cell = measure.Metres(0) },
		"negative cell":    func(d *biome.Definition) { d.Cell = measure.Metres(-5) },
		"not a number":     func(d *biome.Definition) { d.Cell = measure.Metres(math.NaN()) },
		"negative density": func(d *biome.Definition) { d.PerHectare = -1 },
		"density not set":  func(d *biome.Definition) { d.PerHectare = math.NaN() },
		"density overflow": func(d *biome.Definition) { d.PerHectare = math.MaxFloat64 },
		"no species":       func(d *biome.Definition) { d.Species = nil },
		"blank recipe":     func(d *biome.Definition) { d.Species = []biome.Species{{Recipe: " ", Weight: 1}} },
		"zero weight":      func(d *biome.Definition) { d.Species = []biome.Species{{Recipe: "oak"}} },
		"nan weight":       func(d *biome.Definition) { d.Species = []biome.Species{{Recipe: "oak", Weight: math.NaN()}} },
		"duplicate recipe": func(d *biome.Definition) {
			d.Species = []biome.Species{{Recipe: "oak", Weight: 1}, {Recipe: "oak", Weight: 2}}
		},
		"weights overflow": func(d *biome.Definition) {
			d.Species = []biome.Species{{Recipe: "oak", Weight: math.MaxFloat64}, {Recipe: "ash", Weight: math.MaxFloat64}}
		},
	}
	for name, spoil := range cases {
		d := definition()
		spoil(&d)
		if got, err := biome.New(d); err == nil || got != nil {
			t.Errorf("%s: accepted", name)
		}
	}
	if _, err := biome.New(definition()); err != nil {
		t.Errorf("the unspoiled definition: %v", err)
	}
}

// TestSelectRefusesBadQueries checks the guards of Select: a missing region,
// a query that is empty or not finite, and a negative allowance.
func TestSelectRefusesBadQueries(t *testing.T) {
	population := region(t, 1)
	var none *biome.Region
	good := biome.Rect{MaxEast: 100, MaxSouth: 100}
	cases := map[string]func() (biome.Selection, error){
		"nil region": func() (biome.Selection, error) {
			return none.SelectWithUsage(good, biome.Budget{Cells: 99, Candidates: 99})
		},
		"empty query": func() (biome.Selection, error) {
			return population.SelectWithUsage(biome.Rect{}, biome.Budget{Cells: 99, Candidates: 99})
		},
		"infinite query": func() (biome.Selection, error) {
			return population.SelectWithUsage(biome.Rect{MaxEast: math.Inf(1), MaxSouth: 1}, biome.Budget{Cells: 99, Candidates: 99})
		},
		"negative cells": func() (biome.Selection, error) {
			return population.SelectWithUsage(good, biome.Budget{Cells: -1, Candidates: 99})
		},
		"negative candidates": func() (biome.Selection, error) {
			return population.SelectWithUsage(good, biome.Budget{Cells: 99, Candidates: -1})
		},
	}
	for name, call := range cases {
		if got, err := call(); err == nil || !reflect.DeepEqual(got, biome.Selection{}) {
			t.Errorf("%s: %+v, %v", name, got, err)
		}
	}
}

// TestSelectReturnsTheSameRootsAsSelectWithUsage checks that Select is the
// roots of SelectWithUsage, and returns none with its error.
func TestSelectReturnsTheSameRootsAsSelectWithUsage(t *testing.T) {
	population := region(t, 1)
	budget := biome.Budget{Cells: haloWork, Candidates: haloWork}
	roots, err := population.Select(population.Bounds(), budget)
	if err != nil || len(roots) != rootCount {
		t.Fatalf("got %d roots, %v; want %d", len(roots), err, rootCount)
	}
	if want := selection(t, population, population.Bounds()).Roots; !reflect.DeepEqual(roots, want) {
		t.Error("Select and SelectWithUsage disagree")
	}
	roots, err = population.Select(population.Bounds(), biome.Budget{})
	if err == nil || roots != nil {
		t.Errorf("over budget: %v, %v", roots, err)
	}
}

// TestFractionalMeanRoundsRandomly checks that a mean of one and a half per
// cell gives one or two candidates a cell, one and a half on average.
func TestFractionalMeanRoundsRandomly(t *testing.T) {
	d := definition()
	d.Bounds = biome.Rect{MaxEast: 4000, MaxSouth: 4000}
	d.PerHectare = 1.5
	population, err := biome.New(d)
	if err != nil {
		t.Fatal(err)
	}
	got, err := population.SelectWithUsage(d.Bounds, biome.Budget{Cells: 1 << 20, Candidates: 1 << 20})
	if err != nil {
		t.Fatal(err)
	}
	ratio := float64(got.Generated) / float64(got.Cells)
	if got.Generated <= got.Cells || got.Generated >= 2*got.Cells || math.Abs(ratio-1.5) > 0.1 {
		t.Errorf("%d candidates in %d cells, %.3f a cell", got.Generated, got.Cells, ratio)
	}
}

// TestFootprintBeyondTheLatticeIsRefused checks that a footprint too large
// to number its cells exactly is refused along either axis.
func TestFootprintBeyondTheLatticeIsRefused(t *testing.T) {
	big := biome.Budget{Cells: math.MaxInt, Candidates: math.MaxInt}
	for name, bounds := range map[string]biome.Rect{
		"east":  {MaxEast: 1e300, MaxSouth: 100},
		"south": {MaxEast: 100, MaxSouth: 1e300},
	} {
		d := definition()
		d.Cell = measure.Metres(1)
		d.Bounds = bounds
		population, err := biome.New(d)
		if err != nil {
			t.Fatal(err)
		}
		if got, err := population.SelectWithUsage(bounds, big); err == nil || !reflect.DeepEqual(got, biome.Selection{}) {
			t.Errorf("%s: %+v, %v", name, got, err)
		}
	}
}

// TestRootsStayInsideTheirCellFarFromTheOrigin checks that where the spacing
// of numbers is wider than the draw's resolution, no root is rounded onto the
// cell's far edge, which belongs to the next cell.
func TestRootsStayInsideTheirCellFarFromTheOrigin(t *testing.T) {
	const origin = 1 << 50
	d := definition()
	d.Cell = measure.Metres(1)
	d.PerHectare = 10000
	d.Bounds = biome.Rect{MinEast: origin, MaxEast: origin + 64, MinSouth: 0, MaxSouth: 64}
	population, err := biome.New(d)
	if err != nil {
		t.Fatal(err)
	}
	got, err := population.SelectWithUsage(d.Bounds, biome.Budget{Cells: 1 << 20, Candidates: 1 << 20})
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Roots) < 1000 {
		t.Fatalf("only %d roots", len(got.Roots))
	}
	onEdge := 0
	for _, root := range got.Roots {
		low := float64(root.ID.Column)
		if root.East < low || root.East >= low+1 {
			t.Fatalf("root of cell %d at %v is outside it", root.ID.Column, root.East)
		}
		if root.East == math.Nextafter(low+1, low) {
			onEdge++
		}
	}
	if onEdge == 0 {
		t.Error("no root was held just inside the far edge")
	}
}
