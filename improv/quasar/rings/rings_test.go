package rings

import (
	"math"
	"testing"

	"github.com/janearc/slayer/research"
)

// The numbers the tests choose, each picked by the quasar slice,
// 2026-11-12, unless it says it is defined.
const (
	// smokeEye and smokeLean are smoke 36's eye: fifty masses out, ten
	// degrees above the disc (scene/smoke/smoke.go, thirtySix).
	smokeEye  = 50
	smokeLean = 10 * math.Pi / 180
	// smokeInner and smokeOuter are the quasar template's disc, three and
	// thirty schwarzschild radii, in masses.
	smokeInner = 6
	smokeOuter = 60
	// smokeWide and smokePixels are a portrait short's picture of smoke
	// 36: its own forty degrees, across 1080 pixels.
	smokeWide   = 40 * math.Pi / 180
	smokePixels = 1080
	// shadowTolerance is how near root twenty-seven, in masses, the
	// impact parameter at the found edge must be.
	shadowTolerance = 1e-6
	// percent is defined: a share as a percentage.
	percent = 100
	// roundHalf is defined: a percentage rounds to the whole number
	// within a half of it.
	roundHalf = 0.5
	// farOff is an angle, in radians, at which a ray passes well clear
	// of the hole: it goes less than a turn round.
	farOff = 0.5
	// justPast is how far past the edge, in radians, a ray winds more
	// than twice round before it leaves.
	justPast = 1e-8
	// tooMany is more orders than the search reaches.
	tooMany = 9
	// rootTwentySeven is defined: the shadow's impact parameter, in
	// masses, for a still hole.
	rootTwentySeven = 5.196152422706632
)

func smokeLens(t *testing.T) Lens {
	t.Helper()
	lens, err := NewLens(smokeEye, smokeLean)
	if err != nil {
		t.Fatal(err)
	}
	return lens
}

func smokeDisc(t *testing.T) Disc {
	t.Helper()
	disc, err := NewDisc(smokeInner, smokeOuter)
	if err != nil {
		t.Fatal(err)
	}
	return disc
}

func TestNewLensHoldsWhatItIsGiven(t *testing.T) {
	lens := smokeLens(t)
	if lens.Distance() != smokeEye || lens.Lean() != smokeLean {
		t.Fatalf("lens is %v out at %v, want %v at %v", lens.Distance(),
			lens.Lean(), smokeEye, smokeLean)
	}
}

func TestNewLensRefusesAnEyeInsideThePhotonSphere(t *testing.T) {
	for _, distance := range []float64{photonSphere, 1, -smokeEye,
		math.NaN(), math.Inf(1)} {
		if _, err := NewLens(distance, smokeLean); err == nil {
			t.Errorf("an eye %v masses out was taken", distance)
		}
	}
}

func TestNewLensRefusesALeanThatReachesTheAxis(t *testing.T) {
	for _, lean := range []float64{halfTurn / two, -halfTurn / two,
		math.NaN()} {
		if _, err := NewLens(smokeEye, lean); err == nil {
			t.Errorf("a lean of %v was taken", lean)
		}
	}
}

func TestTheEdgeIsRootTwentySeven(t *testing.T) {
	lens := smokeLens(t)
	impact := lens.Impact(lens.Edge())
	if math.Abs(impact-rootTwentySeven) > shadowTolerance {
		t.Fatalf("the edge's impact parameter is %v, want %v",
			impact, rootTwentySeven)
	}
	if math.Abs(lens.Shadow()-rootTwentySeven) > shadowTolerance {
		t.Fatalf("celestial's shadow is %v", lens.Shadow())
	}
}

func TestJustInsideTheEdgeIsBlackAndJustOutsideIsNot(t *testing.T) {
	lens := smokeLens(t)
	edge := lens.Edge()
	if !lens.Follow(edge - justPast).Captured() {
		t.Error("a ray just inside the edge escaped")
	}
	if lens.Follow(edge + justPast).Captured() {
		t.Error("a ray just outside the edge was captured")
	}
}

func TestARayNearTheEdgeGoesRoundAndOneFarOffDoesNot(t *testing.T) {
	lens := smokeLens(t)
	near := lens.Follow(lens.Edge() + justPast)
	if near.Turns() < two {
		t.Errorf("a ray just past the edge went %v turns round",
			near.Turns())
	}
	far := lens.Follow(farOff)
	if far.Turns() >= one || far.Captured() {
		t.Errorf("a ray %v radians off went %v turns, captured %v",
			farOff, far.Turns(), far.Captured())
	}
}

func TestAPathStartsAtTheEye(t *testing.T) {
	lens := smokeLens(t)
	first := lens.Follow(farOff).Points()[none]
	if math.Hypot(first.Along, first.Up)-smokeEye > shadowTolerance {
		t.Fatalf("the path starts at %v, not %v out", first, smokeEye)
	}
}

func TestLandsRefusesAnOrderTheRayNeverMade(t *testing.T) {
	path := smokeLens(t).Follow(farOff)
	disc := smokeDisc(t)
	made := len(path.Crossings())
	if path.Lands(-one, disc) || path.Lands(made, disc) {
		t.Fatalf("a ray with %d crossings landed an order it never "+
			"made", made)
	}
}

