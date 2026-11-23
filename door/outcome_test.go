package door

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"math"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/janearc/slayer/sources"
)

// the preview draw's budget. the first camera of sets/futian-shore.yaml
// at 144 by 81 on the gpu with scrooge on is drawn twice into one empty
// cache, each time fresh, so that the picture's rows are drawn again but
// what is worked out from the shelf stays kept after the first draw.
//
// the first, cold draw works the sources out as well as drawing. it
// measured about seven and a half minutes, and its limit is not a
// multiple of that: it is the bar the owner set for what this picture
// must never again take, a preview in under thirty minutes.
//
// the second, warm draw measured 108 seconds, staging included, and its
// limit is three times that, so a busy machine does not trip it and a
// real regression of the staging, scrooge or the kernel does.
//
// the peak resident memory of both measured 8.6 GB, and its limit is one
// and a half times that, rounded up to 13 GB.
const (
	measuredColdTime      = 450 * time.Second // defined: measured, cold draw
	coldTimeLimit         = 30 * time.Minute  // defined: the owner's bar
	measuredWarmTime      = 108 * time.Second // defined: measured, warm draw
	warmTimeFactor        = 3                 // defined: slack on the warm time
	warmTimeLimit         = warmTimeFactor * measuredWarmTime
	measuredPreviewMemory = 8_600_000_000  // defined: bytes, measured peak
	previewMemoryLimit    = 13_000_000_000 // defined: bytes, 1.5 times
)

// drawBudget is what the two preview draws took, or what they may take:
// the time of the cold draw, the time of the warm draw, and the peak
// resident memory in bytes.
type drawBudget struct {
	cold, warm time.Duration
	resident   int64
}

// limitsBroken is how many limits the preview draws have: the cold
// draw's time, the warm draw's time and the memory.
const limitsBroken = 3 // defined: the limits of a drawBudget

// names is the budget's three numbers as a report says them.
func (budget drawBudget) names() []string {
	return []string{budget.cold.String(), budget.warm.String(),
		fmt.Sprint(budget.resident)}
}

// the budget the preview draws are held to, and the numbers measured that
// it comes from.
var (
	previewLimits = drawBudget{cold: coldTimeLimit, warm: warmTimeLimit,
		resident: previewMemoryLimit}
	previewMeasured = drawBudget{cold: measuredColdTime,
		warm: measuredWarmTime, resident: measuredPreviewMemory}
)

// the haze's speck limit. camera street of the same set at the same size
// drew 499 specks before the speck fix and 64 after it; the limit is the
// geometric mean of the two, rounded.
const (
	measuredSpecksBefore = 499 // defined: specks before the fix
	measuredSpecksAfter  = 64  // defined: specks after the fix
	speckLimit           = 180 // defined: most specks the haze may have
)

// the numbers a speck is counted by. a speck is a pixel off the picture's
// edge whose linear luminance is over speckFloor and over speckRatio times
// the median of its eight neighbours', which is the mean of the fourth and
// fifth of them in order.
const (
	speckFloor     = 0.05 // defined: linear luminance a speck is over
	speckRatio     = 6    // defined: times its neighbours' median
	neighbourCount = 8    // defined: pixels round a pixel
	lowerMiddle    = 3    // defined: the fourth of eight, counted from nought
	upperMiddle    = 4    // defined: the fifth of eight, counted from nought
	reach          = 1    // defined: pixels from a pixel to its neighbours
)

// rec. 709 weights of the linear channels in luminance, and the sRGB
// curve that decodes a coded channel to a linear one.
const (
	redWeight     = 0.2126  // defined: Rec. 709, red
	greenWeight   = 0.7152  // defined: Rec. 709, green
	blueWeight    = 0.0722  // defined: Rec. 709, blue
	codedKnee     = 0.04045 // defined: sRGB, where the curve leaves its line
	linearSlope   = 12.92   // defined: sRGB, the slope of the line
	codedOffset   = 0.055   // defined: sRGB, offset of the curve
	codedScale    = 1.055   // defined: sRGB, scale of the curve
	codedExponent = 2.4     // defined: sRGB, exponent of the curve
	channelFull   = 65535.0 // defined: a channel's most, as RGBA gives it
)

// budgetFaults is what the draws broke of their limits: one line for the
// cold draw when it is over its time, one for the warm draw when it is
// over its time, and one for the memory when it is over its bytes. each
// says which and the number and the limit. draws inside all three give
// none.
func budgetFaults(took, limits drawBudget) []string {
	var faults []string
	if took.cold > limits.cold {
		faults = append(faults, fmt.Sprintf("the cold draw took %v, over "+
			"the limit of %v", took.cold, limits.cold))
	}
	if took.warm > limits.warm {
		faults = append(faults, fmt.Sprintf("the warm draw took %v, over "+
			"the limit of %v", took.warm, limits.warm))
	}
	if took.resident > limits.resident {
		faults = append(faults, fmt.Sprintf("peak resident memory was "+
			"%d bytes, over the limit of %d", took.resident,
			limits.resident))
	}
	return faults
}

