package terrain

import "testing"

// the window the hole tests lay ground on: five points on a side, ten
// metres apart, so four cells on a side and thirty-two faces. picked: the
// least that has a cell wholly inside a hole and one across its edge.
const (
	holeSide  = 5
	holeStep  = 10.0
	holeFaces = 2 * (holeSide - 1) * (holeSide - 1)
)

// holeWindow is a flat window of holeSide points a side.
func holeWindow(t *testing.T) Window {
	t.Helper()
	window, err := BuildWindow(0, 0, holeSide, holeSide, holeStep,
		func(column, row int) (float64, error) {
			return float64(column + row), nil
		})
	if err != nil {
		t.Fatal(err)
	}
	return window
}

// A hole leaves out the cells whose four corners are all in it, two faces
// each, and no other.
func TestGroundOnLeavesOutTheCellsInTheHole(t *testing.T) {
	window := holeWindow(t)
	hole := Hole{East: 10, South: 10, Across: 20, Down: 20}
	made, err := window.GroundOn(0, 0, window.Lowest(), &hole)
	if err != nil {
		t.Fatal(err)
	}
	const cellsInHole = 4 // the 2 by 2 cells from 10 m to 30 m
	if want := holeFaces - 2*cellsInHole; made.FaceCount() != want {
		t.Errorf("%d faces, want %d", made.FaceCount(), want)
	}
}

// Anti: a hole that cuts across a cell leaves that cell in, and a nil hole
// or a hole off the window leaves out nothing.
func TestGroundOnKeepsTheCellsTheHoleOnlyCrosses(t *testing.T) {
	window := holeWindow(t)
	across := Hole{East: 15, South: 15, Across: 10, Down: 10}
	off := Hole{East: 500, South: 500, Across: 10, Down: 10}
	for name, hole := range map[string]*Hole{"across": &across,
		"off": &off, "nil": nil} {
		made, err := window.GroundOn(0, 0, window.Lowest(), hole)
		if err != nil {
			t.Fatal(name, err)
		}
		if made.FaceCount() != holeFaces {
			t.Errorf("%s: %d faces, want %d", name, made.FaceCount(),
				holeFaces)
		}
	}
}

// Ground is GroundOn at the origin, from the window's own lowest height,
// with no hole.
func TestGroundIsGroundOnAtTheOrigin(t *testing.T) {
	window := holeWindow(t)
	made, lowest, err := window.Ground()
	if err != nil {
		t.Fatal(err)
	}
	if lowest != 0 || made.FaceCount() != holeFaces {
		t.Errorf("lowest %v, %d faces", lowest, made.FaceCount())
	}
}

// A window stood east and south of the origin has its corners there, and
// its heights from the floor given: the mesh's bounds say so.
func TestGroundOnStandsWhereTheSceneSays(t *testing.T) {
	window := holeWindow(t)
	made, err := window.GroundOn(100, 200, -5, nil)
	if err != nil {
		t.Fatal(err)
	}
	box := made.Bounds()
	const farCorner = holeStep * (holeSide - 1) // the window's span, metres
	if box.Min.X != 100 || box.Max.X != 100+farCorner || box.Min.Z != 200 ||
		box.Max.Z != 200+farCorner || box.Min.Y != 5 {
		t.Errorf("bounds %+v", box)
	}
}
