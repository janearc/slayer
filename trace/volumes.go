package trace

import (
	"math"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/render/beamstep"
	"github.com/janearc/slayer/scene/medium"
)

// passage is what a ray gathers on its way through volumes: keep, how
// much of the light from behind still gets through, by channel, and
// added, the light the volumes put in front of it. The two are both
// lights, and the names keep them from being taken for each other.
type passage struct {
	keep, added maths.Light
}

// Tinted is a painted volume's medium that is thicker to some light than
// to other, as soot is to blue. Tint is how many times its density each
// channel meets, at the eye's three wavelengths gas.go names. A medium
// that is not Tinted is as thick to every channel.
type Tinted interface {
	medium.Medium
	Tint() maths.Light
}

// untouched is the passage of a ray that has met nothing yet: all of
// what is behind, and nothing added.
func untouched() passage {
	return passage{keep: maths.Grey(1)}
}

// through is what the ray gathers from the volumes between the eye and
// far.
//
// Volumes are added up front to back, step by step: where two overlap,
// as an aurora round a giant or clouds in air, their steps are taken in
// turn by how far along the ray each is.
func through(scene staged, ray maths.Ray, far float64) passage {
	return throughGain(scene, ray, far, nil, strides)
}

// shows is whether enough light still gets through for a march to go
// on: any channel of keep above faint.
func shows(keep maths.Light) bool {
	return keep.R > faint || keep.G > faint || keep.B > faint
}

// most is the channel of keep that lets the most through, which is what
// a pixel's cover is taken from.
func most(keep maths.Light) float64 {
	return math.Max(keep.R, math.Max(keep.G, keep.B))
}

// throughGain is through with the volumes' light scaled by a field's
// shift, nil for none, and a crossing cut into cuts steps, since the bent
// march's segments are short and want few.
func throughGain(scene staged, ray maths.Ray, far float64, gain gainer,
	cuts int) passage {
	passed := untouched()
	across := crossings(scene.volumes, ray, far)
	gains := make([]float64, 0, len(across))
	for _, crossed := range across {
		scale := 1.0
		if gain != nil {
			scale = gain(crossed,
				ray.Point((crossed.enter+crossed.leave)/2))
		}
		gains = append(gains, scale)
	}
	steps := stepping(across, cuts)
	steps.refine = scene.beams.Ray(ray, far)
	for shows(passed.keep) {
		step, found := steps.next()
		if !found {
			break
		}
		crossed, k := &across[step.crossing], step.crossing
		if crossed.volume.gas != nil {
			// a gas: what each channel loses and gathers over the
			// step, by the gas's own numbers at the eye's bands
			passed = gasStep(scene, crossed, step, ray, gains[k],
				passed)
			continue
		}
		absorb := absorbed(crossed.volume, step.thick)
		light := scattered(scene, crossed, step, ray, gain == nil)
		// each channel as the one number was, in the same order,
		// so a painted volume's picture is the one it always was
		keep, scale := &passed.keep, gains[k]
		passed.added.R += light.R * (keep.R * absorb.R * scale)
		passed.added.G += light.G * (keep.G * absorb.G * scale)
		passed.added.B += light.B * (keep.B * absorb.B * scale)
		keep.R *= 1 - absorb.R
		keep.G *= 1 - absorb.G
		keep.B *= 1 - absorb.B
		if scene.walked != nil {
			strode(scene.walked, crossed, step, absorb, light,
				*keep)
		}
	}
	return passed
}

// heated is a medium that says how hot it is at a point, as a flame's
// body does: a replay tells it beside the glow it gives.
type heated interface {
	Heat(p maths.Vec) measure.Temperature
}

// strode tells a replay one step through a painted volume: where along
// the ray, how thick, what it took and gave, what is left, and how hot
// it is there for a medium that says.
func strode(told *walk, crossed *crossing, step stride, absorb,
	light, keep maths.Light) {
	said := []any{"along", step.along, "thick", step.thick,
		"taken", absorb, "given", light, "keep", keep}
	if hot, ok := crossed.volume.med.(heated); ok {
		said = append(said, "kelvin", hot.Heat(step.local).InKelvin())
	}
	what := "a step through a volume"
	if material := crossed.volume.source.Material; material != nil &&
		material.Name != "" {
		what = "a step through the " + material.Name
	}
	told.tell(what, said...)
}

