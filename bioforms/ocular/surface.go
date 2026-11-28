// Package ocular builds procedural reflectance in an eye's rest chart.
package ocular

import (
	"fmt"
	"math"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/noise"
)

// Pupil gives the half axes as fractions of the iris radius. Rotation is
// counterclockwise in the chart plane, in radians. Unequal axes produce
// an elliptical slit approximation.
type Pupil struct {
	HalfWidth, HalfHeight float64
	Rotation              float64
}

// Palette gives linear RGB reflectance. Inner is the peripupillary pigment.
type Palette struct {
	Stroma, Inner, Pupil, Sclera maths.Light
}

// Morphology sets the strength of each mark from zero to one.
type Morphology struct {
	Fibres, Crypts, Furrows, Limbus, InnerPigment, ScleralVessels float64
}

// Spec describes an anterior iris in a chart centred on the globe.
// X and Y lie across the visible iris, +Z points forward, and Radius is
// the limbus radius in chart units. Seed varies the marks within a profile.
type Spec struct {
	Radius     float64
	Pupil      Pupil
	Palette    Palette
	Morphology Morphology
	Seed       uint64
}

const cryptSectors = 23

type crypt struct {
	centre, width, depth, length, strength float64
}

// Surface holds an immutable eye texture for concurrent sampling.
type Surface struct {
	spec         Spec
	sine, cosine float64
	crypts       [cryptSectors]crypt
}

// NewSurface validates and constructs a texture for a single globe.
func NewSurface(spec Spec) (*Surface, error) {
	if !finite(spec.Radius) || spec.Radius <= 0 {
		return nil, fmt.Errorf("ocular: iris radius must be positive and finite")
	}
	pupil := spec.Pupil
	if !(pupil.HalfWidth >= .015 && pupil.HalfWidth <= .79 &&
		pupil.HalfHeight >= .015 && pupil.HalfHeight <= .79 && finite(pupil.Rotation)) {
		return nil, fmt.Errorf("ocular: pupil half axes must lie in [0.015, 0.79] with finite rotation")
	}
	palette := spec.Palette
	for _, colour := range [...]maths.Light{palette.Stroma, palette.Inner, palette.Pupil, palette.Sclera} {
		if !unit(colour.R) || !unit(colour.G) || !unit(colour.B) {
			return nil, fmt.Errorf("ocular: reflectance channels must lie in [0, 1]")
		}
	}
	morph := spec.Morphology
	for _, strength := range [...]float64{morph.Fibres, morph.Crypts, morph.Furrows,
		morph.Limbus, morph.InnerPigment, morph.ScleralVessels} {
		if !unit(strength) {
			return nil, fmt.Errorf("ocular: feature strengths must lie in [0, 1]")
		}
	}
	return newSurface(spec), nil
}

func newSurface(spec Spec) *Surface {
	sine, cosine := math.Sincos(spec.Pupil.Rotation)
	surface := &Surface{spec: spec, sine: sine, cosine: cosine}
	for sector := range surface.crypts {
		i := int64(sector)
		surface.crypts[sector] = crypt{
			centre:   .39 + .22*noise.Hash2(spec.Seed, i, 0),
			width:    .11 + .19*noise.Hash2(spec.Seed, i, 1),
			depth:    -.11 + .18*noise.Hash2(spec.Seed, i, 2),
			length:   .045 + .090*noise.Hash2(spec.Seed, i, 3),
			strength: maths.Smooth(.25, .88, noise.Hash2(spec.Seed, i, 4)),
		}
	}
	return surface
}

// At gives the eye's reflectance at a point in the chart.
func (surface *Surface) At(point maths.Vec) maths.Light {
	return surface.Within(point, 0)
}

