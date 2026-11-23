package manufacture

import (
	"errors"
	"math"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/maths"
)

// errEndlessCells is a branch whose cells along it cannot be counted: a
// spacing of nought, or a length that is not a number. A plant with more
// cells than its budget is grown, at its cost; this is the one refused.
var errEndlessCells = errors.New("manufacture: a branch's cells along it " +
	"are not a count")

// plantAttachment shares a cell without scaling its physical leaves.
type plantAttachment struct {
	prototype *plantCell
	pose      maths.Subspace
}

// attachments fills a terminal branch with physical-size cells.
func (plant seasonalPlant) attachments(plan plantPlan, detail string) ([]plantAttachment, error) {
	var cells []plantAttachment
	variants := int(plantValue("cell-variants"))
	for _, index := range plan.tips {
		axis := plan.axes[index]
		run := axis.end.Sub(axis.root)
		length := run.Len()
		count := spaceOne
		if plant.profile.Form != "rosette" {
			ratio := length / plant.profile.CellSpacing
			if math.IsInf(ratio, spaceZero) || math.IsNaN(ratio) {
				return nil, errEndlessCells
			}
			count = int(ratio) + spaceOne
		}
		for cell := range count {
			share := float64(cell+spaceOne) / float64(count)
			at := axis.root.Add(run.Scale(share))
			variant := int(axis.seed.draw(growing, cell, spaceZero) * float64(variants))
			prototype, err := plant.profile.cell(detail, variant)
			if err != nil {
				return nil, err
			}
			pose := maths.NewSubspace(at, maths.Upright(run), spaceOne)
			cells = append(cells, plantAttachment{prototype: prototype, pose: pose})
		}
	}
	return cells, nil
}

// bark changes roundness while retaining axis roots, ends and physical radii.
func (plan plantPlan) bark(detail string) ([]part, error) {
	var parts []part
	fine := cut{around: int(plantValue("stem-sides-" + detail))}
	for _, axis := range plan.woody() {
		run := axis.end.Sub(axis.root)
		length := measure.Metres(run.Len())
		tube, err := newTube(length,
			measure.Metres(axis.radius), measure.Metres(axis.tip))
		if err != nil {
			return nil, err
		}
		pose, err := spaceAlong(axis.root, run)
		if err != nil {
			return nil, err
		}
		parts = append(parts, part{shape: tube, fine: fine, space: pose, of: body})
	}
	return parts, nil
}

// bounds encloses every possible roundness without manufacturing all details.
func (plan plantPlan) bounds() maths.Box {
	woody := plan.woody()
	first := woody[spaceZero]
	box := maths.SegmentBox(first.root, first.end, first.radius)
	for _, axis := range woody[spaceOne:] {
		box = enclosing(box, maths.SegmentBox(axis.root, axis.end, axis.radius))
	}
	return box
}
