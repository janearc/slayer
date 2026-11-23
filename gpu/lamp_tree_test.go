package gpu

import (
	"bytes"
	"encoding/binary"
	"math"
	"testing"

	"github.com/janearc/slayer/emit"
	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/strategies"
	"github.com/janearc/slayer/scene/thing"
)

// crowdWindows is how many standing lamps the crowd tests light a street
// by: many more than lampsEvery, and not a power of two.
const crowdWindows = 40

// crowdSunAt is where the sun stands among the street's lamps, not first,
// so that an index read back as nought is not the sun by chance.
const crowdSunAt = 3

// crowdFloor is lamp_tree_floor in trace.metal, the least squared distance
// the kernel weighs a node at, as trace's nearestSquare.
const crowdFloor = 1e-6

// rootPowerShare is how far the root's power may be from the sum of the
// windows' as a share of it: the two add the same float32s in different
// orders, and a float32 holds about seven digits.
const rootPowerShare = 1e-3

// crowdStreet is a row of lit windows up a wall and along a street,
// each a little brighter than the last, and a sun among them at
// crowdSunAt.
func crowdStreet(t *testing.T) []thing.Lamp {
	t.Helper()
	sun, err := emit.NewDistant(maths.V(0, 1, 0), maths.Grey(1))
	if err != nil {
		t.Fatal(err)
	}
	var lamps []thing.Lamp
	for each := range crowdWindows {
		if len(lamps) == crowdSunAt {
			lamps = append(lamps, sun)
		}
		at := maths.V(float64(each%13)*4, float64(each/13)*3, 0)
		strength := maths.Grey(1 + float64(each)/crowdWindows)
		lamps = append(lamps, emit.Lit{
			At:       at,
			Beyond:   0.1,
			Strength: strength})
	}
	return lamps
}

// leavesOf is the lamps the leaves of a crowd's tree hold, by how many
// leaves hold each.
func leavesOf(crowd lampCrowd) map[int32]int {
	found := map[int32]int{}
	for _, node := range crowd.nodes {
		if node.leaf() {
			found[node.Lamp]++
		}
	}
	return found
}

// chancesOf is every lamp of a crowd's chance of being walked to from a
// point, worked down the tree by the weighing the kernel's walk has: the
// shared Worth, in float32, with trace's weighing.
func chancesOf(crowd lampCrowd, point [3]float32) map[int32]float32 {
	by := strategies.Weighing[float32]{Floor: crowdFloor,
		InsideHalfDiagonal: true}
	shared := func(at int32) strategies.LightNode[float32] {
		node := crowd.nodes[at]
		return strategies.LightNode[float32]{Low: node.Low,
			High: node.High, Power: node.Power}
	}
	found := map[int32]float32{}
	var down func(at int32, chance float32)
	down = func(at int32, chance float32) {
		node := crowd.nodes[at]
		if node.leaf() {
			found[node.Lamp] += chance
			return
		}
		toLeft := shared(node.Left).Worth(point, by)
		toRight := shared(node.Right).Worth(point, by)
		both := toLeft + toRight
		down(node.Left, chance*toLeft/both)
		down(node.Right, chance*toRight/both)
	}
	down(0, 1)
	return found
}

// wordAt is the float at a place among a buffer's bytes, a place counted
// in floats.
func wordAt(buffer []byte, place int) float32 {
	return math.Float32frombits(binary.LittleEndian.Uint32(buffer[4*place:]))
}

// A world of lampsEvery lamps or fewer has no crowd, and its lamps'
// buffer is the lamps' rows and nothing after: the kernel lights it lamp
// by lamp as it always did.
func TestAWorldOfLampsEveryOrFewerHasNoCrowd(t *testing.T) {
	var lamps []thing.Lamp
	for each := range lampsEvery {
		at := maths.V(float64(each), 1, 0)
		lamps = append(lamps, emit.Lit{At: at,
			Strength: maths.Grey(1)})
	}
	if crowd := lampCrowdOf(lamps); crowd.nodes != nil || crowd.others != nil {
		t.Errorf("%d lamps have a crowd of %d nodes", len(lamps),
			len(crowd.nodes))
	}
	encoded := newScene()
	encoded.lampsOf(&thing.World{Lamps: lamps})
	if !bytes.Equal(encoded.lampsRead(), encoded.lamps.bytes) {
		t.Error("the lamps' buffer holds more than the lamps' rows")
	}
}

