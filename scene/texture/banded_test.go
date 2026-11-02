package texture

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
)

var (
	cream = maths.RGB(0.9, 0.85, 0.7)
	rust  = maths.RGB(0.6, 0.35, 0.2)
	bands = Banded{Paints: []maths.Light{cream, rust, cream, rust, cream}}
)

// With no roil the bands are clean: the colour depends on the latitude
// alone, the poles are the first and last paints, and halfway between two
// paints is their mean.
func TestCleanBandsFollowLatitude(t *testing.T) {
	if got := bands.At(maths.V(0, -1, 0)); got != cream {
		t.Errorf("the south pole is %v", got)
	}
	if got := bands.At(maths.V(0, 1, 0)); got != cream {
		t.Errorf("the north pole is %v", got)
	}
	// the equator is the middle paint
	for _, lon := range []float64{0, 1, 2, 3} {
		p := maths.V(math.Cos(lon), 0, math.Sin(lon))
		if got := bands.At(p); got != cream {
			t.Errorf("the equator at %v is %v", lon, got)
		}
	}
	// a quarter of the way from the south pole is the second paint
	lat := -0.5
	p := maths.V(math.Sqrt(1-lat*lat), lat, 0)
	if got := bands.At(p); math.Abs(got.R-rust.R) > 1e-9 {
		t.Errorf("latitude -0.5 is %v, want %v", got, rust)
	}
}

// Roil moves the band edges by latitude, differently along the band, and
// never beyond the paints' range.
func TestRoilMovesTheEdges(t *testing.T) {
	r := bands
	r.Roil, r.Seed = 0.2, 4
	moved, seen := 0, map[maths.Light]bool{}
	for lon := 0.0; lon < 6; lon += 0.1 {
		lat := -0.25 // the edge between the second paint and the third
		p := maths.V(math.Sqrt(1-lat*lat)*math.Cos(lon), lat,
			math.Sqrt(1-lat*lat)*math.Sin(lon))
		clean, got := bands.At(p), r.At(p)
		if got != clean {
			moved++
		}
		seen[got] = true
		if got.R < rust.R-1e-9 || got.R > cream.R+1e-9 {
			t.Fatalf("roil left the paints: %v", got)
		}
	}
	if moved < 30 || len(seen) < 10 {
		t.Errorf("%d of 60 points moved into %d colours", moved, len(seen))
	}
}

// A spot paints its heart its own colour and leaves the band alone
// outside its size.
func TestASpotHasAHeart(t *testing.T) {
	red := maths.RGB(0.7, 0.2, 0.1)
	s := bands
	s.Spots = []Spot{{Lat: 0, Lon: 0, Size: 0.1, Stretch: 1.5, Turn: 2,
		Paint: red}}
	if got := s.At(maths.V(1, 0, 0)); got != red {
		t.Errorf("the heart is %v", got)
	}
	far := maths.V(math.Cos(1), 0, math.Sin(1))
	if got := s.At(far); got != bands.At(far) {
		t.Errorf("a radian away the spot still shows: %v", got)
	}
	// the axis can point anywhere: with north along x, the old equator
	// point is now a pole
	tilted := bands
	tilted.Axis = maths.XAxis
	if got := tilted.At(maths.V(1, 0, 0)); got != cream {
		t.Errorf("tilted, the new pole is %v", got)
	}
}

// No paints is no colour.
func TestBandedNothing(t *testing.T) {
	if got := (Banded{}).At(maths.Up); got != (maths.Light{}) {
		t.Errorf("no paints gave %v", got)
	}
}

// Fine roil, jets, mottle and streets each add detail: more colours along
// a band than the plain bands have, the jets making neighbouring bands'
// roil differ, and a street putting its storms along its latitude.
func TestDetailAddsDetail(t *testing.T) {
	colours := func(b Banded, lat float64) int {
		seen := map[int]bool{}
		for lon := 0.0; lon < 6.28; lon += 0.02 {
			p := maths.V(math.Sqrt(1-lat*lat)*math.Cos(lon), lat,
				math.Sqrt(1-lat*lat)*math.Sin(lon))
			seen[int(b.At(p).R*200)] = true
		}
		return len(seen)
	}
	plain := bands
	plain.Roil, plain.Seed = 0.05, 3
	fine := plain
	fine.Fine, fine.Mottle = 0.03, 0.4
	if a, b := colours(plain, -0.22), colours(fine, -0.22); !(b > a) {
		t.Errorf("plain has %d colours along a band, with detail %d", a, b)
	}
	jets := plain
	jets.Jets = 6
	differ := 0
	for lon := 0.0; lon < 6.28; lon += 0.1 {
		for _, lat := range []float64{-0.22, -0.3} {
			p := maths.V(math.Sqrt(1-lat*lat)*math.Cos(lon), lat,
				math.Sqrt(1-lat*lat)*math.Sin(lon))
			if jets.At(p) != plain.At(p) {
				differ++
			}
		}
	}
	if differ < 20 {
		t.Errorf("the jets moved only %d samples", differ)
	}
	street := Banded{Paints: []maths.Light{cream},
		Streets: []Street{{Lat: -0.4, Count: 6, Size: 0.06, Turn: 1,
			Paint: rust, Seed: 2}}}
	hits := 0
	for lon := 0.0; lon < 6.28; lon += 0.01 {
		lat := -0.4
		p := maths.V(math.Sqrt(1-lat*lat)*math.Cos(lon), lat,
			math.Sqrt(1-lat*lat)*math.Sin(lon))
		if street.At(p) != cream {
			hits++
		}
	}
	if hits < 30 {
		t.Errorf("the street shows at only %d of 628 samples", hits)
	}
	if got := street.At(maths.V(0, 1, 0)); got != cream {
		t.Errorf("the street reaches the pole: %v", got)
	}
}
