package session

import (
	"math"
	"math/rand/v2"
	"testing"

	"github.com/janearc/slayer/maths"
)

// TestMouthsAreStrewnWithinTheirSpread: every mouth strewn round a middle
// lies within the spread of it, on its level, and the same seed strews the
// same mouths.
func TestMouthsAreStrewnWithinTheirSpread(test *testing.T) {
	middle := maths.V(100, 7, -40)
	first := rand.New(rand.NewPCG(3, 9))
	second := rand.New(rand.NewPCG(3, 9))
	for range 200 {
		at := strewn(first, middle, 25)
		again := strewn(second, middle, 25)
		if at != again {
			test.Fatal("the same seed strewed two places")
		}
		if math.Hypot(at.X-middle.X, at.Z-middle.Z) > 25 || at.Y != middle.Y {
			test.Fatalf("a mouth at %v lies out of its spread", at)
		}
	}
}

// TestMouthsAreNotAllInOnePlace: strewn mouths are spread, not heaped on
// the middle.
func TestMouthsAreNotAllInOnePlace(test *testing.T) {
	places := rand.New(rand.NewPCG(3, 9))
	far := 0
	for range 200 {
		at := strewn(places, maths.Vec{}, 25)
		if math.Hypot(at.X, at.Z) > 12.5 {
			far++
		}
	}
	if far == 0 || far == 200 {
		test.Fatalf("%d of 200 mouths lie past half the spread", far)
	}
}

// TestAPlumeIsTurnedDownTheWind: the plume's own x is the way the wind
// blows: a wind from the north-east carries it south-west.
func TestAPlumeIsTurnedDownTheWind(test *testing.T) {
	downwind := skyward(45, windHorizon).Neg()
	turn := maths.Rotation(maths.YAxis, math.Atan2(-downwind.Z, downwind.X))
	along := maths.NewSubspace(maths.Vec{}, turn, 1).Out(maths.XAxis)
	if along.Sub(downwind).Len() > 1e-9 {
		test.Fatalf("the plume runs %v, the wind %v", along, downwind)
	}
	if !(along.X < 0 && along.Z > 0) {
		test.Fatalf("a wind from the north-east blew %v", along)
	}
}
