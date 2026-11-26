package manufacture

import (
	"fmt"
	"time"

	kelp "github.com/janearc/slayer/manufacture/algae/kelp"
	"github.com/janearc/slayer/scene/shape"
)

const defaultKelpWater = "moderate"
const kelpFloat slot = 2

type kelpRow struct{ specimen *kelp.Specimen }

func (row kelpRow) made(from seed) (good, error) { return row.madeAt(from, 0) }
func (row kelpRow) madeAtTime(from seed, _ string, at time.Duration) (good, error) {
	return row.madeAt(from, at.Seconds())
}
func (row kelpRow) madeAt(from seed, clock float64) (good, error) {
	if row.specimen == nil {
		return good{}, fmt.Errorf("manufacture: no kelp specimen")
	}
	made, err := row.specimen.Grow(from, clock)
	if err != nil {
		return good{}, err
	}
	dressed := looks{
		body:      plain("kelp stipe", made.StipeColour[0], made.StipeColour[1], made.StipeColour[2]),
		leaf:      plain("kelp blade", made.BladeColour[0], made.BladeColour[1], made.BladeColour[2]),
		kelpFloat: plain("kelp bladder", made.FloatColour[0], made.FloatColour[1], made.FloatColour[2]),
	}
	grown := good{dressed: dressed, groupParts: map[slot]int{}}
	return grown.withMeshGroups(map[slot][]shape.MeshMember{body: made.Stipes, leaf: made.Blades, kelpFloat: made.Bladders})
}
