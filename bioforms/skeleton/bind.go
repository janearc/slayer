package skeleton

import (
	"errors"
	"fmt"

	"github.com/janearc/slayer/maths"
)

var errReach = errors.New("skeleton: a holder's reach is not positive")

// Holders is a holder for each named bone, for Nearness: its middle line
// its form's middle where forms gives one, else its rest line from head
// to tail, and each reaching reach metres. A name not here, or a reach
// that is not positive, is refused.
func (skeleton *Skeleton) Holders(names []string,
	forms map[string]BoneForm, reach float64,
) ([]Holder, error) {
	if !(reach > 0) {
		return nil, fmt.Errorf("%w: %v", errReach, reach)
	}
	holders := make([]Holder, 0, len(names))
	for _, name := range names {
		at, found := skeleton.Find(name)
		if !found {
			return nil, fmt.Errorf("%w: %s", errNoBone, name)
		}
		bone := skeleton.bones[at]
		line := []maths.Vec{bone.head, bone.tail}
		if form, formed := forms[name]; formed {
			line = form.Middle()
		}
		holders = append(holders, NewHolder(at, line, reach))
	}
	return holders, nil
}
