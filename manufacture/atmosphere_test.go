package manufacture

import (
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/medium"
	"github.com/janearc/slayer/scene/thing"
)

// flatFloor is a ground ten metres up everywhere.
func flatFloor(x, z float64) float64 { return 10 }

// noPlace finds no thing placed.
func noPlace(name string) (maths.Vec, bool) { return maths.Vec{}, false }

// testAtmosphere is mist over a ground of a kilometre square, 5 to 60
// metres high, reaching a hundred metres beyond it, fifty metres deep.
func testAtmosphere() Atmosphere {
	return Atmosphere{Name: "haze", Albedo: 0.9, Beyond: 100,
		Over: maths.Box{Min: maths.V(0, 5, 0), Max: maths.V(1000, 60, 1000)},
		Weather: medium.NewWeather(0.001, 50, 0.5, 1,
			maths.V(1, 0, 0)),
		Floor: flatFloor}
}

func TestALayersBoxIsTheGroundWidenedAndTheMistsDepthOverIt(t *testing.T) {
	box, err := testAtmosphere().Extent()
	if err != nil {
		t.Fatal(err)
	}
	if box.Min.X != -100 || box.Max.X != 1100 || box.Min.Z != -100 ||
		box.Max.Z != 1100 || box.Min.Y != 5 || box.Max.Y != 110 {
		t.Errorf("a layer over the ground has the box %+v", box)
	}
}

func TestALevelDeckIsHeldBetweenItsBaseAndItsDepthOverIt(t *testing.T) {
	deck := testAtmosphere()
	deck.Level, deck.Base = true, 200
	box, err := deck.Extent()
	if err != nil {
		t.Fatal(err)
	}
	if box.Min.Y != 200 || box.Max.Y != 250 {
		t.Errorf("a deck at 200 metres has the box %+v", box)
	}
}

func TestACloudIsHeldBetweenItsTwoHeights(t *testing.T) {
	cloud := testAtmosphere()
	cloud.Cloud, cloud.CloudBase, cloud.CloudTop = true, 300, 420
	box, err := cloud.Extent()
	if err != nil {
		t.Fatal(err)
	}
	if box.Min.Y != 300 || box.Max.Y != 420 || box.Min.X != -100 {
		t.Errorf("a cloud from 300 to 420 has the box %+v", box)
	}
}

func TestAnAtmosphereOverNoGroundHasNoBox(t *testing.T) {
	bare := testAtmosphere()
	bare.Over = maths.Box{}
	if _, err := bare.Extent(); err == nil {
		t.Error("an atmosphere over no ground was given a box")
	}
}

func TestALayerStagesAsOneVolumeAndOneThinningTowardAMissingThingRefuses(
	t *testing.T) {
	world := thing.NewWorld()
	if err := testAtmosphere().Staged(world, noPlace); err != nil {
		t.Fatal(err)
	}
	if len(world.Volumes) != 1 {
		t.Fatalf("%d volumes, not one", len(world.Volumes))
	}
	thin := testAtmosphere()
	thin.Toward, thin.Reach, thin.Least = "tower", 100, 0.2
	if err := thin.Staged(world, noPlace); err == nil ||
		len(world.Volumes) != 1 {
		t.Errorf("a layer thinning toward nothing left %d volumes (%v)",
			len(world.Volumes), err)
	}
}

func TestGasThroughParticlesWithNoExtinctionIsRefused(t *testing.T) {
	gassed := testAtmosphere()
	gassed.Weather.Extinction = 0
	gassed.Nitrogen = 20
	if err := gassed.Staged(thing.NewWorld(), noPlace); err == nil {
		t.Error("a gas through particles of no extinction was staged")
	}
}
