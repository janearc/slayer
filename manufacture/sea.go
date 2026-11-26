package manufacture

import (
	"errors"
	"fmt"
	"math"

	"github.com/janearc/slayer/arid"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/strategies"
	"github.com/janearc/slayer/oceantex"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
)

// crestOverSignificant is how many significant wave heights the highest
// crest or deepest trough of a sea's surface is allowed to stand from its
// mean: a height field of a million points, whose heights fall in a normal
// spread, reaches five of its standard deviations, a significant height
// being four, and the box is given twice the height to keep it whole.
const crestOverSignificant = 2.0 // picked

// errSea is the refusal of a sea with no edge that is asked its box and
// is not told the ground it lies within.
var errSea = errors.New("manufacture: a sea with no edge is asked its " +
	"box within the box it lies in")

// Sea is the water of a set: the oceantex bands of a wind, a fetch and a
// depth, at one moment and seed, either as a finite sheet sampled on a
// grid or, with no sheet, as the sea with no edge that arid holds whole.
// It is an object of the manufactory that makes no mesh: it puts a
// boundary in a world, and says in that world's ocean how it was made.
//
// The arid package owns the sea with no edge and does not import this one,
// so the object stays here, beside the things that are manufactured.
type Sea struct {
	// Model is the wind, fetch, depth, direction and swell of the sea.
	Model oceantex.Sea
	// Seconds is the moment of the sea's bands and Seed their seed.
	Seconds float64
	Seed    uint64
	// Sheet is the finite sheet the sea is sampled as, or nil for the sea
	// with no edge.
	Sheet *Sheet
	// At is where the sea's boundary stands in the world, its origin the
	// point its mean plane passes through.
	At maths.Subspace
	// Within is the box of the ground the sea lies in, which a sea with no
	// edge, or a sheet continued by an infinite mean plane, is held to when
	// it is asked its own box. It is a box of no volume until it is set.
	Within maths.Box
	// Optics is how the world's ocean says the sea was made: everything of
	// thing.OceanSurface but the boundary, the wave motion, the mean and
	// the extent, which staging puts in.
	Optics thing.OceanSurface
	// Motion is the recipe of a sea that moves with the scene clock, or
	// nil for a snapshot.
	Motion *thing.OceanWaveConfig
}

// Sheet is a finite sheet of the sea's bands: a square of samples.
type Sheet struct {
	// Extent is the sheet's side, metres, and Samples how many samples lie
	// along it.
	Extent  float64
	Samples int
	// Normal names the shading-normal mode, or is empty for the exact cell
	// normal.
	Normal string
	// Wake is the surface wake laid in the sheet, or nil for none.
	Wake *arid.SurfaceWake
	// Mean is the flat mean plane the sheet is continued by, or nil for
	// the sheet alone.
	Mean *MeanPlane
}

// MeanPlane is the flat plane a sheet is continued by past its edge.
type MeanPlane struct {
	// Level is the plane's height, metres, and Collar the width of the
	// fade from the sheet to the plane.
	Level, Collar float64
	// Infinite says the plane goes on without end, and not as far as the
	// collar's edge.
	Infinite bool
}

// Wear is the material a sea's boundary wears, found once the sea's shape
// is made: for the sea with no edge, open is its bands, which a glint is
// taken from, and for a sheet open is nil.
type Wear func(open *arid.Sea) (*thing.Material, error)

// Sea is an object.
var _ object = Sea{}

// edgeless is whether the sea has no edge of its own: it has no sheet, or
// its sheet is continued by a plane that goes on without end.
func (sea Sea) edgeless() bool {
	return sea.Sheet == nil || (sea.Sheet.Mean != nil &&
		sea.Sheet.Mean.Infinite)
}

// reach is how far the surface stands over or under its mean at most: the
// crest allowance of the sea's significant height, the wake's amplitude
// where a sheet has one, and the plane's level where a sheet is continued.
func (sea Sea) reach() float64 {
	reach := crestOverSignificant * sea.Model.Height()
	if sea.Sheet == nil {
		return reach
	}
	if sea.Sheet.Wake != nil {
		reach += math.Abs(sea.Sheet.Wake.AmplitudeMetres)
	}
	if sea.Sheet.Mean != nil {
		reach += math.Abs(sea.Sheet.Mean.Level)
	}
	return reach
}

// extent is the box of the sea, from its numbers alone.
func (sea Sea) extent(from seed) (maths.Box, error) {
	return sea.Extent()
}

