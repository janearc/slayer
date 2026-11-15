package medium_test

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/noise"
	"github.com/janearc/slayer/scene/medium"
)

// oldPath and oldWake are the nebula as the orbital command wrote it before
// it moved here, written out again with every number as it stood: the
// arithmetic the library's Wake must give the bits of. the orbital case's
// noise is the command's own, 1806, 260 and 5, so that case also shows the
// library's numbers are the ones that were written in place.
type oldPath struct {
	space  maths.Subspace
	radius float64
}

type oldWake struct {
	reach, clear, thick float64
	paths               []oldPath
	// the noise, which the command wrote as 1806, 260 and 5 in place
	seed    uint64
	scale   float64
	octaves int
}

func (n oldWake) Bounds() maths.Box {
	r := n.reach
	return maths.Box{Min: maths.V(-r, -r, -r), Max: maths.V(r, r, r)}
}

func (n oldWake) Span(r maths.Ray, near, far float64) (float64, float64, bool) {
	b := r.At.Dot(r.Dir)
	c := r.At.Dot(r.At) - n.reach*n.reach
	disc := b*b - c
	if disc < 0 {
		return 0, 0, false
	}
	root := math.Sqrt(disc)
	enter, leave := math.Max(-b-root, near), math.Min(-b+root, far)
	return enter, leave, leave > enter
}

func (n oldWake) fromPaths(p maths.Vec) float64 {
	nearest := math.Inf(1)
	for _, one := range n.paths {
		q := one.space.In(p)
		away := math.Hypot(math.Hypot(q.X, q.Z)-one.radius, q.Y)
		nearest = math.Min(nearest, away)
	}
	return nearest
}

func (n oldWake) Density(p maths.Vec) float64 {
	if p.Len() > n.reach {
		return 0
	}
	rag := 0.5 + 0.5*noise.Fractal(n.seed, p.X/n.scale, p.Y/n.scale, p.Z/n.scale, n.octaves)
	base := n.thick * rag * rag * rag * 3
	base *= 1 - maths.Smooth(0.45*n.reach, n.reach, p.Len())
	d := n.fromPaths(p)
	swept := maths.Smooth(0.55*n.clear, n.clear, d)
	ridge := math.Exp(-math.Pow((d-n.clear)/(0.3*n.clear), 2))
	return base * (swept + 2.2*ridge)
}

// wakeCase is one wake to build both ways: its numbers, and the spaces and
// radii of its paths.
type wakeCase struct {
	name                string
	reach, clear, thick float64
	spaces              []maths.Subspace
	radii               []float64
	seed                uint64
	scale               float64
	octaves             int
}

// orbitalWake is the wake as orbital builds it: 5200 reach, 384 clear, two
// bands' paths leaning 45 degrees either way, 1840 and 1472 out.
func orbitalWake() wakeCase {
	lean := 45 * maths.Degree
	return wakeCase{name: "orbital", reach: 5200, clear: 384, thick: 4.5e-5,
		spaces: []maths.Subspace{
			{Orientation: maths.Rotation(maths.V(1, 0, 0), lean)},
			{Orientation: maths.Rotation(maths.V(1, 0, 0), -lean)}},
		radii: []float64{1840, 1472}, seed: 1806, scale: 260, octaves: 5}
}

// otherWakes are two more: one small with three paths, one of them moved
// and scaled, and one wide with a single path of another noise.
func otherWakes() []wakeCase {
	return []wakeCase{
		{name: "three paths", reach: 900, clear: 60, thick: 2e-3,
			spaces: []maths.Subspace{
				{Orientation: maths.Rotation(maths.V(0, 0, 1), 0.3)},
				{Origin: maths.V(40, -30, 10), Scale: 2,
					Orientation: maths.Rotation(maths.V(1, 1, 0).Unit(), 1.1)},
				{Orientation: maths.Rotation(maths.V(0, 1, 0), -0.7)}},
			radii: []float64{500, 300, 150}, seed: 77, scale: 90, octaves: 3},
		{name: "one path", reach: 12000, clear: 1500, thick: 1e-5,
			spaces: []maths.Subspace{
				{Orientation: maths.Rotation(maths.V(1, 0, 1).Unit(), 2.2)}},
			radii: []float64{6000}, seed: 5, scale: 700, octaves: 6},
	}
}

