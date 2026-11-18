package emptyspace_test

import (
	"math"
	"strings"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/render/emptyspace"
)

// oneField is a field of the fixture's grid, all zero, for a build.
func oneField() []emptyspace.Field {
	return []emptyspace.Field{{Values: make([]float32, samples*samples*samples),
		Channels: 1}}
}

// build builds over the fixture's cube and grid.
func build(fields []emptyspace.Field, policy emptyspace.Options) (*emptyspace.Mask, error) {
	return emptyspace.Build(cube, [3]int{samples, samples, samples}, fields,
		policy)
}

// The default options are the four picked numbers in the policy file.
func TestDefaultOptionsReadThePickedPolicy(t *testing.T) {
	got, err := emptyspace.DefaultOptions()
	if err != nil {
		t.Fatal(err)
	}
	want := emptyspace.Options{BrickSide: 8, Bytes: 1048576, Rays: 64,
		QueryBricks: 4096}
	if got != want {
		t.Errorf("DefaultOptions is %+v, want %+v", got, want)
	}
}

// Build refuses a grid, a policy or a field that cannot be certified, and
// says why.
func TestBuildRefusesWhatCannotBeCertified(t *testing.T) {
	flat := maths.Box{Min: maths.Vec{}, Max: maths.V(1, 0, 1)}
	nan := maths.Box{Min: maths.V(math.NaN(), 0, 0), Max: maths.V(1, 1, 1)}
	reversed := maths.Box{Min: maths.V(1, 1, 1), Max: maths.Vec{}}
	grid := [3]int{samples, samples, samples}
	badSample := make([]float32, samples*samples*samples)
	negative := append([]float32(nil), badSample...)
	negative[7] = -1
	notANumber := append([]float32(nil), badSample...)
	notANumber[9] = float32(math.NaN())
	endless := append([]float32(nil), badSample...)
	endless[11] = float32(math.Inf(1))
	policy := func(change func(*emptyspace.Options)) emptyspace.Options {
		made := options()
		change(&made)
		return made
	}
	cases := []struct {
		name   string
		bounds maths.Box
		cells  [3]int
		fields []emptyspace.Field
		policy emptyspace.Options
		want   string
	}{
		{"a box with no depth", flat, grid, oneField(), options(), "invalid grid"},
		{"a box with a NaN corner", nan, grid, oneField(), options(), "invalid grid"},
		{"a reversed box", reversed, grid, oneField(), options(), "invalid grid"},
		{"no brick", cube, grid, oneField(), policy(func(o *emptyspace.Options) {
			o.BrickSide = 0
		}), "invalid grid"},
		{"a budget under a word", cube, grid, oneField(),
			policy(func(o *emptyspace.Options) { o.Bytes = 7 }), "invalid grid"},
		{"no rays", cube, grid, oneField(), policy(func(o *emptyspace.Options) {
			o.Rays = 0
		}), "invalid grid"},
		{"no query bricks", cube, grid, oneField(),
			policy(func(o *emptyspace.Options) { o.QueryBricks = 0 }), "invalid grid"},
		{"no fields", cube, grid, nil, options(), "invalid grid"},
		{"one sample on an axis", cube, [3]int{samples, 1, samples},
			oneField(), options(), "at least two samples"},
		{"a field of the wrong size", cube, grid, []emptyspace.Field{{
			Values: make([]float32, 10), Channels: 1}}, options(),
			"does not match"},
		{"a field with no channels", cube, grid, []emptyspace.Field{{
			Values: badSample, Channels: 0}}, options(), "does not match"},
		{"channels that do not fit", cube, grid, []emptyspace.Field{{
			Values: badSample, Channels: 2}}, options(), "does not match"},
		{"occupancy past the budget", cube, grid, oneField(),
			policy(func(o *emptyspace.Options) { o.Bytes = 8 }), "payload budget"},
		{"a negative sample", cube, grid, []emptyspace.Field{{Values: negative,
			Channels: 1}}, options(), "negative or non-finite"},
		{"a NaN sample", cube, grid, []emptyspace.Field{{Values: notANumber,
			Channels: 1}}, options(), "negative or non-finite"},
		{"an infinite sample", cube, grid, []emptyspace.Field{{Values: endless,
			Channels: 1}}, options(), "negative or non-finite"},
	}
	for _, c := range cases {
		mask, err := emptyspace.Build(c.bounds, c.cells, c.fields, c.policy)
		if err == nil || mask != nil ||
			!strings.HasPrefix(err.Error(), "emptyspace: ") ||
			!strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: Build gave %v, %v; want a refusal containing %q",
				c.name, mask, err, c.want)
		}
	}
}

