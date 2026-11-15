//go:build draw && darwin

package gpu

import (
	"errors"
	"fmt"
	"math"
	"strings"
	"testing"

	"github.com/janearc/slayer/emit"
	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/noise"
	"github.com/janearc/slayer/metal"
	"github.com/janearc/slayer/scene/medium"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
	slayer "github.com/janearc/slayer/trace"
)

// the kernels that ask the wake's functions one term at a time, a point a
// thread, of the card the third buffer holds; probe_wake_span asks the span
// of the first volume of the encoded scene in the third buffer, two points
// a thread, a ray's start and its way, and gives back its enter and leave.
const probeWake = `
kernel void probe_wake(device const packed_float3 *points [[buffer(0)]],
                       device float *out [[buffer(1)]],
                       device const float *card [[buffer(2)]],
                       uint id [[thread_position_in_grid]]) {
	out[id] = wake_density(card, float3(points[id]));
}

kernel void probe_wake_fractal(device const packed_float3 *points [[buffer(0)]],
                       device float *out [[buffer(1)]],
                       device const float *card [[buffer(2)]],
                       uint id [[thread_position_in_grid]]) {
	out[id] = wake_fractal(card, float3(points[id]));
}

kernel void probe_wake_edge(device const packed_float3 *points [[buffer(0)]],
                       device float *out [[buffer(1)]],
                       device const float *card [[buffer(2)]],
                       uint id [[thread_position_in_grid]]) {
	out[id] = wake_edge(card, length(float3(points[id])));
}

kernel void probe_wake_away(device const packed_float3 *points [[buffer(0)]],
                       device float *out [[buffer(1)]],
                       device const float *card [[buffer(2)]],
                       uint id [[thread_position_in_grid]]) {
	out[id] = wake_away(card, float3(points[id]));
}

kernel void probe_wake_swept(device const packed_float3 *points [[buffer(0)]],
                       device float *out [[buffer(1)]],
                       device const float *card [[buffer(2)]],
                       uint id [[thread_position_in_grid]]) {
	out[id] = wake_clearing(card, wake_away(card, float3(points[id])));
}

kernel void probe_wake_ridge(device const packed_float3 *points [[buffer(0)]],
                       device float *out [[buffer(1)]],
                       device const float *card [[buffer(2)]],
                       uint id [[thread_position_in_grid]]) {
	out[id] = wake_banks(card, wake_away(card, float3(points[id])));
}

kernel void probe_wake_span(device const packed_float3 *points [[buffer(0)]],
                       device float *out [[buffer(1)]],
                       device const Volume *volumes [[buffer(2)]],
                       uint id [[thread_position_in_grid]]) {
	out[id] = 0;
	if (id % 2 == 0) {
		float2 got = span(volumes[0], float3(points[id]),
		                  float3(points[id + 1]), 0, nowhere);
		out[id] = got.x;
		out[id + 1] = got.y;
	}
}
`

// wakeProbed asks a probe kernel of the wake, or skips the test when there
// is no gpu.
func wakeProbed(t *testing.T, name string, params []byte,
	points [][3]float32) []float32 {
	t.Helper()
	kernel, err := metal.Open(source()+probeWake, name)
	if errors.Is(err, metal.ErrNoDevice) {
		t.Skip("no gpu here")
	}
	if err != nil {
		t.Fatal(err)
	}
	defer kernel.Close()
	got, err := kernel.Points(points, params)
	if err != nil {
		t.Fatal(err)
	}
	return got
}

// wakeSet is a wake to build: its numbers, the noise's, and the radius and
// space of each of its paths.
type wakeSet struct {
	name                string
	reach, clear, thick float64
	seed                uint64
	scale               float64
	octaves             int
	spaces              []maths.Subspace
	radii               []float64
}

