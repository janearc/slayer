package manufacture

import "github.com/janearc/slayer/maths"

// instrumentPlate closes a convex XZ outline with a finite thickness.
// The outline runs from north through east; its centre is inside it.
//
// dish is how far below its rim the plate's middle sits: nought for a
// flat plate, and more for a membrane held at its edges that sags, or
// is drawn, toward its middle. Its faces fan from its middle either way.
type instrumentPlate struct {
	outline   []maths.Vec
	thickness float64
	dish      float64
}

// skin closes both faces and the rim of the convex outline.
func (plate instrumentPlate) skin(cut) surface {
	var skin surface
	origin := maths.V(spaceZero, -plate.dish, spaceZero)
	lift := maths.Up.Scale(plate.thickness)
	crown := origin.Add(lift)
	for index, here := range plate.outline {
		next := plate.outline[(index+spaceOne)%len(plate.outline)]
		top := face{crown, next.Add(lift), here.Add(lift)}
		skin = append(skin, top,
			face{origin, here, next})
		skin = quad(skin, here, here.Add(lift), next.Add(lift), next)
	}
	return skin
}
