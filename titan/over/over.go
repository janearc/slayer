// Package over lays made ground over a map's shape: detail finer than
// the map, on one fixed stretch of it, every part of it said to be made.
//
// The map is the shape. Every height it surveyed stays exactly where the
// survey put it, as a corner of the mesh. Between the survey's samples the
// stretch is cut finer, along the map's own triangles, so that before any
// detail the finer surface is the map's surface. Detail then moves each
// new corner along the shape's smoothed normal, not straight up, so on a
// wall it cuts into the wall and does not smear along it.
//
// The detail reads the shape, so it belongs to the place:
//
//   - flutes are channels cut inward along the fall line, deeper where more
//     ground drains through, on ground steeper than a stated slope;
//   - beds are level ledges at a stated spacing in height, standing out of
//     the face, on ground steeper than a stated slope;
//   - roughness is finer than both. It is read from the map only at the
//     scale the map resolves: how broken it is over a neighbourhood of
//     several samples, never from one facet or one crease. It is made only
//     at wavelengths shorter than the map's own, so it cannot fight the
//     shape, as steep at every scale as the neighbourhood says and no steeper
//     than the ground, so that nothing made at a centimetre stands a metre
//     tall.
//
// The numbers are passed in and no place is named here.
//
// Detail is nothing at a surveyed sample and eases up to its full size
// half a spacing away, and it eases in from the stretch's edge, so the
// stretch meets the plain ground along corners that are the same. However
// the rules add up, no corner moves further than a stated share of the
// map's spacing: that is the bound, and the receipt says how near to it
// the stretch came.
//
// The same rules and seed give the same ground.
//
// Seen lays the whole map instead, each cell as finely as an eye resolves
// it there: cut in halves each way as often as its pixels ask, none where
// the survey is as fine as the eye. Where a cell meets a coarser one, its
// edge is laid on the coarser cell's, so the two meet without a gap. The
// made faces are bounded by a count, so what is made grows with the
// picture and not with the ground.
package over

import (
	"fmt"

	"github.com/janearc/slayer/mesh"
)

// Map is a grid of surveyed heights: so many samples across, eastward, and
// down, southward, so many metres apart. terrain.Window is one.
type Map interface {
	Size() (across, down int, step float64)
	At(across, down int) float64
}

// Stretch is the cells of the map that are laid finer: its first cell's
// column and row, and how many cells it is across and down.
type Stretch struct {
	Column, Row, Across, Down int
}

// Flutes are channels cut along the fall line.
type Flutes struct {
	// Slope is the rise over run on ground steeper than which flutes are
	// cut. They come in over the next quarter of it.
	Slope float64
	// Depth is how deep a flute is cut, metres, where Drainage drains.
	// Nought cuts none.
	Depth float64
	// Drainage is the area, square metres, that drains through a place for
	// its flutes to be cut to Depth. Less drains less deep.
	Drainage float64
	// Width is the metres between flutes, across the fall line.
	Width float64
	// Length is how many widths a flute runs down the fall line.
	Length float64
}

// Beds are level ledges standing out of the face, as stacked layers of
// rock do.
type Beds struct {
	// Slope is the rise over run on ground steeper than which beds show.
	// They come in over the next quarter of it.
	Slope float64
	// Spacing is the metres of height between one ledge and the next.
	Spacing float64
	// Standout is how far a ledge stands out of the face, metres. Nought
	// makes none.
	Standout float64
}

// Roughness is relief made at wavelengths shorter than the map's own,
// from how broken the map is over a neighbourhood of seven samples square.
type Roughness struct {
	// Gain is how much of the map's spread over a neighbourhood, per metre
	// of the neighbourhood's width, is made as steepness at every
	// wavelength. Nought makes none.
	Gain float64
	// Stand is the steepest, rise over run, made relief may lean off the
	// ground it is on, all its octaves together, by their root mean
	// square; its steepest place is peakShare times more. It is also kept
	// under the ground's own slope.
	Stand float64
	// Continued is whether relief falls with scale as the survey's own
	// does, read from how much rougher its wider neighbourhoods are than
	// its narrower. Without it, relief is as steep at every scale.
	Continued bool
}

// Rules are the numbers the ground is laid by. None names a place.
type Rules struct {
	// Seed makes the same ground each time.
	Seed uint64
	// Spacing is the finest wanted, metres between corners along the
	// stretch's surface, along each way of the map.
	Spacing float64
	// Faces is the most faces the stretch may have. If Spacing would
	// make more, the spacing is made coarser until it fits.
	Faces int
	// Departure is the most any corner may move, as a share of the map's
	// spacing.
	Departure float64
	// Margin is how far in from the stretch's edge detail eases in,
	// metres.
	Margin    float64
	Flutes    Flutes
	Beds      Beds
	Roughness Roughness
}

