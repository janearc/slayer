//go:build draw && darwin

package gpu

import (
	"image"
	"strings"
	"testing"
	"time"

	"github.com/janearc/slayer/emit"
	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/metal"
	"github.com/janearc/slayer/render/colour"
	"github.com/janearc/slayer/render/frame"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
)

// farPlanet is how far the planet's near surface is from the meshes, in
// metres: about the 2.4e8 m of the Melancholia ray that never came back.
const farPlanet = 2.4e8

// farScene is a few small meshes at the origin, a planet whose near
// surface is farPlanet metres off along a slant, and a lamp at the eye,
// so a shadow ray from the planet to the lamp runs back along the slant
// farPlanet metres long and crosses the meshes' bounds.
func farScene(t *testing.T) (*thing.World, maths.Subspace) {
	t.Helper()
	group, _ := groupFixture(t)
	world := thing.NewWorld()
	world.Add(thing.Thing{Shape: group,
		Material: &thing.Material{Colour: maths.Grey(0.5)}})
	slant := maths.V(0.9187, 0.3204, -0.2312).Unit()
	radius := farPlanet / 2
	from := maths.V(2, 0, -1).Sub(slant.Scale(10))
	centre := from.Add(slant.Scale(farPlanet + radius))
	world.Add(thing.Thing{Shape: shape.Sphere{Radius: radius},
		Material: &thing.Material{Colour: maths.RGB(0.2, 0.4, 0.8)},
		Motion:   at(centre.X, centre.Y, centre.Z)})
	world.Lamp(emit.Lit{At: from, Strength: maths.Grey(1e6)})
	return world, maths.Subspace{Origin: from,
		Orientation: maths.LookAt(from, centre)}
}

// nativeDrawn draws a scene through the hardware queries, first hits and
// shadows, from a kernel source the caller may have changed, with each
// command limited to limit, and gives the developed picture or the error.
func nativeDrawn(t *testing.T, world *thing.World, eye maths.Subspace,
	lens film.Pinhole, width, height int, limit time.Duration,
	change func(string) string) (*frame.Frame, error) {
	t.Helper()
	encoded, err := encodeGrown(world, nil, eye, lens, 0, width, height)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := encoded.bakeAll(); err != nil {
		t.Fatal(err)
	}
	firsts := encoded.nativeFirsts()
	text := change(sourceWith(encoded.records(), encoded.counts.volumes))
	kernel, err := metal.Open("#define SLAYER_NATIVE_SHADOWS 1\n"+
		"#define SLAYER_NATIVE_MESHES 1\n"+text, "trace")
	if err != nil {
		t.Fatal(err)
	}
	defer kernel.Close()
	kernel.SetLimit(limit)
	plan, root, err := encoded.accelerationPlan()
	if err != nil {
		t.Fatal(err)
	}
	if err = kernel.Accelerate(plan, root); err != nil {
		t.Fatal(err)
	}
	inputs := append(encoded.inputs(), firsts)
	samples, err := kernel.Grid(width, height, rows, encoded.perSample(), inputs)
	if err != nil {
		return nil, err
	}
	return averaged(finishing{world: world, records: encoded.records(),
		paints: encoded.paints,
		wide:   wideOf(world, eye, lens, width, height)},
		samples, width, height, 1), nil
}

// unchanged leaves a kernel source as it is.
func unchanged(source string) string { return source }

// wrongClip makes the clip start where the segment leaves the bounds,
// after every surface inside them.
func wrongClip(source string) string {
	return strings.Replace(source,
		"start = max(max(beam_zero, near), along - half_chord);",
		"start = max(max(beam_zero, near), along + half_chord);", 1)
}

// farOff is the share of pixels past one just-noticeable difference that
// the agreement tests allow, the 2026-10-14 still budget.
const farOff = 0.02

// development makes a frame comparable by eye.
func development(picture *frame.Frame) image.Image {
	return colour.Tone{Knee: 0.7, Top: 8}.Develop(picture)
}

// the hardware queries, clipped to the bounds, draw the same picture as
// the software walk on a scene whose planet and lamp are 2.4e8 m apart:
// the clip does not cost a shadow or a surface.
func TestClippedQueriesAgreeWithSoftwareFromAFarOrigin(t *testing.T) {
	world, eye := farScene(t)
	lens := film.Pinhole{Wide: 10 * maths.Degree}
	const width, height = 96, 72
	truth, err := Sampled(world, eye, lens, 0, width, height, 1)
	if err != nil {
		t.Fatal(err)
	}
	drawn, err := nativeDrawn(t, world, eye, lens, width, height,
		20*time.Second, unchanged)
	if err != nil {
		t.Fatal(err)
	}
	share, worst := perceptualDifference(development(truth), development(drawn))
	if share > farOff {
		t.Fatalf("clipped queries differ from software on %.4f%% of pixels (worst %.3f JND); want at most %.1f%%",
			share*100, worst, farOff*100)
	}
}

// the same measure fails a clip that starts after the first hit: the
// picture drawn with it differs from software past the budget, on a
// scene of meshes near the eye, where every first hit is inside the bounds.
func TestAWrongClipFailsTheAgreementWithSoftware(t *testing.T) {
	world, eye := nativePlacedFixture(t)
	lens := film.Pinhole{Wide: 50 * maths.Degree}
	const width, height = 96, 64
	truth, err := Sampled(world, eye, lens, 0, width, height, 1)
	if err != nil {
		t.Fatal(err)
	}
	right, err := nativeDrawn(t, world, eye, lens, width, height,
		20*time.Second, unchanged)
	if err != nil {
		t.Fatal(err)
	}
	wrong, err := nativeDrawn(t, world, eye, lens, width, height,
		20*time.Second, wrongClip)
	if err != nil {
		t.Fatal(err)
	}
	good, _ := perceptualDifference(development(truth), development(right))
	bad, _ := perceptualDifference(development(truth), development(wrong))
	if !(good <= farOff) || !(bad > farOff) {
		t.Fatalf("the right clip differs on %.4f%% of pixels and the wrong one on %.4f%%; want at most %.1f%% and more than it",
			good*100, bad*100, farOff*100)
	}
}
