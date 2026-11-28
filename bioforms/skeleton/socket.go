package skeleton

import (
	"fmt"

	"github.com/janearc/slayer/maths"
)

// BoneFrames is the named rigid part of a sampled pose. A surface can
// use the same sample to move its skin and carry objects on its bones.
type BoneFrames interface {
	Frame(name string) (Frame, bool)
}

// Socket is an attachment placement measured from a rest bone's head.
// Its offset and orientation use the figure's rest axes.
type Socket struct {
	bone string
	rest maths.Subspace
}

// NewSocket locates a socket on a named bone. The offset is measured from
// the bone's head. Zero scale represents unit scale.
func (skeleton *Skeleton) NewSocket(bone string, offset maths.Subspace) (Socket, error) {
	index, found := skeleton.Find(bone)
	if !found {
		return Socket{}, fmt.Errorf("%w: %s", errNoBone, bone)
	}
	offset.Origin = skeleton.Bone(index).Head().Add(offset.Origin)
	if !rigidSkinFrame(NewFrame(offset.Orientation, offset.Origin)) ||
		!skinNumber(offset.Scale) || offset.Scale < 0 {
		return Socket{}, fmt.Errorf("skeleton: invalid attachment placement on %q", bone)
	}
	return Socket{bone: bone, rest: offset}, nil
}

// Place carries the socket with its bone in the given pose. Use frames
// sampled from the same rest skeleton that authored the socket.
func (socket Socket) Place(frames BoneFrames) (maths.Subspace, error) {
	frame, found := frames.Frame(socket.bone)
	if !found {
		return maths.Subspace{}, fmt.Errorf("%w: %s", errNoBone, socket.bone)
	}
	placement := frame.Place(socket.rest)
	if !skinPointFinite(placement.Origin) {
		return maths.Subspace{}, fmt.Errorf("skeleton: attachment on %q exceeds the placement range", socket.bone)
	}
	return placement, nil
}
