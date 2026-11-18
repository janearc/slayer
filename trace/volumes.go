package trace

import (
	"math"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/render/beamstep"
	"github.com/janearc/slayer/scene/gas"
	"github.com/janearc/slayer/scene/medium"
	"github.com/janearc/slayer/scene/thing"
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
func through(scene *staged, ray maths.Ray, far float64) passage {
	return throughGain(scene, ray, far, nil, strides)
}

// throughClear interleaves water extinction, water single-scattering, and
// explicitly opted-in bounded clouds. Each local source is attenuated by the
// water and cloud between it and the eye; later samples continue through
// earlier segments. Transparent-boundary refraction remains unsupported.
func throughClear(scene *staged, seen look, far float64,
	water *thing.ClearMedium) passage {
	ray := seen.ray
	passed := untouched()
	var crossingRoom [roomy]crossing
	var gainRoom [roomy]float64
	var takenRoom, partsRoom [roomy]int
	across := crossingsIn(crossingRoom[:0], scene.volumes, ray, far)
	gains := gainRoom[:0]
	for range across {
		gains = append(gains, 1)
	}
	steps := steppingIn(across, strides, takenRoom[:], partsRoom[:])
	steps.refine = scene.beams.Ray(ray, far)
	cursor := 0.0
	for shows(passed.keep) {
		step, found := steps.next()
		if !found {
			break
		}
		gap := math.Max(0, step.along-cursor)
		passed.added = passed.added.Add(clearWaterScatterSegment(scene,
			seen, ray, cursor, gap, far, water, passed.keep))
		attenuatePassage(&passed, waterKeep(water, ray, gap))
		crossed, index := &across[step.crossing], step.crossing
		if crossed.volume.gas != nil {
			passed = gasStep(scene, crossed, step, ray, gains[index], passed)
		} else {
			absorb := absorbed(crossed.volume, step.thick)
			light := scatteredClear(scene, crossed, step, seen, water)
			passed.added.R += light.R * passed.keep.R * absorb.R
			passed.added.G += light.G * passed.keep.G * absorb.G
			passed.added.B += light.B * passed.keep.B * absorb.B
			passed.keep.R *= 1 - absorb.R
			passed.keep.G *= 1 - absorb.G
			passed.keep.B *= 1 - absorb.B
			if scene.walked != nil {
				strode(scene.walked, crossed, step, absorb, light,
					passed.keep)
			}
		}
		cursor = step.along
	}
	gap := math.Max(0, far-cursor)
	passed.added = passed.added.Add(clearWaterScatterSegment(scene,
		seen, ray, cursor, gap, far, water, passed.keep))
	attenuatePassage(&passed, waterKeep(water, ray, gap))
	return passed
}

// clearWaterScatterSegment adds the ocean's single-scattered sky source over
// one water-only gap. The incoming camera-path transmission includes every
// earlier water segment and local cloud step. oceanClearInscatter evaluates
// the refracted air-side source and its incoming water/cloud path at each
// sample, then applies outgoing water Beer within this gap.
func clearWaterScatterSegment(scene *staged, seen look, ray maths.Ray,
	begin, length, fullDistance float64, water *thing.ClearMedium, keep maths.Light,
) maths.Light {
	if water == nil || !water.HasScattering() || scene.oceanSun == nil ||
		!sameClearMedium(scene.oceanSun.medium, water) || length <= 0 {
		return maths.Light{}
	}
	segmentRay := maths.Ray{At: ray.Point(begin), Dir: ray.Dir}
	segmentSeen := seen
	segmentSeen.ray = segmentRay
	steps := water.InscatterSteps()
	if scene.oceanSun.waterStepMode == "ray-spacing" {
		var valid bool
		steps, valid = clearCloudWaterSteps(ray, fullDistance, length, water)
		if !valid {
			markClearInscatterFault(scene, clearInscatterNumericalOverflow)
			return maths.Light{}
		}
	}
	added := oceanClearInscatterSteps(scene, segmentSeen, segmentRay, length,
		water, steps)
	return added.Times(keep)
}

// clearCloudWaterSteps preserves the full ray's target water cell spacing in
// each cloud gap. Ceil leaves each positive gap at least one cell; boundaries
// can therefore add cells. This explicit quality tradeoff is not bit-exact.
func clearCloudWaterSteps(ray maths.Ray, full, gap float64, water *thing.ClearMedium) (int, bool) {
	metres := ray.Dir.Len() * water.MetresPerUnit()
	if !finite(metres) || math.IsNaN(full) || math.IsNaN(gap) || metres <= 0 || full <= 0 || gap <= 0 || water.InscatterSteps() < 1 {
		return 0, false
	}
	fullMetres := math.Min(full*metres, water.MaxInscatterDistanceMetres())
	gapMetres := math.Min(gap*metres, water.MaxInscatterDistanceMetres())
	cell := fullMetres / float64(water.InscatterSteps())
	if !finite(fullMetres, gapMetres, cell) || cell <= 0 {
		return 0, false
	}
	count := math.Ceil(gapMetres / cell)
	if !finite(count) || count > float64(water.InscatterSteps()) {
		return 0, false
	}
	return max(1, int(count)), true
}

// scatteredClear returns the light a volume adds at one step of the march.
// Where the water is the ocean sun medium, it adds the ocean clear-cloud
// source as well.
func scatteredClear(scene *staged, crossed *crossing, step stride,
	seen look, water *thing.ClearMedium) maths.Light {
	point, direction := seen.ray.Point(step.along), seen.ray.Dir
	material := crossed.volume.source.Material
	if material == nil {
		material = grey
	}
	sum := emits(material, crossed.volume.shine, step.local, 0)
	if crossed.volume.source.Beam != nil {
		sum = sum.Scale(boosted(crossed.volume, step.local, direction))
	}
	if material.Colour == (maths.Light{}) || scene.oceanSun == nil ||
		!sameClearMedium(scene.oceanSun.medium, water) {
		return sum
	}
	source, usable := oceanClearSourceWithPhase(scene, point, seen,
		direction.Unit(), water, clearCloudPhaseOf(crossed.volume.med))
	if usable {
		sum = sum.Add(material.Colour.Times(source))
	}
	return sum
}

// waterKeep returns the light kept through the water over distance, as Beer
// attenuation per channel. With no water, all light is kept.
func waterKeep(water *thing.ClearMedium, ray maths.Ray,
	distance float64) maths.Light {
	if water == nil || distance <= 0 {
		return maths.Grey(1)
	}
	metres := distance * ray.Dir.Len() * water.MetresPerUnit()
	extinction := water.Extinction()
	return maths.RGB(beerClear(extinction.R, metres),
		beerClear(extinction.G, metres), beerClear(extinction.B, metres))
}

// attenuatePassage multiplies the light kept by a passage by keep.
func attenuatePassage(passed *passage, keep maths.Light) {
	passed.keep = passed.keep.Times(keep)
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
func throughGain(scene *staged, ray maths.Ray, far float64, gain gainer,
	cuts int) passage {
	passed := untouched()
	// the ray's own scratch, on this call's stack, so that a march
	// through a few volumes allocates nothing and no two goroutines
	// share it
	var acrossRoom [roomy]crossing
	var gainRoom [roomy]float64
	var takenRoom, partsRoom [roomy]int
	across := crossingsIn(acrossRoom[:0], scene.volumes, ray, far)
	gains := gainRoom[:0]
	for _, crossed := range across {
		scale := 1.0
		if gain != nil {
			scale = gain(crossed,
				ray.Point((crossed.enter+crossed.leave)/2))
		}
		gains = append(gains, scale)
	}
	steps := steppingIn(across, cuts, takenRoom[:], partsRoom[:])
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

// roomy is how many crossings a march keeps in scratch on the stack
// before it asks the heap for more.
const roomy = 8

// stepping is a stepper over the crossings, each cut into cuts steps.
func stepping(across []crossing, cuts int) *stepper {
	return steppingIn(across, cuts, nil, nil)
}

// steppingIn is stepping with the counts kept in taken and parts, which
// MUST hold zeros, when they are as long as across; otherwise it makes
// its own.
func steppingIn(across []crossing, cuts int, taken, parts []int) *stepper {
	if len(taken) < len(across) || len(parts) < len(across) {
		taken, parts = make([]int, len(across)), make([]int, len(across))
	}
	return &stepper{across: across, cuts: cuts, taken: taken[:len(across)],
		parts: parts[:len(across)]}
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
func scattered(scene *staged, crossed *crossing, step stride, ray maths.Ray,
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
	if colour == (maths.Light{}) {
		return sum
	}
	// Colour is the painted medium's single-scattering albedo in the new
	// opt-in model. Emission remains separate and is not phase-normalized.
	phase := evenly
	if len(scene.volumeSky) > 0 {
		phase = thing.IsotropicPhase()
		sum = sum.Add(colour.Times(scene.volumeSkyAt(point)))
	}
	facing := dir.Neg()
	lobe := asymmetryOf(crossed.volume.source.Medium)
	var room [lampsEvery]lit
	for _, one := range scene.asked(room[:], point, facing, byLight) {
		got := scene.shadowed(one, point)
		if lobe != 0 && !one.ambient() {
			got = got.Scale(lobed(lobe, one.toward, dir))
		}
		sum = sum.Add(colour.Times(got).Scale(phase))
	}
	return sum
}

// lopsided is a medium that scatters more one way than another, by a
// Henyey-Greenstein asymmetry: a mist that says one.
type lopsided interface{ Asymmetry() float64 }

// asymmetryOf is a medium's asymmetry, nought for one that scatters
// evenly or does not say.
func asymmetryOf(made medium.Medium) float64 {
	if said, ok := made.(lopsided); ok {
		return said.Asymmetry()
	}
	return 0
}

// lobed is how many times the even share a lopsided medium sends toward
// the eye of a lamp's light: its lobe at the angle from the lamp's way
// of travel, minus toward, to the eye's, minus dir, over the even
// lobe's one in four pi. It is gas's PhaseAt, as the kernel's
// gas_phase is, so the two tracers take the same lobe.
func lobed(asymmetry float64, toward, dir maths.Vec) float64 {
	cosine := toward.Unit().Neg().Dot(dir.Neg())
	return gas.Gas{Phase: asymmetry}.PhaseAt(cosine) * fullSphere
}

// fullSphere is the solid angle of the whole sphere, four pi steradians,
// what an even lobe's one in four pi is taken over. Defined.
const fullSphere = 4 * math.Pi

// boosted is how much brighter a bit of a volume looks for its beam,
// toward the eye that the ray runs away from: one for a volume at rest.
func boosted(volume *placedVolume, local, dir maths.Vec) float64 {
	if volume.source.Beam == nil {
		return 1
	}
	return volume.source.Beam.Boost(volume.at.Orientation.In(dir.Neg()),
		local)
}
