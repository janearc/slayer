package planet

import (
	"fmt"
	"math"
)

// rungs is how many steps of cooking each zone's column is made in,
// from fresh, no cooked haze, to full. the face reads between the two
// either side of what a point has cooked.
//
// the reading between is straight, and a haze's light is not straight in
// its depth, so the more rungs the closer the air between them comes to
// its own depth's light. its value is in picks.yaml.
var rungs = pickCount("ladder-rungs")

// Cooked is the body with its air cooking as it ages at the top under the
// light: the haze fully cooked air bears, the first hazed layer of a column,
// and the seconds over which air cooks, so air of an age bears that haze
// thinned to 1 - e^(-age/cooking) of it, in its top layer.
//
// the chromophores of a gas giant are thought to be haze so cooked, white
// when fresh, through cream and yellow to orange and brown as it ages; the
// x-rays cook it here. modelled: one haze thickening, not its chemistry.
func (body Body) Cooked(column []Layer, cooking float64) (Body, error) {
	for _, zone := range body.zones {
		if len(zone.column) == nothing {
			return body, fmt.Errorf("cooking needs a top layer in every zone")
		}
	}
	for _, layer := range column {
		if layer.haze.depth > nothing && cooking > nothing {
			body.cooked, body.cooking = layer.haze, cooking
			return body, nil
		}
	}
	return body, fmt.Errorf("cooking over %g s in a column with no haze",
		cooking)
}

// uppermost is defined as the place of a column's top layer: the first.
const uppermost = 0

// ladder is a zone's column cooked by each rung: its top layer bearing
// the cooked haze thinned to the rung's share, from none to all.
func (body Body) ladder(column []Layer) [][]Layer {
	made := make([][]Layer, rungs)
	for rung := range made {
		share := float64(rung) / float64(rungs-adjacent)
		cooked := append([]Layer(nil), column...)
		top := cooked[uppermost]
		top.haze = body.cooked
		top.haze.depth *= share
		cooked[uppermost] = top
		made[rung] = cooked
	}
	return made
}

// cookedBy is how far air of an age in seconds has cooked, from zero to one, as
// a rung of the ladder: the rung below it and how far toward the next. air
// with no age, or none that is a number, is fresh.
func (face *Face) cookedBy(age float64) (rung int, toward float64) {
	if face.cooking == nothing || !(age > nothing) {
		return nothing, nothing
	}
	along := (whole - math.Exp(-age/face.cooking)) *
		float64(rungs-adjacent)
	below := int(math.Floor(along))
	rung = min(below, rungs-double)
	return rung, along - float64(rung)
}
