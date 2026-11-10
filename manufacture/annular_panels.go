package manufacture

import (
	"cmp"
	"math"
	"slices"

	"github.com/janearc/slayer/maths"
)

// axialPanel is one hexagonal-lattice centre in a single sixty-degree sector.
type axialPanel struct{ column, row int }

// norm orders radial shells without floating-point comparisons.
func (cell axialPanel) norm() int {
	return cell.column*cell.column + cell.column*cell.row +
		cell.row*cell.row
}

// rotate preserves the integer lattice through one sixth of a full turn.
func (cell axialPanel) rotate() axialPanel {
	return axialPanel{-cell.row, cell.column + cell.row}
}

// centre maps an axial cell into the primary's XZ plane.
func (cell axialPanel) centre() maths.Vec {
	x := triangleCorners * spaceHalf * float64(cell.column)
	step := float64(cell.row) + float64(cell.column)*spaceHalf
	z := math.Sqrt(triangleCorners) * step
	return maths.V(x, spaceZero, z)
}

// compareAxialPanels fixes tie order so the authored layout is reproducible.
func compareAxialPanels(left, right axialPanel) int {
	if order := cmp.Compare(left.norm(), right.norm()); order != spaceZero {
		return order
	}
	if order := cmp.Compare(left.column, right.column); order != spaceZero {
		return order
	}
	return cmp.Compare(left.row, right.row)
}

// annularSector picks the nearest complete rotation orbits outside a hole.
// The positions are authored approximations, not an ESO segment survey.
func annularSector(count, hole int) []axialPanel {
	if count <= spaceZero || hole < spaceZero {
		return nil
	}
	root := math.Sqrt(float64(count))
	side := int(math.Ceil(root)) + hole
	var cells []axialPanel
	for column := range side {
		for row := range side {
			cell := axialPanel{column + spaceOne, row}
			if cell.norm() >= hole*hole {
				cells = append(cells, cell)
			}
		}
	}
	slices.SortFunc(cells, compareAxialPanels)
	if len(cells) < count {
		return nil
	}
	return cells[:count]
}

// annularPanels retains all six members of each selected rotation orbit.
func annularPanels(count, hole int) []maths.Vec {
	if count < hexagonCorners || count%hexagonCorners != spaceZero {
		return nil
	}
	sector := annularSector(count/hexagonCorners, hole)
	var centres []maths.Vec
	for _, original := range sector {
		cell := original
		for range hexagonCorners {
			centres = append(centres, cell.centre())
			cell = cell.rotate()
		}
	}
	return centres
}
