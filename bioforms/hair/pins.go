package hair

import (
	"fmt"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
)

// PinSurface projects guide handles onto a region of the skin chart.
// Keep its source face and vertex order when constructing the groom.
type PinSurface struct {
	chart   *mesh.Mesh
	indices []int
	faces   [][3]int
}

// NewPinSurface indexes the chart, or the rest points without a chart.
// A filter can restrict projection to eligible triangles. Collapsed chart
// faces are skipped. An empty eligible region has no projections.
func NewPinSurface(surface Surface, filter func([3]maths.Vec) bool) (*PinSurface, error) {
	if err := checkSurface(surface.Points, surface.Faces); err != nil {
		return nil, err
	}
	if err := surface.checkChart(); err != nil {
		return nil, err
	}
	points := surface.Chart
	if points == nil {
		points = surface.Points
	}
	pins := new(PinSurface)
	for i, face := range surface.Faces {
		corners := surface.chartCorners(face)
		if filter != nil && !filter(corners) {
			continue
		}
		area := edges(points, face).area().Len()
		if !(area > 0 && finite(area)) {
			continue
		}
		pins.indices = append(pins.indices, i)
		pins.faces = append(pins.faces, face)
	}
	if len(pins.faces) == 0 {
		return pins, nil
	}
	chart, err := mesh.New(points, pins.faces, nil, false)
	if err != nil {
		return nil, fmt.Errorf("hair: pin surface: %w", err)
	}
	pins.chart = chart
	return pins, nil
}

// GuidePins pull the first cubic's two control handles toward the skin.
// Both zero pins leave the guide in its follicle frame.
type GuidePins struct {
	Lead, Bend GuidePin
}

// GuidePin is a handle target attached to one skin triangle. It has a
// height in metres and a blend share. Create it with PinSurface.Project.
type GuidePin struct {
	face          int
	corners       [3]int
	weights       [3]float64
	height, share float64
}

// Project returns the attachment and its point on the authoring surface.
// Height is measured along the posed normal and share ranges from zero
// to one. Use the returned point to exclude projections across a part or
// too far from the intended patch.
func (pins *PinSurface) Project(point maths.Vec, height, share float64) (GuidePin, maths.Vec, bool) {
	if pins.chart == nil || !(height >= 0 && finite(height) && fraction(share)) || share == 0 {
		return GuidePin{}, maths.Vec{}, false
	}
	near, ok := pins.chart.SurfaceNear(point)
	if !ok {
		return GuidePin{}, maths.Vec{}, false
	}
	pin := GuidePin{
		face: pins.indices[near.Face], corners: pins.faces[near.Face], weights: near.Weights,
		height: height, share: share,
	}
	return pin, near.Point, true
}

func (pin GuidePin) valid(faces [][3]int) bool {
	return pin.share == 0 || (pin.face >= 0 && pin.face < len(faces) && faces[pin.face] == pin.corners)
}

// guideOffsets correct the first cubic in the posed frame. The correction
// fades to zero before the join, retaining its position and tangent.
type guideOffsets struct {
	lead, bend maths.Vec
	end        float64
}

func (strand Strand) pinOffsets(origin maths.Vec, frame shaftFrame, points, normals []maths.Vec) guideOffsets {
	if strand.Pins == (GuidePins{}) {
		return guideOffsets{}
	}
	end := 1.
	if strand.Guide.Tail.active() {
		end = strand.Guide.Tail.At
	}
	return guideOffsets{
		lead: strand.Pins.Lead.offset(origin, frame.out(strand.Guide.Lead).Scale(strand.Length), points, normals),
		bend: strand.Pins.Bend.offset(origin, frame.out(strand.Guide.Bend).Scale(strand.Length), points, normals),
		end:  end,
	}
}

func (pin GuidePin) offset(origin, handle maths.Vec, points, normals []maths.Vec) maths.Vec {
	if pin.share == 0 {
		return maths.Vec{}
	}
	site := locate(points, normals, pin.face, pin.corners, pin.weights, edges(points, pin.corners))
	target := site.Point.Add(site.Normal.Scale(pin.height))
	return target.Sub(origin).Sub(handle).Scale(pin.share)
}

func (offsets guideOffsets) at(t float64) maths.Vec {
	if t >= offsets.end {
		return maths.Vec{}
	}
	u := t / offsets.end
	rest := 1 - u
	fade := 1 - maths.Smooth(.69, 1, u)
	return offsets.lead.Scale(3 * u * rest * rest * fade).
		Add(offsets.bend.Scale(3 * u * u * rest * fade))
}
