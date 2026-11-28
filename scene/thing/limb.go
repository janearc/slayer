package thing

import "github.com/janearc/slayer/maths"

// LimbShare is how much of its central light an emitting ball gives
// where a look meets it at an angle to the normal: one minus the
// coefficient times one minus the cosine of that angle, which is the
// linear limb-darkening law (Milne 1921; from memory, to shelve). It is
// one at the centre of the disc and one minus the coefficient at its
// edge.
//
// The normal and the way back to the eye need not be unit; a look from
// behind the surface, or along it, gets the edge's share. A coefficient
// of nought is no darkening, and one that leaves the edge black or
// below is held to one.
func LimbShare(coefficient float64, normal, toward maths.Vec) float64 {
	if !(coefficient > limbNone) {
		return limbWhole
	}
	coefficient = min(coefficient, limbWhole)
	cosine := limbNone
	if size := normal.Len() * toward.Len(); size > limbNone {
		cosine = max(limbNone, normal.Dot(toward)/size)
	}
	return limbWhole - coefficient*(limbWhole-cosine)
}

// limbNone is no darkening, and the cosine of a look along the surface.
// It is defined as nought.
const limbNone = 0.0

// limbWhole is the share at the centre of the disc, and the greatest a
// limb coefficient may be, which leaves the edge black. It is defined as
// one.
const limbWhole = 1.0
