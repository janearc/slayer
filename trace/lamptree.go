package trace

import (
	"math"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/strategies"
	"github.com/janearc/slayer/scene/thing"
)

// A scene of many small lamps, a city's lit windows, was lit by working
// out every lamp's light at every point it shaded, to pick lampsAsked of
// them: at every step of a ray through haze, thousands of lamps priced
// to light it by eight.
//
// The lamp tree is built once a picture, over the
// lamps that stand somewhere and fall off by the inverse square, and a
// point picks among them by walking down it: at each fork, the way of
// the brighter, nearer half, more often.
//
// A pick then costs the tree's
// depth and not the lamps' count, and its weight is one over the chance
// of the walk that took it, so the sum is the same on average as the
// sum of every lamp's light.
//
// Lamps of no place, a sun or the sky, and lamps that are not points,
// beams and glowing volumes, are outside the tree. Ambient lamps are
// asked every time, and a step through a medium asks every distant lamp;
// the rest are picked among by worth (askedTreed).

// lampTreeLeast is the fewest standing lamps a scene must have for a tree
// to be worth building. picked: past lampsEvery, which every lamp is
// lit exactly up to, the old picking prices each of them at each point.
const lampTreeLeast = lampsEvery

// standingLamp is a lamp at a place whose light falls off by the
// inverse square, with an upper bound on the light it sends: emit.Lit.
type standingLamp interface {
	Stands() maths.Vec
	AtMost(distance float64) float64
}

// treeLamp is a lamp of the tree: its index among the scene's lamps,
// where it stands, and the most light it sends a unit away.
type treeLamp struct {
	index int
	at    maths.Vec
	power float64
}

// lampTree is the tree: the shared light tree over the lamps' places and
// powers, whose leaves are numbers into lamps.
type lampTree struct {
	shared strategies.LightTree[float64]
	lamps  []treeLamp
}

// lampLeaf marks a node with no halves.
const lampLeaf = strategies.NoNode

// lampWeighing is how this tracer weighs a node for a point: a millimetre
// squared at least from it, and half the box's diagonal for a point inside
// the box. The gpu's weighing differs; see strategies.Weighing.
var lampWeighing = strategies.Weighing[float64]{Floor: nearestSquare,
	InsideHalfDiagonal: true}

// lampOrder is how lamps of one coordinate are ordered when a node splits:
// as the sort leaves them, which is how the tracer's trees have always been
// built. The gpu orders them by index; see strategies.Order.
const lampOrder = strategies.ByCoordinate

// treeOf is the tree over a scene's standing lamps, and the indices of
// every other lamp, or no tree when there are too few standing lamps to
// need one. A standing lamp that sends no light is left out of both.
func treeOf(lamps []thing.Lamp) (*lampTree, []int) {
	var standing []treeLamp
	var others []int
	for index, lamp := range lamps {
		placed, stands := lamp.(standingLamp)
		if !stands {
			others = append(others, index)
			continue
		}
		power := placed.AtMost(1)
		if !(power > 0) || math.IsInf(power, 0) {
			if power != 0 {
				others = append(others, index)
			}
			continue
		}
		standing = append(standing, treeLamp{index: index,
			at: placed.Stands(), power: power})
	}
	if len(standing) <= lampTreeLeast {
		return nil, nil
	}
	places := make([][3]float64, len(standing))
	powers := make([]float64, len(standing))
	for number, each := range standing {
		places[number] = [3]float64{each.at.X, each.at.Y, each.at.Z}
		powers[number] = each.power
	}
	// the build refuses only no lamps, or places and powers that are not as
	// many, and these are as many and more than lampTreeLeast
	shared, _ := strategies.BuildLightTree(places, powers, lampOrder)
	return &lampTree{shared: shared, lamps: standing}, others
}

// nearestSquare is the least squared distance a point may be taken to be
// from a node's lamps: a guard against dividing by nothing at a lamp, a
// millimetre squared. picked: far under any lamp's Beyond.
const nearestSquare = 1e-6

// walk is the lamp a walk down the tree from the root takes, with a
// unit share choosing the way at each fork, and the chance of taking it;
// a chance of nought where no lamp could light the point.
func (tree *lampTree) walk(point maths.Vec, share float64) (treeLamp,
	float64) {
	lamp, chance := strategies.Walk(tree.shared,
		[3]float64{point.X, point.Y, point.Z}, share, lampWeighing)
	if lamp == lampLeaf {
		return treeLamp{}, 0
	}
	return tree.lamps[lamp], chance
}

