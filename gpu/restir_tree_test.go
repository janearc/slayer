package gpu

import (
	"math"
	"math/rand"
	"testing"
)

// scatter is n lamps at fixed places in a box 20 across, 4 high, with
// powers that differ.
func scatter(n int) ([][3]float32, []float32) {
	random := rand.New(rand.NewSource(7))
	places := make([][3]float32, n)
	powers := make([]float32, n)
	for index := range n {
		places[index] = [3]float32{float32(random.Float64()*20 - 10),
			float32(random.Float64() * 4), float32(random.Float64()*20 - 10)}
		powers[index] = 0.1 + float32(random.Float64())
	}
	return places, powers
}

func mustTree(t *testing.T, n int) lightTree {
	t.Helper()
	places, powers := scatter(n)
	tree, err := buildLightTree(places, powers)
	if err != nil {
		t.Fatal(err)
	}
	return tree
}

func TestTreeHoldsEveryLampOnceAndTheirPower(t *testing.T) {
	places, powers := scatter(257)
	tree, err := buildLightTree(places, powers)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := len(tree.nodes), 2*257-1; got != want {
		t.Fatalf("%d nodes, want %d", got, want)
	}
	seen := make([]int, 257)
	var sum float32
	for _, node := range tree.nodes {
		if node.leaf() {
			seen[node.Lamp]++
			sum += node.Power
		}
	}
	for lamp, times := range seen {
		if times != 1 {
			t.Fatalf("lamp %d is in %d leaves", lamp, times)
		}
	}
	var want float32
	for _, power := range powers {
		want += power
	}
	if math.Abs(float64(sum-want)) > 1e-3 || math.Abs(float64(tree.nodes[0].Power-want)) > 1e-3 {
		t.Errorf("leaves %v root %v, want %v", sum, tree.nodes[0].Power, want)
	}
}

// the chances of every lamp for a point sum to one, where something
// lights the point.
func TestTreeChancesSumToOne(t *testing.T) {
	tree := mustTree(t, 300)
	point, normal := [3]float32{1, 0, 2}, [3]float32{0, 1, 0}
	var sum float64
	for lamp := range int32(300) {
		sum += float64(tree.chanceOf(point, normal, lamp))
	}
	// lamps near the floor's plane may sit wholly behind; those carry
	// no chance, and the rest sum to one
	if math.Abs(sum-1) > 1e-3 {
		t.Errorf("chances sum to %v", sum)
	}
}

// sample's lamp is chosen at the rate chanceOf says.
func TestTreeSampleFollowsItsChance(t *testing.T) {
	tree := mustTree(t, 12)
	point, normal := [3]float32{0, 0, 0}, [3]float32{0, 1, 0}
	const draws = 200000
	counts := make([]int, 12)
	random := rand.New(rand.NewSource(3))
	for range draws {
		draw := float32(random.Float64())
		lamp, chance := tree.sample(point, normal, draw)
		if lamp >= 0 {
			counts[lamp]++
			if want := tree.chanceOf(point, normal, lamp); math.Abs(float64(want-chance)) > 1e-5 {
				t.Fatalf("sample says %v, chanceOf %v", chance, want)
			}
		}
	}
	for lamp, got := range counts {
		index := int32(lamp)
		want := float64(tree.chanceOf(point, normal, index))
		if math.Abs(float64(got)/draws-want) > 0.01 {
			t.Errorf("lamp %d taken %.4f, chance %.4f", lamp, float64(got)/draws, want)
		}
	}
}

// anti: a lamp wholly behind the surface has no chance, and a point with
// every lamp behind it takes none. If the behind test were dropped, the
// lamps would be taken and this fails.
func TestTreeNeverTakesALampBehindTheSurface(t *testing.T) {
	places := [][3]float32{{0, -1, 0}, {1, -2, 1}, {-3, -1, 2}}
	tree, err := buildLightTree(places, []float32{1, 1, 1})
	if err != nil {
		t.Fatal(err)
	}
	point, normal := [3]float32{0, 0, 0}, [3]float32{0, 1, 0}
	for _, u := range []float32{0, 0.3, 0.99} {
		if lamp, _ := tree.sample(point, normal, u); lamp != -1 {
			t.Errorf("took lamp %d behind the surface", lamp)
		}
	}
	places = append(places, [3]float32{0, 3, 0})
	tree, _ = buildLightTree(places, []float32{1, 1, 1, 1})
	for _, u := range []float32{0, 0.3, 0.99} {
		if lamp, _ := tree.sample(point, normal, u); lamp != 3 {
			t.Errorf("took lamp %d, want the only one in front, 3", lamp)
		}
	}
}

