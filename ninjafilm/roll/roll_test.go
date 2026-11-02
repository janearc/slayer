package roll_test

import (
	"errors"
	"math"
	"reflect"
	"strings"
	"testing"

	"github.com/janearc/libtheme-css/spaces/ok"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/ninjafilm/crystal"
	"github.com/janearc/slayer/ninjafilm/curve"
	"github.com/janearc/slayer/ninjafilm/data"
	"github.com/janearc/slayer/ninjafilm/film"
	"github.com/janearc/slayer/ninjafilm/frame"
	"github.com/janearc/slayer/ninjafilm/lay"
	"github.com/janearc/slayer/ninjafilm/mocks"
	"github.com/janearc/slayer/ninjafilm/record"
	"github.com/janearc/slayer/ninjafilm/roll"
	"go.uber.org/mock/gomock"
	"pgregory.net/rapid"
)

// what these tests count in, each defined
const (
	none          = 0      // defined: no amount at all
	once          = 1      // defined: a call that happens exactly once
	whole         = 1      // defined: the whole of a share, all the light
	single        = 1      // defined: a count of one
	first         = 0      // defined: the number of the first frame of a roll
	upward        = 1      // defined: math.Inf's code for infinity above all numbers
	downward      = -1     // defined: math.Inf's code for infinity below all numbers
	firstDraw     = 1      // defined: the first record of a site draws one on from the seed of the roll
	perMillimetre = 1000.0 // defined: a millimetre is 1000 micrometres by the metric prefixes
	halfway       = 0.5    // defined: the lightness of the middle grey, half of libtheme's scale
	quartering    = 4      // defined: the shadow of a disc is pi times its width squared over four
)

// what these tests choose, each picked in picked-numbers.yml
var (
	chipLeast      = data.Picked("roll-chip-least")
	chipMost       = data.Picked("roll-chip-most")
	frameLeast     = data.Whole("roll-frame-least")
	frameMost      = data.Whole("roll-frame-most")
	acrossLeast    = data.Whole("roll-across-least")
	acrossMost     = data.Whole("roll-across-most")
	brightMost     = data.Picked("roll-bright-most")
	recordsMost    = data.Whole("roll-records-most")
	crystalLeast   = data.Picked("roll-crystal-least")
	crystalMost    = data.Picked("roll-crystal-most")
	stopsMost      = data.Whole("roll-stops-most")
	doublingsMost  = data.Whole("roll-doublings-most")
	sameMost       = data.Whole("roll-same-most")
	grainyWidth    = data.Picked("roll-grainy-width")
	grainyAcross   = data.Whole("roll-grainy-across")
	grainyDown     = data.Whole("roll-grainy-down")
	grainyCrystal  = data.Picked("roll-grainy-crystal")
	grainyLeast    = data.Picked("roll-grainy-least")
	grainyMost     = data.Picked("roll-grainy-most")
	grainySlope    = data.Picked("roll-grainy-slope")
	grainyFrame    = data.Whole("roll-grainy-frame")
	grainySeed     = uint64(data.Whole("roll-grainy-seed"))
	largeWidth     = data.Picked("roll-large-width")
	spreadAcross   = data.Whole("roll-spread-across")
	spreadDown     = data.Whole("roll-spread-down")
	spreadError    = data.Picked("roll-spread-error")
	toleranceFloat = data.Picked("tolerance-single")
	smoothError    = data.Picked("roll-smooth-error")
	lampSteps      = data.Whole("roll-lamp-steps")
)

// stopper is a test that can be stopped: a testing.T, or a rapid.T.
type stopper interface {
	Helper()
	Fatal(args ...any)
}

// lamps is a light with each lamp drawn from zero up to a ceiling.
func lamps(t *rapid.T, label string, ceiling float64) maths.Light {
	return maths.RGB(lamp(t, label+" red", ceiling),
		lamp(t, label+" green", ceiling), lamp(t, label+" blue", ceiling))
}

// lamp is the brightness of one lamp up to a ceiling. It is a whole
// number of the steps the ceiling is cut into.
func lamp(t *rapid.T, label string, ceiling float64) float64 {
	steps := rapid.IntRange(none, lampSteps).Draw(t, label)
	return float64(steps) / float64(lampSteps) * ceiling
}

// shines is whether a light has any light in it.
func shines(light maths.Light) bool {
	return light.R+light.G+light.B > none
}

