// Package beamstep refines volume integration where a ray may cross a narrow
// light beam. Bounds choose sampling effort; they do not discard beam tails.
package beamstep

import (
	_ "embed"
	"math"

	"github.com/janearc/puzzlemouth/numbers"
	"github.com/janearc/slayer/emit"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
)

// zero and one are defined additive/multiplicative identities.
const zero, one = 0, 1

//go:embed choices.yaml
var choicesRaw []byte

type choices struct {
	radii, radial float64
	maximum       int
}

var chosen = readChoices()

// readChoices checks immutable embedded budgets through Puzzlemouth.
func readChoices() choices {
	read, err := numbers.Read("choices.yaml", choicesRaw)
	if err != nil {
		panic(err)
	}
	return choices{radii: choice(read, "refinement-radii"),
		radial:  choice(read, "samples-per-radius"),
		maximum: int(choice(read, "maximum-splits"))}
}

// choice names a checked embedded value; a missing name is a code fault.
func choice(read numbers.Numbers, name string) float64 {
	number, err := read.Get(name)
	if err != nil {
		panic(err)
	}
	return number.Value
}

// Radii is the shared CPU/Metal refinement-envelope padding.
func Radii() float64 { return chosen.radii }

// RadialSamples is the target samples across one launch radius.
func RadialSamples() float64 { return chosen.radial }

// MaximumSplits bounds work within one original volume-march cell.
func MaximumSplits() int { return chosen.maximum }

type source struct {
	box  maths.Box
	step float64
}

// Beams caches immutable launch envelopes once for a render's lamp snapshot.
type Beams struct{ sources []source }

// Of prepares sampling bounds for Gaussian beams; other lamps need no change.
func Of(lamps []thing.Lamp) *Beams {
	plan := &Beams{}
	for _, lamp := range lamps {
		beam, ok := lamp.(*emit.Beam)
		if !ok || beam == nil {
			continue
		}
		point := beam.Point()
		reach := beam.Reach().InMetres()
		radius := beam.Radius().InMetres()
		end := point.At.Add(point.Toward.Scale(reach))
		margin := (radius + reach*beam.Slope()) * chosen.radii
		plan.sources = append(plan.sources, source{
			box:  maths.SegmentBox(point.At, end, margin),
			step: radius / chosen.radial,
		})
	}
	return plan
}

// Refinement is a ray's conservative interval and desired local step size.
// Merging intervals may refine gaps too; it never declares a gap empty.
type Refinement struct{ enter, leave, step float64 }

// Ray computes bounds for this ray. No pilot ray stands in for neighbours.
func (beams *Beams) Ray(ray maths.Ray, far float64) Refinement {
	result := Refinement{}
	if beams == nil {
		return result
	}
	for _, source := range beams.sources {
		// A miss gives an empty interval. Point contacts also have no
		// length and must not extend another beam's interval.
		enter, leave, _ := source.box.Hit(ray, zero, far)
		if leave <= enter {
			continue
		}
		if result.step == zero {
			result = Refinement{enter: enter, leave: leave,
				step: source.step}
			continue
		}
		result.enter = math.Min(result.enter, enter)
		result.leave = math.Max(result.leave, leave)
		result.step = math.Min(result.step, source.step)
	}
	return result
}

// Splits preserves original cell boundaries. Unaffected cells still use their
// original midpoint; intersected cells divide into a bounded regular grid.
func (refine Refinement) Splits(enter, leave float64) int {
	if refine.step == zero || leave <= refine.enter ||
		enter >= refine.leave {
		return one
	}
	count := math.Ceil((leave - enter) / refine.step)
	limit := float64(chosen.maximum)
	count = math.Min(limit, count)
	count = math.Max(one, count)
	return int(count)
}
