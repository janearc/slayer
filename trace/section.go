package trace

import (
	"fmt"
	"image"
	"time"

	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/render/frame"
	"github.com/janearc/slayer/scene/thing"
)

// sampledIn is Sampled for one section of the frame: the pixels of box,
// in a frame of the box's size, each the same in every bit as Sampled
// makes it in the whole frame. So a frame can be drawn in pieces, a piece
// at a time or kept from an earlier frame.
//
// What Sampled refuses is refused, and so is a box that is empty or
// reaches outside the frame.
func sampledIn(world *thing.World, eye maths.Subspace, lens film.Sensor,
	moment time.Duration, width, height, rays int,
	box image.Rectangle) (*frame.Frame, error) {
	drawn, err := SampledEach(world, eye, lens, moment, width, height, rays,
		[]image.Rectangle{box})
	if err != nil {
		return nil, err
	}
	return drawn[firstBox], nil
}

const (
	// firstBox is the place of the one section sampledIn asks for. It is
	// defined as the first place of a list, which is zero.
	firstBox = 0
	// corner is where a frame's pixels start counting, on either side.
	// It is defined as zero.
	corner = 0
	// onePerPixel is a ray count of a single ray a pixel, where nothing
	// is averaged down. It is defined as one.
	onePerPixel = 1
)

// SampledEach is sampledIn for each of several sections of one frame, in
// the order given, with the world made ready once for all of them.
func SampledEach(world *thing.World, eye maths.Subspace, lens film.Sensor,
	moment time.Duration, width, height, rays int,
	boxes []image.Rectangle) ([]*frame.Frame, error) {
	seen, err := newView(world, eye, lens, moment, width, height, rays)
	if err != nil {
		return nil, err
	}
	for _, box := range boxes {
		if box.Empty() || !box.In(image.Rect(corner, corner, width, height)) {
			return nil, fmt.Errorf("trace: a section is some of the "+
				"frame, and %v is not within %d by %d", box, width, height)
		}
	}
	scene := stage(seen.world, seen.at)
	if err := scene.refusal(); err != nil {
		return nil, err
	}
	fine := seen.fine()
	wide := fine.wide()
	drawn := make([]*frame.Frame, len(boxes))
	for at, box := range boxes {
		drawn[at] = fine.section(scene, wide, box, rays)
	}
	return drawn, nil
}

// sectionPixels is what a section's pixels are made from: the fine view,
// the scene made ready, the width of a ray, and the pixel of the fine
// frame where the section starts.
type sectionPixels struct {
	fine      view
	scene     *staged
	wide      float64
	left, top int
}

// pixel is the fine frame's pixel at x, y of the section.
func (from sectionPixels) pixel(x, y int) frame.Pixel {
	return from.fine.through(from.scene, from.wide, x+from.left, y+from.top)
}

// section is one section of a fine view's frame, drawn through a scene
// made ready, and averaged down where there are several rays a pixel.
func (fine view) section(scene *staged, wide float64, box image.Rectangle,
	rays int) *frame.Frame {
	from := sectionPixels{fine: fine, scene: scene, wide: wide,
		left: box.Min.X * rays, top: box.Min.Y * rays}
	drawn := paint(box.Dx()*rays, box.Dy()*rays, from.pixel)
	if rays == onePerPixel {
		return drawn
	}
	return averaged(drawn, box.Dx(), box.Dy(), rays)
}
