package main

import (
	"github.com/janearc/slayer/maths"
)

// launcherUnit is defined as the unchanged unit scale of a placed tube.
const launcherUnit = 1

// launcher shares one ordinary factory mesh across all visible launch tubes.
// The tube ends at the beam origin, with its local y axis along the beam.
func (stage *staging) launcher(exit, direction maths.Vec) error {
	const kind = "guide-star-launcher"
	key := stocked{name: kind, pick: placementZero, detail: "quick"}
	one, found := stage.have[key]
	if !found {
		fresh, err := stage.constructed(kind, "", key.detail, uint64(key.pick))
		if err != nil {
			return err
		}
		one = *fresh
		stage.have[key] = one
		stage.made.kept += one.stored
	}
	orientation := maths.Upright(direction)
	base := exit.Sub(direction.Scale(one.bounds.Max.Y))
	at := maths.NewSubspace(base,
		orientation, launcherUnit)
	return stage.add(one, kind, at, placementZero, false)
}
