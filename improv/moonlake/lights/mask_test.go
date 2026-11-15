package lights

import (
	"reflect"
	"testing"
)

func TestMaskThresholdsAndBrightest(t *testing.T) {
	image := Image{Width: 4, Height: 1, Light: []float64{10, 0, 0, 0.2, 0, 0, 1, 1, 1, 0, 5, 0}}
	before := append([]float64(nil), image.Light...)
	open, err := SkyOf(image)
	if err != nil || !reflect.DeepEqual(open, []bool{true, false, true, true}) {
		t.Fatalf("sky=%v err=%v", open, err)
	}
	covered, err := CoveredBy(image)
	if err != nil || !reflect.DeepEqual(covered, []bool{true, false, false, true}) {
		t.Fatalf("covered=%v err=%v", covered, err)
	}
	bright, err := Brightest(image)
	if err != nil || bright != [3]float64{5, 2.5, 0} {
		t.Fatalf("brightest=%v err=%v", bright, err)
	}
	if !reflect.DeepEqual(image.Light, before) {
		t.Fatal("analysis mutated input")
	}
	black := Image{Width: 1, Height: 1, Light: make([]float64, 3)}
	open, _ = SkyOf(black)
	covered, _ = CoveredBy(black)
	bright, _ = Brightest(black)
	if open[0] || covered[0] || bright != [3]float64{} {
		t.Fatal("black mask classified as lit")
	}
}

func TestRimUsesLowestSkyWithinThirtyColumns(t *testing.T) {
	open := make([]bool, 63*4)
	open[0] = true
	open[2*63+31] = true
	before := append([]bool(nil), open...)
	rim, err := RimOf(open, 63, 4)
	if err != nil {
		t.Fatal(err)
	}
	if rim(-10) != 1 || rim(0) != 1 || rim(1) != 3 || rim(61) != 3 || rim(62) != 0 || rim(90) != 0 {
		t.Fatal("rim smoothing radius, bottom row, or clamping changed")
	}
	if !reflect.DeepEqual(open, before) {
		t.Fatal("rim mutated mask")
	}
}

func TestNearPreservesSeedRowsAndFourConnectedRimGuard(t *testing.T) {
	const width, height = 3, 44
	open := make([]bool, width*height)
	for i := range open {
		open[i] = true
	}
	for _, i := range []int{39 * width, 40 * width, 41 * width, 42 * width, 40*width + 2} {
		open[i] = false
	}
	near, err := NearOf(open, func(int) int { return 44 }, width, height)
	if err != nil {
		t.Fatal(err)
	}
	if !near[39*width] || !near[40*width] || !near[41*width] || near[42*width] || near[40*width+2] {
		t.Fatalf("unexpected flood fill: %v", near[39*width:])
	}
	tiny, err := NearOf([]bool{false}, func(int) int { return 0 }, 1, 1)
	if err != nil || !tiny[0] {
		t.Fatal("legacy seed below rim was discarded")
	}
}

func TestAnalysisRejectsInvalidShapes(t *testing.T) {
	for _, image := range []Image{{}, {Width: -1, Height: 1}, {Width: 1, Height: 1, Light: []float64{1}}, {Width: 1, Height: 1, Light: make([]float64, 6)}} {
		if _, err := SkyOf(image); err == nil {
			t.Fatal("invalid sky image accepted")
		}
		if _, err := CoveredBy(image); err == nil {
			t.Fatal("invalid coverage image accepted")
		}
		if _, err := Brightest(image); err == nil {
			t.Fatal("invalid brightest image accepted")
		}
	}
	if _, err := RimOf(nil, 0, 0); err == nil {
		t.Fatal("empty rim accepted")
	}
	if _, err := RimOf([]bool{true}, 2, 1); err == nil {
		t.Fatal("short rim accepted")
	}
	if _, err := NearOf([]bool{true}, nil, 1, 1); err == nil {
		t.Fatal("nil rim accepted")
	}
	if _, err := NearOf(nil, func(int) int { return 1 }, 1, 1); err == nil {
		t.Fatal("short near mask accepted")
	}
}
