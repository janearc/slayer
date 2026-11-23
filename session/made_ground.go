package session

// Ground a set asks made finer than its survey is laid as the set's
// camera sees it: each cell of the window cut as finely as a pixel
// resolves it there, made from the survey's own roughness continued below
// its spacing and from the processes the set says, and nothing made where
// the camera does not look or resolves no finer than the survey.

import (
	"fmt"
	"math"
	"strings"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/strategies"
	"github.com/janearc/slayer/mesh"
	"github.com/janearc/slayer/set"
	"github.com/janearc/slayer/terrain"
	"github.com/janearc/slayer/titan/over"
)

// The numbers made ground is laid by where the set does not say them.
const (
	// madeFacesPerPixel is how many made faces each pixel of the film may
	// have: a square of corners a pixel apart is two. Picked: what is made
	// grows with the picture, not with the ground.
	madeFacesPerPixel = 2
	// madeFinest is the finest made ground is cut, metres: a centimetre,
	// the finest titan makes. Defined by titan's inchworm.
	madeFinest = 0.01
	// madeDeparture is the most a made corner moves, as a share of the
	// survey's spacing: a quarter. Picked, so made ground cannot become
	// another shape than the survey's.
	madeDeparture = 0.25
	// madeStand is the steepest made relief leans off its ground, rise
	// over run: about 27 degrees, all its octaves together by their root
	// mean square. Picked, at the most titan/over lets an eye's ground
	// have, so what an eye leaves out stays under its pixel.
	madeStand = 0.5
	// madeGain is how much of the survey's roughness is made: all of it,
	// as the survey has it. Defined.
	madeGain = 1
	// madeSteep is the rise over run past which ground is a face flutes and
	// beds work on, where the set says none: about 60 degrees. Picked.
	madeSteep = 1.7
	// madeDrainageCells is how many of the survey's cells must drain
	// through a place for its flutes to be cut full. Picked.
	madeDrainageCells = 20
	// madeFluteLength is how many widths a flute runs down a face. Picked.
	madeFluteLength = 10
	// nearestSight is the least distance, metres, a place is taken to be
	// from the eye: a centimetre. Defined as the leveller's.
	nearestSight = 0.01
)

// sight is the fineness the set's camera wants of the ground: corners a
// pixel apart where it sees, and none anywhere else.
type sight struct {
	place        maths.Subspace
	focal        float64 // pixels a metre spans a metre from the eye
	across, down float64 // the tangents of half the view each way
	reach        float64 // metres a place may stand outside the view
}

// sightOf is the fineness an eye wants of ground whose survey is so many
// metres a sample: a place within a cell's diagonal of the view is seen.
func sightOf(eye strategies.Eye, step float64) sight {
	across := math.Tan(eye.Wide / 2)
	return sight{place: eye.Place,
		focal:  float64(eye.Across) / (2 * across),
		across: across, down: across * float64(eye.Down) /
			float64(eye.Across),
		reach: step * math.Sqrt2}
}

// Wanted is the metres a pixel spans at a place, or plus infinity where
// the place is behind the eye or outside its view.
func (eye sight) Wanted(point maths.Vec) float64 {
	way := point.Sub(eye.place.Origin)
	axes := eye.place.Orientation
	ahead := -way.Dot(axes.Z)
	if ahead < -eye.reach ||
		math.Abs(way.Dot(axes.X)) > ahead*eye.across+eye.reach ||
		math.Abs(way.Dot(axes.Y)) > ahead*eye.down+eye.reach {
		return math.Inf(1)
	}
	return max(way.Len(), nearestSight) / eye.focal
}

// madeRules are the rules made ground is laid by: the set's seed and
// processes, and the faces the film's pixels allow.
func madeRules(made set.Made, step float64, eye strategies.Eye) over.Rules {
	steep, said := made.Steep()
	if !said {
		steep = madeSteep
	}
	rules := over.Rules{Seed: made.Seed(), Spacing: madeFinest,
		Faces:     madeFacesPerPixel * eye.Across * eye.Down,
		Departure: madeDeparture,
		Roughness: over.Roughness{Gain: madeGain, Stand: madeStand,
			Continued: true}}
	if depth, width, fluted := made.Flutes(); fluted {
		rules.Flutes = over.Flutes{Slope: steep, Depth: depth,
			Drainage: madeDrainageCells * step * step, Width: width,
			Length: madeFluteLength}
	}
	if spacing, standout, bedded := made.Beds(); bedded {
		rules.Beds = over.Beds{Slope: steep, Spacing: spacing,
			Standout: standout}
	}
	return rules
}

// groundSurface is the window's ground as a mesh: laid as the camera sees
// it where the set asks made ground, which says what it made in the
// tally, and otherwise the survey's own triangles.
func (stage *staging) groundSurface(hole *terrain.Hole) (*mesh.Mesh,
	error) {
	ground, _ := stage.recipes.Ground()
	made, asked := ground.Made()
	if !asked {
		return stage.ground.GroundOn(0, 0, stage.lowest, hole)
	}
	eye, err := stage.seen()
	if err != nil {
		return nil, fmt.Errorf("made ground: %w", err)
	}
	_, _, step := stage.ground.Size()
	built, receipt, err := over.Seen(stage.ground, stage.lowest,
		sightOf(eye, step), madeRules(made, step, eye))
	if err != nil {
		return nil, err
	}
	stage.made.ground = withReceipt(stage.made.ground, receipt)
	return built, nil
}

// withReceipt is what the tally says of the ground with the made ground's
// receipt after it: a placed window's own sentence is kept.
func withReceipt(said string, receipt over.Receipt) string {
	laid := strings.ReplaceAll(receipt.String(), "\n", "; ")
	if said == "" {
		return laid
	}
	return said + "; " + laid
}
