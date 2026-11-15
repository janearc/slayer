package main

import (
	"context"
	"errors"
	"os"
	"runtime/debug"
	"time"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/scene/frame"
	"github.com/janearc/slayer/scene/island"
	"github.com/janearc/slayer/terrain"
)

// sceneSession retains verified sources, their native height indices, fixed
// mist floors and bounded plant prototypes. View geometry and GPU inputs are
// still rebuilt; this is not persistent GPU acceleration-structure residency.
type sceneSession struct {
	layers        *island.Layers
	opened        []*terrain.Raster
	sphere        frame.Sphere
	ground        *island.Terrain
	terrainMotion *island.Motion
	prototypes    prototypeStore
	floors        floorStore
	binaryHash    string
	build         *debug.BuildInfo
}

// openScene verifies sources, records the executable hash and opens bounded
// input caches. Any partial opening is closed on failure.
func openScene(ctx context.Context, plan *plan) (_ *sceneSession, failure error) {
	session := &sceneSession{prototypes: prototypeStore{values: make(map[prototypeKey]prototype)},
		floors: floorStore{values: make(map[floorKey]floorEntry)}}
	if plan.Site != nil {
		session.floors.limit = plan.Site.Budget.MistFloors
	}
	defer func() {
		if failure != nil {
			failure = errors.Join(failure, session.Close())
		}
	}()
	var err error
	session.layers, session.opened, err = plan.openSources(ctx)
	if err != nil {
		return nil, err
	}
	session.sphere, err = frame.NewSphere(measure.Metres(plan.Radius), plan.Anchor.geographic())
	if err != nil {
		return nil, err
	}
	session.ground, err = plan.terrain(session.sphere, session.layers)
	if err != nil {
		return nil, err
	}
	if controls := plan.Detail.Transition; controls != nil {
		session.terrainMotion, err = island.NewMotion(session.ground,
			time.Duration(controls.Seconds*float64(time.Second)), controls.Patches, controls.Bytes)
		if err != nil {
			return nil, err
		}
	}
	executable, err := os.Executable()
	if err != nil {
		return nil, err
	}
	session.binaryHash, err = fingerprint(ctx, executable)
	if err != nil {
		return nil, err
	}
	session.build, _ = debug.ReadBuildInfo()
	return session, nil
}

// Close attempts every source close and releases retained mist floors.
// Clearing ownership keeps repeated closes safe.
func (session *sceneSession) Close() (failure error) {
	for _, raster := range session.opened {
		failure = errors.Join(failure, raster.Close())
	}
	session.opened = nil
	session.floors.values = nil
	session.floors.bytes = 0
	if session.terrainMotion != nil {
		session.terrainMotion.Reset()
		session.terrainMotion = nil
	}
	return failure
}
