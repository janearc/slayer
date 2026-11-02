package lens_test

import (
	"math"
	"reflect"
	"testing"
	"time"

	"github.com/janearc/slayer/ninjafilm/data"
	"github.com/janearc/slayer/ninjafilm/frame"
	"pgregory.net/rapid"
)

// what the exposure tests choose, each picked in picked-numbers.yml
var (
	pictureLeast  = data.Whole("lens-picture-least")
	pictureMost   = data.Whole("lens-picture-most")
	glimpsesMost  = data.Whole("lens-glimpses-most")
	brightMost    = data.Picked("lens-bright-most")
	doublingsMost = data.Whole("lens-doublings-most")
	lampSteps     = data.Whole("lens-lamp-steps")
	radiance      = data.Picked("lens-radiance")
	rawError      = data.Picked("lens-raw-error")
)

// lit is a frame of columns by rows of one radiance in every channel.
func lit(columns, rows int, level float32) *frame.Frame {
	made := frame.New(columns, rows)
	for index := range made.Pix {
		made.Pix[index] = frame.NewPixel(level, level, level, whole,
			float32(focus))
	}
	return made
}

// drawnFrame is a frame of drawn light, cover and depth.
func drawnFrame(t *rapid.T, columns, rows int) *frame.Frame {
	made := frame.New(columns, rows)
	share := rapid.Float64Range(none, whole)
	for index := range made.Pix {
		cover := float32(share.Draw(t, "cover"))
		made.Pix[index] = frame.NewPixel(lamp(t, "red"),
			lamp(t, "green"), lamp(t, "blue"), cover, lamp(t, "depth"))
	}
	return made
}

// lamp is the brightness of one lamp, a whole number of the steps the
// brightest is cut into.
func lamp(t *rapid.T, label string) float32 {
	steps := rapid.IntRange(none, lampSteps).Draw(t, label)
	return float32(steps) / float32(lampSteps) * float32(brightMost)
}

// glimpsesOf are some drawn frames of one size.
func glimpsesOf(t *rapid.T, columns, rows int) []*frame.Frame {
	many := rapid.IntRange(whole, glimpsesMost).Draw(t, "glimpses")
	var caught []*frame.Frame
	for range many {
		caught = append(caught, drawnFrame(t, columns, rows))
	}
	return caught
}

// Scenes of one radiance that are still for a second expose the film on
// the axis to the radiance times pi times the sine squared of the angle
// to the edge of the aperture times the second. The lens law is worked
// here from the numbers of the lens and not asked of it.
func TestAStillSceneExposesAsTheLawSays(t *testing.T) {
	glass := mustThin(t, focus)
	image := imageAt(focal*millimetre, focus)
	reach := focal * millimetre / (pair * stop)
	onAxis := math.Pi * reach * reach / (reach*reach + image*image)
	level := float32(radiance)
	exposed, err := glass.Expose(time.Second, []*frame.Frame{
		lit(whole, whole, level), lit(whole, whole, level),
		lit(whole, whole, level)})
	if err != nil {
		t.Fatal(err)
	}
	want := radiance * onAxis
	pixel := exposed.At(none, none)
	if got := float64(pixel.G); !(math.Abs(got-want) <= rawError*want) {
		t.Errorf("a second on the axis exposes %g, want %g", got, want)
	}
}

// Shutters open twice as long on a still scene take exactly twice the light bit
// for bit.
func TestTwiceTheShutterIsTwiceTheLight(t *testing.T) {
	rapid.Check(t, shutterDoublesTheLight)
}

func shutterDoublesTheLight(t *rapid.T) {
	glass := mustThin(t, focus)
	columns := rapid.IntRange(pictureLeast, pictureMost).Draw(t, "columns")
	rows := rapid.IntRange(pictureLeast, pictureMost).Draw(t, "rows")
	still := glimpsesOf(t, columns, rows)
	shift := rapid.IntRange(none, doublingsMost).Draw(t, "doublings")
	short := time.Second << shift
	once, err := glass.Expose(short, still)
	if err != nil {
		t.Fatal(err)
	}
	twice, err := glass.Expose(pair*short, still)
	if err != nil {
		t.Fatal(err)
	}
	for index := range once.Pix {
		if twice.Pix[index].R != pair*once.Pix[index].R {
			t.Fatalf("pixel %d: %g, then %g", index, once.Pix[index].R,
				twice.Pix[index].R)
		}
	}
}

// Scenes that change between glimpses take their mean for the whole
// time. One radiance and then another is the mean of them all along.
func TestAChangingSceneTakesTheMeanOfItsGlimpses(t *testing.T) {
	glass := mustThin(t, focus)
	dim, bright := float32(radiance), float32(radiance*pair)
	moving, err := glass.Expose(time.Second, []*frame.Frame{
		lit(pictureMost, pictureLeast, dim),
		lit(pictureMost, pictureLeast, bright)})
	if err != nil {
		t.Fatal(err)
	}
	mean := (dim + bright) / pair
	steady, err := glass.Expose(time.Second, []*frame.Frame{
		lit(pictureMost, pictureLeast, mean)})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := moving.At(none, none).B, steady.At(none, none).B; got != want {
		t.Errorf("one then another exposes %g, the mean all along %g",
			got, want)
	}
}

// One pixel exposed alone is the pixel of the whole frame bit for bit. That is
// what lets a replay prove it.
func TestAPixelExposedAloneIsTheFramesPixel(t *testing.T) {
	rapid.Check(t, pixelAloneIsTheFramesOwn)
}

