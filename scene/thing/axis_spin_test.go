package thing

import (
	"math"
	"testing"
	"time"

	"github.com/janearc/slayer/maths"
)

// The tables the axis spin tests read, beside numbers.yaml.
const (
	axisSpinPoints   = "testdata/axis-spin-points.csv"
	axisSpinRefusals = "testdata/axis-spin-refusals.csv"
)

// axisSpinPoint is the point called name in the points table.
func axisSpinPoint(t *testing.T, name string) maths.Vec {
	t.Helper()
	row := readSheet(t, axisSpinPoints).row(t, axisSpinPoints, name)
	return maths.V(cell(t, row, "x"), cell(t, row, "y"), cell(t, row, "z"))
}

// axisSpinMillis is the picked number called name, as whole milliseconds.
func axisSpinMillis(t *testing.T, name string) time.Duration {
	t.Helper()
	return time.Duration(pickedCount(t, name)) * time.Millisecond
}

// axisTestParent is a parent motion that moves along x, turns about y and is
// scaled, so that a spin on it shows the order the two are composed in.
type axisTestParent struct {
	base         maths.Vec
	angle, scale float64
}

// axisSpinParent is the moving, scaled parent the data describes.
func axisSpinParent(t *testing.T) axisTestParent {
	t.Helper()
	return axisTestParent{
		base:  axisSpinPoint(t, "parent-base"),
		angle: picked(t, "axis-spin-parent-angle"),
		scale: picked(t, "axis-spin-parent-scale"),
	}
}

// At is the parent's frame: its base moved along x by the seconds elapsed.
func (parent axisTestParent) At(at time.Duration) maths.Subspace {
	origin := maths.V(parent.base.X+at.Seconds(), parent.base.Y, parent.base.Z)
	turn := maths.Rotation(maths.YAxis, parent.angle)
	return maths.NewSubspace(origin, turn, parent.scale)
}

// a spin keeps its pivot where the moving, scaled parent puts it, turns the
// quarter turn the clock says, and leaves the parent's scale as it is.
func TestAxisSpinKeepsPivotAndComposesMovingScaledParent(t *testing.T) {
	parent := axisSpinParent(t)
	pivot := axisSpinPoint(t, "pivot")
	axis := axisSpinPoint(t, "twice-z")
	spin, err := NewAxisSpin(parent, pivot, axis, picked(t, "axis-spin-rpm"),
		nought)
	if err != nil {
		t.Fatal(err)
	}
	if got := spin.At(nought); got != parent.At(nought) {
		t.Fatal("zero angle changed parent bits")
	}
	at := axisSpinMillis(t, "axis-spin-moving-milliseconds")
	pose, base := spin.At(at), parent.At(at)
	tolerance := picked(t, "axis-spin-tolerance")
	got, want := pose.Out(pivot), base.Out(pivot)
	if got.Sub(want).Len() > tolerance {
		t.Fatalf("pivot moved: %v want%v", got, want)
	}
	// Quarter-turn about local+Z sends a local+X offset to local+Y.
	got = pose.Out(pivot.Add(maths.XAxis))
	want = base.Out(pivot.Add(maths.YAxis))
	if got.Sub(want).Len() > tolerance {
		t.Fatalf("rotation/parent order wrong: %v want%v", got, want)
	}
	if pose.Scale != base.Scale {
		t.Fatal("parent scale changed")
	}
}

// a negative rate with a phase of a quarter turn takes x to y at the start,
// and a spin whose angle comes to zero, or whose rate is zero, returns its
// parent exactly.
func TestAxisSpinSignedRPMPhaseAndExactRest(t *testing.T) {
	height := picked(t, "axis-spin-rest-height")
	origin := maths.V(nought, height, nought)
	frame := maths.NewSubspace(origin, maths.Identity, nought)
	parent := Still(frame)
	centre := maths.V(nought, nought, nought)
	reverse := picked(t, "axis-spin-reverse-rpm")
	spin, err := NewAxisSpin(parent, centre, maths.ZAxis, reverse,
		math.Pi/pair)
	if err != nil {
		t.Fatal(err)
	}
	got := spin.At(nought).Out(maths.XAxis)
	if got.Sub(axisSpinPoint(t, "reversed-x")).Len() >
		picked(t, "axis-spin-tolerance") {
		t.Fatal(got)
	}
	moving := axisSpinMillis(t, "axis-spin-moving-milliseconds")
	if got := spin.At(moving); got != parent.At(moving) {
		t.Fatal("zero resulting angle did not return exact parent")
	}
	pivot := axisSpinPoint(t, "pivot")
	stopped, err := NewAxisSpin(parent, pivot, maths.ZAxis, nought, nought)
	if err != nil {
		t.Fatal(err)
	}
	seconds := pickedCount(t, "axis-spin-long-seconds")
	long := time.Duration(seconds) * time.Second
	if got := stopped.At(long); got != parent.At(long) {
		t.Fatal("stopped spin changed parent")
	}
}

// axisSpinVector is the vector in the columns of a row that start with
// prefix, as prefix_x, prefix_y and prefix_z.
func axisSpinVector(t *testing.T, row map[string]string,
	prefix string) maths.Vec {
	t.Helper()
	x := cell(t, row, prefix+"_x")
	y := cell(t, row, prefix+"_y")
	z := cell(t, row, prefix+"_z")
	return maths.V(x, y, z)
}

// axisSpinRefused fails the test when the spin the row describes is taken.
func axisSpinRefused(t *testing.T, row map[string]string) {
	t.Helper()
	var parent Motion
	if row["parent"] == "still" {
		parent = Still(maths.Local())
	}
	pivot := axisSpinVector(t, row, "pivot")
	axis := axisSpinVector(t, row, "axis")
	rpm, phase := cell(t, row, "rpm"), cell(t, row, "phase")
	if _, err := NewAxisSpin(parent, pivot, axis, rpm, phase); err == nil {
		t.Fatal("invalid spin accepted")
	}
}

// a spin with no parent, a pivot or axis that is not finite, no axis, a rate
// that is not a number, or a phase that is not finite is refused, and so is a
// rate past the maximum.
func TestAxisSpinRefusesInvalidConfiguration(t *testing.T) {
	for _, row := range readSheet(t, axisSpinRefusals) {
		axisSpinRefused(t, row)
	}
	parent := Still(maths.Local())
	_, err := NewAxisSpin(parent, maths.V(nought, nought, nought), maths.ZAxis,
		MaximumAxisSpinRPM+unity, nought)
	if err == nil {
		t.Fatal("invalid spin accepted")
	}
}
