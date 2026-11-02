package ninjafilm_test

import (
	"math"
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/janearc/libtheme-css/spaces/ok"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/ninjafilm"
	"github.com/janearc/slayer/ninjafilm/data"
	"github.com/janearc/slayer/ninjafilm/film"
	"github.com/janearc/slayer/ninjafilm/frame"
)

// what these tests count in, each defined
const (
	lit      = 1      // defined: a lamp lit to the whole of a light
	dark     = 0      // defined: a lamp not lit at all
	first    = 0      // defined: the number of the first frame of a roll, and the first place of a frame
	halfway  = 0.5    // defined: the lightness of the middle grey, half of libtheme's scale
	quartern = 4      // defined: the shadow of a disc is pi times its width squared over four
	metric   = 1000.0 // defined: a millimetre is 1000 micrometres by the metric prefixes
	whole    = 1      // defined: the whole of a share
)

// the table of the exposures doc/filming.md gives for each stock
const exposuresFile = "testdata/exposures.csv"

// what these tests choose, each picked in picked-numbers.yml
var (
	seed         = uint64(data.Whole("roll-grainy-seed"))
	clearLeast   = data.Picked("stocks-clear-least")
	darkRatio    = data.Picked("stocks-dark-ratio")
	colourRatio  = data.Picked("stocks-colour-ratio")
	leafRatio    = data.Picked("stocks-leaf-ratio")
	leafBlue     = data.Picked("stocks-leaf-blue")
	whiteError   = data.Picked("stocks-white-error")
	mosaicPasses = data.Picked("stocks-mosaic-passes")
	mosaicError  = data.Picked("stocks-mosaic-error")
	mosaicPixels = data.Whole("stocks-mosaic-pixels")
	mosaicLamp   = data.Picked("stocks-mosaic-lamp")
	mosaicRed    = data.Picked("stocks-mosaic-red")
	guideLeast   = data.Picked("stocks-guide-least")
	guideMost    = data.Picked("stocks-guide-most")
	grainError   = data.Picked("stocks-grain-ratio-error")
	spreadAcross = data.Whole("roll-spread-across")
	spreadDown   = data.Whole("roll-spread-down")
	spreadError  = data.Picked("roll-spread-error")
)

// shine is a light of the three lamps lit as given.
func shine(red, green, blue float64) maths.Light {
	return maths.RGB(red, green, blue)
}

// uniform is a frame of columns by rows with the same light in every
// pixel, covered and five away.
func uniform(columns, rows int, light maths.Light) *frame.Frame {
	made := frame.New(columns, rows)
	for index := range made.Pix {
		made.Pix[index] = frame.NewPixel(float32(light.R),
			float32(light.G), float32(light.B), lit, float32(columns))
	}
	return made
}

// grey is the luminance of the grey halfway up libtheme's lightness,
// which a stock is rated so that it lands on the middle of its curve.
func grey() float64 {
	_, luminance, _ := ok.Grey(halfway).Swatch().XYZ()
	return luminance
}

// still is what the stock of a name makes of a light, printed so large
// that no grain shows: one pixel of one glimpse of the light.
func still(t *testing.T, name string, light maths.Light) maths.Light {
	t.Helper()
	made, err := ninjafilm.Roll(name, seed)
	if err != nil {
		t.Fatal(err)
	}
	picture := uniform(lit, lit, light)
	got, err := made.Print(first, []*frame.Frame{picture})
	if err != nil {
		t.Fatal(err)
	}
	pixel := got.At(first, first)
	return pixel.Light()
}

// within is whether two numbers are within a tolerance of each other.
func within(one, other, tolerance float64) bool {
	return math.Abs(one-other) <= tolerance
}

// rowOf is the first record of the stock of a name in the table as it
// ships, so that a row changed in the table changes what the tests
// expect with it.
func rowOf(t *testing.T, name string) (found data.Row) {
	t.Helper()
	rows, err := data.Parse(data.RecordsTable())
	if err != nil {
		t.Fatal(err)
	}
	for _, row := range rows {
		if stock, _ := row.Text("stock"); stock == name {
			return row
		}
	}
	t.Fatalf("no record is for %q", name)
	return found
}

// midDensity is the density midway between the least and the most of
// the curve of the first record of a stock.
func midDensity(t *testing.T, name string) float64 {
	t.Helper()
	row := rowOf(t, name)
	least, err := row.Float("least")
	if err != nil {
		t.Fatal(err)
	}
	most, err := row.Float("most")
	if err != nil {
		t.Fatal(err)
	}
	return (least + most) * halfway
}

// passed is the share of light a density lets past.
func passed(density float64) float64 {
	return math.Exp(-density * math.Ln10)
}

