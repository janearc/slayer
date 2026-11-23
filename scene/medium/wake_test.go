package medium_test

import (
	"fmt"
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/noise"
	"github.com/janearc/slayer/scene/medium"
)

// What the wake tests count in, each true by definition.
const (
	wakeHalf      = 0.5 // defined: takes noise to nought to one
	wakeFirstPath = 0   // defined: the first of a wake's paths
)

// The csv files the wake tests read their cases, paths and refusals from.
const (
	wakeCasesFile         = "testdata/wake-cases.csv"
	wakePathsFile         = "testdata/wake-paths.csv"
	wakeRefusalsFile      = "testdata/wake-refusals.csv"
	wakeGrainRefusalsFile = "testdata/wake-grain-refusals.csv"
	wakePathRefusalsFile  = "testdata/wake-path-refusals.csv"
)

// wakeShape is the shape of a wake's gas as the kernel reads it: where the
// fade starts as a share of the reach, where the swept channel starts as a
// share of the clear, the width of the ridge and its bank, and the lift of
// the cube of the raggedness.
type wakeShape struct{ fade, swept, width, bank, lift float64 }

// wakeShapeOf is the shape the library's constants are the numbers of.
func wakeShapeOf(t *testing.T) wakeShape {
	t.Helper()
	return wakeShape{
		fade:  picked(t, "wake-fade"),
		swept: picked(t, "wake-swept"),
		width: picked(t, "wake-ridge-width"),
		bank:  picked(t, "wake-ridge-bank"),
		lift:  picked(t, "wake-lift"),
	}
}

// longhandPath and longhandWake are the wake written out a second way, every
// number in place: the arithmetic the library's Wake must agree with. The
// orbital case's noise is 1806, 260 and 5, so that case also shows the
// library's constants are the numbers written here.
type longhandPath struct {
	space  maths.Subspace
	radius float64
}

// longhandWake is a wake whose density is worked in the open, from its
// numbers, its shape and its paths.
type longhandWake struct {
	reach, clear, thick float64
	shape               wakeShape
	paths               []longhandPath
	// the noise, written as seed, scale and octaves in place
	seed    uint64
	scale   float64
	octaves int
}

// Bounds is the cube the wake's ball just fits in.
func (wake longhandWake) Bounds() maths.Box {
	reach := wake.reach
	return maths.Box{Min: maths.V(-reach, -reach, -reach),
		Max: maths.V(reach, reach, reach)}
}

// Span is where a ray is inside the wake's ball, between near and far.
func (wake longhandWake) Span(ray maths.Ray, near, far float64) (float64,
	float64, bool) {
	along := ray.At.Dot(ray.Dir)
	beyond := ray.At.Dot(ray.At) - wake.reach*wake.reach
	disc := along*along - beyond
	if disc < nought {
		return nought, nought, false
	}
	root := math.Sqrt(disc)
	enter, leave := math.Max(-along-root, near), math.Min(-along+root, far)
	return enter, leave, leave > enter
}

// fromPaths is how far a point is from the nearest of the wake's paths.
func (wake longhandWake) fromPaths(point maths.Vec) float64 {
	nearest := math.Inf(unity)
	for _, one := range wake.paths {
		local := one.space.In(point)
		away := math.Hypot(math.Hypot(local.X, local.Z)-one.radius, local.Y)
		nearest = math.Min(nearest, away)
	}
	return nearest
}

