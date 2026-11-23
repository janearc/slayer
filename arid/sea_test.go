package arid

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/oceantex"
)

// windSea is a 7 m/s sea, fully developed and deep, running north.
var windSea = oceantex.Sea{Wind: 7, Fetch: 500000, Depth: 0, Toward: 1.5708}

// fixtureSea is the wind sea at time nought, seed 7.
func fixtureSea(t *testing.T) *Sea {
	t.Helper()
	sea, err := NewSea(windSea, 0, 7)
	if err != nil {
		t.Fatal(err)
	}
	return sea
}

// brute is the first place a ray goes from above the surface to below it,
// found by stepping a short fixed distance, or false.
func brute(sea *Sea, ray maths.Ray, far, step float64) (float64, bool) {
	before := sea.gap(ray, 0)
	for along := step; along < far; along += step {
		now := sea.gap(ray, along)
		if before > 0 && now <= 0 {
			return along, true
		}
		before = now
	}
	return 0, false
}

func TestSeaHitLandsOnTheSurface(t *testing.T) {
	sea := fixtureSea(t)
	for index := range 40 {
		slant := 0.002 + 0.01*float64(index)
		angle := float64(index)
		ray := maths.Ray{At: maths.V(float64(index)*3.1, 2, -float64(index)),
			Dir: maths.V(math.Cos(angle), -slant, math.Sin(angle))}
		along, hit := sea.Hit(ray, 0, 1e6)
		if !hit {
			continue
		}
		if gap := sea.gap(ray, along); math.Abs(gap) >= 2*seaTouch {
			t.Fatalf("ray %d: met at %g with %g m between it and the surface",
				index, along, gap)
		}
	}
}

func TestSeaHitFindsWhatSteppingFinds(t *testing.T) {
	sea := fixtureSea(t)
	for index := range 12 {
		angle := float64(index)
		ray := maths.Ray{At: maths.V(0, 1.5, 0),
			Dir: maths.V(math.Cos(angle), -0.02, math.Sin(angle))}
		wanted, found := brute(sea, ray, 2000, 0.01)
		along, hit := sea.Hit(ray, 0, 1e6)
		if hit != found {
			t.Fatalf("ray %d: hit %t, stepping found %t", index, hit, found)
		}
		if hit && along > wanted+0.01 {
			t.Fatalf("ray %d: met at %g, after stepping did at %g", index,
				along, wanted)
		}
	}
}

func TestSeaStillSeaIsAPlane(t *testing.T) {
	sea, err := NewSea(oceantex.Sea{}, 0, 7)
	if err != nil {
		t.Fatal(err)
	}
	along, hit := sea.Hit(maths.Ray{At: maths.V(3, 10, -4),
		Dir: maths.V(0, -1, 0)}, 0, 100)
	if !hit || math.Abs(along-10) > 2*seaTouch {
		t.Fatalf("a still sea met at (%g, %t), want 10", along, hit)
	}
	if normal := sea.Normal(maths.V(50, 0, 50)); normal != maths.Up {
		t.Fatalf("a still sea faces %v", normal)
	}
	if _, hit := sea.Hit(maths.Ray{At: maths.V(0, 0, 0),
		Dir: maths.V(1, 0, 0)}, 0, 100); hit {
		t.Fatal("a ray along a still sea met it")
	}
}

func TestSeaNormalIsOneUnitAndLeansWithTheSlope(t *testing.T) {
	sea := fixtureSea(t)
	for index := range 50 {
		at := maths.V(float64(index)*7.3, 0, float64(index)*-5.1)
		normal := sea.Normal(at)
		if math.Abs(normal.Len()-1) > 1e-12 || normal.Y <= 0 {
			t.Fatalf("normal %v at %v", normal, at)
		}
		eastward, northward := sea.slope(at.X, at.Z)
		if math.Abs(normal.X/normal.Y+eastward) > 1e-9 ||
			math.Abs(normal.Z/normal.Y-northward) > 1e-9 {
			t.Fatalf("normal %v does not lean with slope (%g, %g)", normal,
				eastward, northward)
		}
	}
}

