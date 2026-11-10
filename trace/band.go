package trace

import (
	"fmt"
	"math"
	"time"

	"github.com/janearc/slayer/emit"
	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/render/frame"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
)

// passes is how many things a ray may pass through in a band that gets
// through them, before what is behind is taken as dark.
const passes = 8

// RenderIn is Render in one band of the spectrum: what a sensor for that band
// would record, a narrow filter's picture even within the visible.
//
// A lamp that carries a spectrum, an emit.Banded, lights what scatters the
// band as a lamp lights a matte surface in the visible, and a radar lights what
// echoes it in its own band. What shows is what gives off the band and what
// those lamps light, through what passes it.
//
// A thing whose material has no Radiant is opaque and dark; one with a Radiant
// passes the band by its Through to the power of the path through it, and adds
// what it Emits.
//
// A volume with a Radiant does the same wherever it has any density, and one
// without thins the band as it thins light. The sky is dark. The pixel's light
// is the band's strength, the same in all three parts. The zero band is the
// whole visible at once, which is Render.
//
// Nothing bends in a band: a world's fields are left out of it.
//
// What Render refuses is refused, and so is a band whose wavelength is under
// zero or not a number.
func RenderIn(world *thing.World, eye maths.Subspace, lens film.Sensor,
	moment time.Duration, width, height int,
	band maths.Band) (*frame.Frame, error) {
	seen, err := newView(world, eye, lens, moment, width, height, 1)
	if err != nil {
		return nil, err
	}
	if !finite(band.Nanometres) || band.Nanometres < 0 {
		return nil, fmt.Errorf("trace: a band has a wavelength, or "+
			"nought for the visible, and this is %v nanometres",
			band.Nanometres)
	}
	if band.Nanometres == 0 {
		// the zero band is the whole visible at once
		return render(seen), nil
	}
	return renderIn(seen, band), nil
}

// renderIn is RenderIn of a view, which has been checked, in a band
// that is not the zero band.
func renderIn(seen view, band maths.Band) *frame.Frame {
	width, height := seen.width, seen.height
	// nothing bends in a band: the fields are left out
	scene := stage(seen.world, seen.at)
	scene.fields = nil
	return paint(width, height, func(x, y int) frame.Pixel {
		ray := film.Look(seen.eye, seen.lens, x, y, width, height,
			band)
		return inBand(scene, ray, band, passes)
	})
}

// inBand is what one ray brings back in a band outside the visible: the
// first thing met gives off what it gives off, echoes what radars reach
// it, and passes what it passes of what lies behind, and so on through
// what passes the band.
func inBand(scene staged, ray maths.Ray, band maths.Band,
	left int) frame.Pixel {
	met, found := scene.crowd.first(ray, 0)
	far := math.Inf(1)
	if found {
		far = met.dist
	}
	before := throughIn(scene, ray, far, band)
	cover, strength := 1-before.keep, before.added
	if found {
		cover = 1
		strength += before.keep * given(scene, met, ray, band, left)
	}
	return pixelOf(maths.Grey(strength), float32(cover), float32(far))
}

// given is what a thing a ray has met hands the ray in a band, before
// the volumes in front of it take their share: what it gives off, what
// it echoes of the radars that reach it, and what it passes of what
// lies behind it. A thing whose material has no radiant gives nothing.
func given(scene staged, met hit, ray maths.Ray, band maths.Band,
	left int) float64 {
	material := met.thing.thing.Material
	if material == nil || material.Radiant == nil {
		return 0
	}
	strength := material.Radiant.Emits(band)
	strength += echo(scene, met.thing, ray, met.dist, band)
	strength += lamplitIn(scene, met.thing, ray, met.dist, band)
	strength += passedThrough(scene, met, ray, band, left)
	return strength
}

// lamplitIn is what a thing sends toward the eye in a band of the lamps that
// light in it, as lamplight is in the visible: a matte surface sends out every
// way alike what lands on it, its Scatters over pi times the band's power
// landing there, by how squarely it lands.
//
// A surface in the way stops a lamp, and the volumes in the way take their
// share of the band.
//
// A lamp lights in a band if it is an emit.Banded; a radar's beam is echo's.
func lamplitIn(scene staged, placed placedThing, ray maths.Ray, dist float64,
	band maths.Band) float64 {
	back := placed.thing.Material.Radiant.Scatters(band)
	if back <= 0 {
		return 0
	}
	point := ray.Point(dist)
	normal := placed.facing(point, ray.Dir)
	sum := 0.0
	for _, lamp := range scene.lamps {
		banded, is := lamp.(emit.Banded)
		if !is {
			continue
		}
		power := banded.Lands(point, band)
		toward, far, _ := lamp.Light(point, normal)
		square := normal.Dot(toward)
		if power <= 0 || square <= 0 {
			continue
		}
		from := scene.lifted(point, normal)
		shadow := maths.Ray{At: from, Dir: toward}
		if scene.crowd.blocked(shadow, far) {
			continue
		}
		keep := throughIn(scene, shadow, far, band).keep
		sum += back / math.Pi * power * square * keep
	}
	return sum
}