// shiningLamps draws lights until one has some light in it.
func shiningLamps() *rapid.Generator[maths.Light] {
	return rapid.Custom(allLamps).Filter(shines)
}

// allLamps is a light with each lamp drawn up to the whole of the light.
func allLamps(t *rapid.T) maths.Light { return lamps(t, "lamps", whole) }

// drawnRecord is a record a film could have.
func drawnRecord(t *rapid.T) record.Record {
	window := shiningLamps().Draw(t, "window")
	image := shiningLamps().Draw(t, "image")
	across := rapid.Float64Range(crystalLeast, crystalMost).Draw(t, "across")
	least := rapid.Float64Range(none, whole).Draw(t, "least")
	room := rapid.Float64Range(whole, grainyMost).Draw(t, "room")
	tone := toneOf(t, least, least+room, grainySlope)
	return recordOf(t, window, image, across, tone)
}

// drawnRecords are the records a stock could have.
func drawnRecords(t *rapid.T) []record.Record {
	many := rapid.IntRange(single, recordsMost).Draw(t, "records")
	var made []record.Record
	for range many {
		made = append(made, drawnRecord(t))
	}
	return made
}

// toneOf is a curve, or the test stops.
func toneOf(t stopper, least, most, slope float64) curve.Curve {
	t.Helper()
	made, err := curve.New(least, most, slope)
	if err != nil {
		t.Fatal(err)
	}
	return made
}

// recordOf is a record, or the test stops.
func recordOf(t stopper, window, image maths.Light, across float64,
	tone curve.Curve) record.Record {
	t.Helper()
	grain, err := crystal.New(across)
	if err != nil {
		t.Fatal(err)
	}
	made, err := record.New(window, image, grain, tone)
	if err != nil {
		t.Fatal(err)
	}
	return made
}

// chip is a gauge so small that a pixel holds few crystals.
func chip(width float64) film.Gauge {
	made, err := film.NewGauge("a chip", width, width, none)
	if err != nil {
		panic(err)
	}
	return made
}

// super35 is the gauge of super 35 as it ships, or the test stops.
func super35(t stopper) film.Gauge {
	t.Helper()
	made, err := film.From(data.GaugesTable()).Named("super 35")
	if err != nil {
		t.Fatal(err)
	}
	return made
}

// drawnGauge is a gauge a drawn roll could be cut to.
func drawnGauge(t *rapid.T) film.Gauge {
	return chip(rapid.Float64Range(chipLeast, chipMost).Draw(t, "chip"))
}

// stacked is a lay of records one under another, or the test stops.
func stacked(t stopper) lay.Lay {
	t.Helper()
	made, err := lay.New("stacked", none, none)
	if err != nil {
		t.Fatal(err)
	}
	return made
}

// frameOf is a frame of drawn light, cover and depth.
func frameOf(t *rapid.T, columns, rows int) *frame.Frame {
	made := frame.New(columns, rows)
	share := rapid.Float64Range(none, whole)
	for index := range made.Pix {
		light := lamps(t, "pixel", brightMost)
		cover := float32(share.Draw(t, "cover"))
		depth := float32(share.Draw(t, "depth"))
		made.Pix[index] = frame.NewPixel(float32(light.R),
			float32(light.G), float32(light.B), cover, depth)
	}
	return made
}

// glimpses are some frames the size of a frame, as many as a count.
func glimpses(t *rapid.T, columns, rows, count int) []*frame.Frame {
	var caught []*frame.Frame
	for range count {
		caught = append(caught, frameOf(t, columns, rows))
	}
	return caught
}

// loaded is a roll of a stock, found on a mock shelf once.
func loaded(t stopper, ctrl *gomock.Controller, stock roll.Stock,
	seed uint64) roll.Roll {
	t.Helper()
	shelf := mocks.NewMockShelf(ctrl)
	shelf.EXPECT().Find("any").Return(stock, nil).Times(once)
	made, err := roll.Load(shelf, "any", seed)
	if err != nil {
		t.Fatal(err)
	}
	return made
}

// uniform is a lit frame with the same light in every pixel, covered
// and five away.
func uniform(columns, rows int, light maths.Light) *frame.Frame {
	made := frame.New(columns, rows)
	for index := range made.Pix {
		made.Pix[index] = frame.NewPixel(float32(light.R),
			float32(light.G), float32(light.B), whole, float32(columns))
	}
	return made
}

