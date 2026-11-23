package cities

import (
	"math"
	"testing"
)

// bivouacSeeds is how many seeds the bivouac tests draw, enough that a
// stove is among them and is missing from some.
const bivouacSeeds = 40

// drawnBivouac is a bivouac drawn for a level and a seed, or the test's
// end.
func drawnBivouac(t *testing.T, level Detail, seed Seed) []Rendered {
	t.Helper()
	design, err := BivouacAt(level, seed)
	if err != nil {
		t.Fatal(err)
	}
	drawn, err := design.Rendered(level)
	if err != nil {
		t.Fatal(err)
	}
	return drawn
}

func TestADemoBivouacIsDrawnInMorePartsThanAQuickOne(t *testing.T) {
	for seed := range Seed(8) {
		quick := len(drawnBivouac(t, "quick", seed))
		demo := len(drawnBivouac(t, "demo", seed))
		if demo <= quick {
			t.Errorf("seed %d: %d parts at demo and %d at quick", seed, demo,
				quick)
		}
	}
}

func TestEveryPartOfABivouacIsInsideItsEnvelope(t *testing.T) {
	out := mustChoose(t, "bivouac-reach")
	top := mustChoose(t, "bivouac-reach-top")
	sunk := mustChoose(t, "stake-thickness")
	for _, level := range []Detail{"hull", "quick", "demo"} {
		for seed := range Seed(8) {
			for index, one := range drawnBivouac(t, level, seed) {
				low, high := reach(one)
				reached, _ := outermost(one)
				if low.Y < -sunk || high.Y > top+tolerance ||
					reached > out+tolerance {
					t.Fatalf("%s seed %d part %d (%s) spans %v to %v up and "+
						"reaches %v out", level, seed, index, one.Mesh.Name,
						low.Y, high.Y, reached)
				}
			}
		}
	}
}

func TestTheSameSeedMakesTheSameBivouac(t *testing.T) {
	for seed := range Seed(8) {
		first := digest(drawnBivouac(t, "demo", seed))
		if second := digest(drawnBivouac(t, "demo", seed)); first != second {
			t.Errorf("seed %d: two draws differ", seed)
		}
	}
}

func TestEveryPartOfABivouacIsAClosedSolidWoundOutward(t *testing.T) {
	for _, level := range []Detail{"hull", "quick", "demo"} {
		for index, one := range drawnBivouac(t, level, 1) {
			if !closed(one.Mesh) || volume(one.Mesh) <= 0 {
				t.Fatalf("%s part %d (%s) is not a closed solid wound "+
					"outward", level, index, one.Mesh.Name)
			}
		}
	}
}

func TestABivouacIsASackWithAHoopGuyLinesStakesAPackAndBoots(t *testing.T) {
	drawn := drawnBivouac(t, "quick", 1)
	for part, want := range map[string]int{"foot": 1, "shoulder": 1,
		"hood": 1, "hoop": int(mustChoose(t, "bivy-facets")), "guy": 4,
		"stake": 4, "pack": 1, "sole": 2, "upper": 2} {
		if got := len(named(drawn, part)); got != want {
			t.Errorf("%d %s parts, want %d", got, part, want)
		}
	}
}

func TestTheSackIsTheLengthAndWidthDataGivesAndTheHoodIsTheHighest(
	t *testing.T) {
	drawn := drawnBivouac(t, "quick", 1)
	var low, high [3]float64
	low, high = [3]float64{math.Inf(1), math.Inf(1), math.Inf(1)},
		[3]float64{math.Inf(-1), math.Inf(-1), math.Inf(-1)}
	for _, part := range []string{"foot", "shoulder", "hood"} {
		for _, one := range named(drawn, part) {
			from, to := reach(one)
			low[0], high[0] = math.Min(low[0], from.X), math.Max(high[0], to.X)
			low[1], high[1] = math.Min(low[1], from.Y), math.Max(high[1], to.Y)
			low[2], high[2] = math.Min(low[2], from.Z), math.Max(high[2], to.Z)
		}
	}
	if math.Abs(high[2]-low[2]-mustChoose(t, "bivy-length")) > tolerance ||
		math.Abs(high[0]-low[0]-mustChoose(t, "bivy-width")) > tolerance ||
		math.Abs(high[1]-mustChoose(t, "bivy-hood-height")) > tolerance {
		t.Errorf("the sack spans %v to %v", low, high)
	}
}

func TestSomeBivouacsHaveAStoveAndItsFlameIsALamp(t *testing.T) {
	with := 0
	for seed := range Seed(bivouacSeeds) {
		drawn := drawnBivouac(t, "quick", seed)
		if len(named(drawn, "pot")) == 0 {
			continue
		}
		with++
		burners := named(drawn, "burner")
		if len(burners) != 1 || burners[0].Lamp == nil ||
			burners[0].Lamp.Heat != mustChoose(t, "flame-heat") {
			t.Fatalf("seed %d: the stove's burner is %+v", seed, burners)
		}
	}
	if with < 3 || with > bivouacSeeds-3 {
		t.Errorf("%d of %d bivouacs have a stove, at a share of %v", with,
			bivouacSeeds, mustChoose(t, "stove-share"))
	}
}

func TestTheDemoBivouacHasTapedSeamsAndAPackLidAndRipstop(t *testing.T) {
	demo, quick := drawnBivouac(t, "demo", 1), drawnBivouac(t, "quick", 1)
	if len(named(demo, "seam")) == 0 || len(named(demo, "lid")) != 1 ||
		len(named(demo, "strap")) != 2 {
		t.Error("the demo bivouac lacks its seams, lid or straps")
	}
	if len(named(quick, "seam"))+len(named(quick, "lid")) != 0 {
		t.Error("the quick bivouac has demo detail")
	}
	if named(demo, "hood")[0].Material.Texture == nil ||
		named(quick, "hood")[0].Material.Texture != nil {
		t.Error("the nylon is textured at quick, or plain at demo")
	}
	if gloss := named(quick, "hood")[0].Material.Gloss; gloss <= 0 ||
		gloss >= 0.5 {
		t.Errorf("the nylon shines %v, want a slight sheen", gloss)
	}
}

func TestABivouacSleepsOneAndAPitchHoldsItWithRoomAround(t *testing.T) {
	facts, err := BivouacFacts()
	if err != nil {
		t.Fatal(err)
	}
	footprint := mustChoose(t, "bivy-length") * mustChoose(t, "bivy-width")
	if facts.Sleeps != 1 || facts.FloorArea != 1.6 ||
		facts.FloorArea > footprint || facts.PitchArea <= footprint {
		t.Errorf("facts %+v against a footprint of %v", facts, footprint)
	}
	item := anItem(t, "bivouac")
	if !item.Pitched || !item.Faced ||
		item.Pitch.Width*item.Pitch.Depth != facts.PitchArea {
		t.Errorf("the bivouac's item is %+v", item)
	}
}
