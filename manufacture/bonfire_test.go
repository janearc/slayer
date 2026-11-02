package manufacture

import (
	"math"
	"testing"
)

// a bonfire is its row's logs, eleven to fifteen, their feet in its
// ring, each running in toward the middle and up, their tops between
// five and a half and eight and a half metres high; and it gives no light
// of its own, so the only light on it is the fire a set lights.
func TestABonfireIsLogsLeanedTogether(t *testing.T) {
	row := pyreRow()
	for number := range uint64(20) {
		made := mustMake(t, "bonfire", number)
		if len(made.lights) != 0 {
			t.Fatalf("seed %d: %d lights of its own", number,
				len(made.lights))
		}
		if !row.logs.holds(float64(len(made.parts))) {
			t.Errorf("seed %d: %d logs", number, len(made.parts))
		}
		for _, one := range made.parts {
			foot := one.space.Origin
			out := math.Hypot(foot.X, foot.Z)
			way := one.space.Orientation.Y
			inward := -(way.X*foot.X + way.Z*foot.Z)
			if !row.ring.holds(out) || inward <= 0 || way.Y <= 0 {
				t.Errorf("seed %d: a log at %v running %v", number, foot,
					way)
			}
			top := foot.Y + row.log.long.most*way.Y
			if top < 5.5 || top > 8.5 {
				t.Errorf("seed %d: a log's top at %g m", number, top)
			}
		}
	}
}
