package arid

import (
	"math"
	"testing"

	"github.com/janearc/slayer/oceantex"
)

// gridded is a field of count samples a side over extent metres with
// every grid there, all nought, for a test to fill.
func gridded(count int, extent float64) *oceantex.Field {
	cells := count * count
	return &oceantex.Field{N: count, Extent: extent,
		Height: make([]float64, cells), SlopeX: make([]float64, cells),
		SlopeY: make([]float64, cells), PushX: make([]float64, cells),
		PushY: make([]float64, cells), StretchXX: make([]float64, cells),
		StretchXY: make([]float64, cells),
		StretchYY: make([]float64, cells)}
}

// carrying is a sea of two bands: a long one that stands flat and has
// carried its water push metres east everywhere, and a short one of one
// wave along the east.
func carrying(t *testing.T, push float64) (*Sea, *oceantex.Field) {
	t.Helper()
	long, short := gridded(8, 64), gridded(16, 8)
	for index := range long.PushX {
		long.PushX[index] = push
	}
	for row := range short.N {
		for column := range short.N {
			short.Height[row*short.N+column] = math.Cos(2 * math.Pi *
				float64(column) / float64(short.N))
		}
	}
	sea, err := seaOf([]oceantex.WaveCascade{{Field: short}, {Field: long}})
	if err != nil {
		t.Fatal(err)
	}
	return sea, short
}

// A short band is read where the long band's water came from: with the
// long band's water carried 1.5 m east, the sea's height at a place is
// the short band's 1.5 m west of it. The bands are given shortest first,
// and are nested longest first all the same.
func TestWavesReadShortBandsWhereTheLongWaterCameFrom(t *testing.T) {
	const push = 1.5
	sea, short := carrying(t, push)
	for index := range 40 {
		east, south := float64(index)*0.37, float64(index)*-0.21
		want := periodicSample(short, short.Height,
			short.Extent/2+east-push, short.Extent/2-south)
		if got := sea.waves(east, south); math.Abs(got-want) > 1e-9 {
			t.Fatalf("at %g, %g: %g, want the short band's %g", east, south,
				got, want)
		}
	}
}

// Anti: water that is not carried is read where it is.
func TestWavesReadShortBandsInPlaceWithNoPush(t *testing.T) {
	sea, short := carrying(t, 0)
	for index := range 40 {
		east, south := float64(index)*0.37, float64(index)*-0.21
		want := periodicSample(short, short.Height, short.Extent/2+east,
			short.Extent/2-south)
		if got := sea.waves(east, south); math.Abs(got-want) > 1e-9 {
			t.Fatalf("at %g, %g: %g, want %g", east, south, got, want)
		}
	}
}

// The nested surface rises no faster than the bound the search steps by.
func TestNestedSurfaceRisesNoFasterThanItsBound(t *testing.T) {
	sea := fixtureSea(t)
	const apart = 0.01
	for index := range 4000 {
		east := float64(index)*1.618 - 900
		south := float64(index)*-0.731 + 300
		turn := float64(index)
		rise := sea.waves(east+apart*math.Cos(turn),
			south+apart*math.Sin(turn)) - sea.waves(east, south)
		if math.Abs(rise) > sea.steepest*apart*(1+1e-9) {
			t.Fatalf("at %g, %g: rises %g in %g m, bound %g a metre", east,
				south, rise, apart, sea.steepest)
		}
	}
}

// The slope a hit is shaded by is the nested surface's own, to the
// bands' sampling: over many places the two agree in the mean square to
// a fifth.
func TestSlopeIsTheNestedSurfacesOwn(t *testing.T) {
	sea := fixtureSea(t)
	measure := sea.Measure()
	if !(measure.MetSlope > 0) ||
		math.Abs(measure.ShadedSlope-measure.MetSlope) > 0.2*measure.MetSlope {
		t.Errorf("shaded slope %g, met slope %g", measure.ShadedSlope,
			measure.MetSlope)
	}
}

