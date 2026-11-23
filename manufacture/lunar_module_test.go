package manufacture

import (
	"math"
	"strings"
	"testing"

	"github.com/janearc/slayer/maths"
)

// The lunar module's sizes, as lunar-module.yaml gives them and where it
// says they were read.
const (
	lunarTall  = 7.04 // read: NASA, Apollo 204 spacecraft page, 23 ft 1 in, legs extended
	lunarSpan  = 9.4  // read: NSSDCA, the ends of the footpads on opposite legs
	lunarNear  = 1e-9 // picked: sums of a few lengths, to rounding
	lunarClear = 0.9  // picked: the descent stage stands more than this off the ground; the file's sizes leave 0.977 m
)

// The lunar module stands on the ground, as tall as it is overall and as
// wide as its pads' ends are apart, with its descent stage clear of the
// ground under it and its ascent stage on top of that.
func TestTheLunarModuleStandsOnItsPads(t *testing.T) {
	made := mustMake(t, "apollo-lunar-module", 5)
	box := made.Bounds()
	if math.Abs(box.Min.Y) > lunarNear || math.Abs(box.Max.Y-lunarTall) > lunarNear ||
		math.Abs(box.Max.X-lunarSpan/2) > lunarNear {
		t.Errorf("the lunar module's box is %+v, want nought to %v up and %v out",
			box, lunarTall, lunarSpan/2)
	}
	parts, err := made.Things(maths.Local())
	if err != nil {
		t.Fatal(err)
	}
	low := map[string]float64{}
	high := map[string]float64{}
	for _, each := range parts {
		name := each.Material.Name
		inside := each.Shape.Bounds()
		at := each.At(0).Origin.Y
		if _, seen := low[name]; !seen {
			low[name], high[name] = math.Inf(1), math.Inf(-1)
		}
		low[name] = math.Min(low[name], at+inside.Min.Y)
		high[name] = math.Max(high[name], at+inside.Max.Y)
	}
	if math.Abs(low["landing gear"]) > lunarNear {
		t.Errorf("the gear's lowest is %v, want the ground", low["landing gear"])
	}
	if !(low["descent stage"] > lunarClear) {
		t.Errorf("the descent stage's underside is %v up: not clear of the ground",
			low["descent stage"])
	}
	if math.Abs(high["descent stage"]-low["ascent stage"]) > lunarNear {
		t.Errorf("the descent stage ends at %v and the ascent stage begins at %v",
			high["descent stage"], low["ascent stage"])
	}
	if math.Abs(high["ascent stage"]-lunarTall) > lunarNear {
		t.Errorf("the ascent stage's top is %v, want %v", high["ascent stage"], lunarTall)
	}
}

// The descent stage's blanket is Kapton over aluminium: a metal under a
// film. The anti-test is the ascent stage, bare aluminium with no film.
func TestTheLunarModulesBlanketIsKaptonOverAluminium(t *testing.T) {
	dressed, err := lunarRead.looks()
	if err != nil {
		t.Fatal(err)
	}
	blanket, cabin := dressed[lunarBlanket], dressed[lunarCabin]
	if blanket.Metal == nil || blanket.Film == nil {
		t.Errorf("the blanket's metal is %v and its film %v", blanket.Metal, blanket.Film)
	}
	if cabin.Metal == nil || cabin.Film != nil {
		t.Errorf("the cabin's metal is %v and its film %v: want bare metal",
			cabin.Metal, cabin.Film)
	}
	// through the film the aluminium mirrors more red than blue: gold
	if mirrors := blanket.Reflects(1); !(mirrors.R > mirrors.B) {
		t.Errorf("the blanket mirrors %+v face on: not gold", mirrors)
	}
}

// Sizes that cannot be built are refused when the file is read, by the
// name of what is wrong.
func TestALunarModuleThatCannotBeBuiltIsRefused(t *testing.T) {
	whole := string(lunarModuleRaw)
	refused := map[string]struct{ was, is, names string }{
		"no height":              {"overall: 7.04", "overall: 0", "overall"},
		"stages taller than it":  {"overall: 7.04", "overall: 5", "no room"},
		"pads inside the stage":  {"span: 9.4", "span: 4", "pads"},
		"a strut of no width":    {"strut: 0.07", "strut: 0", "strut"},
		"a blanket below nought": {"blanket: 25.4e-6", "blanket: -1", "blanket"},
		"read nowhere":           {"read: >-", "unread: >-", "unread"},
	}
	for name, each := range refused {
		if !strings.Contains(whole, each.was) {
			t.Fatalf("%s: the file has no %q to change", name, each.was)
		}
		edited := strings.Replace(whole, each.was, each.is, 1)
		_, err := readLunar([]byte(edited))
		if err == nil {
			t.Errorf("%s: read without refusal", name)
			continue
		}
		if !strings.Contains(err.Error(), each.names) {
			t.Errorf("%s: refused as %q, which does not name %q", name, err, each.names)
		}
	}
}
