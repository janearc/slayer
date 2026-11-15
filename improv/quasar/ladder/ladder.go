// Package ladder is the pull-back in steps of ten: frames each ten times
// wider than the last, from something a person knows out to the jets of
// the largest black holes, each holding the frame before it as a marked
// dot a tenth of its width.
//
// Twenty powers of ten cannot be one traced move at one precision, so
// each step is its own picture, drawn in its own units, and the joins
// are shown: the marked dot is where the last picture was.
//
// A Thing is something with a size, from a registered or computed value.
// A Step says which things are big enough to see in it (a pixel or more)
// and small enough to fit (no wider than the frame).
package ladder

import (
	"fmt"
	"math"
)

// The numbers the package is made of, defined.
const (
	none  = 0
	one   = 1
	ten   = 10 // each step is ten times the last
	tenth = 1.0 / ten
)

// Thing is something drawn to scale: its name and its size across, in
// metres, and where the size came from.
type Thing struct {
	name string
	size float64
	from string
}

// NewThing is a thing of a size across in metres, with where the size
// came from. It refuses a size that is not above nought and finite, and
// a thing with no name or no source.
func NewThing(name string, size float64, from string) (Thing, error) {
	if !(size > none) || math.IsInf(size, 0) {
		return Thing{}, fmt.Errorf("ladder: %q is %v metres across: "+
			"give a finite size above nought", name, size)
	}
	if name == "" || from == "" {
		return Thing{}, fmt.Errorf("ladder: a thing %v metres across "+
			"needs a name and a source", size)
	}
	return Thing{name: name, size: size, from: from}, nil
}

// Name is what the thing is.
func (thing Thing) Name() string { return thing.name }

// Size is the thing's size across, in metres.
func (thing Thing) Size() float64 { return thing.size }

// From is where the size came from.
func (thing Thing) From() string { return thing.from }

// Step is one frame of the pull-back: how wide it is, in metres, and how
// wide the frame before it was, which it holds as a marked dot.
type Step struct {
	width  float64
	marked float64
}

// Width is the frame's width, in metres.
func (step Step) Width() float64 { return step.width }

// Marked is the last frame's width, in metres, drawn here as a dot; nought
// for the first step, which holds none.
func (step Step) Marked() float64 { return step.marked }

// Shows is whether a thing can be seen in this frame drawn so many pixels
// across: a pixel or more, and no wider than the frame.
func (step Step) Shows(thing Thing, pixels int) bool {
	across := thing.size / step.width * float64(pixels)
	return across >= one && thing.size <= step.width
}

// Ladder is the pull-back: its steps from the first frame's width out.
type Ladder struct {
	steps []Step
}

// NewLadder is a pull-back from a first frame so many metres wide, out by
// steps of ten until a frame holds the widest thing. It refuses a first
// width that is not above nought and finite, and no things.
func NewLadder(first float64, things []Thing) (Ladder, error) {
	if !(first > none) || math.IsInf(first, 0) {
		return Ladder{}, fmt.Errorf("ladder: a first frame %v metres "+
			"wide is none: give a finite width above nought", first)
	}
	if len(things) == none {
		return Ladder{}, fmt.Errorf("ladder: a pull-back past nothing: " +
			"give one or more things")
	}
	widest := none * first
	for _, thing := range things {
		widest = math.Max(widest, thing.size)
	}
	steps := []Step{{width: first}}
	for steps[len(steps)-one].width < widest {
		last := steps[len(steps)-one].width
		steps = append(steps, Step{width: last * ten, marked: last})
	}
	return Ladder{steps: steps}, nil
}

// Steps is the pull-back's frames, from the first out.
func (ladder Ladder) Steps() []Step {
	return append([]Step(nil), ladder.steps...)
}
