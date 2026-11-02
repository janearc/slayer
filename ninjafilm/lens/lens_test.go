package lens_test

import (
	"math"
	"os"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/ninjafilm/data"
	"github.com/janearc/slayer/ninjafilm/lens"
	"pgregory.net/rapid"
)

// what these tests count in, each defined
const (
	millimetre = 1e-3 // defined: a millimetre is a thousandth of a metre
	half       = 0.5  // defined: half of a width, or of a share
	pair       = 2    // defined: a count of two
	fourth     = 4    // defined: light off the axis falls as the fourth power of the cosine
	none       = 0    // defined: no amount at all
	whole      = 1    // defined: the whole of a share, or one pixel
	upward     = 1    // defined: math.Inf's code for infinity above all numbers
	downward   = -1   // defined: math.Inf's code for infinity below all numbers
	either     = 0    // defined: math.IsInf's code for infinity of either sign
)

// the tables the tests are run over
const (
	framesFile = "testdata/frames-refused.csv"
	lensesFile = "testdata/lenses-refused.csv"
)

// what these tests choose, each picked in picked-numbers.yml
var (
	frameWide      = data.Picked("lens-frame-wide")
	frameHigh      = data.Picked("lens-frame-high")
	focal          = data.Picked("lens-focal")
	stop           = data.Picked("lens-stop")
	focus          = data.Picked("lens-focus")
	farFocus       = data.Picked("lens-far-focus")
	pixelsAcross   = data.Whole("lens-across")
	pixelsDown     = data.Whole("lens-down")
	missError      = data.Picked("lens-miss-error")
	apertureAcross = data.Whole("lens-aperture-across")
	apertureDown   = data.Whole("lens-aperture-down")
	apertureError  = data.Picked("lens-aperture-error")
	edgeError      = data.Picked("lens-edge-error")
	millimetresMax = data.Picked("lens-millimetres-most")
	millimetresMin = data.Picked("lens-millimetres-least")
	focalMost      = data.Picked("lens-focal-most")
	focalLeast     = data.Picked("lens-focal-least")
	stopMost       = data.Picked("lens-stop-most")
	focusFactor    = data.Picked("lens-focus-factor")
	focusMost      = data.Picked("lens-focus-most")
	oddLeast       = data.Picked("lens-odd-least")
	oddMost        = data.Picked("lens-odd-most")
	toleranceTight = data.Picked("tolerance-tight")
	rayError       = data.Picked("lens-ray-error")
)

// stopper is a test that can be stopped: a testing.T, or a rapid.T.
type stopper interface {
	Helper()
	Fatal(args ...any)
}

// mustThin is the shot's lens focused so far off, or the test stops.
func mustThin(t stopper, away float64) lens.Thin {
	t.Helper()
	frame, err := lens.FrameOf(frameWide, frameHigh)
	if err != nil {
		t.Fatal(err)
	}
	glass, err := lens.NewThin(frame, focal, stop, away)
	if err != nil {
		t.Fatal(err)
	}
	return glass
}

// imageAt is how far behind a lens the image of a point so far in front
// stands by the lens law. One over the focal length is one over each
// distance added. Lengths are in metres.
func imageAt(length, distance float64) float64 {
	return whole / (whole/length - whole/distance)
}

// apart is how far a point is from the line a ray runs along.
func apart(ray maths.Ray, point maths.Vec) float64 {
	off := point.Sub(ray.At)
	along := ray.Dir.Scale(off.Dot(ray.Dir))
	return off.Sub(along).Len()
}

// pixel is a pixel of a picture the shot's size, drawn.
func pixel(t *rapid.T) (x, y int) {
	x = rapid.IntRange(none, pixelsAcross-whole).Draw(t, "x")
	y = rapid.IntRange(none, pixelsDown-whole).Draw(t, "y")
	return x, y
}

// Every ray from a pixel goes through the point at the focus distance
// that the pixel's middle on the film sees through the lens's middle,
// where the lens law puts the film. Rays from a point on the film meet
// at its image, whatever part of the aperture they cross.
func TestARaySeesSharpWhereTheLensLawSays(t *testing.T) {
	rapid.Check(t, rayAimsAtItsSharpPoint)
}

