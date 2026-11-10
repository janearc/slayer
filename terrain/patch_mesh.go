package terrain

import (
	"fmt"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
)

// MeshBudget bounds output vertex/triangle counts before mesh construction.
// It excludes the constructor's copies, edge checks and acceleration storage.
type MeshBudget struct{ Points, Faces int }

// Mesh maps available source samples into a caller's fixed world coordinates.
// The mapper owns projection and height-datum conversion; none is inferred.
// Each triangle needs three available corners. No available triangle is
// absence.
func (grid PatchGrid) Mesh(budget MeshBudget,
	mapPoint func(east, north, height float64) (maths.Vec, error),
) (*mesh.Mesh, bool, error) {
	across, down := grid.Size()
	if mapPoint == nil || across < 2 || down < 2 || budget.Points < 1 || budget.Faces < 1 ||
		across > budget.Points/down || across-1 > (budget.Faces/2)/(down-1) {
		return nil, false, fmt.Errorf("terrain: patch mesh exceeds its output count budget")
	}
	points := make([]maths.Vec, across*down)
	for row := range down {
		for column := range across {
			east, north, height, available := grid.At(column, row)
			if !available {
				continue
			}
			point, err := mapPoint(east, north, height)
			if err != nil {
				return nil, false, err
			}
			if !finiteCoordinate(point.X) || !finiteCoordinate(point.Y) || !finiteCoordinate(point.Z) {
				return nil, false, fmt.Errorf("terrain: mapped patch point is not finite")
			}
			points[row*across+column] = point
		}
	}
	faces := grid.meshFaces(across, down)
	if len(faces) == 0 {
		return nil, false, nil
	}
	made, err := mesh.New(points, faces, nil, false)
	if err != nil {
		return nil, false, err
	}
	return made, true, nil
}

// meshFaces retains the legacy north-west/south-east terrain diagonal.
// Missing corners suppress only triangles which actually reference them.
func (grid PatchGrid) meshFaces(across, down int) [][3]int {
	faces := make([][3]int, 0, 2*(across-1)*(down-1))
	for row := range down - 1 {
		for column := range across - 1 {
			near := row*across + column
			far := near + across
			for _, face := range [2][3]int{{near, far + 1, near + 1}, {near, far, far + 1}} {
				if grid.present[face[0]] && grid.present[face[1]] && grid.present[face[2]] {
					faces = append(faces, face)
				}
			}
		}
	}
	return faces
}
