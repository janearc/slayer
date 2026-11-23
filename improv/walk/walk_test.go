package walk

import (
	"errors"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/janearc/slayer/hud"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/set"
)

// markerPixels is how far a marker's traced middle may stand from its
// projected pixel. Picked here: a boulder of up to 2 m is a few pixels
// at the marker stills' 640 across, its middle known to about a pixel,
// and it stands up off its place by its own half height.
const markerPixels = 2.0

// markerDir is the traced marker fixture: the set, its places, and a
// mask of each camera's still, white where a marker glows.
var markerDir = filepath.Join("testdata", "marker")

// markerSet reads the fixture's set and places.
func markerSet(t *testing.T) (set.Set, Places) {
	t.Helper()
	path := filepath.Join(markerDir, "set.yaml")
	source, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	read, err := set.Read(path, source)
	if err != nil {
		t.Fatal(err)
	}
	table, err := os.ReadFile(filepath.Join(markerDir, "places.csv"))
	if err != nil {
		t.Fatal(err)
	}
	places, err := ReadPlaces(table)
	if err != nil {
		t.Fatal(err)
	}
	return read, places
}

// blobs is the middle of each patch of lit pixels in a mask, each
// pixel weighed by how brightly it glows.
func blobs(t *testing.T, path string) ([][2]float64, image.Point) {
	t.Helper()
	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	mask, err := png.Decode(file)
	if err != nil {
		t.Fatal(err)
	}
	size := mask.Bounds().Size()
	seen := make([]bool, size.X*size.Y)
	glow := func(x, y int) float64 {
		r, _, _, _ := mask.At(x, y).RGBA()
		return float64(r)
	}
	lit := func(x, y int) bool { return glow(x, y) > 0 }
	var middles [][2]float64
	for y := range size.Y {
		for x := range size.X {
			if seen[y*size.X+x] || !lit(x, y) {
				continue
			}
			var sumX, sumY, count float64
			stack := []image.Point{{x, y}}
			seen[y*size.X+x] = true
			for len(stack) > 0 {
				at := stack[len(stack)-1]
				stack = stack[:len(stack)-1]
				weight := glow(at.X, at.Y)
				sumX, sumY, count = sumX+weight*float64(at.X),
					sumY+weight*float64(at.Y), count+weight
				for _, step := range []image.Point{{1, 0}, {-1, 0}, {0, 1},
					{0, -1}} {
					next := at.Add(step)
					if !next.In(image.Rectangle{Max: size}) ||
						seen[next.Y*size.X+next.X] || !lit(next.X, next.Y) {
						continue
					}
					seen[next.Y*size.X+next.X] = true
					stack = append(stack, next)
				}
			}
			middles = append(middles, [2]float64{sumX / count, sumY / count})
		}
	}
	return middles, size
}

// leastMarkers is how many traced markers each camera's still must
// show for the test to mean anything. Picked here.
const leastMarkers = 3

// TestProjectMarkers: every marker the tracer drew on each camera's
// still lies within markerPixels of a marker's projected pixel. A
// marker the ground hides from the camera draws nothing, so the test
// goes from what was drawn to the places, not the other way.
func TestProjectMarkers(t *testing.T) {
	read, places := markerSet(t)
	for _, name := range []string{"down", "above"} {
		middles, size := blobs(t, filepath.Join(markerDir, name+".png"))
		view, err := CameraOf(read, name, places, size.X)
		if err != nil {
			t.Fatal(err)
		}
		if _, height := view.size(); height != size.Y {
			t.Fatalf("%s: the camera makes %d rows, the still has %d",
				name, height, size.Y)
		}
		if len(middles) < leastMarkers {
			t.Errorf("%s: %d markers drawn; the test sees too little",
				name, len(middles))
		}
		for _, middle := range middles {
			nearest, which := nearestMarker(view, places, middle)
			if nearest > markerPixels {
				t.Errorf("%s: a marker drawn at %.1f, %.1f is %.2f pixels "+
					"from the nearest projected, %s", name, middle[0],
					middle[1], nearest, which)
			}
		}
	}
}

// TestProjectMarkersNearMiss: the same camera with its eye 5 m to the
// side misses the drawn markers, so the test can tell a wrong camera.
func TestProjectMarkersNearMiss(t *testing.T) {
	read, places := markerSet(t)
	middles, size := blobs(t, filepath.Join(markerDir, "down.png"))
	view, err := CameraOf(read, "down", places, size.X)
	if err != nil {
		t.Fatal(err)
	}
	moved := view.eye
	moved.Origin = moved.Origin.Add(maths.V(0, 0, 5))
	wrong, err := newCamera(moved, view.glass, size.X)
	if err != nil {
		t.Fatal(err)
	}
	worst := 0.0
	for _, middle := range middles {
		nearest, _ := nearestMarker(wrong, places, middle)
		worst = math.Max(worst, nearest)
	}
	if worst <= markerPixels {
		t.Errorf("a camera 5 m off lands every marker within %.2f pixels",
			worst)
	}
}

