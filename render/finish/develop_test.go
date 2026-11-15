package finish

import (
	"image"
	"strings"
	"testing"

	"github.com/janearc/slayer/ninjafilm"
	"github.com/janearc/slayer/render/colour"
	"github.com/janearc/slayer/render/frame"
	"pgregory.net/rapid"
)

// side is how many pixels each way the test frames are: small, so that
// a stock develops in a moment. It is picked.
const side = 48

// fallback is the tone a job that says neither print nor stock gets.
var fallback = colour.Tone{Exposure: 3, Knee: 0.5, Top: 4}

// developed is the picture a job's developer makes of frame number.
func developed(t *testing.T, text string, number int,
	glimpses ...*frame.Frame) *image.RGBA {
	t.Helper()
	made, err := NewDeveloper(jobOf(t, text), fallback)
	if err != nil {
		t.Fatal(err)
	}
	picture, err := made.Develop(number, glimpses)
	if err != nil {
		t.Fatal(err)
	}
	return picture
}

// a print develops as the tone does, byte for byte, its stops doubling
// the light each.
func TestAPrintDevelopsAsTheToneDoes(t *testing.T) {
	light := gradient(side, side)
	got := developed(t, "print: {knee: 0.7, top: 8, bloom: 2, stops: 1}\n",
		0, light)
	want := colour.Tone{Exposure: 2, Knee: 0.7, Top: 8, Bloom: 2}.
		Develop(light)
	if string(got.Pix) != string(want.Pix) {
		t.Error("the print is not the tone's picture")
	}
	other := colour.Tone{Exposure: 1, Knee: 0.7, Top: 8, Bloom: 2}.
		Develop(light)
	if string(got.Pix) == string(other.Pix) {
		t.Error("a stop of exposure made no difference")
	}
}

// a print that says no stops or bloom develops with none of either.
func TestAPrintWithoutStopsHasNone(t *testing.T) {
	light := gradient(side, side)
	got := developed(t, "print: {knee: 0.7, top: 8}\n", 0, light)
	want := colour.Tone{Exposure: 1, Knee: 0.7, Top: 8}.Develop(light)
	if string(got.Pix) != string(want.Pix) {
		t.Error("a print with no stops is not at one exposure")
	}
}

// a job that says neither uses the caller's tone; one that says a print
// does not.
func TestAJobWithNeitherUsesTheCallersTone(t *testing.T) {
	light := gradient(side, side)
	got := developed(t, "", 0, light)
	if string(got.Pix) != string(fallback.Develop(light).Pix) {
		t.Error("the caller's tone was not used")
	}
	printed := developed(t, "print: {knee: 0.7, top: 8}\n", 0, light)
	if string(printed.Pix) == string(got.Pix) {
		t.Error("a job with a print used the caller's tone")
	}
}

// a plain tone takes the mean of the glimpses: two glimpses are the
// picture of their mean, and not of the first alone.
func TestATakesTheMeanOfItsGlimpses(t *testing.T) {
	first, second := gradient(side, side), scaled(gradient(side, side), 3)
	got := developed(t, "print: {knee: 0.7, top: 8}\n", 0, first, second)
	mean, err := Mean([]*frame.Frame{first, second})
	if err != nil {
		t.Fatal(err)
	}
	want := colour.Tone{Exposure: 1, Knee: 0.7, Top: 8}.Develop(mean)
	if string(got.Pix) != string(want.Pix) {
		t.Error("two glimpses are not their mean")
	}
	alone := developed(t, "print: {knee: 0.7, top: 8}\n", 0, first)
	if string(alone.Pix) == string(got.Pix) {
		t.Error("the second glimpse made no difference")
	}
}

// the same job and the same frame number give the same bytes.
func TestAStockGivesTheSameBytesForTheSameFrame(t *testing.T) {
	text := "stock: {name: ash, seed: 7}\n"
	light := gradient(side, side)
	one := developed(t, text, 3, light)
	again := developed(t, text, 3, light)
	if string(one.Pix) != string(again.Pix) {
		t.Error("the same frame number gave other grain")
	}
}

// the next frame number gives different grain, and so does another
// seed.
func TestAStocksGrainIsNewOnTheNextFrameAndForAnotherSeed(t *testing.T) {
	light := gradient(side, side)
	text := "stock: {name: smoke, seed: 7}\n"
	base := developed(t, text, 3, light)
	next := developed(t, text, 4, light)
	if string(base.Pix) == string(next.Pix) {
		t.Error("the next frame number had the same grain")
	}
	seeded := developed(t, "stock: {name: smoke, seed: 8}\n", 3, light)
	if string(base.Pix) == string(seeded.Pix) {
		t.Error("another seed had the same grain")
	}
}

// brightness is the mean of a picture's green channel, 0 to 255.
func brightness(img *image.RGBA) float64 {
	var sum float64
	for i := 1; i < len(img.Pix); i += 4 {
		sum += float64(img.Pix[i])
	}
	return sum / float64(len(img.Pix)/4)
}

// rating a stock over brightens it and under darkens it, and a stock of
// no rating is between.
func TestRatingAStockBrightensOrDarkens(t *testing.T) {
	light := gradient(side, side)
	at := func(rated string) float64 {
		return brightness(developed(t,
			"stock: {name: smoke, seed: 1"+rated+"}\n", 0, light))
	}
	over, rated, under := at(", rated: 2"), at(""), at(", rated: -2")
	if !(over > rated && rated > under) {
		t.Errorf("rated +2, 0, -2 gave brightness %.1f, %.1f, %.1f",
			over, rated, under)
	}
}

