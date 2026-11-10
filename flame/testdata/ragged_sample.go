// Sample the authored rag and continuous-border regression points.
package main

import (
	"fmt"
	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/flame"
	"github.com/janearc/slayer/maths"
	"time"
)

func main() {
	points := []struct {
		seed uint64
		hot  bool
		at   time.Duration
		p    maths.Vec
	}{
		{0, false, 10656000000, maths.V(0.020000000000000000, 0.30867638448175339, 0.010000000000000000)},
		{0, false, 10693000000, maths.V(0.020000000000000000, 0.43154495789912678, 0.010000000000000000)},
		{0, false, 14689000000, maths.V(0.020000000000000000, 0.11579495598303301, 0.010000000000000000)},
		{0, false, 14763000000, maths.V(0.020000000000000000, 0.29222758611473221, 0.010000000000000000)},
		{0, false, 14800000000, maths.V(0.020000000000000000, 0.40978719132010533, 0.010000000000000000)},
		{0, false, 30229000000, maths.V(0.020000000000000000, 0.39322393981492493, 0.010000000000000000)},
		{0, false, 37259000000, maths.V(0.020000000000000000, 0.27808572126083275, 0.010000000000000000)},
		{0, false, 37296000000, maths.V(0.020000000000000000, 0.39133725090777621, 0.010000000000000000)},
		{0, false, 62197000000, maths.V(0.020000000000000000, 0.12191189114017993, 0.010000000000000000)},
		{0, false, 62271000000, maths.V(0.020000000000000000, 0.31199464151775441, 0.010000000000000000)},
		{0, false, 62308000000, maths.V(0.020000000000000000, 0.44237116190424980, 0.010000000000000000)},
		{0, false, 64861000000, maths.V(0.020000000000000000, 0.11654481753505189, 0.010000000000000000)},
		{4, false, 41922133000, maths.V(0.070000000000000007, 0.25199490397206487, 0.095000000000000001)},
		{0, false, 14726398000, maths.V(0.02, 0.2329236139369916, 0.01)},
		{0, true, 14726398000, maths.V(0.02, 0.2329236139369916, 0.01)},
	}
	fmt.Println("# Authored soot rags and exact continuous borders; not experimental data.")
	fmt.Println("# seed hot-tongues nanoseconds x y z absorption-per-metre kelvin")
	for _, row := range points {
		spec := flame.Fire{Radius: .25, Height: .6, Absorption: .8,
			Hottest: measure.Kelvin(1100), Edge: measure.Kelvin(800),
			Seed: row.seed, HotTongues: row.hot}
		source, err := flame.NewBody(spec, 1)
		if err != nil {
			panic(err)
		}
		source = source.Tick(row.at).(*flame.Body)
		fmt.Printf("%d %t %d %.17g %.17g %.17g %.17g %.17g\n", row.seed, row.hot, row.at,
			row.p.X, row.p.Y, row.p.Z, source.Density(row.p), source.Heat(row.p).InKelvin())
	}
}
