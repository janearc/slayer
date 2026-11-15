package finish

import (
	"crypto/sha256"
	"encoding/hex"
	"image"
	"math"
	"os"
	"strings"
	"testing"

	"github.com/janearc/libtheme-css/spaces/ok"
	"github.com/janearc/libtheme-css/spaces/srgb"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/ninjafilm"
	ninjacolour "github.com/janearc/slayer/ninjafilm/colour"
	"github.com/janearc/slayer/render/frame"
)

// what the shoulder tests choose, each picked by the fable-team stock
// slice, 2026-11-12, unless it says it is defined
const (
	// rampSteps is how many pixels a ramp runs over. Few, so that each
	// pixel is wide on the film and its grain small beside a step, and
	// odd, so that the middle one is a white card's light.
	rampSteps = 49
	// rampDarkest and rampBrightest are where a ramp starts and ends,
	// in times the light of a white card: a thousandth to a thousand.
	rampDarkest   = 1e-3
	rampBrightest = 1e3
	// flameRed, flameGreen and flameBlue are an orange: a flame's
	// colour, red over green over a little blue.
	flameRed   = 1.0
	flameGreen = 0.35
	flameBlue  = 0.05
	// fireGreen and fireBlue are the night fire's own colour against its
	// red: measured, the brightest pixel of the short's fire frame,
	// linear (17.17, 3.897, 0.0002315), on 2026-11-12. It has next to
	// no blue.
	fireGreen = 3.897 / 17.17
	fireBlue  = 0.0002315 / 17.17
	// paper is the byte of the screen's white. It is defined by the
	// eight bits of a byte.
	paper = 255
	// degrees is half a turn of hue, in degrees. It is defined.
	degrees = 180
	// riseLeast is the least lightness, on OK's scale of nought to one,
	// a brighter step must gain over a dimmer one: a thousandth, a
	// twentieth of what an eye tells apart (ok.Eye), above what the
	// grain left in a mean of rampRows rows moves it.
	riseLeast = 1e-3
	// rampRows is how many rows each step of a ramp is drawn in, so
	// that the grain of a step is seen as its mean.
	rampRows = 16
	// paleShare is how much of its chroma at white an orange may keep
	// once its hue is nearer yellow's: a pale yellow on the way to
	// white is how film shows a flame, a strong one is the clamp.
	paleShare = 0.5
)

// flame is an orange with a little blue, which every stock can print
// white. fire is the night fire's colour, with next to no blue: no
// colour stock clears its blue layer for it, so its brightest prints a
// pale yellow, not white.
var (
	flame = maths.RGB(flameRed, flameGreen, flameBlue)
	fire  = maths.RGB(flameRed, fireGreen, fireBlue)
)

// ramp is rampSteps columns of one light, from rampDarkest to
// rampBrightest times it, each a like step brighter than the last, and
// rampRows rows of each.
func ramp(light maths.Light) *frame.Frame {
	made := frame.New(rampSteps, rampRows)
	growth := math.Log(rampBrightest/rampDarkest) / (rampSteps - one)
	for x := range rampSteps {
		amount := light.Scale(rampDarkest * math.Exp(growth*float64(x)))
		for y := range rampRows {
			pixel := made.At(x, y)
			pixel.R, pixel.G = float32(amount.R), float32(amount.G)
			pixel.B = float32(amount.B)
			made.Set(x, y, pixel)
		}
	}
	return made
}

// looks is each column of a picture as the eye sees it: the mean of its
// rows in linear light, in OK's lightness, chroma and hue.
func looks(picture *image.RGBA) []ok.OKLCH {
	var seen []ok.OKLCH
	bounds := picture.Bounds()
	for x := range bounds.Dx() {
		var total maths.Light
		for y := range bounds.Dy() {
			at := picture.RGBAAt(x, y)
			total = total.Add(maths.RGB(ninjacolour.Decode(at.R),
				ninjacolour.Decode(at.G), ninjacolour.Decode(at.B)))
		}
		mean := total.Scale(whole / float64(bounds.Dy()))
		colour := srgb.FromLight(mean.R, mean.G, mean.B)
		seen = append(seen, ok.FromSwatch(colour).Polar())
	}
	return seen
}

