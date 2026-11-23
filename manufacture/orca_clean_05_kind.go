package manufacture

import (
	"fmt"
	"math"
	"time"

	"github.com/janearc/slayer/maths"
)

// orcaClean05Kind exposes the approved original as a timed skin study.
// The three-second cycle is authored; it is not measured swimming.
type orcaClean05Kind struct{}

// made is the orca at phase nought of its swim. It refuses a seed that was
// not made.
func (orcaClean05Kind) made(from seed) (good, error) {
	if !from.made() {
		return good{}, errGood
	}
	parts, err := OrcaClean05(0, maths.Local())
	if err != nil {
		return good{}, err
	}
	box := parts[0].Shape.Bounds()
	return good{native: parts, extent: &box}, nil
}

// madeAtTime is the orca at a time in its three-second swim. It refuses a
// seed that was not made, and any detail but quick.
func (orcaClean05Kind) madeAtTime(from seed, detail string, at time.Duration) (good, error) {
	if !from.made() {
		return good{}, errGood
	}
	if detail != "quick" {
		return good{}, fmt.Errorf("orca-clean-05: unsupported complexity %q", detail)
	}
	parts, err := OrcaClean05(2*math.Pi*at.Seconds()/3, maths.Local())
	if err != nil {
		return good{}, err
	}
	box := parts[0].Shape.Bounds()
	return good{native: parts, extent: &box}, nil
}
