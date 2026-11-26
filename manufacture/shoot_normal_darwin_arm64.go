//go:build darwin && arm64

package manufacture

import (
	"math"

	"github.com/janearc/slayer/maths"
)

// shootNormal retains the original arm64 squared-length rounding order when
// profile-driven foliage changes the compiler's inlining decisions.
func shootNormal(v maths.Vec) maths.Vec {
	sum := math.FMA(v.Y, v.Y, v.X*v.X)
	sum = math.FMA(v.Z, v.Z, sum)
	length := math.Sqrt(sum)
	if length == 0 {
		return v
	}
	return v.Scale(1 / length)
}
