package session

// A terrain that holds a near window is two grounds from one model: the
// coarse window the set's places are metres in, and a finer one about a
// place of its own, which is stood where it lies in the first. The coarse
// ground is not drawn where the fine one is. This file says how the fine
// window is cut, where it lies, and how heights are read from it.

import (
	"fmt"
	"math"
	"strings"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
	"github.com/janearc/slayer/set"
	"github.com/janearc/slayer/terrain"
)

// halfOf is the share of a window's width or depth that is its middle.
// It is defined: half.
const halfOf = 0.5

// nearGround is the finer window of a terrain, and where its first point
// is in the scene: metres east and south of the coarse window's first.
type nearGround struct {
	window      terrain.Window
	east, south float64
}

// span is how far the window reaches east and south from its first point,
// in metres: the points are a step apart, so one fewer than their count
// of steps.
func (near nearGround) span() (across, down float64) {
	columns, rows, step := near.window.Size()
	return float64(columns-1) * step, float64(rows-1) * step
}

// hole is the part of the coarse ground the near window covers.
func (near nearGround) hole() *terrain.Hole {
	across, down := near.span()
	return &terrain.Hole{East: near.east, South: near.south,
		Across: across, Down: down}
}

// holds is whether the near window has a height at a place of the scene,
// east x and south z of the coarse window's first point.
func (near nearGround) holds(x, z float64) bool {
	across, down := near.span()
	return x >= near.east && x <= near.east+across &&
		z >= near.south && z <= near.south+down
}

// floor is the near window's heights as a floor to read them from, the
// scene's lowest height being the one it stands on.
func (near nearGround) floor(lowest float64) terrainFloor {
	return terrainFloor{window: near.window, lowest: lowest}
}

// height is the ground at a place of the scene the near window holds, on
// the diagonals of its mesh.
func (near nearGround) height(x, z, lowest float64) float64 {
	return near.floor(lowest).height(x-near.east, z-near.south)
}

// offsetFrom is where the near window's first point is, east and south of
// the first point of the coarse window, for the two places by lat and lon
// and their sizes in metres. A window's first point is at its north west
// corner, half its width west and half its depth north of its middle; and
// the metres between two middles are the flat patch's, as laidOn has them.
func offsetFrom(far, near groundPlace, radius float64) (east, south float64) {
	middleEast := (near.longitude - far.longitude) * maths.Degree * radius *
		math.Cos(far.latitude*maths.Degree)
	middleNorth := (near.latitude - far.latitude) * maths.Degree * radius
	east = middleEast - near.across*halfOf + far.across*halfOf
	south = far.down*halfOf - middleNorth - near.down*halfOf
	return east, south
}

// lies refuses a near window that is not wholly inside the coarse window
// it is laid in, which has no ground to be drawn there instead of.
func (near nearGround) lies(far terrain.Window) error {
	columns, rows, step := far.Size()
	across, down := float64(columns-1)*step, float64(rows-1)*step
	reach, depth := near.span()
	if near.east < 0 || near.south < 0 || near.east+reach > across ||
		near.south+depth > down {
		return fmt.Errorf("the near window, %.0f m east and %.0f m south "+
			"of the terrain's first point and %.0f by %.0f m, is not "+
			"inside the terrain, %.0f by %.0f m", near.east, near.south,
			reach, depth, across, down)
	}
	return nil
}

// cutNear cuts the near window of a terrain from the model, as the coarse
// window is, and keeps it as stage.near, where the coarse window's place
// is the one it is laid in.
func (stage *staging) cutNear(ground, finer set.Terrain,
	asset *pinnedAsset) error {
	latitude, longitude, _ := ground.Place()
	nearLatitude, nearLongitude, _ := finer.Place()
	_, _, across, down, step := finer.Window()
	cut := groundCut{west: 0, north: down, step: step,
		smooth: ground.Smooth(), across: int(across/step) + 1,
		down: int(down/step) + 1}
	cut.place = &groundPlace{latitude: nearLatitude,
		longitude: nearLongitude, across: across, down: down,
		source: ground.From()}
	found, note, err := keptPlace(asset.path, asset.sum, cut)
	stage.given.cached = append(stage.given.cached, note)
	if err != nil {
		return err
	}
	_, _, farAcross, farDown, _ := ground.Window()
	far := groundPlace{latitude: latitude, longitude: longitude,
		across: farAcross, down: farDown}
	east, south := offsetFrom(far, *cut.place, terrain.EarthRadius)
	stage.near = &nearGround{window: found.window, east: east, south: south}
	stage.made.ground += "; near " + strings.TrimPrefix(
		found.sentence(ground.From()), "ground: ")
	return stage.near.lies(stage.ground)
}

// groundLowest is the lowest height of the scene's ground, near window
// and all.
func (stage *staging) groundLowest() float64 {
	lowest := stage.ground.Lowest()
	if stage.near != nil {
		lowest = min(lowest, stage.near.window.Lowest())
	}
	return lowest
}

// addGround adds the ground to the world: the coarse window, with a hole
// where a near window lies, and the near window in it. The scene's lowest
// height is the one they both stand on. It gives the ground's bounds.
func (stage *staging) addGround(look *thing.Material) (maths.Box, error) {
	var hole *terrain.Hole
	if stage.near != nil {
		hole = stage.near.hole()
	}
	surface, err := stage.groundSurface(hole)
	if err != nil {
		return maths.Box{}, err
	}
	stage.world.Add(thing.Thing{Shape: surface, Material: look})
	bounds := surface.Bounds()
	if stage.near == nil {
		return bounds, nil
	}
	finer, err := stage.near.window.GroundOn(stage.near.east,
		stage.near.south, stage.lowest, nil)
	if err != nil {
		return maths.Box{}, err
	}
	stage.world.Add(thing.Thing{Shape: finer, Material: look})
	return union(bounds, finer.Bounds()), nil
}

// union is the smallest box that holds both boxes.
func union(one, other maths.Box) maths.Box {
	return maths.Box{
		Min: maths.V(math.Min(one.Min.X, other.Min.X),
			math.Min(one.Min.Y, other.Min.Y),
			math.Min(one.Min.Z, other.Min.Z)),
		Max: maths.V(math.Max(one.Max.X, other.Max.X),
			math.Max(one.Max.Y, other.Max.Y),
			math.Max(one.Max.Z, other.Max.Z))}
}

// waterTop is the place of the scene the still water's curve is highest
// at: the coarse window's corner, where it always was, or its middle where
// the terrain holds a near window, so that a window of many kilometres
// has its far shore no further from the water's top than its corners are.
func (stage *staging) waterTop() (x, z float64) {
	if stage.near == nil {
		return 0, 0
	}
	columns, rows, step := stage.ground.Size()
	return float64(columns-1) * step * halfOf,
		float64(rows-1) * step * halfOf
}