// lampSplitMost is the most nodes the tree's lamps are split into at a
// step through a medium, so the most lamps the tree gives a step. It is
// lampsAsked, so that a step through haze costs no more shadow rays than
// a point on a surface does. picked.
const lampSplitMost = lampsAsked

// lampSplitSpread is how wide across a node's box may be, in distances
// from the point to the box, before the node is split and not picked
// among (LightNode.Reliable). picked: one.
//
// A lamp of a node left whole is then at most twice as far as the box is,
// and lights the point within four times as much as any other of its
// node's, so one pick weighs at most four times the light of its node.
const lampSplitSpread = 1.0

// lampShareStep is added to a point's start for each node after the
// first, to take a node's share: the golden ratio's fraction, so that the
// shares of the nodes of one step are not alike.
const lampShareStep = 0.6180339887498949

// walkFrom is walk begun at a node, which is the whole of the lamps that
// can be taken. For a node whose worth is Reliable the point sees its
// lamps at near enough one distance, and they are taken by their power,
// whatever the point; for one still in doubt, which only a full cut leaves,
// by the point's worth of each half (strategies.WalkFrom).
func (tree *lampTree) walkFrom(node int32, point maths.Vec,
	share float64) (treeLamp, float64) {
	at := [3]float64{point.X, point.Y, point.Z}
	var lamp int32
	var chance float64
	if tree.shared.Nodes[node].Reliable(at, lampSplitSpread) {
		lamp, chance = strategies.WalkByPower(tree.shared, node, share)
	} else {
		lamp, chance = strategies.WalkFrom(tree.shared, node, at, share,
			lampWeighing)
	}
	if lamp == lampLeaf {
		return treeLamp{}, 0
	}
	return tree.lamps[lamp], chance
}

// splitFrom is pickedFrom for a step through a medium, after Conty
// Estevez and Kulla, "Importance sampling of many lights with adaptive
// tree splitting" (2018). A walk from the root picks a fork's half by its
// worth, and a node's worth is wrong where the point is inside its box, or
// near it for its size: a lamp close by that sits in a wide box is taken
// rarely and then counted for it, a weight many times the step's light.
//
// Here the tree's lamps are cut into at most most nodes (strategies.Cut),
// the doubtful nodes split and not picked among, and one lamp is taken
// from each node, counted one over its chance among the node's lamps.
//
// The cut depends on the point alone, and the nodes' lamps are all the
// lamps, so the sum is the same on average as before. The picks are
// appended to into.
func (scene *staged) splitFrom(into []lit, point, facing maths.Vec,
	start float64, most int) []lit {
	made := into
	at := [3]float64{point.X, point.Y, point.Z}
	cut := strategies.Cut(scene.tree.shared, at, lampSplitSpread, most,
		nearestSquare)
	for number, node := range cut {
		share := math.Mod(start+float64(number)*lampShareStep, 1)
		taken, chance := scene.tree.walkFrom(node, point, share)
		if !(chance > 0) {
			continue
		}
		toward, far, light := scene.lamps[taken.index].Light(point, facing)
		if light == (maths.Light{}) {
			continue
		}
		made = append(made, lit{index: taken.index, toward: toward,
			far: far, light: light, counts: 1 / chance})
	}
	return made
}

// pickedFrom is walks down the tree from a point, their shares spread
// evenly from start, each lamp taken lit as it lights the point and
// weighed by one over its walk's chance and the count of walks. A lamp
// taken twice is one entry counted twice; the lamp at skip is taken for
// nothing; a lamp that sends the point no light adds none. The picks are
// appended to into.
func (scene *staged) pickedFrom(into []lit, point, facing maths.Vec,
	start float64, skip, walks int) []lit {
	made := into
	first := len(into)
	for walk := range walks {
		share := (start + float64(walk)) / float64(walks)
		taken, chance := scene.tree.walk(point, share)
		if !(chance > 0) || taken.index == skip {
			continue
		}
		counts := 1 / (chance * float64(walks))
		again := false
		for k := first; k < len(made); k++ {
			if made[k].index == taken.index {
				made[k].counts += counts
				again = true
				break
			}
		}
		if again {
			continue
		}
		toward, far, light := scene.lamps[taken.index].Light(point, facing)
		if light == (maths.Light{}) {
			continue
		}
		made = append(made, lit{index: taken.index, toward: toward,
			far: far, light: light, counts: counts})
	}
	return made
}
