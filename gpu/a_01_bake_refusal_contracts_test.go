//go:build darwin

package gpu

import (
	"errors"
	"fmt"
	"math"
	"strings"
	"testing"

	"github.com/janearc/slayer/maths"
)

// Host refusals have no samples and cannot have spent GPU time. Retaining
// that distinction matters to the measured profile even when drawing fails.
func TestRejectedFlameBakesHaveNoOutputOrGPUWork(t *testing.T) {
	valid := fireGridFixture(t, 5, 0)
	check := func(t *testing.T, grid boxed, custom bool) {
		t.Helper()
		density, shine, elapsed, err := bakeFlame(grid)
		if err == nil {
			t.Fatal("invalid flame grid accepted")
		}
		if custom {
			var cause unsupported
			if !errors.As(err, &cause) || !strings.Contains(err.Error(), "custom flame grid") {
				t.Fatalf("unsupported field lost its cause: %v", err)
			}
		} else if !strings.Contains(err.Error(), "invalid flame grid dimensions") {
			t.Fatalf("invalid dimensions lost their cause: %v", err)
		}
		if density != nil || shine != nil || elapsed != 0 {
			t.Fatalf("host refusal returned samples or GPU work: density=%d shine=%d elapsed=%v",
				len(density), len(shine), elapsed)
		}
	}
	t.Run("custom emission", func(t *testing.T) {
		grid := valid
		grid.shine = nil
		check(t, grid, true)
	})
	for axis := range 3 {
		for _, cells := range []int{0, mostBoxCells + 1} {
			t.Run(fmt.Sprintf("axis %d cells %d", axis, cells), func(t *testing.T) {
				grid := valid
				grid.cells[axis] = cells
				check(t, grid, false)
			})
		}
	}
}

// A one-cell axis samples its centre. The upper accepted axis length is
// checked in a thin grid so the boundary test needs only a few KiB.
func TestFlameBakeAcceptsItsExactCellEndpoints(t *testing.T) {
	base := fireGridFixture(t, 5, 0)
	var centre maths.Vec
	lit := false
	// Choose a nonempty region using only the CPU field. The one-cell
	// endpoint then exercises emitted data, rather than an empty centre.
	for step := 1; step <= 20; step++ {
		point := maths.V(0, float64(step)/20, 0)
		if base.med.Density(point) > 0 {
			centre, lit = point, true
			break
		}
	}
	if !lit {
		t.Fatal("endpoint fixture has no lit CPU region")
	}
	half := maths.V(0.001, 0.001, 0.001)
	base.low, base.high = centre.Sub(half), centre.Add(half)
	for axis := range 3 {
		for _, cells := range []int{1, mostBoxCells} {
			t.Run(fmt.Sprintf("axis %d cells %d", axis, cells), func(t *testing.T) {
				grid := base
				grid.cells = [3]int{1, 1, 1}
				grid.cells[axis] = cells
				density, shine, elapsed, err := bakeFlame(grid)
				if err != nil || elapsed <= 0 {
					t.Fatalf("valid endpoint was not baked: %v %v", elapsed, err)
				}
				truth, light, _ := grid.asked()
				if len(density) != len(truth) || len(shine) != len(light) {
					t.Fatal("endpoint grid returned the wrong sample count")
				}
				var total, apart, emitted, lightError float64
				for i, want := range truth {
					got := float64(density[i])
					if !(got >= 0) || math.IsInf(got, 0) {
						t.Fatalf("invalid density: %g", got)
					}
					total += float64(want)
					apart += math.Abs(got - float64(want))
					for c := range 3 {
						wantLight := float64(want) * float64(light[3*i+c])
						gotLight := got * float64(shine[3*i+c])
						if !(gotLight >= 0) || math.IsInf(gotLight, 0) {
							t.Fatalf("invalid emission: %g", gotLight)
						}
						emitted += wantLight
						lightError += math.Abs(gotLight - wantLight)
					}
				}
				if !(total > 0 && emitted > 0) || apart/total > 1e-3 || lightError/emitted > 1e-3 {
					t.Fatalf("endpoint CPU/GPU disagreement: density=%g/%g light=%g/%g",
						apart, total, lightError, emitted)
				}
			})
		}
	}
}
