//go:build draw && darwin

package metal

import (
	"encoding/binary"
	"errors"
	"math"
	"testing"
)

// openWhere opens the where kernel, failing, not skipping, where there
// is no device: a check that cannot run fails.
func openWhere(t *testing.T) *Kernel {
	t.Helper()
	kernel, err := Open(where, "where")
	if errors.Is(err, ErrNoDevice) {
		t.Fatal("these checks need a Metal device and there is none:", err)
	}
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(kernel.Close)
	return kernel
}

// the where kernel's shapes in these checks: a picture, a band's rows,
// and the numbers it writes a pixel, by the kernel's own float4
const (
	streamWidth, streamHeight, streamRows = 5, 4, 2
	wherePer                              = 4
)

// TestAStreamKnowsTheBytesItHolds compares an input with the bytes it
// was given, the same bytes in another slice, and an empty input with
// an empty one.
func TestAStreamKnowsTheBytesItHolds(t *testing.T) {
	kernel := openWhere(t)
	stream, err := kernel.OpenGrid(streamWidth, streamHeight, streamRows,
		wherePer, [][]byte{floats(7, 8), nil})
	if err != nil {
		t.Fatal(err)
	}
	defer stream.Close()
	for _, check := range []struct {
		index int
		input []byte
	}{{0, floats(7, 8)}, {1, nil}, {1, []byte{}}} {
		same, err := stream.Same(check.index, check.input)
		if err != nil || !same {
			t.Errorf("input %d with %v is not the same (%v)", check.index,
				check.input, err)
		}
	}
}

// TestAStreamTellsADifferentInputApart is the near misses: one bit
// changed, a byte short, an empty input's zeros given as bytes, and
// bytes for an input that was empty; each is not the same.
func TestAStreamTellsADifferentInputApart(t *testing.T) {
	kernel := openWhere(t)
	stream, err := kernel.OpenGrid(streamWidth, streamHeight, streamRows,
		wherePer, [][]byte{floats(7, 8), nil})
	if err != nil {
		t.Fatal(err)
	}
	defer stream.Close()
	flipped := floats(7, 8)
	flipped[len(flipped)-1] ^= 1
	for name, check := range map[string]struct {
		index int
		input []byte
	}{
		"one bit":            {0, flipped},
		"a byte short":       {0, floats(7, 8)[:7]},
		"zeros for nothing":  {1, make([]byte, wherePer)},
		"bytes for nothing":  {1, floats(7)},
		"nothing for bytes":  {0, nil},
		"the other's values": {0, floats(8, 7)},
	} {
		same, err := stream.Same(check.index, check.input)
		if err != nil || same {
			t.Errorf("%s: input %d is the same as %v (%v)", name,
				check.index, check.input, err)
		}
	}
}

// TestAStreamSameFollowsUpdates says the same after an update to new
// bytes, and not the same for the old ones.
func TestAStreamSameFollowsUpdates(t *testing.T) {
	kernel := openWhere(t)
	stream, err := kernel.OpenGrid(streamWidth, streamHeight, streamRows,
		wherePer, [][]byte{floats(7)})
	if err != nil {
		t.Fatal(err)
	}
	defer stream.Close()
	for _, given := range [][]byte{floats(11), floats(3, 9), nil} {
		if err := stream.Update(0, given); err != nil {
			t.Fatal(err)
		}
		if same, err := stream.Same(0, given); err != nil || !same {
			t.Errorf("after an update to %v it is not the same (%v)",
				given, err)
		}
		if same, _ := stream.Same(0, floats(7)); same {
			t.Errorf("after an update to %v the first bytes are the same",
				given)
		}
	}
}

