//go:build draw

package cells_test

import (
	"image"
	"testing"

	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/render/cells/mocks"
	"github.com/janearc/slayer/render/colour"
	"github.com/janearc/slayer/trace"
	"go.uber.org/mock/gomock"
)

// directly is what the tracer draws of the smoke scene for frame number,
// developed with a plain tone, with nothing of the scene package in it.
func directly(t failer, number, wide, high int) *image.RGBA {
	t.Helper()
	var lens film.Pinhole
	var tone colour.Tone
	world, eye := smokeScene(t)
	at := paceOf(t, picked("start-ms"), picked("rate")).At(number)
	drawn, err := trace.Sampled(world, eye, lens, at, wide, high, once)
	if err != nil {
		t.Fatal(err)
	}
	return tone.Develop(drawn)
}

// A frame is what the tracer draws at the frame's time, developed. It is
// the size asked for.
func TestAFrameIsWhatTheTracerDrawsDeveloped(t *testing.T) {
	number, wide, high := picked("frame-number"), picked("frame-wide"),
		picked("frame-high")
	drawn, err := placed(t, nil).Frame(number, wide, high)
	if err != nil {
		t.Fatal(err)
	}
	got, ok := drawn.(*image.RGBA)
	if !ok {
		t.Fatalf("the frame is a %T; want an RGBA", drawn)
	}
	want := directly(t, number, wide, high)
	if got.Bounds() != want.Bounds() || string(got.Pix) != string(want.Pix) {
		t.Errorf("frame %d is not what the tracer draws developed", number)
	}
}

// The clock is told the time of the frame, once, before it is drawn.
func TestTheClockIsToldTheTimeOfEachFrame(t *testing.T) {
	clock := mocks.NewMockClock(gomock.NewController(t))
	pace := paceOf(t, picked("start-ms"), picked("rate"))
	number := picked("frame-number")
	clock.EXPECT().Set(pace.At(number)).Times(once)
	clock.EXPECT().Set(pace.At(number + once)).Times(once)
	scene := placed(t, clock)
	for next := number; next <= number+once; next++ {
		if _, err := scene.Frame(next, picked("frame-wide"),
			picked("frame-high")); err != nil {
			t.Fatal(err)
		}
	}
}
