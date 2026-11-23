package scrooge

import (
	"iter"
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
)

// fake is a made-up node: a cube that gives fan smaller cubes along its
// x, each a fan-th of its side and of its light, for depth layers. It
// counts what it makes. Its flags make it break the contract one way
// each, for the checker's anti-tests.
type fake struct {
	box          maths.Box
	light        maths.Light
	fan, depth   int
	worth, under float64
	made         *int
	stray        bool // its children stand outside it
	dim          bool // its children lose half its light
	eager        bool // it gives children finer than asked
}

func cube(low maths.Vec, side float64) maths.Box {
	return maths.Box{Min: low, Max: low.Add(maths.V(side, side, side))}
}

func (one *fake) side() float64 { return one.box.Max.X - one.box.Min.X }

func (one *fake) Aggregate() Aggregate {
	made := Aggregate{Box: one.box, Emission: one.light}
	if one.depth > 0 {
		made.Grain = made.Size() / float64(one.fan)
	}
	return made
}

func (one *fake) Coverage(float64) float64 { return 1 }

func (one *fake) Detail(at Footprint) iter.Seq[Node] {
	return func(yield func(Node) bool) {
		if one.depth == 0 {
			return
		}
		side := one.side() / float64(one.fan)
		finer := side*math.Sqrt(3) < at.WidthAt(at.Nearest(one.box))
		if finer && !one.eager {
			return
		}
		share := 1 / float64(one.fan)
		if one.dim {
			share /= 2
		}
		for i := range one.fan {
			offset := maths.V(float64(i)*side, 0, 0)
			low := one.box.Min.Add(offset)
			if one.stray {
				lift := maths.V(0, 2*one.side(), 0)
				low = low.Add(lift)
			}
			child := &fake{box: cube(low, side),
				light: one.light.Scale(share), fan: one.fan,
				depth: one.depth - 1, worth: one.under, under: one.under,
				made: one.made, eager: one.eager}
			*one.made++
			if !yield(child) {
				return
			}
		}
	}
}

// flat asks with a footprint of one width everywhere, and takes a fake's
// own worth.
type flat struct{ width float64 }

func (asker flat) Footprint(Node) Footprint {
	return Footprint{Width: asker.width}
}

func (asker flat) Worth(node Node) float64 { return node.(*fake).worth }

func top(side float64, depth int, worth, under float64, made *int) *fake {
	return &fake{box: cube(maths.Vec{}, side), light: maths.Grey(64),
		fan: 4, depth: depth, worth: worth, under: under, made: made}
}

// TestACutOpensNothingFinerThanItsFootprint says that a cube 64 m a side,
// asked at 5 m, is opened twice, to sixteen cubes 4 m a side, and that
// their own layer, 1 m a side, is never made.
func TestACutOpensNothingFinerThanItsFootprint(t *testing.T) {
	made := 0
	cut, err := NewCut([]Node{top(64, 3, 1, 1, &made)}, flat{5}, Budget{})
	if err != nil {
		t.Fatal(err)
	}
	spent := cut.Refine()
	if got := len(cut.Leaves()); got != 16 {
		t.Errorf("%d leaves, want 16", got)
	}
	if made != 20 || spent.Opened != 5 || spent.Short {
		t.Errorf("made %d nodes in %d opens (short %v), want 20 in 5",
			made, spent.Opened, spent.Short)
	}
}

// TestACutAskedCoarselyOpensNothing is the anti-test: at a footprint
// wider than the cube, it stays one node and makes nothing.
func TestACutAskedCoarselyOpensNothing(t *testing.T) {
	made := 0
	cut, _ := NewCut([]Node{top(64, 3, 1, 1, &made)}, flat{200}, Budget{})
	cut.Refine()
	if len(cut.Leaves()) != 1 || made != 0 {
		leaves := len(cut.Leaves())
		t.Errorf("%d leaves and %d made, want 1 and none",
			leaves, made)
	}
}

// TestANodeWorthNothingIsNeverOpened says that a node the picture cannot
// show, as one behind closed pixels, is never asked for detail.
func TestANodeWorthNothingIsNeverOpened(t *testing.T) {
	made := 0
	cut, _ := NewCut([]Node{top(64, 3, 0, 1, &made)}, flat{5}, Budget{})
	if spent := cut.Refine(); spent.Opened != 0 || made != 0 {
		t.Errorf("opened %d and made %d, want none", spent.Opened, made)
	}
}

