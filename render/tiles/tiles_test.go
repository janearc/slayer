package tiles_test

import (
	"errors"
	"image"
	"math"
	"testing"

	"github.com/janearc/slayer/render/tiles"
)

func TestPartitionsCoverFrameAndRespectBudget(t *testing.T) {
	for _, tc := range []struct {
		width, height, side      int
		perSample, fixed, budget uint64
		wantCount                int
	}{
		{2160, 1920, 4, 16, 100, 1 << 30, 16},
		{13, 7, 1, 4, 20, 36, 29},
		{2, 3, 4, 1, 0, 1000, 6},
	} {
		seen := make([]int, tc.width*tc.height)
		count := 0
		err := tiles.Each(tc.width, tc.height, tc.side, tc.perSample, tc.fixed, tc.budget, func(tile image.Rectangle, estimated uint64) error {
			count++
			if tile.Empty() || !tile.In(image.Rect(0, 0, tc.width, tc.height)) || estimated > tc.budget {
				t.Fatalf("invalid tile %v with %d bytes", tile, estimated)
			}
			for y := tile.Min.Y; y < tile.Max.Y; y++ {
				for x := tile.Min.X; x < tile.Max.X; x++ {
					seen[y*tc.width+x]++
				}
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
		for _, visits := range seen {
			if visits != 1 {
				t.Fatalf("pixel visited %d times", visits)
			}
		}
		if tc.wantCount != 0 && count != tc.wantCount {
			t.Fatalf("got %d tiles, want %d", count, tc.wantCount)
		}
	}
}

func TestRejectsInvalidAndOverflowingInputs(t *testing.T) {
	for _, tc := range []struct {
		w, h, side            int
		sample, fixed, budget uint64
	}{
		{0, 1, 1, 1, 0, 1}, {1, 1, 0, 1, 0, 1}, {1, 1, 1, 0, 0, 1},
		{1, 1, 1, 1, 1, 1}, {1, 1, 2, 1, 0, 3}, {1, 1, 2, math.MaxUint64, 0, math.MaxUint64},
	} {
		if err := tiles.Each(tc.w, tc.h, tc.side, tc.sample, tc.fixed, tc.budget, func(image.Rectangle, uint64) error { return nil }); err == nil {
			t.Fatalf("accepted invalid input %+v", tc)
		}
	}
	if err := tiles.Each(1, 1, 1, 1, 0, 1, nil); err == nil {
		t.Fatal("accepted nil consumer")
	}
}

func TestConsumerFailureStopsIteration(t *testing.T) {
	stop := errors.New("stop")
	calls := 0
	err := tiles.Each(8, 8, 2, 1, 0, 4, func(image.Rectangle, uint64) error { calls++; return stop })
	if !errors.Is(err, stop) || calls != 1 {
		t.Fatalf("got %v after %d calls", err, calls)
	}
}

func TestRefineUsesSpatialCostAndRefusesUnfitPixels(t *testing.T) {
	var result []image.Rectangle
	estimate := func(r image.Rectangle) (uint64, error) {
		cost := uint64(r.Dx() * r.Dy())
		if r.Max.X > 4 {
			cost *= 4
		}
		return cost, nil
	}
	err := tiles.Refine(image.Rect(0, 0, 8, 4), 16, estimate, func(r image.Rectangle, bytes uint64) error {
		if bytes > 16 {
			t.Fatalf("%v exceeds budget", r)
		}
		result = append(result, r)
		return nil
	})
	if err != nil || len(result) < 3 {
		t.Fatalf("got %v and %v", result, err)
	}
	if err := tiles.Refine(image.Rect(0, 0, 1, 1), 3, func(image.Rectangle) (uint64, error) { return 4, nil }, func(image.Rectangle, uint64) error { return nil }); err == nil {
		t.Fatal("accepted an unfit pixel")
	}
}

func TestRefinePropagatesFailures(t *testing.T) {
	stop := errors.New("stop")
	for _, tc := range []struct {
		rect     image.Rectangle
		budget   uint64
		estimate func(image.Rectangle) (uint64, error)
		visit    func(image.Rectangle, uint64) error
	}{
		{image.Rect(math.MinInt, 0, math.MaxInt, 1), 1, func(image.Rectangle) (uint64, error) { return 0, nil }, func(image.Rectangle, uint64) error { return nil }},
		{image.Rectangle{}, 1, func(image.Rectangle) (uint64, error) { return 0, nil }, func(image.Rectangle, uint64) error { return nil }},
		{image.Rect(0, 0, 1, 1), 0, nil, nil},
	} {
		if tiles.Refine(tc.rect, tc.budget, tc.estimate, tc.visit) == nil {
			t.Fatal("accepted invalid planner inputs")
		}
	}
	err := tiles.Refine(image.Rect(0, 0, 4, 2), 1, func(image.Rectangle) (uint64, error) { return 0, stop }, func(image.Rectangle, uint64) error { return nil })
	if !errors.Is(err, stop) {
		t.Fatalf("estimate error: %v", err)
	}
	err = tiles.Refine(image.Rect(0, 0, 4, 2), 1, func(r image.Rectangle) (uint64, error) { return uint64(r.Dx() * r.Dy()), nil }, func(image.Rectangle, uint64) error { return stop })
	if !errors.Is(err, stop) {
		t.Fatalf("visitor error: %v", err)
	}
}