// Anti-test: a world of many lamps with few that stand has no tree, and
// no lamp is kept outside it, for the kernel prices every lamp as it did.
func TestFewStandingLampsAmongManyAreNoCrowd(t *testing.T) {
	sun, err := emit.NewDistant(maths.V(0, 1, 0), maths.Grey(1))
	if err != nil {
		t.Fatal(err)
	}
	lamps := []thing.Lamp{sun}
	for each := range lampTreeLeast {
		at := maths.V(float64(each), 1, 0)
		lamps = append(lamps, emit.Lit{At: at,
			Strength: maths.Grey(1)})
	}
	if len(lamps) <= lampsEvery {
		t.Fatalf("%d lamps are not more than lampsEvery", len(lamps))
	}
	if crowd := lampCrowdOf(lamps); crowd.nodes != nil || crowd.others != nil {
		t.Errorf("%d standing lamps made a crowd of %d nodes and %d "+
			"outside", lampTreeLeast, len(crowd.nodes), len(crowd.others))
	}
}

// The street's windows are the tree's leaves, each once, and the sun,
// which stands nowhere, is outside it.
func TestTheStreetIsATreeAndTheSunStandsOutside(t *testing.T) {
	crowd := lampCrowdOf(crowdStreet(t))
	if want := 2*crowdWindows - 1; len(crowd.nodes) != want {
		t.Errorf("the tree has %d nodes, want %d", len(crowd.nodes), want)
	}
	if len(crowd.others) != 1 || crowd.others[0] != crowdSunAt {
		t.Errorf("the lamps outside are %v, want only the sun, %d",
			crowd.others, crowdSunAt)
	}
	leaves := leavesOf(crowd)
	if len(leaves) != crowdWindows {
		t.Errorf("the leaves hold %d lamps, want %d", len(leaves),
			crowdWindows)
	}
	for lamp, count := range leaves {
		if lamp == crowdSunAt || lamp < 0 || lamp > crowdWindows {
			t.Errorf("a leaf holds lamp %d, which is not a window", lamp)
		}
		if count != 1 {
			t.Errorf("lamp %d is in %d leaves", lamp, count)
		}
	}
}

// Anti-test: a lamp that stands and sends no light is in neither the
// tree nor outside it, as in trace's treeOf: a pick of it would be a pick
// for nothing, and asking it every time would ask nothing.
func TestALampThatSendsNoLightIsLeftOutOfBoth(t *testing.T) {
	lamps := append(crowdStreet(t), emit.Lit{At: maths.V(1, 1, 1)})
	dark := int32(len(lamps) - 1)
	crowd := lampCrowdOf(lamps)
	if _, found := leavesOf(crowd)[dark]; found {
		t.Error("a dark lamp is a leaf of the tree")
	}
	for _, outside := range crowd.others {
		if outside == dark {
			t.Error("a dark lamp is kept outside the tree")
		}
	}
}

// A spot stands somewhere, and falls off by the inverse square, so it is
// a leaf as trace's treeOf makes it one.
func TestASpotIsALeafOfTheTree(t *testing.T) {
	lamps := append(crowdStreet(t), film.Spot{At: maths.V(2, 2, 2),
		Aim: maths.V(0, -1, 0), Cone: 1, Soft: 0.25,
		Strength: maths.Grey(3)})
	spot := int32(len(lamps) - 1)
	if leavesOf(lampCrowdOf(lamps))[spot] != 1 {
		t.Error("the spot is not a leaf of the tree")
	}
}

