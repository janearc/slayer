package oceantex

import (
	"math"
	"testing"

	"github.com/janearc/celestial/grissom/mesh"
	"github.com/janearc/slayer/titan"
)

// ramp is a map for testing: a sea floor that climbs steadily from the
// west to a wall at the middle, and level land behind it. Everything in
// it is invented.
type ramp struct{}

func (ramp) Source() string            { return "a test ramp, invented" }
func (ramp) Finest(mesh.Point) float64 { return 2 }
func (ramp) Height(at mesh.Point) (float64, bool, error) {
	x := math.Remainder(at.Lon(), 360) * EarthRadius * degrees
	switch {
	case x >= 0:
		return 20, true, nil
	case x > -200:
		return x / 10, true, nil // a metre under at the wall's foot... none at it
	}
	return -20, true, nil
}

func rampShore(t *testing.T) *Shore {
	t.Helper()
	ground, err := titan.New(ramp{}, EarthRadius, 1)
	if err != nil {
		t.Fatal(err)
	}
	shore, err := NewShore(ground, EarthRadius, 0, 0, 0, 600, 1)
	if err != nil {
		t.Fatal(err)
	}
	return shore
}

// The shore knows land from sea floor, and what titan made of each.
func TestShoreIsTitansGround(t *testing.T) {
	shore := rampShore(t)
	near(t, "the land", shore.Height(50, 0), 20, 0.5)
	near(t, "the deep floor", shore.Height(-280, 0), -20, 0.5)
	near(t, "the shelf", shore.Height(-100, 10), -10, 0.6)
	if !shore.Holds(0, 0) || shore.Holds(400, 0) {
		t.Errorf("the shore does not know its own square")
	}
	// the map's samples are two metres apart and the shore's one, so
	// some of the ground between them is titan's own
	var made float64
	for x := -150.0; x < -50; x += 0.37 {
		made = math.Max(made, math.Abs(shore.Made(x, 3)))
	}
	if made == 0 {
		t.Errorf("titan made none of a shelf sampled finer than its map")
	}
	if _, err := NewShore(nil, EarthRadius, 0, 0, 0, 600, 1); err == nil {
		t.Errorf("a shore with no ground was made")
	}
}

// Waves coming in over a shelf first dip a little, then grow, then are
// cut down to what the depth allows; and on land there are none.
func TestWavesShoalAndBreak(t *testing.T) {
	shore := rampShore(t)
	sea, err := SeaState(4).Raise(300e3, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	shore.meet(sea)
	gain := func(x float64) float64 { return shore.read(shore.gain, x, 0) }
	shoal := func(x float64) float64 { return shore.read(shore.shoal, x, 0) }
	near(t, "gain in twenty metres of water", gain(-280), 1, 0.1)
	if !(shoal(-20) > shoal(-100) && shoal(-20) > 1) {
		t.Errorf("waves do not grow as the water shoals: %v at 10 m, "+
			"%v at 2 m", shoal(-100), shoal(-20))
	}
	rms := sea.Height() / overRMS
	for _, x := range []float64{-30, -15, -8} {
		depth := -shore.Height(x, 0)
		if got := gain(x) * rms; got > saturated*depth*1.05 {
			t.Errorf("at %v m deep the waves are %v m, more than %v of "+
				"the depth", depth, got, saturated)
		}
	}
	if gain(-8) >= shoal(-8) {
		t.Errorf("waves in a metre of water were not cut down")
	}
	if gain(40) != 0 {
		t.Errorf("there are waves on land: %v", gain(40))
	}
	// a crest too tall for its depth is white; a low one, or one in
	// deep water, is not
	if shore.foam(-10, 0, 1.0) != 1 || shore.foam(-150, 0, 0.05) != 0 ||
		shore.foam(-280, 0, 1.0) != 0 || shore.foam(-10, 0, -1) != 0 {
		t.Errorf("foam: %v %v %v %v", shore.foam(-10, 0, 1.0),
			shore.foam(-150, 0, 0.05), shore.foam(-280, 0, 1.0),
			shore.foam(-10, 0, -1))
	}
}

// The wave number solves the dispersion relation at any depth.
func TestWaveNumber(t *testing.T) {
	for _, depth := range []float64{0.5, 3, 20, 500} {
		for _, omega := range []float64{0.3, 1, 2.5} {
			k := (Sea{Depth: depth}).number(omega)
			near(t, "dispersion", Gravity*k*math.Tanh(k*depth), omega*omega,
				1e-9)
		}
	}
	near(t, "deep water", (Sea{}).number(1), 1/Gravity, 1e-12)
}

// A wall mirrors the sea in front of it and nothing behind it.
func TestWallMirrors(t *testing.T) {
	shore := rampShore(t)
	if _, _, _, faces := shore.mirror(-3, 5); faces {
		t.Errorf("a shore with no wall mirrors")
	}
	if err := shore.Wall(Vec{}, Vec{X: -1}, 0.5); err != nil {
		t.Fatal(err)
	}
	x, y, returns, faces := shore.mirror(-3, 5)
	if !faces || x != 3 || y != 5 || returns != 0.5 {
		t.Errorf("three metres off the wall mirrors to %v, %v", x, y)
	}
	if x, _, _, _ := shore.mirror(2, 5); x != -2 {
		t.Errorf("two metres behind the wall mirrors to %v", x)
	}
	if shore.Wall(Vec{}, Vec{Z: 1}, 0.5) == nil ||
		shore.Wall(Vec{}, Vec{X: 1}, 2) == nil {
		t.Errorf("a bad wall was accepted")
	}
}

// A view over a shore shows rock where there is land, water where there
// is sea, and white where the waves are breaking.
func TestViewOfAShore(t *testing.T) {
	shore := rampShore(t)
	if err := shore.Wall(Vec{}, Vec{X: -1}, 0.5); err != nil {
		t.Fatal(err)
	}
	sea, _ := SeaState(4).Raise(300e3, 0, 0)
	scene := Scene{Sea: sea, Salinity: 35, Celsius: 15, Shore: shore,
		Sun: Vec{-0.5, -0.3, 0.6}}
	down := func(x float64) [3]float64 {
		picture, err := View(scene, Camera{
			Eye:  Subspace{Origin: Vec{X: x, Z: 80}},
			Lens: NewLens(200, Gauge{}, 0), Width: 24, Height: 24, Seed: 3})
		if err != nil {
			t.Fatal(err)
		}
		var mean [3]float64
		for _, pixel := range picture.Light {
			for i, v := range pixel {
				if math.IsNaN(v) || math.IsInf(v, 0) || v < -1e-9 {
					t.Fatalf("a pixel over x = %v is %v", x, pixel)
				}
				mean[i] += v / float64(len(picture.Light))
			}
		}
		return mean
	}
	land, deep, surf := down(60), down(-260), down(-10)
	l, _ := scene.lighting()
	rock := l.rock(0, 0)
	if math.Abs(land[1]-rock[1]) > 0.1*rock[1] {
		t.Errorf("level land is %v, level rock %v", land, rock)
	}
	if !(deep[2] > deep[0] && deep[1] < land[1]) {
		t.Errorf("deep water is %v, land %v", deep, land)
	}
	if !(surf[1] > 1.5*deep[1]) {
		t.Errorf("the surf, %v, is no whiter than deep water, %v", surf, deep)
	}
}
