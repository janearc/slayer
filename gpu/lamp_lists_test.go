package gpu

import (
	"bytes"
	"math"
	"slices"
	"testing"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/emit"
	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
)

// mixedLamps is lamps of every kind the lists tell apart, and some they do
// not, in an order that puts none of a kind first: a plain lamp, a flood,
// a spot, a beam, the sun, a flood, a spot, and a plain lamp. The floods
// are at 1 and 5, the cones at 2 and 6, the beams at 3.
func mixedLamps(t *testing.T) []thing.Lamp {
	t.Helper()
	sun, err := emit.NewDistant(maths.V(0, 1, 0), maths.Grey(1))
	if err != nil {
		t.Fatal(err)
	}
	beam, err := emit.NewBeam(emit.Lit{At: maths.V(0, 3, 0),
		Toward: maths.V(1, 0, 0), Strength: maths.Grey(5)},
		measure.Metres(0.1), measure.Metres(4), measure.Degrees(1))
	if err != nil {
		t.Fatal(err)
	}
	spot := func(at maths.Vec) film.Spot {
		return film.Spot{At: at, Aim: maths.Vec{}.Sub(at),
			Strength: maths.Grey(30), Cone: 20 * maths.Degree}
	}
	plain := emit.Lit{At: maths.V(2, 2, 2), Strength: maths.Grey(1)}
	flood := film.Flood{Strength: maths.Grey(0.5)}
	return []thing.Lamp{plain, flood, spot(maths.V(0, 5, 1)), beam, sun,
		flood, spot(maths.V(0, 6, 2)), plain}
}

func TestTheListsHoldExactlyTheFloodsConesAndBeams(t *testing.T) {
	encoded := newScene()
	encoded.lampsOf(&thing.World{Lamps: mixedLamps(t)})
	want := lampLists{floods: []int32{1, 5}, cones: []int32{2, 6},
		beams: []int32{3}}
	got := encoded.lists
	if !slices.Equal(got.floods, want.floods) ||
		!slices.Equal(got.cones, want.cones) ||
		!slices.Equal(got.beams, want.beams) {
		t.Errorf("lists are %v, want %v", got, want)
	}
}

// Anti-test: lamps that are none of the three are on no list, and a world
// of them is given no lists in its buffer.
func TestPlainLampsAreOnNoListAndKeepNoBytes(t *testing.T) {
	encoded := newScene()
	encoded.lampsOf(&thing.World{Lamps: []thing.Lamp{
		emit.Lit{At: maths.V(1, 1, 1), Strength: maths.Grey(1)},
		emit.Lit{At: maths.V(2, 1, 1), Strength: maths.Grey(2)}}})
	if !encoded.lists.empty() {
		t.Errorf("plain lamps are on lists: %v", encoded.lists)
	}
	if len(encoded.lists.bytes()) != 0 {
		held := len(encoded.lists.bytes())
		t.Errorf("an empty list has %d bytes", held)
	}
	if !bytes.Equal(encoded.lampsRead(), encoded.lamps.bytes) {
		t.Error("the lamps' buffer holds more than the lamps' rows")
	}
}

// Anti-test: a flood is never a cone, though its toward is nought and a
// cone's is not: the cone list is what the kernel's loop counted, the
// lamps that are not floods with a width.
func TestAFloodIsNotACone(t *testing.T) {
	encoded := newScene()
	encoded.lampsOf(&thing.World{Lamps: []thing.Lamp{
		film.Flood{Strength: maths.Grey(1)}}})
	if len(encoded.lists.cones) != 0 || len(encoded.lists.floods) != 1 {
		t.Errorf("a flood is listed as %v", encoded.lists)
	}
}

// The lists follow the crowd in the lamps' buffer, in the order the
// header counts them, each index as a float.
func TestTheListsAreKeptAfterTheCrowd(t *testing.T) {
	lamps := append(crowdStreet(t), mixedLamps(t)...)
	crowd := lampCrowdOf(lamps)
	if len(crowd.nodes) == 0 {
		t.Fatal("the street has no crowd")
	}
	encoded := newScene()
	encoded.lampsOf(&thing.World{Lamps: lamps})
	encoded.camera(maths.Subspace{}, film.Pinhole{Wide: 1, Pixel: 1}, 8, 8)
	read := encoded.lampsRead()
	crowdBytes := (len(crowd.nodes)*nodeFloats + len(crowd.others)) * 4
	after := read[len(encoded.lamps.bytes)+crowdBytes:]
	if !bytes.Equal(after, encoded.lists.bytes()) {
		t.Error("the lists do not follow the crowd")
	}
	first := len(crowdStreet(t))
	var want []float32
	for _, index := range []int{1, 5, 2, 6, 3} {
		want = append(want, float32(first+index))
	}
	for at, index := range want {
		if got := wordAt(after, at); got != index {
			t.Errorf("list place %d holds %v, want %v", at, got, index)
		}
	}
	if len(after) != len(want)*4 {
		t.Errorf("%d bytes of lists, want %d", len(after), len(want)*4)
	}
}

// Anti-test: a lamp's index is a whole number a float holds exactly.
func TestAListedIndexSurvivesAFloat(t *testing.T) {
	lists := lampLists{floods: []int32{1<<24 - 1}}
	var packed = lists.bytes()
	if got := wordAt(packed, 0); float64(got) != math.Floor(float64(got)) ||
		int32(got) != 1<<24-1 {
		t.Errorf("index %d came back as %v", 1<<24-1, got)
	}
}
