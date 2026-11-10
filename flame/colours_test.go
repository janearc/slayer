package flame

import (
	"math"
	"testing"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/libtheme-css/spaces/srgb"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/observer"
)

func TestThermalColoursRetainTheirPhysicalEndpointsAndScale(t *testing.T) {
	spec := ColourSpec{Coolest: measure.Kelvin(700), Hottest: measure.Kelvin(1700), Exposure: 2}
	table, err := Colours(spec)
	if err != nil {
		t.Fatal(err)
	}
	for _, heat := range []measure.Temperature{spec.Coolest, spec.Hottest} {
		r, g, b := srgb.Light(observer.Swatch(Glowing(heat).Scale(spec.Exposure)))
		want := maths.RGB(max(0, r), max(0, g), max(0, b))
		got := table.At(heat)
		if !(got.R > 0) || math.Abs(got.R-want.R)+math.Abs(got.G-want.G)+math.Abs(got.B-want.B) > 1e-10*max(1, want.Luminance()) {
			t.Fatal("black-body endpoint changed")
		}
	}
	spec.Exposure *= 2
	doubled, _ := Colours(spec)
	if doubled.At(measure.Kelvin(1200)) != table.At(measure.Kelvin(1200)).Scale(2) {
		t.Fatal("lookup changed exposure scaling")
	}
	if table.At(measure.Kelvin(500)) != table.At(spec.Coolest) || table.At(measure.Kelvin(2000)) != table.At(spec.Hottest) {
		t.Fatal("lookup escaped its temperature range")
	}
}

func TestThermalColoursRejectInvalidRangesAndExposure(t *testing.T) {
	for _, spec := range []ColourSpec{
		{Coolest: measure.Kelvin(900), Hottest: measure.Kelvin(800), Exposure: 1},
		{Coolest: measure.Kelvin(900), Hottest: measure.Kelvin(1800)},
		{Coolest: measure.Kelvin(math.NaN()), Hottest: measure.Kelvin(1800), Exposure: 1},
	} {
		if _, err := Colours(spec); err == nil {
			t.Fatal("invalid thermal table accepted")
		}
	}
}
