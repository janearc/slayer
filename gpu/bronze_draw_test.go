//go:build draw && darwin

package gpu

import (
	"math"
	"testing"

	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/sky"
	"github.com/janearc/slayer/scene/thing"
)

// These air-side samples test transport of the declared appearance estimate,
// not the alloy's measured BRDF or the separate underwater relative-IOR issue.
func TestMetalNickelAluminiumBronzeEstimateMatchesCPU(t *testing.T) {
	bronze := thing.NickelAluminiumBronze()
	for _, cosine := range []float64{1, .5, .1} {
		world := thing.NewWorld()
		world.Sky = sky.Gradient{Horizon: maths.Grey(1),
			Zenith: maths.Grey(1), Ground: maths.Grey(1)}
		world.Add(thing.Thing{Shape: shape.Plane{},
			Material: &thing.Material{Gloss: 1, Metal: bronze}})
		direction := maths.V(math.Sqrt(1-cosine*cosine), -cosine, 0)
		eye := opticalEye(maths.V(0, 1, 0), direction)
		var expected [3]float64
		for band := range expected {
			expected[band] = thing.Conductor(cosine,
				bronze.N[band], bronze.K[band])
		}
		renderOpticalCase(t, "bronze air reflection", world, eye,
			film.Pinhole{}, nil, expected, 3e-5)
	}
}
