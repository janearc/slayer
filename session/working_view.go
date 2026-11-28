package session

import (
	"fmt"

	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/session/placement"
)

// WorkingView is the inspected renderer frame's finite surface mesh and camera.
// It is an aggregate without writable object identities. Its surface hitters
// may reference staged shapes, so the owning Session must remain open.
type WorkingView struct {
	World         placement.StagedWorld
	Eye           maths.Subspace
	Lens          film.Pinhole
	Width, Height int
	// RigProxy is a pinhole view of a physical rig without blur or exposure.
	RigProxy bool
}

// SharedWorkingView keeps geometry shared alongside the staged camera.
type SharedWorkingView struct {
	World           placement.StagedInstances
	Origins         map[string]maths.Vec
	Lights          []placement.DisplayLight
	VolumeBounds    []placement.DisplayVolumeBound
	GiantBoundaries []placement.DisplayGiantBoundary
	SkyGradient     *placement.DisplaySkyGradient
	Eye             maths.Subspace
	Lens            film.Pinhole
	Width, Height   int
	// RigProxy is a pinhole view of a physical rig without blur or exposure.
	RigProxy bool
}

// WorkingInstances inspects one staged frame without flattening repeated
// geometry. The returned meshes retain native hit targets until Close.
func (session *Session) WorkingInstances(index int) (placement.StagedInstances, error) {
	world, err := session.inspected(index)
	if err != nil {
		return placement.StagedInstances{}, err
	}
	return placement.WorkingInstances(world, session.times[index])
}

// WorkingViewShared keeps the inspected frame's shared geometry and camera.
func (session *Session) WorkingViewShared(index int) (SharedWorkingView, error) {
	world, err := session.inspected(index)
	if err != nil {
		return SharedWorkingView{}, err
	}
	eye, lens, proxy, err := session.workingEye(index)
	if err != nil {
		return SharedWorkingView{}, err
	}
	staged, err := placement.WorkingInstancesFrom(world, session.times[index], eye.Origin)
	if err != nil {
		return SharedWorkingView{}, err
	}
	width, height := session.Size()
	origins := make(map[string]maths.Vec, len(session.given.authoredOrigins))
	for name, origin := range session.given.authoredOrigins {
		origins["scene.set.things/"+name] = origin
	}
	return SharedWorkingView{
		World: staged, Origins: origins, Lights: placement.WorkingLights(world),
		VolumeBounds:    placement.WorkingVolumeBounds(world, session.times[index]),
		GiantBoundaries: placement.WorkingGiantBoundaries(world, session.times[index]),
		SkyGradient:     placement.WorkingSkyGradient(world.Sky),
		Eye:             eye, Lens: lens, Width: width, Height: height, RigProxy: proxy,
	}, nil
}

// WorkingView inspects one staged frame without rendering it.
func (session *Session) WorkingView(index int) (WorkingView, error) {
	world, err := session.inspected(index)
	if err != nil {
		return WorkingView{}, err
	}
	eye, lens, proxy, err := session.workingEye(index)
	if err != nil {
		return WorkingView{}, err
	}
	staged, err := placement.WorkingWorld(world, session.times[index])
	if err != nil {
		return WorkingView{}, err
	}
	width, height := session.Size()
	return WorkingView{World: staged, Eye: eye, Lens: lens, Width: width, Height: height, RigProxy: proxy}, nil
}

// workingEye resolves the staged frame's supported camera and pinhole lens.
func (session *Session) workingEye(index int) (maths.Subspace, film.Pinhole, bool, error) {
	source, ok := session.source.(eyeInspector)
	if !ok {
		return maths.Subspace{}, film.Pinhole{}, false, fmt.Errorf("working view: %s", unseen)
	}
	given, err := session.asking(Ask{})
	if err != nil {
		return maths.Subspace{}, film.Pinhole{}, false, err
	}
	given, err = source.eyed(session.times[index], given)
	if err != nil {
		return maths.Subspace{}, film.Pinhole{}, false, err
	}
	if given.eye == nil {
		return maths.Subspace{}, film.Pinhole{}, false, fmt.Errorf("working view: %s", unseen)
	}
	wide, proxy := given.wide*maths.Degree, false
	if given.frame != nil {
		glass, err := given.refocused(*given.eye)
		if err != nil {
			return maths.Subspace{}, film.Pinhole{}, false, err
		}
		wide, proxy = glass.Wide(), true
	} else if given.optics.aperture > nothing {
		return maths.Subspace{}, film.Pinhole{}, false, fmt.Errorf("working view: physical lenses without a rig are not supported yet")
	}
	lens, err := film.PinholeOf(wide)
	if err != nil {
		return maths.Subspace{}, film.Pinhole{}, false, err
	}
	return *given.eye, *lens, proxy, nil
}