func TestSeaDistanceIsAtMostTheGapAndSignedBySide(t *testing.T) {
	sea := fixtureSea(t)
	for index := range 30 {
		at := maths.V(float64(index)*11, 0, float64(index)*3)
		surface := sea.height(at.X, at.Z)
		above, below := at, at
		above.Y, below.Y = surface+2, surface-2
		if distance := sea.Distance(above); distance <= 0 || distance > 2 {
			t.Fatalf("2 m above the surface the distance is %g", distance)
		}
		if distance := sea.Distance(below); distance >= 0 || distance < -2 {
			t.Fatalf("2 m below the surface the distance is %g", distance)
		}
	}
}

func TestSeaBoundsHoldTheSurface(t *testing.T) {
	sea := fixtureSea(t)
	box := sea.Bounds()
	if !math.IsInf(box.Min.X, -1) || !math.IsInf(box.Max.Z, 1) {
		t.Fatalf("the slab has an edge: %v", box)
	}
	for index := range 200 {
		height := sea.height(float64(index)*37.9, float64(index)*-23.3)
		if height < box.Min.Y || height > box.Max.Y {
			t.Fatalf("height %g outside the slab %g to %g", height, box.Min.Y,
				box.Max.Y)
		}
	}
}

func TestSeaRepeatsOnlyAtItsLongestBand(t *testing.T) {
	sea := fixtureSea(t)
	far := 3 * sea.Period()
	if far < 1e4 {
		t.Fatalf("the sea repeats every %g m, too soon to have no edge", sea.Period())
	}
	if sea.height(far, far) == 0 {
		t.Fatal("the sea is flat three periods from its middle")
	}
}

func TestSeaGlintBudgetSplitsTheTotal(t *testing.T) {
	sea := fixtureSea(t)
	budget, err := sea.GlintBudget(windSea)
	if err != nil {
		t.Fatal(err)
	}
	grid, err := windSea.SlopeBudget(sea.finest)
	if err != nil {
		t.Fatal(err)
	}
	if budget.Total != grid.Total {
		t.Fatalf("total %g, want Cox and Munk's for the wind, %g",
			budget.Total, grid.Total)
	}
	if !budget.Floored && math.Abs(budget.Resolved+budget.Unresolved-budget.Total) > 1e-12 {
		t.Fatalf("resolved %g and unresolved %g do not make %g",
			budget.Resolved, budget.Unresolved, budget.Total)
	}
	want := oceantex.CascadeCut() * math.Pi / sea.finest
	if budget.Nyquist != want {
		t.Fatalf("nyquist %g, want %g", budget.Nyquist, want)
	}
}

func TestSeaMissesWhatCannotMeetIt(t *testing.T) {
	sea := fixtureSea(t)
	top := sea.Bounds().Max.Y
	for name, row := range map[string]struct {
		ray       maths.Ray
		near, far float64
	}{
		"above the slab, level": {maths.Ray{At: maths.V(0, top+1, 0),
			Dir: maths.V(1, 0, 0)}, 0, 1e5},
		"above the slab, climbing": {maths.Ray{At: maths.V(0, top+1, 0),
			Dir: maths.V(1, 0.1, 0)}, 0, 1e5},
		"leaving the surface": {maths.Ray{At: maths.V(0, sea.height(0, 0), 0),
			Dir: maths.V(0, 1, 0)}, 0, 1e5},
		"nothing in range": {maths.Ray{At: maths.V(0, top+50, 0),
			Dir: maths.V(0, -1, 0)}, 0, 10},
		"no length": {maths.Ray{At: maths.V(0, 5, 0)}, 0, 10},
		"not a number": {maths.Ray{At: maths.V(math.NaN(), 5, 0),
			Dir: maths.V(0, -1, 0)}, 0, 10},
		"empty range": {maths.Ray{At: maths.V(0, 5, 0),
			Dir: maths.V(0, -1, 0)}, 10, 10},
	} {
		if _, hit := sea.Hit(row.ray, row.near, row.far); hit {
			t.Errorf("%s: met the surface", name)
		}
	}
}