// wakeOf is the wake of a set, through the library's constructors.
func wakeOf(t *testing.T, set wakeSet) *medium.Wake {
	t.Helper()
	var paths []*medium.Path
	for k, space := range set.spaces {
		path, err := medium.NewPath(space, set.radii[k])
		if err != nil {
			t.Fatal(err)
		}
		paths = append(paths, path)
	}
	grain, err := medium.NewGrain(set.seed, set.scale, set.octaves)
	if err != nil {
		t.Fatal(err)
	}
	made, err := medium.NewWake(set.reach, set.clear, set.thick, grain, paths)
	if err != nil {
		t.Fatal(err)
	}
	return made
}

// orbitalSet is the nebula as the orbital command builds it: 5200 thousand
// kilometres across its reach, a channel of 384 about each of two bands'
// paths, 1840 and 1472 out, each leaning 45 degrees from the lantern's
// axis toward the eye or away from it. thick is as given, since the command
// takes it from a flag.
func orbitalSet(thick float64) wakeSet {
	axis := maths.V(0.3, 1, 0.2).Unit()
	toEye := maths.V(0, 0, 1)
	across := toEye.Sub(axis.Scale(toEye.Dot(axis))).Unit()
	var spaces []maths.Subspace
	for _, lean := range []float64{45 * maths.Degree, -45 * maths.Degree} {
		tip := maths.Subspace{Orientation: maths.Rotation(across, lean)}
		spaces = append(spaces, maths.Subspace{
			Orientation: maths.Upright(tip.Out(axis))})
	}
	return wakeSet{name: "orbital", reach: 5200, clear: 384, thick: thick,
		seed: 1806, scale: 260, octaves: 5, spaces: spaces,
		radii: []float64{1840, 1472}}
}

// otherSets are two more wakes, with other numbers, other noise and other
// paths: three, one moved and scaled, in a ball of 900; and one wide path
// of a ball of 12000 over six octaves.
func otherSets() []wakeSet {
	return []wakeSet{
		{name: "three paths", reach: 900, clear: 60, thick: 2e-3, seed: 77,
			scale: 90, octaves: 3,
			spaces: []maths.Subspace{
				{Orientation: maths.Rotation(maths.V(0, 0, 1), 0.3)},
				{Origin: maths.V(40, -30, 10), Scale: 2,
					Orientation: maths.Rotation(maths.V(1, 1, 0).Unit(), 1.1)},
				{Orientation: maths.Rotation(maths.V(0, 1, 0), -0.7)}},
			radii: []float64{500, 300, 150}},
		{name: "one path", reach: 12000, clear: 1500, thick: 1e-5, seed: 5,
			scale: 700, octaves: 6,
			spaces: []maths.Subspace{
				{Orientation: maths.Rotation(maths.V(1, 0, 1).Unit(), 2.2)}},
			radii: []float64{6000}},
	}
}

// wakeProbes is how many points a wake's probes ask.
const wakeProbes = 20000

// wakePoints are count points of a wake as float32, as the kernel is given
// them: half of them seeded through its ball and a tenth beyond it, and half
// about its paths, within two and a half clears of one in every direction,
// where the channel and its banks are.
func wakePoints(each wakeSet, count int) [][3]float32 {
	points := make([][3]float32, count)
	for at := range points {
		k := int64(at)
		var point maths.Vec
		if at%2 == 0 {
			reach := each.reach * 1.1 * math.Cbrt(noise.Hash(1, k))
			cos := 2*noise.Hash(2, k) - 1
			sin := math.Sqrt(1 - cos*cos)
			turn := 2 * math.Pi * noise.Hash(3, k)
			point = maths.V(reach*sin*math.Cos(turn), reach*cos,
				reach*sin*math.Sin(turn))
		} else {
			which := int(noise.Hash(4, k) * float64(len(each.spaces)))
			turn := 2 * math.Pi * noise.Hash(5, k)
			out := each.radii[which] + each.clear*5*(noise.Hash(6, k)-0.5)
			up := each.clear * 5 * (noise.Hash(7, k) - 0.5)
			point = each.spaces[which].Out(maths.V(out*math.Cos(turn), up,
				out*math.Sin(turn)))
		}
		points[at] = [3]float32{float32(point.X), float32(point.Y),
			float32(point.Z)}
	}
	return points
}

// worldOf is a float32 point as the tracer is asked it.
func worldOf(point [3]float32) maths.Vec {
	return maths.V(float64(point[0]), float64(point[1]), float64(point[2]))
}

