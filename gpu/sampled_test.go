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
	"github.com/janearc/slayer/metal"
	"github.com/janearc/slayer/render/frame"
	"github.com/janearc/slayer/scene/gas"
	"github.com/janearc/slayer/scene/medium"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/sky"
	"github.com/janearc/slayer/scene/thing"
	slayer "github.com/janearc/slayer/trace"
)

// floor is a glossy floor of squares, side by side, two faces each:
// enough faces for the tree to branch many times over
func floor(t *testing.T, side int) *shape.Mesh {
	var corners []maths.Vec
	var faces [][3]int
	middle := float64(side) / 2
	for row := range side + 1 {
		for column := range side + 1 {
			corner := maths.V(float64(column), 0, float64(row))
			corners = append(corners, corner.Sub(maths.V(middle, 0,
				middle)))
		}
	}
	for row := range side {
		for column := range side {
			near := row*(side+1) + column
			far := near + side + 1
			faces = append(faces, [3]int{near, far, near + 1},
				[3]int{near + 1, far, far + 1})
		}
	}
	mesh, err := shape.NewMesh(corners, faces, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	return mesh
}

// at is a thing standing still at a point
func at(x, y, z float64) thing.Motion {
	return thing.Still(maths.Subspace{Origin: maths.V(x, y, z)})
}

// hall is everything the kernel draws in one picture, each placed so
// it changes the picture: a glossy floor of 1,800 faces mirroring the
// star and a ball, the ball's shadow on the floor, a clear ball and a
// clear awning that cast none, a solid awning that does, a sky, and
// two volumes that overlap, one wispy, lit by a point emitter. the
// star's own clear flag changes nothing, since the lamp starts beyond
// its surface; the clear ball and awning are what test the flag.
func hall(t *testing.T) (*thing.World, maths.Subspace) {
	world := thing.NewWorld()
	world.Sky = sky.Stars{Dark: maths.RGB(0.01, 0.01, 0.02), Seed: 3}
	clear := thing.Attrs{Clear: true}
	star := &thing.Material{Name: "star", Glow: maths.RGB(2, 1.8, 1.5)}
	world.Add(thing.Thing{Shape: shape.Sphere{Radius: 1}, Material: star,
		Attrs: clear})
	ball := &thing.Material{Name: "ball", Colour: maths.RGB(0.3, 0.5, 0.7)}
	world.Add(thing.Thing{Shape: shape.Sphere{Radius: 0.5}, Material: ball,
		Motion: at(1.2, -0.6, 1.5)})
	world.Add(thing.Thing{Shape: shape.Sphere{Radius: 0.4}, Material: ball,
		Motion: at(-1.5, -0.7, 1.5), Attrs: clear})
	gloss := &thing.Material{Name: "floor", Colour: maths.Grey(0.4),
		Gloss: 0.8, Index: 2}
	world.Add(thing.Thing{Shape: floor(t, 30), Material: gloss,
		Motion: at(0, -1.5, 0)})
	awning := &thing.Material{Name: "awning", Colour: maths.Grey(0.6)}
	world.Add(thing.Thing{Shape: floor(t, 4), Material: awning,
		Motion: at(0, -0.8, -3)})
	world.Add(thing.Thing{Shape: floor(t, 2), Material: awning,
		Motion: at(3, -0.9, 0), Attrs: clear})
	world.Lamp(emit.Lit{Beyond: 1.02, Strength: maths.RGB(9, 8, 7)})
	haze := &thing.Material{Name: "haze", Colour: maths.RGB(0.9, 0.8, 0.7)}
	world.AddVolume(thing.Volume{Medium: medium.Wispy{
		Of: medium.Puff{Radius: 5, Thick: 0.1}, Seed: 3, Scale: 1.5,
		Amount: 0.5}, Material: haze})
	world.AddVolume(thing.Volume{Medium: medium.Puff{Radius: 3,
		Thick: 0.2}, Material: haze, Motion: at(-2, 0, 1)})
	from := maths.V(2, 5, 12)
	return world, maths.Subspace{Origin: from,
		Orientation: maths.LookAt(from, maths.Vec{})}
}

// orrery is spheres alone, so the tree is empty and the kernel must not
// walk it: a star and three balls round it, one clear
func orrery() (*thing.World, maths.Subspace) {
	world := thing.NewWorld()
	star := &thing.Material{Name: "star", Glow: maths.RGB(2, 1.8, 1.5)}
	world.Add(thing.Thing{Shape: shape.Sphere{Radius: 1}, Material: star,
		Attrs: thing.Attrs{Clear: true}})
	ball := &thing.Material{Name: "ball", Colour: maths.RGB(0.7, 0.5, 0.3),
		Gloss: 0.5}
	places := []maths.Vec{maths.V(3, 0, 0), maths.V(-2, 1, 2),
		maths.V(0, -2, -3)}
	for index, place := range places {
		world.Add(thing.Thing{Shape: shape.Sphere{Radius: 0.6},
			Material: ball, Motion: at(place.X, place.Y, place.Z),
			Attrs: thing.Attrs{Clear: index == 1}})
	}
	world.Lamp(emit.Lit{Beyond: 1.02, Strength: maths.RGB(9, 8, 7)})
	from := maths.V(1, 3, 10)
	return world, maths.Subspace{Origin: from,
		Orientation: maths.LookAt(from, maths.Vec{})}
}

// difference is how far apart two frames are: the mean difference in
// light as a share of the mean light, the largest difference in any one
// pixel as a share of the brightest, the pixels whose cover or depth
// disagree, the pixels whose light difference is not finite, and whether
// the truth frame has no light at all
type difference struct {
	mean, largest float64
	cover, depth  int
	notFinite     int    // pixels whose light difference is NaN or infinite
	first         [2]int // column and row of the first such pixel
	dark          bool   // the truth frame has no light, so there is no mean
}

// compare measures two frames of the same size against each other
func compare(truth, drawn *frame.Frame, width, height int) difference {
	var found difference
	var total, brightest float64
	for row := range height {
		for column := range width {
			one := truth.At(column, row)
			other := drawn.At(column, row)
			light := float64(one.R + one.G + one.B)
			apart := math.Abs(float64(one.R-other.R)) +
				math.Abs(float64(one.G-other.G)) +
				math.Abs(float64(one.B-other.B))
			total += light
			brightest = math.Max(brightest, light)
			if math.IsNaN(apart) || math.IsInf(apart, 0) {
				if found.notFinite == 0 {
					found.first = [2]int{column, row}
				}
				found.notFinite++
			}
			found.mean += apart
			found.largest = math.Max(found.largest, apart)
			if !(math.Abs(float64(one.Cover-other.Cover)) <= 1e-3) {
				found.cover++
			}
			if !sameDepth(one.Depth, other.Depth) {
				found.depth++
			}
		}
	}
	if total == 0 {
		found.dark = true
	} else {
		found.mean /= total
		found.largest /= brightest
	}
	return found
}

// findings lists what is wrong with a difference, one message each, and
// returns none when the frames agree within allowed. A black truth frame
// is reported as nothing to compare, not as agreement.
func findings(found difference, allowed tolerance) []string {
	var out []string
	if found.dark {
		out = append(out, "the slayer frame has no light, so there is nothing to compare")
	}
	if found.notFinite > 0 {
		out = append(out, fmt.Sprintf("%d pixels differ by a value that is "+
			"not finite; the first is at column %d, row %d",
			found.notFinite, found.first[0], found.first[1]))
	} else if !(found.mean <= allowed.mean && found.largest <= allowed.largest) {
		out = append(out, fmt.Sprintf("the gpu's light is %.6f off slayer's "+
			"on average, and %.4f at most", found.mean, found.largest))
	}
	if found.cover > 0 || found.depth > 0 {
		out = append(out, fmt.Sprintf("%d pixels' cover and %d pixels' "+
			"depth differ", found.cover, found.depth))
	}
	return out
}

// sameDepth is two depths within a part in a thousand, or both nothing
func sameDepth(one, other float32) bool {
	if math.IsInf(float64(one), 1) || math.IsInf(float64(other), 1) {
		return one == other
	}
	return math.Abs(float64(one-other)) <= 1e-3*float64(one)
}

// against draws a world both ways at 96 by 64, four rays per pixel
func against(t *testing.T, world *thing.World, eye maths.Subspace) {
	againstAt(t, world, eye, 96, 64, 2)
}

// tolerance is how far apart the gpu may draw from slayer's tracer: on
// average, and at most, as shares of the light
type tolerance struct{ mean, largest float64 }

// exact is the tolerance with nothing baked: float where the tracer has
// double, and a triangle test that is not watertight. measured on
// 2026-10-15, the hall is 2.2 millionths apart on average and 69 at
// most; these are about five and fifteen times those.
var exact = tolerance{mean: 1e-5, largest: 1e-3}

// againstAt draws a world both ways at a size, side by side rays a pixel, with
// nothing baked, and fails when they differ by more than the exact tolerance.
// one ray a pixel at an odd size puts a ray through the picture's very middle,
// which more rays per pixel never do.
func againstAt(t *testing.T, world *thing.World, eye maths.Subspace,
	width, height, side int) {
	baking := bakeSide
	bakeSide = 0
	defer func() { bakeSide = baking }()
	within(t, world, eye, width, height, side, exact)
}

// within draws a world both ways and fails when they differ by more
// than a tolerance
func within(t *testing.T, world *thing.World, eye maths.Subspace,
	width, height, side int, allowed tolerance) {
	t.Helper()
	lens := film.Pinhole{Wide: 50 * maths.Degree}
	withinThrough(t, world, eye, lens, width, height, side, allowed)
}

// withinThrough is within through a lens
func withinThrough(t *testing.T, world *thing.World, eye maths.Subspace,
	lens film.Sensor, width, height, side int, allowed tolerance) {
	t.Helper()
	drawn, err := Sampled(world, eye, lens, 0, width, height, side)
	if errors.Is(err, metal.ErrNoDevice) {
		t.Skip("no gpu here")
	}
	if err != nil {
		t.Fatal(err)
	}
	truth, err := slayer.Sampled(world, eye, lens, 0, width, height, side)
	if err != nil {
		t.Fatal(err)
	}
	found := compare(truth, drawn, width, height)
	t.Logf("apart: %+v", found)
	for _, problem := range findings(found, allowed) {
		t.Error(problem)
	}
}

// the gpu draws what slayer's tracer draws, near enough
func TestTheGPUDrawsTheHallAsSlayerDoes(t *testing.T) {
	world, eye := hall(t)
	against(t, world, eye)
}

// the hall gilded: its glossy floor is Johnson and Christy's gold, and
// a ball of aluminium under an amber film, as Kapton lies over a
// blanket's aluminium, sits on it. the gpu mirrors each colour by its
// own share, as slayer's tracer does.
func TestTheGPUDrawsGoldAndFilmedMetalAsSlayerDoes(t *testing.T) {
	world, eye := hall(t)
	gold := &thing.Metal{N: [3]float64{0.21, 0.43, 1.31},
		K: [3]float64{3.272, 2.455, 1.849}}
	aluminium := &thing.Metal{N: [3]float64{1.1911, 0.93525, 0.6444},
		K: [3]float64{7.0711, 6.3099, 5.3994}}
	gilded := 0
	for _, one := range world.Things {
		if one.Material != nil && one.Material.Name == "floor" {
			one.Material = &thing.Material{Name: "gold floor", Gloss: 1,
				Metal: gold}
			gilded++
		}
	}
	if gilded != 1 {
		t.Fatalf("%d floors gilded, want the hall's one", gilded)
	}
	blanket := &thing.Material{Name: "blanket", Gloss: 0.9,
		Metal: aluminium, Film: &thing.Film{Index: 1.7, Thickness: 25e-6,
			Absorbs: [3]float64{4e3, 3e4, 2e5}}}
	world.Add(thing.Thing{Shape: shape.Sphere{Radius: 0.6},
		Material: blanket, Motion: at(0.2, -0.9, 3)})
	against(t, world, eye)
}

// the gilded hall under a sun drawn as a disc, five degrees across so
// its edges show at this size, standing where the gold floor mirrors it
// to the eye: its disc in the mirror, its shadows softened across it, on
// the gpu as in slayer's tracer.
func TestTheGPUDrawsASunsDiscAsSlayerDoes(t *testing.T) {
	world, eye := hall(t)
	for _, one := range world.Things {
		if one.Material != nil && one.Material.Name == "floor" {
			one.Material = &thing.Material{Name: "gold floor", Gloss: 1,
				Metal: thing.Gold()}
		}
	}
	world.Lamp(film.Sun{Toward: maths.V(-0.15, 0.47, -0.87),
		Strength: maths.RGB(2, 1.9, 1.7), Size: 5 * maths.Degree, Disc: true})
	against(t, world, eye)
}

// a crinkled blanket in the gilded hall: aluminium under Kapton, its
// face tilting by noise as a crumpled blanket's does, mirroring and
// glinting in scattered pieces, on the gpu as in slayer's tracer.
func TestTheGPUDrawsACrinkledBlanketAsSlayerDoes(t *testing.T) {
	world, eye := hall(t)
	world.Add(thing.Thing{Shape: shape.Sphere{Radius: 0.9},
		Material: &thing.Material{Name: "crinkled blanket", Gloss: 0.9,
			Satin: 0.25, Metal: thing.Aluminium(), Film: thing.Kapton(25e-6),
			Crinkle: &thing.Crinkle{Size: 0.2, Slope: 0.35, Seed: 5}},
		Motion: at(0.2, -0.6, 3)})
	against(t, world, eye)
}

// the gilded hall in satin: the lamp leaves highlights on the gold
// floor, the blanket ball and a glossy painted ball, gold's gold and the
// paint's white, on the gpu as in slayer's tracer, through the mirror
// and the clear ball as well.
func TestTheGPUDrawsHighlightsAsSlayerDoes(t *testing.T) {
	world, eye := hall(t)
	for _, one := range world.Things {
		if one.Material != nil && one.Material.Name == "floor" {
			one.Material = &thing.Material{Name: "satin gold", Gloss: 1,
				Satin: 0.3, Metal: thing.Gold()}
		}
	}
	world.Add(thing.Thing{Shape: shape.Sphere{Radius: 0.6},
		Material: &thing.Material{Name: "blanket", Gloss: 0.9, Satin: 0.5,
			Metal: thing.Aluminium(), Film: thing.Kapton(25e-6)},
		Motion: at(0.2, -0.9, 3)})
	world.Add(thing.Thing{Shape: shape.Sphere{Radius: 0.4},
		Material: &thing.Material{Name: "gloss paint",
			Colour: maths.RGB(0.7, 0.1, 0.1), Gloss: 1, Index: 1.5, Satin: 0.2},
		Motion: at(-0.6, -1.1, 2.4)})
	against(t, world, eye)
}

// baked, the haze is read from a grid of cells blended between, which
// the tracer never does: measured on 2026-10-15, the hall is then 46
// millionths apart on average and 124 at most, and is held to about
// six and eight times those
func TestTheBakedHallIsCloseToSlayers(t *testing.T) {
	world, eye := hall(t)
	within(t, world, eye, 96, 64, 2,
		tolerance{mean: 3e-4, largest: 1e-3})
}

// with no faces at all the tree is an empty root, which the kernel must
// not walk; at an odd size the middle pixel's ray runs through the
// origin, where an empty root of one point would have been entered
func TestTheGPUDrawsSpheresAloneAsSlayerDoes(t *testing.T) {
	world, eye := orrery()
	againstAt(t, world, eye, 97, 65, 1)
}

// a regular floor seen head on: the middle row's rays run straight down
// an edge two faces share, and without slack at the edges 18 of them
// slipped between the faces into the sky
func TestAFloorSeenHeadOnHasNoGaps(t *testing.T) {
	world := thing.NewWorld()
	grey := &thing.Material{Name: "floor", Colour: maths.Grey(0.6)}
	world.Add(thing.Thing{Shape: floor(t, 30), Material: grey})
	world.Lamp(emit.Lit{At: maths.V(0, 6, 4),
		Strength: maths.RGB(20, 20, 20)})
	from := maths.V(0, 3, 12)
	eye := maths.Subspace{Origin: from,
		Orientation: maths.LookAt(from, maths.Vec{})}
	againstAt(t, world, eye, 97, 65, 1)
}

// bespoke is a lamp of a kind of its own, which the tracer asks like any
// other and the kernel has no way to read. It is private to this test so
// that no later work on the kernel can make it a lamp the kernel draws.
type bespoke struct{}

// Light is a lamp straight above, at no distance and one unit of light.
func (bespoke) Light(_, _ maths.Vec) (maths.Vec, float64, maths.Light) {
	return maths.V(0, 1, 0), math.Inf(1), maths.Grey(1)
}

// shine is a texture of one light
type shine struct{}

func (shine) At(maths.Vec) maths.Light { return maths.Grey(1) }

// every way a world can hold what the kernel does not draw yet, each
// refused by name rather than drawn without it
func TestWhatItCannotDrawIsRefused(t *testing.T) {
	smooth, err := shape.NewMesh([]maths.Vec{maths.V(0, 0, 0),
		maths.V(1, 0, 0), maths.V(0, 1, 0)}, [][3]int{{0, 1, 2}},
		[]maths.Vec{maths.V(0, 0, 1), maths.V(0, 0, 1),
			maths.V(0, 0, 1)}, true)
	if err != nil {
		t.Fatal(err)
	}
	// a smooth mesh is drawn; one that has lost all but one of its
	// normals since it was made is not
	smooth.Normals = smooth.Normals[:1]
	cases := map[string]func(*thing.World) film.Sensor{
		"a lamp of its own kind": func(world *thing.World) film.Sensor {
			world.Lamp(bespoke{})
			return nil
		},
		"a smooth mesh without a normal for every vertex": func(world *thing.World) film.Sensor {
			world.Add(thing.Thing{Shape: smooth})
			return nil
		},
		"a gas with no lobe": func(world *thing.World) film.Sensor {
			world.AddVolume(thing.Volume{Medium: gas.Of{
				Shape: medium.Puff{Radius: 1, Thick: 1},
				In:    gas.Gas{Name: "flat", Phase: 1}}})
			return nil
		},
		"multiple fields": func(world *thing.World) film.Sensor {
			hole := &thing.Field{Mass: 1, Reach: 1}
			world.Fields = append(world.Fields, hole, hole)
			return nil
		},
		"the world's haze": func(world *thing.World) film.Sensor {
			world.Haze.Density = 0.1
			return nil
		},
		"a beam": func(world *thing.World) film.Sensor {
			ball := medium.Puff{Radius: 1}
			world.AddVolume(thing.Volume{Medium: ball,
				Beam: &thing.Beam{}})
			return nil
		},
		"too many volumes": func(world *thing.World) film.Sensor {
			for range mostVolumes + 1 {
				world.AddVolume(thing.Volume{
					Medium: medium.Puff{Radius: 1}})
			}
			return nil
		},
	}
	for name, add := range cases {
		world := thing.NewWorld()
		lens := add(world)
		_, err := Sampled(world, maths.Subspace{}, lens, 0, 4, 4, 1)
		var refused unsupported
		if !errors.As(err, &refused) {
			t.Errorf("%s was not refused: %v", name, err)
		}
	}
}

// TestTheComparisonFailsOnWhatItCannotMeasure feeds compare two 2 by 2
// frames by hand. Each truth pixel has light 0.5 in each channel, so the
// whole truth frame has light 6.
func TestTheComparisonFailsOnWhatItCannotMeasure(t *testing.T) {
	lit := func() *frame.Frame {
		f := frame.New(2, 2)
		for i := range f.Pix {
			f.Pix[i].R, f.Pix[i].G, f.Pix[i].B = 0.5, 0.5, 0.5
		}
		return f
	}
	dark := func() *frame.Frame { return frame.New(2, 2) }
	nan := func() *frame.Frame {
		f := lit()
		f.Pix[1].G = float32(math.NaN())
		return f
	}
	inf := func() *frame.Frame {
		f := lit()
		f.Pix[3].B = float32(math.Inf(1))
		return f
	}
	off := func() *frame.Frame {
		f := lit()
		f.Pix[0].R = 1.5
		return f
	}
	covered := func() *frame.Frame {
		f := lit()
		f.Pix[0].Cover = 1
		return f
	}
	deep := func() *frame.Frame {
		f := lit()
		f.Pix[2].Depth = 5
		return f
	}
	cases := []struct {
		name         string
		truth, drawn *frame.Frame
		want         string // a substring of the one finding, or "" for none
	}{
		{"identical frames agree", lit(), lit(), ""},
		{"a black truth frame", dark(), dark(), "has no light"},
		{"a NaN pixel at column 1, row 0", lit(), nan(),
			"1 pixels differ by a value that is not finite; the first is at column 1, row 0"},
		{"an infinite pixel at column 1, row 1", lit(), inf(),
			"not finite; the first is at column 1, row 1"},
		{"one pixel's cover differs", lit(), covered(),
			"1 pixels' cover and 0 pixels' depth differ"},
		{"one pixel's depth differs", lit(), deep(),
			"0 pixels' cover and 1 pixels' depth differ"},
		{"a finite miss of one unit", lit(), off(),
			"the gpu's light is 0.166667 off slayer's on average"},
	}
	for _, c := range cases {
		msgs := findings(compare(c.truth, c.drawn, 2, 2), exact)
		if c.want == "" {
			if len(msgs) != 0 {
				t.Errorf("%s: got findings %q, want none", c.name, msgs)
			}
			continue
		}
		if len(msgs) != 1 || !strings.Contains(msgs[0], c.want) {
			t.Errorf("%s: got findings %q, want one containing %q",
				c.name, msgs, c.want)
		}
	}
}