// grey is the luminance of the grey halfway up libtheme's lightness,
// which a stock is rated so that it lands on the middle of its curve.
func grey() float64 {
	_, luminance, _ := ok.Grey(halfway).Swatch().XYZ()
	return luminance
}

// within is whether two numbers are within a tolerance of each other.
func within(one, other, tolerance float64) bool {
	return math.Abs(one-other) <= tolerance
}

// passed is the share of a lamp's light that a density lets past: ten
// to the minus density.
func passed(density float64) float64 {
	return math.Exp(-density * math.Ln10)
}

func TestALoadedRollIsCutToTheGaugeOfItsStock(t *testing.T) {
	ctrl := gomock.NewController(t)
	empty := mocks.NewMockLay(ctrl)
	stock := roll.NewStock(super35(t), empty, nil)
	made := loaded(t, ctrl, stock, grainySeed)
	if got := made.Gauge(); got != super35(t) {
		t.Errorf("a roll of super 35 is cut to %v", got)
	}
}

// anti-tests: the shelf's refusal is the roll's, and no roll comes of it
func TestARollIsNotLoadedWhereTheShelfHasNoStock(t *testing.T) {
	ctrl := gomock.NewController(t)
	shelf := mocks.NewMockShelf(ctrl)
	refusal := errors.New("the shelf holds no such stock")
	var unstocked roll.Stock
	shelf.EXPECT().Find("celluloid").Return(unstocked, refusal).Times(once)
	var unloaded roll.Roll
	got, err := roll.Load(shelf, "celluloid", grainySeed)
	if !errors.Is(err, refusal) || !reflect.DeepEqual(got, unloaded) {
		t.Errorf("a stock not on the shelf loaded %v, %v", got, err)
	}
}

// noPixels is a frame of a size that has had its pixels taken away.
func noPixels(columns, rows int) *frame.Frame {
	made := frame.New(columns, rows)
	made.Pix = nil
	return made
}

// faults are lists of glimpses a roll cannot print, by the words of the
// refusal each is given.
func faults() map[string][][]*frame.Frame {
	good := uniform(frameLeast, frameLeast, maths.Light{})
	wider := uniform(frameMost, frameLeast, maths.Light{})
	short := uniform(frameMost, frameMost, maths.Light{})
	short.Pix = short.Pix[:frameLeast]
	endless, lost := uniform(frameMost, frameMost, maths.Light{}),
		uniform(frameMost, frameMost, maths.Light{})
	endless.Pix[frameLeast].G = float32(math.Inf(upward))
	lost.Pix[first].B = float32(math.NaN())
	return map[string][][]*frame.Frame{
		"one glimpse or more": {nil, {}},
		"one pixel or more": {{nil}, {good, nil},
			{noPixels(whole, whole)}, {frame.New(none, whole)},
			{frame.New(whole, none)}, {short}},
		"all one size": {{good, wider}, {wider, good}},
		"not a number": {{endless}, {lost}},
	}
}

// anti-tests: a roll never loaded prints nothing. Nor does one rated past
// what a number holds, or one given glimpses that are not frames of one
// size. The lay is a mock with no expectations. So any reach for it
// fails the test.
func TestARollRefusesWhatItCannotPrint(t *testing.T) {
	ctrl := gomock.NewController(t)
	untouched := mocks.NewMockLay(ctrl)
	stock := roll.NewStock(super35(t), untouched, nil)
	made := loaded(t, ctrl, stock, grainySeed)
	var unloaded roll.Roll
	cases := map[string]roll.Roll{
		"a roll never loaded":       unloaded,
		"a roll rated past numbers": made.Rated(math.Inf(upward)),
		"a roll rated to nothing":   made.Rated(math.Inf(downward)),
		"a roll rated by no number": made.Rated(math.NaN()),
	}
	good := []*frame.Frame{uniform(frameLeast, frameLeast, maths.Light{})}
	for name, bad := range cases {
		got, err := bad.Print(first, good)
		if err == nil || got != nil ||
			!strings.Contains(err.Error(), "comes from Roll") {
			t.Errorf("%s printed %v, %v", name, got, err)
		}
	}
	for words, lists := range faults() {
		for _, list := range lists {
			got, err := made.Print(first, list)
			if err == nil || got != nil ||
				!strings.Contains(err.Error(), words) {
				t.Errorf("%d glimpses printed %v, %v, not a refusal "+
					"that says %q", len(list), got, err, words)
			}
		}
	}
}

