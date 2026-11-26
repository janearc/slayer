package branching

import (
	"errors"
	"fmt"
	"math"

	"github.com/janearc/celestial/measure"
)

// Habit sets where and how large the branches of an ordered scaffold are.
// Start, Top, Thinning, Stray, Shortening and Uneven are unitless shares.
type Habit struct {
	Turn                      measure.Angle
	Start, Top, Thinning      float64
	Fewest                    int
	Stray                     float64
	Low, Lift, Sway           measure.Angle
	Reach, Shortening         float64
	Uneven, Girth, Least, Own float64
}

// ErrHabit marks a malformed branching habit.
var ErrHabit = errors.New("manufacture: no plant has this habit")

// Sound checks all branching choices before an arrangement is made.
func (set Habit) Sound() error {
	finite := func(n float64) bool { return !math.IsNaN(n) && !math.IsInf(n, 0) }
	share := func(n float64) bool { return n >= 0 && n <= 1 }
	positive := func(n float64) bool { return n > 0 && !math.IsInf(n, 0) }
	checks := []struct {
		name string
		good bool
	}{
		{"turn", finite(set.Turn.InRadians())},
		{"start", share(set.Start)},
		{"top", set.Top > set.Start && set.Top <= 1},
		{"thinning", share(set.Thinning)},
		{"fewest", set.Fewest >= 1},
		{"stray", share(set.Stray)},
		{"low", finite(set.Low.InRadians())},
		{"lift", finite(set.Lift.InRadians())},
		{"sway", finite(set.Sway.InRadians())},
		{"reach", positive(set.Reach)},
		{"shortening", share(set.Shortening)},
		{"uneven", set.Uneven >= 0 && set.Uneven < 1},
		{"girth", positive(set.Girth)},
		{"least", set.Least >= 0 && finite(set.Least)},
		{"own", set.Own >= 0 && finite(set.Own)},
	}
	for _, check := range checks {
		if !check.good {
			return fmt.Errorf("%w: its %s", ErrHabit, check.name)
		}
	}
	return nil
}