// wakeTerm is one of the pieces of the wake's density the kernel's functions
// give one at a time, each with the probe that asks it and the number the
// library's own pieces give: the noise as noise.Fractal reads the point over
// the grain's scale, the edge fade, the distance to the nearest path, the
// channel's clearing and its banks.
type wakeTerm struct {
	name, kernel string
	want         func(made *medium.Wake, point maths.Vec) float64
}

// awayFrom is the distance from the nearest path of a wake, as its paths and
// maths say it is: the circle at the path's radius in its plane.
func awayFrom(made *medium.Wake, point maths.Vec) float64 {
	nearest := math.Inf(1)
	for _, path := range made.Paths() {
		q := path.Space().In(point)
		away := math.Hypot(math.Hypot(q.X, q.Z)-path.Radius(), q.Y)
		nearest = math.Min(nearest, away)
	}
	return nearest
}

// wakeTerms lists the terms.
func wakeTerms() []wakeTerm {
	return []wakeTerm{
		{"fractal", "probe_wake_fractal", fractalOf},
		{"edge", "probe_wake_edge", edgeOf},
		{"away", "probe_wake_away", awayFrom},
		{"swept", "probe_wake_swept", sweptOf},
		{"ridge", "probe_wake_ridge", ridgeOf},
		{"density", "probe_wake", densityOf},
	}
}

// fractalOf is the grain's noise at a point.
func fractalOf(made *medium.Wake, point maths.Vec) float64 {
	grain := made.Grain()
	scale := grain.Scale()
	return noise.Fractal(grain.Seed(), point.X/scale, point.Y/scale,
		point.Z/scale, grain.Octaves())
}

// edgeOf is how much of the gas is left by the fade toward the edge.
func edgeOf(made *medium.Wake, point maths.Vec) float64 {
	return 1 - maths.Smooth(made.Fade()*made.Reach(), made.Reach(),
		point.Len())
}

// sweptOf is how much of the gas the channel leaves.
func sweptOf(made *medium.Wake, point maths.Vec) float64 {
	return maths.Smooth(made.Swept()*made.Clear(), made.Clear(),
		awayFrom(made, point))
}

// ridgeOf is the banks along the channel's edges, as many times the gas as
// they hold.
func ridgeOf(made *medium.Wake, point maths.Vec) float64 {
	width, bank := made.Ridge()
	across := (awayFrom(made, point) - made.Clear()) / (width * made.Clear())
	return bank * math.Exp(-across*across)
}

// densityOf is the wake's own density.
func densityOf(made *medium.Wake, point maths.Vec) float64 {
	return made.Density(point)
}

// wakeApart asks every term's probe of the wake's points, and says how far
// the kernel is from the library at each: the worst absolute difference of
// every term, and the worst of the density as a share of the wake's
// thick, which is where the gas's own density is read.
func wakeApart(t *testing.T, set wakeSet) map[string]float64 {
	t.Helper()
	made := wakeOf(t, set)
	card, err := wakeCardOf(made)
	if err != nil {
		t.Fatal(err)
	}
	points := wakePoints(set, wakeProbes)
	worst := map[string]float64{}
	for _, term := range wakeTerms() {
		got := wakeProbed(t, term.kernel, cardBytes(card), points)
		for at, point := range points {
			want := term.want(made, worldOf(point))
			if term.name == "density" {
				want /= set.thick
				got[at] /= float32(set.thick)
			}
			apart := math.Abs(float64(got[at]) - want)
			if term.name == "away" {
				apart /= set.reach
			}
			if !(apart <= worst[term.name]) {
				worst[term.name] = apart
			}
		}
	}
	return worst
}

