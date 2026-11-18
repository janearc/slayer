package gpu

import (
	"math"
	"testing"
	"time"

	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/medium"
	"github.com/janearc/slayer/scene/thing"
)

func residualFizz() medium.Fizz {
	return medium.Fizz{Radius: .6, Height: 20, Thick: .02, Rise: 10, Churn: .8, Seed: 0xdeadbeefcafe0705}
}

func TestFizzNativeUsesOneSmallCardWithoutDensityGrid(t *testing.T) {
	encoded := scene{}
	if err := encoded.fizzVolume(maths.Subspace{}, residualFizz(), 0); err != nil {
		t.Fatal(err)
	}
	if encoded.counts.volumes != 1 || encoded.boxedCPU.Cells != 0 || len(encoded.boxes.bytes) != 32 {
		t.Fatalf("native Fizz was baked/expanded: volumes%d cells%d bytes%d", encoded.counts.volumes, encoded.boxedCPU.Cells, len(encoded.boxes.bytes))
	}
}

func TestFizzStandaloneKeepsLegacyBakedRepresentation(t *testing.T) {
	world := thing.NewWorld()
	world.AddVolume(thing.Volume{Medium: residualFizz(), Material: &thing.Material{Glow: maths.Grey(1)}})
	encoded, err := encode(world, maths.Subspace{}, film.Pinhole{Wide: 1}, 0, 2, 2)
	if err != nil {
		t.Fatal(err)
	}
	if encoded.boxedCPU.Cells == 0 || len(encoded.boxes.bytes) <= 32 {
		t.Fatal("standalone Fizz silently switched to native density")
	}
}

func TestFizzNativeRefusesInvalidAndUnsafeClockButAllowsZeroDensity(t *testing.T) {
	f := residualFizz()
	f.Thick = 0
	if _, err := fizzCardOf(f); err != nil {
		t.Fatal("zero density control refused", err)
	}
	for _, bad := range []medium.Fizz{
		{Radius: .6, Height: 20, Thick: .02, Churn: math.NaN()},
		{Radius: 1e-50, Height: 20, Thick: .02},
		residualFizz().Tick(1000000 * time.Second).(medium.Fizz),
	} {
		if _, err := fizzCardOf(bad); err == nil {
			t.Fatalf("accepted unsafe native controls %+v", bad)
		}
	}
}

func TestFizzNativeRefusesFTZControlsAndSquaredRadius(t *testing.T) {
	cases := []medium.Fizz{}
	smallRadius := residualFizz()
	smallRadius.Radius = 1e-20
	smallRadius.Height = 1e-20
	smallRadius.Churn = 0
	cases = append(cases, smallRadius)
	tinyHeight := residualFizz()
	tinyHeight.Height = 1e-40
	cases = append(cases, tinyHeight)
	tinyDensity := residualFizz()
	tinyDensity.Thick = 1e-40
	cases = append(cases, tinyDensity)
	tinyClock := residualFizz()
	tinyClock.Rise = 1e-40
	cases = append(cases, tinyClock.Tick(time.Second).(medium.Fizz))
	for _, bad := range cases {
		if _, err := fizzCardOf(bad); err == nil {
			t.Fatalf("accepted FTZ-sensitive controls %+v", bad)
		}
	}
}