// TestAStreamRefusesToCompareWhatIsNotThere refuses an input it does
// not have and a closed stream.
func TestAStreamRefusesToCompareWhatIsNotThere(t *testing.T) {
	kernel := openWhere(t)
	stream, err := kernel.OpenGrid(streamWidth, streamHeight, streamRows,
		wherePer, [][]byte{floats(7)})
	if err != nil {
		t.Fatal(err)
	}
	for _, index := range []int{-1, 1} {
		if _, err := stream.Same(index, floats(7)); err == nil {
			t.Errorf("comparing input %d was not refused", index)
		}
	}
	stream.Close()
	if _, err := stream.Same(0, floats(7)); err == nil {
		t.Error("comparing on a closed stream was not refused")
	}
	if err := stream.Reshape(streamWidth, streamHeight, streamRows,
		wherePer); err == nil {
		t.Error("reshaping a closed stream was not refused")
	}
}

// TestAStreamDrawsAnotherShapeOverTheSameInputs reshapes a stream to a
// wider picture in other bands and back, and each picture is what the
// where kernel writes at that shape, with the input it was given.
func TestAStreamDrawsAnotherShapeOverTheSameInputs(t *testing.T) {
	kernel := openWhere(t)
	stream, err := kernel.OpenGrid(streamWidth, streamHeight, streamRows,
		wherePer, [][]byte{floats(7)})
	if err != nil {
		t.Fatal(err)
	}
	defer stream.Close()
	wider, taller, deeper := streamWidth+4, streamHeight+3, streamRows+1
	for _, shape := range [][3]int{{wider, taller, deeper},
		{streamWidth, streamHeight, streamRows}, {wider, taller, deeper}} {
		if err := stream.Reshape(shape[0], shape[1], shape[2],
			wherePer); err != nil {
			t.Fatal(err)
		}
		// the where kernel writes its band's first row, so the bands
		// are fixed at the shape's most rows, where expected finds them
		kernel.FixBands(shape[2])
		out, err := stream.Draw()
		if err != nil {
			t.Fatal(err)
		}
		if len(out) != shape[0]*shape[1]*wherePer {
			t.Fatalf("a %v picture has %d numbers", shape, len(out))
		}
		for row := range shape[1] {
			for column := range shape[0] {
				pixel := [4]float32(out[(row*shape[0]+column)*wherePer:])
				if want := expected(column, row, shape[2]); pixel != want {
					t.Fatalf("at %v, pixel %d,%d is %v, want %v", shape,
						column, row, pixel, want)
				}
			}
		}
	}
}

// TestAStreamRefusesAShapeItCannotTake keeps the old shape after a
// refusal: no width, no rows, no numbers, and too many numbers.
func TestAStreamRefusesAShapeItCannotTake(t *testing.T) {
	kernel := openWhere(t)
	stream, err := kernel.OpenGrid(streamWidth, streamHeight, streamRows,
		wherePer, [][]byte{floats(7)})
	if err != nil {
		t.Fatal(err)
	}
	defer stream.Close()
	for name, shape := range map[string][4]int{
		"no width":       {0, streamHeight, streamRows, wherePer},
		"no rows":        {streamWidth, streamHeight, 0, wherePer},
		"no numbers":     {streamWidth, streamHeight, streamRows, 0},
		"too big a grid": {math.MaxInt32, math.MaxInt32, 1, wherePer},
	} {
		if err := stream.Reshape(shape[0], shape[1], shape[2],
			shape[3]); err == nil {
			t.Errorf("%s: reshaping to %v was not refused", name, shape)
		}
	}
	out, err := stream.Draw()
	if err != nil || len(out) != streamWidth*streamHeight*wherePer {
		t.Fatalf("after the refusals the picture has %d numbers (%v)",
			len(out), err)
	}
}

// layoutOf is a plan of one triangle placed once at a matrix and mask,
// as its layout.
func layoutOf(t *testing.T, matrix [12]float32, mask uint32) Layout {
	t.Helper()
	plan, err := NewAcceleration(triangleData(), 16)
	if err != nil {
		t.Fatal(err)
	}
	triangle, err := plan.Triangles(0, 1)
	if err != nil {
		t.Fatal(err)
	}
	placed, err := Placed(triangle, matrix, mask)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = plan.Instances(placed); err != nil {
		t.Fatal(err)
	}
	return plan.Layout()
}

