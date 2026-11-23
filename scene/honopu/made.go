package honopu

import (
	"fmt"
	"io"
	"slices"

	"github.com/janearc/slayer/mesh"
	"github.com/janearc/slayer/terrain"
	"github.com/janearc/slayer/titan/over"
)

// Details that lay made ground over the survey, finer than the survey is:
// all three rules, or one of them alone to see what it does.
const (
	madeAll       = "made"
	madeFlutes    = "made-flutes"
	madeBeds      = "made-beds"
	madeRoughness = "made-roughness"
)

// madeKind is whether a detail lays made ground.
func madeKind(detail string) bool {
	return slices.Contains([]string{madeAll, madeFlutes, madeBeds,
		madeRoughness}, detail)
}

// madeStretch is the cells of the survey laid finer: a wall facing north,
// about 105 m tall, rising from a shelf a few metres over the sea and
// steeper than 60 degrees over most of its height. Cells count from the
// window's first sample, which stands at 4 m a sample.
var madeStretch = over.Stretch{Column: 388, Row: 378, Across: 18, Down: 10}

// The numbers the made ground is laid by. Each is picked, and none is
// measured, unless it says it is.
const (
	// madeSeed makes the same ground each time.
	madeSeed = 2318
	// madeSpacing is the finest wanted, metres between corners along the
	// surface, along each way of the survey.
	madeSpacing = 0.25
	// madeFaces is the most faces the stretch may have; the spacing is
	// made coarser to fit them.
	madeFaces = 2_000_000
	// madeDeparture is the most any corner moves, as a share of the survey's
	// spacing: a quarter of 4 m is a metre.
	madeDeparture = 0.25
	// madeMargin is how far in from the stretch's edge detail eases in,
	// metres: two of the survey's cells.
	madeMargin = 8.0
	// madeSteep is the rise over run beyond which ground is bare rock
	// the rules work on: the tangent of 60 degrees, about.
	madeSteep = 1.7
	// madeFluteDepth is how deep a flute is cut, metres, where enough
	// drains.
	madeFluteDepth = 1.0
	// madeDrainage is the square metres that must drain through a place
	// for its flutes to be cut full. It is picked: a few dozen cells of the
	// survey's 16 m2 each.
	madeDrainage = 300.0
	// madeFluteWidth is the metres between flutes; madeFluteLength is how
	// many widths a flute runs down the face.
	madeFluteWidth  = 1.6
	madeFluteLength = 10.0
	// madeBedSpacing is the metres of height between ledges, and
	// madeBedStandout how far a ledge stands out of the face, metres.
	madeBedSpacing  = 3.0
	madeBedStandout = 0.4
	// madeRoughnessGain is how much of the survey's spread over a
	// neighbourhood, per metre of its width, is made as steepness at every
	// wavelength; madeRoughnessStand is the steepest, rise over run, made
	// relief may lean off the ground: about 30 degrees.
	madeRoughnessGain  = 1.0
	madeRoughnessStand = 0.6
)

// madeRules are the rules the made ground is laid by, with the rules a
// detail leaves out set to nothing.
func madeRules(detail string) over.Rules {
	rules := over.Rules{Seed: madeSeed, Spacing: madeSpacing,
		Faces: madeFaces, Departure: madeDeparture, Margin: madeMargin,
		Flutes: over.Flutes{Slope: madeSteep, Depth: madeFluteDepth,
			Drainage: madeDrainage, Width: madeFluteWidth,
			Length: madeFluteLength},
		Beds: over.Beds{Slope: madeSteep, Spacing: madeBedSpacing,
			Standout: madeBedStandout},
		Roughness: over.Roughness{Gain: madeRoughnessGain,
			Stand: madeRoughnessStand}}
	if detail != madeAll && detail != madeFlutes {
		rules.Flutes.Depth = 0
	}
	if detail != madeAll && detail != madeBeds {
		rules.Beds.Standout = 0
	}
	if detail != madeAll && detail != madeRoughness {
		rules.Roughness.Gain = 0
	}
	return rules
}

// ground is the window as a mesh to stand the scene on, and the height
// taken from it. A made detail lays the stretch finer and says to said what
// it laid.
func ground(window terrain.Window, detail string, said io.Writer) (*mesh.Mesh,
	float64, error) {
	if !madeKind(detail) {
		return window.Ground()
	}
	across, down, _ := window.Size()
	low := window.At(0, 0)
	for row := range down {
		for column := range across {
			low = min(low, window.At(column, row))
		}
	}
	solid, receipt, err := over.Lay(window, low, madeStretch, madeRules(detail))
	if err != nil {
		return nil, 0, fmt.Errorf("honopu: made ground: %w", err)
	}
	fmt.Fprintln(said, receipt)
	return solid, low, nil
}
