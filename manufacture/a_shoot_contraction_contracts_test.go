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
		if len(plan.axes) < 2 || len(plan.axes) > 145 {
			t.Fatalf("style %d has %d axes", style, len(plan.axes))
		}
		for i, axis := range plan.axes {
			if i == 0 {
				continue
			}
			if axis.parent < 0 || axis.parent >= i {
				t.Fatalf("style %d invalid parent %d", style, axis.parent)
			}
			parent := plan.axes[axis.parent]
			if !(axis.length > 0 && axis.length < parent.length && axis.radius > 0 && axis.radius < parent.radius) {
				t.Fatalf("style %d child %d did not contract from its parent", style, i)
			}
		}
	}
}
