// This producer records the authored flame shape at fixed points. It does not
// supply experimental data or an independent physical reference.
package main

import (
	"fmt"
	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/flame"
	"github.com/janearc/slayer/maths"
	"time"
)

func main() {
	fmt.Println("# authored flame regression: radius hot-tongues time-ns x y z absorption-per-m heat-K")
	for _, radius := range []float64{.25, 1.3} {
		for _, hot := range []bool{false, true} {
			spec := flame.Fire{Radius: radius, Height: radius * 2.4, Absorption: .8, Hottest: measure.Kelvin(1100), Edge: measure.Kelvin(800), Seed: 7, HotTongues: hot, Intermittency: .2}
			original, err := flame.NewBody(spec, 1)
			if err != nil {
				panic(err)
			}
			for _, at := range []time.Duration{0, 130 * time.Millisecond, 2 * time.Second} {
				body := original.Tick(at).(*flame.Body)
				for _, level := range []float64{.1, .3, .5, .7, .9} {
					for _, aside := range []float64{0, .4} {
						p := maths.V(aside*radius, level*spec.Height, -.2*radius)
						fmt.Printf("%.17g %t %d %.17g %.17g %.17g %.17g %.17g\n", radius, hot, at, p.X, p.Y, p.Z, body.Density(p), body.Heat(p).InKelvin())
					}
				}
			}
		}
	}
}
