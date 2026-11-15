package embers

import (
	"errors"
	"math"
	"time"

	"github.com/janearc/celestial/measure"
)

// A display's aerial shells, by diameter, from Poulton and Kosanke,
// "Fireworks and their hazards", Fire Engineering, 1995-06-01:
//
//   - the lift is "a muzzle velocity of approximately 250 miles per
//     hour" whatever the shell's size: 111.76 m/s;
//   - the time fuse burns "approximately three to six seconds, depending
//     on the size of the shell";
//   - the shell bursts at "roughly 120 feet per inch of shell diameter",
//     36.576 m an inch;
//   - the burst spreads "approximately 90 feet in diameter per inch",
//     a radius of 13.716 m an inch.
//
// The article speaks of shells from under two inches to over twelve,
// most between three and six; DisplayShell makes two to twelve.
//
// Picked, not read anywhere: the fuse is three seconds at three inches
// and under, six at six and over, and in between a second an inch. The
// stars burn 2 s, take 1 s to lose most of their speed to the air (a
// linear drag of 1 s), and are 1 cm across; a shell holds eight stars
// for each square inch of its diameter; the stars burn from 2,000 K
// down to 1,500 K, and the time fuse glows at 1,500 K.
//
// What follows from those: the shell's terminal speed is whatever makes
// it reach the burst height when its fuse burns out, and the stars'
// throw is whatever carries a star the spread's radius in its burn,
// against the drag, before gravity and the shell's own motion.
const (
	displayLift    = 111.76
	displayPerInch = 36.576
	spreadPerInch  = 13.716
	displayGravity = 9.80665
)

// DisplayShell is an aerial shell of a display, inches across, as the
// numbers above make it.
func DisplayShell(inches float64, seed uint64) (Shell, error) {
	if !(inches >= 2 && inches <= 12) {
		return Shell{}, errors.New("embers: a display shell is two to " +
			"twelve inches across")
	}
	fuse := math.Min(6, math.Max(3, inches))
	burn, drag := 2.0, 1.0
	shell := Shell{
		Lift:     displayLift,
		Gravity:  displayGravity,
		Fuse:     seconds(fuse),
		StarDrag: seconds(drag),
		Burn:     seconds(burn),
		Stars:    int(math.Round(8 * inches * inches)),
		Radius:   measure.Metres(0.005),
		Hot:      measure.Kelvin(2000),
		Cool:     measure.Kelvin(1500),
		Throw:    spreadPerInch * inches / (drag * -math.Expm1(-burn/drag)),
		Seed:     seed,
	}
	terminal, err := terminalFor(shell, displayPerInch*inches)
	if err != nil {
		return Shell{}, err
	}
	shell.Terminal = terminal
	return shell, nil
}

// terminalFor is the terminal speed at which the shell is height metres
// up when its fuse burns out: halved and halved again between a speed
// slow enough that the shell never gets there and one fast enough that
// the air hardly matters. More drag always means lower, so there is one
// answer, unless the height is past what the lift reaches with no air
// at all.
func terminalFor(shell Shell, height float64) (float64, error) {
	at := shell.Fuse.Seconds()
	reach := func(terminal float64) float64 {
		shell.Terminal = terminal
		up, _ := shell.rise(at)
		return up
	}
	low, high := 1e-3, 1e6
	if !(reach(high) >= height) {
		return 0, errors.New("embers: the lift cannot carry the shell so " +
			"high by the time its fuse burns out")
	}
	for range 200 {
		middle := math.Sqrt(low * high)
		if reach(middle) < height {
			low = middle
		} else {
			high = middle
		}
	}
	return high, nil
}

// seconds is a duration of so many seconds.
func seconds(s float64) time.Duration {
	return time.Duration(s * float64(time.Second))
}
