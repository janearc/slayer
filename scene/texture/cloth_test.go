package texture

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
)

// clothSamples are points round a vertical axis, low and high, for the
// cloth textures to be asked at.
func clothSamples() []maths.Vec {
	var points []maths.Vec
	for turn := 0; turn < 40; turn++ {
		angle := float64(turn) * 0.37
		for _, height := range []float64{0.01, 0.4, 1.3, 2.5} {
			sine := math.Sin(angle)
			cosine := math.Cos(angle)
			points = append(points, maths.V(3*sine, height,
				3*cosine))
		}
	}
	return points
}

// lit is whether every part of a light is a finite share from nought to
// one.
func lit(light maths.Light) bool {
	for _, part := range []float64{light.R, light.G, light.B} {
		if math.IsNaN(part) || part < 0 || part > 1 {
			return false
		}
	}
	return true
}

func aCanvas() Canvas {
	return Canvas{Cloth: maths.RGB(0.8, 0.77, 0.68),
		Soil: maths.RGB(0.2, 0.15, 0.1), Pitch: 0.012, Depth: 0.35,
		Radius: 3, SoilHeight: 0.6, Soiling: 0.75, Streaks: 0.25}
}

func TestCanvasIsAWeaveSoiledNearTheGroundAndCleanerHigher(t *testing.T) {
	canvas := aCanvas()
	weaves := map[maths.Light]bool{}
	low, high := 0.0, 0.0
	for _, point := range clothSamples() {
		if !lit(canvas.At(point)) {
			t.Fatalf("the canvas at %v is %v", point, canvas.At(point))
		}
		weaves[canvas.At(point)] = true
		if point.Y < 0.1 {
			low += canvas.At(point).Luminance()
		}
		if point.Y > 2 {
			high += canvas.At(point).Luminance()
		}
	}
	if len(weaves) < 20 || low >= high {
		t.Errorf("%d colours, and %v of light low against %v high",
			len(weaves), low, high)
	}
}

func TestCanvasSeenFromFarIsItsAverageWithNoWeave(t *testing.T) {
	canvas := aCanvas()
	point := maths.V(0, 2.5, 3)
	near, far := canvas.Within(point, 0), canvas.Within(point, 1)
	flat := canvas
	flat.Depth = 0
	if far != flat.At(point) || near == far {
		t.Errorf("near %v, far %v, flat %v", near, far, flat.At(point))
	}
}

func TestARopeIsTwoStrandsWoundRoundEachOther(t *testing.T) {
	twist := Twist{A: maths.RGB(0.1, 0.08, 0.06), B: maths.RGB(0.3, 0.25, 0.2),
		Pitch: 0.06, Lay: 0.02}
	seen := map[maths.Light]bool{}
	for _, point := range clothSamples() {
		if !lit(twist.At(point)) {
			t.Fatalf("the rope at %v is %v", point, twist.At(point))
		}
		seen[twist.At(point)] = true
	}
	if len(seen) < 20 {
		t.Errorf("%d colours in a rope", len(seen))
	}
}

func TestABandIsGroundInkAndEdgeInChevrons(t *testing.T) {
	woven := Woven{Ground: maths.RGB(0.6, 0.15, 0.1),
		Ink: maths.RGB(0.9, 0.7, 0.2), Edge: maths.RGB(0.1, 0.2, 0.5),
		Cell: 0.08, Rise: 0.12}
	seen := map[maths.Light]int{}
	for _, point := range clothSamples() {
		seen[woven.At(point)]++
	}
	for _, colour := range []maths.Light{woven.Ground, woven.Ink, woven.Edge} {
		if seen[colour] == 0 {
			t.Errorf("no %v in the band", colour)
		}
	}
}

func TestFeltShowsItsFibreAndTheOddFleck(t *testing.T) {
	felt := Fibre{Wool: maths.RGB(0.5, 0.45, 0.35),
		Fleck: maths.RGB(0.1, 0.08, 0.05), Scale: 0.01}
	seen := map[maths.Light]bool{}
	for index := 0; index < 600; index++ {
		point := maths.V(float64(index)*0.0037, float64(index)*0.0011,
			float64(index)*0.0023)
		if !lit(felt.At(point)) {
			t.Fatalf("the felt at %v is %v", point, felt.At(point))
		}
		seen[felt.At(point)] = true
	}
	if len(seen) < 100 {
		t.Errorf("%d colours in felt", len(seen))
	}
}

func TestPaintOverGrainIsBetweenThePaintAndTheWood(t *testing.T) {
	grain := Wood{Early: maths.RGB(0.5, 0.35, 0.2), Late: maths.RGB(0.2, 0.1, 0.05),
		Along: maths.YAxis, Ring: 0.012}
	paint := maths.RGB(0.7, 0.1, 0.05)
	stained := Stained{Grain: grain, Paint: paint, Cover: 0.6}
	point := maths.V(0.31, 1, 0.07)
	got := stained.At(point)
	want := grain.At(point).Scale(0.4).Add(paint.Scale(0.6))
	if math.Abs(got.R-want.R) > 1e-9 || math.Abs(got.G-want.G) > 1e-9 {
		t.Errorf("stained %v, want %v", got, want)
	}
}

func TestAnOrnamentRepeatsInRingsOfThreeInksOnWhatIsBeneath(t *testing.T) {
	under := Wood{Early: maths.Grey(0.4), Late: maths.Grey(0.3), Ring: 0.01}
	inks := [3]maths.Light{maths.RGB(0.7, 0.1, 0.1), maths.RGB(0.9, 0.5, 0.1),
		maths.RGB(0.1, 0.2, 0.7)}
	ornament := Ornament{Under: under, Inks: inks, Cell: 0.2, Ring: 0.02,
		Coords: func(p maths.Vec) (float64, float64) { return p.X, p.Y }}
	at := func(across, up float64) maths.Light {
		return ornament.At(maths.V(across, up, 0))
	}
	if at(0.1, 0.1) != inks[0] || at(0.1+0.03, 0.1) != inks[1] ||
		at(0.1+0.05, 0.1) != inks[2] || at(0.1+0.07, 0.1) != inks[0] {
		t.Error("the rings are not red, orange, blue and red again")
	}
	if at(0.3, 0.1) != at(0.1, 0.1) {
		t.Error("the ornament does not repeat across a cell")
	}
	if corner := at(0.001, 0.001); corner != under.At(maths.V(0.001, 0.001, 0)) {
		t.Errorf("the corner of a cell is %v, not what is under it", corner)
	}
}