// TestACutStaysInsideItsBudget says that where every node matters the
// same, a cut of at most ten nodes stops short: it holds no more than
// ten, and the node whose opening would pass ten is closed again.
func TestACutStaysInsideItsBudget(t *testing.T) {
	made := 0
	cut, _ := NewCut([]Node{top(64, 3, 1, 1, &made)}, flat{5},
		Budget{Nodes: 10})
	spent := cut.Refine()
	if !spent.Short || spent.Held > 10 {
		t.Errorf("held %d (short %v), want at most 10 and short",
			spent.Held, spent.Short)
	}
	if got := len(cut.Leaves()); got != 7 {
		t.Errorf("%d leaves, want 7: three cubes unopened and one "+
			"opened to four", got)
	}
}

// TestAWorthierNodeDisplacesALesser says that under a budget the cut
// forgets an opened node for one that matters more: a lesser cube opened
// first is closed when a worthier one's children are opened, and stands
// as a leaf again.
func TestAWorthierNodeDisplacesALesser(t *testing.T) {
	made := 0
	lesser := top(64, 1, 8.5, 1, &made)
	greater := top(64, 2, 8, 9, &made)
	greater.box = cube(maths.V(100, 0, 0), 64)
	cut, _ := NewCut([]Node{lesser, greater}, flat{5}, Budget{Nodes: 16})
	spent := cut.Refine()
	if spent.Held > 16 || spent.Forgotten == 0 {
		t.Errorf("held %d, forgot %d", spent.Held, spent.Forgotten)
	}
	found := false
	for _, leaf := range cut.Leaves() {
		found = found || leaf == Node(lesser)
	}
	if !found {
		t.Error("the lesser cube is not a leaf again")
	}
}

// TestAnOpenLimitStopsARefining says that a refining opens no more nodes
// than its budget allows, and says it stopped short.
func TestAnOpenLimitStopsARefining(t *testing.T) {
	made := 0
	cut, _ := NewCut([]Node{top(64, 3, 1, 1, &made)}, flat{5},
		Budget{Opens: 2})
	if spent := cut.Refine(); spent.Opened != 2 || !spent.Short {
		t.Errorf("opened %d (short %v), want 2 and short", spent.Opened,
			spent.Short)
	}
}

// TestACutIsRefinedForSomebody is the refusal of a cut with no asker.
func TestACutIsRefinedForSomebody(t *testing.T) {
	if _, err := NewCut(nil, nil, Budget{}); err != ErrNoAsker {
		t.Errorf("got %v, want ErrNoAsker", err)
	}
}

// TestFrugalPassesAnHonestNode says the checker finds nothing wrong with
// a node that keeps the contract, at a fine footprint and a coarse one.
func TestFrugalPassesAnHonestNode(t *testing.T) {
	made := 0
	for _, width := range []float64{5, 200} {
		if err := Frugal(top(64, 3, 1, 1, &made),
			Footprint{Width: width}, 1e-9); err != nil {
			t.Errorf("at %g m: %v", width, err)
		}
	}
}

// TestFrugalCatchesEachBreach is the anti-test: a node whose children
// stand outside it, one whose children lose its light, and one that
// gives layers finer than asked are each caught.
func TestFrugalCatchesEachBreach(t *testing.T) {
	made := 0
	stray, dim, eager := top(64, 3, 1, 1, &made), top(64, 3, 1, 1, &made),
		top(64, 3, 1, 1, &made)
	stray.stray, dim.dim, eager.eager = true, true, true
	for name, one := range map[string]struct {
		node  *fake
		width float64
	}{"stray": {stray, 5}, "dim": {dim, 5}, "eager": {eager, 200}} {
		if Frugal(one.node, Footprint{Width: one.width}, 1e-9) == nil {
			t.Errorf("a %s node passed", name)
		}
	}
}

// TestAFootprintWidensWithDistance says a cone's width is its start's
// plus its spread a metre, and that its nearest distance to a box it
// starts inside is nought.
func TestAFootprintWidensWithDistance(t *testing.T) {
	cone := Footprint{From: maths.V(-10, 5, 5), Width: 1, Spread: 0.5}
	if got := cone.WidthAt(10); got != 6 {
		t.Errorf("width %g at 10 m, want 6", got)
	}
	box := cube(maths.Vec{}, 10)
	if got := cone.Nearest(box); got != 10 {
		t.Errorf("nearest %g, want 10", got)
	}
	cone.From = maths.V(5, 5, 5)
	if got := cone.Nearest(box); got != 0 {
		t.Errorf("nearest %g from inside, want 0", got)
	}
}