// nearestMarker is how far a drawn marker is from the nearest marker's
// projected pixel, and which marker that is.
func nearestMarker(view Camera, places Places, middle [2]float64) (float64,
	string) {
	nearest, which := math.Inf(1), ""
	for name, at := range places.named {
		seen := view.Project(at)
		if !strings.HasPrefix(name, "close ") || !seen.Ahead() {
			continue
		}
		apart := math.Hypot(middle[0]-seen.X(), middle[1]-seen.Y())
		if apart < nearest {
			nearest, which = apart, name
		}
	}
	return nearest, which
}

// TestProjectBehindAndOff: a place behind the eye is said to be behind;
// one to the side, off the picture; a near miss just inside is on.
func TestProjectBehindAndOff(t *testing.T) {
	read, places := markerSet(t)
	view, err := CameraOf(read, "down", places, 320)
	if err != nil {
		t.Fatal(err)
	}
	eye := placeOf(view.eye.Origin)
	far, _ := places.of("the far tree")
	behind, _ := NewPlace(2*eye.east-far.east,
		2*eye.north-far.north, eye.height)
	if seen := view.Project(behind); seen.Ahead() || seen.on {
		t.Errorf("a place behind the eye is %+v", seen)
	}
	// north east, 65 degrees round from the line of sight: ahead, and
	// off the side of a picture 45 degrees across
	aside, _ := NewPlace(eye.east+300, eye.north+300, eye.height)
	if seen := view.Project(aside); !seen.Ahead() || seen.on {
		t.Errorf("a place well off the side is %+v", seen)
	}
	if seen := view.Project(far); !seen.Ahead() || !seen.on {
		t.Errorf("the far tree, which the camera points to, is %+v", seen)
	}
	if seen := view.toward(270, 10); seen.Ahead() {
		t.Errorf("a way behind the camera is ahead: %+v", seen)
	}
}

// TestCameraOfRefuses: a camera the set lacks, a focus not among the
// places, and a picture of no width are refused by name.
func TestCameraOfRefuses(t *testing.T) {
	read, places := markerSet(t)
	if _, err := CameraOf(read, "nowhere", places, 320); err == nil ||
		!strings.Contains(err.Error(), "nowhere") {
		t.Errorf("a camera the set lacks: %v", err)
	}
	if _, err := CameraOf(read, "down", Places{}, 320); err == nil {
		t.Error("a camera with no places was made")
	}
	if _, err := CameraOf(read, "down", places, 0); err == nil {
		t.Error("a picture of no width was made")
	}
	if _, err := newCamera(maths.Subspace{}, lens{}, 10); err == nil {
		t.Error("a camera with no lens was made")
	}
}

// overhead is a camera straight down from a height over a place, its
// rows running east, through a lens focused on the ground.
func overhead(t *testing.T, height float64, width int) (Camera, Place) {
	t.Helper()
	ground, _ := NewPlace(1000, 2000, 50)
	eye := ground.world().Add(maths.V(0, height, 0))
	glass, err := newLens(36, 36, 50, height)
	if err != nil {
		t.Fatal(err)
	}
	orientation := maths.LookAtUp(eye, ground.world(),
		maths.V(0, 0, -1))
	view, err := newCamera(maths.Subspace{Origin: eye,
		Orientation: orientation}, glass, width)
	if err != nil {
		t.Fatal(err)
	}
	return view, ground
}

// TestScaleBar: straight down from 500 m through a 50 mm lens on a 36
// mm frame focused on the ground, a metre is the film's image distance
// over the height, over the frame, times the width, in pixels.
func TestScaleBar(t *testing.T) {
	const height, width = 500.0, 1000
	view, ground := overhead(t, height, width)
	image := 50 * height / millimetre / (height/millimetre - 50)
	want := image / height / 36 * width
	per, err := view.PerMetre(ground)
	if err != nil || math.Abs(per-want) > 1e-9*want {
		t.Fatalf("a metre is %v pixels (%v); want %v", per, err, want)
	}
	scale, err := view.ScaleBar(ground, 400)
	if err != nil {
		t.Fatal(err)
	}
	// 400 pixels is 144 m at this height; the round length under it 100
	if scale.metres != 100 || math.Abs(scale.pixels-100*want) > 1e-6 ||
		scale.Words() != "100 m" {
		t.Errorf("the bar is %v m, %v pixels, %q", scale.metres,
			scale.pixels, scale.Words())
	}
	seen := scale.at
	if math.Abs(seen.X()-(width-1)/2.0) > 1e-6 {
		t.Errorf("the bar's place is at column %v, not the middle", seen.X())
	}
}

