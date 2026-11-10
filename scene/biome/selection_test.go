package biome_test

import (
	"math"
	"reflect"
	"strings"
	"testing"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/scene/biome"
)

func TestDensityRefusesAnUnrepresentablePerCellOrdinal(test *testing.T) {
	definition := biome.Definition{Name: "ordinal boundary", Seed: 1,
		Bounds: biome.Rect{MaxEast: 100, MaxSouth: 100}, Cell: measure.Metres(100),
		PerHectare: math.MaxUint32 - 1,
		Species:    []biome.Species{{Recipe: "brush", Weight: 1}}}
	if _, err := biome.New(definition); err != nil {
		test.Fatal("last supported integral cell mean refused", err)
	}
	definition.PerHectare = math.MaxUint32
	if _, err := biome.New(definition); err == nil {
		test.Fatal("ordinal range can overflow after randomized rounding")
	}
}

// Each 100 m cell has exactly one candidate at one candidate per hectare.
// A two-by-two-cell region therefore owns four roots. The query halo examines
// four-by-four cells, charging sixteen candidates including twelve rejected.
const (
	cellMetres = 100
	regionSide = 2 * cellMetres
	rootCount  = 4
	haloWork   = 16
)

func region(test *testing.T, density float64) *biome.Region {
	test.Helper()
	value, err := biome.New(biome.Definition{Name: "overlap", Seed: 1,
		Bounds: biome.Rect{MaxEast: regionSide, MaxSouth: regionSide},
		Cell:   measure.Metres(cellMetres), PerHectare: density,
		Species: []biome.Species{{Recipe: "oak", Weight: 1}, {Recipe: "brush", Weight: 3}}})
	if err != nil {
		test.Fatal(err)
	}
	return value
}

func selection(test *testing.T, population *biome.Region,
	query biome.Rect) biome.Selection {
	test.Helper()
	got, err := population.SelectWithUsage(query,
		biome.Budget{Cells: haloWork, Candidates: haloWork})
	if err != nil {
		test.Fatal(err)
	}
	return got
}

func byID(test *testing.T, roots []biome.Candidate) map[biome.ID]biome.Candidate {
	test.Helper()
	found := make(map[biome.ID]biome.Candidate, len(roots))
	for _, root := range roots {
		if _, duplicate := found[root.ID]; duplicate {
			test.Fatal("duplicate spatial identity", root.ID)
		}
		found[root.ID] = root
	}
	return found
}

func TestQueriesPartitionRootsWithoutLosingOrDuplicatingTheSeam(test *testing.T) {
	population := region(test, 1)
	whole := selection(test, population, population.Bounds())
	if len(whole.Roots) != rootCount || whole.Cells != haloWork || whole.Generated != haloWork {
		test.Fatal("region or halo accounting changed", whole)
	}
	left := biome.Rect{MaxEast: cellMetres, MaxSouth: regionSide}
	right := biome.Rect{MinEast: cellMetres, MaxEast: regionSide, MaxSouth: regionSide}
	// Enumerate the right crop first; identities must not follow query order.
	second := selection(test, population, right)
	first := selection(test, population, left)
	joined := append(second.Roots, first.Roots...)
	if !reflect.DeepEqual(byID(test, joined), byID(test, whole.Roots)) {
		test.Fatal("adjoining queries changed candidate identity, seed, form or position")
	}
}

func TestOffsetCropKeepsEveryOverlappingCandidateValue(test *testing.T) {
	population := region(test, 1)
	whole := selection(test, population, population.Bounds())
	crop := biome.Rect{MinEast: cellMetres / 2, MaxEast: regionSide,
		MaxSouth: regionSide}
	want := map[biome.ID]biome.Candidate{}
	for _, root := range whole.Roots {
		if crop.Contains(root.East, root.South) {
			want[root.ID] = root
		}
	}
	got := selection(test, population, crop)
	if !reflect.DeepEqual(byID(test, got.Roots), want) {
		test.Fatal("offset crop reseeded or changed an overlapping candidate")
	}
}

func TestRejectedCandidatesStillConsumeTheSharedAllowance(test *testing.T) {
	population := region(test, 1)
	first := selection(test, population, population.Bounds())
	if first.Generated <= len(first.Roots) {
		test.Fatal("fixture must include query-edge rejects")
	}
	left := biome.Budget{Cells: 2*haloWork - 1, Candidates: 2*haloWork - 1}
	left.Cells -= first.Cells
	left.Candidates -= first.Generated
	got, err := population.SelectWithUsage(population.Bounds(), left)
	if err == nil || len(got.Roots) != 0 {
		test.Fatal("a second individually affordable region exceeded the shared allowance", got, err)
	}
}

func TestCellAndCandidateRefusalsReturnNoPartialSelection(test *testing.T) {
	population := region(test, 1)
	for _, allowance := range []biome.Budget{
		{Cells: haloWork - 1, Candidates: haloWork},
		{Cells: haloWork, Candidates: haloWork - 1},
	} {
		got, err := population.SelectWithUsage(population.Bounds(), allowance)
		if err == nil || !reflect.DeepEqual(got, biome.Selection{}) {
			test.Fatal("budget failure published partial roots or usage", got, err)
		}
	}
}

func TestAnEmptyRegionCostsNothingAfterTheAllowanceIsSpent(test *testing.T) {
	for _, population := range []*biome.Region{region(test, 0), region(test, 1)} {
		query := biome.Rect{MinEast: regionSide, MaxEast: 2 * regionSide,
			MaxSouth: regionSide}
		got, err := population.SelectWithUsage(query, biome.Budget{})
		if err != nil || !reflect.DeepEqual(got, biome.Selection{}) {
			test.Fatal("empty intersection consumed an exhausted allowance", got, err)
		}
	}
	empty := region(test, 0)
	got, err := empty.SelectWithUsage(empty.Bounds(), biome.Budget{})
	if err != nil || got.Cells != 0 || got.Generated != 0 {
		test.Fatal("zero density should require no cell traversal", got, err)
	}
}

func TestAnExhaustedNonemptyQueryNamesItsWorkBudget(test *testing.T) {
	population := region(test, 1)
	_, err := population.SelectWithUsage(population.Bounds(), biome.Budget{})
	if err == nil || !strings.Contains(err.Error(), "0-cell work budget") {
		test.Fatal("exhaustion was not distinguished from malformed input", err)
	}
}

func TestOrientationDoesNotReuseTheGeometrySeedBits(test *testing.T) {
	population := region(test, 1)
	roots := selection(test, population, population.Bounds()).Roots
	for _, root := range roots {
		if root.Turn == float64(root.Seed>>11)*0x1p-53 {
			test.Fatal("orientation reused the seed's top bits", root.ID)
		}
	}
}
