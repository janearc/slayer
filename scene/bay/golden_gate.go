package bay

import (
	"bytes"
	_ "embed"
	"fmt"
	"math"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
	"github.com/janearc/slayer/terrain"
)

//go:embed golden-gate-3dep.tif
var goldenGate3DEP []byte

const (
	gateEast, gateNorth = 545876.42, 4185922.74
	gateHalfMain        = 640.08
	gateSide            = 342.90
	gateTowerTop        = 227.38
)

var gateOrange = &thing.Material{
	Name: "Golden Gate International Orange", Colour: maths.RGB(.62, .125, .051),
}

var gateDeck = &thing.Material{
	Name: "Golden Gate deck", Colour: maths.RGB(.24, .18, .15),
}

var gateShore = &thing.Material{
	Name: "Golden Gate coastal scrub", Colour: maths.RGB(.24, .27, .18),
}

// GoldenGateCentre is the centre of the main span at the selected sea plane.
func GoldenGateCentre(grid Grid) maths.Vec {
	return grid.at(gateEast, gateNorth, grid.SeaElevation)
}

// goldenGateShore builds the mainland and headlands from a pinned USGS crop.
func goldenGateShore(grid Grid) (thing.Thing, error) {
	raster, err := terrain.OpenReader(bytes.NewReader(goldenGate3DEP), int64(len(goldenGate3DEP)))
	if err != nil {
		return thing.Thing{}, fmt.Errorf("golden gate 3DEP: %w", err)
	}
	defer raster.Close()
	ref := raster.Reference()
	if ref.Model != 1 || ref.ProjectedCRS != 26910 || ref.LinearUnits != 9001 {
		return thing.Thing{}, fmt.Errorf("golden gate 3DEP: expected EPSG:26910 metre raster")
	}
	land, err := islandLand(raster, grid)
	return thing.Thing{Shape: land, Material: gateShore}, err
}

// goldenGateBridge builds the two towers, deck, suspension cables and
// approaches in the same grid as the pier and island geometry.
func goldenGateBridge(grid Grid) ([]thing.Thing, error) {
	if !grid.valid() {
		return nil, fmt.Errorf("golden gate bridge: finite local grid required")
	}
	var made scenery
	axis := maths.Rotation(maths.Up, math.Atan2(1240.47, -75.58))
	site := buildingSite{made: &made, base: GoldenGateCentre(grid), turn: axis}
	for _, sign := range [...]float64{-1, 1} {
		site.gateTower(sign * gateHalfMain)
	}
	const halfLength = 1368.5
	const panel = 22.
	count := int(math.Ceil(2 * halfLength / panel))
	for i := range count {
		low := -halfLength + 2*halfLength*float64(i)/float64(count)
		high := -halfLength + 2*halfLength*float64(i+1)/float64(count)
		x := (low + high) / 2
		deck := gateRoad(x)
		site.block(maths.V(x, deck-.55, 0), maths.V(high-low+.22, 1.1, 27.43), gateDeck)
		for _, side := range [...]float64{-1, 1} {
			z := side * 12.6
			for _, y := range [...]float64{-1.35, -7.15} {
				site.gateMember(maths.V(low, gateRoad(low)+y, z), maths.V(high, gateRoad(high)+y, z), 1.1)
			}
			site.gateMember(maths.V(low, gateRoad(low)-7.15, z), maths.V(high, gateRoad(high)-1.35, z), .82)
		}
		if math.Abs(x) > gateHalfMain+gateSide && i%4 == 0 {
			site.block(maths.V(x, (deck-8)/2, 0), maths.V(6.2, deck-8, 19), concrete)
		}
	}
	site.gateCables()
	return made.parts, made.err
}

// gateTower adds the tower at x along the bridge: a base, and on each side
// a leg in tapering courses with cross members between.
func (site buildingSite) gateTower(x float64) {
	site.block(maths.V(x, 2, 0), maths.V(41, 12, 49), concrete)
	site.block(maths.V(x, 10, 0), maths.V(29, 5, 39), concrete)
	for _, side := range [...]float64{-1, 1} {
		z := side * 13.45
		for _, course := range [...][4]float64{
			{12, 76, 15.5, 9.6}, {76, 132, 12.2, 8}, {132, 176, 10.6, 7.4},
			{176, 211, 9.3, 6.7}, {211, 223, 7.9, 6.5}, {223, gateTowerTop, 10.0, 8.9},
		} {
			site.block(maths.V(x, (course[0]+course[1])/2, z),
				maths.V(course[2], course[1]-course[0]+.1, course[3]), gateOrange)
		}
	}
	for _, cross := range [...][2]float64{{46, 5.2}, {66, 5.4}, {109, 7.2}, {151, 7.4}, {190, 7.4}, {212, 6.0}} {
		site.block(maths.V(x, cross[0], 0), maths.V(7, cross[1], 27), gateOrange)
	}
}

// gateMember adds a post of a width between two points in the bridge's own
// frame.
func (site buildingSite) gateMember(from, to maths.Vec, width float64) {
	site.made.post(site.base.Add(site.turn.Out(from)), site.base.Add(site.turn.Out(to)), width, gateOrange)
}

// gateCables adds the main cable on each side as short posts: the main span
// and the two side spans, with a hanger down to the road where there is room.
func (site buildingSite) gateCables() {
	for _, span := range [...][3]float64{
		{-gateHalfMain - gateSide, -gateHalfMain, 16},
		{-gateHalfMain, gateHalfMain, 64},
		{gateHalfMain, gateHalfMain + gateSide, 16},
	} {
		low, high, count := span[0], span[1], int(span[2])
		for _, side := range [...]float64{-1, 1} {
			z := side * 13.45
			last := maths.V(low, gateCable(low), z)
			for step := 1; step <= count; step++ {
				x := low + (high-low)*float64(step)/float64(count)
				next := maths.V(x, gateCable(x), z)
				site.gateMember(last, next, .92)
				if step < count && next.Y > gateRoad(x)+1 {
					site.gateMember(maths.V(x, gateRoad(x), z), next, .47)
				}
				last = next
			}
		}
	}
}

// gateRoad is the height of the roadway at x along the bridge, in metres.
func gateRoad(x float64) float64 {
	along := x / 1368.5
	return 75.4 - 12*along*along
}

// gateCable is the height of the main cable at x along the bridge, in
// metres.
func gateCable(x float64) float64 {
	offset := math.Abs(x)
	if offset <= gateHalfMain {
		ratio := offset / gateHalfMain
		return 82.7 + (gateTowerTop-82.7)*ratio*ratio
	}
	along := (offset - gateHalfMain) / gateSide
	return gateTowerTop - 202*along + 41*along*along
}
