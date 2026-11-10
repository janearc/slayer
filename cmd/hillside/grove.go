package main

import (
	"math"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/medium"
	"github.com/janearc/slayer/scene/thing"
	"github.com/janearc/slayer/set"
	"github.com/janearc/slayer/terrain"
)

type groveTree struct {
	at     maths.Vec
	bounds maths.Box
}

// grove shares one field across a stand, including all its repeated meshes.
// Optical values are authored in YAML; no tree count becomes a fog rate.
func (stage *staging) grove(stand set.Stand, trees []groveTree) error {
	name, fade := stand.Atmosphere()
	return stage.populationMist(name, fade, trees)
}

// populationMist shares the same prescribed field for stands and mixed regions.
// Population size never becomes a water flux or condensation rate.
func (stage *staging) populationMist(name string, fade bool, trees []groveTree) error {
	if name == "" || len(trees) == 0 {
		return nil
	}
	mist, _ := stage.recipes.Mist(name)
	canopies := make([]medium.Canopy, len(trees))
	for i, tree := range trees {
		canopies[i] = medium.Canopy{At: tree.at,
			Radius: horizontalRadius(tree.bounds), Height: tree.bounds.Max.Y}
	}
	from, speed := stage.recipes.Wind()
	// Wind is named for where it comes from; north is negative z.
	wind := skyward(from, 0).Scale(-speed)
	floor := terrainFloor{window: stage.ground, lowest: stage.lowest}
	field, err := medium.NewGrove(medium.GroveWeather{
		Extinction: mist.Extinction(), Depth: mist.Depth(), Coverage: mist.Coverage(),
		Seed: mist.Seed(), Wind: wind, Fade: fade}, canopies, floor.height)
	if err != nil {
		return err
	}
	stage.world.AddVolume(thing.Volume{Medium: field,
		Material: &thing.Material{Name: name, Colour: maths.Grey(0.95)}})
	return nil
}

// groundAt interpolates on the same diagonals as the terrain mesh.
func (stage *staging) groundAt(x, z float64) float64 {
	floor := terrainFloor{window: stage.ground, lowest: stage.lowest}
	return floor.height(x, z)
}

// terrainFloor retains heights and their scene offset without retaining the
// staging builder, its prepared geometry or its placement caches.
type terrainFloor struct {
	window terrain.Window
	lowest float64
}

// nextGridPoint is defined as the adjacent height sample.
// firstGridPoint is defined as the first Go slice index.
// gridCellPoints is defined by the two points along each side of a cell.
const nextGridPoint, firstGridPoint, gridCellPoints = 1, 0, 2

// height interpolates on the same diagonals as the terrain mesh.
func (floor terrainFloor) height(x, z float64) float64 {
	across, down, step := floor.window.Size()
	lastColumn, lastRow := float64(across-nextGridPoint),
		float64(down-nextGridPoint)
	x = math.Max(firstGridPoint,
		math.Min(lastColumn, x/step))
	z = math.Max(firstGridPoint,
		math.Min(lastRow, z/step))
	column, row := min(across-gridCellPoints, int(x)),
		min(down-gridCellPoints, int(z))
	acrossShare, downShare := x-float64(column), z-float64(row)
	northWest := floor.window.At(column, row)
	northEast := floor.window.At(column+nextGridPoint, row)
	southWest := floor.window.At(column, row+nextGridPoint)
	southEast := floor.window.At(column+nextGridPoint, row+nextGridPoint)
	if acrossShare >= downShare {
		return northWest + acrossShare*(northEast-northWest) +
			downShare*(southEast-northEast) - floor.lowest
	}
	return northWest + downShare*(southWest-northWest) +
		acrossShare*(southEast-southWest) - floor.lowest
}
