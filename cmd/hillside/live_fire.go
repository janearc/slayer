package main

import (
	"fmt"
	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/emit"
	"github.com/janearc/slayer/flame"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/medium"
	"github.com/janearc/slayer/scene/thing"
	"slices"
	"time"
)

type emittingField interface {
	Tick(time.Duration) medium.Medium
	Sources(int) []flame.Source
}

type retainedFire struct {
	volumes []*thing.Volume
	field   emittingField
	place   maths.Subspace
	cells   int
	sparks  *retainedSparks
}

type liveScene struct {
	base        thing.World
	fires       []retainedFire
	fronts      []retainedFront
	beds        []retainedBed
	made        tally
	unsupported string
}

// lampsAt samples only changing source strength; it retains the thermal LUT.
func (fire retainedFire) lampsAt(at time.Duration) ([]thing.Lamp, error) {
	field, ok := fire.field.Tick(at).(emittingField)
	if !ok {
		return nil, fmt.Errorf("sequence: unsupported timed fire")
	}
	var lamps []thing.Lamp
	for _, source := range field.Sources(fire.cells) {
		source = source.Placed(fire.place)
		point := emit.Lit{At: source.At, Strength: source.Strength}
		lamp, err := emit.NewVolume(point, measure.Metres(source.Across/2))
		if err != nil {
			return nil, err
		}
		lamps = append(lamps, lamp)
	}
	return lamps, nil
}

// frameAt creates frame containers while retaining immutable geometry/fields.
func (live *liveScene) frameAt(at time.Duration) (*thing.World, tally, error) {
	if live.unsupported != "" {
		return nil, tally{}, fmt.Errorf("sequence: %s", live.unsupported)
	}
	world := live.base
	world.Things = slices.Clone(live.base.Things)
	world.Volumes = slices.Clone(live.base.Volumes)
	world.Fields = slices.Clone(live.base.Fields)
	world.Lamps = slices.Clone(live.base.Lamps)
	made := live.made
	made.sparks = 0
	for _, fire := range live.fires {
		world.Volumes = append(world.Volumes, fire.volumes...)
		lamps, err := fire.lampsAt(at)
		if err != nil {
			return nil, tally{}, err
		}
		world.Lamps = append(world.Lamps, lamps...)
		particles := fire.sparks.thingsAt(at)
		for _, particle := range particles {
			world.Add(particle)
		}
		made.sparks += len(particles)
	}
	for i := range live.fronts {
		if err := live.fronts[i].addAt(&world, &made, at); err != nil {
			return nil, tally{}, err
		}
	}
	for i := range live.beds {
		if err := live.beds[i].addAt(&world, &made, at); err != nil {
			return nil, tally{}, err
		}
	}
	made.lamps = len(world.Lamps)
	return &world, made, nil
}

// retain removes initial dynamic samples from the reusable scene base.
func (stage *staging) retain() *liveScene {
	live := &liveScene{base: *stage.world, fires: slices.Clone(stage.fires), fronts: stage.fronts, beds: stage.beds, made: stage.made}
	live.base.Things = nil
	for _, placed := range stage.world.Things {
		if !stage.dynamicThings[placed] {
			live.base.Things = append(live.base.Things, placed)
		}
	}
	live.base.Volumes = nil
	for _, volume := range stage.world.Volumes {
		if !stage.dynamicVolumes[volume] {
			live.base.Volumes = append(live.base.Volumes, volume)
		}
	}
	live.base.Lamps = nil
	for i, lamp := range stage.world.Lamps {
		if !stage.dynamicLamps[i] {
			live.base.Lamps = append(live.base.Lamps, lamp)
		}
	}
	live.base.Fields = slices.Clone(stage.world.Fields)
	if stage.unretained != "" {
		live.unsupported = stage.unretained
	}
	if stage.made.burning != len(stage.fires) {
		live.unsupported = "legacy alight has frame-time lamps; use fire recipes"
	}
	return live
}
