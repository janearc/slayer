// Record the authored patch noise at fixed physical coordinates.
package main

import (
	"fmt"
	"time"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/medium"
)

func main() {
	weather := medium.GroveWeather{Extinction: .008, Depth: 8, Coverage: .6,
		Seed: 23, Wind: maths.V(2, .1, -.5), GroundFloor: -2, Fade: true}
	grove, err := medium.NewGrove(weather, []medium.Canopy{
		{At: maths.V(3, 0, -5), Radius: 10, Height: 20},
		{At: maths.V(-8, 0, 7), Radius: 5, Height: 12},
	}, func(x, z float64) float64 { return -.5 + .01*x - .02*z })
	if err != nil {
		panic(err)
	}
	fmt.Println("# Authored grove patch appearance; not measured fog or condensation.")
	fmt.Println("# nanoseconds x y z absorption-per-metre")
	for _, at := range []time.Duration{0, time.Second, 7 * time.Second} {
		moved := grove.Tick(at)
		for _, x := range []float64{-10, -8, -6, -2, 0, 3, 5, 9, 12} {
			for _, z := range []float64{-6, -5, -2, 4, 7, 10} {
				for _, y := range []float64{.5, 3, 6} {
					p := maths.V(x, y, z)
					fmt.Printf("%d %.17g %.17g %.17g %.17g\n", at, p.X, p.Y, p.Z, moved.Density(p))
				}
			}
		}
	}
}