// Within softens marks finer than the given footprint in chart units.
func (surface *Surface) Within(point maths.Vec, footprint float64) maths.Light {
	palette := surface.spec.Palette
	if surface.spec.Radius == 0 || !(point.Z > 0) || !finite(point.X) || !finite(point.Y) {
		return palette.Sclera
	}
	radius := math.Hypot(point.X, point.Y) / surface.spec.Radius
	width := math.Min(.6, math.Max(0, footprint)/surface.spec.Radius)
	soften := .35 * width
	if radius >= 2 || math.IsNaN(width) {
		return palette.Sclera
	}
	if radius == 0 {
		return palette.Pupil
	}
	angle := math.Atan2(point.Y, point.X) / (2 * math.Pi)
	sclera := surface.sclera(angle, radius, width)
	seed, limbus := surface.spec.Seed, surface.spec.Morphology.Limbus
	edgeBreak := limbus * (.025*(field(seed, angle, 17, 11)-.5) + .016*(field(seed, angle, 57, 12)-.5))
	irisShare := 1 - maths.Smooth(.968+edgeBreak-soften, 1.024+edgeBreak+soften, radius)
	if irisShare == 0 {
		return sclera
	}
	pupil := surface.spec.Pupil
	x := point.X / surface.spec.Radius
	y := point.Y / surface.spec.Radius
	u := surface.cosine*x + surface.sine*y
	v := surface.cosine*y - surface.sine*x
	edge := radius / math.Hypot(u/pupil.HalfWidth, v/pupil.HalfHeight)
	edge += .008 * (field(surface.spec.Seed, angle, 18, 0) - .5)
	pupilSoftness := math.Min(soften, .45*math.Min(pupil.HalfWidth, pupil.HalfHeight))
	irisFromPupil := maths.Smooth(-.004-pupilSoftness, .009+pupilSoftness, radius-edge)
	if irisFromPupil == 0 {
		return mix(sclera, palette.Pupil, irisShare)
	}
	progress := math.Max(0, math.Min(1, (radius-edge)/(1-edge)))
	iris := surface.stroma(angle, progress, radius, width)
	pupilRuff := 1 - maths.Smooth(.006, .029, radius-edge)
	ruffStrength := .27 * pupilRuff * (1 - maths.Smooth(.02, .09, width))
	iris = mix(iris, palette.Pupil, ruffStrength)
	return mix(sclera, mix(palette.Pupil, iris, irisFromPupil), irisShare)
}

func (surface *Surface) sclera(angle, radius, width float64) maths.Light {
	colour := surface.spec.Palette.Sclera
	strength := surface.spec.Morphology.ScleralVessels * (1 - maths.Smooth(.012, .12, width))
	fade := maths.Smooth(1.03, 1.37, radius) * (1 - maths.Smooth(1.70, 2, radius))
	if strength*fade == 0 {
		return colour
	}
	seed := surface.spec.Seed
	bend := .04 * (field(seed, angle+.16*radius, 7, 1) - .5)
	line := field(seed, angle+bend, 30, 2)
	vessels := strength * fade * maths.Smooth(.85, .99, line)
	return colour.Times(maths.RGB(1, 1-.25*vessels, 1-.21*vessels))
}

func (surface *Surface) stroma(angle, progress, radius, width float64) maths.Light {
	seed, palette, morph := surface.spec.Seed, surface.spec.Palette, surface.spec.Morphology
	coarse := 1 - maths.Smooth(.065, .26, width)
	middle := 1 - maths.Smooth(.022, .090, width)
	fine := 1 - maths.Smooth(.008, .035, width)

	point := stromaPoint{seed: seed, angle: angle, progress: progress}
	cloud := point.field(25, 5, 1) - .5
	collarette := .30 + .20*(field(seed, angle, 13, 3)-.5) + .12*(field(seed, angle, 43, 4)-.5)
	inside := 1 - maths.Smooth(collarette-.09, collarette+.17, progress+.15*coarse*cloud)
	inner := inside * (.71 + .29*maths.Smooth(-.20, .35, cloud)) * morph.InnerPigment
	colour := mix(palette.Stroma, palette.Inner, inner)

	// Bundle direction, span and contrast vary through the annulus.
	// Short fields share the bends of parent fields, then drift apart.
	point.angle += .012 * (point.field(17, 4, 2) - .5)
	bundle := point.field(62, 4, 3) - .5
	point.angle += .008*bundle + .004*progress
	strand := point.field(117, 6, 4) - .5
	point.angle += .004 * strand
	filament := point.field(211, 8, 5) - .5
	wisp := maths.Smooth(.10, .43, strand+.40*filament)
	thread := maths.Smooth(.21, .46, filament) * (.60 + .40*wisp)
	grain := .22*coarse*cloud + .33*middle*bundle + .48*middle*strand + .51*fine*filament
	fibres := morph.Fibres * (grain + .16*middle*wisp + .28*fine*thread)
	crest := (1 - maths.Smooth(.013, .071, math.Abs(progress-collarette))) *
		morph.Fibres * middle * (.05 + .19*wisp)
	crypts := surface.cryptRelief(angle, progress-collarette, strand) * morph.Crypts * middle
	furrow := surface.furrows(angle, progress) * morph.Furrows * middle
	rimStart := .85 + .14*(field(seed, angle, 19, 13)-.5)
	rimShade := .39 + .28*field(seed, angle, 31, 14)
	rim := morph.Limbus * maths.Smooth(rimStart, 1.01, radius) * rimShade
	reflectance := (1 + fibres + crest + crypts - .19*furrow) * (1 - rim)
	return capColour(colour.Scale(reflectance))
}

