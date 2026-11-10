// These are authored firebrand release samples, not experimental data.
package main

import (
	"fmt"
	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/embers"
	"github.com/janearc/slayer/maths"
	"time"
)

func main() {
	spec := embers.Spec{Rate: 2, Life: 3 * time.Second, Drag: 700 * time.Millisecond, Radius: measure.Metres(.02), Disk: measure.Metres(2), Height: measure.Metres(1), Velocity: maths.V(0, 9, 0), Jitter: maths.V(2, 2, 2), Air: maths.V(2, 6, 0), Gravity: 9.80665, Hot: measure.Kelvin(1400), Cool: measure.Kelvin(700), Seed: 31}
	source, err := embers.New(spec)
	if err != nil {
		panic(err)
	}
	fmt.Println("# authored release regression: ID age-ns x-m y-m z-m heat-K")
	for _, p := range source.At(4125 * time.Millisecond) {
		v := p.Position()
		fmt.Printf("%d %d %.17g %.17g %.17g %.17g\n", p.ID(), p.Age(), v.X, v.Y, v.Z, p.Heat().InKelvin())
	}
}
