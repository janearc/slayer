package trace

import (
	"fmt"
	"math"
	"strings"
	"testing"

	"github.com/janearc/slayer/emit"
	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/gas"
	"github.com/janearc/slayer/scene/thing"
)

// lobeless is a world of a slab of air with a given asymmetry between the
// eye and the middle, lit by a distant lamp.
func lobeless(asymmetry float64) *thing.World {
	air := gas.Air()
	air.Phase = asymmetry
	lamp, err := emit.NewDistant(maths.V(0, 0, 1), maths.Grey(1))
	if err != nil {
		panic(err)
	}
	world := thing.NewWorld()
	world.AddVolume(thing.Volume{Medium: gas.Of{Shape: plate(1, -1, 1),
		In: air, Unit: 1}})
	world.Lamp(lamp)
	return world
}

// entries is each entry point drawing a world, as a function that says
// what came back, so one table of worlds serves all three.
func entries(world *thing.World) map[string]func() (bool, error) {
	eye, lens := tenBack(), film.Pinhole{Wide: 10 * maths.Degree}
	return map[string]func() (bool, error){
		"Render": func() (bool, error) {
			got, err := Render(world, eye, lens, 0, 4, 4)
			return got != nil, err
		},
		"Sampled": func() (bool, error) {
			got, err := Sampled(world, eye, lens, 0, 4, 4, 2)
			return got != nil, err
		},
		"RenderIn": func() (bool, error) {
			got, err := RenderIn(world, eye, lens, 0, 4, 4, magellan)
			return got != nil, err
		},
	}
}

// A gas whose asymmetry is no lobe is refused by every entry point, with an
// error that names the gas and its asymmetry, and no frame; none panics.
func TestAGasWithNoLobeIsRefused(t *testing.T) {
	for _, asymmetry := range []float64{1.5, 1, -1, math.NaN(),
		math.Inf(1)} {
		for name, draw := range entries(lobeless(asymmetry)) {
			func() {
				defer func() {
					if caught := recover(); caught != nil {
						t.Errorf("%s of asymmetry %v panics: %v", name,
							asymmetry, caught)
					}
				}()
				drawn, err := draw()
				if drawn || err == nil {
					t.Errorf("%s of asymmetry %v gives a frame %v and "+
						"error %v, want an error and no frame", name,
						asymmetry, drawn, err)
					return
				}
				said := err.Error()
				want := fmt.Sprintf("%v", asymmetry)
				if !strings.Contains(said, gas.Air().Name) ||
					!strings.Contains(said, want) ||
					!strings.Contains(said, "between minus one and one") {
					t.Errorf("%s of asymmetry %v says %q, want the gas, "+
						"%q and what is allowed", name, asymmetry, said,
						want)
				}
			}()
		}
	}
}

// A gas whose asymmetry is a lobe, however near the edge, is drawn.
func TestAGasWithALobeIsDrawn(t *testing.T) {
	for _, asymmetry := range []float64{0.999, -0.999, 0} {
		for name, draw := range entries(lobeless(asymmetry)) {
			drawn, err := draw()
			if !drawn || err != nil {
				t.Errorf("%s of asymmetry %v gives a frame %v and error "+
					"%v, want a frame", name, asymmetry, drawn, err)
			}
		}
	}
}
