package manufacture

import (
	"errors"
	"math"
	"testing"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/maths"
)

func mustLump(t *testing.T, long, middle, short, worn float64,
	number uint64) lump {
	t.Helper()
	made, err := newLump(measure.Metres(long), measure.Metres(middle),
		measure.Metres(short), worn, sow(number))
	if err != nil {
		t.Fatal(err)
	}
	return made
}

// extent is how far a surface reaches along x, y and z, and the lowest
// y it has.
func extent(skin surface) (wide, tall, deep, lowest float64) {
	low := [3]float64{math.Inf(1), math.Inf(1), math.Inf(1)}
	high := [3]float64{math.Inf(-1), math.Inf(-1), math.Inf(-1)}
	for _, face := range skin {
		for _, point := range face {
			at := [3]float64{point.X, point.Y, point.Z}
			for axis := range at {
				low[axis] = math.Min(low[axis], at[axis])
				high[axis] = math.Max(high[axis], at[axis])
			}
		}
	}
	return high[0] - low[0], high[1] - low[1], high[2] - low[2], low[1]
}

// slayer's mesh is the judge again: whatever the seed, however
// worn, however finely cut, a lump is closed, wound outward, with no
// face of no area, and holds something.
func TestALumpIsSoundWhateverItsSeed(t *testing.T) {
	for number := range uint64(40) {
		for _, worn := range []float64{0, 0.35, 1} {
			for _, around := range []int{5, 12, 24} {
				made := mustLump(t, 1.1, 0.8, 0.6, worn, number)
				skin := made.skin(mustCut(t, around))
				flaw := unsound(skin)
				if flaw != "" {
					t.Fatalf("seed %d, worn %v, %d: %s",
						number, worn, around, flaw)
				}
			}
		}
	}
}

// a lump has the three axes it was asked for, however broken: its
// longest along x, its shortest up, and it sits on the ground. (a ball
// cut no finer than an icosahedron has no corner on any axis, so the
// cut here is finer.)
func TestALumpHasItsThreeAxes(t *testing.T) {
	const long, middle, short = 1.1, 0.8, 0.6
	for number := range uint64(40) {
		for _, worn := range []float64{0, 0.5, 1} {
			skin := mustLump(t, long, middle, short, worn,
				number).skin(mustCut(t, 12))
			wide, tall, deep, lowest := extent(skin)
			flaw := ""
			if lowest != 0 {
				flaw = "it is off the ground"
			}
			for _, pair := range [3][2]float64{{wide, long},
				{tall, short}, {deep, middle}} {
				if math.Abs(pair[0]-pair[1]) > 1e-9 {
					flaw = "an axis is not what was asked"
				}
			}
			if flaw != "" {
				t.Fatalf("seed %d, worn %v: %s: %v, %v, %v, "+
					"lowest %v", number, worn, flaw, wide,
					tall, deep, lowest)
			}
		}
	}
}

// a stone worn all the way is its three axes' ellipsoid, exactly: no
// roughness, no squaring, no cleaving, no flat. every corner is on
// the ellipsoid standing on the ground.
func TestALumpWornSmoothIsItsEllipsoid(t *testing.T) {
	const long, middle, short = 0.3, 0.2, 0.12
	skin := mustLump(t, long, middle, short, 1, 5).skin(mustCut(t, 24))
	for index, face := range skin {
		for _, point := range face {
			across := point.X / (long / 2)
			up := (point.Y - short/2) / (short / 2)
			along := point.Z / (middle / 2)
			on := across*across + up*up + along*along
			if math.Abs(on-1) > 1e-9 {
				t.Fatalf("face %d has a corner %v off the "+
					"ellipsoid: %v", index, point, on)
			}
		}
	}
}

// and so its volume is the ellipsoid's, four thirds pi a b c, from
// below, nearer as the cut gets finer.
func TestAWornLumpsVolumeIsTheEllipsoids(t *testing.T) {
	const long, middle, short = 0.3, 0.2, 0.12
	whole := 4.0 / 3 * math.Pi * (long / 2) * (middle / 2) * (short / 2)
	made := mustLump(t, long, middle, short, 1, 5)
	under := math.Inf(1)
	for _, around := range []int{5, 10, 20, 40} {
		got := made.skin(mustCut(t, around)).volume()
		if got >= whole || whole-got >= under {
			t.Errorf("%d round: it holds %v of an ellipsoid's "+
				"%v, no nearer than the coarser cut", around,
				got, whole)
		}
		under = whole - got
	}
	if under > whole/100 {
		t.Errorf("the finest cut is %v short of %v", under, whole)
	}
}

