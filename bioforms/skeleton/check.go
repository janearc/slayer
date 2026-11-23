package skeleton

import (
	"errors"
	"fmt"
	"slices"
	"strings"
)

// ErrNoJoint is a pose that turns a joint no bone of the skeleton has:
// Frames would pass over the turn without a word.
var ErrNoJoint = errors.New("skeleton: a pose turns a joint with no bone")

// Check is nil when every joint pose turns is the head of a bone of the
// skeleton, and otherwise an ErrNoJoint that names the others.
func (skeleton *Skeleton) Check(pose Pose) error {
	var missing []string
	for name := range pose.turns {
		if _, ok := skeleton.byName[name]; !ok {
			missing = append(missing, name)
		}
	}
	if len(missing) == 0 {
		return nil
	}
	slices.Sort(missing)
	return fmt.Errorf("%w: %s", ErrNoJoint, strings.Join(missing, ", "))
}
