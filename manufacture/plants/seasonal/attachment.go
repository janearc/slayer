package seasonal

import (
	"errors"
	"math"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
)

// errEndlessCells is a branch whose cells along it cannot be counted: a
// spacing of nought, or a length that is not a number. A plant with more
// cells than its budget is grown, at its cost; this is the one refused.
var errEndlessCells = errors.New("manufacture: a branch's cells along it " +
	"are not a count")

// maxPlantAttachments bounds the number of placed cells in one specimen.
// An over-budget specimen is refused, never silently simplified.
const maxPlantAttachments = 100000

// plantAttachment shares a cell without scaling its physical leaves.
type Attachment struct {
	Wood, Green, Dry *mesh.Mesh
	Bounds           maths.Box
	Pose             maths.Subspace
}

// attachments fills a terminal branch with physical-size cells.
func (profile Profile) Attachments(plan Plan, detail string) ([]Attachment, error) {
	var cells []Attachment
	variants := int(profile.value("cell-variants"))
	for _, index := range plan.tips {
		axis := plan.axes[index]
		run := axis.end.Sub(axis.root)
		length := run.Len()
		count := spaceOne
		if profile.Form != "rosette" {
			ratio := length / profile.CellSpacing
			if math.IsInf(ratio, spaceZero) || math.IsNaN(ratio) || ratio < 0 || ratio >= float64(maxPlantAttachments) {
				return nil, errEndlessCells
			}
			count = int(ratio) + spaceOne
		}
		if count > maxPlantAttachments-len(cells) {
			return nil, errEndlessCells
		}
		for cell := range count {
			share := float64(cell+spaceOne) / float64(count)
			at := axis.root.Add(run.Scale(share))
			variant := int(axis.seed.Draw(growing, cell, spaceZero) * float64(variants))
			prototype, err := profile.cell(detail, variant)
			if err != nil {
				return nil, err
			}
			pose := maths.NewSubspace(at, maths.Upright(run), spaceOne)
			cells = append(cells, Attachment{Wood: prototype.wood, Green: prototype.green, Dry: prototype.dry, Bounds: prototype.bounds, Pose: pose})
		}
	}
	return cells, nil
}

// bounds encloses every possible roundness without manufacturing all details.
func (plan Plan) Bounds() maths.Box {
	woody := plan.woody()
	first := woody[spaceZero]
	box := maths.SegmentBox(first.root, first.end, first.radius)
	for _, axis := range woody[spaceOne:] {
		box = enclosing(box, maths.SegmentBox(axis.root, axis.end, axis.radius))
	}
	return box
}

type Axis struct {
	Root, End   maths.Vec
	Radius, Tip float64
}

func (plan Plan) Woody() []Axis {
	rows := plan.woody()
	result := make([]Axis, len(rows))
	for i, row := range rows {
		result[i] = Axis{row.root, row.end, row.radius, row.tip}
	}
	return result
}