func TestTheShelfHoldsTheStocksOfTheHills(t *testing.T) {
	names, err := ninjafilm.Stocks()
	want := []string{"ash", "dusk", "lantern", "pollen", "smoke", "soot"}
	if err != nil || !slices.Equal(names, want) {
		t.Errorf("the shelf holds %v, %v", names, err)
	}
}

// anti-tests: a stock not on the shelf is refused, and the refusal says
// the name it was asked for and the names that are
func TestAStockNotOnTheShelfIsRefusedWithTheNamesThatAre(t *testing.T) {
	got, err := ninjafilm.Roll("celluloid", seed)
	if err == nil {
		t.Fatalf("a roll of celluloid was loaded: %v", got)
	}
	for _, word := range []string{"celluloid", "lantern"} {
		if !strings.Contains(err.Error(), word) {
			t.Errorf("the refusal %q does not say %s", err, word)
		}
	}
}

// shippedGauge is a gauge of the table as it ships, or the test stops.
func shippedGauge(t *testing.T, name string) film.Gauge {
	t.Helper()
	made, err := film.From(data.GaugesTable()).Named(name)
	if err != nil {
		t.Fatal(err)
	}
	return made
}

func TestTheGaugeIsTheStocks(t *testing.T) {
	want := shippedGauge(t, "full frame").Grainless()
	slide, err := ninjafilm.Roll("lantern", seed)
	if err != nil || slide.Gauge() != want {
		t.Errorf("a slide is cut to %v, %v", slide.Gauge(), err)
	}
	ash, err := ninjafilm.Roll("ash", seed)
	if err != nil || ash.Gauge().Name() != "super 35" {
		t.Errorf("ash is cut to %v, %v", ash.Gauge(), err)
	}
}

func TestEachStockSeesWhatItIsSensitiveTo(t *testing.T) {
	red, green := shine(lit, dark, dark), shine(dark, lit, dark)
	blue, black := shine(dark, dark, lit), maths.Light{}
	if got := still(t, "soot", red); got != still(t, "soot", black) {
		t.Errorf("soot saw red: it prints as %v", got)
	}
	if got := still(t, "soot", blue); got.R < clearLeast {
		t.Errorf("soot prints blue as %v: it should be nearly clear", got)
	}
	ashRed, ashBlue := still(t, "ash", red), still(t, "ash", blue)
	if ashRed.R >= ashBlue.R/darkRatio {
		t.Errorf("ash prints red %v and blue %v: red should be far "+
			"darker", ashRed, ashBlue)
	}
	if still(t, "ash", green) != ashBlue {
		t.Error("ash should see green as it sees blue")
	}
	smokeRed := still(t, "smoke", red)
	if smokeRed != still(t, "smoke", blue) ||
		smokeRed != still(t, "smoke", green) {
		t.Error("smoke should see red, green and blue alike")
	}
	if still(t, "dusk", blue) != still(t, "dusk", black) {
		t.Errorf("dusk saw blue: it prints as %v", still(t, "dusk", blue))
	}
}

func TestTheColourStocksKeepColour(t *testing.T) {
	red := shine(lit, dark, dark)
	slide := still(t, "lantern", red)
	if slide.R < colourRatio*slide.G || slide.R < colourRatio*slide.B {
		t.Errorf("a slide prints red as %v", slide)
	}
	white := still(t, "lantern", maths.Grey(lit))
	shade := maths.Grey(grey())
	middle := still(t, "lantern", shade)
	want := passed(midDensity(t, "lantern"))
	if !within(white.G/white.R, whole, whiteError) ||
		!within(white.B/white.R, whole, whiteError) ||
		white.R < colourRatio*middle.R ||
		!within(middle.R, want, whiteError) {
		t.Errorf("a slide prints white as %v and grey as %v", white,
			middle)
	}
	pair := still(t, "dusk", red)
	if pair.R < colourRatio*pair.G || pair.R < colourRatio*pair.B {
		t.Errorf("the two-colour print prints red as %v", pair)
	}
	leaf := still(t, "dusk", shine(dark, lit, dark))
	if leaf.G < leafRatio*leaf.R || !within(leaf.B/leaf.G, whole, leafBlue) {
		t.Errorf("the two-colour print prints green as %v: blue-green",
			leaf)
	}
}

// mean is the light a stock passes of a light across a frame of pixels.
// Each pixel has a mosaic of its own.
func mean(t *testing.T, name string, light maths.Light) maths.Light {
	t.Helper()
	made, err := ninjafilm.Roll(name, seed)
	if err != nil {
		t.Fatal(err)
	}
	picture := uniform(mosaicPixels, lit, light)
	got, err := made.Print(first, []*frame.Frame{picture})
	if err != nil {
		t.Fatal(err)
	}
	var total maths.Light
	for _, pixel := range got.Pix {
		total = total.Add(pixel.Light())
	}
	return total.Scale(whole / float64(mosaicPixels))
}

