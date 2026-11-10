package main

import (
	"fmt"
	"math"
	"time"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/medium"
	"github.com/janearc/slayer/scene/thing"
)

// floorGrid is an immutable local terrain approximation for concurrent density
// queries. It retains no open raster, scene builder or procedural geometry.
type floorGrid struct {
	bounds  localBounds
	side    int
	heights []float64
}

// mist places each prescribed layer in the site's fixed local frame.
func (site *siteBuilder) mist() error {
	for _, mist := range site.plan.Atmosphere {
		if err := site.mistLayer(mist); err != nil {
			return err
		}
	}
	return nil
}

// mistLayer samples a source-grounded floor before constructing its weather deck.
func (site *siteBuilder) mistLayer(mist mistPlan) error {
	floor := floorGrid{bounds: mist.Bounds, side: mist.FloorSide, heights: make([]float64, mist.FloorSide*mist.FloorSide)}
	for row := 0; row < floor.side; row++ {
		for column := 0; column < floor.side; column++ {
			x := floor.bounds.MinEast + float64(column)/float64(floor.side-1)*(floor.bounds.MaxEast-floor.bounds.MinEast)
			z := floor.bounds.MinSouth + float64(row)/float64(floor.side-1)*(floor.bounds.MaxSouth-floor.bounds.MinSouth)
			height, err := site.floorHeight(x, z, mist.FloorTolerance, mist.FloorIterations)
			if err != nil {
				return err
			}
			floor.heights[row*floor.side+column] = height
		}
	}
	wind := maths.V(mist.Wind[0], mist.Wind[1], mist.Wind[2])
	weather := medium.NewWeather(mist.Extinction, mist.Depth, mist.Coverage, mist.Seed, wind)
	box := maths.Box{Min: maths.V(mist.Bounds.MinEast, mist.Base, mist.Bounds.MinSouth),
		Max: maths.V(mist.Bounds.MaxEast, mist.Base+mist.Depth, mist.Bounds.MaxSouth)}
	field, err := medium.NewDeck(weather, box, floor.height, mist.Base)
	if err != nil {
		return err
	}
	pose, err := site.relative.Placement(site.space)
	if err != nil {
		return err
	}
	var density medium.Medium = field
	if mist.EdgeFade > 0 {
		density = taperedMist{field: field, fade: mist.EdgeFade, finest: field.Finest()}
	}
	site.world.AddVolume(thing.Volume{Medium: density, Material: &thing.Material{Name: "authored regional mist", Colour: rgb(mist.Colour)}, Motion: thing.Still(pose)})
	return nil
}

// taperedMist gives an authored bank elliptical horizontal support and a
// smooth outer transition. Zero fade leaves the original rectangular deck.
// The weather's vertical profile, source-floor clipping and clock are shared.
type taperedMist struct {
	field        medium.Medium
	fade, finest float64
}

// Bounds retains the deck's enclosing box while density supplies the taper.
func (mist taperedMist) Bounds() maths.Box { return mist.field.Bounds() }

// Finest preserves the underlying deck's sampling step.
func (mist taperedMist) Finest() float64 {
	return mist.finest
}

// Tick advances the underlying weather while retaining the horizontal taper.
func (mist taperedMist) Tick(at time.Duration) medium.Medium {
	if moving, ok := mist.field.(medium.Timed); ok {
		mist.field = moving.Tick(at)
	}
	return mist
}

// Density fades the deck to zero at its authored elliptical boundary.
func (mist taperedMist) Density(at maths.Vec) float64 {
	box := mist.Bounds()
	xRadius, zRadius := (box.Max.X-box.Min.X)/2, (box.Max.Z-box.Min.Z)/2
	x := (at.X - (box.Min.X+box.Max.X)/2) / xRadius
	z := (at.Z - (box.Min.Z+box.Max.Z)/2) / zRadius
	distance := math.Hypot(x, z)
	if distance >= 1 {
		return 0
	}
	inner := 1 - mist.fade/math.Min(xRadius, zRadius)
	return mist.field.Density(at) * (1 - maths.Smooth(inner, 1, distance))
}

// floorHeight solves a source surface along one fixed site-up line. Each
// source lookup uses the candidate's actual spherical geography. Exhaustion
// is an error, never an invented height; the caller declares its tolerance.
func (site *siteBuilder) floorHeight(x, z, tolerance float64, iterations int) (float64, error) {
	y := 0.0
	for step := 0; step < iterations; step++ {
		if err := site.ctx.Err(); err != nil {
			return 0, err
		}
		location, err := site.frame.Location(site.space.Out(maths.V(x, y, z)))
		if err != nil {
			return 0, err
		}
		height, has, err := site.heights.Height(location.Longitude.InDegrees(), location.Latitude.InDegrees())
		if err != nil {
			return 0, err
		}
		if !has {
			return 0, fmt.Errorf("island: missing source below mist floor")
		}
		location.Height = measure.Metres(height)
		world, err := site.frame.Position(location)
		if err != nil {
			return 0, err
		}
		local := site.space.In(world)
		if math.Abs(local.Y-y) <= tolerance && math.Hypot(local.X-x, local.Z-z) <= tolerance {
			return local.Y, nil
		}
		y = local.Y
	}
	return 0, fmt.Errorf("island: mist floor projection did not converge")
}

// height interpolates the retained floor grid and returns NaN outside it.
func (floor floorGrid) height(x, z float64) float64 {
	if x < floor.bounds.MinEast || x > floor.bounds.MaxEast || z < floor.bounds.MinSouth || z > floor.bounds.MaxSouth {
		return math.NaN()
	}
	u := (x - floor.bounds.MinEast) / (floor.bounds.MaxEast - floor.bounds.MinEast) * float64(floor.side-1)
	v := (z - floor.bounds.MinSouth) / (floor.bounds.MaxSouth - floor.bounds.MinSouth) * float64(floor.side-1)
	column, row := min(int(u), floor.side-2), min(int(v), floor.side-2)
	u, v = u-float64(column), v-float64(row)
	a, b := floor.heights[row*floor.side+column], floor.heights[row*floor.side+column+1]
	c, d := floor.heights[(row+1)*floor.side+column], floor.heights[(row+1)*floor.side+column+1]
	return (1-v)*((1-u)*a+u*b) + v*((1-u)*c+u*d)
}