// white is whether a look is the paper's white, as near as an eye can
// tell.
func white(look ok.OKLCH) bool {
	return ok.Distance(look.Rect(), ok.White) < ok.Eye
}

// throughStock is a frame developed through a stock at its rating, by
// the developer a job of that stock makes.
func throughStock(t *testing.T, name string, light *frame.Frame) *image.RGBA {
	t.Helper()
	made, err := stockDeveloper(name, none, none)
	if err != nil {
		t.Fatal(err)
	}
	picture, err := made.Develop(none, []*frame.Frame{light})
	if err != nil {
		t.Fatal(err)
	}
	return picture
}

// throughClamp is the same, through the screen the stocks had before
// the shoulder: the exposure and no knee.
func throughClamp(t *testing.T, name string, light *frame.Frame) *image.RGBA {
	t.Helper()
	made, err := stockDeveloper(name, none, none)
	if err != nil {
		t.Fatal(err)
	}
	made.stock.screen = ninjacolour.NewTone(transmission[name], none, none,
		none)
	picture, err := made.Develop(none, []*frame.Frame{light})
	if err != nil {
		t.Fatal(err)
	}
	return picture
}

// plateau is the first pair of steps, from the white card's up, where
// the brighter is not lighter than the dimmer by riseLeast and the two
// are not both white. It is -1, -1 if there is none.
func plateau(seen []ok.OKLCH) (dimmer, brighter int) {
	for x := whiteStep(); x < len(seen); x++ {
		for y := whiteStep(); y < x; y++ {
			if !(seen[x].L > seen[y].L+riseLeast) && !(white(seen[x]) &&
				white(seen[y])) {
				return y, x
			}
		}
	}
	return -one, -one
}

// whiteStep is the step of a ramp that is the light of a white card.
// It is defined by the ramp: a thousandth to a thousand, the middle
// step of an odd count is one.
func whiteStep() int { return (rampSteps - one) / pair }

// An orange from a white card to a thousand times it prints through
// every stock lighter at every step than at every dimmer one, until
// both are the paper's white, and a thousand times white is white. A
// stock's toe below the card is its own.
func TestARampRisesToThePapersWhite(t *testing.T) {
	stocks, err := ninjafilm.Stocks()
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range stocks {
		seen := looks(throughStock(t, name, ramp(flame)))
		if dimmer, brighter := plateau(seen); dimmer >= none {
			t.Errorf("%s prints step %d as %v, no lighter than step %d, "+
				"%v; from white: %v", name, brighter, seen[brighter], dimmer,
				seen[dimmer], seen[whiteStep():])
		}
		if top := seen[rampSteps-one]; !white(top) {
			t.Errorf("%s prints a thousand times white as %v", name, top)
		}
	}
}

// Anti: through the screen the stocks had, with no knee, the night
// fire's colour stops on a flat yellow short of white, as the short's
// flames did: the test can see a plateau.
func TestTheClampPrintsARampFlat(t *testing.T) {
	seen := looks(throughClamp(t, "lantern", ramp(fire)))
	if dimmer, _ := plateau(seen); dimmer < none {
		t.Errorf("the clamp printed the ramp with no plateau; the ramp "+
			"test cannot tell a clamp from a shoulder: %v",
			seen[whiteStep():])
	}
}

// hueFrom is how far round the wheel one hue is from another, in
// degrees, the short way.
func hueFrom(hue, other float64) float64 {
	apart := math.Mod(math.Abs(hue-other), degrees+degrees)
	return math.Min(apart, degrees+degrees-apart)
}

// yellow is the hue of the screen's yellow, red and green at full.
func yellow() float64 {
	return ok.FromSwatch(srgb.RGB8(paper, paper, none).Swatch()).Polar().H
}

// turned is the first step of a ramp, as seen, whose hue is nearer
// yellow's than the ramp's at white while it keeps more than paleShare
// of its chroma at white, or -1 if there is none.
func turned(seen []ok.OKLCH, start ok.OKLCH) int {
	toward := yellow()
	for x, look := range seen {
		nearer := hueFrom(look.H, toward) < hueFrom(look.H, start.H)
		if nearer && look.C > paleShare*start.C {
			return x
		}
	}
	return -one
}

