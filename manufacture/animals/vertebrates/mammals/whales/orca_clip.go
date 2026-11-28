package whales

import (
	"github.com/janearc/slayer/bioforms/skeleton"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
)

// RestSkeleton gives a copy of the rest bones used by the profile's skin.
func (p Profile) RestSkeleton() (*skeleton.Skeleton, error) {
	skin, err := p.model.orcaBinding()
	if err != nil {
		return nil, err
	}
	return skin.RestSkeleton(), nil
}

// OrcaPose shares the named frames and skin deformed by those frames.
type OrcaPose struct {
	profile Profile
	skin    *skeleton.SkinPose
}

// Pose samples the selected profile at authored joint turns.
func (p Profile) Pose(turns skeleton.Pose) (*OrcaPose, error) {
	skin, err := p.model.orcaBinding()
	if err != nil {
		return nil, err
	}
	posed, err := skin.Pose(turns)
	if err != nil {
		return nil, err
	}
	return &OrcaPose{profile: p, skin: posed}, nil
}

// Frame returns the motion carrying the named rest bone.
func (pose *OrcaPose) Frame(name string) (skeleton.Frame, bool) { return pose.skin.Frame(name) }

// Surface builds the orca's exterior with its original paint chart.
func (pose *OrcaPose) Surface() ([]thing.Thing, error) {
	p := pose.profile
	return p.model.orcaCarried(pose.skin, maths.Local(), p.Palette, p.Satin)
}
