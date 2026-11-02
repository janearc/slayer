//go:build darwin

package gpu

import (
	"errors"
	"testing"
	"time"

	"github.com/janearc/slayer/emit"
	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/metal"
	"github.com/janearc/slayer/scene/medium"
	"github.com/janearc/slayer/scene/thing"
)

// hazeWorld is one wispy volume, which is baked, and a lamp, so that
// every stage of a picture has something to do
func hazeWorld() *thing.World {
	world := thing.NewWorld()
	puff := medium.Wispy{Of: medium.Puff{Radius: 4, Thick: 0.5}, Seed: 3,
		Scale: 1, Amount: 0.5}
	world.AddVolume(thing.Volume{Medium: puff,
		Material: &thing.Material{Colour: maths.Grey(0.8)}})
	world.Lamp(emit.Lit{At: maths.V(0, 8, -8), Strength: maths.Grey(60)})
	return world
}

// a picture says where its time went, and the parts are real: every
// stage took some, the gpu's two stages took no more than the whole
// picture did by the clock, and a world with nothing to bake baked none
func TestAPictureSaysWhereItsTimeWent(t *testing.T) {
	eye := maths.Subspace{Origin: maths.V(0, 0, -12)}
	start := time.Now()
	_, spent, err := sampledTimed(hazeWorld(), nil, eye, film.Pinhole{},
		0, 64, 48, 1)
	whole := time.Since(start)
	if errors.Is(err, metal.ErrNoDevice) {
		t.Skip("no gpu here")
	}
	if err != nil {
		t.Fatal(err)
	}
	stages := map[string]time.Duration{"encode": spent.encode,
		"compile": spent.compile, "bake": spent.bake,
		"trace": spent.trace, "average": spent.average}
	for name, took := range stages {
		if took <= 0 {
			t.Errorf("the %s stage took %v", name, took)
		}
	}
	if spent.bake+spent.trace > whole {
		t.Errorf("the gpu spent %v of a picture that took %v",
			spent.bake+spent.trace, whole)
	}
	_, plain, err := sampledTimed(sceneWithoutTrees(), nil, eye,
		film.Pinhole{}, 0, 16, 12, 1)
	if err != nil {
		t.Fatal(err)
	}
	if plain.bake != 0 {
		t.Errorf("nothing to bake, and %v spent baking", plain.bake)
	}
}