// the kernel's wake is the library's, at 20000 points through each of three
// wakes' balls and about their paths, to a bound for every term: float32
// against float64, each bound seven to ten times the worst difference
// measured on 2026-11-10 (the test logs them; the command is
// GOMAXPROCS=4 go test -count=1 -run TestTheKernelsWakeIsTheLibrarys -v
// ./gpu/). the noise is 6e-6 at worst over five or six octaves, though its
// last octave reads a point 17 times as far out as the first; the distance
// to a path is one float32 step at the wake's reach, so it is held as a
// share of the reach; the density is asked as a share of the wake's thick,
// which is where its gas is read.
func TestTheKernelsWakeIsTheLibrarys(t *testing.T) {
	bounds := map[string]float64{"fractal": wakeFractalBound,
		"edge": wakeEdgeBound, "away": wakeAwayBound,
		"swept": wakeSweptBound, "ridge": wakeRidgeBound,
		"density": wakeDensityBound}
	sets := append([]wakeSet{orbitalSet(4.5e-5)}, otherSets()...)
	for _, set := range sets {
		t.Run(set.name, func(t *testing.T) {
			worst := wakeApart(t, set)
			for _, term := range wakeTerms() {
				t.Logf("%s: worst %g, bound %g", term.name,
					worst[term.name], bounds[term.name])
				if !(worst[term.name] <= bounds[term.name]) {
					t.Errorf("the kernel's %s is %g from the library's, "+
						"want within %g", term.name, worst[term.name],
						bounds[term.name])
				}
			}
		})
	}
}

// the bounds each term is held to, from the measurements the test above
// states. measured worst over the three wakes: fractal 5.7e-6, edge 3.2e-7,
// away 1.8e-7 of the reach, swept 2.5e-6, ridge 6.3e-6, density 1.1e-5.
const (
	wakeFractalBound = 5e-5
	wakeEdgeBound    = 3e-6
	wakeAwayBound    = 1.5e-6
	wakeSweptBound   = 2.5e-5
	wakeRidgeBound   = 6e-5
	wakeDensityBound = 1e-4
)

// wakeRays are count rays at a wake, a start and a way each, from seeded
// points about its ball toward seeded ways: some miss it, and some cross it.
func wakeRays(each wakeSet, count int) [][3]float32 {
	var points [][3]float32
	for at := range count {
		k := int64(at)
		start := maths.V(each.reach*3*(noise.Hash(11, k)-0.5),
			each.reach*3*(noise.Hash(12, k)-0.5),
			each.reach*3*(noise.Hash(13, k)-0.5))
		way := maths.V(noise.Hash(14, k)-0.5, noise.Hash(15, k)-0.5,
			noise.Hash(16, k)-0.5).Unit()
		points = append(points,
			[3]float32{float32(start.X), float32(start.Y), float32(start.Z)},
			[3]float32{float32(way.X), float32(way.Y), float32(way.Z)})
	}
	return points
}

// wakeSpanned encodes a wake as a world's one volume and gives back the
// volume's bytes as the kernel reads them.
func wakeSpanned(t *testing.T, set wakeSet) []byte {
	t.Helper()
	world := thing.NewWorld()
	world.AddVolume(thing.Volume{Medium: wakeOf(t, set),
		Material: &thing.Material{Name: "gas", Colour: maths.Grey(0.5)}})
	encoded, err := encode(world, maths.Subspace{}, film.Pinhole{Wide: 1}, 0,
		2, 2)
	if err != nil {
		t.Fatal(err)
	}
	return encoded.volumes.bytes
}

// the kernel's span of a wake is its ball's, as the library's Span has it:
// the rays that cross it enter and leave where the library's do, those that
// miss it leave no span, and a ray from outside the ball is not given the
// box's longer one.
func TestTheKernelsWakeSpanIsItsBall(t *testing.T) {
	for _, set := range append([]wakeSet{orbitalSet(4.5e-5)}, otherSets()...) {
		t.Run(set.name, func(t *testing.T) {
			made := wakeOf(t, set)
			rays := wakeRays(set, 4000)
			got := wakeProbed(t, "probe_wake_span", wakeSpanned(t, set), rays)
			crossed, worst := 0, 0.0
			for at := 0; at < len(rays); at += 2 {
				ray := maths.Ray{At: worldOf(rays[at]), Dir: worldOf(rays[at+1])}
				enter, leave, ok := made.Span(ray, 0, nowhere)
				kept := float64(got[at+1]) > float64(got[at])
				if ok != kept {
					t.Fatalf("ray %d: the library says %v, the kernel says "+
						"%v, from %v to %v", at/2, ok, kept, got[at],
						got[at+1])
				}
				if !ok {
					continue
				}
				crossed++
				worst = math.Max(worst, math.Abs(float64(got[at])-enter)/
					set.reach)
				worst = math.Max(worst, math.Abs(float64(got[at+1])-leave)/
					set.reach)
			}
			t.Logf("%d of 4000 rays cross the ball; the worst end is %g of "+
				"its reach off", crossed, worst)
			if crossed < 500 || !(worst <= wakeSpanBound) {
				t.Errorf("%d rays cross, and the worst end is %g of the "+
					"reach off, want 500 and within %g", crossed, worst,
					wakeSpanBound)
			}
		})
	}
}

