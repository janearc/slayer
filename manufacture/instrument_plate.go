package manufacture

import "github.com/janearc/slayer/maths"

// instrumentPlate closes a convex XZ outline with a finite thickness.
// The outline runs from north through east; its centre is inside it.
type instrumentPlate struct {
	outline   []maths.Vec
	thickness float64
}

// skin closes both faces and the rim of the convex outline.
func (plate instrumentPlate) skin(cut) surface {
	var skin surface
	lift := maths.Up.Scale(plate.thickness)
	origin := maths.V(spaceZero, spaceZero, spaceZero)
	for index, here := range plate.outline {
		next := plate.outline[(index+spaceOne)%len(plate.outline)]
		top := face{lift, next.Add(lift), here.Add(lift)}
		skin = append(skin, top,
			face{origin, here, next})
		skin = quad(skin, here, here.Add(lift), next.Add(lift), next)
	}
	return skin
}
