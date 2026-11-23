package session

import (
	"fmt"
	"time"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
	"github.com/janearc/slayer/set"
)

// retainedDeformation holds the construction recipe and placement, never a
// mutable frame mesh. The builder must return newly owned frame geometry.
type retainedDeformation struct {
	build        func(time.Duration) ([]thing.Thing, error)
	place        maths.Subspace
	initialFaces int
}

// thingsAt is the deformation's things at a time: built in its own frame,
// then each held still at its place at that time within the deformation's
// placement.
func (one retainedDeformation) thingsAt(at time.Duration) ([]thing.Thing, error) {
	local, err := one.build(at)
	if err != nil {
		return nil, err
	}
	out := make([]thing.Thing, len(local))
	for i, part := range local {
		space := maths.Local()
		if part.Motion != nil {
			space = part.Motion.At(at)
		}
		part.Motion = thing.Still(space.Within(one.place))
		out[i] = part
	}
	return out, nil
}

// deformingRefusal makes unsupported rendering modes fail before rendering.
func deformingRefusal(given asked, shuttered bool) error {
	switch {
	case given.onGPU:
		return fmt.Errorf("deforming geometry currently requires the CPU renderer")
	case shuttered:
		return fmt.Errorf("deforming geometry currently refuses a job shutter")
	case given.glass != nil:
		return fmt.Errorf("deforming geometry currently refuses a lens rig shutter")
	case given.closeGlow:
		return fmt.Errorf("deforming geometry currently refuses the close-glow mask")
	case given.composite != nil || given.cachedLighting || given.fastLighting:
		return fmt.Errorf("deforming geometry currently refuses composite or cached lighting")
	}
	return nil
}

// deformingExclusionRefusal avoids using a single-pose radius as a promise
// of clearance over an animal's entire deformation cycle.
func deformingExclusionRefusal(read set.Set) error {
	for _, stand := range read.Stands() {
		if name, _ := stand.Exclude(); name != "" {
			return fmt.Errorf("deforming geometry currently refuses stand exclusions")
		}
	}
	for _, ecosystem := range read.Ecosystems() {
		if name, _ := ecosystem.Exclude(); name != "" {
			return fmt.Errorf("deforming geometry currently refuses ecosystem exclusions")
		}
	}
	return nil
}
