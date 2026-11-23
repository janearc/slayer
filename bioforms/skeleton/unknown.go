package skeleton

import (
	"slices"
)

// Unknown is the names pose turns that name no bone of the skeleton, in
// order. Frames passes over such a turn, so a maker checks a pose with
// this before drawing it: a mistyped or renamed joint would otherwise
// leave a limb where it was, with no word said.
func (skeleton *Skeleton) Unknown(pose Pose) []string {
	unknown := []string{}
	for name := range pose.turns {
		if _, ok := skeleton.byName[name]; !ok {
			unknown = append(unknown, name)
		}
	}
	slices.Sort(unknown)
	return unknown
}
