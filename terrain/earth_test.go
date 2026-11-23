package terrain

import (
	"math"
	"testing"
)

// groundSlack is how far a ground spacing may be from its working, in metres.
const groundSlack = 1e-6

func TestADegreeGridIsAMapOfTheEarthAndReadsAsOne(t *testing.T) {
	raster := openSpec(t, geoSpec(2))
	earth, ok := raster.Earth()
	if !ok {
		t.Fatal("a degree grid was not taken for a map of the earth")
	}
	radius, ok := earth.EquirectangularSphere()
	if !ok || radius != earthRadius {
		t.Errorf("radius %v %v, want %v", radius, ok, earthRadius)
	}
	height, present, err := earth.EquirectangularElevation(10.5, 19.5)
	if err != nil || !present || height != 5.5 {
		t.Errorf("height %v %v %v, want 5.5 as GeographicElevation gives",
			height, present, err)
	}
}

func TestOneDegreeIsAboutOneHundredAndElevenKilometresOnTheGround(t *testing.T) {
	earth, _ := openSpec(t, geoSpec(2)).Earth()
	eastward, southward, err := earth.GroundSpacing(60)
	if err != nil {
		t.Fatal(err)
	}
	degree := earthRadius * math.Pi / 180
	if math.Abs(southward-degree) > groundSlack ||
		math.Abs(eastward-degree/2) > groundSlack {
		t.Errorf("a degree is %v south and %v east at 60 degrees, want "+
			"%v and half of it", southward, eastward, degree)
	}
}

func TestAMapInMetresOrAbsentIsNotAMapOfTheEarthInDegrees(t *testing.T) {
	var nothing *Raster
	if _, ok := nothing.Earth(); ok {
		t.Error("no raster was taken for a map of the earth")
	}
	metres := openSpec(t, newTiffSpec(tiffLittle, false, 5, 4, 4, 4,
		rampAt).place(10, 20, 1, 1))
	if _, ok := metres.Earth(); ok {
		t.Error("a grid that does not say degrees was taken for the earth")
	}
}

// nad83 is the EPSG code of the geographic system NOAA's tiles name.
const nad83 = 4269

func TestACRSThatImpliesDegreesNeedNotNameTheUnit(t *testing.T) {
	cases := map[string]struct {
		ref  SourceReference
		want bool
	}{
		"unit left to NAD83": {SourceReference{Model: 2,
			GeographicCRS: nad83}, true},
		"degrees named": {SourceReference{Model: 2, GeographicCRS: nad83,
			AngularUnits: angularDegree}, true},
		"another unit named": {SourceReference{Model: 2,
			GeographicCRS: nad83, AngularUnits: 9105}, false},
		"unit left to an unknown system": {SourceReference{Model: 2,
			GeographicCRS: 32767}, false},
		"no system": {SourceReference{Model: 2}, false},
		"projected": {SourceReference{Model: 1,
			GeographicCRS: nad83}, false},
	}
	for name, each := range cases {
		if got := each.ref.degreeGrid(); got != each.want {
			t.Errorf("%s: %v, want %v", name, got, each.want)
		}
	}
}

func TestAnEarthAtNoLatitudeIsRefused(t *testing.T) {
	earth, _ := openSpec(t, geoSpec(2)).Earth()
	if _, _, err := earth.GroundSpacing(91); err == nil {
		t.Error("a latitude past the pole was read")
	}
}
