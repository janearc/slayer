package skeleton

import (
	"errors"
	"fmt"

	"github.com/janearc/slayer/maths"
)

// taper is how much thinner each bone of a ray is than the one before
// it: a phalanx tapers from its base to its tip. Chosen.
const taper = .78

var errRay = errors.New("skeleton: a ray needs a bone and a thickness")

// Ray adds one ray of a foot or a hand laid out bone by bone: a
// metapodial (metatarsal or metacarpal) and its phalanges, end to end
// from start along direction, lengths in metres from the base outward.
//
// The first is prefix followed by 1, under parent, thick in half
// thickness at its base; each later one hangs on the one before and
// tapers. Hand makes a fore flipper's; Ray is for a foot, or any ray
// Hand's layout does not fit.
func (skeleton *Skeleton) Ray(prefix, parent string, start,
	direction maths.Vec, lengths []float64, thick float64,
) ([]string, error) {
	if len(lengths) == 0 || !(thick > 0) {
		return nil, fmt.Errorf("%w: %s", errRay, prefix)
	}
	way := direction.Unit()
	names := make([]string, len(lengths))
	head := start
	for index, length := range lengths {
		names[index] = fmt.Sprintf("%s%d", prefix, index+1)
		tail := head.Add(way.Scale(length))
		err := skeleton.Add(names[index], parent, head, tail, thick,
			thick*taper)
		if err != nil {
			return nil, err
		}
		head, parent, thick = tail, names[index], thick*taper
	}
	return names, nil
}
