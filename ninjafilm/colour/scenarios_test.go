package colour_test

import (
	"fmt"
	"hash/fnv"
	"image"
	"os"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/ninjafilm/colour"
	"github.com/janearc/slayer/ninjafilm/data"
	"github.com/janearc/slayer/ninjafilm/frame"
)

// what these tests count in, each defined
const (
	none  = 0 // defined: no amount at all
	whole = 1 // defined: one
	pair  = 2 // defined: two
)

// the tables the tests are run over
const (
	frameAFile   = "testdata/frame-a.csv"
	frameBFile   = "testdata/frame-b.csv"
	scenarioFile = "testdata/scenarios.csv"
)

// rows are the rows of a table in testdata, or the test stops.
func rows(t *testing.T, name string) []data.Row {
	t.Helper()
	text, err := os.ReadFile(name)
	if err != nil {
		t.Fatal(err)
	}
	found, err := data.Parse(string(text))
	if err != nil {
		t.Fatal(err)
	}
	return found
}

// amount is the number in a column of a row, or the test stops.
func amount(t *testing.T, row data.Row, column string) float64 {
	t.Helper()
	found, err := row.Float(column)
	if err != nil {
		t.Fatal(err)
	}
	return found
}

// text is the text in a column of a row, or the test stops.
func text(t *testing.T, row data.Row, column string) string {
	t.Helper()
	found, err := row.Text(column)
	if err != nil {
		t.Fatal(err)
	}
	return found
}

// light is the frame a table of pixels makes. There is a row for each
// pixel. It has the column x and the row y with red and green and blue.
func light(t *testing.T, name string) *frame.Frame {
	t.Helper()
	pixels := rows(t, name)
	var across, down int
	for _, row := range pixels {
		across = max(across, count(t, row, "x")+whole)
		down = max(down, count(t, row, "y")+whole)
	}
	made := frame.New(across, down)
	for _, row := range pixels {
		colours := maths.RGB(amount(t, row, "r"), amount(t, row, "g"),
			amount(t, row, "b"))
		made.Set(count(t, row, "x"), count(t, row, "y"), pixel(colours))
	}
	return made
}

// digest is a picture as a number: its bounds and then its bytes.
func digest(img *image.RGBA) string {
	sum := fnv.New64a()
	bounds := img.Bounds()
	fmt.Fprintf(sum, "%d,%d,%d,%d;", bounds.Min.X, bounds.Min.Y,
		bounds.Max.X, bounds.Max.Y)
	sum.Write(img.Pix)
	return fmt.Sprintf("%x", sum.Sum64())
}

// scene is the picture a row of the scenarios table asks for.
func scene(t *testing.T, row data.Row) *image.RGBA {
	t.Helper()
	seen, other := light(t, frameAFile), light(t, frameBFile)
	tone := colour.NewTone(amount(t, row, "exposure"), amount(t, row, "knee"),
		amount(t, row, "top"), amount(t, row, "bloom"))
	switch text(t, row, "kind") {
	case "develop":
		return tone.Develop(seen)
	case "false":
		band := maths.Band{Nanometres: amount(t, row, "band")}
		tint := maths.RGB(amount(t, row, "tint_r"),
			amount(t, row, "tint_g"), amount(t, row, "tint_b"))
		first := colour.NewLayer(maths.Band{}, seen, none, maths.Light{})
		gain := amount(t, row, "gain")
		second := colour.NewLayer(band, other, gain, tint)
		return tone.False([]colour.Layer{first, second})
	case "masked":
		return colour.Masked(colour.Develop(seen, whole),
			amount(t, row, "aspect"), amount(t, row, "pixel"))
	}
	developed := colour.Develop(seen, whole)
	seed := uint64(count(t, row, "seed"))
	return colour.Grained(developed, amount(t, row, "size"),
		amount(t, row, "amount"), seed)
}

// These are the pictures the audited darkroom made of two frames of
// light, by a digest of each picture. They are not right because they
// are these numbers. Changing a step of the darkroom cannot pass unseen.
func TestThePicturesAreTheOnesTheDarkroomAlwaysMade(t *testing.T) {
	for _, row := range rows(t, scenarioFile) {
		if got, want := digest(scene(t, row)), text(t, row, "digest"); got !=
			want {
			t.Errorf("%s came out %s, it always came out %s",
				text(t, row, "label"), got, want)
		}
	}
}

// what the other tests of the darkroom count in, each defined
const (
	bytes    = 256 // defined: how many values a byte holds
	byteMost = 255 // defined: the largest value a byte holds
	half     = 0.5 // defined: half a share
	four     = 4   // defined: four
)

// the tables the other tests are run over
const (
	encodeFile   = "testdata/encode.csv"
	decodeFile   = "testdata/decode.csv"
	shoulderFile = "testdata/shoulder.csv"
)

// what the other tests choose, each picked in picked-numbers.yml
var (
	within       = data.Picked("colour-within")
	lightMost    = data.Picked("colour-light-most")
	pictureMost  = data.Whole("colour-picture-most")
	kneeDrawn    = data.Picked("colour-knee")
	brightLight  = data.Picked("colour-bright")
	radiusLeast  = data.Picked("colour-radius-least")
	exposureMost = data.Picked("colour-exposure-most")
)

// refusalOf is what a call panics with as a string. It is nothing when
// the call does not panic.
func refusalOf(call func()) (said string) {
	defer recovered(&said)
	call()
	return said
}

func refusal(call func(float64), radius float64) (said string) {
	defer recovered(&said)
	call(radius)
	return said
}

// recovered puts what a panic was into a string.
func recovered(said *string) {
	*said, _ = recover().(string)
}

// count is the whole number in a column of a row, or the test stops.
func count(t *testing.T, row data.Row, column string) int {
	t.Helper()
	return int(amount(t, row, column))
}

// pixel is a pixel of light with full cover and no depth.
func pixel(light maths.Light) frame.Pixel {
	return frame.NewPixel(float32(light.R), float32(light.G),
		float32(light.B), whole, none)
}

// snapped is a number as a frame holds it, to a float32.
func snapped(number float64) float64 { return float64(float32(number)) }

// greyPixel is a pixel of three equal parts of light.
func greyPixel(level float64) frame.Pixel { return pixel(maths.Grey(level)) }