// TestALayoutIsTheSameAsItsTwin builds the same plan twice: the layouts
// are the same, and a layout copied out of a plan does not change when
// the plan grows.
func TestALayoutIsTheSameAsItsTwin(t *testing.T) {
	one := layoutOf(t, translated(0, 0, -5), 3)
	if !one.Same(layoutOf(t, translated(0, 0, -5), 3)) {
		t.Error("two layouts of the same plan are not the same")
	}
	plan, err := NewAcceleration(triangleData(), 16)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = plan.Triangles(0, 1); err != nil {
		t.Fatal(err)
	}
	before := plan.Layout()
	if _, err = plan.Triangles(0, 1); err != nil {
		t.Fatal(err)
	}
	if before.Same(plan.Layout()) {
		t.Error("a layout taken before the plan grew is the grown one's")
	}
	if !(*Acceleration)(nil).Layout().Same(Layout{}) {
		t.Error("no plan's layout is not the empty one")
	}
}

// TestALayoutTellsANearMissApart is a placement moved by a negative
// zero, another mask, and another place: none is the same.
func TestALayoutTellsANearMissApart(t *testing.T) {
	one := layoutOf(t, translated(0, 0, -5), 3)
	negative := translated(0, 0, -5)
	negative[9] = float32(math.Copysign(0, -1))
	for name, other := range map[string]Layout{
		"a negative zero": layoutOf(t, negative, 3),
		"another mask":    layoutOf(t, translated(0, 0, -5), 1),
		"another place":   layoutOf(t, translated(0, 0, -4), 3),
		"no plan":         {},
	} {
		if one.Same(other) {
			t.Errorf("%s is the same layout", name)
		}
	}
}

// chunks is an input of three whole refresh parts and a few bytes more,
// its first number seven, so the where kernel draws a seven.
func chunks() []byte {
	input := make([]byte, 3*refreshChunk+10)
	copy(input, floats(7))
	return input
}

// TestAStreamRefreshCopiesOnlyTheChangedParts changes one byte in the
// second part and then the first number: each refresh copies one part,
// the input is then the same, and the picture draws the new number.
func TestAStreamRefreshCopiesOnlyTheChangedParts(t *testing.T) {
	kernel := openWhere(t)
	input := chunks()
	stream, err := kernel.OpenGrid(streamWidth, streamHeight, streamRows,
		wherePer, [][]byte{input})
	if err != nil {
		t.Fatal(err)
	}
	defer stream.Close()
	input[refreshChunk+1] ^= 1
	if sent, err := stream.Refresh(0, input); err != nil ||
		sent != refreshChunk {
		t.Fatalf("one changed byte sent %d (%v), want one part", sent, err)
	}
	copy(input, floats(11))
	if sent, err := stream.Refresh(0, input); err != nil ||
		sent != refreshChunk {
		t.Fatalf("a changed number sent %d (%v), want one part", sent, err)
	}
	if same, err := stream.Same(0, input); err != nil || !same {
		t.Fatalf("after the refresh the input is not the same (%v)", err)
	}
	out, err := stream.Draw()
	if err != nil || out[3] != 11 {
		t.Fatalf("the refreshed input draws %v (%v), want 11", out[3], err)
	}
}

