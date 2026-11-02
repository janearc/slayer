package manufacture

import (
	"testing"

	"github.com/janearc/slayer/geom"
)

func TestProperBranchCountFalls(t *testing.T) {
	params := DefaultProperConiferParams()

	prevCount := BranchCount(params, 0)
	for whorl := 1; whorl < params.Whorls; whorl++ {
		count := BranchCount(params, whorl)
		if count > prevCount {
			t.Errorf("BranchCount should decrease with height: whorl %d has %d, previous was %d",
				whorl, count, prevCount)
		}
		prevCount = count
	}
}

func TestProperBranchLengthTrend(t *testing.T) {
	params := DefaultProperConiferParams()

	firstLength := BranchOf(params, 0, 0).Length
	lastLength := BranchOf(params, params.Whorls-1, 0).Length

	if firstLength <= lastLength {
		t.Errorf("Base branches should be longer than top: first %.2f <= last %.2f",
			firstLength, lastLength)
	}
}

func TestProperSeedDeterminism(t *testing.T) {
	params := DefaultProperConiferParams()
	params.Seed = 42

	b1 := BranchOf(params, 5, 3)
	b2 := BranchOf(params, 5, 3)

	if b1.Base != b2.Base || b1.Dir != b2.Dir || b1.Length != b2.Length {
		t.Errorf("Same seed should produce identical branches")
	}
}

func TestProperSeedVariation(t *testing.T) {
	params1 := DefaultProperConiferParams()
	params1.Seed = 1

	params2 := DefaultProperConiferParams()
	params2.Seed = 2

	b1 := BranchOf(params1, 5, 3)
	b2 := BranchOf(params2, 5, 3)

	if b1.Dir == b2.Dir {
		t.Errorf("Different seeds should produce different branches")
	}
}

func TestProperWhorlsDistributed(t *testing.T) {
	params := DefaultProperConiferParams()

	for whorl := 0; whorl < params.Whorls; whorl++ {
		count := BranchCount(params, whorl)
		if count < 3 {
			t.Errorf("Each whorl should have at least 3 branches, whorl %d has %d",
				whorl, count)
		}
	}
}

func TestProperConiferProperGeneration(t *testing.T) {
	params := DefaultProperConiferParams()
	conifer := ConiferProper(geom.Vec3{0, 0, 0}, params)

	if len(conifer) == 0 {
		t.Errorf("ConiferProper should generate triangles")
	}

	expectedBranches := 0
	for whorl := 0; whorl < params.Whorls; whorl++ {
		expectedBranches += BranchCount(params, whorl)
	}

	minTriangles := (1 + expectedBranches*3) * 2
	if len(conifer) < minTriangles {
		t.Logf("Generated %d triangles for %d branches (expected ~%d)",
			len(conifer), expectedBranches, minTriangles)
	}
}

func TestProperConiferBounds(t *testing.T) {
	params := DefaultProperConiferParams()
	conifer := ConiferProper(geom.Vec3{0, 0, 0}, params)
	bounds := geom.BoundsOfTriangles(conifer)

	if bounds.Max[2] < params.Height*0.9 {
		t.Errorf("Conifer should reach near its full height")
	}
}

func TestProperConiferPosition(t *testing.T) {
	params := DefaultProperConiferParams()
	pos := geom.Vec3{5, 10, 1}
	conifer := ConiferProper(pos, params)
	bounds := geom.BoundsOfTriangles(conifer)

	if bounds.Min[2] < pos[2]-2 {
		t.Errorf("Conifer should be positioned near base height")
	}
}

func BenchmarkBranchOf(b *testing.B) {
	params := DefaultProperConiferParams()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		BranchOf(params, 5, 3)
	}
}

func BenchmarkConiferProper(b *testing.B) {
	params := DefaultProperConiferParams()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ConiferProper(geom.Vec3{0, 0, 0}, params)
	}
}

func TestProperUnitFunction(t *testing.T) {
	seed := uint64(42)
	val1 := unit(seed, 0, 0, 0)
	val2 := unit(seed, 0, 0, 0)

	if val1 != val2 {
		t.Errorf("unit() should be deterministic")
	}

	if val1 < 0 || val1 > 1 {
		t.Errorf("unit() should return [0,1], got %.3f", val1)
	}
}

func TestProperBranchOrientation(t *testing.T) {
	params := DefaultProperConiferParams()

	lowZ := BranchOf(params, 0, 0).Dir[2]
	highZ := BranchOf(params, params.Whorls-1, 0).Dir[2]

	if lowZ >= highZ {
		t.Errorf("Bottom branches should droop lower than top branches: low z=%.3f, high z=%.3f",
			lowZ, highZ)
	}

	if highZ <= 0.2 {
		t.Errorf("Top whorls should point somewhat upward, got z=%.3f", highZ)
	}
}

func TestProperBranchPieceChaining(t *testing.T) {
	params := DefaultProperConiferParams()
	branch := BranchOf(params, 5, 0)

	const pieces = 3
	step := branch.Length / pieces

	at := branch.Base
	dir := branch.Dir

	for piece := 0; piece < pieces; piece++ {
		next := at.Add(dir.Scale(step))

		dir = dir.Add(geom.Vec3{0, 0, -0.22}).Normalize()

		if next[0] == at[0] && next[1] == at[1] && next[2] == at[2] {
			t.Errorf("piece %d: next position equals current position", piece)
		}

		at = next
	}
}
