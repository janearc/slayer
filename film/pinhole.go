package film

import (
	"fmt"
	"math"
)

// noAngle is defined as zero radians.
const noAngle = 0

// PinholeOf constructs a square-pixel pinhole with a finite field of view.
// Wide is radians across, strictly between no angle and a half turn.
func PinholeOf(wide float64) (*Pinhole, error) {
	if math.IsNaN(wide) || wide <= noAngle || wide >= math.Pi {
		return nil, fmt.Errorf("pinhole: field of view " +
			"must be in (0, pi)")
	}
	return &Pinhole{Wide: wide}, nil
}