// TestAStreamRefreshSendsNothingUnchangedAndOtherLengthsWhole sends no
// bytes for the same input or for nothing given nothing, every byte of
// an input of another length, and refuses what is not there.
func TestAStreamRefreshSendsNothingUnchangedAndOtherLengthsWhole(t *testing.T) {
	kernel := openWhere(t)
	stream, err := kernel.OpenGrid(streamWidth, streamHeight, streamRows,
		wherePer, [][]byte{chunks(), nil})
	if err != nil {
		t.Fatal(err)
	}
	for _, check := range []struct {
		index int
		input []byte
		sent  int64
	}{{0, chunks(), 0}, {1, nil, 0}, {0, floats(3, 9), 8}, {1, floats(2), 4},
		{0, nil, 0}} {
		sent, err := stream.Refresh(check.index, check.input)
		if err != nil || sent != check.sent {
			t.Errorf("refreshing %d with %d bytes sent %d (%v), want %d",
				check.index, len(check.input), sent, err, check.sent)
		}
		if same, _ := stream.Same(check.index, check.input); !same {
			t.Errorf("after refreshing %d it is not the same", check.index)
		}
	}
	if _, err := stream.Refresh(2, floats(1)); err == nil {
		t.Error("refreshing an input that is not there was not refused")
	}
	stream.Close()
	if _, err := stream.Refresh(0, floats(1)); err == nil {
		t.Error("refreshing a closed stream was not refused")
	}
}

// probeRays are rays down from x of nought and of three, every mask,
// ten metres long: a triangle under one is met by it alone.
func probeRays() []byte {
	var rays []byte
	for _, across := range []float32{0, 3} {
		for _, value := range []float32{across, 0, 0, 3, 0, 0, -1, 10} {
			rays = binary.LittleEndian.AppendUint32(rays,
				math.Float32bits(value))
		}
	}
	return rays
}

// placedPlan is one triangle placed at x, as a plan and its root.
func placedPlan(t *testing.T, across float32) (*Acceleration, int) {
	t.Helper()
	plan, err := NewAcceleration(triangleData(), 16)
	if err != nil {
		t.Fatal(err)
	}
	triangle, err := plan.Triangles(0, 1)
	if err != nil {
		t.Fatal(err)
	}
	placed, err := Placed(triangle, translated(across, 0, -5), 3)
	if err != nil {
		t.Fatal(err)
	}
	root, err := plan.Instances(placed)
	if err != nil {
		t.Fatal(err)
	}
	return plan, root
}

// TestAKernelRebuildsItsHierarchyInPlace gives a kernel a first
// hierarchy by Rebuild, then moves the triangle by another: each query
// meets the triangle where it now is. A plan that is refused leaves the
// last one, and a closed kernel refuses.
func TestAKernelRebuildsItsHierarchyInPlace(t *testing.T) {
	kernel, err := Open(acceleratedProbe, "probe")
	if errors.Is(err, ErrNoDevice) {
		t.Fatal("this check needs a Metal device and there is none:", err)
	}
	if err != nil {
		t.Fatal(err)
	}
	defer kernel.Close()
	for _, step := range []struct {
		across float32
		want   []float32
	}{{0, []float32{1, 0}}, {3, []float32{0, 1}}} {
		plan, root := placedPlan(t, step.across)
		if err := kernel.Rebuild(plan, root); err != nil {
			t.Fatal(err)
		}
		got, err := kernel.Grid(len(step.want), 1, 1, 1,
			[][]byte{probeRays()})
		if err != nil || got[0] != step.want[0] || got[1] != step.want[1] {
			t.Fatalf("at %v the queries met %v (%v), want %v",
				step.across, got, err, step.want)
		}
	}
	if err := kernel.Rebuild(nil, 0); err == nil {
		t.Fatal("no plan was built")
	}
	got, err := kernel.Grid(2, 1, 1, 1, [][]byte{probeRays()})
	if err != nil || got[0] != 0 || got[1] != 1 {
		t.Fatalf("a refused rebuild changed the hierarchy: %v (%v)", got, err)
	}
	kernel.Close()
	plan, root := placedPlan(t, 0)
	if err := kernel.Rebuild(plan, root); err == nil {
		t.Fatal("a closed kernel rebuilt its hierarchy")
	}
}