// The coast's steepness near a place covers every cell of its tile and
// the tiles beside it, and is never over the whole coast's.
func TestSteepestNearCoversItsTilesAndIsWithinTheCoasts(t *testing.T) {
	ground := shaped(40, 30, 4, basinHeight)
	shore, err := newCoast(ground, -20, -12, 0, 1)
	if err != nil {
		t.Fatal(err)
	}
	whole := shore.steepest()
	for row := range shore.down - 1 {
		for column := range shore.across - 1 {
			east := shore.east + (float64(column)+0.5)*shore.step
			south := shore.south + (float64(row)+0.5)*shore.step
			near := shore.steepestNear(east, south)
			if near < shore.cellSteepest(column, row) || near > whole {
				t.Fatalf("cell %d, %d: near %g, its own %g, the coast's %g",
					column, row, near, shore.cellSteepest(column, row), whole)
			}
		}
	}
}

// Anti: a level bed changes no wave's share anywhere, so nothing near any
// place is steep, on the grid or off it.
func TestSteepestNearIsNoughtOverALevelBed(t *testing.T) {
	shore, err := newCoast(shaped(20, 20, 4, deepHeight), 0, 0, 0, 1)
	if err != nil {
		t.Fatal(err)
	}
	for _, place := range [][2]float64{{10, 10}, {-500, 3}, {40, 9000}} {
		if near := shore.steepestNear(place[0], place[1]); near != 0 {
			t.Errorf("at %v: %g", place, near)
		}
	}
}

// folding is a sea of one band whose carrying has squeezed the water to
// squeeze of its resting width both ways, everywhere.
func folding(squeeze float64) *Sea {
	band := gridded(4, 8)
	for index := range band.StretchXX {
		band.StretchXX[index] = 1 - 1/squeeze
		band.StretchYY[index] = 1 - 1/squeeze
	}
	return &Sea{bands: []*oceantex.Field{band}, slopes: []float64{0},
		finest: 2, period: 8}
}

// Water squeezed to 0.7 of its width both ways falls at 0.6 of gravity,
// past the threshold: all of it is breaking.
func TestWhitecapIsAllWhereTheWaterFallsPastTheThreshold(t *testing.T) {
	sea := folding(0.7)
	if falling := sea.falling(1, 2); math.Abs(falling-0.6) > 1e-9 {
		t.Errorf("falling at %g of gravity, want 0.6", falling)
	}
	if cover := sea.whitecap(1, 2); cover != 1 {
		t.Errorf("breaking over %g of the water", cover)
	}
}

// Anti: still water breaks nowhere, and water squeezed a little, under
// the threshold, breaks nowhere either.
func TestWhitecapIsNoneOnStillAndGentleWater(t *testing.T) {
	for _, sea := range []*Sea{folding(1), folding(0.9)} {
		for index := range 20 {
			east, south := float64(index)*0.7, float64(index)*-1.3
			if cover := sea.whitecap(east, south); cover != 0 {
				t.Fatalf("at %g, %g: breaking over %g", east, south, cover)
			}
		}
	}
}

// The grain that stands for the unseen waves has a mean of nought and a
// variance of one.
func TestGrainHasAMeanOfNoughtAndAVarianceOfOne(t *testing.T) {
	sea := folding(1)
	var sum, squares float64
	const places = 40000
	for index := range places {
		value := sea.grain(float64(index)*1.37, float64(index)*-0.91)
		sum += value
		squares += value * value
	}
	mean, variance := sum/places, squares/places
	if math.Abs(mean) > 0.05 || math.Abs(variance-1) > 0.1 {
		t.Errorf("mean %g, variance %g", mean, variance)
	}
}

// A sea under a wind that leaves it like a mirror says no unseen slope;
// one under a working wind says Cox and Munk's less its bands'.
func TestUnseenIsNoughtUnderAMirrorCalmWind(t *testing.T) {
	calm, err := NewSea(oceantex.Sea{Wind: 0.5, Fetch: 30}, 0, 7)
	if err != nil {
		t.Fatal(err)
	}
	if calm.Unseen() != 0 {
		t.Errorf("a calm sea's unseen slope is %g", calm.Unseen())
	}
	if windy := fixtureSea(t); !(windy.Unseen() > 0.01) {
		t.Errorf("a 7 m/s sea's unseen slope is %g", windy.Unseen())
	}
}
