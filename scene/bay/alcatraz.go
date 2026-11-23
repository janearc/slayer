// Package bay builds the distant landmarks of the Aquatic Park shot.
package bay

import (
	"bytes"
	_ "embed"
	"fmt"
	"math"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
	"github.com/janearc/slayer/scene/thing"
	"github.com/janearc/slayer/terrain"
)

//go:embed alcatraz-3dep.tif
var alcatraz3DEP []byte

// embeddedAlcatraz builds the island from the pinned interior USGS crop.
func embeddedAlcatraz(grid Grid) ([]thing.Thing, error) {
	raster, err := terrain.OpenReader(bytes.NewReader(alcatraz3DEP), int64(len(alcatraz3DEP)))
	if err != nil {
		return nil, fmt.Errorf("embedded Alcatraz 3DEP: %w", err)
	}
	defer raster.Close()
	return alcatrazRaster(raster, grid)
}

// Grid locates the local origin in NAD83 / UTM zone 10N, EPSG:26910.
// X increases east, Y increases up, and Z increases south. SeaElevation
// is an authored source-datum elevation for the scene's sea plane at Y=0.
type Grid struct {
	East, North, SeaElevation float64
}

// AquaticPark returns a swimmer position near the cove mouth, just east
// of the documented Municipal Pier concession structure. The sea plane
// initially uses DEM zero. Neither the camera nor sea level is surveyed.
func AquaticPark() Grid {
	return Grid{East: 550682.05, North: 4184945.93}
}

const lighthouseEast, lighthouseNorth = 550847.18, 4186679.06

// LighthouseBearing is the grid bearing in radians clockwise from north.
func (grid Grid) LighthouseBearing() float64 {
	return math.Atan2(lighthouseEast-grid.East, lighthouseNorth-grid.North)
}

// at places a projected coordinate and source height in the local scene.
func (grid Grid) at(east, north, elevation float64) maths.Vec {
	return maths.V(east-grid.East, elevation-grid.SeaElevation, grid.North-north)
}

// Alcatraz builds the land and distant architectural masses from a 3DEP
// raster. The caller owns lights, ocean, haze, and swimmer.
func Alcatraz(path string, grid Grid) ([]thing.Thing, error) {
	raster, err := terrain.Open(path)
	if err != nil {
		return nil, err
	}
	defer raster.Close()
	return alcatrazRaster(raster, grid)
}

// alcatrazRaster is the island drawn from a raster of ground heights: its
// land, then the buildings and landmarks on it. It refuses a raster that is
// not projected in UTM zone 10 north, in metres, and a grid that is not
// valid.
func alcatrazRaster(raster *terrain.Raster, grid Grid) ([]thing.Thing, error) {
	if !grid.valid() {
		return nil, fmt.Errorf("alcatraz: finite local origin and sea elevation required")
	}
	ref := raster.Reference()
	if ref.Model != 1 || ref.ProjectedCRS != 26910 || ref.LinearUnits != 9001 {
		return nil, fmt.Errorf("alcatraz: expected EPSG:26910 metre raster")
	}
	land, err := islandLand(raster, grid)
	if err != nil {
		return nil, err
	}
	architecture, err := landmarks(raster, grid)
	if err != nil {
		return nil, err
	}
	return append([]thing.Thing{{Shape: land, Material: sandstone}}, architecture...), nil
}

// valid is whether the grid's east, north and sea elevation are all
// finite numbers.
func (grid Grid) valid() bool {
	for _, value := range [3]float64{grid.East, grid.North, grid.SeaElevation} {
		if math.IsNaN(value) || math.IsInf(value, 0) {
			return false
		}
	}
	return true
}

var sandstone = &thing.Material{
	Name: "Alcatraz sandstone and low scrub", Colour: maths.RGB(.30, .28, .21),
}

// islandLand is the island's dry ground as a mesh over the raster's cells.
// It refuses a crop of fewer than 2 or more than 512 cells a side.
func islandLand(raster *terrain.Raster, grid Grid) (*mesh.Mesh, error) {
	width, height := raster.Size()
	if width < 2 || height < 2 || width > 512 || height > 512 {
		return nil, fmt.Errorf("alcatraz: raster crop must be 2–512 cells per side")
	}
	points, valid, err := islandSamples(raster, grid, width, height)
	if err != nil {
		return nil, err
	}
	faces := islandFaces(points, valid, width, height)
	if len(faces) == 0 {
		return nil, fmt.Errorf("alcatraz: no land above the selected sea elevation")
	}
	return mesh.New(points, faces, nil, false)
}

// islandSamples is every cell of the raster as a point in local metres,
// with whether the cell holds a height.
func islandSamples(raster *terrain.Raster, grid Grid, width, height int) ([]maths.Vec, []bool, error) {
	points := make([]maths.Vec, width*height)
	valid := make([]bool, len(points))
	for row := range height {
		for col := range width {
			east, north, err := raster.SampleLocation(col, row)
			if err != nil {
				return nil, nil, err
			}
			elevation, present, err := raster.Elevation(east, north)
			if err != nil {
				return nil, nil, err
			}
			index := row*width + col
			points[index] = grid.at(east, north, elevation)
			valid[index] = present
		}
	}
	return points, valid, nil
}

// islandFaces is two triangles to each square of four neighbouring cells,
// kept where they are dry.
func islandFaces(points []maths.Vec, valid []bool, width, height int) [][3]int {
	var faces [][3]int
	for row := range height - 1 {
		for col := range width - 1 {
			nw := row*width + col
			ne, sw := nw+1, nw+width
			se := sw + 1
			for _, face := range [...][3]int{{nw, sw, ne}, {sw, se, ne}} {
				if dryFace(face, points, valid) {
					faces = append(faces, face)
				}
			}
		}
	}
	return faces
}

// dryFace is whether all three corners of a face hold a height and the
// highest of them is above a tenth of a metre.
func dryFace(face [3]int, points []maths.Vec, valid []bool) bool {
	a, b, c := face[0], face[1], face[2]
	return valid[a] && valid[b] && valid[c] &&
		math.Max(points[a].Y, math.Max(points[b].Y, points[c].Y)) > 0.1
}
