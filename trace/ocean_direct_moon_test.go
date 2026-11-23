package trace

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/medium"
	"github.com/janearc/slayer/scene/thing"
)

func TestOceanDirectMoonRefractsAndAttenuatesItsTaggedLight(t *testing.T) {
	for _, angle := range []float64{0, math.Pi / 3} {
		world, water, boundary, seen, _ := oceanSunFixture(t, angle)
		world.Lamps = world.Lamps[:1]
		ocean := world.OceanSurface
		ocean.DirectSunMeanPlane, ocean.DirectMoonMeanPlane = false, true
		ocean.SunRoute, ocean.FillLampIndex = thing.OceanSunRouteMoon, -1
		if err := validateOceanDirectSunWorld(world); err != nil {
			t.Fatal(err)
		}
		scene := stage(world, 0)
		toward, light, usable := scene.oceanSunAtSample(maths.V(0, -2, 0), seen)
		wantX := math.Sin(angle) / boundary.Material.Dense()
		wantY := math.Sqrt(1 - wantX*wantX)
		if !usable || math.Abs(toward.X-wantX) > 1e-12 ||
			math.Abs(toward.Y-wantY) > 1e-12 {
			t.Fatalf("Moon ray=%v usable=%t; want Snell direction (%g,%g)",
				toward, usable, wantX, wantY)
		}
		gain := 10 * (1 - maths.Fresnel(math.Cos(angle),
			boundary.Material.Dense())) * math.Cos(angle) / wantY
		extinction := water.Extinction()
		for _, channel := range []struct{ got, sigma float64 }{
			{light.R, extinction.R}, {light.G, extinction.G},
			{light.B, extinction.B},
		} {
			want := gain * math.Exp(-channel.sigma*2/wantY)
			if math.Abs(channel.got-want) > 1e-12 {
				t.Fatalf("Moon source=%g, want incoming Beer/Fresnel=%g", channel.got, want)
			}
		}
	}
}

func TestOceanDirectMoonLightsCloudWithoutSkyQuadrature(t *testing.T) {
	world, _, seen, _ := diffuseSkyOceanFixture(t, 1.333, 8)
	water, err := thing.NewClearMediumWithScattering("lunar-water",
		maths.Grey(.1), maths.Grey(.01), 0, 1, 4, 20)
	if err != nil {
		t.Fatal(err)
	}
	ocean := world.OceanSurface
	ocean.Boundary.Material.ClearMedium = water
	ocean.DirectSunMeanPlane, ocean.DirectMoonMeanPlane = false, true
	ocean.SunRoute, ocean.FillLampIndex = thing.OceanSunRouteMoon, -1
	ocean.ClearCloudSkySamples = 0
	world.Lamps = world.Lamps[:1]
	cloud, err := medium.NewClearCloud(medium.Fizz{Radius: 1,
		Height: 1, Thick: 1})
	if err != nil {
		t.Fatal(err)
	}
	cloudAt := thing.Still(maths.Subspace{Origin: maths.V(0, -2, 0)})
	world.AddVolume(thing.Volume{Medium: cloud,
		Material: &thing.Material{Colour: maths.Grey(1)},
		Motion:   cloudAt})
	if err := ValidateClearState(world, []InitialMedium{{Index: 1.333,
		ClearMedium: water}}); err != nil {
		t.Fatal(err)
	}
	seen.inside[0].clearMedium = water
	seen.ray = maths.Ray{At: maths.V(0, -3, 0), Dir: maths.Up}
	scene := stage(world, 0)
	if scene.oceanSun == nil {
		t.Fatal("direct Moon source context was not staged")
	}
	if len(scene.oceanSun.skyDirections) != 0 {
		t.Fatal("direct Moon silently enabled sky quadrature")
	}
	got := throughClear(scene, seen, 3, water)
	if got.added.R <= 0 || got.added.G <= 0 || got.added.B <= 0 {
		t.Fatalf("lunar water/cloud source absent: %v; %v", got, scene.refusal())
	}
}

func TestOceanDirectMoonRefusesForgedRoutesAndFill(t *testing.T) {
	for _, fault := range []string{"wrong route", "both sources", "fill"} {
		t.Run(fault, func(t *testing.T) {
			world, _, _, _, _ := oceanSunFixture(t, 0)
			world.Lamps = world.Lamps[:1]
			ocean := world.OceanSurface
			ocean.DirectSunMeanPlane, ocean.DirectMoonMeanPlane = false, true
			ocean.SunRoute, ocean.FillLampIndex = thing.OceanSunRouteMoon, -1
			switch fault {
			case "wrong route":
				ocean.SunRoute = thing.OceanSunRouteGroundlessAir
			case "both sources":
				ocean.DirectSunMeanPlane = true
			case "fill":
				ocean.FillLampIndex = 0
			}
			if err := validateOceanDirectSunWorld(world); err == nil {
				t.Fatal("accepted forged Moon source contract")
			}
		})
	}
}