// A bright orange does not turn into a strong yellow on the way up
// through lantern: where its hue has gone nearer yellow's than its own
// at white, it has bleached to a pale colour on its way to white.
func TestABrightOrangeDoesNotTurnYellow(t *testing.T) {
	seen := looks(throughStock(t, "lantern", ramp(flame)))
	start := seen[whiteStep()]
	if x := turned(seen, start); x >= none {
		t.Errorf("step %d is %v, a yellow; at white it was %v", x,
			seen[x], start)
	}
}

// Anti: through the clamp the same orange turns to a strong yellow, so
// the test can see a turn.
func TestTheClampTurnsOrangeYellow(t *testing.T) {
	seen := looks(throughClamp(t, "lantern", ramp(flame)))
	if turned(seen, seen[whiteStep()]) < none {
		t.Error("the clamp kept the orange; the hue test cannot tell")
	}
}

// A print that passes more than white gets a shoulder: its knee is the
// share of its stops above white, that many below, and its top is the
// clearest at the exposure.
func TestAScreenLandsTheClearestOnWhite(t *testing.T) {
	const exposure, passes = 4.0, 0.5 // picked: a lantern-like print
	screen := screenOf(exposure, maths.Grey(passes))
	top := exposure * passes
	want := ninjacolour.NewTone(exposure, math.Pow(top, -shoulderShare),
		top, none)
	if screen != want {
		t.Errorf("the screen is %v, want %v", screen, want)
	}
	if got := screen.Squeeze(maths.Grey(top)); got != maths.Grey(whole) {
		t.Errorf("the clearest lands on %v, not white", got)
	}
}

// Anti: a print that passes no more than white has no shoulder; its
// screen is the clamp it always was, and light under white is as it is.
func TestAScreenWithNothingPastWhiteHasNoShoulder(t *testing.T) {
	const exposure, passes = 2.0, 0.5 // picked: white exactly
	screen := screenOf(exposure, maths.Grey(passes))
	if want := ninjacolour.NewTone(exposure, none, none, none); screen != want {
		t.Errorf("the screen is %v, want the clamp %v", screen, want)
	}
}

// unchangedFile holds, by stock, the sha256 of the picture each stock
// made of the gradient at unchangedScale before the screen had a
// shoulder. It was recorded on 2026-11-12 from the clamp.
const unchangedFile = "testdata/unchanged.csv"

// unchangedScale is the gradient's light times this: nothing in any
// stock's print of it passes the screen's white. It is picked by the
// fable-team stock slice, 2026-11-12: the brightest part is 0.87 of
// white through dusk and lantern.
const unchangedScale = 0.15

// A picture with nothing past white prints through every stock byte
// for byte as it did before the shoulder.
func TestAPictureUnderWhitePrintsAsBefore(t *testing.T) {
	text, err := os.ReadFile(unchangedFile)
	if err != nil {
		t.Fatal(err)
	}
	rows := strings.Split(strings.TrimSpace(string(text)), "\n")[one:]
	stocks, err := ninjafilm.Stocks()
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != len(stocks) {
		t.Fatalf("%d stocks recorded, the shelf holds %d", len(rows),
			len(stocks))
	}
	light := scaled(gradient(side, side), unchangedScale)
	for _, row := range rows {
		name, want, _ := strings.Cut(row, ",")
		sum := sha256.Sum256(throughStock(t, name, light).Pix)
		if got := hex.EncodeToString(sum[:]); got != want {
			t.Errorf("%s prints %s, before %s", name, got, want)
		}
	}
}

// Anti: the record can tell a change. The same picture a little past
// white, through lantern, is not what lantern printed under white.
func TestThePictureRecordSeesAChange(t *testing.T) {
	light := scaled(gradient(side, side), unchangedScale+unchangedScale)
	under := throughStock(t, "lantern", scaled(gradient(side, side),
		unchangedScale))
	if string(throughStock(t, "lantern", light).Pix) == string(under.Pix) {
		t.Error("twice the light printed the same picture")
	}
}