func rayAimsAtItsSharpPoint(t *rapid.T) {
	glass := mustThin(t, focus)
	image := imageAt(focal*millimetre, focus)
	x, y := pixel(t)
	wide, high := frameWide*millimetre, frameHigh*millimetre
	right := ((float64(x)+half)/float64(pixelsAcross) - half) * wide
	up := (half - (float64(y)+half)/float64(pixelsDown)) * high
	sharp := maths.V(right*focus/image, up*focus/image, -focus)
	ray := glass.Ray(x, y, pixelsAcross, pixelsDown, maths.Band{})
	if miss := apart(ray, sharp); !(miss <= missError*focus) {
		t.Fatalf("pixel %d,%d misses its sharp point by %g m", x, y, miss)
	}
}

// The lens sees across the frame twice the angle whose tangent is half
// the frame's width over the film's distance. Focused far off that is
// the focal length's, about six degrees for 672 mm on 15/70.
func TestALensSeesWhatItsFrameSays(t *testing.T) {
	near, far := mustThin(t, focus), mustThin(t, farFocus)
	wide := frameWide * millimetre
	image := imageAt(focal*millimetre, focus)
	wantNear := pair * math.Atan(wide/pair/image)
	wantFar := pair * math.Atan(wide/pair/(focal*millimetre))
	if math.Abs(near.Wide()-wantNear) > toleranceTight ||
		math.Abs(far.Wide()-wantFar) > toleranceTight {
		t.Errorf("wide %g and %g, want %g and %g", near.Wide(), far.Wide(),
			wantNear, wantFar)
	}
	seen := wide / float64(pixelsAcross) / image
	if got := near.Pixel(pixelsAcross); !(math.Abs(got-seen) <=
		toleranceTight) {
		t.Errorf("a pixel sees %g, want %g", got, seen)
	}
}

// Rays leave from all over the open aperture. It is half the focal
// length over the f-number across. They leave evenly. Over a round
// opening of radius a the mean of the distance squared from the middle
// is a squared over two.
func TestTheApertureIsOpenAsTheStopSays(t *testing.T) {
	glass := mustThin(t, focus)
	reach := focal * millimetre / (pair * stop)
	var widest, total float64
	for y := range apertureDown {
		for x := range apertureAcross {
			start := glass.Ray(x, y, apertureAcross, apertureDown,
				maths.Band{}).At
			if start.Z != none {
				t.Fatalf("a ray starts off the lens: %v", start)
			}
			out := start.X*start.X + start.Y*start.Y
			widest, total = math.Max(widest, out), total+out
		}
	}
	mean := total / float64(apertureAcross*apertureDown)
	wantMean := reach * reach * half
	if !(math.Sqrt(widest) <= reach*(whole+edgeError)) ||
		!(math.Abs(mean-wantMean) <= apertureError*wantMean) {
		t.Errorf("widest %g of %g, mean square %g of %g",
			math.Sqrt(widest), reach, mean, wantMean)
	}
}

// On the axis the lens gathers pi times the sine squared of the angle to
// the edge of the aperture. That is pi over four times the working
// f-number squared. The working f-number is the f-number times one
// plus the magnification. The sine has its own correction of one over
// one plus a quarter over that squared. At the corner the light falls
// by the cosine to the fourth.
func TestTheLensGathersAsTheLawSays(t *testing.T) {
	glass := mustThin(t, focus)
	image := imageAt(focal*millimetre, focus)
	working := stop * image / (focal * millimetre)
	paraxial := math.Pi / (fourth * working * working)
	want := paraxial / (whole + whole/(fourth*working*working))
	if got := glass.Gather(none, none, whole, whole); !(math.Abs(got-want) <=
		edgeError*want) {
		t.Errorf("on the axis %g, want %g", got, want)
	}
	cornerRight := (half - half/float64(pixelsAcross)) * frameWide *
		millimetre
	cornerUp := (half - half/float64(pixelsDown)) * frameHigh * millimetre
	corner := math.Hypot(cornerRight, cornerUp)
	cosine := image / math.Hypot(image, corner)
	edge := want * math.Pow(cosine, fourth)
	got := glass.Gather(none, none, pixelsAcross, pixelsDown)
	if !(math.Abs(got-edge) <= edgeError*edge) {
		t.Errorf("at the corner %g, want %g", got, edge)
	}
}

