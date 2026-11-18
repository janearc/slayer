package hillside

import (
	"fmt"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
	"github.com/janearc/slayer/set"
)

// partSpin validates the complete selection before changing any staged motion.
// The retained prototype and its shared shape/material pointers are untouched.
func (stage *staging) partSpin(start int, each set.Thing) error {
	spin, has := each.PartSpin()
	if !has {
		return nil
	}
	pivot, axis := spin.PivotMetres(), spin.Axis()
	var selected []*thing.Thing
	var motions []thing.Motion
	for _, part := range stage.world.Things[start:] {
		if part.Material == nil || part.Material.Name != spin.Material() {
			continue
		}
		motion, err := thing.NewAxisSpin(part.Motion, maths.V(pivot[0], pivot[1], pivot[2]), maths.V(axis[0], axis[1], axis[2]), spin.RPM(), spin.PhaseRadians())
		if err != nil {
			return fmt.Errorf("part-spin %s: %w", each.Name(), err)
		}
		selected = append(selected, part)
		motions = append(motions, motion)
	}
	if len(selected) == 0 {
		return fmt.Errorf("part-spin %s: native material %q matched no newly placed parts", each.Name(), spin.Material())
	}
	for index, part := range selected {
		part.Motion = motions[index]
	}
	return nil
}