// Receipt is what was laid, said plainly. Every corner counted as made is
// made; no survey saw it.
type Receipt struct {
	// Surveyed is the corners the survey gave, all of them in the mesh.
	// Made is the corners laid between them.
	Surveyed, Made int
	// Faces is the faces of the stretch.
	Faces int
	// Finest and Coarsest are the shortest and longest edges along the
	// map's ways in the stretch, metres, after detail; where the ground is
	// laid as an eye sees it, of every edge of the made faces.
	Finest, Coarsest float64
	// Bound is the most a corner may move, metres. Largest is the most
	// one did.
	Bound, Largest float64
	// Median, Ninetieth and Ninetyninth are how far the made corners
	// moved, metres, at those shares of them, in order of how far. Near is
	// the share of made corners that moved within a tenth of the bound.
	Median, Ninetieth, Ninetyninth, Near float64
	// Sag is the most the stretch's facets stand off the map's own
	// triangles, vertically, metres. Every facet lies in one of them, so it
	// is nothing but the rounding of the arithmetic.
	Sag float64
	// Moved is the most any surveyed corner stands from where the survey
	// put it, metres, read from the mesh.
	Moved float64
	// Cells is how many of the survey's cells were laid finer, where the
	// ground was laid as an eye sees it, and Doubled how many times the
	// spacing the eye wanted was doubled everywhere to fit the faces.
	Cells, Doubled int
	// Scaling is the power of width the made roughness grows by: the
	// survey's own, read, where relief is continued, or one.
	Scaling float64
}

// String is the receipt in lines, and a line of what the eye asked where
// the ground was laid as an eye sees it.
func (receipt Receipt) String() string {
	said := receipt.laid()
	if receipt.Cells > 0 {
		said += fmt.Sprintf("\nmade ground: %d cells laid as the eye "+
			"sees them, the spacing it wanted doubled %d times to fit; "+
			"roughness grows as width to the power %.2f",
			receipt.Cells, receipt.Doubled, receipt.Scaling)
	}
	return said
}

// laid is the receipt's lines of what was laid.
func (receipt Receipt) laid() string {
	return fmt.Sprintf("made ground: %d corners surveyed, %d made, "+
		"%d faces in the stretch\n"+
		"made ground: edges %.3f to %.3f m; surveyed corners moved %g m; "+
		"facets sag %.2g m\n"+
		"made ground: made corners moved %.3f m at the median, %.3f at "+
		"the 90th, %.3f at the 99th, %.3f at most, of %.3f m allowed; "+
		"%.2f%% moved within a tenth of the bound",
		receipt.Surveyed, receipt.Made, receipt.Faces, receipt.Finest,
		receipt.Coarsest, receipt.Moved, receipt.Sag, receipt.Median,
		receipt.Ninetieth, receipt.Ninetyninth, receipt.Largest,
		receipt.Bound, 100*receipt.Near)
}

// Lay is the whole map as a mesh, x east and z south from its first
// sample and y up from the floor, with one stretch of it laid finer under
// the rules. Outside the stretch it is the map's own triangles, two to a
// cell, shaded smooth across the samples. The floor is the height taken
// from every sample, so that the mesh stands near its own origin.
func Lay(from Map, floor float64, stretch Stretch, rules Rules) (*mesh.Mesh,
	Receipt, error) {
	read := survey(from, floor)
	if err := check(read, stretch, rules); err != nil {
		return nil, Receipt{}, err
	}
	grid := plan(read, stretch, rules)
	return grid.lay(read, rules).assemble(read)
}

// check is what is wrong with a stretch and its rules, if anything.
func check(read *shape, stretch Stretch, rules Rules) error {
	switch {
	case read.across < 2 || read.down < 2:
		return fmt.Errorf("over: a map %d by %d samples has no cells",
			read.across, read.down)
	case stretch.Column < 0 || stretch.Row < 0 || stretch.Across < 1 ||
		stretch.Down < 1 || stretch.Column+stretch.Across >= read.across ||
		stretch.Row+stretch.Down >= read.down:
		return fmt.Errorf("over: stretch %+v is not cells of the map", stretch)
	case !(rules.Spacing > 0) || rules.Faces < 2*stretch.Across*stretch.Down:
		return fmt.Errorf("over: spacing %g m and %d faces say nothing",
			rules.Spacing, rules.Faces)
	}
	return checkRules(rules)
}

// checkRules is what is wrong with the rules, whatever stretch they lay.
func checkRules(rules Rules) error {
	switch {
	case !(rules.Departure > 0) || rules.Margin < 0:
		return fmt.Errorf("over: departure %g and margin %g m say nothing",
			rules.Departure, rules.Margin)
	case rules.Flutes.Depth > 0 && !(rules.Flutes.Width > 0 &&
		rules.Flutes.Length > 0 && rules.Flutes.Drainage > 0 &&
		rules.Flutes.Slope > 0):
		return fmt.Errorf("over: flutes %+v cut nowhere", rules.Flutes)
	case rules.Beds.Standout > 0 && !(rules.Beds.Spacing > 0 &&
		rules.Beds.Slope > 0):
		return fmt.Errorf("over: beds %+v stand nowhere", rules.Beds)
	case rules.Roughness.Gain > 0 && !(rules.Roughness.Stand > 0):
		return fmt.Errorf("over: roughness %+v leans nowhere", rules.Roughness)
	}
	return nil
}