// table is the rows of a table of cases in testdata, or the test stops.
func table(t *testing.T, name string) []data.Row {
	t.Helper()
	text, err := os.ReadFile(name)
	if err != nil {
		t.Fatal(err)
	}
	rows, err := data.Parse(string(text))
	if err != nil {
		t.Fatal(err)
	}
	return rows
}

// number is the number in a column of a row, or the test stops.
func number(t *testing.T, row data.Row, column string) float64 {
	t.Helper()
	amount, err := row.Float(column)
	if err != nil {
		t.Fatal(err)
	}
	return amount
}

// anti-tests: a frame with no width is not made. Nor is one with a
// negative height. Lenses with no focal length are not made. Nor are
// those with a stop below a half or a focus nearer than the focal
// length. What comes back is the zero value.
func TestALensRefusesWhatItCannotBe(t *testing.T) {
	var noFrame lens.Frame
	for _, row := range table(t, framesFile) {
		width, height := number(t, row, "width"), number(t, row, "height")
		if got, err := lens.FrameOf(width, height); err == nil ||
			got != noFrame {
			t.Errorf("a frame %v by %v was made: %v", width, height, got)
		}
	}
	frame, err := lens.FrameOf(frameWide, frameHigh)
	if err != nil {
		t.Fatal(err)
	}
	var noLens lens.Thin
	for _, row := range table(t, lensesFile) {
		length, aperture := number(t, row, "focal"), number(t, row, "stop")
		away := number(t, row, "focus")
		if got, err := lens.NewThin(frame, length, aperture, away); err ==
			nil || got != noLens {
			t.Errorf("a lens %v %v %v was made: %v", length, aperture,
				away, got)
		}
	}
}

// odd draws an ordinary number or an odd one.
func odd() *rapid.Generator[float64] {
	strange := rapid.SampledFrom([]float64{math.NaN(), math.Inf(upward),
		math.Inf(downward), none})
	return rapid.OneOf(rapid.Float64Range(oddLeast, oddMost), strange)
}

// sized is whether a number is greater than zero and finite.
func sized(amount float64) bool {
	return amount > none && !math.IsInf(amount, either)
}

// Whether a frame is made follows from its numbers: each is above zero and
// finite.
func TestAFrameIsMadeExactlyWhenItHasASize(t *testing.T) {
	rapid.Check(t, frameIsMadeWhenItHasASize)
}

func frameIsMadeWhenItHasASize(t *rapid.T) {
	width, height := odd().Draw(t, "width"), odd().Draw(t, "height")
	want := sized(width) && sized(height)
	if _, err := lens.FrameOf(width, height); (err == nil) != want {
		t.Fatalf("a frame %v by %v: made is %v, want %v", width, height,
			err == nil, want)
	}
}

// Whether a lens is made follows from its numbers. The focal length is greater
// than zero and finite. The f-number is a half or more and finite. The focus is
// further off than the focal length and finite.
func TestALensIsMadeExactlyWhenItCanBe(t *testing.T) {
	rapid.Check(t, lensIsMadeWhenItCan)
}

func lensIsMadeWhenItCan(t *rapid.T) {
	frame, err := lens.FrameOf(frameWide, frameHigh)
	if err != nil {
		t.Fatal(err)
	}
	length, aperture := odd().Draw(t, "focal"), odd().Draw(t, "stop")
	away := odd().Draw(t, "focus")
	metres := length * millimetre
	want := length > none && !math.IsInf(length, either) &&
		aperture >= half && !math.IsInf(aperture, either) &&
		away > metres && !math.IsInf(away, either)
	if _, err := lens.NewThin(frame, length, aperture, away); (err == nil) !=
		want {
		t.Fatalf("a lens of %v, f/%v at %v: made is %v, want %v", length,
			aperture, away, err == nil, want)
	}
}

// Drawn lenses are made. They see what their frames say. The angle
// across grows with the frame. Each pixel sees its share of it.
func TestDrawnLensesAreMadeAndSeeWhatTheirFramesSay(t *testing.T) {
	rapid.Check(t, drawnLensSeesItsFrame)
}

