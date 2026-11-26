package manufacture

import (
	"testing"
	"time"
)

// Check the small plan before constructing any of its expensive meshes.
func TestSmallShootPlansFinishWithContractingChildren(t *testing.T) {
	for _, row := range []struct {
		style  int
		length float64
	}{{1, .025}, {2, .025}, {1, .2}, {2, .2}} {
		style := row.style
		done := make(chan shootPlan, 1)
		go func() { done <- planShoot(style, row.length, Sow(11)) }()
		var plan shootPlan
		select {
		case plan = <-done:
		case <-time.After(time.Second):
			t.Fatalf("style %d small plan did not finish within one second", style)
		}
		// The picked two-order grammar has at most 1 + 16 + 16*8 axes.
		if len(plan.Axes) < 2 || len(plan.Axes) > 145 {
			t.Fatalf("style %d has %d axes", style, len(plan.Axes))
		}
		for i, axis := range plan.Axes {
			if i == 0 {
				continue
			}
			if axis.Parent < 0 || axis.Parent >= i {
				t.Fatalf("style %d invalid parent %d", style, axis.Parent)
			}
			parent := plan.Axes[axis.Parent]
			if !(axis.Length > 0 && axis.Length < parent.Length && axis.Radius > 0 && axis.Radius < parent.Radius) {
				t.Fatalf("style %d child %d did not contract from its parent", style, i)
			}
		}
	}
}
