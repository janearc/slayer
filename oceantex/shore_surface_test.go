package oceantex

import (
	"math"
	"testing"
)

func TestShoreSurfaceSamplesExistingShoreModel(t *testing.T) {
	shore := &Shore{extent: 100, step: 50, across: 3,
		height: []float64{-10, -2, 2, -10, -2, 2, -10, -2, 2}}
	sea := Sea{Wind: 6, Fetch: 5000, Depth: 10, Toward: 0}
	snapshot, err := NewShoreSurface(shore, sea, 1, 42)
	if err != nil {
		t.Fatal(err)
	}
	for east := -45.0; east <= 45; east += 5 {
		height, foam, ok := snapshot.Sample(east, 0)
		if !ok {
			t.Fatalf("no shore sample at %g", east)
		}
		wave := snapshot.waves.arriving(east, 0, 0, 3)
		wantHeight := wave * snapshot.waves.shore.read(snapshot.waves.shore.gain, east, 0)
		wantFoam := snapshot.waves.shore.foam(east, 0, wave)
		if height != wantHeight || foam != wantFoam || foam < 0 || foam > 1 {
			t.Fatalf("sample %g: height=%g foam=%g, want %g and %g", east, height, foam, wantHeight, wantFoam)
		}
	}
	if _, _, ok := snapshot.Sample(51, 0); ok {
		t.Fatal("accepted point outside shore")
	}
	if _, _, ok := snapshot.Sample(math.NaN(), 0); ok {
		t.Fatal("accepted nonfinite point")
	}
	if !snapshot.Wet(-45, 0) || snapshot.Wet(45, 0) ||
		snapshot.Wet(51, 0) || snapshot.Wet(math.NaN(), 0) {
		t.Fatal("shore wet test did not distinguish water, land and invalid points")
	}
	if shore.gain != nil {
		t.Fatal("sampling changed the caller's shore")
	}
}

func TestShoreSurfaceRejectsCalmOrInvalidInputs(t *testing.T) {
	shore := &Shore{extent: 1, step: 1, across: 2, height: []float64{-1, -1, -1, -1}}
	for _, sea := range []Sea{{}, {Wind: math.NaN(), Fetch: 1}, {Wind: 1, Fetch: 1, Depth: -1}} {
		if _, err := NewShoreSurface(shore, sea, 0, 1); err == nil {
			t.Fatalf("accepted invalid sea %+v", sea)
		}
	}
}

func TestShoreSurfaceRejectsUnconstructedShore(t *testing.T) {
	sea := Sea{Wind: 6, Fetch: 5000, Depth: 10}
	for _, shore := range []*Shore{nil, {}} {
		if _, err := NewShoreSurface(shore, sea, 0, 1); err == nil {
			t.Fatal("accepted nil or unconstructed shore")
		}
	}
}