func TestSeaMeetsFromBelowToo(t *testing.T) {
	sea := fixtureSea(t)
	ray := maths.Ray{At: maths.V(0, sea.height(0, 0)-5, 0), Dir: maths.V(0, 1, 0)}
	along, hit := sea.Hit(ray, 0, 100)
	if !hit || math.Abs(sea.gap(ray, along)) >= 2*seaTouch {
		t.Fatalf("a ray rising from below met at (%g, %t)", along, hit)
	}
}

// The rays of the upright tests stand on a square of uprightSide places a
// side, uprightApart metres apart, and begin uprightAway metres above or
// below the mean.
const (
	uprightSide  = 33  // picked: a thousand rays and a few
	uprightApart = 2.5 // picked: a few to a wave of this sea
	uprightAway  = 10  // picked: well outside the slab of a 7 m/s sea
)

// upright is the rays of the upright tests that go a way, up or down,
// from the other side of the mean.
func upright(way float64) []maths.Ray {
	var rays []maths.Ray
	for row := range uprightSide {
		for column := range uprightSide {
			start := maths.V(uprightApart*float64(column), -way*uprightAway,
				uprightApart*float64(row))
			rays = append(rays, maths.Ray{At: start, Dir: maths.V(0, way, 0)})
		}
	}
	return rays
}

// A ray straight down or straight up meets the sea wherever it falls. Its
// step lands on the surface itself, a hair before it or a hair past it by
// the last bit of the sum, and either is a meeting.
func TestAnUprightRayMeetsTheSeaWhereverItFalls(t *testing.T) {
	sea := fixtureSea(t)
	for name, way := range map[string]float64{"down": -1, "up": 1} {
		for _, ray := range upright(way) {
			along, hit := sea.Hit(ray, 0, 2*uprightAway)
			if !hit {
				t.Fatalf("a ray straight %s at %g, %g met nothing", name,
					ray.At.X, ray.At.Z)
			}
			if gap := sea.gap(ray, along); math.Abs(gap) >= 2*seaTouch {
				t.Fatalf("a ray straight %s met %g m from the surface", name,
					gap)
			}
		}
	}
}

// Anti: a ray that begins a hair under the surface, or a hair over it, has
// begun at the sea and has met nothing, whether its first step takes it
// further off or to the surface's other side: a ray thrown back from the
// sea, or bent into it, begins a hair to either side.
func TestARayThatBeginsWithinATouchOfTheSeaMeetsNothing(t *testing.T) {
	sea := fixtureSea(t)
	hair := seaTouch / 10
	for name, ray := range map[string]maths.Ray{
		"from a hair under, up": {At: maths.V(3, sea.height(3, 4)-hair, 4),
			Dir: maths.V(0, 1, 0)},
		"from a hair over, up": {At: maths.V(3, sea.height(3, 4)+hair, 4),
			Dir: maths.V(0, 1, 0)},
		"from a hair over, down": {At: maths.V(3, sea.height(3, 4)+hair, 4),
			Dir: maths.V(0, -1, 0)},
		"from a hair under, down": {At: maths.V(3, sea.height(3, 4)-hair, 4),
			Dir: maths.V(0, -1, 0)},
	} {
		if along, hit := sea.Hit(ray, 0, uprightAway); hit {
			t.Errorf("%s: met the surface %g along", name, along)
		}
	}
}

func TestArrivedIsAStepFromOutsideTheTouchToTheOtherSide(t *testing.T) {
	for name, row := range map[string]struct {
		before, gap float64
		want        bool
	}{
		"from above to a hair under": {5, -1e-16, true},
		"from below to a hair over":  {-5, 1e-16, true},
		"from the touch itself":      {seaTouch, -1e-16, true},
		"still above":                {5, 1e-16, false},
		"still below":                {-5, -1e-16, false},
		"began within the touch":     {seaTouch / 2, -seaTouch / 2, false},
		"the first step":             {0, -1e-16, false},
		"on the surface":             {5, 0, false},
	} {
		if got := arrived(row.before, row.gap); got != row.want {
			t.Errorf("%s: arrived %t, want %t", name, got, row.want)
		}
	}
}

