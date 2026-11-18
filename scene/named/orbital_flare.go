package named

import (
	"math"
	"time"

	"github.com/janearc/slayer/emit"
	"github.com/janearc/slayer/maths"
)

// orbitalFlare is a finite, irregular sequence on the scene clock. The RGB
// illumination is an artistic visible translation of high-energy radiation,
// not a claim that X-rays themselves are visible to a human eye.
func orbitalFlare(at time.Duration) float64 {
	seconds := at.Seconds()
	peaks := [...]float64{2.5, 8.2, 17.7, 31.4, 54.1, 88.8}
	widths := [...]float64{.36, .24, .45, .3, .38, .26}
	heights := [...]float64{1, .65, .85, .72, 1, .8}
	var light float64
	for i, peak := range peaks {
		distance := (seconds - peak) / widths[i]
		light += heights[i] * math.Exp(-.5*distance*distance)
	}
	return light
}

// addOrbitalFlares places a point source at the magnetar in the centre. Its
// inverse-square illumination is occluded by the real ring and structures.
// The scene owns the changing lamp and updates it before a frame is drawn.
func addOrbitalFlares(view *Built, offset time.Duration) {
	index := len(view.world.Lamps)
	view.world.Lamp(emit.Lit{})
	previous := view.tick
	view.tick = func(at time.Duration) {
		if previous != nil {
			previous(at)
		}
		strength := orbitalRadius * orbitalRadius * orbitalFlare(at+offset)
		view.world.Lamps[index] = emit.Lit{At: maths.Vec{},
			Strength: maths.RGB(.55, 1.25, 3.1).Scale(strength)}
	}
	view.tick(0)
}
