package coral

import (
	"github.com/janearc/slayer/maths"
	"math"
	"reflect"
	"testing"
)

func TestCoralliteReliefWrapsAndChangesTheBranchSkin(t *testing.T) {
	const radius = .02
	for _, height := range []float64{0, .013, .041} {
		left := coralliteRelief(0, height, radius)
		right := coralliteRelief(2*math.Pi, height, radius)
		if math.Abs(left-right) > 1e-12 {
			t.Fatalf("surface does not wrap at height %g: %g != %g", height, left, right)
		}
	}
	around := math.Round(2 * math.Pi * radius / .008)
	if coralliteRelief(0, 0, radius) >= coralliteRelief(.31*2*math.Pi/around, 0, radius) {
		t.Fatal("expected the cup centre to sit below its rim")
	}
	plain := Skin[[3]maths.Vec](Branch{length: .14, radius: radius, seed: 7})
	detailed := Skin[[3]maths.Vec](Branch{length: .14, radius: radius, seed: 7, corallites: true})
	if len(detailed) <= len(plain) {
		t.Fatal("the detailed surface needs enough faces to resolve its cups")
	}
}

type traceDraws struct{ calls [][3]int }

func (draws *traceDraws) Draw(row, index, which int) float64 {
	draws.calls = append(draws.calls, [3]int{row, index, which})
	return .5
}
func (draws *traceDraws) Swing(row, index, which int) float64 {
	return draws.Draw(row, index, which)
}

func TestGrowKeepsDepthFirstSeedDraws(t *testing.T) {
	spec := Default()
	spec.Levels = 2
	draws := &traceDraws{}
	parts, err := spec.Grow(draws, 14)
	if err != nil {
		t.Fatal(err)
	}
	want := [][3]int{
		{0, 1, 2}, {0, 1, 0},
		{1, 3, 2}, {1, 3, 0}, {1, 3, 1}, {1, 3, 2},
		{0, 1, 1},
		{1, 4, 2}, {1, 4, 0}, {1, 4, 1}, {1, 4, 2},
		{0, 1, 2},
		{1, 5, 2}, {1, 5, 0}, {1, 5, 1}, {1, 5, 2},
	}
	if len(parts) != 13 || !reflect.DeepEqual(draws.calls, want) {
		t.Fatalf("got %d branches and draws %v", len(parts), draws.calls)
	}
}

func TestGrowRejectsMissingSeedAndInvalidSpec(t *testing.T) {
	if _, err := Default().Grow(nil, 1); err == nil {
		t.Fatal("accepted a missing seed")
	}
	if _, err := (Spec{}).Grow(&traceDraws{}, 1); err == nil {
		t.Fatal("accepted an invalid specification")
	}
}
