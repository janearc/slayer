package medium_test

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/medium"
)

// airsBox is a region 2 km across and 400 m up, the size of a city's air.
var airsBox = maths.Box{Min: maths.V(-1000, 0, -1000),
	Max: maths.V(1000, 400, 1000)}

// airsFlat is flat ground at nought.
func airsFlat(float64, float64) float64 { return 0 }

// airsWeather is an even mist 300 m deep, 0.004 a metre where it is
// thickest, with a lid 60 m deep.
func airsWeather() medium.GroveWeather {
	weather := medium.NewWeather(0.004, 300, 1, 1, maths.Vec{})
	weather.Edge = 60
	return weather
}

// airsClose is whether two numbers agree to a share of the second.
func airsClose(got, want, share float64) bool {
	return math.Abs(got-want) <= share*math.Abs(want)
}

// TestAnAerosolsGasTakesItsShareOfEachChannel: the gas's absorption adds
// to each channel's tint, and the albedo is the particles' share of it.
func TestAnAerosolsGasTakesItsShareOfEachChannel(test *testing.T) {
	layer, err := medium.NewLayer(airsWeather(), airsBox, airsFlat)
	if err != nil {
		test.Fatal(err)
	}
	gas := maths.RGB(0.1, 0.2, 0.6)
	aerosol, err := medium.NewAbsorbingAerosol(layer, 0, gas)
	if err != nil {
		test.Fatal(err)
	}
	tint := aerosol.Tint()
	if !airsClose(tint.R, 1.1, 1e-12) || !airsClose(tint.G, 1.2, 1e-12) ||
		!airsClose(tint.B, 1.6, 1e-12) {
		test.Fatalf("a grey aerosol with a gas tints %v", tint)
	}
	albedo := aerosol.Albedo(0.9)
	if !airsClose(albedo.B, 0.9/1.6, 1e-12) || !(albedo.B < albedo.R) {
		test.Fatalf("the gas did not take the blue: %v", albedo)
	}
}

// TestAnAerosolRefusesAGasThatGivesLight: a gas that absorbs less than
// nothing, or by no number, is refused.
func TestAnAerosolRefusesAGasThatGivesLight(test *testing.T) {
	layer, err := medium.NewLayer(airsWeather(), airsBox, airsFlat)
	if err != nil {
		test.Fatal(err)
	}
	for _, gas := range []maths.Light{maths.RGB(0, -0.1, 0),
		maths.RGB(math.NaN(), 0, 0), maths.RGB(0, 0, math.Inf(1))} {
		if _, err := medium.NewAbsorbingAerosol(layer, 1, gas); err == nil {
			test.Errorf("a gas of %v was taken", gas)
		}
	}
}

// TestALayerUnderALidHoldsItsDensityToTheEdge: under the edge the layer is
// as thick as it is at the ground; at the top it is nothing.
func TestALayerUnderALidHoldsItsDensityToTheEdge(test *testing.T) {
	layer, err := medium.NewLayer(airsWeather(), airsBox, airsFlat)
	if err != nil {
		test.Fatal(err)
	}
	low, high := layer.Density(maths.V(0, 1, 0)),
		layer.Density(maths.V(0, 239, 0))
	if !airsClose(high, low, 1e-6) || layer.Density(maths.V(0, 300, 0)) != 0 {
		test.Fatalf("the lid's layer is %v at 1 m and %v at 239 m", low, high)
	}
	plain := medium.NewWeather(0.004, 300, 1, 1, maths.Vec{})
	smooth, err := medium.NewLayer(plain, airsBox, airsFlat)
	if err != nil {
		test.Fatal(err)
	}
	if !(smooth.Density(maths.V(0, 239, 0)) < 0.5*low) {
		test.Fatal("a layer with no edge did not thin from the ground")
	}
}

// TestALayerRefusesAnEdgeDeeperThanItself: an edge past the depth or under
// nought is refused.
func TestALayerRefusesAnEdgeDeeperThanItself(test *testing.T) {
	for _, edge := range []float64{301, -1, math.NaN()} {
		weather := airsWeather()
		weather.Edge = edge
		if _, err := medium.NewLayer(weather, airsBox, airsFlat); err == nil {
			test.Errorf("an edge of %v was taken", edge)
		}
	}
}