// built is the library's wake of a case, and the written-out one beside it.
func built(t *testing.T, each wakeCase) (*medium.Wake, oldWake) {
	t.Helper()
	var paths []*medium.Path
	var old []oldPath
	for k, space := range each.spaces {
		path, err := medium.NewPath(space, each.radii[k])
		if err != nil {
			t.Fatal(err)
		}
		paths = append(paths, path)
		old = append(old, oldPath{space: space, radius: each.radii[k]})
	}
	grain, err := medium.NewGrain(each.seed, each.scale, each.octaves)
	if err != nil {
		t.Fatal(err)
	}
	made, err := medium.NewWake(each.reach, each.clear, each.thick, grain,
		paths)
	if err != nil {
		t.Fatal(err)
	}
	return made, oldWake{reach: each.reach, clear: each.clear,
		thick: each.thick, paths: old, seed: each.seed, scale: each.scale,
		octaves: each.octaves}
}

// wakePoints are points for a wake: seeded ones inside its ball and outside
// it, one on its edge, one on each path, and some on the channel's ridge.
func wakePoints(each wakeCase, count int) []maths.Vec {
	var points []maths.Vec
	for k := range count {
		at := int64(k)
		reach := each.reach * 1.3 * math.Cbrt(noise.Hash(1, at))
		cos := 2*noise.Hash(2, at) - 1
		sin := math.Sqrt(1 - cos*cos)
		turn := 2 * math.Pi * noise.Hash(3, at)
		points = append(points, maths.V(reach*sin*math.Cos(turn),
			reach*cos, reach*sin*math.Sin(turn)))
	}
	points = append(points, maths.V(each.reach, 0, 0),
		maths.V(0, -each.reach, 0), maths.V(0, 0, 0))
	for k, space := range each.spaces {
		for turn := range 40 {
			angle := 2 * math.Pi * float64(turn) / 40
			round := maths.V(each.radii[k]*math.Cos(angle), 0,
				each.radii[k]*math.Sin(angle))
			points = append(points, space.Out(round))
			up := maths.V(round.X, each.clear, round.Z)
			points = append(points, space.Out(up))
		}
	}
	return points
}

// the library's wake gives the very bits the command's did, for its density
// at points inside, outside, on its edge, on a path and on the ridge of its
// channel, for its box, and for rays that miss, graze, cross and are cut by
// near and far.
func TestWakeIsBitIdenticalToTheCommandsNebula(t *testing.T) {
	cases := append([]wakeCase{orbitalWake()}, otherWakes()...)
	for _, each := range cases {
		t.Run(each.name, func(t *testing.T) {
			made, old := built(t, each)
			wakeIdentity(t, each, made, old)
		})
	}
}

// wakeIdentity compares a wake with its written-out copy.
func wakeIdentity(t *testing.T, each wakeCase, made *medium.Wake,
	old oldWake) {
	t.Helper()
	nonzero := 0
	for _, point := range wakePoints(each, 20000) {
		got, want := made.Density(point), old.Density(point)
		if math.Float64bits(got) != math.Float64bits(want) {
			t.Fatalf("density at %v is %v, want %v", point, got, want)
		}
		if want != 0 {
			nonzero++
		}
	}
	if nonzero < 1000 {
		t.Errorf("only %d points have gas; the test asks nothing", nonzero)
	}
	if made.Bounds() != old.Bounds() {
		t.Errorf("bounds %v, want %v", made.Bounds(), old.Bounds())
	}
	for k := range 5000 {
		at := int64(k)
		ray := maths.Ray{
			At: maths.V(each.reach*3*(noise.Hash(11, at)-0.5),
				each.reach*3*(noise.Hash(12, at)-0.5),
				each.reach*3*(noise.Hash(13, at)-0.5)),
			Dir: maths.V(noise.Hash(14, at)-0.5, noise.Hash(15, at)-0.5,
				noise.Hash(16, at)-0.5).Unit()}
		near, far := each.reach*noise.Hash(17, at), each.reach*4
		a, b, okay := made.Span(ray, near, far)
		c, d, other := old.Span(ray, near, far)
		if !sameEnd(a, c) || !sameEnd(b, d) || okay != other {
			t.Fatalf("span of %v is %v %v %v, want %v %v %v", ray, a, b,
				okay, c, d, other)
		}
	}
}

// coveredSlack is how far apart, as a share of the value, a span's end may
// be from the written-out copy's when the test is built for coverage.
// defined: a few units in the last place of a float64.
const coveredSlack = 1e-12

// sameEnd is whether a span's end is the written-out copy's: the very bits,
// in an ordinary build. A build for coverage puts counters between the
// library's statements and not the copy's, so the compiler fuses a multiply
// and an add in one and not the other, and the last bits differ; there the
// two are held within coveredSlack of each other.
func sameEnd(got, want float64) bool {
	if testing.CoverMode() == "" {
		return math.Float64bits(got) == math.Float64bits(want)
	}
	return math.Abs(got-want) <= coveredSlack*math.Abs(want)
}

