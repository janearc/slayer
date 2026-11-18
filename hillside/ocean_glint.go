package hillside

import (
	"fmt"

	"github.com/janearc/slayer/set"
)

// refuseGlintOnGPU keeps a cox-munk ocean glint to the CPU. The GPU's
// materials carry Satin's GGX lobe and no Beckmann lobe, so it would
// draw the sea with no glint at all and say nothing.
func refuseGlintOnGPU(read set.Set, given asked) error {
	world, has := read.World()
	if !has || !given.onGPU {
		return nil
	}
	surface, has := world.OceanSurface()
	if has && surface.Glint().Model() == set.GlintCoxMunk {
		return fmt.Errorf("%s: an ocean glint of model cox-munk is "+
			"CPU-only; select renderer: cpu", given.set)
	}
	return nil
}
