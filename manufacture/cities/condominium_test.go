package cities

import (
	"math"
	"testing"
)

// leastCore and mostCore are how tall a stair core may be, in metres:
// five storeys of the least height the culture draws, and six of the
// most.
const (
	leastCore = 5 * 2.8
	mostCore  = 6 * 3.0
)

// drawnCondominium is a condominium of a seed drawn at the quick level,
// or the test's end.
func drawnCondominium(t *testing.T, seed Seed) []Rendered {
	t.Helper()
	design, err := Condominium(seed)
	if err != nil {
		t.Fatal(err)
	}
	drawn, err := design.Rendered("quick")
	if err != nil {
		t.Fatal(err)
	}
	return drawn
}

// stairCores are the drawn walls that are a stair core's body: a box as
// wide as the culture's core, in the mesh's own space.
func stairCores(t *testing.T, drawn []Rendered) []Rendered {
	t.Helper()
	width := mustCultureFixed(t, "core-width")
	var cores []Rendered
	for _, one := range drawn {
		if one.Mesh.Name != "wall" {
			continue
		}
		if low, high := meshReach(one); math.Abs(high[0]-low[0]-width) <
			tolerance && high[1]-low[1] > 3*width {
			cores = append(cores, one)
		}
	}
	return cores
}

// mustCultureFixed is a fixed number of the condominium's culture.
func mustCultureFixed(t *testing.T, name string) float64 {
	t.Helper()
	value := drawerOf(t, condominiumCulture, 1).fixed(name)
	return value
}

// meshReach is the corners of a drawn part's mesh, in its own space.
func meshReach(one Rendered) (low, high [3]float64) {
	low = [3]float64{math.Inf(1), math.Inf(1), math.Inf(1)}
	high = [3]float64{math.Inf(-1), math.Inf(-1), math.Inf(-1)}
	for _, triangle := range one.Mesh.Triangles {
		for _, corner := range triangle {
			for axis, value := range [3]float64{corner.X, corner.Y,
				corner.Z} {
				low[axis] = math.Min(low[axis], value)
				high[axis] = math.Max(high[axis], value)
			}
		}
	}
	return low, high
}

func TestACondominiumIsTwoToFourBuildingsEachWithAStairCore(t *testing.T) {
	counts := map[int]bool{}
	district := Seed(drawerOf(t, condominiumCulture, 1).district)
	for index := range Seed(seeds) {
		seed := index * district
		drawing := drawerOf(t, condominiumCulture, seed)
		want := drawing.count("buildings", place{})
		counts[want] = true
		drawn := drawnCondominium(t, seed)
		if got := len(stairCores(t, drawn)); got !=
			want || want < 2 || want > 4 {
			t.Errorf("seed %d: %d buildings and %d stair cores", seed,
				want, got)
		}
	}
	if len(counts) < 2 {
		t.Errorf("the building count never varied: %v", counts)
	}
}

func TestEveryBuildingOfACondominiumIsFourOrFiveStoreys(t *testing.T) {
	for seed := range Seed(seeds) {
		drawing := drawerOf(t, condominiumCulture, seed)
		for bay := range drawing.count("buildings", place{}) {
			at := place{Bay: bay}
			storeys := drawing.count("storeys", at)
			if storeys < 4 || storeys > 5 {
				t.Errorf("seed %d: building %d is %d storeys", seed, bay,
					storeys)
			}
		}
	}
}

func TestAStairCoreStandsOneStoreyOverItsBuildingAndProudOfIt(t *testing.T) {
	drawing := drawerOf(t, condominiumCulture, 5)
	proud := drawing.fixed("core-proud")
	for _, core := range stairCores(t, drawnCondominium(t, 5)) {
		low, high := meshReach(core)
		if height := high[1] - low[1]; height < leastCore ||
			height > mostCore {
			t.Errorf("a core %v tall, want %v to %v: one storey over a "+
				"building of four or five", height, leastCore, mostCore)
		}
		if depth := high[2] - low[2]; depth < proud {
			t.Errorf("a core %v deep does not stand %v proud", depth, proud)
		}
	}
}

func TestACondominiumStandsRoundACourtItShares(t *testing.T) {
	drawing := drawerOf(t, condominiumCulture, 9)
	width := drawing.number("court-width", place{})
	var paving *Rendered
	for _, one := range drawnCondominium(t, 9) {
		if one.Mesh.Name == "court" {
			paving = &one
		}
	}
	if paving == nil {
		t.Fatal("no court was paved")
	}
	low, high := reach(*paving)
	if math.Abs(high.X-low.X-width) > tolerance || low.Y < 0 {
		t.Errorf("the court spans %v to %v, want %v wide", low, high, width)
	}
}

func TestACondominiumIsTheSameForTheSameSeed(t *testing.T) {
	first := digest(drawnCondominium(t, 6))
	again := digest(drawnCondominium(t, 6))
	if first != again {
		t.Error("one seed made two condominiums")
	}
	if first == digest(drawnCondominium(t, 7)) {
		t.Error("two seeds made the same condominium")
	}
}