// TestALayerThinsAwayFromItsPoint: at the point the layer is as thick as
// a plain one; past the reach it is the least of that.
func TestALayerThinsAwayFromItsPoint(test *testing.T) {
	layer, err := medium.NewLayer(airsWeather(), airsBox, airsFlat)
	if err != nil {
		test.Fatal(err)
	}
	plain := layer.Density(maths.V(500, 50, 0))
	if err := layer.Toward(maths.V(500, 0, 0), 400, 0.25); err != nil {
		test.Fatal(err)
	}
	near, far := layer.Density(maths.V(500, 50, 0)),
		layer.Density(maths.V(-500, 50, 0))
	if !airsClose(near, plain, 1e-12) || !airsClose(far, 0.25*plain, 1e-12) {
		test.Fatalf("near %v and far %v of %v", near, far, plain)
	}
}

// TestALayerRefusesToThinOverNoReach: a reach of nought, a least past one
// and a point of no number are refused.
func TestALayerRefusesToThinOverNoReach(test *testing.T) {
	layer, err := medium.NewLayer(airsWeather(), airsBox, airsFlat)
	if err != nil {
		test.Fatal(err)
	}
	nowhere := maths.V(math.NaN(), 0, 0)
	if layer.Toward(maths.V(0, 0, 0), 0, 0.5) == nil ||
		layer.Toward(maths.V(0, 0, 0), 100, 1.5) == nil ||
		layer.Toward(nowhere, 100, 0.5) == nil {
		test.Fatal("a layer thinned toward nothing")
	}
}

// TestACloudDeckHasBodiesAndClearSky: between its base and top a deck of
// half cover has points inside a body and points between them, and none
// below the base, over the top or out of its region.
func TestACloudDeckHasBodiesAndClearSky(test *testing.T) {
	weather := medium.NewWeather(0.05, 0, 0.5, 7, maths.Vec{})
	weather.PatchScale = 300
	cloud, err := medium.NewCloud(weather, airsBox, 1500, 2500)
	if err != nil {
		test.Fatal(err)
	}
	inside, between := 0, 0
	for step := range 400 {
		at := maths.V(-950+float64(step)*4.75, 1600, 0)
		switch cloud.Density(at) {
		case 0.05:
			inside++
		case 0:
			between++
		}
	}
	if inside == 0 || between == 0 {
		test.Fatalf("a deck of half cover had %d inside and %d between",
			inside, between)
	}
	for _, at := range []maths.Vec{maths.V(0, 1499, 0),
		maths.V(0, 2501, 0), maths.V(1200, 1600, 0)} {
		if cloud.Density(at) != 0 {
			test.Errorf("cloud at %v", at)
		}
	}
}

// TestACloudDeckIsClearFarFromItsPoint: a deck that thins to nothing away
// from its point has no cloud past its reach, and keeps its bodies near.
func TestACloudDeckIsClearFarFromItsPoint(test *testing.T) {
	weather := medium.NewWeather(0.05, 0, 0.9, 7, maths.Vec{})
	weather.PatchScale = 300
	cloud, err := medium.NewCloud(weather, airsBox, 1500, 2500)
	if err != nil {
		test.Fatal(err)
	}
	if err := cloud.Toward(maths.V(900, 0, 0), 600, 0); err != nil {
		test.Fatal(err)
	}
	near := 0
	for step := range 200 {
		far := maths.V(-950+float64(step)*2, 1510, float64(step))
		if cloud.Density(far) != 0 {
			test.Fatalf("cloud past the reach at %v", far)
		}
		spot := maths.V(700+float64(step), 1510, 0)
		if cloud.Density(spot) > 0 {
			near++
		}
	}
	if near == 0 {
		test.Fatal("the deck held no cloud near its point")
	}
	if cloud.Toward(maths.V(0, 0, 0), 0, 0.5) == nil {
		test.Error("a deck thinned over no reach")
	}
}

// TestACloudDeckRefusesAnUpsideDownDeck: a top at or under the base and a
// cover of nought are refused.
func TestACloudDeckRefusesAnUpsideDownDeck(test *testing.T) {
	weather := medium.NewWeather(0.05, 0, 0.5, 7, maths.Vec{})
	if _, err := medium.NewCloud(weather, airsBox, 2500, 1500); err == nil {
		test.Error("a deck with its top under its base was taken")
	}
	bare := medium.NewWeather(0.05, 0, 0, 7, maths.Vec{})
	if _, err := medium.NewCloud(bare, airsBox, 1500, 2500); err == nil {
		test.Error("a deck that covers nothing was taken")
	}
}

