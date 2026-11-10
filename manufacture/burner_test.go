package manufacture

import (
	"math"
	"testing"
	"time"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/flame"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/plume"
)

func burnerFixture() BurnSpec {
	return BurnSpec{Flame: flame.Fire{Radius: 0.6, Height: 2.4, Absorption: 1,
		Hottest: measure.Kelvin(1603), Edge: measure.Kelvin(900), Seed: 5},
		Smoke: &plume.Fire{Heat: 100, Convective: 0.71, Base: 1.2, Yield: 0.0024,
			Combustion: 21, Extinction: 8.7, Air: 288, Density: 1.2, Top: 12, Seed: 7},
		Albedo: 0.3, LightCells: 3}
}

func TestBurnerPlacesVolumesAndItsOwnLight(t *testing.T) {
	made, err := Burn(burnerFixture())
	if err != nil {
		t.Fatal(err)
	}
	space := maths.Subspace{Origin: maths.V(2, 3, 4), Orientation: maths.About(maths.YAxis, 0.7)}
	volumes, lights, err := made.Place(space, 1e-4, 137*time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	if len(volumes) != 2 || len(lights) == 0 {
		t.Fatal("missing fire fields or light")
	}
	if _, ok := volumes[0].Medium.(*flame.Body); !ok {
		t.Fatal("missing flame field")
	}
	if _, ok := volumes[1].Medium.(*plume.Column); !ok {
		t.Fatal("missing smoke field")
	}
	for _, volume := range volumes {
		if volume.At(0).Origin != space.Origin {
			t.Fatal("volume lost its placement")
		}
	}
	for _, light := range lights {
		if !(light.Strength.R > 0) || !(light.Across > 0) || math.IsInf(light.Across, 0) || math.IsNaN(light.Strength.R+light.Strength.G+light.Strength.B) || math.IsInf(light.Strength.R+light.Strength.G+light.Strength.B, 0) {
			t.Fatal("invalid fire source")
		}
	}
	if made.Bounds().Max.Y != 12 || made.Bounds().Min.Y != 0 {
		t.Fatal("fire lost its physical extent")
	}
}

func TestBurnerRejectsInvalidModelAndSamplingInputs(t *testing.T) {
	for _, breakIt := range []func(*BurnSpec){
		func(s *BurnSpec) { s.Flame.Radius = 0 },
		func(s *BurnSpec) { s.Flame.Absorption = -1 },
		func(s *BurnSpec) { s.Flame.Edge = measure.Kelvin(2000) },
		func(s *BurnSpec) { s.Smoke.Top = 0.1 },
		func(s *BurnSpec) { s.Albedo = math.NaN() },
		func(s *BurnSpec) { s.Albedo = math.Inf(1) },
		func(s *BurnSpec) { s.Albedo = -0.1 },
		func(s *BurnSpec) { s.Albedo = 1.1 },
		func(s *BurnSpec) { s.LightCells = 2 },
		func(s *BurnSpec) { s.LightCells = 9 },
	} {
		spec := burnerFixture()
		breakIt(&spec)
		if _, err := Burn(spec); err == nil {
			t.Fatal("invalid fire accepted")
		}
	}
}

func TestBurnerCopiesSmokeAndCanOmitIt(t *testing.T) {
	spec := burnerFixture()
	made, err := Burn(spec)
	if err != nil {
		t.Fatal(err)
	}
	spec.Smoke.Top = 999
	volumes, _, err := made.Place(maths.Subspace{}, 1e-4, 0)
	if err != nil || len(volumes) != 2 || volumes[1].Medium.Bounds().Max.Y != 12 {
		t.Fatal("caller changed the stored smoke")
	}
	spec = burnerFixture()
	spec.Smoke = nil
	made, err = Burn(spec)
	if err != nil {
		t.Fatal(err)
	}
	volumes, _, err = made.Place(maths.Subspace{}, 1e-4, 0)
	if err != nil || len(volumes) != 1 {
		t.Fatal("absent smoke was manufactured")
	}
	if made.Bounds().Max.Y >= 12 {
		t.Fatal("absent smoke retained its old extent")
	}
}

func TestBurnerRejectsInvalidPlacementOrExposure(t *testing.T) {
	made, err := Burn(burnerFixture())
	if err != nil {
		t.Fatal(err)
	}
	for _, space := range []maths.Subspace{
		{Scale: -1}, {Scale: math.NaN()}, {Scale: math.Inf(1)},
		{Origin: maths.V(math.Inf(1), 0, 0)},
	} {
		if _, _, err := made.Place(space, 1, 0); err == nil {
			t.Fatal("invalid placement accepted")
		}
	}
	for _, exposure := range []float64{0, -1, math.NaN(), math.Inf(1)} {
		if _, _, err := made.Place(maths.Subspace{}, exposure, 0); err == nil {
			t.Fatal("invalid exposure accepted")
		}
	}
}

func TestBurnerTimeIsRepeatableAndMovesItsLight(t *testing.T) {
	made, err := Burn(burnerFixture())
	if err != nil {
		t.Fatal(err)
	}
	_, one, err := made.Place(maths.Subspace{}, 1e-4, 0)
	if err != nil {
		t.Fatal(err)
	}
	_, again, err := made.Place(maths.Subspace{}, 1e-4, 0)
	if err != nil {
		t.Fatal(err)
	}
	_, later, err := made.Place(maths.Subspace{}, 1e-4, 137*time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	if len(one) == 0 || len(one) != len(again) {
		t.Fatal("empty or inconsistent source samples")
	}
	for i := range one {
		if one[i].At != again[i].At || one[i].Strength != again[i].Strength {
			t.Fatal("same time changed")
		}
	}
	changed := len(one) != len(later)
	for i := 0; i < min(len(one), len(later)); i++ {
		changed = changed || one[i].Strength != later[i].Strength
	}
	if !changed {
		t.Fatal("light was frozen in time")
	}
}

func TestBurnerScalingPreservesThePhysicalSourceIntegral(t *testing.T) {
	made, err := Burn(burnerFixture())
	if err != nil {
		t.Fatal(err)
	}
	_, one, err := made.Place(maths.Subspace{Scale: 1}, 1e-4, 0)
	if err != nil {
		t.Fatal(err)
	}
	space := maths.Subspace{Origin: maths.V(2, 0, 3), Scale: 2}
	_, two, err := made.Place(space, 1e-4, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(one) == 0 || len(one) != len(two) {
		t.Fatal("source sampling changed under scale")
	}
	for i := range one {
		if two[i].At != space.Out(one[i].At) || two[i].Across != 2*one[i].Across ||
			two[i].Strength != one[i].Strength.Scale(8) {
			t.Fatal("source scaling changed its position, cell size or volume")
		}
	}
}
