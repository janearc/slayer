package manufacture

import (
	"math"

	"github.com/janearc/slayer/maths"
)

// panel lays one panel in a hinge frame, centred at a place across the
// run: the substrate, its frame, its cells with their covers, and the
// bus bars between the columns.
func (array solarArray) panel(assembly *instrumentAssembly,
	pose maths.Subspace, middle float64,
) {
	panels := array.panels
	half := panels.thickness * spaceHalf
	assembly.boxAt(panels.width, panels.thickness, panels.depth,
		maths.V(spaceZero, -half, middle-panels.depth*spaceHalf), pose,
		array.looks.substrate)
	array.edges(assembly, pose, middle)
	array.cellGrid(assembly, pose, middle)
	array.busBars(assembly, pose, middle)
}

// cellGrid lays each cell on the substrate's face, and its cover on it.
func (array solarArray) cellGrid(assembly *instrumentAssembly,
	pose maths.Subspace, middle float64,
) {
	cells := array.cells
	along, across := array.margins()
	floor := array.panels.thickness * spaceHalf
	start := middle - array.panels.depth*spaceHalf + across
	for column := range cells.columns {
		x := along + float64(column)*cells.pitchX
		for line := range cells.rows {
			at := maths.V(x, floor, start+float64(line)*cells.pitchZ)
			assembly.boxAt(cells.width, cells.height, cells.depth, at, pose,
				array.cellMatter(line))
			if cells.cover > spaceZero {
				at.Y += cells.height
				assembly.boxAt(cells.width, cells.cover, cells.depth, at,
					pose, array.looks.cover)
			}
		}
	}
}

// cellMatter is the matter of the cells in a row across the run.
func (array solarArray) cellMatter(line int) slot {
	cells := array.cells
	if cells.secondFrom > 0 && line >= cells.secondFrom {
		return cells.second
	}
	return array.looks.cell
}

// busBars lays a bar in the gap between each two columns of cells, the
// length of the grid and as high as a cell, a share of the gap wide.
func (array solarArray) busBars(assembly *instrumentAssembly,
	pose maths.Subspace, middle float64,
) {
	cells := array.cells
	gap := cells.pitchX - cells.width
	wide := gap * cells.barShare
	if !positive(wide) {
		return
	}
	along, _ := array.margins()
	long := array.panels.depth - 2*array.rail()
	floor := array.panels.thickness * spaceHalf
	for column := spaceOne; column < cells.columns; column++ {
		x := along + float64(column)*cells.pitchX - (gap+wide)*spaceHalf
		assembly.boxAt(wide, cells.height, long,
			maths.V(x, floor, middle-long*spaceHalf), pose, array.looks.bar)
	}
}

// edges is the frame round a rigid panel, and the spreader bars and
// booms of a rolled one.
func (array solarArray) edges(assembly *instrumentAssembly,
	pose maths.Subspace, middle float64,
) {
	rail := array.rail()
	if !positive(rail) {
		return
	}
	array.spreaders(assembly, pose, middle, rail)
	if array.rolled {
		array.booms(assembly, pose, middle, rail)
		return
	}
	array.sideRails(assembly, pose, middle, rail)
}

// standing is how high a frame stands from the substrate's far face: the
// substrate, the cells and their covers.
func (array solarArray) standing() float64 {
	return array.panels.thickness + array.cells.height + array.cells.cover
}

// spreaders are the rails across the run at the two ends of a panel.
// Each stands out past the panel's sides by a rail's width, so that it
// meets a side rail across a face and not along a shared edge: two boxes
// of one matter that share a whole edge cannot be meshed together.
func (array solarArray) spreaders(assembly *instrumentAssembly,
	pose maths.Subspace, middle, rail float64,
) {
	panels := array.panels
	proud := rail * solarRailProud
	back := middle - panels.depth*spaceHalf - rail
	for _, x := range []float64{-proud, panels.width - rail} {
		assembly.boxAt(rail+proud, array.standing()+proud,
			panels.depth+2*rail, maths.V(x,
				-panels.thickness*spaceHalf-proud, back), pose,
			array.looks.frame)
	}
}

