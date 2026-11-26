package pinnipeds

import (
	"fmt"
	"github.com/janearc/slayer/scene/thing"
)

// Walrus retains the checked anatomy and representation of one authored profile.
type Walrus struct {
	rig            *walrusRig
	pose           walrusPose
	representation string
}

// NewWalrus checks a profile, pose and representation before it is used.
func NewWalrus(profile Profile, pose, representation string) (*Walrus, error) {
	if profile.Form != "walrus" {
		return nil, fmt.Errorf("manufacture: expected a walrus profile")
	}
	switch representation {
	case "skin":
	case "bones":
		if pose != "" {
			return nil, fmt.Errorf("manufacture: pinniped skeleton is at rest")
		}
	case "volumes":
		if pose != "" && pose != "lying" {
			return nil, fmt.Errorf("manufacture: volumes require a resting walrus")
		}
	default:
		return nil, fmt.Errorf("manufacture: unknown pinniped representation %q", representation)
	}
	profile = profile.snapshot()
	posture := walrusLying
	switch pose {
	case "", "lying":
	case "hauled-out":
		posture = walrusHauledOut
	case "swimming":
		posture = walrusSwimming
	default:
		return nil, fmt.Errorf("manufacture: unsupported %s pose %q", profile.Kind, pose)
	}
	rig, err := newWalrusRig(profile)
	if err != nil {
		return nil, err
	}
	row := &Walrus{rig: rig, pose: posture, representation: representation}
	if representation == "bones" {
		return row, nil
	}
	outline, err := rig.outline()
	if err != nil {
		return nil, err
	}
	if representation == "volumes" {
		if err := rig.checkVolumeGrid(outline); err != nil {
			return nil, err
		}
		return row, nil
	}
	const meshingHalfWidth = 1.9
	if outline.first < -meshingHalfWidth+walrusOutlineRim || outline.last > meshingHalfWidth-walrusOutlineRim {
		return nil, fmt.Errorf("manufacture: walrus outline exceeds its field grid")
	}
	if _, err := walrusPosedFrames(rig.body, walrusPoseOf(rig.body, posture)); err != nil {
		return nil, err
	}
	if err := rig.checkFieldGrid(outline, posture); err != nil {
		return nil, err
	}
	return row, nil
}
func (row *Walrus) Build() ([]thing.Thing, error) {
	switch row.representation {
	case "bones":
		return walrusSkeletonThings(row.rig.body)
	case "volumes":
		return row.rig.volumes()
	default:
		return row.rig.walrusAtPose(row.pose, 0, 0)
	}
}