// stromaPoint samples irregular fibres with mostly radial spans. Each
// radial row has its own periodic angular lattice.
type stromaPoint struct {
	seed            uint64
	angle, progress float64
}

func (point stromaPoint) field(around, across, layer int64) float64 {
	radius := point.progress*float64(across) + .37
	row := int64(radius)
	seed := noise.Mix(point.seed + uint64(layer)*noise.Golden)
	first := field(seed, point.angle, around, row)
	last := field(seed, point.angle, around, row+1)
	return first + (last-first)*maths.Smooth(0, 1, radius-float64(row))
}

func (surface *Surface) cryptRelief(angle, progressFromCollarette, strand float64) float64 {
	at := angle * cryptSectors
	sector := int(math.Floor(at))
	mark := surface.crypts[(sector+cryptSectors)%cryptSectors]
	along := (progressFromCollarette - mark.depth) / mark.length
	across := (at - float64(sector) - mark.centre) / mark.width
	across += .14 * along
	opening := along*along + across*across
	shadow := 1 - maths.Smooth(.09, .84, opening)
	rim := maths.Smooth(.26, .65, opening) * (1 - maths.Smooth(.74, 1.10, opening))
	return mark.strength * ((.10+.21*strand)*rim - (.33-.20*strand)*shadow)
}

func (surface *Surface) furrows(angle, progress float64) float64 {
	var deepest float64
	for fold := range int64(3) {
		a := angle + .071*float64(fold)
		seed := surface.spec.Seed
		contour := .69 + .109*float64(fold) + .12*(field(seed, a, 9, 8)-.5) + .049*(field(seed, a, 23, 10)-.5)
		length := maths.Smooth(.49, .88, field(seed, a, 14, 9))
		breadth := .015 + .019*field(seed, a, 19, 15)
		groove := 1 - maths.Smooth(.004, breadth, math.Abs(progress-contour))
		deepest = math.Max(deepest, length*groove)
	}
	return deepest
}

// field is a smooth, periodic angular field. The seam wraps the lattice
// indices as well as the evaluation angle.
func field(seed uint64, turn float64, sectors, layer int64) float64 {
	turn -= math.Floor(turn)
	position := turn * float64(sectors)
	index := int64(math.Floor(position))
	first := noise.Hash2(seed, index%sectors, layer)
	last := noise.Hash2(seed, (index+1)%sectors, layer)
	blend := maths.Smooth(0, 1, position-float64(index))
	return first + (last-first)*blend
}

func finite(value float64) bool { return !math.IsNaN(value) && !math.IsInf(value, 0) }

func unit(value float64) bool { return value >= 0 && value <= 1 }

func mix(first, last maths.Light, share float64) maths.Light {
	return first.Scale(1 - share).Add(last.Scale(share))
}

func capColour(colour maths.Light) maths.Light {
	return maths.RGB(math.Max(0, math.Min(1, colour.R)), math.Max(0, math.Min(1, colour.G)),
		math.Max(0, math.Min(1, colour.B)))
}