// passedThrough is what shows through a thing in a band that it passes: what a
// ray going on from its far side brings back, times the thing's Through to the
// power of the way through it.
//
// It is zero for a thing that passes none of the band, for one whose shape
// cannot say where a ray leaves it, and once left has run out.
//
// The way through is measured in the thing's own units, not the world's, as a
// medium's density is: a ball of glass passes the same share of a band at any
// scale, and a thing scaled up keeps its look.
func passedThrough(scene staged, met hit, ray maths.Ray, band maths.Band,
	left int) float64 {
	placed := met.thing
	transmission := placed.thing.Material.Radiant.Through(band)
	spanner, can := placed.thing.Shape.(shape.Spanner)
	if transmission <= 0 || left <= 0 || !can {
		return 0
	}
	enter, leave, crossed := spanner.Span(placed.at.RayIn(ray), 0, nowhere)
	if !crossed {
		return 0
	}
	past := leave * placed.scale
	out := ray.Point(past + scene.lifting(ray.Point(past)))
	onward := maths.Ray{At: out, Dir: ray.Dir}
	behind := inBand(scene, onward, band, left-1)
	return math.Pow(transmission, leave-enter) * float64(behind.R)
}

// echo is what a thing scatters back of the radars that reach it in a
// band: for each beam, how squarely it lands times how rough the thing
// is there times what it returns, if no surface stands in the way and
// through whatever volumes do.
func echo(scene staged, placed placedThing, ray maths.Ray, dist float64,
	band maths.Band) float64 {
	material := placed.thing.Material
	back := material.Radiant.Scatters(band)
	if back <= 0 {
		return 0
	}
	point := ray.Point(dist)
	local := placed.at.In(point)
	normal := placed.facing(point, ray.Dir)
	// how rough the thing is there, as its roughness is at the
	// picture's time: rough everywhere alike without one
	rough := 1.0
	if placed.rough != nil {
		rough = placed.rough.At(local).Luminance()
	}
	sum := 0.0
	for _, lamp := range scene.lamps {
		sum += back * rough * scene.beamed(lamp, point, normal, band)
	}
	return sum
}

// beamed is how much of one lamp's beam in a band lands on a point of a
// surface: the beam's strength there times how squarely it lands, through the
// volumes in its way. It is zero for a lamp that is no radar, one that does not
// beam the band, and one a surface hides.
func (scene staged) beamed(lamp thing.Lamp, point, normal maths.Vec,
	band maths.Band) float64 {
	beamer, is := lamp.(film.Beamer)
	if !is {
		return 0
	}
	strength := beamer.Beams(band)
	toward, far, _ := lamp.Light(point, normal)
	square := normal.Dot(toward)
	if strength <= 0 || square <= 0 {
		return 0
	}
	shadow := maths.Ray{At: scene.lifted(point, normal), Dir: toward}
	if scene.crowd.blocked(shadow, far) {
		return 0
	}
	keep := throughIn(scene, shadow, far, band).keep
	return square * strength * keep
}

// bandPassage is a passage in one band: how much of the band from
// behind gets through, and how much the volumes give off in front of
// it, each one number.
type bandPassage struct {
	keep, added float64
}

// throughIn is what the ray gathers from the volumes in a band outside
// the visible: how much of the band from far gets through, and how much
// the volumes give off in front of it.
func throughIn(scene staged, ray maths.Ray, far float64,
	band maths.Band) bandPassage {
	keep, added := 1.0, 0.0
	across := crossings(scene.volumes, ray, far)
	steps := stepping(across, strides)
	steps.refine = scene.beams.Ray(ray, far)
	// the march stops on the test the eye's march stops on
	for shows(maths.Grey(keep)) {
		step, found := steps.next()
		if !found {
			break
		}
		crossed, local := &across[step.crossing], step.local
		material := crossed.volume.source.Material
		if air := crossed.volume.gas; air != nil {
			// a gas passes the band by its own extinction there,
			// and gives off nothing in it: its lines are not wired
			// yet
			keep *= math.Exp(-air.Extinction(band) * step.thick)
			continue
		}
		absorb := 1 - math.Exp(-step.thick)
		if material != nil && material.Radiant != nil {
			// what the volume gives off in the band, where it is,
			// as much as a volume that thick absorbs of light. that
			// is meant to hold for one that passes all of the band
			// too: it glows and takes nothing
			emit := material.Radiant.Emits(band) *
				boosted(crossed.volume, local, ray.Dir)
			if lit, is := material.Radiant.(thing.Lit); is {
				// an ionisation front: the gas glows where the
				// lamps reach it, as its line wants
				at := ray.Point(step.along)
				light := shone(scene, at, ray.Dir.Neg())
				emit *= math.Pow(light, lit.Excitation(band))
			}
			added += keep * absorb * emit
			// the band passes by transmission to the power of the
			// path at unit density: thin gas is clear
			transmission := material.Radiant.Through(band)
			absorb = 1 - math.Pow(transmission, step.thick)
		}
		keep *= 1 - absorb
	}
	return bandPassage{keep: keep, added: added}
}
