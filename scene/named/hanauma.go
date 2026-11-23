package named

import (
	"io"
	"time"

	"github.com/janearc/slayer/emit"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/render/colour"
	"github.com/janearc/slayer/scene/hanauma"
	"github.com/janearc/slayer/scene/sky"
)

// hanaumaScene describes the Hanauma Bay scene: its name, what it shows,
// and the parameters a set may give it.
func hanaumaScene() Scene {
	return Scene{
		name: "hanauma-bay", about: "Hānauma Bay crater, beach and reef from an elevated overlook", unit: "m",
		renderers: []Renderer{CPU},
		parameters: []Parameter{
			oneOf("surface", "water", []string{"water", "relief"},
				"Refracting sea or the exposed terrain for comparison"),
			oneOf("lighting", "authored", []string{"authored", "solar"},
				"Original lighting or solar position for the supplied Hawaii local date and time"),
			whole("year", 2026, between(1970, 2100), "Hawaii local calendar year"),
			whole("month", 6, between(1, 12), "Hawaii local calendar month"),
			whole("day", 21, between(1, 31), "Hawaii local calendar day"),
			whole("hour", 12, between(0, 23), "Hawaii local hour, on the 24-hour clock"),
			whole("minute", 0, between(0, 59), "Hawaii local minute"),
		},
		defaults: Defaults{width: 960, height: 600, rays: 2, wide: 63 * maths.Degree,
			tone: colour.Tone{Exposure: 1.04, Knee: .72, Top: 4}},
		build: buildHanauma,
	}
}

// buildHanauma builds the bay with water over the reef or with the bare
// relief, as the held surface value chooses.
func buildHanauma(held values, _ io.Writer) (*Built, error) {
	wet := held.choice("surface") == "water"
	location, err := hanauma.New(wet)
	if err != nil {
		return nil, err
	}
	world := location.World()
	night := false
	if held.choice("lighting") == "solar" {
		night, err = hanaumaSolarLighting(world, held)
		if err != nil {
			return nil, err
		}
	} else {
		world.Sky = sky.Gradient{
			Horizon: maths.RGB(.17, .39, .70),
			Zenith:  maths.RGB(.026, .156, .49),
			Ground:  maths.RGB(.03, .11, .22),
		}
		sun, err := emit.NewDistant(maths.V(-.49, .80, -.36), maths.RGB(1.7, 1.61, 1.47))
		if err != nil {
			return nil, err
		}
		fill, err := emit.NewDistant(maths.V(.28, .92, .27), maths.RGB(.19, .24, .36))
		if err != nil {
			return nil, err
		}
		world.Lamp(sun)
		world.Lamp(fill)
	}
	from := maths.V(-337, 190, -406)
	eye := maths.NewSubspace(from, maths.LookAt(from, maths.V(95, -39, 235)), 1)
	if err := location.Advance(0, nil); err != nil {
		return nil, err
	}
	if night {
		darkenHanaumaSeabed(world)
	}
	if !wet {
		return &Built{world: world, eye: eye}, nil
	}
	advance := func(at time.Duration) (maths.Subspace, error) {
		return eye, location.Advance(at, nil)
	}
	return &Built{world: world, eye: eye, advance: advance}, nil
}