// budgetReport says the faults of the preview draws, and then what each
// draw took, its limit and the number measured that the limit comes from.
func budgetReport(took drawBudget, faults []string) string {
	return fmt.Sprintf("the preview draws broke their budget (%s): the cold "+
		"draw took %v against a limit of %v, measured %v; the warm draw "+
		"took %v against a limit of %v, measured %v; peak resident memory "+
		"was %d bytes against a limit of %d, measured %d",
		strings.Join(faults, "; "), took.cold, previewLimits.cold,
		previewMeasured.cold, took.warm, previewLimits.warm,
		previewMeasured.warm, took.resident, previewLimits.resident,
		previewMeasured.resident)
}

// speckReport says how many specks a picture has, the limit, and the two
// counts the limit lies between.
func speckReport(count int) string {
	return fmt.Sprintf("the haze has %d specks, over the limit of %d; "+
		"measured %d after the speck fix and %d before it", count,
		speckLimit, measuredSpecksAfter, measuredSpecksBefore)
}

// shelfRefusal is why the city draw tests will not run, given the error
// sources.Here gave for the shelf: its own words, after what the test
// draws and why it needs a shelf. a shelf found gives no refusal.
func shelfRefusal(err error) string {
	if err == nil {
		return ""
	}
	return "this test draws sets/futian-shore.yaml, which reads surveyed " +
		"sources from the shelf, and will not run without one: " +
		err.Error()
}

// linearOf decodes one sRGB channel, from nought to one, to a linear one.
func linearOf(coded float64) float64 {
	if coded <= codedKnee {
		return coded / linearSlope
	}
	return math.Pow((coded+codedOffset)/codedScale, codedExponent)
}

// luminanceAt is the linear luminance of the pixel at x, y.
func luminanceAt(picture image.Image, x, y int) float64 {
	red, green, blue, _ := picture.At(x, y).RGBA()
	return redWeight*linearOf(float64(red)/channelFull) +
		greenWeight*linearOf(float64(green)/channelFull) +
		blueWeight*linearOf(float64(blue)/channelFull)
}

// neighbourMedian is the median luminance of the eight pixels round x, y:
// the mean of the fourth and fifth in order.
func neighbourMedian(picture image.Image, x, y int) float64 {
	around := make([]float64, nothing, neighbourCount)
	for dy := -reach; dy <= reach; dy++ {
		for dx := -reach; dx <= reach; dx++ {
			if dx != nothing || dy != nothing {
				around = append(around, luminanceAt(picture, x+dx, y+dy))
			}
		}
	}
	sort.Float64s(around)
	return (around[lowerMiddle] + around[upperMiddle]) / pair
}

// specks counts the pixels of a picture that stand far above the pixels
// round them. a pixel on the picture's edge has no eight neighbours and
// is never counted. a bright block is not a speck inside, where its
// pixels have bright neighbours, but its corners are, for only three of
// their eight neighbours are bright.
func specks(picture image.Image) int {
	bounds := picture.Bounds()
	count := nothing
	for y := bounds.Min.Y + reach; y < bounds.Max.Y-reach; y++ {
		for x := bounds.Min.X + reach; x < bounds.Max.X-reach; x++ {
			here := luminanceAt(picture, x, y)
			if here > speckFloor &&
				here > speckRatio*neighbourMedian(picture, x, y) {
				count++
			}
		}
	}
	return count
}

// the grey picture the speck anti-tests plant bright pixels in: its side,
// the coded grey of its ground, which is under the speck floor in linear
// luminance, and the coded grey of what is planted, which is white.
const (
	plantedSide    = 24  // defined: pixels across and down
	groundGrey     = 40  // defined: coded grey, linear 0.02
	plantedGrey    = 255 // defined: coded grey, white
	blockSide      = 3   // defined: pixels across and down of a block
	blockCorners   = 4   // defined: the corners of a block
	picturesEdgeAt = 0   // defined: the first row and column
)

// planted is the grey picture with a bright pixel at each point of
// singles, and, where block says so, a bright blockSide by blockSide
// block whose first corner is at blockAt.
func planted(singles []image.Point, block bool,
	blockAt image.Point) *image.Gray {
	bounds := image.Rect(nothing, nothing, plantedSide, plantedSide)
	picture := image.NewGray(bounds)
	draw.Draw(picture, bounds, image.NewUniform(color.Gray{Y: groundGrey}),
		image.Point{}, draw.Src)
	for _, point := range singles {
		picture.SetGray(point.X, point.Y, color.Gray{Y: plantedGrey})
	}
	if block {
		for y := blockAt.Y; y < blockAt.Y+blockSide; y++ {
			for x := blockAt.X; x < blockAt.X+blockSide; x++ {
				picture.SetGray(x, y, color.Gray{Y: plantedGrey})
			}
		}
	}
	return picture
}

// the places the anti-tests plant: single pixels well inside the picture
// and apart from the block, one bright pixel on the picture's edge, and
// the block's first corner.
var (
	singlePlaces = []image.Point{{X: 4, Y: 4}, {X: 12, Y: 6},
		{X: 20, Y: 10}, {X: 8, Y: 20}}
	edgePlace  = image.Point{X: picturesEdgeAt, Y: 9}
	blockPlace = image.Point{X: 14, Y: 14}
)

