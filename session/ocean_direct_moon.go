package session

import (
	"fmt"

	"github.com/janearc/slayer/emit"
	"github.com/janearc/slayer/scene/thing"
	"github.com/janearc/slayer/set"
)

// tagOceanDirectMoon retains the exact modeled emitter, never a lamp chosen
// by its brightness or an incidental matching direction.
func (stage *staging) tagOceanDirectMoon(read set.Set) error {
	world, has := read.World()
	if !has {
		return nil
	}
	surface, has := world.OceanSurface()
	if !has || surface.DirectMoonMode() == "" {
		return nil
	}
	if read.Light() != "moon" {
		return fmt.Errorf("%s: ocean direct-moon requires light: moon", stage.given.set)
	}
	if err := stage.checkLunarSource(); err != nil {
		return err
	}
	stage.given.oceanSunTag = &oceanSunTag{
		sun: stage.lunarSource.index, fill: -1,
		route: thing.OceanSunRouteMoon,
	}
	return nil
}

// checkLunarSource refuses a moon route whose lunar lamp is not staged.
func (stage *staging) checkLunarSource() error {
	source := stage.lunarSource
	if source == nil || source.index < 0 || source.index >= len(stage.world.Lamps) {
		return fmt.Errorf("%s: ocean direct-moon needs its staged lunar source", stage.given.set)
	}
	actual, ok := stage.world.Lamps[source.index].(emit.Distant)
	if !ok || actual != source.lamp {
		return fmt.Errorf("%s: ocean direct-moon source was replaced", stage.given.set)
	}
	return nil
}
