package main

import (
	"context"
	"errors"
	"os"
	"runtime/debug"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/scene/frame"
	"github.com/janearc/slayer/scene/island"
	"github.com/janearc/slayer/terrain"
)

// sceneSession retains verified sources, their native height indices and
// bounded plant prototypes. View geometry and GPU inputs are still rebuilt;
// this is not persistent GPU acceleration-structure residency.
type sceneSession struct {
	layers     *island.Layers
	opened     []*terrain.Raster
	sphere     frame.Sphere
	ground     *island.Terrain
	prototypes prototypeStore
	binaryHash string
	build      *debug.BuildInfo
}

// openScene verifies sources and the executable identity, retaining source and
// prototype caches. Any partial opening is closed on failure.
func openScene(ctx context.Context, plan *plan) (_ *sceneSession, failure error) {
	session := &sceneSession{prototypes: prototypeStore{values: make(map[prototypeKey]prototype)}}
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

// Close attempts every source close and clears ownership so repeats are safe.
func (session *sceneSession) Close() (failure error) {
	for _, raster := range session.opened {
		failure = errors.Join(failure, raster.Close())
	}
	session.opened = nil
	return failure
}