// The mask reports its brick grid and its payload by hand-worked counts:
// sixteen samples in bricks of two make eight bricks an axis, or sixty-four
// bytes of bits; a grid of zeros keeps no bits.
func TestMaskReportsItsBricksAndPayload(t *testing.T) {
	mask := prepared(t, 1, 0, options())
	if got := mask.Dimensions(); got != [3]int{8, 8, 8} {
		t.Errorf("Dimensions is %v, want 8 8 8", got)
	}
	if got := mask.Bytes(); got != payloadBytes {
		t.Errorf("Bytes is %d, want %d", got, payloadBytes)
	}
	wide := options()
	wide.BrickSide = 3
	if got := prepared(t, 1, 0, wide).Dimensions(); got != [3]int{6, 6, 6} {
		t.Errorf("Dimensions with bricks of 3 is %v, want 6 6 6", got)
	}
	empty, err := build(oneField(), options())
	if err != nil {
		t.Fatal(err)
	}
	if empty.Bytes() != 0 {
		t.Errorf("a grid of zeros keeps %d bytes, want none", empty.Bytes())
	}
}

// The fingerprint is the same for the same input and differs when a sample,
// the brick policy or the bounds differ.
func TestFingerprintFollowsItsInputs(t *testing.T) {
	base := prepared(t, 1, 0, options()).Fingerprint()
	if again := prepared(t, 1, 0, options()).Fingerprint(); again != base {
		t.Error("the same input gave two fingerprints")
	}
	if prepared(t, 2, 0, options()).Fingerprint() == base {
		t.Error("a different sample kept the fingerprint")
	}
	wide := options()
	wide.BrickSide = 4
	if prepared(t, 1, 0, wide).Fingerprint() == base {
		t.Error("a different brick side kept the fingerprint")
	}
	moved := maths.Box{Min: maths.Vec{}, Max: maths.V(2, 1, 1)}
	other, err := emptyspace.Build(moved, [3]int{samples, samples, samples},
		oneField(), options())
	if err != nil {
		t.Fatal(err)
	}
	flat, err := build(oneField(), options())
	if err != nil {
		t.Fatal(err)
	}
	if other.Fingerprint() == flat.Fingerprint() {
		t.Error("different bounds kept the fingerprint")
	}
}

// A field with several channels is searched in every channel.
func TestAChannelOtherThanTheFirstKeepsSupport(t *testing.T) {
	values := make([]float32, samples*samples*samples*2)
	at := ((centreSample*samples+centreSample)*samples + centreSample) * 2
	values[at+1] = 1
	mask, err := emptyspace.Build(cube, [3]int{samples, samples, samples},
		[]emptyspace.Field{{Values: values, Channels: 2}}, options())
	if err != nil {
		t.Fatal(err)
	}
	expectBox(t, mask, maths.Box{Min: maths.V(.5, .5, .5),
		Max: maths.V(.5, .5, .5)}, false)
}

// A query that is no box, or asked of a mask that was never built, is
// refused.
func TestEmptyBoxRefusesWhatIsNoQuery(t *testing.T) {
	mask := prepared(t, 1, 0, options())
	bad := []maths.Box{
		{Min: maths.V(math.NaN(), 0, 0), Max: maths.V(1, 1, 1)},
		{Min: maths.V(1, 1, 1), Max: maths.Vec{}},
		{Min: maths.Vec{}, Max: maths.V(math.Inf(1), 1, 1)},
	}
	for _, box := range bad {
		if got, err := mask.EmptyBox(box); err == nil || got {
			t.Errorf("EmptyBox(%v) = %v, %v; want a refusal", box, got, err)
		}
	}
	var none *emptyspace.Mask
	if got, err := none.EmptyBox(cube); err == nil || got {
		t.Errorf("a nil mask answered %v, %v", got, err)
	}
	if got, err := (&emptyspace.Mask{}).EmptyBox(cube); err == nil || got {
		t.Errorf("a mask never built answered %v, %v", got, err)
	}
}

// PacketEmpty refuses a packet that is no packet, a ray that is no ray and
// an interval that is no interval.
func TestPacketEmptyRefusesWhatIsNoPacket(t *testing.T) {
	mask := prepared(t, 0, 0, options())
	ray := maths.Ray{At: maths.V(.5, .5, .5), Dir: maths.XAxis}
	huge := maths.Ray{At: maths.V(.5, .5, .5), Dir: maths.V(1e308, 0, 0)}
	cases := []struct {
		name       string
		rays       []maths.Ray
		begin, end float64
	}{
		{"no rays", nil, 0, 1},
		{"more rays than the policy", []maths.Ray{ray, ray, ray}, 0, 1},
		{"a NaN start", []maths.Ray{ray}, math.NaN(), 1},
		{"an infinite end", []maths.Ray{ray}, 0, math.Inf(1)},
		{"a start below zero", []maths.Ray{ray}, -1, 1},
		{"an end before the start", []maths.Ray{ray}, 1, 0.5},
		{"a ray with no direction", []maths.Ray{{At: maths.Vec{}}}, 0, 1},
		{"a ray from a NaN point", []maths.Ray{{At: maths.V(math.NaN(), 0, 0),
			Dir: maths.XAxis}}, 0, 1},
		{"a ray that overflows", []maths.Ray{huge}, 0, 1e300},
	}
	for _, c := range cases {
		if got, err := mask.PacketEmpty(c.rays, c.begin, c.end); err == nil || got {
			t.Errorf("%s: PacketEmpty = %v, %v; want a refusal", c.name, got, err)
		}
	}
	var none *emptyspace.Mask
	if got, err := none.PacketEmpty([]maths.Ray{ray}, 0, 1); err == nil || got {
		t.Errorf("a nil mask answered %v, %v", got, err)
	}
	if got, err := mask.PacketEmpty([]maths.Ray{ray}, 0, .125); err != nil || !got {
		t.Errorf("a short segment of an empty grid gave %v, %v; want empty",
			got, err)
	}
}

