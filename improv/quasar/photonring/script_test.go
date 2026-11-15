package photonring

import (
	"math"
	"os"
	"slices"
	"testing"

	yaml "gopkg.in/yaml.v3"

	"github.com/janearc/slayer/improv/quasar/facts"
	"github.com/janearc/slayer/improv/quasar/rings"
)

// The numbers the tests choose: the quasar slice, 2026-11-12.
const (
	scriptFile = "photon-ring.yaml"
	// the smoke's geometry and the short's pictures, as ringshots draws
	// them: defined by smoke 36 and ringshots' picks
	smokeEye    = 50
	smokeLean   = 10 * math.Pi / 180
	smokeInner  = 6
	smokeOuter  = 60
	smokeWide   = 40 * math.Pi / 180
	picture     = 1080
	closeInWide = 0.384717 * math.Pi / 180
	closeInAim  = 0.105546525
	orders      = 3
	half        = 0.5
	// pixelSlack is how far a label may sit from the measured pixel
	pixelSlack = 1.0
	// places the pixel figures are written to
	wholePixels = 0
	tenthPixels = 1
)

type script struct {
	Shorts []struct {
		Shots []struct {
			Picture  string `yaml:"picture"`
			Sentence string `yaml:"sentence"`
			Figures  []struct {
				Value string `yaml:"value"`
				From  string `yaml:"from"`
			} `yaml:"figures"`
			Labels []struct {
				Pixel []float64 `yaml:"pixel"`
				Words string    `yaml:"words"`
			} `yaml:"labels"`
		} `yaml:"shots"`
	} `yaml:"shorts"`
}

func readScript(t *testing.T) script {
	t.Helper()
	raw, err := os.ReadFile(scriptFile)
	if err != nil {
		t.Fatal(err)
	}
	var read script
	if err := yaml.Unmarshal(raw, &read); err != nil {
		t.Fatal(err)
	}
	return read
}

func measured(t *testing.T) []rings.Band {
	t.Helper()
	lens, err := rings.NewLens(smokeEye, smokeLean)
	if err != nil {
		t.Fatal(err)
	}
	disc, err := rings.NewDisc(smokeInner, smokeOuter)
	if err != nil {
		t.Fatal(err)
	}
	bands, err := lens.Bands(disc, orders)
	if err != nil {
		t.Fatal(err)
	}
	return bands
}

// known is every number the script may say, by its from.
func known(t *testing.T, bands []rings.Band) map[string]facts.Number {
	t.Helper()
	camera, err := rings.NewCamera(smokeWide, picture)
	if err != nil {
		t.Fatal(err)
	}
	second, err := facts.Computed(camera.Pixels(bands[1]), wholePixels,
		"rings second image width")
	if err != nil {
		t.Fatal(err)
	}
	third, err := facts.Computed(camera.Pixels(bands[2]), tenthPixels,
		"rings third image width")
	if err != nil {
		t.Fatal(err)
	}
	table := map[string]facts.Number{}
	for _, number := range []facts.Number{facts.SubringPercent(),
		facts.M87Ring(), facts.SgrARing(), second, third} {
		table[number.From()] = number
	}
	return table
}

func TestEverySentenceSaysOnlyItsSourcedFigures(t *testing.T) {
	table := known(t, measured(t))
	for _, short := range readScript(t).Shorts {
		for _, shot := range short.Shots {
			var numbers []facts.Number
			for _, figure := range shot.Figures {
				number, found := table[figure.From]
				if !found || number.Shown() != figure.Value {
					t.Errorf("figure %s from %q is not a facts "+
						"number written so (%v)", figure.Value,
						figure.From, number.Shown())
					continue
				}
				if numeralIn(shot.Sentence, number.Shown()) {
					numbers = append(numbers, number)
				}
			}
			if _, err := facts.NewSentence(shot.Sentence,
				numbers...); err != nil {
				t.Error(err)
			}
		}
	}
}

// numeralIn is whether a sentence says a numeral, as facts finds them.
func numeralIn(sentence, shown string) bool {
	return slices.Contains(facts.Numerals(sentence), shown)
}

func TestTheLabelsSitWhereTheMarchPutsTheImages(t *testing.T) {
	bands := measured(t)
	full, err := rings.NewCamera(smokeWide, picture)
	if err != nil {
		t.Fatal(err)
	}
	close, err := rings.NewCamera(closeInWide, picture)
	if err != nil {
		t.Fatal(err)
	}
	middle := func(band rings.Band) float64 {
		return band.Near() + half*band.Width()
	}
	want := map[string]float64{
		"the far side of the disc": picture*half -
			full.Offset(middle(bands[0])),
		"where the ring appears": picture*half -
			full.Offset(middle(bands[1])),
		"the second image": picture*half -
			close.Offset(middle(bands[1])-closeInAim),
		"the third image": picture*half -
			close.Offset(middle(bands[2])-closeInAim),
	}
	for _, short := range readScript(t).Shorts {
		for _, shot := range short.Shots {
			for _, label := range shot.Labels {
				row, found := want[label.Words]
				if !found {
					t.Errorf("label %q has no measured place",
						label.Words)
					continue
				}
				if math.Abs(label.Pixel[1]-row) > pixelSlack {
					t.Errorf("%q is at row %v, the march puts "+
						"it at %.1f", label.Words,
						label.Pixel[1], row)
				}
			}
		}
	}
}