// wakeSpanBound is how far an end of the kernel's span may be from the
// library's, as a share of the reach: measured at 1.4e-6 at most over 1072
// rays that cross each of three balls, float32's step at the ball's size.
const wakeSpanBound = 1e-5

// the wake is seen from outside, a distance of two and a half reaches along
// z, a little above, through a lens 40 degrees wide, 96 by 64.
const (
	wakeWide = 96
	wakeHigh = 64
)

// wakeAgainst draws a world both ways from an eye a distance out from a
// ball, one ray a pixel with nothing baked, and fails when they are
// further apart than a tolerance.
func wakeAgainst(t *testing.T, world *thing.World, reach float64,
	allowed tolerance) {
	t.Helper()
	from := maths.V(0, 0.3*reach, -2.5*reach)
	eye := maths.Subspace{Origin: from,
		Orientation: maths.LookAt(from, maths.Vec{})}
	baking := bakeSide
	bakeSide = 0
	defer func() { bakeSide = baking }()
	lens := film.Pinhole{Wide: 40 * maths.Degree}
	drawn, err := Sampled(world, eye, lens, 0, wakeWide, wakeHigh, 1)
	if errors.Is(err, metal.ErrNoDevice) {
		t.Skip("no gpu here")
	}
	if err != nil {
		t.Fatal(err)
	}
	truth, err := slayer.Sampled(world, eye, lens, 0, wakeWide, wakeHigh, 1)
	if err != nil {
		t.Fatal(err)
	}
	found := compare(truth, drawn, wakeWide, wakeHigh)
	t.Logf("apart: %+v", found)
	for _, problem := range findings(found, allowed) {
		t.Error(problem)
	}
}

// wakeLit is a wake of a set lit by a lamp far off, with the material given.
func wakeLit(t *testing.T, set wakeSet, material *thing.Material) *thing.World {
	t.Helper()
	world := thing.NewWorld()
	world.AddVolume(thing.Volume{Medium: wakeOf(t, set), Material: material})
	star, err := emit.NewDistant(maths.V(1, 0.4, -0.3), maths.RGB(3, 2.8, 2.5))
	if err != nil {
		t.Fatal(err)
	}
	world.Lamp(star)
	return world
}

// wakeThick is a wake's thick so that its gas is about one mean free path
// through its middle, the noise cubed and lifted taking about a tenth of it.
func wakeThick(reach float64) float64 { return 10 / reach }

// nebulaGas is a nebula's look: dim, reddish, its own colour only.
func nebulaGas() *thing.Material {
	return &thing.Material{Name: "nebula", Colour: maths.RGB(0.22, 0.14, 0.17)}
}

// glowingGas is the same gas with a glow of its own, as orbital's nebula
// has from its hydrogen and neon.
func glowingGas() *thing.Material {
	return &thing.Material{Name: "nebula", Colour: maths.RGB(0.22, 0.14, 0.17),
		Glow: maths.RGB(0.4, 0.1, 0.07)}
}

// the wake as orbital builds it, alone, lit by a lamp far off: the kernel
// draws the gas as the tracer does.
func TestTheGPUDrawsTheWakeAsSlayerDoes(t *testing.T) {
	set := orbitalSet(wakeThick(5200))
	wakeAgainst(t, wakeLit(t, set, nebulaGas()), set.reach, wakeTolerance)
}

