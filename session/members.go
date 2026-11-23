package session

import (
	"math/rand/v2"

	"github.com/janearc/slayer/manufacture/cities"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/set"
)

// A stand's members are placed each by itself: the place of member number
// so many comes from a stream keyed by the stand's seed and that number,
// and from nothing placed before it. So the places of a stand's members are
// the same whichever of them are asked for, and one region of a stand is
// found without placing the rest.
//
// The stand's count is how many numbers it has, from nought up, and every
// number is placed or refused, so the total is the count whatever the
// ground. Each stream is uniform over the region the stand offers, so the
// density over its band is as even as when one stream laid them all in turn.

// memberStream is the stream of random draws of one member of a stand, from
// the stand's seed and the member's number alone: the same stream the
// scatter habit draws from.
func memberStream(seed uint64, member int) *rand.Rand {
	return cities.MemberStream(seed, member)
}

// chooser is how a stand's member is given a place from its stream: the
// ground it is offered on, the band it is held to.
type chooser func(set.Stand, *rand.Rand) (maths.Vec, error)

// memberPlace is the place of one member of a stand, from its own stream.
func memberPlace(each set.Stand, choose chooser, seed uint64,
	member int) (maths.Vec, error) {
	return choose(each, memberStream(seed, member))
}

// holds is whether a point is inside a box, edges included.
func holds(box maths.Box, point maths.Vec) bool {
	return point.X >= box.Min.X && point.X <= box.Max.X &&
		point.Y >= box.Min.Y && point.Y <= box.Max.Y &&
		point.Z >= box.Min.Z && point.Z <= box.Max.Z
}
