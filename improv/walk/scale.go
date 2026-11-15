package walk

import (
	"fmt"
	"math"
)

// Bar is a scale bar: a round length in metres, how many pixels it is
// across the picture where it stands, and the pixel of its place.
type Bar struct {
	metres, pixels float64
	at             Pixel
	unit           string
}

// Metres is the bar's length on the ground, in the camera's unit.
func (bar Bar) Metres() float64 { return bar.metres }

// Pixels is the bar's length on the picture.
func (bar Bar) Pixels() float64 { return bar.pixels }

// At is the pixel of the place the bar is true at.
func (bar Bar) At() Pixel { return bar.at }

// Words is the bar's length as the picture says it: "500 m", "2 km" on
// a set; "1,000 km" in a scene measured in kilometres.
func (bar Bar) Words() string {
	if bar.unit == metres && bar.metres >= kilometre {
		return grouped(bar.metres/kilometre) + " km"
	}
	return grouped(bar.metres) + " " + bar.unit
}

// grouped is a length written with its thousands set apart by commas.
func grouped(length float64) string {
	words := fmt.Sprintf("%g", length)
	if length < kilometre || length != math.Trunc(length) {
		return words
	}
	words = fmt.Sprintf("%.0f", length)
	for at := len(words) - thousand; at > 0; at -= thousand {
		words = words[:at] + "," + words[at:]
	}
	return words
}

// thousand is the digits a comma sets apart. Defined.
const thousand = 3

// kilometre is a kilometre in metres. Defined.
const kilometre = 1000.0

// PerMetre is how many pixels a metre of level ground at a place spans
// across the picture: a metre along the level way the picture's rows
// run, projected. A place behind the eye, or one whose level metre is
// seen end on, is refused.
func (view Camera) PerMetre(at Place) (float64, error) {
	across := view.eye.Orientation.X
	if view.level {
		across.Y = 0
	}
	if across.Len() == 0 {
		return 0, fmt.Errorf("%w: the camera's rows run straight up or "+
			"down; there is no level way across it", ErrWalk)
	}
	step := placeOf(at.world().Add(across.Unit()))
	one, other := view.Project(at), view.Project(step)
	if !one.ahead || !other.ahead {
		return 0, fmt.Errorf("%w: %.1f e, %.1f n is behind the camera; "+
			"put the bar where the camera sees", ErrWalk, at.east, at.north)
	}
	return math.Hypot(other.x-one.x, other.y-one.y), nil
}

// ScaleBar is the longest round length (scaleSteps times a power of
// ten metres) whose bar at a place is no longer than most pixels.
func (view Camera) ScaleBar(at Place, most float64) (Bar, error) {
	per, err := view.PerMetre(at)
	if err != nil {
		return Bar{}, err
	}
	if !(per > 0) || !(most > 0) {
		return Bar{}, fmt.Errorf("%w: a bar of %v pixels at %v a metre; "+
			"give a length on the picture", ErrWalk, most, per)
	}
	return Bar{metres: roundDown(most / per), pixels: roundDown(most/per) *
		per, at: view.Project(at), unit: view.unit}, nil
}

// roundDown is the largest of scaleSteps times a power of ten at or
// under a length.
func roundDown(length float64) float64 {
	power := math.Pow(10, math.Floor(math.Log10(length*(1+roundingSlack))))
	best := power
	for _, step := range scaleSteps {
		if step*power <= length*(1+roundingSlack) {
			best = step * power
		}
	}
	return best
}
