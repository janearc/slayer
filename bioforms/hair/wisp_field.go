package hair

import (
	"fmt"

	"github.com/janearc/slayer/maths"
)

// WispGuide authors relative shape at one sparse chart anchor. Vector
// offsets are in the chart and metres. Lift adjusts the contact height.
type WispGuide struct {
	Heading, Shoulder, Tip maths.Vec
	Lift                   float64
}

// WispField shares relative shape across compatible scalp guides.
// It leaves each child on its own rooted course.
type WispField struct {
	surface *SurfaceGuides
	guides  []WispGuide
}

// NewWispField associates relative shape with each sparse surface guide.
func NewWispField(surface *SurfaceGuides, author func(GuideAnchor) WispGuide) (*WispField, error) {
	if surface == nil || author == nil {
		return nil, fmt.Errorf("hair: wisp field needs guides and an author")
	}
	field := &WispField{surface: surface, guides: make([]WispGuide, len(surface.anchors))}
	for i, anchor := range surface.anchors {
		guide := author(anchor)
		if !finite(guide.Heading.Len()) || !finite(guide.Shoulder.Len()) ||
			!finite(guide.Tip.Len()) || !finite(guide.Lift) {
			return nil, fmt.Errorf("hair: invalid wisp guide %d", i)
		}
		guide.Heading = tangentTo(guide.Heading, anchor.Normal).Unit()
		field.guides[i] = guide
	}
	return field, nil
}

// Shape blends neighbouring guides after transporting their frames to
// the root's chart normal and heading. The returned free offsets are
// measured in rest-model space. Low compatibility fades influence.
func (field *WispField) Shape(site Site, region int, heading maths.Vec) (CourseEnvelope, float64) {
	normal := site.ChartNormal
	direction := tangentTo(heading, normal).Unit()
	if direction == (maths.Vec{}) {
		return CourseEnvelope{}, 0
	}
	var shoulder, tip maths.Vec
	lift, total := 0., 0.
	for _, choice := range field.surface.Weights(site, region) {
		if choice.Share == 0 {
			continue
		}
		anchor := field.surface.anchors[choice.Anchor]
		guide := field.guides[choice.Anchor]
		along := tangentTo(transport(guide.Heading, anchor.Normal, normal), normal).Unit()
		share := choice.Share * maths.Smooth(.06, .75, along.Dot(direction))
		if share == 0 {
			continue
		}
		carry := func(offset maths.Vec) maths.Vec {
			return transport(transport(offset, anchor.Normal, normal), along, direction).Scale(share)
		}
		shoulder = shoulder.Add(carry(guide.Shoulder))
		tip = tip.Add(carry(guide.Tip))
		lift += share * guide.Lift
		total += share
	}
	scale := 1 / max(.28, total)
	return CourseEnvelope{Shoulder: site.ChartVector(shoulder.Scale(scale)), Tip: site.ChartVector(tip.Scale(scale))},
		lift * scale
}
