package trace

import (
	"context"
	"errors"
	"runtime"
	"sync/atomic"
	"testing"

	"github.com/janearc/slayer/render/frame"
	"go.uber.org/mock/gomock"
)

// These scheduler checks use synthetic pixels and do not render a scene.
func TestExactRowsReturnsEveryCompletedPixel(t *testing.T) {
	var calls atomic.Int64
	picture, err := exactRows(Keeping{}, 3, 2, func(x, y int) (frame.Pixel, bool) {
		calls.Add(1)
		return frame.Pixel{R: float32(x), G: float32(y)}, true
	}, func() bool { return true })
	if err != nil || picture == nil {
		t.Fatalf("drawing synthetic rows: %v", err)
	}
	if calls.Load() != 6 || picture.At(2, 1).R != 2 || picture.At(2, 1).G != 1 {
		t.Fatalf("rows lost pixels or calls: %d, %v", calls.Load(), picture.At(2, 1))
	}
}

func TestExactRowsRefusesAStoppedPartialPicture(t *testing.T) {
	stop, cancel := context.WithCancel(context.Background())
	cancel()
	var calls atomic.Int64
	picture, err := exactRows(NewKeeping(stop, nil), 3, 2, func(x, y int) (frame.Pixel, bool) {
		calls.Add(1)
		return frame.Pixel{}, true
	}, func() bool { return true })
	if !errors.Is(err, context.Canceled) || picture != nil || calls.Load() != 0 {
		t.Fatalf("cancelled rows returned picture=%v, error=%v, calls=%d", picture, err, calls.Load())
	}
}

func TestExactRowsStopInsideARowDoesNotReturnAPartialPicture(t *testing.T) {
	prior := runtime.GOMAXPROCS(1)
	t.Cleanup(func() { runtime.GOMAXPROCS(prior) })
	stop, cancel := context.WithCancel(context.Background())
	defer cancel()
	rows := NewMockRows(gomock.NewController(t))
	rows.EXPECT().Begin(1)
	rows.EXPECT().Taken(0, 3).Return(nil, false)
	picture, err := exactRows(NewKeeping(stop, rows), 3, 1, func(x, y int) (frame.Pixel, bool) {
		cancel()
		return frame.Pixel{R: 1}, true
	}, func() bool { return true })
	if !errors.Is(err, context.Canceled) || picture != nil {
		t.Fatalf("partial row returned picture=%v, error=%v", picture, err)
	}
}

func TestExactRowsReturnsACompletePictureWhenStopFollowsItsLastPixel(t *testing.T) {
	stop, cancel := context.WithCancel(context.Background())
	defer cancel()
	picture, err := exactRows(NewKeeping(stop, nil), 1, 1, func(x, y int) (frame.Pixel, bool) {
		cancel()
		return frame.Pixel{R: 3}, true
	}, func() bool { return true })
	if err != nil || picture == nil || picture.At(0, 0).R != 3 {
		t.Fatalf("completed picture returned picture=%v, error=%v", picture, err)
	}
}

func TestExactRowsTakesFinishedRowsAndKeepsTheRest(t *testing.T) {
	rows := NewMockRows(gomock.NewController(t))
	rows.EXPECT().Begin(2)
	rows.EXPECT().Taken(0, 2).Return([]frame.Pixel{{R: 4}, {R: 5}}, true)
	rows.EXPECT().Taken(1, 2).Return(nil, false)
	rows.EXPECT().Drawn(1, []frame.Pixel{{R: 6}, {R: 7}})
	var calls atomic.Int64
	picture, err := exactRows(NewKeeping(nil, rows), 2, 2, func(x, y int) (frame.Pixel, bool) {
		calls.Add(1)
		return frame.Pixel{R: float32(6 + x)}, true
	}, func() bool { return true })
	if err != nil || picture == nil || calls.Load() != 2 {
		t.Fatalf("resumed rows: picture=%v, error=%v, calls=%d", picture, err, calls.Load())
	}
	if picture.At(0, 0).R != 4 || picture.At(1, 0).R != 5 || picture.At(0, 1).R != 6 || picture.At(1, 1).R != 7 {
		t.Fatalf("resumed picture has wrong rows: %v", picture.Pix)
	}
}

func TestExactRowsRedrawsAStoredRowOfTheWrongWidth(t *testing.T) {
	rows := NewMockRows(gomock.NewController(t))
	rows.EXPECT().Begin(1)
	rows.EXPECT().Taken(0, 2).Return([]frame.Pixel{{R: 99}}, true)
	rows.EXPECT().Drawn(0, []frame.Pixel{{R: 2}, {R: 3}})
	picture, err := exactRows(NewKeeping(nil, rows), 2, 1, func(x, y int) (frame.Pixel, bool) {
		return frame.Pixel{R: float32(2 + x)}, true
	}, func() bool { return true })
	if err != nil || picture == nil || picture.At(0, 0).R != 2 || picture.At(1, 0).R != 3 {
		t.Fatalf("wrong-width row was taken: picture=%v, error=%v", picture, err)
	}
}

func TestExactRowsDoNotKeepARefusedSceneRow(t *testing.T) {
	rows := NewMockRows(gomock.NewController(t))
	rows.EXPECT().Begin(1)
	rows.EXPECT().Taken(0, 1).Return(nil, false)
	_, err := exactRows(NewKeeping(nil, rows), 1, 1, func(x, y int) (frame.Pixel, bool) {
		return frame.Pixel{R: 1}, true
	}, func() bool { return false })
	if err != nil {
		t.Fatalf("row scheduling failed before the caller could report the refusal: %v", err)
	}
}

func TestExactKeptRefusesAlreadyCancelledBeforeTracing(t *testing.T) {
	stop, cancel := context.WithCancel(context.Background())
	cancel()
	picture, err := ExactKept(smoking(), overTheScene, nil, 0, 1, 1, 7, nil, NewKeeping(stop, nil))
	if !errors.Is(err, context.Canceled) || picture != nil {
		t.Fatalf("cancelled exact draw returned picture=%v, error=%v", picture, err)
	}
}

func TestExactKeptTakesACompleteStoredPictureWithoutTracing(t *testing.T) {
	rows := NewMockRows(gomock.NewController(t))
	rows.EXPECT().Begin(1)
	rows.EXPECT().Taken(0, 1).Return([]frame.Pixel{{R: 8, G: 3, Depth: 2}}, true)
	picture, err := ExactKept(smoking(), overTheScene, nil, 0, 1, 1, 7, nil, NewKeeping(nil, rows))
	if err != nil || picture == nil || picture.At(0, 0).R != 8 || picture.At(0, 0).Depth != 2 {
		t.Fatalf("stored exact picture=%v, error=%v", picture, err)
	}
}