// absorbed is how much of the light a step of a painted volume takes,
// by channel: one less e to the minus its thickness, the tint's times
// in each channel for a tinted volume.
func absorbed(volume *placedVolume, thick float64) maths.Light {
	if volume.tint == nil {
		return maths.Grey(1 - math.Exp(-thick))
	}
	tint := volume.tint
	return maths.RGB(1-math.Exp(-thick*tint.R), 1-math.Exp(-thick*tint.G),
		1-math.Exp(-thick*tint.B))
}

// stride is one step of a ray across a volume it crosses: which of the
// ray's crossings it is in, how far along the ray its middle is and
// how long it is, in the world's distances, and its middle in the
// volume's own space.
//
// thick is how thick the step is: the density there times its length
// in the medium's own units.
type stride struct {
	crossing      int
	along, length float64
	local         maths.Vec
	thick         float64
}

// stepper hands out the strides across a ray's crossings front to
// back: each crossing cut into cuts steps, and the steps of crossings that
// overlap taken in turn by how far along the ray each is. Crossings
// that do not overlap come one after another.
type stepper struct {
	across []crossing
	cuts   int
	taken  []int
	parts  []int
	refine beamstep.Refinement
}

// marchZero is defined as the first subdivision index.
const marchZero = 0

// marchHalf is defined as the midpoint of a unit cell.
const marchHalf = 0.5

// stepping is a stepper over the crossings, each cut into cuts steps.
func stepping(across []crossing, cuts int) *stepper {
	return &stepper{across: across, cuts: cuts, taken: make([]int,
		len(across)), parts: make([]int, len(across))}
}

// nearest is the nearest step not yet taken, as a stride with its
// place and length and no density read yet. A tie goes to the crossing
// entered first.
func (steps *stepper) nearest() (stride, bool) {
	found, best := false, stride{}
	for k := range steps.across {
		crossed := &steps.across[k]
		if steps.taken[k] >= steps.cuts {
			continue
		}
		length := (crossed.leave - crossed.enter) / float64(steps.cuts)
		begin := crossed.enter + float64(steps.taken[k])*length
		split := steps.refine.Splits(begin, begin+length)
		middle := float64(steps.taken[k]) +
			(float64(steps.parts[k])+marchHalf)/float64(split)
		along := crossed.enter + middle*length
		if !found || along < best.along {
			found = true
			best = stride{crossing: k, along: along,
				length: length / float64(split)}
		}
	}
	if found {
		k := best.crossing
		crossed := steps.across[k]
		length := (crossed.leave - crossed.enter) / float64(steps.cuts)
		begin := crossed.enter + float64(steps.taken[k])*length
		split := steps.refine.Splits(begin, begin+length)
		steps.parts[k]++
		if steps.parts[k] >= split {
			steps.parts[k] = marchZero
			steps.taken[k]++
		}
	}
	return best, found
}

// next is the nearest stride not yet taken that has anything in it: steps where
// the volume's density is zero or less are passed over.
func (steps *stepper) next() (stride, bool) {
	for {
		step, found := steps.nearest()
		if !found {
			return stride{}, false
		}
		crossed := &steps.across[step.crossing]
		own := step.along / crossed.volume.scale
		step.local = crossed.local.Point(own)
		density := crossed.volume.med.Density(step.local)
		if density <= 0 {
			continue
		}
		step.thick = density * step.length / crossed.volume.scale
		return step, true
	}
}

// scattered is the light a bit of volume sends toward the eye: what it
// scatters of every lamp's light that reaches it, past the surfaces and
// through the volumes in the way, evenly in all directions, and its glow.
func scattered(scene staged, crossed *crossing, step stride, ray maths.Ray,
	beam bool) maths.Light {
	local, point, dir := step.local, ray.Point(step.along), ray.Dir
	material := crossed.volume.source.Material
	if material == nil {
		material = grey
	}
	colour := material.Colour
	if crossed.volume.paint != nil {
		colour = crossed.volume.paint.At(local)
	}
	sum := emits(material, crossed.volume.shine, local, 0)
	if beam {
		sum = sum.Scale(boosted(crossed.volume, local, dir))
	}
	facing := dir.Neg()
	for _, one := range scene.asked(point, facing, byLight) {
		got := scene.shadowed(one, point)
		sum = sum.Add(colour.Times(got).Scale(evenly))
	}
	return sum
}

// boosted is how much brighter a bit of a volume looks for its beam,
// toward the eye that the ray runs away from: one for a volume at rest.
func boosted(volume *placedVolume, local, dir maths.Vec) float64 {
	if volume.source.Beam == nil {
		return 1
	}
	return volume.source.Beam.Boost(volume.at.Orientation.In(dir.Neg()),
		local)
}
