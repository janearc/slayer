package pinnipeds

import (
	"github.com/janearc/slayer/manufacture/geometry"
	"math"

	"github.com/janearc/slayer/maths"
)

// sealEase is a share clamped to nought to one and eased, so that it has
// no slope at either end.
func sealEase(value float64) float64 {
	value = math.Max(0, math.Min(1, value))
	return value * value * (3 - 2*value)
}

// sealSection locates an elliptical cross section along the seal's X axis.
// Dimensions are artistic choices in metres for a roughly 1.7 m animal.
type sealSection struct{ x, y, width, height float64 }

var harborSealProfile = []sealSection{
	{-0.84, -0.020, 0.029, 0.038},
	{-0.75, -0.010, 0.060, 0.066},
	{-0.63, -0.003, 0.112, 0.105},
	{-0.48, 0.002, 0.174, 0.159},
	{-0.28, 0.007, 0.224, 0.205},
	{-0.06, 0.019, 0.242, 0.222},
	{0.14, 0.039, 0.223, 0.205},
	{0.30, 0.052, 0.179, 0.172},
	{0.43, 0.060, 0.130, 0.134},
	{0.54, 0.082, 0.127, 0.130},
	{0.64, 0.084, 0.114, 0.110},
	{0.73, 0.056, 0.089, 0.075},
	{0.79, 0.048, 0.066, 0.051},
	{0.82, 0.049, 0.027, 0.028},
	{0.83, 0.053, 0.005, 0.008},
}

// sealRestRings subdivides the authored trunk profile into its rest rings.
func sealRestRings() [][]maths.Vec {
	const subdivisions = 4
	profile := harborSealProfile
	rings := make([][]maths.Vec, 0, (len(profile)-1)*subdivisions+1)
	for index := 0; index < len(profile)-1; index++ {
		before := profile[max(0, index-1)]
		low, high := profile[index], profile[index+1]
		after := profile[min(len(profile)-1, index+2)]
		for step := range subdivisions {
			share := float64(step) / subdivisions
			section := sealSection{
				x:      low.x + (high.x-low.x)*share,
				y:      geometry.CatmullRom(before.y, low.y, high.y, after.y, share),
				width:  geometry.CatmullRom(before.width, low.width, high.width, after.width, share),
				height: geometry.CatmullRom(before.height, low.height, high.height, after.height, share),
			}
			rings = append(rings, sealBodyRing(section))
		}
	}
	return append(rings, sealBodyRing(profile[len(profile)-1]))
}

// sealBodyRing is the ring around one rest section.
func sealBodyRing(section sealSection) []maths.Vec {
	return geometry.Ring(maths.V(section.x, section.y, 0), maths.V(0, section.height, 0),
		maths.V(0, 0, section.width), func(point maths.Vec) maths.Vec { return point })
}
