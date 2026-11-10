package main

import (
	"fmt"
	"github.com/janearc/slayer/manufacture"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
)

// putFireRecipe retains an immutable fire and builds its initial light samples.
func (stage *staging) putFireRecipe(recipe string, seed, index uint64,
	place maths.Vec, name string) (maths.Box, error) {
	row, ok := stage.recipes.Recipe(recipe)
	if !ok {
		return maths.Box{}, fmt.Errorf("fire needs a named recipe")
	}
	spec, ok := row.Fire()
	if !ok {
		return maths.Box{}, fmt.Errorf("%s is not a fire recipe", recipe)
	}
	seed = stir(stir(stage.given.seed, int(seed)), int(index))
	spec.Flame.Seed = seed
	if spec.Smoke != nil {
		spec.Smoke.Seed = seed
	}
	made, err := manufacture.Burn(spec)
	if err != nil {
		return maths.Box{}, err
	}
	at := maths.Subspace{Origin: place}
	volumes, _, err := made.Place(at, stage.given.exposure, 0)
	if err != nil {
		return maths.Box{}, err
	}
	fire := retainedFire{place: at, cells: spec.LightCells}
	fire.field, ok = volumes[0].Medium.(emittingField)
	if !ok {
		return maths.Box{}, fmt.Errorf("fire has no sampled emission")
	}
	fire.sparks, err = stage.prepareSparks(spec, at)
	if err != nil {
		return maths.Box{}, err
	}
	lamps, err := fire.lampsAt(stage.given.at)
	if err != nil {
		return maths.Box{}, err
	}
	particles := fire.sparks.thingsAt(stage.given.at)
	if stage.dynamicThings == nil {
		stage.dynamicThings = map[*thing.Thing]bool{}
		stage.dynamicVolumes = map[*thing.Volume]bool{}
		stage.dynamicLamps = map[int]bool{}
	}
	for _, volume := range volumes {
		pointer := stage.world.AddVolume(volume)
		stage.dynamicVolumes[pointer] = true
		fire.volumes = append(fire.volumes, pointer)
	}
	for _, lamp := range lamps {
		stage.dynamicLamps[len(stage.world.Lamps)] = true
		stage.world.Lamp(lamp)
	}
	for _, particle := range particles {
		stage.dynamicThings[stage.world.Add(particle)] = true
	}
	stage.fires = append(stage.fires, fire)
	stage.made.things++
	stage.made.burning++
	stage.made.lamps += len(lamps)
	stage.made.sparks += len(particles)
	if name != "" {
		stage.radii[name] = spec.FuelRadius().InMetres()
	}
	return made.Bounds(), nil
}