func pixelAloneIsTheFramesOwn(t *rapid.T) {
	glass := mustThin(t, focus)
	columns := rapid.IntRange(pictureLeast, pictureMost).Draw(t, "columns")
	rows := rapid.IntRange(pictureLeast, pictureMost).Draw(t, "rows")
	caught := glimpsesOf(t, columns, rows)
	x := rapid.IntRange(none, columns-whole).Draw(t, "x")
	y := rapid.IntRange(none, rows-whole).Draw(t, "y")
	open := time.Duration(rapid.IntRange(whole, doublingsMost).Draw(t,
		"open")) * time.Second
	exposed, err := glass.Expose(open, caught)
	if err != nil {
		t.Fatal(err)
	}
	var seen []frame.Pixel
	for _, each := range caught {
		seen = append(seen, each.At(x, y))
	}
	alone := glass.Exposing(x, y, columns, rows, open, seen)
	if alone != exposed.At(x, y) {
		t.Fatalf("alone %+v, in the frame %+v", alone, exposed.At(x, y))
	}
}

// Exposing nothing at all is a pixel that met nothing. It has no light
// and no cover and is infinitely far.
func TestExposingNothingIsAPixelThatMetNothing(t *testing.T) {
	glass := mustThin(t, focus)
	got := glass.Exposing(none, none, whole, whole, time.Second, nil)
	want := frame.New(whole, whole).At(none, none)
	if got != want {
		t.Errorf("exposing nothing gave %+v, want %+v", got, want)
	}
}

// anti-tests: an exposure with no time is refused. So are one with no
// glimpses and one of glimpses that are not frames of one size. What
// comes back is no frame.
func TestAnExposureRefusesWhatItCannotBe(t *testing.T) {
	glass := mustThin(t, focus)
	small, wider := lit(pictureLeast, pictureLeast, none),
		lit(pictureMost, pictureLeast, none)
	taller := lit(pictureLeast, pictureMost, none)
	cases := map[string]struct {
		open     time.Duration
		glimpses []*frame.Frame
	}{
		"no time":           {none, []*frame.Frame{small}},
		"time below nought": {-time.Second, []*frame.Frame{small}},
		"no glimpses":       {time.Second, nil},
		"a wider glimpse":   {time.Second, []*frame.Frame{small, wider}},
		"a taller glimpse":  {time.Second, []*frame.Frame{small, taller}},
		"a missing second":  {time.Second, []*frame.Frame{small, nil}},
		"a missing first":   {time.Second, []*frame.Frame{nil, small}},
	}
	for name, made := range cases {
		got, err := glass.Expose(made.open, made.glimpses)
		if err == nil || got != nil {
			t.Errorf("%s was exposed: %v, %v", name, got, err)
		}
	}
}

// Exposures are the same forever: the same glimpses make the same frame.
func TestAnExposureIsTheSameForEver(t *testing.T) {
	rapid.Check(t, exposureIsStable)
}

func exposureIsStable(t *rapid.T) {
	glass := mustThin(t, focus)
	columns := rapid.IntRange(pictureLeast, pictureMost).Draw(t, "columns")
	rows := rapid.IntRange(pictureLeast, pictureMost).Draw(t, "rows")
	caught := glimpsesOf(t, columns, rows)
	first, err := glass.Expose(time.Second, caught)
	if err != nil {
		t.Fatal(err)
	}
	again, err := glass.Expose(time.Second, caught)
	if err != nil || !reflect.DeepEqual(first, again) {
		t.Fatalf("one exposure made two frames: %v", err)
	}
}

// Each channel of a pixel's exposure is the light of the glimpses added,
// then taken by the gathering and the share of time each glimpse stands
// for. Its cover is the mean of the glimpses' covers. Its depth is the
// nearest depth any glimpse met.
func TestExposingFollowsTheLawChannelByChannel(t *testing.T) {
	rapid.Check(t, exposingFollowsTheLaw)
}

// drawnPixel is a pixel of drawn light, cover and depth.
func drawnPixel(t *rapid.T) frame.Pixel {
	share := rapid.Float64Range(none, whole)
	cover := float32(share.Draw(t, "cover"))
	return frame.NewPixel(lamp(t, "red"), lamp(t, "green"),
		lamp(t, "blue"), cover, lamp(t, "depth"))
}

func exposingFollowsTheLaw(t *rapid.T) {
	glass := mustThin(t, focus)
	many := rapid.IntRange(whole, glimpsesMost).Draw(t, "glimpses")
	var seen []frame.Pixel
	for range many {
		seen = append(seen, drawnPixel(t))
	}
	x, y := pixel(t)
	open := time.Duration(rapid.IntRange(whole, doublingsMost).Draw(t,
		"open")) * time.Second
	held := glass.Gather(x, y, pixelsAcross, pixelsDown) * open.Seconds() /
		float64(many)
	var red, green, blue, cover float64
	depth := seen[none].Depth
	for _, each := range seen {
		red, green = red+float64(each.R), green+float64(each.G)
		blue, cover = blue+float64(each.B), cover+float64(each.Cover)
		depth = min(depth, each.Depth)
	}
	mean := float32(cover / float64(many))
	want := frame.NewPixel(float32(red*held), float32(green*held),
		float32(blue*held), mean, depth)
	got := glass.Exposing(x, y, pixelsAcross, pixelsDown, open, seen)
	if got != want {
		t.Fatalf("exposing gave %+v, want %+v", got, want)
	}
}