// SafePrefix finds the longest run of existing samples that cannot touch
// occupied support. The ray runs along x through the occupied middle at
// sixteen samples a unit: the brick of its last sample plus the one
// beside it must stay clear of bricks three and four, so the last sample
// is below a quarter, and four samples is the most.
func TestSafePrefixStopsBeforeOccupiedSupport(t *testing.T) {
	mask := prepared(t, 1, 0, options())
	runs := []emptyspace.Run{{Ray: maths.Ray{At: maths.V(0, .5, .5),
		Dir: maths.XAxis}, First: 0, Stride: .0625, Count: 16}}
	got, err := mask.SafePrefix(runs, 16)
	if err != nil || got != 4 {
		t.Errorf("SafePrefix of 16 = %d, %v; want 4", got, err)
	}
	got, err = mask.SafePrefix(runs, 4)
	if err != nil || got != 4 {
		t.Errorf("SafePrefix of an empty 4 = %d, %v; want 4", got, err)
	}
	got, err = mask.SafePrefix(runs, 0)
	if err != nil || got != 0 {
		t.Errorf("SafePrefix of 0 = %d, %v; want 0", got, err)
	}
}

// SafePrefix refuses a packet or a run that cannot be checked.
func TestSafePrefixRefusesWhatCannotBeChecked(t *testing.T) {
	mask := prepared(t, 0, 0, options())
	good := emptyspace.Run{Ray: maths.Ray{At: maths.V(.5, .5, .5),
		Dir: maths.XAxis}, First: 0, Stride: .125, Count: 4}
	with := func(change func(*emptyspace.Run)) []emptyspace.Run {
		made := good
		change(&made)
		return []emptyspace.Run{made}
	}
	cases := []struct {
		name      string
		runs      []emptyspace.Run
		suggested int
	}{
		{"no runs", nil, 2},
		{"more runs than the policy", []emptyspace.Run{good, good, good}, 2},
		{"a negative prefix", []emptyspace.Run{good}, -1},
		{"a run with no direction", with(func(r *emptyspace.Run) {
			r.Ray.Dir = maths.Vec{}
		}), 2},
		{"a run from a NaN point", with(func(r *emptyspace.Run) {
			r.Ray.At = maths.V(math.NaN(), 0, 0)
		}), 2},
		{"a run with a NaN direction", with(func(r *emptyspace.Run) {
			r.Ray.Dir = maths.V(math.NaN(), 0, 0)
		}), 2},
		{"a run starting below zero", with(func(r *emptyspace.Run) {
			r.First = -1
		}), 2},
		{"a run with no stride", with(func(r *emptyspace.Run) {
			r.Stride = 0
		}), 2},
		{"a run with a NaN start", with(func(r *emptyspace.Run) {
			r.First = math.NaN()
		}), 2},
		{"a run with a NaN stride", with(func(r *emptyspace.Run) {
			r.Stride = math.NaN()
		}), 2},
		{"a run of negative length", with(func(r *emptyspace.Run) {
			r.Count = -1
		}), 2},
		{"a run whose interval overflows", with(func(r *emptyspace.Run) {
			r.Stride, r.Count = 1e308, 4
		}), 4},
		{"a run whose endpoint overflows", with(func(r *emptyspace.Run) {
			r.Ray.Dir, r.First = maths.V(1e308, 0, 0), 10
		}), 4},
	}
	for _, c := range cases {
		if got, err := mask.SafePrefix(c.runs, c.suggested); err == nil || got != 0 {
			t.Errorf("%s: SafePrefix = %d, %v; want a refusal", c.name, got, err)
		}
	}
	var none *emptyspace.Mask
	if got, err := none.SafePrefix([]emptyspace.Run{good}, 2); err == nil || got != 0 {
		t.Errorf("a nil mask answered %d, %v", got, err)
	}
}