// a broken stone stands on a flat: many of its corners are on the
// ground, where a worn one touches it at one.
func TestABrokenLumpStandsOnAFlat(t *testing.T) {
	fine := mustCut(t, 12)
	onGround := func(skin surface) int {
		grounded := map[[2]float64]bool{}
		for _, face := range skin {
			for _, point := range face {
				where := [2]float64{point.X, point.Z}
				if point.Y == 0 {
					grounded[where] = true
				}
			}
		}
		return len(grounded)
	}
	for number := range uint64(20) {
		broken := mustLump(t, 1, 0.8, 0.6, 0, number).skin(fine)
		smooth := mustLump(t, 1, 0.8, 0.6, 1, number).skin(fine)
		if on := onGround(broken); on < 9 {
			t.Errorf("seed %d: broken, %d corners on the ground",
				number, on)
		}
		if on := onGround(smooth); on != 1 {
			t.Errorf("seed %d: worn smooth, %d on the ground",
				number, on)
		}
	}
}

func TestTheSameSeedIsTheSameLumpAndAnotherIsNot(t *testing.T) {
	fine := mustCut(t, 12)
	one := mustLump(t, 1, 0.8, 0.6, 0.2, 9).skin(fine)
	same := mustLump(t, 1, 0.8, 0.6, 0.2, 9).skin(fine)
	other := mustLump(t, 1, 0.8, 0.6, 0.2, 10).skin(fine)
	differ := 0
	for index := range one {
		if one[index] != same[index] {
			t.Fatalf("face %d changed for the same seed", index)
		}
		if one[index] != other[index] {
			differ++
		}
	}
	if differ < len(one)/2 {
		t.Errorf("%d of %d faces differ for another seed", differ,
			len(one))
	}
}

// a ball is cut as finely as it takes for its middle to have as many
// sides as the cut asks: an icosahedron has five round its middle, and
// each cutting doubles them and makes four faces of every one.
func TestALumpIsCutAsFinelyAsAsked(t *testing.T) {
	faces := map[int]int{3: 20, 5: 20, 6: 80, 10: 80, 11: 320, 20: 320,
		21: 1280, 40: 1280, 41: 5120, 64: 5120}
	made := mustLump(t, 1, 1, 1, 1, 1)
	for around, want := range faces {
		if got := len(made.skin(mustCut(t, around))); got != want {
			t.Errorf("%d round: %d faces, not %d", around, got,
				want)
		}
	}
}

func TestALumpThatCannotBeIsRefused(t *testing.T) {
	nan := math.NaN()
	cases := map[string][4]float64{
		"no long axis":            {0, 0, 0, 0.5},
		"an axis below nought":    {1, 0.5, -0.2, 0.5},
		"an endless axis":         {math.Inf(1), 1, 1, 0.5},
		"an axis of no number":    {1, nan, 0.5, 0.5},
		"a middle past the long":  {1, 1.2, 0.5, 0.5},
		"a short past the middle": {1, 0.5, 0.6, 0.5},
		"worn past one":           {1, 0.8, 0.6, 1.1},
		"worn below nought":       {1, 0.8, 0.6, -0.1},
		"worn by no number":       {1, 0.8, 0.6, nan},
	}
	for name, with := range cases {
		_, err := newLump(measure.Metres(with[0]),
			measure.Metres(with[1]), measure.Metres(with[2]),
			with[3], sow(1))
		if !errors.Is(err, errLump) {
			t.Errorf("a lump with %s: %v, not %v", name, err,
				errLump)
		}
	}
	_, err := newLump(measure.Metres(1), measure.Metres(1),
		measure.Metres(1), 0, seed{})
	if !errors.Is(err, errLump) {
		t.Errorf("a lump from no seed: %v, not %v", err, errLump)
	}
	if _, err := newLump(measure.Metres(1), measure.Metres(1),
		measure.Metres(1), 0, sow(0)); err != nil {
		t.Errorf("a lump of three equal axes was refused: %v", err)
	}
}

// a stone not worn at all is cleaved by as many planes as there are,
// a stone half worn by half of them, a stone worn smooth by none; each
// plane faces a way of its own, a unit long, and lies between the
// nearest and the furthest a plane may, by the seed.
func TestAStoneIsCleavedByFewerPlanesTheMoreItIsWorn(t *testing.T) {
	counts := map[float64]int{0: mostCleaves, 0.5: mostCleaves / 2, 1: 0}
	for worn, want := range counts {
		planes := mustLump(t, 1, 1, 1, worn, 3).cleaves()
		if len(planes) != want {
			t.Fatalf("worn %v: %d planes, not %d", worn,
				len(planes), want)
		}
		ways := map[maths.Vec]bool{}
		for index, cleft := range planes {
			if math.Abs(cleft.facing.Len()-1) > close {
				t.Errorf("worn %v: plane %d faces %v, no unit",
					worn, index, cleft.facing)
			}
			furthest := nearest + cleaveRange
			if cleft.out < nearest || cleft.out > furthest {
				t.Errorf("worn %v: plane %d lies %v out", worn,
					index, cleft.out)
			}
			ways[cleft.facing] = true
		}
		if len(ways) != want {
			t.Errorf("worn %v: %d planes face %d ways", worn, want,
				len(ways))
		}
	}
	one := mustLump(t, 1, 1, 1, 0, 3).cleaves()
	other := mustLump(t, 1, 1, 1, 0, 4).cleaves()
	if one[0] == other[0] {
		t.Error("two seeds cleave the same way")
	}
}

