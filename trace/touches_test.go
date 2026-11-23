package trace

import (
	"fmt"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
)

// touchNear and touchFar are the span a shadow ray is asked over, and
// touchScale the scale of the thing it is asked of, so that the span the
// shape is asked over is the one divided by the scale.
const (
	touchNear  = 2.0
	touchFar   = 10.0
	touchScale = 2.0
)

// saysWhere is a shape that answers where a ray meets it, and counts how
// often it was asked and over what span.
type saysWhere struct {
	asked     int
	near, far float64
	met       bool
}

// Bounds is the box of the shape, which these checks never ask.
func (*saysWhere) Bounds() maths.Box { return maths.Box{} }

// Distance is a distance the checks never ask.
func (*saysWhere) Distance(maths.Vec) float64 { return 1 }

// Hit counts the ask and answers as the shape is set to.
func (shape *saysWhere) Hit(_ maths.Ray, near, far float64) (float64,
	bool) {
	shape.asked++
	shape.near, shape.far = near, far
	return 1, shape.met
}

// saysWhether is a shape that answers whether too, and counts how often
// it was asked that, and over what span.
type saysWhether struct {
	saysWhere
	asked     int
	near, far float64
}

// Meets counts the ask and answers as the shape is set to.
func (shape *saysWhether) Meets(_ maths.Ray, near, far float64) bool {
	shape.asked++
	shape.near, shape.far = near, far
	return shape.met
}

// placedAs is a thing of the shape, standing at the origin at the scale.
func placedAs(solid *thing.Thing) placedThing {
	return placedThing{thing: solid, placement: placement{
		at: maths.Local(), scale: touchScale}}
}

// touchRay is the ray the checks ask about.
var touchRay = maths.Ray{At: maths.V(0, 0, 0), Dir: maths.V(1, 0, 0)}

// askFaults is what is wrong with how a shape that can say whether was
// asked: it is asked whether that many times, and where never.
func askFaults(shape *saysWhether, whether int) []string {
	var faults []string
	if shape.asked != whether {
		faults = append(faults, fmt.Sprintf("asked whether %d times, "+
			"want %d", shape.asked, whether))
	}
	if shape.saysWhere.asked != 0 {
		faults = append(faults, fmt.Sprintf("asked where %d times, want "+
			"none", shape.saysWhere.asked))
	}
	return faults
}

// whereFaults is what is wrong with how a shape that can only say where
// was asked: it is asked where once.
func whereFaults(shape *saysWhere) []string {
	if shape.asked != 1 {
		return []string{fmt.Sprintf("asked where %d times, want 1",
			shape.asked)}
	}
	return nil
}

// A shape that can say whether a ray meets it is asked that, once, over
// its own lengths, and is never asked where: that is the work a shadow
// ray need not do.
func TestAShapeThatCanSayWhetherIsAskedWhetherAndNotWhere(t *testing.T) {
	for _, answer := range []bool{true, false} {
		shape := &saysWhether{}
		shape.met = answer
		placed := placedAs(&thing.Thing{Shape: shape})
		got := placed.touches(touchRay, touchNear, touchFar)
		if got != answer {
			t.Errorf("a shape that says %v touched %v", answer, got)
		}
		for _, fault := range askFaults(shape, 1) {
			t.Error(fault)
		}
		if shape.near != touchNear/touchScale ||
			shape.far != touchFar/touchScale {
			t.Errorf("asked over %v to %v, not in the shape's lengths",
				shape.near, shape.far)
		}
	}
}

// Anti-test: a shape that is asked where as well as whether is reported.
func TestAShapeAskedWhereAsWellAsWhetherIsReported(t *testing.T) {
	shape := &saysWhether{}
	shape.Meets(touchRay, touchNear, touchFar)
	shape.Hit(touchRay, touchNear, touchFar)
	if faults := askFaults(shape, 1); len(faults) != 1 {
		t.Errorf("a shape asked where gave %v, want one fault", faults)
	}
}

// A shape that can only say where is asked where, once, and what it says
// is what touches says.
func TestAShapeThatCanOnlySayWhereIsAskedWhereOnce(t *testing.T) {
	for _, answer := range []bool{true, false} {
		shape := &saysWhere{met: answer}
		placed := placedAs(&thing.Thing{Shape: shape})
		got := placed.touches(touchRay, touchNear, touchFar)
		if got != answer {
			t.Errorf("a shape that says %v touched %v", answer, got)
		}
		for _, fault := range whereFaults(shape) {
			t.Error(fault)
		}
	}
}

// Anti-test: a shape that is asked where twice, or not at all, is
// reported.
func TestAShapeAskedWhereTwiceOrNeverIsReported(t *testing.T) {
	shape := &saysWhere{}
	if len(whereFaults(shape)) != 1 {
		t.Error("a shape never asked where is not reported")
	}
	shape.Hit(touchRay, touchNear, touchFar)
	shape.Hit(touchRay, touchNear, touchFar)
	if len(whereFaults(shape)) != 1 {
		t.Error("a shape asked where twice is not reported")
	}
}

// castFaults is what is wrong with a crowd's shadow check of two things,
// the second of which is clear if the flag says so: the first casts a
// shadow and is asked whether only, and a clear one casts none and is
// asked nothing.
func castFaults(opaque, passed *saysWhether, clearSecond bool) []string {
	seen := &crowd{things: []placedThing{
		placedAs(&thing.Thing{Shape: opaque}),
		placedAs(&thing.Thing{Shape: passed,
			Attrs: thing.Attrs{Clear: clearSecond}})}}
	var faults []string
	if !seen.casts(0, touchRay, touchFar) {
		faults = append(faults, "a thing that is met casts no shadow")
	}
	faults = append(faults, askFaults(opaque, 1)...)
	if seen.casts(1, touchRay, touchFar) {
		faults = append(faults, "a clear thing casts a shadow")
	}
	return append(faults, askFaults(passed, 0)...)
}

// A crowd's shadow check asks a thing that can say whether only that, and
// asks a clear thing nothing at all.
func TestACrowdsShadowCheckAsksWhetherAndNotWhere(t *testing.T) {
	opaque, passed := &saysWhether{}, &saysWhether{}
	opaque.met, passed.met = true, true
	for _, fault := range castFaults(opaque, passed, true) {
		t.Error(fault)
	}
}

// Anti-test: a second thing that is not clear is a shadow cast and is
// asked, which the check reports.
func TestAThingThatIsNotClearIsReportedAsAskedAndCasting(t *testing.T) {
	opaque, passed := &saysWhether{}, &saysWhether{}
	opaque.met, passed.met = true, true
	if faults := castFaults(opaque, passed, false); len(faults) != 2 {
		t.Errorf("a thing that is not clear gave %v, want two faults",
			faults)
	}
}