// siteCalls is what a mocked lay is told to expect of one print, and
// what each site is to give back.
type siteCalls struct {
	kept    []int
	filters []maths.Light
}

// expectSites tells a mock lay to expect one call for each site of a
// print, and says what each call gives back.
func expectSites(t *rapid.T, laid *mocks.MockLay, records []record.Record,
	seed uint64, which, wide, high int) (calls siteCalls) {
	for y := range high {
		for x := range wide {
			kept := rapid.IntRange(none, len(records)).Draw(t, "kept")
			filter := lamps(t, "filter", whole)
			calls.kept = append(calls.kept, kept)
			calls.filters = append(calls.filters, filter)
			laid.EXPECT().Site(records, seed, int64(x), int64(y),
				int64(which)).Return(records[:kept], filter).Times(once)
		}
	}
	return calls
}

// expectedPixel is the light that comes through one pixel of a print. It
// is worked from the pieces the roll is made of. They are the lay's
// answer for each site and the chance each record draws. The last is the
// density each develops.
func expectedPixel(light maths.Light, records []record.Record, seed uint64,
	which int, column, row, across int, pitch, gain float64,
	calls siteCalls, wide int) maths.Light {
	var total maths.Light
	area := pitch * pitch / float64(across*across)
	for down := range across {
		for along := range across {
			x := column*across + along
			y := row*across + down
			place := y*wide + x
			var blocked maths.Light
			for index, one := range records[:calls.kept[place]] {
				draw := crystal.Chance(seed+uint64(index)+firstDraw,
					int64(x), int64(y), int64(which))
				dense := one.Developed(light.Scale(gain), area, draw)
				blocked = blocked.Add(one.Image().Scale(dense))
			}
			through := maths.RGB(passed(blocked.R), passed(blocked.G),
				passed(blocked.B))
			total = total.Add(calls.filters[place].Times(through))
		}
	}
	return total.Scale(whole / float64(across*across))
}

// Prints ask the lay how many sites lie across a pixel. They ask once.
// Then they ask for each site in turn by its place in the frame, the
// seed of the roll and the number of the frame. Every pixel is the mean
// of the light through its sites. The filter of a site passes what its
// records let by. Each record has a chance of its own.
func TestAPrintAsksTheLayForEachSiteAndAddsWhatComesThrough(t *testing.T) {
	rapid.Check(t, printAsksTheLay)
}

func printAsksTheLay(t *rapid.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	records, gauge := drawnRecords(t), drawnGauge(t)
	columns := rapid.IntRange(frameLeast, frameMost).Draw(t, "columns")
	rows := rapid.IntRange(frameLeast, frameMost).Draw(t, "rows")
	across := rapid.IntRange(acrossLeast, acrossMost).Draw(t, "across")
	seed := rapid.Uint64().Draw(t, "seed")
	which := int(rapid.Uint32().Draw(t, "which"))
	stops := rapid.IntRange(-stopsMost, stopsMost).Draw(t, "stops")
	picture := frameOf(t, columns, rows)
	laid := mocks.NewMockLay(ctrl)
	pitch := gauge.Width() * perMillimetre / float64(columns)
	laid.EXPECT().Across(pitch).Return(across).Times(once)
	wide, high := columns*across, rows*across
	calls := expectSites(t, laid, records, seed, which, wide, high)
	made := loaded(t, ctrl, roll.NewStock(gauge, laid, records), seed)
	gain := math.Exp2(float64(stops))
	got, err := made.Rated(float64(stops)).Print(which,
		[]*frame.Frame{picture})
	if err != nil {
		t.Fatal(err)
	}
	for row := range rows {
		for column := range columns {
			light := picture.At(column, row).Light()
			want := expectedPixel(light, records, seed, which, column,
				row, across, pitch, gain, calls, wide)
			checkPixel(t, got, picture, column, row, want)
		}
	}
}

// checkPixel stops the test unless a printed pixel has the light it is
// to have, and the cover and depth of the pixel it came from.
func checkPixel(t *rapid.T, got, from *frame.Frame, column, row int,
	want maths.Light) {
	made, before := got.At(column, row), from.At(column, row)
	have := made.Light()
	if !within(have.R, want.R, toleranceFloat) ||
		!within(have.G, want.G, toleranceFloat) ||
		!within(have.B, want.B, toleranceFloat) {
		t.Fatalf("pixel %d,%d printed %v, want %v", column, row, have,
			want)
	}
	if made.Cover != before.Cover || made.Depth != before.Depth {
		t.Fatalf("pixel %d,%d lost its cover or depth: %v from %v",
			column, row, made, before)
	}
}

