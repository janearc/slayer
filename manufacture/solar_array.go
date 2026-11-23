package manufacture

import (
	"errors"
	"math"
	"slices"

	"github.com/janearc/slayer/maths"
)

// A solar array is built from boxes and tubes: a substrate panel with a
// frame round it, cells laid on it in a grid with a gap between them,
// bus bars in the gaps between columns, an optional cover glass over
// each cell, a hinge line with a fitting at each end, a yoke from the bus
// with a fitting at its root, and a harness along the yoke.
//
// Every number
// is the craft's: how many panels and how big, the cells and their
// pitch, where the hinge lines fall, how long the yoke is and which face
// carries the cells.
//
// One maker serves a rigid wing and a rolled one. A rolled wing is a
// blanket between a spreader bar at each end and a pair of bi-stem
// booms along its long edges; its panel is built the same way, and only
// its edges differ.
//
// The array is laid in its own frame. The run goes out along x from the
// root on the bus, the panels lie across z, and the cells sit on the +y
// face. A craft turns that frame into its own with solarFrame.

const (
	// solarRailThickness is how many panel thicknesses wide a frame rail
	// or a spreader bar is wanted. Chosen, as no source gives one.
	solarRailThickness = 2.0
	// solarRailShare is the most of the bare margin between the cells and
	// the panel's edge that a rail takes, so a rail never reaches a cell.
	// Chosen.
	solarRailShare = 0.6
	// solarRailProud is how far a frame stands proud of the substrate's
	// edge and back face, as a share of the rail's width, so that their
	// faces are not in one plane. Chosen.
	solarRailProud = 0.1
	// solarHingeShare is a hinge barrel's radius as a share of the
	// panel's thickness. Chosen.
	solarHingeShare = 0.75
	// solarFittingGirth is how many radii across a hinge or root fitting
	// is. Chosen.
	solarFittingGirth = 3.0
	// solarHarnessShare is the harness's radius as a share of the yoke's.
	// Chosen.
	solarHarnessShare = 0.4
	// solarBoomShare is the radius of one tube of a bi-stem boom as a
	// share of the spreader bar's width. Chosen.
	solarBoomShare = 0.25
)

// solarCover is the matter of a cover glass. It is after every matter
// a spacecraft already names, so a craft that wants a cover dresses it.
const solarCover slot = romanGoldBlanket + spaceOne

// solarBar is the matter of a polished bus bar, for a craft whose mirror
// is held to its own extent: a bar dressed as the mirror would be part of
// the mirror. A craft that wants one dresses it.
const solarBar slot = solarCover + spaceOne

// solarMetal is the matter of an array's frame and hardware, for a craft
// whose structure is held to a count of faces: an array's rails, hinges
// and fittings dressed as the structure would be one mesh with it. A
// craft that wants one dresses it.
const solarMetal slot = solarBar + spaceOne

var errSolarArray = errors.New("manufacture: a solar array needs panels " +
	"and cells of a size above nought, and a cell grid that fits the panel")

// solarLooks is the matter each kind of part of an array is made of. The
// cells and covers should be matters that hold nothing else, so that
// shareSolar can store them once. The substrate and the frame must be
// different matters, as they share edges, which a mesh of one matter
// cannot hold.
type solarLooks struct {
	cell, cover, bar, substrate, frame, hardware slot
}

// solarPanels is the panels of one array: how many go along the run and
// how many across it, each width along the run by depth across it, and
// the gap between them. Reach is how far the first hinge line is from
// the root, and fold is the angle each panel is turned past the one
// before it about their hinge line, nought for a deployed array.
type solarPanels struct {
	along, across                int
	width, depth, thickness, gap float64
	reach, fold                  float64
}

// solarCells is the grid on each panel: columns along the run and rows
// across it, each cell its width by depth and height above the
// substrate, laid at a pitch. A bus bar fills barShare of the gap
// between two columns. A cover, when it is above nought, is a layer that
// thick over each cell. Rows from secondFrom on, when that is above
// nought, are the second string and are of the matter second.
type solarCells struct {
	columns, rows        int
	width, depth, height float64
	pitchX, pitchZ       float64
	barShare, cover      float64
	second               slot
	secondFrom           int
}

// solarYoke is the arm from the bus to the array: a tube from the root
// to its tip along the run, at each of the offsets across it. Its ends
// sit so far behind the mid-plane of the panels, away from the cells.
// A length of nought is no yoke.
type solarYoke struct {
	length, radius        float64
	rootOffset, tipOffset float64
	across                []float64
}

// solarArray is everything the maker needs, in metres, in the craft's
// numbers. A rolled wing has bars at its ends and booms along its
// edges where a rigid one has a frame.
type solarArray struct {
	frame  maths.Subspace
	panels solarPanels
	cells  solarCells
	yoke   solarYoke
	looks  solarLooks
	rolled bool
}

