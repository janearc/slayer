package trace

import (
	"math"
	"testing"

	"github.com/janearc/slayer/emit"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
)

// windows is how many standing lamps the tree tests light a street by:
// many more than lampsEvery, and not a power of two. picked.
const windows = 61

// streetOfWindows is a row of lit windows up a wall and along a
// street, each a little brighter than the last, and a sun.
func streetOfWindows(t *testing.T) []thing.Lamp {
	t.Helper()
	sun, err := emit.NewDistant(maths.V(0, 1, 0), maths.Grey(1))
	if err != nil {
		t.Fatal(err)
	}
	lamps := []thing.Lamp{sun}
	for each := range windows {
		at := maths.V(float64(each%13)*4, float64(each/13)*3, 0)
		strength := maths.Grey(1 + float64(each)/windows)
		lamps = append(lamps, emit.Lit{
			At:       at,
			Beyond:   0.1,
			Strength: strength})
	}
	return lamps
}

// chances is every lamp of the tree's chance of being walked to from a
// point, worked down the tree by the same worths a walk takes.
func chances(tree *lampTree, point maths.Vec) map[int]float64 {
	found := map[int]float64{}
	var down func(at int32, chance float64)
	down = func(at int32, chance float64) {
		node := &tree.shared.Nodes[at]
		if node.IsLeaf() {
			found[tree.lamps[node.Leaf].index] += chance
			return
		}
		left, right := &tree.shared.Nodes[node.Left],
			&tree.shared.Nodes[node.Right]
		at3 := [3]float64{point.X, point.Y, point.Z}
		toLeft, toRight := left.Worth(at3, lampWeighing),
			right.Worth(at3, lampWeighing)
		down(node.Left, chance*toLeft/(toLeft+toRight))
		down(node.Right, chance*toRight/(toLeft+toRight))
	}
	down(0, 1)
	return found
}

// TestEveryStandingLampCanBeWalkedTo says that the chances of the walks
// down the tree from a point come to one, and that every standing lamp
// has some chance: a lamp that could never be taken would leave its
// light out of every picture.
func TestEveryStandingLampCanBeWalkedTo(t *testing.T) {
	tree, others := treeOf(streetOfWindows(t))
	if tree == nil {
		t.Fatal("no tree over a street of windows")
	}
	if len(others) != 1 || others[0] != 0 {
		t.Errorf("others %v, want the sun alone", others)
	}
	for _, point := range []maths.Vec{maths.V(10, 1, 5),
		maths.V(-200, 50, 300), maths.V(24, 6, 0)} {
		found := chances(tree, point)
		sum := 0.0
		for index, chance := range found {
			if !(chance > 0) {
				t.Errorf("from %v lamp %d has chance %v", point, index,
					chance)
			}
			sum += chance
		}
		if len(found) != windows || math.Abs(sum-1) > 1e-9 {
			t.Errorf("from %v: %d lamps, chances to %v; want %d and one",
				point, len(found), sum, windows)
		}
	}
}

// TestAWalkTakesItsChance says that a walk with a share lands on a lamp
// with the chance the tree gives that lamp, so a pick's weight is right.
func TestAWalkTakesItsChance(t *testing.T) {
	tree, _ := treeOf(streetOfWindows(t))
	point := maths.V(10, 1, 5)
	want := chances(tree, point)
	const shares = 4096
	walked := map[int]int{}
	for each := range shares {
		taken, chance := tree.walk(point, (float64(each)+0.5)/shares)
		if math.Abs(chance-want[taken.index]) > 1e-9 {
			t.Fatalf("lamp %d walked with chance %v, the tree gives %v",
				taken.index, chance, want[taken.index])
		}
		walked[taken.index]++
	}
	for index, chance := range want {
		if got := float64(walked[index]) / shares; math.Abs(got-chance) >
			2.0/shares+1e-3 {
			t.Errorf("lamp %d taken %v of the time, chance %v", index,
				got, chance)
		}
	}
}

// treeStarts is how many evenly spaced starts the tree's picks are
// averaged over. With lampsAsked walks a start they lay 16,384 shares
// evenly from nought to one.
const treeStarts = 2048

// treePoint and treeFacing are where the street's picks are averaged.
var treePoint, treeFacing = maths.V(10, 1, 5), maths.V(0, 0, -1)

// street is the street of windows staged by hand, tree and all.
func street(t *testing.T) *staged {
	t.Helper()
	lamps := streetOfWindows(t)
	scene := &staged{lamps: lamps}
	scene.tree, scene.others = treeOf(lamps)
	return scene
}

