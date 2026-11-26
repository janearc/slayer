package trace

import (
	"context"
	"errors"
	"image"
	"runtime"
	"testing"

	"github.com/janearc/slayer/film"
)

func TestKeptRegionMatchesTheWholePicture(t *testing.T) {
	data := sectionFixture(t)
	eye, lens := sectionEye(data)
	for _, world := range sectionScenes(t) {
		for _, rays := range data.Rays {
			whole := must(Sampled(world, eye, lens, atStart, data.Frame.Width, data.Frame.Height, rays))
			for _, tile := range data.Tiles {
				box := tile.box()
				part := must(SampledRegionKept(world, eye, lens, atStart, data.Frame.Width, data.Frame.Height, rays, box, nil, Keeping{}))
				sameSection(t, tile.Name, whole, part, box)
			}
		}
	}
}

func TestSixtyFourRaysMatchAWholePicture(t *testing.T) {
	box := image.Rect(1, 0, 2, 1)
	whole := must(Sampled(smoking(), overTheScene, film.Pinhole{}, 0, 2, 2, 64))
	part := must(SampledRegionKept(smoking(), overTheScene, film.Pinhole{}, 0, 2, 2, 64, box, nil, Keeping{}))
	sameSection(t, "sixty-four rays", whole, part, box)
}

// Anti: an empty or outside rectangle cannot be drawn.
func TestKeptRegionRefusesAnInvalidRectangle(t *testing.T) {
	for _, box := range []image.Rectangle{
		{Min: image.Pt(1, 1), Max: image.Pt(1, 2)},
		{Min: image.Pt(-1, 0), Max: image.Pt(2, 2)},
		{Min: image.Pt(15, 0), Max: image.Pt(17, 2)},
	} {
		if _, err := SampledRegionKept(smoking(), overTheScene, film.Pinhole{}, 0, keptSide, keptSide, keptRays, box, nil, Keeping{}); err == nil {
			t.Errorf("%v was drawn", box)
		}
	}
}

func TestKeptRegionResumesItsOwnRows(t *testing.T) {
	workers := runtime.GOMAXPROCS(oneWorker)
	t.Cleanup(func() { runtime.GOMAXPROCS(workers) })
	box := image.Rect(2, 3, 10, 11)
	rows := newMemoryRows()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	rows.stopAt, rows.stop = stopAfter, cancel
	_, err := SampledRegionKept(smoking(), overTheScene, film.Pinhole{}, 0, keptSide, keptSide, keptRays, box, nil, NewKeeping(ctx, rows))
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("stopped draw gave %v", err)
	}
	if len(rows.kept) != stopAfter {
		t.Fatalf("kept %d rows", len(rows.kept))
	}
	rows.stop, rows.drawn = nil, 0
	part := must(SampledRegionKept(smoking(), overTheScene, film.Pinhole{}, 0, keptSide, keptSide, keptRays, box, nil, NewKeeping(context.Background(), rows)))
	if rows.drawn != box.Dy()-stopAfter {
		t.Errorf("redrew %d rows", rows.drawn)
	}
	whole := must(Sampled(smoking(), overTheScene, film.Pinhole{}, 0, keptSide, keptSide, keptRays))
	sameSection(t, "resumed", whole, part, box)
}

// Anti: a region stopped before drawing keeps no rows.
func TestKeptRegionStoppedFirstKeepsNothing(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	rows := newMemoryRows()
	_, err := SampledRegionKept(smoking(), overTheScene, film.Pinhole{}, 0, keptSide, keptSide, keptRays, image.Rect(1, 1, 4, 4), nil, NewKeeping(ctx, rows))
	if !errors.Is(err, context.Canceled) || len(rows.kept) != 0 {
		t.Fatalf("stopped: %v, %d rows", err, len(rows.kept))
	}
}

func TestRegionWorkLimitCountsOnlyItsSamples(t *testing.T) {
	_, err := SampledRegionKept(smoking(), overTheScene, film.Pinhole{}, 0, 4096, 4096, 8, image.Rect(4, 4, 5, 5), nil, Keeping{})
	if err != nil {
		t.Fatal(err)
	}
}

func TestKeptRegionWithSpreadMatchesTheWholePicture(t *testing.T) {
	box := image.Rect(3, 4, 11, 12)
	whole := must(Sampled(spreading(), overTheScene, film.Pinhole{}, 0, keptSide, keptSide, keptRays))
	part := must(SampledRegionKept(spreading(), overTheScene, film.Pinhole{}, 0, keptSide, keptSide, keptRays, box, nil, Keeping{}))
	sameSection(t, "spread", whole, part, box)
}

// Anti: invalid media and unrepresentable fine dimensions are refused.
func TestRegionRefusesBadMediaAndOverflow(t *testing.T) {
	box := image.Rect(0, 0, 1, 1)
	if _, err := SampledRegionKept(smoking(), overTheScene, film.Pinhole{}, 0, 8, 8, 1, box, []InitialMedium{{Index: -1}}, Keeping{}); err == nil {
		t.Error("invalid medium was taken")
	}
	maxInt := int(^uint(0) >> 1)
	if _, err := SampledRegionKept(smoking(), overTheScene, film.Pinhole{}, 0, maxInt, 8, 2, box, nil, Keeping{}); err == nil {
		t.Error("overflowing fine width was taken")
	}
}