// The mosaic passes 7.5 percent of a white light as its plate did. It
// keeps the colour of a red one.
func TestTheMosaicIsDimAndKeepsColour(t *testing.T) {
	white := mean(t, "pollen", maths.Grey(mosaicLamp))
	for _, lamp := range []float64{white.R, white.G, white.B} {
		if !within(lamp, mosaicPasses, mosaicError) {
			t.Errorf("the mosaic passes %v of a white light, want %v",
				white, mosaicPasses)
		}
	}
	red := mean(t, "pollen", shine(mosaicLamp, dark, dark))
	if red.R < darkRatio*red.G || red.R < darkRatio*red.B ||
		red.R < mosaicRed {
		t.Errorf("the mosaic prints red as %v", red)
	}
}

// doc/filming.md says what exposure puts a white card on the screen's
// white for each stock. This holds the guide to it.
func TestTheGuidesExposuresPutAWhiteCardOnWhite(t *testing.T) {
	text, err := os.ReadFile(exposuresFile)
	if err != nil {
		t.Fatal(err)
	}
	rows, err := data.Parse(string(text))
	if err != nil {
		t.Fatal(err)
	}
	for _, row := range rows {
		name, _ := row.Text("stock")
		exposure, err := row.Float("exposure")
		if err != nil {
			t.Fatal(err)
		}
		shown := cardAt(t, name, exposure)
		if shown < guideLeast || shown > guideMost {
			t.Errorf("%s shows a white card at %v of the screen's white",
				name, shown)
		}
	}
}

// cardAt is the share of the screen's white that a white card is shown
// at, filmed on a stock and shown at an exposure. The mosaic is
// averaged over many pixels, for no one pixel of it is a plate.
func cardAt(t *testing.T, name string, exposure float64) float64 {
	t.Helper()
	if name == "pollen" {
		return mean(t, name, maths.Grey(lit)).G * exposure
	}
	return still(t, name, maths.Grey(lit)).G * exposure
}

// densities is the density of every pixel's red lamp.
func densities(made *frame.Frame) []float64 {
	var found []float64
	for _, pixel := range made.Pix {
		red := float64(pixel.R)
		found = append(found, -math.Log10(red))
	}
	return found
}

// spread is how far numbers stand from their mean, root mean square.
func spread(numbers []float64) float64 {
	var total, squares float64
	for _, each := range numbers {
		total += each
		squares += each * each
	}
	count := float64(len(numbers))
	average := total / count
	return math.Sqrt(squares/count - average*average)
}

// grainOf is how far the density of a stock wanders over a frame of its
// middle grey, and the width of the crystal of its first record.
func grainOf(t *testing.T, name string) (wander, across float64) {
	t.Helper()
	made, err := ninjafilm.Roll(name, seed)
	if err != nil {
		t.Fatal(err)
	}
	shade := maths.Grey(grey())
	picture := uniform(spreadAcross, spreadDown, shade)
	got, err := made.Print(first, []*frame.Frame{picture})
	if err != nil {
		t.Fatal(err)
	}
	across, err = rowOf(t, name).Float("crystal_um")
	if err != nil {
		t.Fatal(err)
	}
	return spread(densities(got)), across
}

// The grain of a frame is Siedentopf's. Over many pixels the density
// wanders by the square root of the logarithm of e times the shadow of
// a crystal times the density over the area of a pixel on the film. Two
// stocks of one curve wander in proportion to the widths of their
// crystals.
func TestTheGrainIsWhatTheCrystalsAndThePixelMake(t *testing.T) {
	smoke, err := ninjafilm.Roll("smoke", seed)
	if err != nil {
		t.Fatal(err)
	}
	coarse, wide := grainOf(t, "smoke")
	fine, narrow := grainOf(t, "soot")
	pitch := smoke.Gauge().Width() * metric / float64(spreadAcross)
	shadow := math.Pi * wide * wide / quartern
	want := math.Sqrt(math.Log10E * shadow * midDensity(t, "smoke") /
		(pitch * pitch))
	if !within(coarse/want, whole, spreadError) {
		t.Errorf("smoke's density wanders by %v, want %v", coarse, want)
	}
	if !within(coarse/fine, wide/narrow, grainError) {
		t.Errorf("smoke wanders %v and soot %v: their crystals are %v to "+
			"one in width", coarse, fine, wide/narrow)
	}
}