// what a cleaving does: a point past the plane is brought back onto
// it, and a point behind it is left where it was. the flat a stone
// sits on can lift a point afterwards, and those are not counted.
func TestACleavingBringsWhatIsPastItBackToThePlane(t *testing.T) {
	made := mustLump(t, 1, 1, 1, 0, 7)
	cleft := made.cleaves()[0]
	floor := made.flat()
	moved := 0
	for _, face := range ball(mustCut(t, 12)) {
		for _, on := range face {
			whole := made.shaped(on, nil)
			cut := made.shaped(on, []plane{cleft})
			if whole.Y == floor || cut.Y == floor {
				continue
			}
			past := whole.Dot(cleft.facing) - cleft.out
			if past <= 0 && cut != whole {
				t.Fatalf("%v is behind the plane and moved", on)
			}
			if past <= 0 {
				continue
			}
			moved++
			if off := cut.Dot(cleft.facing) - cleft.out; math.Abs(
				off) > 1e-12 {
				t.Fatalf("%v was past the plane and is now %v "+
					"off it", on, off)
			}
		}
	}
	if moved == 0 {
		t.Error("the plane cleaved nothing")
	}
}

// the noise that roughens a stone is the stone's own: two seeds, with
// no cleaving in the way, are rough in different places.
func TestAStonesRoughnessIsItsSeeds(t *testing.T) {
	one, other := mustLump(t, 1, 1, 1, 0, 7), mustLump(t, 1, 1, 1, 0, 8)
	same := 0
	round := ball(mustCut(t, 12))
	for _, face := range round {
		if one.shaped(face[0], nil) == other.shaped(face[0], nil) {
			same++
		}
	}
	// the corners already on the flat underneath are alike for both
	if same > len(round)/2 {
		t.Errorf("%d of %d corners are alike for two seeds", same,
			len(round))
	}
}

// a stone not worn at all is cut flat a share of its radius below its
// middle; a stone worn smooth reaches the whole way down.
func TestAStonesFlatRisesAsItIsLessWorn(t *testing.T) {
	fresh, smooth := mustLump(t, 1, 1, 1, 0, 1), mustLump(t, 1, 1, 1, 1, 1)
	if got := fresh.flat(); math.Abs(got+sits) > close {
		t.Errorf("a fresh stone's flat is at %v, not %v below", got,
			sits)
	}
	if got := smooth.flat(); got != -1 {
		t.Errorf("a smooth stone's flat is at %v, not the bottom", got)
	}
}

// the ball a stone starts as is a regular icosahedron on the unit
// ball: twenty faces, every corner one from the middle, and every
// edge the length an icosahedron's is, four over the root of ten and
// two root five.
func TestTheBallStartsAsARegularIcosahedron(t *testing.T) {
	start := icosahedron()
	if len(start) != 20 {
		t.Fatalf("%d faces", len(start))
	}
	edge := 4 / math.Sqrt(10+2*math.Sqrt(5))
	for index, face := range start {
		for corner, at := range face {
			out := at.Len()
			long := face[(corner+1)%3].Sub(at).Len()
			if math.Abs(out-1) > close {
				t.Errorf("face %d: a corner is %v out", index,
					out)
			}
			if math.Abs(long-edge) > close {
				t.Errorf("face %d: an edge is %v long, not %v",
					index, long, edge)
			}
		}
	}
	if flaw := unsound(start); flaw != "" {
		t.Error(flaw)
	}
}

// noise only takes stone away, and no more than the roughest share of
// its radius: before any cleaving, every point of a fresh stone lies
// inside where the squared ball put it, and outside that share of it.
func TestNoiseOnlyTakesStoneAway(t *testing.T) {
	made := mustLump(t, 1, 1, 1, 0, 11)
	for _, face := range ball(mustCut(t, 12)) {
		for _, on := range face {
			got := made.shaped(on, nil)
			if got.Y == made.flat() {
				continue
			}
			power := squarest
			whole := maths.V(squared(on.X, power),
				squared(on.Y, power), squared(on.Z, power))
			kept := got.Len() / whole.Len()
			if kept > 1+close || kept < 1-roughest-close {
				t.Fatalf("%v keeps %v of its radius", on, kept)
			}
		}
	}
}