// stockOn is a stock of drawn records on a gauge, laid one under another.
func stockOn(t *rapid.T, gauge film.Gauge) roll.Stock {
	return roll.NewStock(gauge, stacked(t), drawnRecords(t))
}

// Glimpses are gathered as light adds. The print of several glimpses is
// the print of their mean. That keeps the cover and depth of the first.
// The counts are one, two and four. Their reciprocals are exact.
func TestGlimpsesAreGatheredAsLightAdds(t *testing.T) {
	rapid.Check(t, glimpsesGatherAsLightAdds)
}

func glimpsesGatherAsLightAdds(t *rapid.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	shift := rapid.IntRange(none, doublingsMost).Draw(t, "doublings")
	count := single << shift
	columns := rapid.IntRange(frameLeast, frameMost).Draw(t, "columns")
	rows := rapid.IntRange(frameLeast, frameMost).Draw(t, "rows")
	seed := rapid.Uint64().Draw(t, "seed")
	stock := stockOn(t, drawnGauge(t))
	made := loaded(t, ctrl, stock, seed)
	caught := glimpses(t, columns, rows, count)
	kept := copied(caught)
	mean := meanOf(caught)
	together, err := made.Print(first, caught)
	if err != nil {
		t.Fatal(err)
	}
	alone, err := made.Print(first, []*frame.Frame{mean})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(together.Pix, alone.Pix) {
		t.Fatalf("%d glimpses printed %v, their mean %v", count,
			together.Pix, alone.Pix)
	}
	if !reflect.DeepEqual(caught, kept) {
		t.Fatal("printing changed a glimpse it was given")
	}
}

// copied are frames copied, so that a print cannot change them unseen.
func copied(frames []*frame.Frame) []*frame.Frame {
	var kept []*frame.Frame
	for _, each := range frames {
		again := frame.New(each.W, each.H)
		copy(again.Pix, each.Pix)
		kept = append(kept, again)
	}
	return kept
}

// meanOf is the frame whose light is the mean of the light of frames,
// with the cover and depth of the first.
func meanOf(frames []*frame.Frame) *frame.Frame {
	mean := copied(frames[:single])[first]
	for index := range mean.Pix {
		var total maths.Light
		for _, each := range frames {
			total = total.Add(each.Pix[index].Light())
		}
		many := float64(len(frames))
		total = total.Scale(whole / many)
		mean.Pix[index].R, mean.Pix[index].G = float32(total.R),
			float32(total.G)
		mean.Pix[index].B = float32(total.B)
	}
	return mean
}

// Stops over are doublings of the light bit for bit. Rating a roll leaves the
// roll it came from as it was.
func TestAStopOverIsTwiceTheLight(t *testing.T) {
	rapid.Check(t, stopsAreDoublings)
}

func stopsAreDoublings(t *rapid.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	stops := rapid.IntRange(-stopsMost, stopsMost).Draw(t, "stops")
	columns := rapid.IntRange(frameLeast, frameMost).Draw(t, "columns")
	rows := rapid.IntRange(frameLeast, frameMost).Draw(t, "rows")
	stock := stockOn(t, drawnGauge(t))
	seed := rapid.Uint64().Draw(t, "seed")
	made := loaded(t, ctrl, stock, seed)
	picture := frameOf(t, columns, rows)
	scaled := copied([]*frame.Frame{picture})[first]
	doubled := math.Exp2(float64(stops))
	factor := float32(doubled)
	for index := range scaled.Pix {
		scaled.Pix[index].R *= factor
		scaled.Pix[index].G *= factor
		scaled.Pix[index].B *= factor
	}
	before := printed(t, made, picture)
	rated := made.Rated(float64(stops))
	pushed := printed(t, rated, picture)
	twice := printed(t, made, scaled)
	if !reflect.DeepEqual(pushed.Pix, twice.Pix) {
		t.Fatalf("%d stops over printed %v, %d times the light %v", stops,
			pushed.Pix, stops, twice.Pix)
	}
	if !reflect.DeepEqual(before.Pix, printed(t, made, picture).Pix) {
		t.Fatal("rating a roll changed the roll it came from")
	}
}

