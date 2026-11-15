package named

import (
	"fmt"
	"io"
	"math"
	"time"

	"github.com/janearc/slayer/biome"
	"github.com/janearc/slayer/lantern"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/hexes"
	"github.com/janearc/slayer/scene/medium"
	"github.com/janearc/slayer/scene/texture"
	"github.com/janearc/slayer/scene/thing"
)

// the pair, her design of 2026-10-04: the inner band's radius as a share
// of the outer's, and how far each leans from the lantern's axis, in
// degrees, on either side
const (
	innerShare = 0.8
	hoopLean   = 45
)

// a band's floor: how its biome is made, how its tiles are lined, and its
// hull's colour, gloss and index
const (
	floorScale, floorSea, floorSprawl = 45, 0.02, 4
	floorLines                        = 0.12
	hullRed, hullGreen, hullBlue      = 0.12, 0.12, 0.14
	hullGloss, hullIndex              = 0.3, 4
)

// hoop is one band of the pair: how far out it runs, how far it leans
// from the lantern's axis, and which way round it turns
type hoop struct {
	radius, lean, sense float64
}

// nested is the pair, one inside the other: the outer at the radius for
// one g in a day, the inner four fifths of it, each leaning 45 degrees
// from the lantern's axis on either side, so the two stand at right
// angles and the lantern's axis splits the angle between them; they turn
// opposite ways.
//
// her design, 2026-10-04.
func nested() []hoop {
	return []hoop{
		{radius: orbitalRadius, lean: hoopLean * maths.Degree, sense: 1},
		{radius: innerShare * orbitalRadius,
			lean: -hoopLean * maths.Degree, sense: -1},
	}
}

// zappyDefined is zappy, the beamed lantern at the far view's middle, as
// package lantern defines it. It is read once and never changed; a view
// that changes it changes its own copy.
var zappyDefined = lantern.Zappy()

// lanternAxis is the axis zappy's beams spin about, on average, since the
// spin axis precesses round it; the hoops are set from it
func lanternAxis() maths.Vec {
	return zappyDefined.Axis.Unit()
}

// space is where the band sits before any turn: its axis is the
// lantern's, tipped by its lean toward the eye's side
func (one hoop) space() maths.Subspace {
	axis := lanternAxis()
	toEye := maths.V(0, 0, 1)
	across := toEye.Sub(axis.Scale(toEye.Dot(axis))).Unit()
	tip := maths.Subspace{Orientation: maths.Rotation(across, one.lean)}
	return maths.Subspace{Orientation: maths.Upright(tip.Out(axis))}
}

// turning is the band turning about its own axis: once a day at one g's
// radius, longer or shorter with the square root of its own, and the
// other way round for the inner
func (one hoop) turning(day time.Duration) thing.Motion {
	space := one.space()
	period := float64(day) * math.Sqrt(one.radius/orbitalRadius) *
		one.sense
	return thing.Spin{In: space, Axis: space.Out(maths.V(0, 1, 0)),
		Period: time.Duration(period)}
}

// place puts the band in the world: its floor painted by biome with the
// seed for its number, its gap wall, and its hull, all turning together;
// and says what its floor holds
func (one hoop) place(world *thing.World, number int,
	gapWall *thing.Material, day time.Duration, said io.Writer) error {
	tiles := hexes.Band{Radius: one.radius, Width: bandWidth,
		Around: hexAround}
	painted := biome.Make(biome.Fields{Tiling: tiles,
		Seed: uint64(orbitalSeed + number), Scale: floorScale,
		Sea: floorSea, Sprawl: floorSprawl})
	painted.Lights = urbanLights
	fmt.Fprintf(said, "ring %d: %d rivers, %d lakes, %d volcanoes, "+
		"%d cities\n", number, painted.Rivers, painted.Lakes,
		painted.Volcanoes, painted.Cities)
	floorMat := &thing.Material{Name: "floor",
		Texture: texture.Tiled{Tiling: tiles, Paints: painted,
			Lines: floorLines},
		Shine: texture.Tiled{Tiling: tiles, Paints: painted.Shine()}}
	// the outside is hull: the people live on the inside, so it is dark
	hull := &thing.Material{Name: "hull",
		Colour: maths.RGB(hullRed, hullGreen, hullBlue),
		Gloss:  hullGloss, Index: hullIndex}
	floor, facing, rest, err := hoopMeshes(0, one.radius, number == 0)
	if err != nil {
		return err
	}
	turning := one.turning(day)
	world.Add(thing.Thing{Shape: floor, Material: floorMat,
		Motion: turning})
	world.Add(thing.Thing{Shape: facing, Material: gapWall,
		Motion: turning})
	world.Add(thing.Thing{Shape: rest, Material: hull, Motion: turning})
	return nil
}

// pathsOf is the bands' paths through the gas, each placed once
func pathsOf(pair []hoop) ([]*medium.Path, error) {
	paths := make([]*medium.Path, 0, len(pair))
	for _, one := range pair {
		path, err := medium.NewPath(one.space(), one.radius)
		if err != nil {
			return nil, err
		}
		paths = append(paths, path)
	}
	return paths, nil
}
