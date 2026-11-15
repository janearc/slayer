package rig

import "fmt"

// Layer is a traced pass of the set lit by one fixture alone, by its
// name, and how much of it a moment adds: an amount, and a colour each
// channel is taken through. Light adds, so a moment's frame from any
// camera is its plate, dimmed by the master, plus each layer's pass
// times its amount and colour, in linear light.
type Layer struct {
	Name   string
	Amount float64
	Colour [3]float64
}

// Floods is the two flood layers at their flashes, as lightshow lays
// them: the left pass white, the right through the magenta gel, each
// gain times its flash. names are the passes the floods fire, left then
// right (lightshow's FLOOD_PASSES: the film's are the backlights).
func Floods(names [2]string, left, right, gain float64) []Layer {
	return []Layer{{names[0], gain * left, [3]float64{1, 1, 1}},
		{names[1], gain * right, Gel}}
}

// Gel is the magenta gel the right flood and the magenta strobe shine
// through. Lightshow's.
var Gel = [3]float64{1, 0.25, 0.85}

// OverheadColour is the overhead's daylight colour, a cool blue that is
// not quite tungsten's. Lightshow's.
var OverheadColour = [3]float64{0.62, 0.8, 1.0}

// Overhead is the ship's light as a layer: the overhead pass, gain
// times the cue's level, through OverheadColour.
func Overhead(level, gain float64) Layer {
	return Layer{"overhead", gain * level, OverheadColour}
}

// Add lays a layer's pass into a frame's light: both three channels a
// pixel, interleaved, the same length.
func (layer Layer) Add(into, pass []float64) error {
	if len(into) != len(pass) {
		return fmt.Errorf("rig: the pass %s holds %d numbers and the frame "+
			"%d; trace it at the frame's size", layer.Name, len(pass),
			len(into))
	}
	for i := 0; i < len(into); i += len(layer.Colour) {
		for c := range layer.Colour {
			into[i+c] += layer.Amount * layer.Colour[c] * pass[i+c]
		}
	}
	return nil
}