// a wake refuses what has no meaning, each by name and value: a reach, clear
// or thick that is not a finite number of the right sign, no path, a path
// that is nothing, no grain.
func TestWakeRefusesWhatIsNotAWake(t *testing.T) {
	grain, err := medium.NewGrain(1, 10, 2)
	if err != nil {
		t.Fatal(err)
	}
	path, err := medium.NewPath(maths.Local(), 5)
	if err != nil {
		t.Fatal(err)
	}
	paths := []*medium.Path{path}
	nan, inf := math.NaN(), math.Inf(1)
	for _, bad := range []struct {
		name                string
		reach, clear, thick float64
		grain               *medium.Grain
		paths               []*medium.Path
	}{
		{"zero reach", 0, 1, 1, grain, paths},
		{"negative reach", -1, 1, 1, grain, paths},
		{"nan reach", nan, 1, 1, grain, paths},
		{"infinite reach", inf, 1, 1, grain, paths},
		{"zero clear", 10, 0, 1, grain, paths},
		{"nan clear", 10, nan, 1, grain, paths},
		{"infinite clear", 10, inf, 1, grain, paths},
		{"negative thick", 10, 1, -1, grain, paths},
		{"nan thick", 10, 1, nan, grain, paths},
		{"no grain", 10, 1, 1, nil, paths},
		{"no paths", 10, 1, 1, grain, nil},
		{"a nil path", 10, 1, 1, grain, []*medium.Path{nil}},
	} {
		if made, err := medium.NewWake(bad.reach, bad.clear, bad.thick,
			bad.grain, bad.paths); err == nil || made != nil {
			t.Errorf("%s: got %v and %v, want nil and an error", bad.name,
				made, err)
		}
	}
}

// a grain refuses a scale that is not a finite number above zero, and
// octaves that are not above zero.
func TestGrainRefusesWhatIsNotAGrain(t *testing.T) {
	nan := math.NaN()
	for _, bad := range []struct {
		name    string
		scale   float64
		octaves int
	}{{"zero scale", 0, 2}, {"nan scale", nan, 2}, {"negative scale", -1, 2},
		{"no octaves", 10, 0}, {"negative octaves", 10, -1}} {
		if made, err := medium.NewGrain(1, bad.scale, bad.octaves); err == nil ||
			made != nil {
			t.Errorf("%s: got %v and %v, want nil and an error", bad.name,
				made, err)
		}
	}
}

// a path refuses a radius that is not a finite number above zero, a scale
// below zero, and an origin that is not a number.
func TestPathRefusesWhatIsNotAPath(t *testing.T) {
	nan := math.NaN()
	for _, bad := range []struct {
		name   string
		space  maths.Subspace
		radius float64
	}{{"zero radius", maths.Local(), 0},
		{"nan radius", maths.Local(), nan},
		{"negative scale", maths.Subspace{Scale: -1}, 5},
		{"nan origin", maths.Subspace{Origin: maths.V(nan, 0, 0)}, 5}} {
		if made, err := medium.NewPath(bad.space, bad.radius); err == nil ||
			made != nil {
			t.Errorf("%s: got %v and %v, want nil and an error", bad.name,
				made, err)
		}
	}
}

// a wake tells what it was made of, as the kernel reads it: its numbers,
// grain and paths, and the shape of its gas.
func TestWakeTellsWhatItIsMadeOf(t *testing.T) {
	each := orbitalWake()
	made, _ := built(t, each)
	if made.Reach() != each.reach || made.Clear() != each.clear ||
		made.Thick() != each.thick {
		t.Errorf("reach %v clear %v thick %v", made.Reach(), made.Clear(),
			made.Thick())
	}
	grain := made.Grain()
	if grain.Seed() != each.seed || grain.Scale() != each.scale ||
		grain.Octaves() != each.octaves {
		t.Errorf("grain %v %v %v", grain.Seed(), grain.Scale(), grain.Octaves())
	}
	paths := made.Paths()
	if len(paths) != len(each.spaces) {
		t.Fatalf("%d paths, want %d", len(paths), len(each.spaces))
	}
	for k, path := range paths {
		if path.Space() != each.spaces[k] || path.Radius() != each.radii[k] {
			t.Errorf("path %d is %v at %v", k, path.Space(), path.Radius())
		}
	}
	width, bank := made.Ridge()
	if made.Fade() != 0.45 || made.Swept() != 0.55 || width != 0.3 ||
		bank != 2.2 || made.Lift() != 3 {
		t.Errorf("shape %v %v %v %v %v", made.Fade(), made.Swept(), width,
			bank, made.Lift())
	}
	paths[0] = nil
	if made.Paths()[0] == nil {
		t.Error("a reader's list is the wake's own")
	}
}