// a stock takes every glimpse: a second, brighter glimpse changes the
// picture.
func TestAStockTakesAllItsGlimpses(t *testing.T) {
	text := "stock: {name: ash, seed: 2}\n"
	first := gradient(side, side)
	alone := developed(t, text, 0, first)
	both := developed(t, text, 0, first, scaled(first, 4))
	if string(alone.Pix) == string(both.Pix) {
		t.Error("a second glimpse made no difference to the stock")
	}
}

// a stock not on the shelf is refused, and the refusal lists the shelf;
// so is a seed no roll takes.
func TestAStockNotOnTheShelfIsRefusedWithTheShelf(t *testing.T) {
	_, err := stockDeveloper("tintype", 0, 0)
	if err == nil {
		t.Fatal("a stock that is not on the shelf was taken")
	}
	shelf, _ := ninjafilm.Stocks()
	for _, word := range append(shelf, "tintype") {
		if !strings.Contains(err.Error(), word) {
			t.Errorf("%q does not say %q", err, word)
		}
	}
	for _, seed := range []int64{-1, seedMost + 1} {
		if _, err := stockDeveloper("ash", seed, 0); err == nil ||
			!strings.Contains(err.Error(), "seed") {
			t.Errorf("seed %d gave %v", seed, err)
		}
	}
	if _, err := stockDeveloper("ash", seedMost, 0); err != nil {
		t.Errorf("the largest seed was refused: %v", err)
	}
}

// every stock on the shelf has an exposure, and no exposure is for a
// stock that is not on it: a stock added to the shelf is not left black.
func TestEveryStockOnTheShelfHasAnExposure(t *testing.T) {
	shelf, err := ninjafilm.Stocks()
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range shelf {
		if transmission[name] <= none {
			t.Errorf("stock %s has no exposure", name)
		}
	}
	if len(transmission) != len(shelf) {
		t.Errorf("%d exposures for %d stocks", len(transmission),
			len(shelf))
	}
}

// a developer says whether it is a stock, and refuses a frame number
// below nought.
func TestADeveloperKnowsItsKindAndRefusesNegativeFrames(t *testing.T) {
	stock, err := NewDeveloper(jobOf(t, "stock: {name: ash}\n"), fallback)
	if err != nil {
		t.Fatal(err)
	}
	plain, err := NewDeveloper(jobOf(t, ""), fallback)
	if err != nil {
		t.Fatal(err)
	}
	if !stock.Stocked() || plain.Stocked() {
		t.Error("a developer is mistaken for the other kind")
	}
	for _, made := range []Developer{stock, plain} {
		if _, err := made.Develop(-1, []*frame.Frame{gradient(2, 2)}); err ==
			nil {
			t.Error("frame -1 was developed")
		}
	}
}

// a developer refuses glimpses it cannot develop: none for either kind.
func TestADeveloperRefusesNoGlimpses(t *testing.T) {
	for _, text := range []string{"", "stock: {name: ash}\n"} {
		made, err := NewDeveloper(jobOf(t, text), fallback)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := made.Develop(0, nil); err == nil {
			t.Errorf("%q developed no glimpses", text)
		}
	}
}

// the mean of identical glimpses is the glimpse, exactly.
func TestTheMeanOfIdenticalGlimpsesIsTheGlimpse(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		light := rapid.Float32Range(0, 1000).Draw(t, "light")
		count := rapid.IntRange(1, 9).Draw(t, "count")
		one := frame.New(2, 2)
		for i := range one.Pix {
			one.Pix[i].R, one.Pix[i].G, one.Pix[i].B = light, light/3,
				light*2
		}
		var glimpses []*frame.Frame
		for range count {
			glimpses = append(glimpses, one)
		}
		mean, err := Mean(glimpses)
		if err != nil {
			t.Fatal(err)
		}
		for i, pixel := range mean.Pix {
			if pixel != one.Pix[i] {
				t.Fatalf("pixel %d is %v, want %v", i, pixel,
					one.Pix[i])
			}
		}
	})
}

// the mean of two glimpses is halfway, and the mean is a new frame that
// leaves its glimpses alone.
func TestTheMeanOfTwoIsHalfwayAndNew(t *testing.T) {
	dark, bright := frame.New(1, 1), frame.New(1, 1)
	bright.Pix[0].R, bright.Pix[0].G, bright.Pix[0].B = 4, 8, 2
	mean, err := Mean([]*frame.Frame{dark, bright})
	if err != nil {
		t.Fatal(err)
	}
	if got := mean.Pix[0]; got.R != 2 || got.G != 4 || got.B != 1 {
		t.Errorf("the mean is %v", got)
	}
	if dark.Pix[0].R != 0 || bright.Pix[0].R != 4 {
		t.Error("the mean changed a glimpse")
	}
}

// the anti-test: no glimpses, a nil one, or one of another size is
// refused, the odd one by its place.
func TestTheMeanRefusesGlimpsesItCannotTake(t *testing.T) {
	cases := []struct {
		name     string
		glimpses []*frame.Frame
		says     string
	}{
		{"none", nil, "one glimpse or more"},
		{"nil first", []*frame.Frame{nil}, "glimpse 0 is nil"},
		{"nil second", []*frame.Frame{frame.New(1, 1), nil},
			"glimpse 1 is nil"},
		{"other size", []*frame.Frame{frame.New(1, 1), frame.New(2, 1)},
			"glimpse 1 is 2x1 among glimpses 1x1"},
	}
	for _, each := range cases {
		_, err := Mean(each.glimpses)
		if err == nil || !strings.Contains(err.Error(), each.says) {
			t.Errorf("%s: %v, want it to say %q", each.name, err,
				each.says)
		}
	}
}