// sideRails are the rails along the run at the two sides of a panel,
// between its spreaders.
func (array solarArray) sideRails(assembly *instrumentAssembly,
	pose maths.Subspace, middle, rail float64,
) {
	panels := array.panels
	proud := rail * solarRailProud
	back := middle - panels.depth*spaceHalf
	for _, z := range []float64{back - proud, back + panels.depth - rail} {
		assembly.boxAt(panels.width-2*rail, array.standing()+proud,
			rail+proud, maths.V(rail,
				-panels.thickness*spaceHalf-proud, z), pose,
			array.looks.frame)
	}
}

// booms lays a bi-stem boom along each long edge of a rolled panel: two
// tubes, one above the other, outside the blanket's edge.
func (array solarArray) booms(assembly *instrumentAssembly,
	pose maths.Subspace, middle, rail float64,
) {
	panels := array.panels
	radius := rail * solarBoomShare
	reach := panels.depth*spaceHalf + radius
	for _, side := range []float64{-spaceOne, spaceOne} {
		for _, tier := range []float64{-spaceOne, spaceOne} {
			from := maths.V(spaceZero, tier*radius, middle+side*reach)
			to := maths.V(panels.width, tier*radius, middle+side*reach)
			assembly.rod(pose.Out(from), pose.Out(to), radius,
				array.looks.hardware)
		}
	}
}

// hingeLine lays the barrel of a hinge across the run along the inboard
// edge of the panels at a hinge frame, and a fitting at each end of it.
// The barrel fits the gap between two panels. A fitting stands inside the
// barrel's end, its outer face at the panels' side, so that no part of a
// hinge is wider than the panels it turns.
func (array solarArray) hingeLine(assembly *instrumentAssembly,
	pose maths.Subspace,
) {
	panels := array.panels
	radius := panels.thickness * solarHingeShare
	if panels.gap > spaceZero {
		radius = math.Min(radius, panels.gap*spaceHalf)
	}
	span := panels.total() * spaceHalf
	from := maths.V(-radius, spaceZero, -span)
	to := maths.V(-radius, spaceZero, span)
	assembly.rod(pose.Out(from), pose.Out(to), radius, array.looks.hardware)
	girth := radius * solarFittingGirth
	inside := span - girth*spaceHalf
	for _, side := range []float64{-spaceOne, spaceOne} {
		array.fitting(assembly, pose, maths.V(-radius, spaceZero,
			side*inside), girth)
	}
}

// fitting is a closed block a girth across, centred on a point.
func (array solarArray) fitting(assembly *instrumentAssembly,
	pose maths.Subspace, centre maths.Vec, girth float64,
) {
	half := maths.V(girth, girth, girth).Scale(spaceHalf)
	assembly.boxAt(girth, girth, girth, centre.Sub(half), pose,
		array.looks.hardware)
}

// yokeRun lays the yoke at each of its offsets across the run, with a
// fitting at its root, and the harness beside the last of them.
func (array solarArray) yokeRun(assembly *instrumentAssembly) {
	yoke := array.yoke
	if !positive(yoke.length) || len(yoke.across) == 0 {
		return
	}
	for _, across := range yoke.across {
		root, tip := array.yokeEnds(across)
		assembly.rod(array.frame.Out(root), array.frame.Out(tip),
			yoke.radius, array.looks.hardware)
		array.fitting(assembly, array.frame, root,
			yoke.radius*solarFittingGirth)
	}
	// the harness lies on the yoke's surface, not tangent to it: tubes
	// that touch along a line share their edges, which cannot be meshed
	last := yoke.across[len(yoke.across)-spaceOne]
	radius := yoke.radius * solarHarnessShare
	root, tip := array.yokeEnds(last + yoke.radius)
	assembly.rod(array.frame.Out(root), array.frame.Out(tip), radius,
		array.looks.hardware)
}

// yokeEnds are the root and the tip of a tube along the yoke, at a place
// across the run.
func (array solarArray) yokeEnds(across float64) (root, tip maths.Vec) {
	yoke := array.yoke
	root = maths.V(spaceZero, -yoke.rootOffset, across)
	tip = maths.V(yoke.length, -yoke.tipOffset, across)
	return root, tip
}
