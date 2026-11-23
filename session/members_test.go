package session

import (
	"math/rand/v2"
	"testing"

	"github.com/janearc/slayer/maths"
)

// drawsOf is the first few draws of a member's stream.
func drawsOf(seed uint64, member int) [4]float64 {
	stream := memberStream(seed, member)
	return [4]float64{stream.Float64(), stream.Float64(), stream.Float64(),
		stream.Float64()}
}

func TestAMembersStreamIsTheSameForTheSameSeedAndNumber(t *testing.T) {
	if drawsOf(9, 5) != drawsOf(9, 5) {
		t.Error("a member's stream changed between two askings")
	}
}

func TestMembersAndSeedsHaveStreamsOfTheirOwn(t *testing.T) {
	seen := map[[4]float64]string{}
	for seed := range uint64(3) {
		for member := range 50 {
			draws := drawsOf(seed, member)
			if other, same := seen[draws]; same {
				t.Fatalf("seed %d member %d draws as %s", seed, member, other)
			}
			seen[draws] = "another"
		}
	}
}

func TestAMembersStreamDoesNotDependOnTheOnesAskedBefore(t *testing.T) {
	alone := drawsOf(4, 17)
	for member := range 17 {
		drawsOf(4, member)
	}
	if drawsOf(4, 17) != alone {
		t.Error("member 17 drew differently after the members before it")
	}
}

func TestEachStreamIsEvenOverTheUnitInterval(t *testing.T) {
	const members, halves = 4000, 2
	below := 0
	for member := range members {
		if memberStream(1, member).Float64() < 1.0/halves {
			below++
		}
	}
	if below < members*45/100 || below > members*55/100 {
		t.Errorf("%d of %d first draws fell below a half", below, members)
	}
}

func TestHoldsIsWhetherAPointIsInsideABoxEdgesIncluded(t *testing.T) {
	box := maths.Box{Min: maths.V(0, 0, 0), Max: maths.V(1, 1, 1)}
	for point, want := range map[maths.Vec]bool{maths.V(.5, .5, .5): true,
		maths.V(1, 1, 1): true, maths.V(1.1, .5, .5): false,
		maths.V(.5, -.1, .5): false} {
		if holds(box, point) != want {
			t.Errorf("%v: holds is not %v", point, want)
		}
	}
}

func TestAChooserIsGivenTheMembersStream(t *testing.T) {
	var used *rand.Rand
	pick := func(stream *rand.Rand) { used = stream }
	pick(memberStream(2, 3))
	if used.Float64() != memberStream(2, 3).Float64() {
		t.Error("the stream a chooser is given is not the member's")
	}
}
