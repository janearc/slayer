package bay

import (
	"fmt"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/pier"
	"github.com/janearc/slayer/scene/thing"
)

type sitePoint struct{ east, north float64 }

// Centre lines are coarse picks from the USGS Aquatic Park aerial crop.
// The final Municipal point is the HALS CA-113-A survey reference.
var municipalLine = []sitePoint{
	{550473, 4184656}, {550466, 4184704}, {550460, 4184746},
	{550462, 4184791}, {550470, 4184834}, {550485, 4184874},
	{550508, 4184909}, {550540, 4184939}, {550575, 4184958},
	{550614, 4184967}, {550641, 4184965}, {550658.85, 4184953.55},
}

var hydeLine = []sitePoint{
	{550978, 4184718}, {550960, 4184780},
	{550918, 4184825}, {550866, 4184870}, {550819, 4184900},
}

// Piers builds the distant Municipal and Hyde Street decks and pilings.
// Deck heights are visual estimates above the scene's sea plane.
func Piers(grid Grid) ([]thing.Thing, error) {
	if !grid.valid() {
		return nil, fmt.Errorf("aquatic park piers: finite local origin required")
	}
	var scenery scenery
	municipal, err := placedPier(grid, municipalLine, 18.3, 3.1)
	if err != nil {
		return nil, err
	}
	scenery.parts = municipal
	end := municipalLine[len(municipalLine)-1]
	bulb := grid.at(end.east, end.north, grid.SeaElevation)
	scenery.frustum(bulb.Add(maths.V(0, 2.2, 0)), 16, 16, .9, pierConcrete)
	scenery.block(bulb.Add(maths.V(-1, 4.2, 0)), maths.V(5, 2.2, 5),
		maths.Identity, concrete)
	if scenery.err != nil {
		return nil, scenery.err
	}
	hyde, err := placedPier(grid, hydeLine, 10.5, 2.7)
	return append(scenery.parts, hyde...), err
}

// placedPier converts Aquatic Park's projected centerline to scene metres.
func placedPier(grid Grid, marks []sitePoint, width, height float64) ([]thing.Thing, error) {
	line := make([]maths.Vec, 0, len(marks))
	for _, mark := range marks {
		line = append(line, grid.at(mark.east, mark.north, grid.SeaElevation))
	}
	return pier.New(line, pier.Deck{
		Width: width, Height: height, Thickness: .82, Overlap: 1.1,
		PileBottom: -1.5, PileSpacing: 4, PileWidth: .55,
		Surface: pierConcrete, Pilings: pierConcrete,
	})
}
