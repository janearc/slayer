package main

import (
	"fmt"
	"math"
	"time"

	"github.com/janearc/slayer/embers"
	"github.com/janearc/slayer/firefront"
	"github.com/janearc/slayer/flame"
	"github.com/janearc/slayer/gpu"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/plume"
	"github.com/janearc/slayer/scene/thing"
	"github.com/janearc/slayer/set"
)

type scheduledGroup struct {
	fire       retainedFire
	start, end time.Duration
}
type scheduledSparks struct {
	sparks          *retainedSparks
	start, duration time.Duration
}
type retainedFront struct {
	groups                  []scheduledGroup
	sparks                  []scheduledSparks
	smoke                   *thing.Volume
	smokeSpec               *plume.Fire
	fade                    time.Duration
	peakFuel                int
	plan                    firefront.Schedule
	peakVolumes, peakSparks int
}

// prepareFronts retains checked fuel schedules and budgets their GPU work.
func (stage *staging) prepareFronts(read set.Set) error {
	volumes := len(stage.world.Volumes)
	particles := 0
	for _, fire := range stage.fires {
		particles += fire.sparks.budget()
	}
	if particles > 8192 {
		return fmt.Errorf("scene exceeds the budget of 8192 sparks")
	}
	for _, request := range read.FireFronts() {
		origin, has := stage.places[request.From()]
		if !has {
			return fmt.Errorf("front %s has no placed origin", request.Name())
		}
		var targets []firefront.Target
		for i, tree := range stage.fuels[request.Stand()] {
			at := tree.at.Add(maths.YAxis.Scale(request.IgnitionHeight().InMetres()))
			targets = append(targets, firefront.Target{At: at, Seed: stir(stage.given.seed, i+1)})
		}
		plan, err := firefront.New(request.Motion(origin), targets)
		if err != nil {
			return fmt.Errorf("front %s: %w", request.Name(), err)
		}
		row, _ := read.Recipe(request.Recipe())
		spec, _ := row.Fire()
		front := retainedFront{plan: plan, peakVolumes: plan.PeakGroups(), fade: request.FadeTime(), peakFuel: plan.PeakTargets(0)}
		if spec.Smoke != nil {
			columnSpec := *spec.Smoke
			columnSpec.Base = 2 * request.SmokeRadius().InMetres()
			column, err := plume.New(columnSpec)
			if err != nil {
				return err
			}
			front.smokeSpec = &columnSpec
			front.smoke = &thing.Volume{Medium: column, Material: &thing.Material{Name: "front smoke", Colour: maths.Grey(spec.Albedo)}}
			front.peakVolumes++
		}
		if spec.Sparks != nil {
			livePerSource := int(math.Ceil(spec.Sparks.Rate*spec.Sparks.Life.Seconds())) + 2
			front.peakSparks = livePerSource * plan.PeakTargets(spec.Sparks.Life)
		}
		volumes += front.peakVolumes
		particles += front.peakSparks
		if volumes > gpu.VolumeLimit() {
			return fmt.Errorf("front %s needs up to %d volumes; GPU limit is %d", request.Name(), volumes, gpu.VolumeLimit())
		}
		if particles > 8192 {
			return fmt.Errorf("front %s exceeds the budget of "+
				"8192 sparks", request.Name())
		}
		// Share the thermal palette and mesh among this front's
		// firebrands.
		prototype, err := stage.prepareSparks(spec, maths.Subspace{})
		if err != nil {
			return err
		}
		for _, events := range plan.Groups() {
			anchor := events[0].At
			ignitions := make([]flame.Ignition, len(events))
			start, end := events[0].Start, events[0].Start+events[0].Duration
			for i, event := range events {
				fire := spec.Flame
				fire.Seed = event.Seed
				ignitions[i] = flame.Ignition{At: event.At.Sub(anchor), Fire: fire, Start: event.Start, Duration: event.Duration, Fade: request.FadeTime()}
				start = min(start, event.Start)
				end = max(end, event.Start+event.Duration)
				if prototype != nil {
					sparks := *prototype
					sourceSpec := *spec.Sparks
					sourceSpec.Seed = event.Seed
					from, speed := stage.recipes.Wind()
					sourceSpec.Air = sourceSpec.Air.Add(skyward(from, 0).Neg().Scale(speed))
					source, err := embers.New(sourceSpec)
					if err != nil {
						return err
					}
					sparks.source = source
					sparks.place = maths.Subspace{Origin: event.At}
					front.sparks = append(front.sparks, scheduledSparks{&sparks, event.Start, event.Duration})
				}
			}
			field, err := flame.NewField(ignitions, stage.given.exposure)
			if err != nil {
				return err
			}
			pose := maths.Subspace{Origin: anchor}
			volume := &thing.Volume{Medium: field, Material: &thing.Material{Name: "front flame", Shine: field.Shine()}, Motion: thing.Still(pose)}
			front.groups = append(front.groups, scheduledGroup{fire: retainedFire{field: field, place: pose, cells: spec.LightCells, volumes: []*thing.Volume{volume}}, start: start, end: end})
		}
		stage.fronts = append(stage.fronts, front)
	}
	return nil
}

// addAt samples the same global clock for occupied fuel, lamp quadrature and
// firebrands. The shared smoke plume follows the active targets' centroid;
// this is prescribed aggregate smoke, without a transported burn history.
func (front *retainedFront) addAt(world *thing.World, made *tally, at time.Duration) error {
	active := front.plan.Active(at)
	for _, group := range front.groups {
		if at < group.start || at >= group.end {
			continue
		}
		field := group.fire.field.(*flame.Field).Tick(at).(*flame.Field)
		if len(field.Pieces()) == 0 {
			continue
		}
		world.Volumes = append(world.Volumes, group.fire.volumes...)
		lamps, err := group.fire.lampsAt(at)
		if err != nil {
			return err
		}
		world.Lamps = append(world.Lamps, lamps...)
	}
	if len(active) > 0 && front.smoke != nil {
		centre := maths.Vec{}
		weight := 0.0
		for _, event := range active {
			share := 1.0
			if front.fade > 0 {
				x := math.Min(1, math.Min(float64(event.Age), float64(event.Duration-event.Age))/float64(front.fade))
				share = x * x * (3 - 2*x)
			}
			centre = centre.Add(event.At.Scale(share))
			weight += share
		}
		if weight > 0 {
			centre = centre.Scale(1 / weight)
			spec := *front.smokeSpec
			spec.Yield *= math.Min(1, weight/float64(front.peakFuel))
			column, err := plume.New(spec)
			if err != nil {
				return err
			}
			smoke := *front.smoke
			smoke.Medium = column
			smoke.Motion = thing.Still(maths.Subspace{Origin: centre})
			world.Volumes = append(world.Volumes, &smoke)
		}
	}

	for _, event := range front.sparks {
		age := at - event.start
		if at < event.start || age >= event.duration+event.sparks.life {
			continue
		}
		for _, particle := range event.sparks.thingsIn(age, event.duration) {
			world.Add(particle)
			made.sparks++
		}
	}
	made.burning += len(active)
	made.lamps = len(world.Lamps)
	return nil
}
