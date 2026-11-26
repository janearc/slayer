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

// SampledRegionKept draws box from a full frame, keeping region-local rows.
// The caller must separate retained rows by full draw identity and box.
func SampledRegionKept(world *thing.World, eye maths.Subspace, lens film.Sensor,
	moment time.Duration, width, height, rays int, box image.Rectangle,
	media []InitialMedium, keeping Keeping) (*frame.Frame, error) {
	if width < 1 || height < 1 || rays < 1 || box.Empty() ||
		box.Min.X < 0 || box.Min.Y < 0 || box.Max.X > width || box.Max.Y > height {
		return nil, fmt.Errorf("trace: region %v must be inside a positive %d by %d frame with positive rays", box, width, height)
	}
	maximum := int(^uint(0) >> 1)
	if width > maximum/rays || height > maximum/rays {
		return nil, fmt.Errorf("trace: the fine frame dimensions overflow")
	}
	seen, err := newView(world, eye, lens, moment, box.Dx(), box.Dy(), rays)
	if err != nil {
		return nil, err
	}
	seen.width, seen.height = width, height
	seen, err = seen.withInitialMedia(media)
	if err != nil {
		return nil, err
	}
	fine := seen.fine()
	scene, err := fine.staging()
	if err != nil {
		return nil, err
	}
	if scene.walked != nil {
		keeping.rows = nil
	}
	made := &painting{keeping: keeping, fine: fine, rays: rays,
		scene: scene, wide: fine.wide(), regionLeft: box.Min.X, regionTop: box.Min.Y,
		picture: frame.New(box.Dx(), box.Dy()), left: make(chan int, box.Dy())}
	made.paint()
	if err := scene.refusal(); err != nil {
		return nil, err
	}
	if done := int(made.finished.Load()); done < box.Dy() {
		return nil, fmt.Errorf("trace: the region draw stopped with %d of %d rows finished: %w", done, box.Dy(), keeping.stop.Err())
	}
	return made.picture, nil
}