func drawnLensSeesItsFrame(t *rapid.T) {
	wide := rapid.Float64Range(millimetresMin, millimetresMax).Draw(t, "w")
	high := rapid.Float64Range(millimetresMin, millimetresMax).Draw(t, "h")
	length := rapid.Float64Range(focalLeast, focalMost).Draw(t, "focal")
	aperture := rapid.Float64Range(half, stopMost).Draw(t, "stop")
	away := length * millimetre * rapid.Float64Range(focusFactor,
		focusMost).Draw(t, "factor")
	frame, err := lens.FrameOf(wide, high)
	if err != nil {
		t.Fatal(err)
	}
	glass, err := lens.NewThin(frame, length, aperture, away)
	if err != nil {
		t.Fatalf("a lens of %v, f/%v at %v was refused: %v", length,
			aperture, away, err)
	}
	across := rapid.IntRange(whole, pixelsAcross).Draw(t, "across")
	image := imageAt(length*millimetre, away)
	wantWide := pair * math.Atan(wide*millimetre/pair/image)
	if got := glass.Wide(); math.Abs(got-wantWide) > toleranceTight*wantWide {
		t.Fatalf("the lens sees %v, want %v", got, wantWide)
	}
	wantPixel := wide * millimetre / float64(across) / image
	if got := glass.Pixel(across); math.Abs(got-wantPixel) >
		toleranceTight*wantPixel {
		t.Fatalf("a pixel sees %v, want %v", got, wantPixel)
	}
	rows := frame.Rows(across)
	square := float64(across) * high / wide
	if math.Abs(float64(rows)-square) > half {
		t.Fatalf("%d rows for %d across a %v by %v frame, want %v", rows,
			across, wide, high, square)
	}
}

// what the closed form of the plastic number counts in, each defined
const (
	ninth       = 9           // defined: the closed form of the plastic number takes the cube roots of nine plus and minus the root of sixty nine, over eighteen
	sixtyNinth  = 69          // defined: see ninth
	eighteenth  = 18          // defined: see ninth
	eighthTurn  = math.Pi / 4 // defined: an eighth of a turn
	quarterTurn = math.Pi / 2 // defined: a quarter of a turn
)

// plasticNumber is the real root of x cubed equals x plus one, from its
// closed form. It is worked here and not read from the lens.
func plasticNumber() float64 {
	root := math.Sqrt(sixtyNinth)
	plus := math.Cbrt((ninth + root) / eighteenth)
	minus := math.Cbrt((ninth - root) / eighteenth)
	return plus + minus
}

// startOf is where the published r2 sequence and the concentric map put
// the start of the ray for the pixel at an index. The aperture has a
// radius. It is worked from their definitions and not from the lens.
func startOf(index int, reach float64) maths.Vec {
	rho := plasticNumber()
	abscissa := math.Mod(half+float64(index)/rho, whole)*pair - whole
	ordinate := math.Mod(half+float64(index)/(rho*rho), whole)*pair - whole
	if abscissa == none && ordinate == none {
		return maths.Vec{}
	}
	radius, turn := ordinate, quarterTurn-eighthTurn*(abscissa/ordinate)
	if math.Abs(abscissa) > math.Abs(ordinate) {
		radius, turn = abscissa, eighthTurn*(ordinate/abscissa)
	}
	return maths.V(reach*radius*math.Cos(turn), reach*radius*math.Sin(turn),
		none)
}

// Rays start where the r2 sequence of the pixel puts them on the
// aperture, carried onto the round opening by the concentric map.
func TestARayStartsWhereTheSequenceAndTheMapPutIt(t *testing.T) {
	rapid.Check(t, rayStartsWhereTheSequencePutsIt)
}

func rayStartsWhereTheSequencePutsIt(t *rapid.T) {
	glass := mustThin(t, focus)
	x, y := pixel(t)
	reach := focal * millimetre / (pair * stop)
	want := startOf(y*pixelsAcross+x, reach)
	got := glass.Ray(x, y, pixelsAcross, pixelsDown, maths.Band{}).At
	if got.Sub(want).Len() > rayError*reach {
		t.Fatalf("pixel %d,%d starts at %v, want %v", x, y, got, want)
	}
}