func TestEachSubringIsFourPercentOfTheOneOutsideIt(t *testing.T) {
	bands, err := smokeLens(t).Bands(smokeDisc(t), photonSphere)
	if err != nil {
		t.Fatal(err)
	}
	said := research.Value("photon-subring-dimming-schwarzschild-percent")
	ratio := bands[two].Width() / bands[one].Width() * percent
	if math.Abs(ratio-said) > roundHalf {
		t.Fatalf("the third image is %.2f%% of the second, which does "+
			"not round to the paper's %v%%", ratio, said)
	}
}

func TestTheDirectImageIsNotYetTheLimit(t *testing.T) {
	bands, err := smokeLens(t).Bands(smokeDisc(t), two)
	if err != nil {
		t.Fatal(err)
	}
	said := research.Value("photon-subring-dimming-schwarzschild-percent")
	ratio := bands[one].Width() / bands[none].Width() * percent
	if math.Abs(ratio-said) <= roundHalf {
		t.Fatalf("the second image is %.2f%% of the first: the factor "+
			"is a limit for later images, and the first is not one",
			ratio)
	}
}

func TestEachBandLiesInsideTheLast(t *testing.T) {
	bands, err := smokeLens(t).Bands(smokeDisc(t), photonSphere)
	if err != nil {
		t.Fatal(err)
	}
	for order := one; order < len(bands); order++ {
		inner, outer := bands[order], bands[order-one]
		if inner.Order() != order || !(inner.Far() < outer.Near()) {
			t.Errorf("band %d reaches %v, past band %d's %v", order,
				inner.Far(), order-one, outer.Near())
		}
	}
}

func TestBandsRefusesNoOrders(t *testing.T) {
	if _, err := smokeLens(t).Bands(smokeDisc(t), none); err == nil {
		t.Fatal("no orders were taken")
	}
}

func TestBandsSaysWhenAnOrderIsPastTheSearch(t *testing.T) {
	if _, err := smokeLens(t).Bands(smokeDisc(t), tooMany); err == nil {
		t.Fatalf("%d orders were all found", tooMany)
	}
}

func TestNewDiscRefusesADiscWithNoRing(t *testing.T) {
	for _, edges := range [][two]float64{{horizon, smokeOuter},
		{smokeInner, smokeInner}, {smokeInner, math.Inf(1)},
		{math.NaN(), smokeOuter}} {
		if _, err := NewDisc(edges[none], edges[one]); err == nil {
			t.Errorf("a disc from %v to %v was taken", edges[none],
				edges[one])
		}
	}
}

func TestPixelsIsTheTangentOverTheHalfWidth(t *testing.T) {
	camera, err := NewCamera(smokeWide, smokePixels)
	if err != nil {
		t.Fatal(err)
	}
	band := Band{near: farOff / two, far: farOff}
	half := math.Tan(smokeWide / two)
	want := (math.Tan(farOff) - math.Tan(farOff/two)) / half *
		smokePixels / two
	if got := camera.Pixels(band); math.Abs(got-want) > shadowTolerance {
		t.Fatalf("band is %v pixels, want %v", got, want)
	}
	if camera.Wide() != smokeWide {
		t.Fatalf("camera is %v wide", camera.Wide())
	}
}

func TestNewCameraRefusesNoPinhole(t *testing.T) {
	for _, wide := range []float64{none, halfTurn, math.NaN()} {
		if _, err := NewCamera(wide, smokePixels); err == nil {
			t.Errorf("a camera %v wide was taken", wide)
		}
	}
	if _, err := NewCamera(smokeWide, none); err == nil {
		t.Error("a picture of no pixels was taken")
	}
}

func TestCloseInMakesTheBandSoManyPixels(t *testing.T) {
	camera, err := NewCamera(smokeWide, smokePixels)
	if err != nil {
		t.Fatal(err)
	}
	band := Band{order: two, near: justPast, far: justPast * two}
	closer, err := camera.CloseIn(band, smokePixels/percent)
	if err != nil {
		t.Fatal(err)
	}
	pixel := two * math.Tan(closer.Wide()/two) / smokePixels
	if got := band.Width() / pixel; math.Abs(got-smokePixels/percent) >
		shadowTolerance*smokePixels {
		t.Fatalf("the close-in makes the band %v pixels", got)
	}
}

func TestCloseInRefusesUnderAPixelOrNoNarrowing(t *testing.T) {
	camera, err := NewCamera(smokeWide, smokePixels)
	if err != nil {
		t.Fatal(err)
	}
	band := Band{near: none, far: farOff}
	if _, err := camera.CloseIn(band, roundHalf); err == nil {
		t.Error("half a pixel was taken")
	}
	if _, err := camera.CloseIn(band, one); err == nil {
		t.Error("a close-in wider than the camera was taken")
	}
}

func TestADiscSaysItsEdges(t *testing.T) {
	disc := smokeDisc(t)
	if disc.Inner() != smokeInner || disc.Outer() != smokeOuter {
		t.Fatalf("the disc runs %v to %v", disc.Inner(), disc.Outer())
	}
}