// TestScaleBarRefuses: no length, and a place behind, are refused; a
// near miss rounds down, not up.
func TestScaleBarRefuses(t *testing.T) {
	view, ground := overhead(t, 500, 1000)
	if _, err := view.ScaleBar(ground, 0); err == nil {
		t.Error("a bar of no pixels was made")
	}
	above, _ := NewPlace(ground.east, ground.north, 2000)
	if _, err := view.PerMetre(above); err == nil {
		t.Error("a bar behind the camera was measured")
	}
	for length, want := range map[float64]float64{999.9: 500, 1000: 1000,
		2001: 2000, 4.99: 2, 0.07: 0.05} {
		if got := roundDown(length); math.Abs(got-want) > 1e-12 {
			t.Errorf("%v rounds down to %v, not %v", length, got, want)
		}
	}
	if (Bar{metres: 2000, unit: metres}).Words() != "2 km" {
		t.Error("two thousand metres are not said in kilometres")
	}
}

// TestReadPlaces reads a table and refuses a wrong header, a word for a
// number, a name twice, and a place that is not finite.
func TestReadPlaces(t *testing.T) {
	header := "of,kind,east,north,height\n"
	read, err := ReadPlaces([]byte(header + "\"a b\",cobble,1,2,3\n"))
	if err != nil {
		t.Fatal(err)
	}
	if at, found := read.of("a b"); !found || at.height != 3 {
		t.Errorf("a b is %+v, %v", at, found)
	}
	if _, found := read.of("c"); found {
		t.Error("a name not in the table was found")
	}
	for _, table := range []string{
		"name,east\n",
		"",
		header + "a,cobble,one,2,3\n",
		header + "a,cobble,1,2,3\na,cobble,1,2,3\n",
		header + "a,cobble,1,2,NaN\n",
		header + "a,cobble,1,2\n",
	} {
		if _, err := ReadPlaces([]byte(table)); !errors.Is(err, errWalk) {
			t.Errorf("%q was read: %v", table, err)
		}
	}
}

// contrast is the ratio of two colours' relative luminances, the
// lighter's over the darker's, each with a twentieth added, as WCAG 2.1
// defines it: 1 to 21.
func contrast(one, other color.RGBA) float64 {
	a, b := luminance(one)+contrastFlare, luminance(other)+contrastFlare
	return math.Max(a, b) / math.Min(a, b)
}

// contrastFlare is the twentieth WCAG 2.1 adds to each luminance.
// Defined by WCAG 2.1, contrast ratio.
const contrastFlare = 0.05

// luminance is an sRGB colour's relative luminance, WCAG 2.1's weights
// of its linear channels.
func luminance(colour color.RGBA) float64 {
	return lumaRed*linear(colour.R) + lumaGreen*linear(colour.G) +
		lumaBlue*linear(colour.B)
}

// The channels' weights in luminance. Defined by WCAG 2.1, from
// ITU-R BT.709.
const lumaRed, lumaGreen, lumaBlue = 0.2126, 0.7152, 0.0722

// linear is an sRGB channel's linear light, as WCAG 2.1 gives it.
func linear(channel uint8) float64 {
	share := float64(channel) / math.MaxUint8
	if share <= srgbKnee {
		return share / srgbToe
	}
	return math.Pow((share+srgbOffset)/(1+srgbOffset), srgbPower)
}

// The sRGB curve's knee, toe slope, offset and power. Defined by
// WCAG 2.1, relative luminance.
const (
	srgbKnee   = 0.04045
	srgbToe    = 12.92
	srgbOffset = 0.055
	srgbPower  = 2.4
)

// TestContrast holds the frame's words to the house band, by
// arithmetic, and its accent to a state chip's level.
func TestContrast(t *testing.T) {
	tones, err := colours()
	if err != nil {
		t.Fatal(err)
	}
	ink := contrast(tones["ink"], tones["ground"])
	if ink < proseLeast || ink > proseMost {
		t.Errorf("ink on ground is %.2f to 1; the band is %v to %v", ink,
			proseLeast, proseMost)
	}
	if accent := contrast(tones["ok"], tones["ground"]); accent < chipLeast {
		t.Errorf("the accent on ground is %.2f to 1; under %v", accent,
			chipLeast)
	}
	white := tones["ink"]
	white.R, white.G, white.B = 255, 255, 255
	black := white
	black.R, black.G, black.B = 0, 0, 0
	if got := contrast(black, white); math.Abs(got-21) > 1e-9 {
		t.Errorf("black on white is %v, not 21", got)
	}
	if got := contrast(white, white); got != 1 {
		t.Errorf("white on white is %v, not 1", got)
	}
	if _, err := paletteOf(hud.Theme{}); err == nil {
		t.Error("a theme with no colours gave a palette")
	}
}