// meanOfPicks is the red light a way of picking brings treePoint, each
// pick counted as it says, averaged over treeStarts evenly spaced starts.
func meanOfPicks(pick func(start float64) []lit) float64 {
	mean := 0.0
	for each := range treeStarts {
		for _, one := range pick((float64(each) + 0.5) / treeStarts) {
			mean += one.light.R * one.counts / treeStarts
		}
	}
	return mean
}

// everyWindow is the red light every standing lamp of the street sends
// treePoint, and how near to it the mean over evenly spaced starts must
// come.
//
// A walk takes a lamp from the shares in one stretch of nought to one,
// as long as the lamp's chance c. The starts and their walks lay n =
// 16,384 shares evenly, and a stretch of length c holds within one of
// n c of them. So a lamp's light l, counted one over its chance, is
// taken n c times give or take one, and averages l give or take
// l / (c n). The mean is every lamp's light to within the sum of those,
// and no nearer however right the walk: for 61 lamps taken about by
// their light that is 61 parts in 16,384 of it, four in a thousand.
func everyWindow(scene *staged) (every, near float64) {
	chance := chances(scene.tree, treePoint)
	for index, lamp := range scene.lamps[1:] {
		_, _, light := lamp.Light(treePoint, treeFacing)
		every += light.R
		near += light.R / chance[index+1] / (treeStarts * lampsAsked)
	}
	return every, near
}

// TestTheTreeSumsTheLampsOnAverage says that the picks from the tree,
// each weighed by its chance, sum on average over their starts to the
// light of every standing lamp at the point, as near as evenly spaced
// starts can tell (everyWindow).
//
// And it holds pickedFrom's own counting exactly: its picks' mean is the
// mean of each walk's lamp over its chance, walked one by one at the
// same shares, to nine places. A lamp taken twice dropped, or a count
// not shared among the walks, shows there.
func TestTheTreeSumsTheLampsOnAverage(t *testing.T) {
	scene := street(t)
	every, near := everyWindow(scene)
	mean := meanOfPicks(func(start float64) []lit {
		return scene.pickedFrom(nil, treePoint, treeFacing, start, -1,
			lampsAsked)
	})
	if !within(mean, every, near) || !(near < 0.02*every) {
		t.Errorf("picks sum to %v on average, every lamp to %v, which "+
			"evenly spaced starts tell to within %v", mean, every, near)
	}
	walked := 0.0
	for share := range treeStarts * lampsAsked {
		taken, chance := scene.tree.walk(treePoint,
			(float64(share)+0.5)/(treeStarts*lampsAsked))
		_, _, light := scene.lamps[taken.index].Light(treePoint, treeFacing)
		walked += light.R / chance / (treeStarts * lampsAsked)
	}
	if !within(mean, walked, 1e-9*every) {
		t.Errorf("picks sum to %v on average, their walks one by one "+
			"to %v", mean, walked)
	}
}

// Anti: a picking that can never take one lamp is told. The window at
// 8, 0, 0 is one of the two nearest to the point, the square root of
// thirty away (the window at 12, 0, 0 is the other),
// and sends about a seventeenth of the street's light there; picks with
// it left out fall short by that, many times what the starts' spacing
// allows.
func TestPicksThatNeverTakeOneWindowFallShort(t *testing.T) {
	scene := street(t)
	every, near := everyWindow(scene)
	const nearest = 3
	_, _, lost := scene.lamps[nearest].Light(treePoint, treeFacing)
	mean := meanOfPicks(func(start float64) []lit {
		var kept []lit
		for _, one := range scene.pickedFrom(nil, treePoint, treeFacing,
			start, -1, lampsAsked) {
			if one.index != nearest {
				kept = append(kept, one)
			}
		}
		return kept
	})
	if within(mean, every, near) || !within(mean, every-lost.R, near) {
		t.Errorf("picks without the nearest window sum to %v, every "+
			"lamp to %v and the window to %v, within %v", mean, every,
			lost.R, near)
	}
}

// TestFewLampsBuildNoTree is the tree's anti-test: a street of lampsEvery
// standing lamps or fewer is lit lamp by lamp, as before.
func TestFewLampsBuildNoTree(t *testing.T) {
	lamps := streetOfWindows(t)[:lampTreeLeast+1]
	if tree, others := treeOf(lamps); tree != nil || others != nil {
		t.Errorf("a tree over %d standing lamps", lampTreeLeast)
	}
}
