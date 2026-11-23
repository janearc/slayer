package medium

import (
	"testing"
	"time"

	"github.com/janearc/slayer/maths"
)

// groveContractGrid is the grid the contracts are drawn in: trees across by
// across on a grid, each a canopy of a radius and a height, spacing apart, on
// flat ground, under mist of an extinction.
type groveContractGrid struct {
	across                              int
	spacing, radius, height, extinction float64
}

// groveContractSized is the grid the contracts are drawn in, from numbers.yaml.
func groveContractSized(t *testing.T) groveContractGrid {
	t.Helper()
	return groveContractGrid{
		across:     pickedCount(t, "grove-contract-across"),
		spacing:    picked(t, "grove-contract-spacing"),
		radius:     picked(t, "grove-contract-canopy-radius"),
		height:     picked(t, "grove-contract-canopy-height"),
		extinction: picked(t, "grove-contract-extinction")}
}

// flatGround is ground at height zero everywhere.
func flatGround(float64, float64) float64 { return nought }

// gridGrove is a grove of a grid of equal canopies under mist of a depth and
// an asymmetry.
func gridGrove(t *testing.T, depth, asymmetry float64) *Grove {
	t.Helper()
	grid := groveContractSized(t)
	var trees []Canopy
	for row := range grid.across {
		for column := range grid.across {
			at := maths.V(float64(row)*grid.spacing, nought,
				float64(column)*grid.spacing)
			trees = append(trees, Canopy{At: at, Radius: grid.radius,
				Height: grid.height})
		}
	}
	weather := GroveWeather{Extinction: grid.extinction, Depth: depth,
		Coverage: unity, Asymmetry: asymmetry}
	grove, err := NewGrove(weather, trees, flatGround)
	if err != nil {
		t.Fatal(err)
	}
	return grove
}

// a query looks at the same trees however deep the mist is: the bucket that
// holds a point is as wide as a canopy, not as deep as the mist, so the count
// of trees a query visits does not depend on Depth, and it is a handful in
// four hundred.
func TestGroveQueryVisitsTheSameTreesAtAnyDepth(t *testing.T) {
	grid := groveContractSized(t)
	at := maths.V(picked(t, "grove-contract-query-row")*grid.spacing,
		picked(t, "grove-contract-height-in-mist"),
		picked(t, "grove-contract-query-column")*grid.spacing)
	shallowMist := picked(t, "grove-contract-shallow-mist")
	deepMist := picked(t, "grove-contract-deep-mist")
	shallow := len(gridGrove(t, shallowMist, nought).nearby(at))
	deep := len(gridGrove(t, deepMist, nought).nearby(at))
	if shallow != deep {
		t.Errorf("a query visits %d trees in mist %v deep and %d in mist %v",
			shallow, shallowMist, deep, deepMist)
	}
	limit := pickedCount(t, "grove-contract-nearby-limit")
	if deep < unity || deep > limit {
		t.Errorf("a query visits %d of %d trees, want from 1 to %d", deep,
			grid.across*grid.across, limit)
	}
}

// the smaller buckets lose no tree: the density is still the mist inside a
// canopy and clear between canopies, at either depth.
func TestGroveBucketsStillFindTheCanopies(t *testing.T) {
	grid := groveContractSized(t)
	height := picked(t, "grove-contract-height-in-mist")
	underRow := picked(t, "grove-contract-under-row") * grid.spacing
	underColumn := picked(t, "grove-contract-under-column") * grid.spacing
	betweenTrees := grid.spacing * halfway
	depths := []float64{picked(t, "grove-contract-shallow-mist"),
		picked(t, "grove-contract-deep-mist")}
	for _, depth := range depths {
		grove := gridGrove(t, depth, nought)
		under := maths.V(underRow, height, underColumn)
		if density := grove.Density(under); density <= nought {
			t.Errorf("depth %v: no mist under a canopy, density %v", depth,
				density)
		}
		between := maths.V(underRow+betweenTrees, height,
			underColumn+betweenTrees)
		if density := grove.Density(between); density != nought {
			t.Errorf("depth %v: mist %v between canopies", depth, density)
		}
	}
}

// lobed is what the GPU asks a medium that scatters more one way than
// another.
type lobed interface{ Asymmetry() float64 }

// a grove says the asymmetry it was given, so the lobe reaches whoever reads
// it, the ticked grove too; one given none says nought.
func TestGroveSaysItsAsymmetry(t *testing.T) {
	mist := picked(t, "grove-contract-shallow-mist")
	lobeSample := picked(t, "grove-contract-lobe-sample")
	grove := gridGrove(t, mist, lobeSample)
	said, ok := any(grove).(lobed)
	if !ok || said.Asymmetry() != lobeSample {
		t.Fatalf("a grove of asymmetry %v says %v, %v", lobeSample, said, ok)
	}
	ticked, ok := grove.Tick(time.Second).(lobed)
	if !ok || ticked.Asymmetry() != lobeSample {
		t.Errorf("a ticked grove says %v, %v", ticked, ok)
	}
	if plain := gridGrove(t, mist, nought); plain.Asymmetry() != nought {
		t.Errorf("a grove of none says %v", plain.Asymmetry())
	}
}