func TestTreeRefusesNone(t *testing.T) {
	if _, err := buildLightTree(nil, nil); err == nil {
		t.Error("a tree of no lamps")
	}
	if _, err := buildLightTree([][3]float32{{}}, nil); err == nil {
		t.Error("a tree of lamps and no powers")
	}
}

func TestTreeFloatsHoldTheNodes(t *testing.T) {
	tree := mustTree(t, 5)
	floats := tree.floats()
	if len(floats) != len(tree.nodes)*nodeFloats {
		t.Fatal("wrong length")
	}
	for index, node := range tree.nodes {
		at := floats[index*nodeFloats:]
		if at[3] != node.Power || int32(at[7]) != node.Left || int32(at[8]) != node.Right || int32(at[9]) != node.Lamp {
			t.Errorf("node %d read back wrong", index)
		}
	}
}

func TestCrowdRefusesALampWithNoPlace(t *testing.T) {
	row := func(w, kind float32) []float32 {
		r := make([]float32, lampFloats)
		r[3], r[19], r[4] = w, kind, 1
		return r
	}
	good := row(0, 0)
	if _, _, err := crowdOf(good); err != nil {
		t.Fatal(err)
	}
	sun := row(-1, 0)
	flood := row(0, 3)
	beam := row(0, 0)
	beam[13] = 1
	for name, bad := range map[string][]float32{"sun": sun, "flood": flood, "beam": beam} {
		mixed := append(append([]float32{}, good...), bad...)
		if _, _, err := crowdOf(mixed); err == nil {
			t.Errorf("a %s among the lamps was taken", name)
		}
	}
	if _, _, err := crowdOf(nil); err == nil {
		t.Error("no lamps was taken")
	}
}

// the resampling estimate of a sum is right, with the chance divided
// out. The anti-test below divides nothing and is wrong, so this test
// can tell.
func estimate(tree lightTree, f []float32, point, normal [3]float32,
	candidates int, random *rand.Rand, divide bool) float64 {
	kept := emptyReservoir()
	var phat float32
	for range candidates {
		sampleDraw := float32(random.Float64())
		lamp, chance := tree.sample(point, normal, sampleDraw)
		if lamp < 0 {
			kept.M++
			continue
		}
		weight := f[lamp]
		if divide {
			weight /= chance
		}
		offerDraw := float32(random.Float64())
		if kept.offer(lamp, weight, offerDraw) {
			phat = f[lamp]
		}
	}
	kept.finish(phat)
	if kept.Lamp < 0 {
		return 0
	}
	return float64(f[kept.Lamp] * kept.W)
}

func TestReservoirEstimateIsUnbiased(t *testing.T) {
	tree := mustTree(t, 40)
	random := rand.New(rand.NewSource(11))
	f := make([]float32, 40)
	for i := range f {
		f[i] = 0.1 + float32(random.Float64())*3
	}
	point, normal := [3]float32{0, 0, 0}, [3]float32{0, 1, 0}
	var truth float64
	for lamp := range int32(40) {
		if tree.chanceOf(point, normal, lamp) > 0 {
			truth += float64(f[lamp])
		}
	}
	mean := func(divide bool) float64 {
		var sum float64
		const trials = 60000
		for range trials {
			sum += estimate(tree, f, point, normal, 4, random, divide)
		}
		return sum / trials
	}
	if got := mean(true); math.Abs(got-truth)/truth > 0.02 {
		t.Errorf("estimate %v, truth %v", got, truth)
	}
	// anti-test: without dividing by the chance the estimate is not the
	// sum; if this passes the test above cannot tell right from wrong
	if got := mean(false); math.Abs(got-truth)/truth < 0.05 {
		t.Errorf("the undivided estimate %v is near the truth %v; the test is blind", got, truth)
	}
}

func TestReservoirKeepsNothingFromNoWeight(t *testing.T) {
	kept := emptyReservoir()
	if kept.offer(3, 0, 0) {
		t.Error("kept a candidate of no weight")
	}
	kept.finish(1)
	if kept.W != 0 || kept.Lamp != -1 {
		t.Errorf("empty reservoir has %+v", kept)
	}
}

func TestJoinCapsTheOthersCount(t *testing.T) {
	mine := reservoir{Lamp: 1, WSum: 1, M: 2, W: 1}
	theirs := reservoir{Lamp: 2, WSum: 100, M: 1000, W: 1}
	mine.join(theirs, 1, 0.5, 20)
	if mine.M != 2+40 {
		t.Errorf("M %v, want 42: the other's 1000 capped at 20 times mine", mine.M)
	}
	mine = reservoir{Lamp: 1, WSum: 1, M: 2, W: 1}
	mine.join(theirs, 1, 0.5, 0)
	if mine.M != 1002 {
		t.Errorf("uncapped M %v, want 1002", mine.M)
	}
}