// each limit broken alone, by a second or a byte, is reported by name and
// nothing else is: the cold draw's time, the warm draw's time and the
// memory, the others being as measured.
func TestTheBudgetCheckReportsEachLimitBrokenAlone(t *testing.T) {
	over := map[string]drawBudget{
		"the cold draw": {previewLimits.cold + time.Second,
			previewMeasured.warm, previewMeasured.resident},
		"the warm draw": {previewMeasured.cold,
			previewLimits.warm + time.Second, previewMeasured.resident},
		"peak resident memory": {previewMeasured.cold, previewMeasured.warm,
			previewLimits.resident + one},
	}
	for name, took := range over {
		faults := budgetFaults(took, previewLimits)
		if len(faults) != one || !strings.Contains(faults[nothing], name) {
			t.Errorf("%s over its limit gave the faults %q", name, faults)
		}
	}
}

// the draws over every limit are reported for each, and the report says
// the numbers a person reads to see how far over they are.
func TestTheBudgetReportNamesEveryLimitAndEveryMeasuredNumber(t *testing.T) {
	took := drawBudget{previewLimits.cold * pair, previewLimits.warm * pair,
		previewLimits.resident * pair}
	faults := budgetFaults(took, previewLimits)
	if len(faults) != limitsBroken {
		t.Fatalf("draws over every limit gave the faults %q", faults)
	}
	report := budgetReport(took, faults)
	var want []string
	for _, each := range []drawBudget{took, previewLimits, previewMeasured} {
		want = append(want, each.names()...)
	}
	for _, say := range want {
		if !strings.Contains(report, say) {
			t.Errorf("the report %q does not say %s", report, say)
		}
	}
}

// the numbers measured are inside the limits, and so are draws at the
// limits, so the check does not report draws it should pass. the cold
// draw as measured is over the warm draw's limit and is not faulted for
// it.
func TestTheBudgetCheckPassesDrawsInsideTheirLimits(t *testing.T) {
	for name, took := range map[string]drawBudget{
		"as measured":   previewMeasured,
		"at the limits": previewLimits,
	} {
		if faults := budgetFaults(took, previewLimits); len(faults) !=
			nothing {
			t.Errorf("draws %s were faulted: %q", name, faults)
		}
	}
}

// the specks of a picture with single bright pixels planted in it are
// exactly those pixels: one on the picture's edge is not counted, and a
// bright block adds its four corners and nothing from its edges or middle.
func TestTheSpeckCountFindsEachPlantedPixelAndNoBlockInterior(t *testing.T) {
	singles := append([]image.Point{edgePlace}, singlePlaces...)
	got := specks(planted(singles, true, blockPlace))
	want := len(singlePlaces) + blockCorners
	if got != want {
		t.Errorf("the count is %d, want %d: %d single pixels and the "+
			"four corners of the block", got, want, len(singlePlaces))
	}
}

// a block alone counts as its four corners, so its edges and its middle,
// whose neighbours are bright, are not specks.
func TestTheSpeckCountOfABlockIsItsCornersAlone(t *testing.T) {
	if got := specks(planted(nil, true, blockPlace)); got != blockCorners {
		t.Errorf("the count is %d, want the block's %d corners", got,
			blockCorners)
	}
}

// a flat picture has no specks, so the count does not find any where
// there is none.
func TestTheSpeckCountOfAFlatPictureIsNone(t *testing.T) {
	if got := specks(planted(nil, false, image.Point{})); got != nothing {
		t.Errorf("the count is %d, want none", got)
	}
}

// a shelf that is not named, or is named by a relative path, is refused
// in the words of sources, which name the variable and how to set it.
func TestTheCityDrawsRefuseAShelfThatIsUnsetOrRelative(t *testing.T) {
	for name, shelf := range map[string]string{
		"unset": "", "relative": "sources/shelf"} {
		t.Setenv(sources.Placed, shelf)
		_, err := sources.Here()
		refusal := shelfRefusal(err)
		if !strings.Contains(refusal, sources.Placed) ||
			!strings.Contains(refusal, "futian-shore.yaml") {
			t.Errorf("a shelf %s gave the refusal %q", name, refusal)
		}
	}
}

// a shelf named by an absolute path is not refused.
func TestTheCityDrawsDoNotRefuseAnAbsoluteShelf(t *testing.T) {
	t.Setenv(sources.Placed, t.TempDir())
	_, err := sources.Here()
	if refusal := shelfRefusal(err); refusal != "" {
		t.Errorf("an absolute shelf was refused: %q", refusal)
	}
}

// the speck report says the count, the limit and the two counts the limit
// lies between.
func TestTheSpeckReportNamesTheLimitAndTheMeasuredCounts(t *testing.T) {
	report := speckReport(speckLimit + one)
	for _, want := range []int{speckLimit + one, speckLimit,
		measuredSpecksAfter, measuredSpecksBefore} {
		if !strings.Contains(report, fmt.Sprint(want)) {
			t.Errorf("the report %q does not say %d", report, want)
		}
	}
}
