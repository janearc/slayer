package session

import (
	"context"
	"fmt"
	"slices"

	"github.com/janearc/slayer/job"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
	"github.com/janearc/slayer/set"
)

// WorkingDetail selects a named authored placement or stand for a working view.
type WorkingDetail struct {
	Collection string
	Name       string
}

// OpenWorkingDetail stages one named terrain placement or stand for inspection.
func OpenWorkingDetail(ctx context.Context, wanted job.Job, detail WorkingDetail) (*Session, error) {
	if wanted.Scene().Form() != sceneSet || !wanted.Clock().Still() {
		return nil, fmt.Errorf("scene detail requires a still job or a set")
	}
	return opened(ctx, wanted, Stamp{}, &detail)
}

// stagedDetail preserves terrain coordinates and authored recipe indices.
// Referenced placements need anchors, but only an exclusion needs manufactured
// geometry: the stand's clearance includes that object's physical radius.
func stagedDetail(read set.Set, given asked) (*thing.World, tally, asked, error) {
	detail := *given.detail
	if _, ground := read.Ground(); !ground || read.InSpace() {
		return nil, tally{}, given, fmt.Errorf("scene detail currently requires a terrain set")
	}
	things, stands := read.Things(), read.Stands()
	thingIndex, standIndex := -1, -1
	switch detail.Collection {
	case "scene.set.things":
		thingIndex = slices.IndexFunc(things, func(placed set.Thing) bool { return placed.Name() == detail.Name })
	case "scene.set.stands":
		standIndex = slices.IndexFunc(stands, func(stand set.Stand) bool { return stand.Name() == detail.Name })
	}
	if thingIndex < 0 && standIndex < 0 {
		return nil, tally{}, given, fmt.Errorf("no terrain placement or stand named %q in %s", detail.Name, detail.Collection)
	}
	stage := &staging{given: given, world: thing.NewWorld(), have: stock{},
		places: map[string]maths.Vec{}, radii: map[string]float64{}, recipes: read}
	if err := stage.stepped("terrain", stage.terrain, read); err != nil {
		return nil, tally{}, given, err
	}
	if elevation, floods := stillWater(read); floods {
		if err := stage.flooded(elevation); err != nil {
			return nil, tally{}, given, err
		}
	}
	stage.detailAnchors = make(map[string]set.Thing, len(things))
	for _, placed := range things {
		stage.detailAnchors[placed.Name()] = placed
	}
	if err := stage.makeDetail(things, stands, thingIndex, standIndex); err != nil {
		return nil, tally{}, given, err
	}
	if len(stage.deformations) > 0 {
		return nil, tally{}, given, fmt.Errorf("scene detail cannot inspect deforming geometry")
	}
	if stage.given.camera == "" {
		if cameras := read.Cameras(); len(cameras) > 0 {
			stage.given.camera = cameras[0]
		}
	}
	eye, err := stage.eye(read)
	if err != nil {
		return nil, tally{}, given, fmt.Errorf("scene detail camera: %w", err)
	}
	stage.given.eye = &eye
	if film, _ := read.Film(); stage.given.frame != nil {
		rig, _ := film.Rig()
		if err := stage.focused(rig, eye); err != nil {
			return nil, tally{}, given, fmt.Errorf("scene detail focus: %w", err)
		}
	}
	stage.given.authoredOrigins = stage.places
	return stage.world, stage.made, stage.given, nil
}

func (stage *staging) makeDetail(things []set.Thing, stands []set.Stand, thingIndex, standIndex int) error {
	if err := stage.given.steps.step("detail", 0, 1); err != nil {
		return err
	}
	if thingIndex >= 0 {
		return stage.stageThing(things[thingIndex], uint64(thingIndex))
	}
	stand := stands[standIndex]
	if name, _, _, _, _, has := stand.Within(); has {
		if _, found := stage.where(name); !found {
			return fmt.Errorf("scene detail cannot place stand anchor %q on the terrain", name)
		}
	}
	if name, _ := stand.Exclude(); name != "" {
		index := slices.IndexFunc(things, func(placed set.Thing) bool { return placed.Name() == name })
		if index < 0 {
			return fmt.Errorf("scene detail cannot find exclusion placement %q", name)
		}
		if err := stage.stageThing(things[index], uint64(index)); err != nil {
			return fmt.Errorf("scene detail exclusion: %w", err)
		}
		if _, found := stage.radii[name]; !found {
			return fmt.Errorf("scene detail cannot establish radius of exclusion placement %q", name)
		}
	}
	start := len(stage.world.Things)
	if err := stage.scatter(stand); err != nil {
		return fmt.Errorf("%s: %w", stand.Name(), err)
	}
	stage.recordAuthor(start, "scene.set.stands", stand.Name())
	return nil
}
