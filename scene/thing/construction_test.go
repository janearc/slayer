package thing

import (
	"testing"

	"github.com/janearc/celestial/radiation"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/medium"
)

// A diffuse look is its name and its colour and nothing else; a polished
// one is those and its gloss and its index. Neither is a metal, wears a
// film or glows.
func TestDiffuseAndPolishedAreWhatTheyAreGiven(t *testing.T) {
	red := maths.RGB(0.8, 0.1, 0.1)
	if got, want := *Diffuse("brick", red), (Material{Name: "brick",
		Colour: red}); got != want {
		t.Errorf("a diffuse look is %+v, want %+v", got, want)
	}
	if got, want := *Polished("lacquer", red, 0.9, 1.5), (Material{
		Name: "lacquer", Colour: red, Gloss: 0.9, Index: 1.5}); got != want {
		t.Errorf("a polished look is %+v, want %+v", got, want)
	}
	// the anti-test: the two are not one another
	if *Diffuse("brick", red) == *Polished("brick", red, 0.9, 1.5) {
		t.Error("a polished look is the diffuse one")
	}
}

// A new volume is its medium in its look. The anti-test: it is still,
// at the world's zero, with no motion and no beam until a caller gives
// it one.
func TestANewVolumeIsStillAndCarriesWhatItIsGiven(t *testing.T) {
	look := Diffuse("mist", maths.Grey(0.5))
	made := NewVolume(medium.Fizz{}, look)
	if _, fizz := made.Medium.(medium.Fizz); !fizz || made.Material != look {
		t.Errorf("the volume carries %T in %v", made.Medium, made.Material)
	}
	if made.Motion != nil || made.Beam != nil {
		t.Errorf("a new volume moves by %v and beams by %v", made.Motion,
			made.Beam)
	}
}

// A spectrum sends back what it says it returns in a range. The
// anti-test: in a range it does not name it sends back nothing.
func TestASpectrumScattersByRange(t *testing.T) {
	s := Spectrum{Returns: map[radiation.Range]float64{radiation.XRay(): 0.25}}
	gamma, xray := maths.Band{Nanometres: 2e-4}, maths.Band{Nanometres: 0.5}
	if s.Scatters(xray) != 0.25 || s.Scatters(gamma) != 0 {
		t.Errorf("scatters: x-ray %v, gamma %v", s.Scatters(xray),
			s.Scatters(gamma))
	}
}