func TestSeaRefusesBandsItCannotHold(t *testing.T) {
	good := oceantex.WaveCascade{Field: &oceantex.Field{N: 2, Extent: 1,
		Height: make([]float64, 4), SlopeX: make([]float64, 4),
		SlopeY: make([]float64, 4)}}
	if _, err := seaOf([]oceantex.WaveCascade{good}); err != nil {
		t.Fatalf("refused a sound band: %v", err)
	}
	short := good
	short.Field = &oceantex.Field{N: 2, Extent: 1, Height: make([]float64, 4),
		SlopeX: make([]float64, 3), SlopeY: make([]float64, 4)}
	for name, cascades := range map[string][]oceantex.WaveCascade{
		"none":         nil,
		"no field":     {{}},
		"short slopes": {short},
		"not a number": {{Field: good.Field, ResolvedAlong: math.NaN()}},
		"not a height": {{Field: &oceantex.Field{N: 2, Extent: 1,
			Height: []float64{0, math.NaN(), 0, 0},
			SlopeX: make([]float64, 4), SlopeY: make([]float64, 4)}}},
		"no side":    {{Field: &oceantex.Field{N: 2}}},
		"one sample": {{Field: &oceantex.Field{N: 1, Extent: 1, Height: []float64{0}}}},
	} {
		if _, err := seaOf(cascades); err == nil {
			t.Errorf("accepted %s", name)
		}
	}
	if _, err := NewSea(windSea, -1, 7); err == nil {
		t.Error("accepted a negative time")
	}
}

func TestSteepestRiseIsAtLeastEverySampleStep(t *testing.T) {
	sea := fixtureSea(t)
	for index, band := range sea.bands {
		spacing := band.Extent / float64(band.N)
		bound := steepestRise(band)
		for at := 0; at+1 < band.N*band.N; at++ {
			if at%band.N == band.N-1 {
				continue
			}
			rise := math.Abs(band.Height[at+1]-band.Height[at]) / spacing
			if rise > bound+1e-12 {
				t.Fatalf("band %d: a step rises %g, over the bound %g", index,
					rise, bound)
			}
		}
	}
}

func TestEndOnMeanIsTheMeanPlaneBetweenNearAndFar(t *testing.T) {
	down := maths.Ray{At: maths.V(0, 4, 0), Dir: maths.V(1, -2, 0)}
	if along, ok := endOnMean(down, 0, 10); !ok || along != 2 {
		t.Fatalf("a descending ray ends at (%g, %t), want 2", along, ok)
	}
	for name, row := range map[string]struct {
		ray       maths.Ray
		near, far float64
	}{
		"before near":   {down, 3, 10},
		"after far":     {down, 0, 1},
		"level":         {maths.Ray{At: maths.V(0, 4, 0), Dir: maths.V(1, 0, 0)}, 0, 10},
		"climbing away": {maths.Ray{At: maths.V(0, 4, 0), Dir: maths.V(1, 1, 0)}, 0, 10},
	} {
		if _, ok := endOnMean(row.ray, row.near, row.far); ok {
			t.Errorf("%s: ended on the mean plane", name)
		}
	}
}

func TestSeaCountsNoRayEndedOnTheMeanWhenAllMeetTheSurface(t *testing.T) {
	sea := fixtureSea(t)
	ray := maths.Ray{At: maths.V(0, 2, 0), Dir: maths.V(0, -1, 0)}
	if _, hit := sea.Hit(ray, 0, 100); !hit {
		t.Fatal("a ray straight down missed")
	}
	if counted := sea.onMean.Load(); counted != 0 {
		t.Fatalf("%d rays ended on the mean plane, want none", counted)
	}
}