// The tree and the outside lamps are kept after the lamps' rows, and the
// header says how many there are of each.
func TestTheCrowdIsKeptAfterTheLampsRows(t *testing.T) {
	lamps := crowdStreet(t)
	crowd := lampCrowdOf(lamps)
	encoded := newScene()
	encoded.lampsOf(&thing.World{Lamps: lamps})
	encoded.camera(maths.Subspace{}, film.Pinhole{Wide: 1, Pixel: 1}, 8, 8)
	read := encoded.lampsRead()
	rows := len(lamps) * lampStruct.size()
	if !bytes.Equal(read[:rows], encoded.lamps.bytes) {
		t.Error("the lamps' rows are not first in the buffer")
	}
	want := (len(crowd.nodes)*nodeFloats + len(crowd.others)) * 4
	if len(read)-rows != want {
		t.Errorf("%d bytes follow the lamps' rows, want %d",
			len(read)-rows, want)
	}
	after := read[rows:]
	var power float32
	for _, window := range lamps[:crowdSunAt] {
		power += float32(window.(emit.Lit).AtMost(1))
	}
	for _, window := range lamps[crowdSunAt+1:] {
		power += float32(window.(emit.Lit).AtMost(1))
	}
	got := wordAt(after, 3)
	if math.Abs(float64(got-power)) > rootPowerShare*float64(power) {
		t.Errorf("the root's power is %v, want the windows' %v", got, power)
	}
	outside := len(crowd.nodes) * nodeFloats
	if got := wordAt(after, outside); got != crowdSunAt {
		t.Errorf("the lamp outside is %v, want the sun, %d", got, crowdSunAt)
	}
	offset := fieldOffset(headerStruct, "lamp_tree")
	header := encoded.header.bytes[offset:]
	nodes := binary.LittleEndian.Uint32(header)
	others := binary.LittleEndian.Uint32(header[4:])
	if int(nodes) != len(crowd.nodes) || others != 1 {
		t.Errorf("the header says %d nodes and %d outside, want %d and 1",
			nodes, others, len(crowd.nodes))
	}
}

// Anti-test: a world with no crowd says none in its header, which is what
// keeps the kernel on the lamp by lamp path.
func TestAWorldWithNoCrowdSaysNoneInItsHeader(t *testing.T) {
	encoded := newScene()
	encoded.lampsOf(&thing.World{Lamps: []thing.Lamp{
		emit.Lit{At: maths.V(0, 1, 0), Strength: maths.Grey(1)}}})
	encoded.camera(maths.Subspace{}, film.Pinhole{Wide: 1, Pixel: 1}, 8, 8)
	offset := fieldOffset(headerStruct, "lamp_tree")
	field := encoded.header.bytes[offset : offset+16]
	for _, word := range field {
		if word != 0 {
			t.Fatalf("the header's lamp_tree is %v", field)
		}
	}
}

// From any point, the chances of the lamps a walk takes come to one, and
// every window has some: a lamp that could never be taken would leave its
// light out of every picture. The points are among the windows, on one,
// far from all and in a gap.
func TestTheWalksChancesComeToOneFromEveryPoint(t *testing.T) {
	crowd := lampCrowdOf(crowdStreet(t))
	points := [][3]float32{{10, 3, 0}, {4, 0, 0}, {500, 400, 300},
		{-80, 2, 9}, {2, 1.5, 0.01}}
	for _, point := range points {
		chances := chancesOf(crowd, point)
		var sum float32
		for _, chance := range chances {
			sum += chance
		}
		if math.Abs(float64(sum-1)) > 1e-4 {
			t.Errorf("from %v the chances sum to %v", point, sum)
		}
		if len(chances) != crowdWindows {
			t.Errorf("from %v %d windows can be taken, want %d", point,
				len(chances), crowdWindows)
		}
		for lamp, chance := range chances {
			if !(chance > 0) {
				t.Errorf("from %v lamp %d has chance %v", point, lamp,
					chance)
			}
		}
	}
}
