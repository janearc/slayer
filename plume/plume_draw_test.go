//go:build draw

package plume_test

import (
	"math"
	"testing"
	"time"

	"github.com/janearc/slayer/maths"
)

// the eddies roll the soot about its mean and rise with it: the same
// time is the same smoke, another time is other smoke, and over many
// times a slice carries the soot the fire makes.
func TestTheEddiesRollAboutTheMean(t *testing.T) {
	column := mustColumn(t, megawatt)
	at := maths.V(0.4, 12, -0.3)
	once := column.Tick(time.Second).Density(at)
	if again := column.Tick(time.Second).Density(at); again != once {
		t.Errorf("one time, %g then %g", once, again)
	}
	if later := column.Tick(3 * time.Second).Density(at); later == once {
		t.Errorf("a later time is the same smoke, %g", later)
	}
	made := 0.0024 * 1000 / 21
	const height, times = 12.0, 60
	radius := column.Radius(height)
	step := radius / 40
	passed := 0.0
	for moment := range times {
		ticked := column.Tick(time.Duration(moment) * 700 *
			time.Millisecond)
		for x := -4 * radius; x < 4*radius; x += step {
			for z := -4 * radius; z < 4*radius; z += step {
				soot := ticked.Density(maths.V(x, height, z)) / 8.7
				speed := column.Speed(height) *
					math.Exp2(-(x*x+z*z)/(radius*radius))
				passed += soot * speed * step * step
			}
		}
	}
	passed /= times
	t.Logf("over %d times %g g a second pass, the fire makes %g", times,
		passed, made)
	if !(math.Abs(passed-made) <= 0.02*made) {
		t.Errorf("over time %g g a second pass, the fire makes %g",
			passed, made)
	}
}