// printed is one frame through a roll, or the test stops.
func printed(t stopper, made roll.Roll, picture *frame.Frame) *frame.Frame {
	t.Helper()
	got, err := made.Print(first, []*frame.Frame{picture})
	if err != nil {
		t.Fatal(err)
	}
	return got
}

// grainyStock is a stock of one record. Its gauge is so small that a
// pixel holds few crystals. So every pixel has grain of its own.
func grainyStock(t stopper, width float64) roll.Stock {
	t.Helper()
	tone := toneOf(t, grainyLeast, grainyMost, grainySlope)
	only := recordOf(t, maths.Grey(whole), maths.Grey(whole),
		grainyCrystal, tone)
	return roll.NewStock(chip(width), stacked(t), []record.Record{only})
}

// same is how many pixels of two frames are alike.
func same(one, other *frame.Frame) int {
	alike := none
	for index, pixel := range one.Pix {
		if pixel == other.Pix[index] {
			alike++
		}
	}
	return alike
}

// The same frame of the same roll has the same grain forever. Another frame
// matches only where chance puts it. So does another roll.
func TestARollAndAFrameMakeTheSameGrainForEver(t *testing.T) {
	ctrl := gomock.NewController(t)
	shade := maths.Grey(grey())
	middle := uniform(grainyAcross, grainyDown, shade)
	stock := grainyStock(t, grainyWidth)
	made := loaded(t, ctrl, stock, grainySeed)
	other := loaded(t, ctrl, stock, grainySeed+once)
	base := printedAs(t, made, middle, grainyFrame)
	again := printedAs(t, made, middle, grainyFrame)
	next := printedAs(t, made, middle, grainyFrame+once)
	another := printedAs(t, other, middle, grainyFrame)
	if alike := same(base, again); alike != len(base.Pix) {
		t.Errorf("one frame printed twice matched at %d pixels of %d",
			alike, len(base.Pix))
	}
	if alike := same(base, next); alike > sameMost {
		t.Errorf("the next frame matched at %d pixels", alike)
	}
	if alike := same(base, another); alike > sameMost {
		t.Errorf("another roll matched at %d pixels", alike)
	}
}

// printedAs is frame number which of a roll, or the test stops.
func printedAs(t stopper, made roll.Roll, picture *frame.Frame,
	which int) *frame.Frame {
	t.Helper()
	got, err := made.Print(which, []*frame.Frame{picture})
	if err != nil {
		t.Fatal(err)
	}
	return got
}

// Stills printed large have too many crystals a pixel for grain to
// show. So a still is the curve of the stock alone. The middle grey
// lands on the middle of the curve. The lamps its image blocks are
// dimmed by the density there.
func TestAMiddleGreyLandsOnTheMiddleOfTheCurve(t *testing.T) {
	ctrl := gomock.NewController(t)
	stock := grainyStock(t, largeWidth)
	made := loaded(t, ctrl, stock, grainySeed)
	shade := maths.Grey(grey())
	picture := uniform(frameLeast, frameLeast, shade)
	pixel := printed(t, made, picture).At(first, first)
	got := pixel.Light()
	middle := (grainyLeast + grainyMost) * halfway
	want := passed(middle)
	if !within(got.R, want, smoothError) || got.G != got.R ||
		got.B != got.R {
		t.Errorf("a middle grey prints as %v, want %v", got, want)
	}
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
	mean := total / count
	return math.Sqrt(squares/count - mean*mean)
}

// The grain of a frame is Siedentopf's. Over many pixels the density
// wanders by the square root of the logarithm of e times the shadow of
// a crystal times the density over the area of a pixel on the film.
func TestTheGrainIsWhatTheCrystalsAndThePixelMake(t *testing.T) {
	ctrl := gomock.NewController(t)
	stock := grainyStock(t, largeWidth)
	made := loaded(t, ctrl, stock, grainySeed)
	shade := maths.Grey(grey())
	picture := uniform(spreadAcross, spreadDown, shade)
	pitch := largeWidth * perMillimetre / float64(spreadAcross)
	shadow := math.Pi * grainyCrystal * grainyCrystal / quartering
	middle := (grainyLeast + grainyMost) * halfway
	want := math.Sqrt(math.Log10E * shadow * middle / (pitch * pitch))
	shot := printed(t, made, picture)
	got := spread(densities(shot))
	if !within(got/want, whole, spreadError) {
		t.Errorf("the density wanders by %v, want %v", got, want)
	}
}
