package cheeptune

import (
	"math"
)

func near(a, b float64) bool { return math.Abs(a-b) < 1e-9 }