// the same wake with a glow of its own, as the nebula has.
func TestTheGPUDrawsAGlowingWakeAsSlayerDoes(t *testing.T) {
	set := orbitalSet(wakeThick(5200))
	wakeAgainst(t, wakeLit(t, set, glowingGas()), set.reach, wakeTolerance)
}

// a ball inside the wake, between the lamp and some of its gas: the shadow
// the ball casts through the gas falls where the tracer's does.
func TestTheGPUDrawsABallsShadowThroughTheWakeAsSlayerDoes(t *testing.T) {
	set := orbitalSet(wakeThick(5200))
	world := wakeLit(t, set, nebulaGas())
	world.Add(thing.Thing{Shape: shape.Sphere{Radius: 0.15 * set.reach},
		Material: &thing.Material{Name: "ball", Colour: maths.Grey(0.6)},
		Motion: thing.Still(maths.Subspace{
			Origin: maths.V(0.1*set.reach, 0.05*set.reach, 0.05*set.reach)})})
	wakeAgainst(t, world, set.reach, wakeTolerance)
}

// the wake in another set of numbers, the smaller of the others, at its own
// scale: three paths, one moved and scaled.
func TestTheGPUDrawsAnotherWakeAsSlayerDoes(t *testing.T) {
	set := otherSets()[0]
	set.thick = wakeThick(set.reach)
	wakeAgainst(t, wakeLit(t, set, glowingGas()), set.reach, wakeTolerance)
}

// wakeTolerance is what the wake is drawn to: the exact tolerance. measured on
// 2026-11-10 by the four tests above, no more than 8.7e-7 apart on average
// and 3.8e-6 at most, against the exact tolerance's 1e-5 and 1e-3.
var wakeTolerance = exact

// a wake of more paths than the kernel's card holds is refused by name,
// with the count and the most.
func TestAWakeOfTooManyPathsIsRefusedByName(t *testing.T) {
	set := orbitalSet(wakeThick(5200))
	for len(set.spaces) <= mostWakePaths {
		set.spaces = append(set.spaces, set.spaces[0])
		set.radii = append(set.radii, set.radii[0])
	}
	world := wakeLit(t, set, nebulaGas())
	_, err := encode(world, maths.Subspace{}, film.Pinhole{Wide: 1}, 0, 2, 2)
	var refused unsupported
	if !errors.As(err, &refused) {
		t.Fatalf("a wake of %d paths gave %v, want an unsupported error",
			len(set.spaces), err)
	}
	for _, part := range []string{"wake", fmt.Sprint(len(set.spaces)),
		fmt.Sprint(mostWakePaths)} {
		if !strings.Contains(err.Error(), part) {
			t.Errorf("the refusal %q does not name %q", err, part)
		}
	}
}

// the wakes the kernel asks itself are drawn with no box sampled on the cpu;
// a wake whose gas shines by a texture, which only a box can hold, is boxed
// as any medium is, once, and the count says so.
func TestAWakeIsBoxedOnlyWhenItsGasShinesByATexture(t *testing.T) {
	set := orbitalSet(wakeThick(5200))
	eye := maths.Subspace{Origin: maths.V(0, 0, -3*set.reach)}
	lens := film.Pinhole{Wide: 40 * maths.Degree}
	for _, each := range []struct {
		name     string
		material *thing.Material
		boxed    int
	}{{"plain", nebulaGas(), 0},
		{"glowing", glowingGas(), 0},
		{"textured", &thing.Material{Name: "banded", Shine: banded{}}, 1}} {
		_, spent, err := sampledTimed(wakeLit(t, set, each.material), nil,
			eye, lens, 0, 8, 8, 1)
		if errors.Is(err, metal.ErrNoDevice) {
			t.Skip("no gpu here")
		}
		if err != nil {
			t.Fatal(err)
		}
		if got := spent.boxedCPU.Volumes; got != each.boxed {
			t.Errorf("%s: %d volumes boxed, want %d", each.name, got,
				each.boxed)
		}
	}
}
