package thing

import (
	"testing"

	"github.com/janearc/celestial/radiation"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/medium"
)

// constructionLooksFile is the table of looks these tests make: a colour
// of three shares, a gloss and an index, by name.
const constructionLooksFile = "testdata/construction-looks.csv"

// constructionLook is the colour, gloss and index a row of the looks
// table names.
func constructionLook(t *testing.T, name string) (colour maths.Light, gloss, index float64) {
	t.Helper()
	row := readSheet(t, constructionLooksFile).row(t, constructionLooksFile, name)
	colour = maths.RGB(cell(t, row, "red"), cell(t, row, "green"),
		cell(t, row, "blue"))
	return colour, cell(t, row, "gloss"), cell(t, row, "index")
}

// A diffuse look is its name and its colour and nothing else; a polished
// one is those and its gloss and its index. Neither is a metal, wears a
// film or glows.
func TestDiffuseAndPolishedAreWhatTheyAreGiven(t *testing.T) {
	red, gloss, index := constructionLook(t, "brick")
	if got, want := *Diffuse("brick", red), (Material{Name: "brick",
		Colour: red}); got != want {
		t.Errorf("a diffuse look is %+v, want %+v", got, want)
	}
	polished := *Polished("lacquer", red, gloss, index)
	if want := (Material{Name: "lacquer", Colour: red, Gloss: gloss,
		Index: index}); polished != want {
		t.Errorf("a polished look is %+v, want %+v", polished, want)
	}
	// the anti-test: a polished look is the diffuse one with a gloss and an
	// index added, and a diffuse look has neither
	diffuse := *Diffuse("lacquer", red)
	if diffuse.Gloss != nought || diffuse.Index != nought {
		t.Errorf("a diffuse look glosses %v at index %v", diffuse.Gloss,
			diffuse.Index)
	}
	if polished == diffuse {
		t.Error("a polished look is the diffuse one")
	}
	polished.Gloss, polished.Index = nought, nought
	if polished != diffuse {
		t.Errorf("a polished look differs from the diffuse one by more than "+
			"its gloss and index: %+v, %+v", polished, diffuse)
	}
}

// A new volume is its medium in its look. The anti-test: it is still,
// at the world's zero, with no motion and no beam until a caller gives
// it one.
func TestANewVolumeIsStillAndCarriesWhatItIsGiven(t *testing.T) {
	grey, _, _ := constructionLook(t, "mist")
	look := Diffuse("mist", grey)
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
	share := picked(t, "construction-returned-share")
	returns := Spectrum{Returns: map[radiation.Range]float64{
		radiation.XRay(): share}}
	gamma := maths.NewBand(picked(t, "construction-gamma-nanometres"))
	xray := maths.NewBand(picked(t, "construction-xray-nanometres"))
	if returns.Scatters(xray) != share || returns.Scatters(gamma) != nought {
		t.Errorf("scatters: x-ray %v, gamma %v", returns.Scatters(xray),
			returns.Scatters(gamma))
	}
}