// Density is the wake's gas at a point: the raggedness cubed, faded towards
// the edge of the ball, and swept clear about the paths but for a ridge.
func (wake longhandWake) Density(point maths.Vec) float64 {
	if point.Len() > wake.reach {
		return nought
	}
	rag := math.Max(nought, wakeHalf+wakeHalf*noise.Fractal(wake.seed,
		point.X/wake.scale, point.Y/wake.scale, point.Z/wake.scale,
		wake.octaves))
	base := wake.thick * rag * rag * rag * wake.shape.lift
	base *= unity - maths.Smooth(wake.shape.fade*wake.reach, wake.reach,
		point.Len())
	distance := wake.fromPaths(point)
	swept := maths.Smooth(wake.shape.swept*wake.clear, wake.clear, distance)
	across := (distance - wake.clear) / (wake.shape.width * wake.clear)
	ridge := math.Exp(-math.Pow(across, pair))
	return base * (swept + wake.shape.bank*ridge)
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

// wakeSpace is the space of the path in a row of the paths file: its origin,
// scale and the rotation about its axis, in degrees or in radians.
func wakeSpace(t *testing.T, row map[string]string) maths.Subspace {
	t.Helper()
	origin := maths.V(cell(t, row, "origin_x"), cell(t, row, "origin_y"),
		cell(t, row, "origin_z"))
	axis := maths.V(cell(t, row, "axis_x"), cell(t, row, "axis_y"),
		cell(t, row, "axis_z"))
	if row["axis"] == "unit" {
		axis = axis.Unit()
	}
	angle := cell(t, row, "angle")
	if row["angle_unit"] == "degree" {
		angle *= maths.Degree
	}
	return maths.NewSubspace(origin, maths.Rotation(axis, angle),
		cell(t, row, "scale"))
}

// wakeCaseNamed is the wake case called name in the cases file, with each of
// its paths from the paths file, named for the case and the path's place.
func wakeCaseNamed(t *testing.T, name string) wakeCase {
	t.Helper()
	row := readSheet(t, wakeCasesFile).row(t, wakeCasesFile, name)
	made := wakeCase{name: name, reach: cell(t, row, "reach"),
		clear: cell(t, row, "clear"), thick: cell(t, row, "thick"),
		seed: uint64(cellCount(t, row, "seed")), scale: cell(t, row, "scale"),
		octaves: cellCount(t, row, "octaves")}
	for place := range cellCount(t, row, "paths") {
		path := readSheet(t, wakePathsFile).row(t, wakePathsFile,
			fmt.Sprintf("%s-%d", name, place))
		made.spaces = append(made.spaces, wakeSpace(t, path))
		made.radii = append(made.radii, cell(t, path, "radius"))
	}
	return made
}

// orbitalWake is the wake as orbital builds it: 5200 reach, 384 clear, two
// bands' paths leaning 45 degrees either way, 1840 and 1472 out.
func orbitalWake(t *testing.T) wakeCase {
	t.Helper()
	return wakeCaseNamed(t, "orbital")
}

// otherWakes are two more: one small with three paths, one of them moved
// and scaled, and one wide with a single path of another noise.
func otherWakes(t *testing.T) []wakeCase {
	t.Helper()
	return []wakeCase{wakeCaseNamed(t, "three paths"),
		wakeCaseNamed(t, "one path")}
}

// built is the library's wake of a case, and the longhand one beside it.
func built(t *testing.T, each wakeCase) (*medium.Wake, longhandWake) {
	t.Helper()
	var paths []*medium.Path
	var longhand []longhandPath
	for place, space := range each.spaces {
		path, err := medium.NewPath(space, each.radii[place])
		if err != nil {
			t.Fatal(err)
		}
		paths = append(paths, path)
		longhand = append(longhand, longhandPath{space: space,
			radius: each.radii[place]})
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
	return made, longhandWake{reach: each.reach, clear: each.clear,
		thick: each.thick, shape: wakeShapeOf(t), paths: longhand,
		seed: each.seed, scale: each.scale, octaves: each.octaves}
}

// wakeSeededPoint is the index-th seeded point for a wake: out to beyond
// times its reach from the middle, so some are outside the ball.
func wakeSeededPoint(t *testing.T, each wakeCase, index int64) maths.Vec {
	t.Helper()
	seedReach := uint64(pickedCount(t, "wake-points-seed-reach"))
	seedCos := uint64(pickedCount(t, "wake-points-seed-cos"))
	seedTurn := uint64(pickedCount(t, "wake-points-seed-turn"))
	reach := each.reach * picked(t, "wake-points-beyond") *
		math.Cbrt(noise.Hash(seedReach, index))
	cos := pair*noise.Hash(seedCos, index) - unity
	sin := math.Sqrt(unity - cos*cos)
	turn := pair * math.Pi * noise.Hash(seedTurn, index)
	return maths.V(reach*sin*math.Cos(turn), reach*cos,
		reach*sin*math.Sin(turn))
}

// wakePathPoints are the points on each path of a wake, and above each on
// the ridge of the channel, around it in as many steps as the tests chose.
func wakePathPoints(t *testing.T, each wakeCase) []maths.Vec {
	t.Helper()
	var points []maths.Vec
	around := pickedCount(t, "wake-points-around")
	for place, space := range each.spaces {
		for turn := range around {
			angle := pair * math.Pi * float64(turn) / float64(around)
			round := maths.V(each.radii[place]*math.Cos(angle), nought,
				each.radii[place]*math.Sin(angle))
			points = append(points, space.Out(round))
			up := maths.V(round.X, each.clear, round.Z)
			points = append(points, space.Out(up))
		}
	}
	return points
}

// wakePoints are points for a wake: seeded ones inside its ball and outside
// it, one on its edge, one on each path, and some on the channel's ridge.
func wakePoints(t *testing.T, each wakeCase, count int) []maths.Vec {
	t.Helper()
	var points []maths.Vec
	for place := range count {
		index := int64(place)
		seeded := wakeSeededPoint(t, each, index)
		points = append(points, seeded)
	}
	points = append(points, maths.V(each.reach, nought, nought),
		maths.V(nought, -each.reach, nought), maths.V(nought, nought, nought))
	return append(points, wakePathPoints(t, each)...)
}

// wakeCaseRun is a test of one wake case, to be run under its name.
type wakeCaseRun struct{ each wakeCase }

// matchesLonghand holds that the case's wake is its longhand.
func (run wakeCaseRun) matchesLonghand(t *testing.T) {
	made, longhand := built(t, run.each)
	wakeIdentity(t, run.each, made, longhand)
}

// the library's wake gives the very bits of the longhand one's density at
// points inside, outside, on its edge, on a path and on the ridge of its
// channel, and its box, and agrees with its spans for rays that miss, graze,
// cross and are cut by near and far.
func TestWakeMatchesItsLonghand(t *testing.T) {
	cases := append([]wakeCase{orbitalWake(t)}, otherWakes(t)...)
	for _, each := range cases {
		t.Run(each.name, wakeCaseRun{each}.matchesLonghand)
	}
}

// wakeDensities holds that the wake and its longhand give the very bits of
// one density at every point of the case, and that enough points have gas.
func wakeDensities(t *testing.T, each wakeCase, made *medium.Wake,
	longhand longhandWake) {
	t.Helper()
	nonzero := nought
	count := pickedCount(t, "wake-identity-points")
	for _, point := range wakePoints(t, each, count) {
		got, want := made.Density(point), longhand.Density(point)
		if math.Float64bits(got) != math.Float64bits(want) {
			t.Fatalf("density at %v is %v, want %v", point, got, want)
		}
		if want != nought {
			nonzero++
		}
	}
	if nonzero < pickedCount(t, "wake-identity-gas-least") {
		t.Errorf("only %d points have gas; the test asks nothing", nonzero)
	}
}

// wakeRaySeeds are the seeds of the hashes a ray's start, direction and near
// end are drawn from.
type wakeRaySeeds struct {
	startX, startY, startZ uint64
	dirX, dirY, dirZ       uint64
	near                   uint64
}

// wakeRaySeedsOf reads the seeds the tests chose for the rays.
func wakeRaySeedsOf(t *testing.T) wakeRaySeeds {
	t.Helper()
	return wakeRaySeeds{
		startX: uint64(pickedCount(t, "wake-ray-seed-start-x")),
		startY: uint64(pickedCount(t, "wake-ray-seed-start-y")),
		startZ: uint64(pickedCount(t, "wake-ray-seed-start-z")),
		dirX:   uint64(pickedCount(t, "wake-ray-seed-dir-x")),
		dirY:   uint64(pickedCount(t, "wake-ray-seed-dir-y")),
		dirZ:   uint64(pickedCount(t, "wake-ray-seed-dir-z")),
		near:   uint64(pickedCount(t, "wake-ray-seed-near")),
	}
}

// wakeRay is the index-th ray for a wake: from within a few reaches of the
// middle, in a direction drawn at random.
func wakeRay(t *testing.T, each wakeCase, seeds wakeRaySeeds,
	index int64) maths.Ray {
	t.Helper()
	reach := each.reach * picked(t, "wake-ray-spread")
	startX := noise.Hash(seeds.startX, index) - wakeHalf
	startY := noise.Hash(seeds.startY, index) - wakeHalf
	startZ := noise.Hash(seeds.startZ, index) - wakeHalf
	return maths.Ray{
		At: maths.V(reach*startX, reach*startY, reach*startZ),
		Dir: maths.V(noise.Hash(seeds.dirX, index)-wakeHalf,
			noise.Hash(seeds.dirY, index)-wakeHalf,
			noise.Hash(seeds.dirZ, index)-wakeHalf).Unit()}
}

// wakeSpans holds that the wake agrees with its longhand on the span of each
// of the rays the tests drew.
func wakeSpans(t *testing.T, each wakeCase, made *medium.Wake,
	longhand longhandWake) {
	t.Helper()
	seeds := wakeRaySeedsOf(t)
	farReaches := picked(t, "wake-ray-far")
	for place := range pickedCount(t, "wake-identity-rays") {
		index := int64(place)
		ray := wakeRay(t, each, seeds, index)
		near := each.reach * noise.Hash(seeds.near, index)
		far := each.reach * farReaches
		gotEnter, gotLeave, gotHit := made.Span(ray, near, far)
		wantEnter, wantLeave, wantHit := longhand.Span(ray, near, far)
		if !sameEnd(gotEnter, wantEnter) || !sameEnd(gotLeave, wantLeave) ||
			gotHit != wantHit {
			t.Fatalf("span of %v is %v %v %v, want %v %v %v", ray, gotEnter,
				gotLeave, gotHit, wantEnter, wantLeave, wantHit)
		}
	}
}

// wakeIdentity compares a wake with its longhand.
func wakeIdentity(t *testing.T, each wakeCase, made *medium.Wake,
	longhand longhandWake) {
	t.Helper()
	wakeDensities(t, each, made, longhand)
	if made.Bounds() != longhand.Bounds() {
		t.Errorf("bounds %v, want %v", made.Bounds(), longhand.Bounds())
	}
	wakeSpans(t, each, made, longhand)
}

// spanSlack is how far apart, as a share of the value, a span's end may be
// from the longhand one's. The library's span fuses its products as the
// other media's do and the longhand does not, so the last bits differ.
// defined: well above the last places of a float64 and well below the
// difference between one ray's span and another's.
const spanSlack = 1e-9

// sameEnd is whether a span's end is the longhand one's, within spanSlack.
func sameEnd(got, want float64) bool {
	return math.Abs(got-want) <= spanSlack*math.Abs(want)
}

// wakeRefusalGrain is the grain a refused wake is given, or none.
func wakeRefusalGrain(t *testing.T, kind string) *medium.Grain {
	t.Helper()
	if kind == "none" {
		return nil
	}
	seed := uint64(pickedCount(t, "wake-refusal-grain-seed"))
	scale := picked(t, "wake-refusal-grain-scale")
	octaves := pickedCount(t, "wake-refusal-grain-octaves")
	grain, err := medium.NewGrain(seed, scale, octaves)
	if err != nil {
		t.Fatal(err)
	}
	return grain
}

// wakeRefusalPaths are the paths a refused wake is given: one good path, none
// at all, or a list of one nil path.
func wakeRefusalPaths(t *testing.T, kind string) []*medium.Path {
	t.Helper()
	path, err := medium.NewPath(maths.Local(),
		picked(t, "wake-refusal-path-radius"))
	if err != nil {
		t.Fatal(err)
	}
	switch kind {
	case "none":
		return nil
	case "nil-path":
		return []*medium.Path{nil}
	}
	return []*medium.Path{path}
}

// a wake refuses what has no meaning, each by name and value: a reach, clear
// or thick that is not a finite number of the right sign, no path, a path
// that is nothing, no grain.
func TestWakeRefusesWhatIsNotAWake(t *testing.T) {
	for _, bad := range readSheet(t, wakeRefusalsFile) {
		made, err := medium.NewWake(cell(t, bad, "reach"),
			cell(t, bad, "clear"), cell(t, bad, "thick"),
			wakeRefusalGrain(t, bad["grain"]),
			wakeRefusalPaths(t, bad["paths"]))
		if err == nil || made != nil {
			t.Errorf("%s: got %v and %v, want nil and an error", bad["name"],
				made, err)
		}
	}
}

// a grain refuses a scale that is not a finite number above zero, and
// octaves that are not above zero.
func TestGrainRefusesWhatIsNotAGrain(t *testing.T) {
	seed := uint64(pickedCount(t, "wake-refusal-grain-seed"))
	for _, bad := range readSheet(t, wakeGrainRefusalsFile) {
		made, err := medium.NewGrain(seed, cell(t, bad, "scale"),
			cellCount(t, bad, "octaves"))
		if err == nil || made != nil {
			t.Errorf("%s: got %v and %v, want nil and an error", bad["name"],
				made, err)
		}
	}
}

// a path refuses a radius that is not a finite number above zero, a scale
// below zero, and an origin that is not a number.
func TestPathRefusesWhatIsNotAPath(t *testing.T) {
	for _, bad := range readSheet(t, wakePathRefusalsFile) {
		space := maths.Local()
		space.Scale = cell(t, bad, "scale")
		space.Origin = maths.V(cell(t, bad, "origin_x"), nought, nought)
		made, err := medium.NewPath(space, cell(t, bad, "radius"))
		if err == nil || made != nil {
			t.Errorf("%s: got %v and %v, want nil and an error", bad["name"],
				made, err)
		}
	}
}

// a wake tells what it was made of, as the kernel reads it: its numbers,
// grain and paths, and the shape of its gas.
func TestWakeTellsWhatItIsMadeOf(t *testing.T) {
	each := orbitalWake(t)
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
	for place, path := range paths {
		if path.Space() != each.spaces[place] ||
			path.Radius() != each.radii[place] {
			t.Errorf("path %d is %v at %v", place, path.Space(), path.Radius())
		}
	}
	shape := wakeShapeOf(t)
	width, bank := made.Ridge()
	if made.Fade() != shape.fade || made.Swept() != shape.swept ||
		width != shape.width || bank != shape.bank ||
		made.Lift() != shape.lift {
		t.Errorf("shape %v %v %v %v %v", made.Fade(), made.Swept(), width,
			bank, made.Lift())
	}
	paths[wakeFirstPath] = nil
	if made.Paths()[wakeFirstPath] == nil {
		t.Error("a reader's list is the wake's own")
	}
}