// Extent is the box of the water, in the world's frame, without making its
// bands. A sheet's is its square, and its collar, about the boundary's
// origin; the water with no edge is its plane within the box of the ground
// it lies in. Both stand as high over their mean as the sea's significant
// height allows its crests, and as deep under it.
func (sea Sea) Extent() (maths.Box, error) {
	reach := sea.reach()
	if sea.edgeless() {
		within := sea.Within
		if !(within.Max.X > within.Min.X && within.Max.Z > within.Min.Z) {
			return maths.Box{}, errSea
		}
		level := sea.At.Origin.Y
		return maths.Box{Min: maths.V(within.Min.X, level-reach,
			within.Min.Z), Max: maths.V(within.Max.X, level+reach,
			within.Max.Z)}, nil
	}
	half := sea.Sheet.Extent / 2
	if sea.Sheet.Mean != nil {
		half += sea.Sheet.Mean.Collar
	}
	local := maths.Box{Min: maths.V(-half, -reach, -half),
		Max: maths.V(half, reach, half)}
	return transformed(local, sea.At), nil
}

// seen is how an eye sees the sea's box.
func (sea Sea) seen(from seed, at maths.Subspace,
	by strategies.Eye) (strategies.Seen, error) {
	return seenObject(sea, at, by)
}

// Staged adds the sea to a world as a set's ocean has always been added:
// its boundary a thing at the end of the things there are, wearing what
// wear finds, and the world's ocean surface made of what the sea says.
func (sea Sea) Staged(world *thing.World, wear Wear) error {
	if sea.Sheet == nil {
		return sea.stagedOpen(world, wear)
	}
	return sea.stagedSheet(world, wear)
}

// stagedOpen puts the sea with no edge into the world: the bands summed at
// every point of the plane. Its extent, which the sun's path reads, is the
// longest band's side, the distance after which the sea repeats.
func (sea Sea) stagedOpen(world *thing.World, wear Wear) error {
	open, err := arid.NewSea(sea.Model, sea.Seconds, sea.Seed)
	if err != nil {
		return fmt.Errorf("ocean: %w", err)
	}
	material, err := wear(open)
	if err != nil {
		return err
	}
	boundary := world.Add(thing.Thing{Shape: open, Material: material,
		Motion: thing.Still(sea.At)})
	sea.put(world, boundary, nil, open.Period())
	return nil
}

// stagedSheet puts the finite sheet into the world, continued by its mean
// plane where it has one.
func (sea Sea) stagedSheet(world *thing.World, wear Wear) error {
	sheet := sea.Sheet
	field, err := arid.NewHeightField(arid.HeightFieldConfig{
		Sea: sea.Model, Extent: sheet.Extent, Samples: sheet.Samples,
		Seed: sea.Seed, Seconds: sea.Seconds,
		SpectralShadingNormals: sheet.Normal == arid.SpectralSlopeNormals,
		SurfaceWake:            sheet.Wake,
	})
	if err != nil {
		return fmt.Errorf("ocean: %w", err)
	}
	if sheet.Normal == arid.SmoothNormalBilinearVertexSlopes {
		field, err = field.WithApproximateSmoothNormals()
		if err != nil {
			return fmt.Errorf("ocean shading-normal: %w", err)
		}
	}
	authored, err := sea.continued(field)
	if err != nil {
		return err
	}
	material, err := wear(nil)
	if err != nil {
		return err
	}
	boundary := world.Add(thing.Thing{Shape: authored, Material: material,
		Motion: thing.Still(sea.At)})
	var motion *thing.OceanWaveMotion
	if sea.Motion != nil {
		motion = thing.NewOceanWaveMotion(*sea.Motion)
	}
	sea.put(world, boundary, motion, sheet.Extent)
	return nil
}

// continued is the sheet's field, or the field continued by the flat mean
// plane the sheet says.
func (sea Sea) continued(field *shape.HeightField) (shape.Shape, error) {
	mean := sea.Sheet.Mean
	if mean == nil {
		return field, nil
	}
	if mean.Infinite {
		collared, err := arid.NewInfiniteMeanContinuation(field,
			mean.Level, mean.Collar)
		if err != nil {
			return nil, fmt.Errorf("ocean infinite mean-plane: %w", err)
		}
		return collared, nil
	}
	collared, err := arid.NewFlatMeanContinuation(field, mean.Level,
		mean.Collar)
	if err != nil {
		return nil, fmt.Errorf("ocean flat-mean: %w", err)
	}
	return collared, nil
}

// put says in the world how the ocean was made: the optics the sea holds,
// its boundary, wave motion, mean and extent.
func (sea Sea) put(world *thing.World, boundary *thing.Thing,
	motion *thing.OceanWaveMotion, extent float64) {
	surface := sea.Optics
	surface.Boundary = boundary
	surface.WaveMotion = motion
	surface.Mean = sea.At.Origin
	surface.ExtentMetres = extent
	world.OceanSurface = &surface
}
