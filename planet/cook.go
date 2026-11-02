package planet

import (
	"fmt"
	"math"
)

// rungs is how many steps of cooking each zone's column is made in,
// from fresh, no cooked haze, to full: picked. the face reads between
// the two either side of what a point has cooked.
//
// the reading between is straight, and a haze's light is not straight in
// its depth, so the more rungs the closer the air between them comes to
// its own depth's light. sixteen is picked.
const rungs = 16

// Cooked is the body with its air cooking as it ages at the top under the
// light: the haze fully cooked air bears, the first hazed layer of a column,
// and the seconds over which air cooks, so air of an age bears that haze
// thinned to 1 - e^(-age/cooking) of it, in its top layer.
//
// on jupiter the chromophores are thought to be haze so cooked, white when
// fresh, through cream and yellow to orange and brown as it ages; the
// x-rays cook it here. modelled: one haze thickening, not its chemistry.
func (body Body) Cooked(column []Layer, cooking float64) (Body, error) {
	for _, layer := range column {
		if layer.haze.depth > 0 && cooking > 0 {
			body.cooked, body.cooking = layer.haze, cooking
			return body, nil
		}
	}
	return body, fmt.Errorf("cooking over %g s in a column with no haze",
		cooking)
}

// ladder is a zone's column cooked by each rung: its top layer bearing
// the cooked haze thinned to the rung's share, from none to all.
func (body Body) ladder(column []Layer) [][]Layer {
	made := make([][]Layer, rungs)
	for rung := range made {
		share := float64(rung) / (rungs - 1)
		cooked := append([]Layer(nil), column...)
		top := cooked[0]
		top.haze = body.cooked
		top.haze.depth *= share
		cooked[0] = top
		made[rung] = cooked
	}
	return made
}

// cookedBy is how far air of an age in seconds has cooked, from zero to one, as
// a rung of the ladder: the rung below it and how far toward the next.
func (face *Face) cookedBy(age float64) (rung int, toward float64) {
	if face.cooking == 0 {
		return 0, 0
	}
	along := (1 - math.Exp(-math.Max(0, age)/face.cooking)) * (rungs - 1)
	rung = min(int(math.Floor(along)), rungs-2)
	return rung, along - float64(rung)
}