// airsStack is a boat's stack: a 0.4 m mouth, a flux of 2, 5 m/s of wind,
// 2 g a second of soot at 10 square metres a gram.
func airsStack(test *testing.T, lid float64) medium.Stack {
	test.Helper()
	stack, err := medium.NewStack(0.4, 2, 5, 2, 10, lid)
	if err != nil {
		test.Fatal(err)
	}
	return stack
}

// airsSection is the extinction summed over the section of a stretch at a
// distance down the wind, square metres a metre of plume.
func airsSection(bent *medium.Bent, along float64) float64 {
	box := bent.Bounds()
	const cells = 400
	high, wide := box.Max.Y-box.Min.Y, box.Max.Z-box.Min.Z
	sum := 0.0
	for row := range cells {
		for column := range cells {
			at := maths.V(along, box.Min.Y+(float64(row)+0.5)*high/cells,
				box.Min.Z+(float64(column)+0.5)*wide/cells)
			sum += bent.Density(at)
		}
	}
	return sum * high * wide / (cells * cells)
}

// TestABentPlumeCarriesWhatLeavesIt: across any section the extinction is
// the rate times the mass extinction over the wind, with no lid and with
// a lid over the middle, and the middle stands at Briggs's rise.
func TestABentPlumeCarriesWhatLeavesIt(test *testing.T) {
	want := 2.0 * 10 / 5
	for _, lid := range []float64{0, 120} {
		bent, err := medium.NewBent(airsStack(test, lid), 100, 400)
		if err != nil {
			test.Fatal(err)
		}
		if got := airsSection(bent, 200); !airsClose(got, want, 0.03) {
			test.Errorf("lid %v: a section carries %v, want %v", lid, got,
				want)
		}
	}
	bent, err := medium.NewBent(airsStack(test, 0), 100, 400)
	if err != nil {
		test.Fatal(err)
	}
	middle := math.Cbrt(3 * 2 * 200 * 200 / (2 * 0.6 * 0.6 * 125))
	if !(bent.Density(maths.V(200, middle, 0)) >
		bent.Density(maths.V(200, middle+20, 0))) ||
		!(bent.Density(maths.V(200, middle, 0)) >
			bent.Density(maths.V(200, middle-20, 0))) {
		test.Error("the plume's middle is not at Briggs's rise")
	}
}

// TestABentPlumeHoldsNothingOverItsLidOrOutOfItsStretch: over the lid,
// before the stretch and after it the plume is nothing.
func TestABentPlumeHoldsNothingOverItsLidOrOutOfItsStretch(
	test *testing.T) {
	bent, err := medium.NewBent(airsStack(test, 50), 100, 400)
	if err != nil {
		test.Fatal(err)
	}
	for _, at := range []maths.Vec{maths.V(200, 51, 0),
		maths.V(99, 10, 0), maths.V(401, 10, 0)} {
		if bent.Density(at) != 0 {
			test.Errorf("plume at %v", at)
		}
	}
}

// TestAStackRefusesWhatCannotLeaveIt: no wind, no mouth, no rate, a
// stretch that runs back and a lobe of one are refused.
func TestAStackRefusesWhatCannotLeaveIt(test *testing.T) {
	if _, err := medium.NewStack(0.4, 2, 0, 2, 10, 0); err == nil {
		test.Error("a stack in no wind was taken")
	}
	if _, err := medium.NewStack(0, 2, 5, 2, 10, 0); err == nil {
		test.Error("a stack with no mouth was taken")
	}
	if _, err := medium.NewStack(0.4, 2, 5, 0, 10, 0); err == nil {
		test.Error("a stack that gives nothing off was taken")
	}
	if _, err := medium.NewBent(airsStack(test, 0), 300, 200); err == nil {
		test.Error("a stretch that runs back was taken")
	}
	if _, err := airsStack(test, 0).Scattering(1, 1); err == nil {
		test.Error("a lobe of one was taken")
	}
	if _, err := airsStack(test, 0).Scattering(0.5, 3); err == nil {
		test.Error("an Angstrom exponent of three was taken")
	}
}