// solarFrame is the frame an array is laid in: its root on the bus, its
// run out along a direction, and the cells on the face that direction
// and the normal give. The run and the face are at right angles.
func solarFrame(root, run, face maths.Vec) maths.Subspace {
	return maths.NewSubspace(root, maths.Orientation{
		X: run, Y: face, Z: run.Cross(face)}, spaceOne)
}

// solarCellsFilling is a grid of columns by rows that fills a panel to
// an inset from its edge, each cell its pitch less the gap.
func solarCellsFilling(panels solarPanels, inset float64,
	columns, rows int, gap float64,
) solarCells {
	pitchX := (panels.width - 2*inset + gap) / float64(columns)
	pitchZ := (panels.depth - 2*inset + gap) / float64(rows)
	return solarCells{columns: columns, rows: rows,
		width: pitchX - gap, depth: pitchZ - gap,
		pitchX: pitchX, pitchZ: pitchZ}
}

// wanted is the width of a rail the panel would like.
func (panels solarPanels) wanted() float64 {
	return panels.thickness * solarRailThickness
}

// total is how far the panels reach across the run, gaps included.
func (panels solarPanels) total() float64 {
	return float64(panels.across)*panels.depth +
		float64(panels.across-spaceOne)*panels.gap
}

// middle is where the panel at an index across the run is centred.
func (panels solarPanels) middle(index int) float64 {
	centre := float64(panels.across-spaceOne) * spaceHalf
	return (float64(index) - centre) * (panels.depth + panels.gap)
}

// valid says the panels have a count and a size.
func (panels solarPanels) valid() bool {
	return panels.along > 0 && panels.across > 0 &&
		positive(panels.width) && positive(panels.depth) &&
		positive(panels.thickness) && panels.gap >= spaceZero
}

// valid says the cells have a count and a size.
func (cells solarCells) valid() bool {
	return cells.columns > 0 && cells.rows > 0 &&
		positive(cells.width) && positive(cells.depth) &&
		positive(cells.height) && positive(cells.pitchX) &&
		positive(cells.pitchZ)
}

// margins are the bare strips between the cells and the panel's edge,
// along the run and across it.
func (array solarArray) margins() (along, across float64) {
	panels, cells := array.panels, array.cells
	along = (panels.width - float64(cells.columns-spaceOne)*cells.pitchX -
		cells.width) * spaceHalf
	across = (panels.depth - float64(cells.rows-spaceOne)*cells.pitchZ -
		cells.depth) * spaceHalf
	return along, across
}

// rail is the width of a frame rail: the width wanted, kept inside the
// margin so that no rail reaches a cell.
func (array solarArray) rail() float64 {
	along, across := array.margins()
	return math.Min(array.panels.wanted(),
		solarRailShare*math.Min(along, across))
}

// overhang is how far the frame stands out past the far end of the last
// panel: the frame stands proud of the substrate's edge by a share of the
// rail's width.
func (array solarArray) overhang() float64 {
	return array.rail() * solarRailProud
}

// pulledIn is the array with its panels moved toward the root by the
// overhang. A craft whose published span runs to the outer face of the
// frame, and not to the end of the substrate, lays this array, so that
// nothing of it stands past the span.
func (array solarArray) pulledIn() solarArray {
	array.panels.reach -= array.overhang()
	return array
}

// fits says the panels and cells are valid and the grid is on the panel.
func (array solarArray) fits() bool {
	along, across := array.margins()
	return array.panels.valid() && array.cells.valid() &&
		along >= spaceZero && across >= spaceZero
}

// lay builds the array into the assembly. An array that does not fit
// leaves the assembly's first error.
func (array solarArray) lay(assembly *instrumentAssembly) {
	if assembly.trouble != nil {
		return
	}
	if !array.fits() {
		assembly.trouble = errSolarArray
		return
	}
	for along := range array.panels.along {
		pose := array.hinge(along)
		for across := range array.panels.across {
			array.panel(assembly, pose, array.panels.middle(across))
		}
		array.hingeLine(assembly, pose)
	}
	array.yokeRun(assembly)
}

// hinge is the frame of the panels at an index along the run: its origin
// on their hinge line at the mid-plane, its x along the panel, and each
// turned by the fold past the one before.
func (array solarArray) hinge(index int) maths.Subspace {
	panels := array.panels
	pose := maths.NewSubspace(maths.V(panels.reach, spaceZero, spaceZero),
		maths.Identity, spaceOne).Within(array.frame)
	step := maths.NewSubspace(
		maths.V(panels.width+panels.gap, spaceZero, spaceZero),
		maths.About(maths.ZAxis, panels.fold), spaceOne)
	for range index {
		pose = step.Within(pose)
	}
	return pose
}

// shareSolar stores the cells of the given matters once each, as
// shareRepeats does for a matter, however many of them are placed. The
// matters must hold flat boxes and nothing else.
func (assembly *instrumentAssembly) shareSolar(cells ...slot) error {
	return assembly.shareWhere(func(piece part) bool {
		return slices.Contains(cells, piece.of)
	})
}
