//go:build draw && darwin

package gpu

import (
	"math"
	"testing"

	"github.com/janearc/slayer/emit"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/render/frame"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
)

// seaApart is the tolerance the kernel's sea is held to against the
// tracer's: floats against doubles in a search that stops within a touch
// of the surface, so wider than the exact hall's. Chosen, not yet measured;
// the first run says what it is.
var seaApart = tolerance{mean: 2e-3, largest: 1e-1}

// seaColumns, seaRows and seaRays are the picture's width and height in
// pixels and its rays along a side of a pixel.
const (
	seaColumns = 96
	seaRows    = 64
	seaRays    = 2
)

// seaEyeUp and seaLookAhead are where the eye stands over the sea's mean
// and how far ahead and down it looks, in metres: about twenty degrees
// down, where the sea fills the picture without grazing. Chosen.
const (
	seaEyeUp     = 2.0
	seaLookAhead = 5.5
)

// seaLevelEyeUp is how high the level eye stands: inside the slab the sea
// stays in, so its rays below the middle of the picture skim the waves
// before they meet them. Chosen.
const seaLevelEyeUp = 0.3

// seaFloor is how deep under the sea's mean a lit floor stands, for the
// clear sea's bend to find. Chosen.
const seaFloor = 6.0

// seaWorld is the trade sea with no edge, clear or opaque, over a lit
// floor under a lamp, and an eye at height looking at a point ahead.
func seaWorld(t *testing.T, clear bool, up, ahead float64) (*thing.World,
	maths.Subspace) {
	t.Helper()
	water := &thing.Material{Name: "sea", Gloss: 1, Index: 1.333,
		Colour: maths.RGB(0.02, 0.05, 0.07), Satin: 0.2}
	if clear {
		water.Through = 1
	}
	world := thing.NewWorld()
	world.Add(thing.Thing{Shape: newTradeSea(t), Material: water})
	floor := &thing.Material{Name: "floor", Glow: maths.RGB(0.6, 0.55, 0.4)}
	world.Add(thing.Thing{Shape: shape.Plane{}, Material: floor,
		Motion: at(0, -seaFloor, 0)})
	world.Lamp(emit.Lit{At: maths.V(-10, 30, -40),
		Strength: maths.RGB(4e4, 3.8e4, 3.5e4)})
	from := maths.V(0, up, 0)
	look := maths.V(0, 0, -ahead)
	if ahead == 0 {
		look = maths.V(0, up, -1)
	}
	return world, maths.Subspace{Origin: from,
		Orientation: maths.LookAt(from, look)}
}

// The kernel meets and shades an opaque sea with no edge as arid.Sea does
// on the tracer: its search, its spectral normal, its highlight.
func TestTheGPUDrawsAnOpaqueSeaAsSlayerDoes(t *testing.T) {
	world, eye := seaWorld(t, false, seaEyeUp, seaLookAhead)
	within(t, world, eye, seaColumns, seaRows, seaRays, seaApart)
}

// The kernel bends and mirrors a clear sea by its one normal, as the
// tracer does for a shape with no geometric normal apart.
func TestTheGPUDrawsAClearSeaAsSlayerDoes(t *testing.T) {
	world, eye := seaWorld(t, true, seaEyeUp, seaLookAhead)
	within(t, world, eye, seaColumns, seaRows, seaRays, seaApart)
}

// Anti-test: the waves reach the picture. The sea drawn on the kernel is
// further from a plane at its mean than the two tracers are from each
// other, so the parity above is not a flat plane passing for a sea.
func TestTheGPUSeaIsNotItsMeanPlane(t *testing.T) {
	world, eye := seaWorld(t, false, seaEyeUp, seaLookAhead)
	flat, _ := seaWorld(t, false, seaEyeUp, seaLookAhead)
	flat.Things[0].Shape = shape.Plane{}
	sea, _ := bothWaysAt(t, world, eye, seaColumns, seaRows, seaRays)
	plane, _ := bothWaysAt(t, flat, eye, seaColumns, seaRows, seaRays)
	found := compare(plane, sea, seaColumns, seaRows)
	if len(findings(found, seaApart)) == 0 {
		t.Errorf("the kernel drew the sea as its mean plane: %+v", found)
	}
}

// Anti-test: a level eye inside the slab, whose rays below the middle of
// the picture skim the crests a long way before they meet the water. Every
// one of them meets it on the kernel, by the search or on the mean plane
// when its steps run out; none is a miss. The floor is taken away, so a
// ray the sea let through meets nothing and leaves its pixel uncovered.
func TestALevelEyeInsideTheSlabMeetsTheSeaBelowTheMiddle(t *testing.T) {
	world, eye := seaWorld(t, false, seaLevelEyeUp, 0)
	world.Things = world.Things[:1]
	drawn, _ := bothWaysAt(t, world, eye, seaColumns, seaRows, seaRays)
	for row := seaRows/2 + 1; row < seaRows; row++ {
		for column := range seaColumns {
			if uncovered(drawn.At(column, row)) {
				t.Fatalf("the ray at column %d, row %d missed the sea",
					column, row)
			}
		}
	}
}

// uncovered is whether any of a pixel's rays met nothing: the pixel not
// wholly covered, or no depth to it.
func uncovered(pixel frame.Pixel) bool {
	return math.IsInf(float64(pixel.Depth), 1) || pixel.Cover < 1
}
