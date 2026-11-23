package arid

import (
	"math"
	"strings"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/oceantex"
)

// surfSea is a 9 m/s sea, fully developed and deep, running toward the
// east-north-east, onto a beach that rises eastward.
var surfSea = oceantex.Sea{Wind: 9, Fetch: 500000, Depth: 0, Toward: 0.4}

// surfed is the surf sea, at time nought, seed 7, over a ground whose
// first sample is at the sea's middle and whose heights are counted from
// its mean.
func surfed(t *testing.T, ground Ground) (*Sea, *Sea, Surfline) {
	t.Helper()
	open, err := NewSea(surfSea, 0, 7)
	if err != nil {
		t.Fatal(err)
	}
	shored, counted, err := open.Surf(surfSea, ground, 0, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	return open, shored, counted
}

// beachHeight is a beach rising 0.08 m a metre eastward, sampled every
// 4 m: 12 m deep at the west edge, the waterline at 150 m, land beyond.
func beachHeight(column, _ int) float64 { return -12 + 0.32*float64(column) }

// beach is the beach ground, 60 samples east and 20 south.
func beach() Ground { return shaped(60, 20, 4, beachHeight) }

func TestSurfTroughsStayOverTheBed(t *testing.T) {
	_, shored, _ := surfed(t, beach())
	for index := range 2000 {
		east := math.Mod(float64(index)*0.731, 236)
		south := math.Mod(float64(index)*3.17, 76)
		depth := -shored.coast.ground(shored.coast.locate(east, south))
		height := shored.height(east, south)
		if math.Abs(height) > math.Max(depth, 0)+1e-12 {
			t.Fatalf("at %g, %g the surface is %g from its mean over "+
				"%g m of water", east, south, height, depth)
		}
	}
}

func TestSurfRisesNoFasterThanItsBound(t *testing.T) {
	_, shored, _ := surfed(t, beach())
	const apart = 0.3
	for index := range 4000 {
		turn := float64(index) * 2.399
		east := -40 + math.Mod(float64(index)*7.31, 320)
		south := -20 + math.Mod(float64(index)*3.17, 120)
		there := shored.height(east+apart*math.Cos(turn),
			south+apart*math.Sin(turn))
		change := math.Abs(there - shored.height(east, south))
		if change > shored.steepest*apart*(1+1e-9) {
			t.Fatalf("at %g, %g the surface rose %g in %g m, over the "+
				"bound %g a metre", east, south, change, apart,
				shored.steepest)
		}
	}
}

// smoothSea is one band of a 16 m swell running east, 0.3 m high, sampled
// so finely that its bilinear heights lean as its slopes say.
func smoothSea(t *testing.T) *Sea {
	t.Helper()
	const count, side, length, reach = 512, 64.0, 16.0, 0.15
	field := &oceantex.Field{N: count, Extent: side,
		Height: make([]float64, count*count),
		SlopeX: make([]float64, count*count),
		SlopeY: make([]float64, count*count)}
	for at := range field.Height {
		east := float64(at%count) * side / count
		field.Height[at] = reach * math.Sin(2*math.Pi*east/length)
		field.SlopeX[at] = reach * 2 * math.Pi / length *
			math.Cos(2*math.Pi*east/length)
	}
	sea, err := seaOf([]oceantex.WaveCascade{{Field: field, Reach: reach}})
	if err != nil {
		t.Fatal(err)
	}
	return sea
}

func TestSurfNormalLeansWithTheSurface(t *testing.T) {
	shored, _, err := smoothSea(t).Surf(surfSea, beach(), 0, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	const reach = 1e-5
	for index := range 40 {
		// inside one triangle's rows, away from its edges
		east := 60.5 + float64(index)*2.3
		south := 41.3
		normal := shored.Normal(maths.V(east, 0, south))
		riseEast := (shored.height(east+reach, south) -
			shored.height(east-reach, south)) / (2 * reach)
		riseSouth := (shored.height(east, south+reach) -
			shored.height(east, south-reach)) / (2 * reach)
		lean := maths.V(-riseEast, 1, -riseSouth).Unit()
		if normal.Sub(lean).Len() > 0.01 {
			t.Fatalf("at %g m east the normal %v, the surface's lean %v",
				east, normal, lean)
		}
	}
}

// surfNormalAgrees is how far, as the length of the difference of two unit
// normals, the surf's in deep water may be from the open sea's: the surf
// reads the bands' slopes by their bilinear grids and the open sea by
// cubic convolution, which differ by the grids' sampling: 0.0095 at these
// places as it stands, 0.0216 with the stretch left out of the carrying;
// picked between.
const surfNormalAgrees = 0.015

func TestSurfNormalIsTheOpenSeasInDeepWater(t *testing.T) {
	ground := shaped(20, 20, 8, evenHeight(500))
	open, shored, _ := surfed(t, ground)
	var most float64
	for index := range 50 {
		point := maths.V(3+float64(index)*2.9, 0, 5+float64(index)*1.7)
		most = math.Max(most,
			shored.Normal(point).Sub(open.Normal(point)).Len())
	}
	if most > surfNormalAgrees {
		t.Errorf("deep water's surf normal is %g from the open sea's, "+
			"over %g", most, surfNormalAgrees)
	}
}

func TestSurfHitFindsWhatSteppingFinds(t *testing.T) {
	_, shored, _ := surfed(t, beach())
	for index := range 12 {
		angle := float64(index)
		ray := maths.Ray{At: maths.V(20, 1.5, 40),
			Dir: maths.V(math.Cos(angle), -0.02, math.Sin(angle))}
		wanted, found := brute(shored, ray, 300, 0.01)
		along, hit := shored.Hit(ray, 0, 300)
		if hit != found {
			t.Fatalf("ray %d: hit %t, stepping found %t", index, hit, found)
		}
		if hit && along > wanted+0.01 {
			t.Fatalf("ray %d: met at %g, after stepping did at %g", index,
				along, wanted)
		}
	}
}

func TestSurfIsTheOpenSeaInDeepWater(t *testing.T) {
	ground := shaped(20, 20, 8, evenHeight(500))
	open, shored, _ := surfed(t, ground)
	for index := range 50 {
		east, south := 3+float64(index)*2.9, 5+float64(index)*1.7
		if math.Abs(shored.height(east, south)-open.height(east, south)) >
			1e-6 {
			t.Fatalf("at %g, %g deep water's surf is not the open sea",
				east, south)
		}
	}
}

func TestSurfBreaksAndCountsWhatItDid(t *testing.T) {
	_, _, counted := surfed(t, beach())
	if counted.Calm || counted.Capped == 0 || !(counted.CappedFrom > 0) {
		t.Fatalf("no waves were cut by the depth: %+v", counted)
	}
	if counted.Spilling+counted.Plunging+counted.Surging != 20 {
		t.Fatalf("the waterline's 20 samples were not all counted: %+v",
			counted)
	}
	if !(counted.RunupMost >= counted.RunupLeast && counted.RunupLeast > 0) {
		t.Fatalf("the run-up is %g to %g", counted.RunupLeast,
			counted.RunupMost)
	}
	if len(counted.Rounds) == 0 || !(counted.Whitecaps > 0) {
		t.Fatalf("no bands bent or no whitecaps: %+v", counted)
	}
}

func TestSurfFoamsMoreWhereTheWavesBreak(t *testing.T) {
	_, shored, _ := surfed(t, beach())
	share := func(from, to float64) float64 {
		var sum float64
		for index := range 2000 {
			east := from + math.Mod(float64(index)*0.913, to-from)
			south := math.Mod(float64(index)*0.377, 76)
			cover, _ := shored.Foam(maths.V(east, 0, south))
			sum += cover
		}
		return sum / 2000
	}
	if deep, surf := share(0, 40), share(120, 148); !(surf > deep) {
		t.Fatalf("foam covers %g of the surf and %g of deep water", surf,
			deep)
	}
	if cover, _ := shored.Foam(maths.V(220, 0, 30)); cover != 0 {
		t.Fatalf("foam covers %g of dry land", cover)
	}
}

func TestSurfIsAllBrokenAtTheWaterline(t *testing.T) {
	_, shored, _ := surfed(t, beach())
	// the waterline is between 148 m (0.16 m deep) and 152 m (0.16 m up)
	if cover, _ := shored.Foam(maths.V(149.9, 0, 30)); !(cover > 0) {
		t.Fatalf("the water at the waterline is %g foam", cover)
	}
	if breaking := shored.surf.breaking[10*60+38]; breaking != 1 {
		t.Fatalf("land at the waterline carries %g of the waves broken",
			breaking)
	}
}

func TestSurfFoamReflectsAsFoam(t *testing.T) {
	_, shored, _ := surfed(t, beach())
	for index := range 400 {
		east := math.Mod(float64(index)*0.41, 150)
		cover, light := shored.Foam(maths.V(east, 0, 13))
		if cover > 0 && (light.R < whitecapReflectance ||
			light.R > surfFoamReflectance || light.R != light.B) {
			t.Fatalf("foam at %g m east reflects %v", east, light)
		}
	}
}

func TestSurfWetsTheBeachUpToTheRunup(t *testing.T) {
	_, shored, counted := surfed(t, beach())
	if counted.Swashed == 0 {
		t.Fatal("the swash wet no land")
	}
	// the first land sample is 0.16 m up, at 152 m east
	if wet := shored.Wet(maths.V(152, 0, 30)); !(wet > 0 && wet <= 1) {
		t.Errorf("the beach just over the sea is %g wet", wet)
	}
	if wet := shored.Wet(maths.V(230, 0, 30)); wet != 0 {
		t.Errorf("land 6 m up is %g wet", wet)
	}
	if wet := shored.Wet(maths.V(20, 0, 30)); wet != 1 {
		t.Errorf("the sea floor is %g wet", wet)
	}
}

func TestSurfOfACalmSeaIsTheShoredSea(t *testing.T) {
	open, err := NewSea(surfSea, 0, 7)
	if err != nil {
		t.Fatal(err)
	}
	shored, counted, err := open.Surf(oceantex.Sea{}, beach(), 0, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	if !counted.Calm || shored.surf != nil {
		t.Fatal("a calm sea was given a surf")
	}
	if cover, _ := shored.Foam(maths.V(140, 0, 3)); cover != 0 {
		t.Fatal("a calm sea has foam")
	}
	if !strings.Contains(counted.String(), "calm") {
		t.Fatalf("%q does not say the sea is calm", counted.String())
	}
}

func TestSurfRefusesWhatItCannotRead(t *testing.T) {
	open, err := NewSea(surfSea, 0, 7)
	if err != nil {
		t.Fatal(err)
	}
	for name, made := range map[string]oceantex.Sea{
		"no wind":       {Wind: math.NaN(), Fetch: 1},
		"endless fetch": {Wind: 5, Fetch: math.Inf(1)},
		"no way":        {Wind: 5, Fetch: 1, Toward: math.NaN()},
		"a depth below": {Wind: 5, Fetch: 1, Depth: -1},
	} {
		if _, _, err := open.Surf(made, beach(), 0, 0, 0); err == nil {
			t.Errorf("accepted %s", name)
		}
	}
	if _, _, err := open.Surf(surfSea, nil, 0, 0, 0); err == nil {
		t.Error("accepted no ground")
	}
	var none *Sea
	if _, _, err := none.Surf(surfSea, beach(), 0, 0, 0); err == nil {
		t.Error("no sea met the ground")
	}
}

func TestSurfHasNoFoamOrWetnessWithoutASurf(t *testing.T) {
	open, err := NewSea(surfSea, 0, 7)
	if err != nil {
		t.Fatal(err)
	}
	if cover, _ := open.Foam(maths.V(1, 0, 1)); cover != 0 {
		t.Error("an open sea with no coast has surf foam")
	}
	if open.Wet(maths.V(1, 0, 1)) != 0 {
		t.Error("an open sea with no coast wets ground")
	}
	_, shored, _ := surfed(t, beach())
	nowhere := maths.V(math.NaN(), 0, 1)
	if cover, _ := shored.Foam(nowhere); cover != 0 {
		t.Error("a place that is not a number has foam")
	}
}

func TestSurflineSaysItsCounts(t *testing.T) {
	said := Surfline{Gain: 1.2, Capped: 40, CappedFrom: 2.5, Spilling: 3,
		Plunging: 4, Surging: 5, RunupLeast: 0.4, RunupMost: 1.1,
		Swashed: 7, Whitecaps: 0.01, Rounds: []int{3, 2, 2}}.String()
	for _, part := range []string{"sea over ground", "at most 1.2",
		"at 40 heights from 2.5 m", "spilling at 3", "plunging at 4",
		"surging at 5", "0.4 to 1.1 m", "wets 7", "[3 2 2] rounds"} {
		if !strings.Contains(said, part) {
			t.Errorf("%q does not say %q", said, part)
		}
	}
}
