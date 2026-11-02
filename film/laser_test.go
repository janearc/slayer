package film

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
)

// A laser lights its axis in full, falls off across its width by a
// gaussian, lights nothing behind its source or past its reach, says
// the way back to the source, and fills a box round its beam.
func TestALaserIsANarrowBeam(t *testing.T) {
	l := Laser{At: maths.V(0, 1, 0), Aim: maths.V(0, 0, -1), Width: 0.1, Strength: maths.RGB(0, 10, 0), Length: 50}
	toward, far, light := l.Light(maths.V(0, 1, -5), maths.Up)
	if light != l.Strength || toward != maths.V(0, 0, 1) || far != 5 {
		t.Errorf("on the axis: %v %v %v", toward, far, light)
	}
	_, _, half := l.Light(maths.V(0.05, 1, -5), maths.Up)
	if want := 10 * math.Exp(-0.5); math.Abs(half.G-want) > 1e-9 {
		t.Errorf("half a width off the axis the light is %v, want %v", half.G, want)
	}
	if _, _, far := l.Light(maths.V(0.5, 1, -5), maths.Up); far != (maths.Light{}) {
		t.Errorf("five widths off the axis the light is %v", far)
	}
	if _, _, behind := l.Light(maths.V(0, 1, 3), maths.Up); behind != (maths.Light{}) {
		t.Errorf("behind the source the light is %v", behind)
	}
	if _, _, past := l.Light(maths.V(0, 1, -60), maths.Up); past != (maths.Light{}) {
		t.Errorf("past the reach the light is %v", past)
	}
	b := l.Bounds()
	if b.Min.Z > -50 || b.Max.Z < 0 || b.Max.X-b.Min.X < 0.6 {
		t.Errorf("the beam's box is %v", b)
	}
}
