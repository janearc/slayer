package main

import (
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/ninjafilm/lens"
	"github.com/janearc/slayer/scene/thing"
	"strings"
	"testing"
	"time"
)

func TestSequencePicturePropagatesRetainedDrawAndOverlayFailures(t *testing.T) {
	for _, fault := range []string{"retained", "draw", "overlay"} {
		t.Run(fault, func(t *testing.T) {
			given := mustParse(t)
			given.width, given.height, given.samples = 4, 4, 1
			given.live = &liveScene{base: *thing.NewWorld()}
			eye := maths.Subspace{Origin: maths.V(0, 1, 2), Orientation: maths.LookAt(maths.V(0, 1, 2), maths.Vec{})}
			given.eye = &eye
			want := ""
			switch fault {
			case "retained":
				given.live.unsupported = "fixture retained lighting failure"
				want = "fixture retained lighting failure"
			case "draw":
				given.onGPU = true
				given.cachedLighting = true
				given.glass = &lens.Thin{}
				want = "cached requires a pinhole film"
			case "overlay":
				given.onGPU = false
				given.overlay = &overlay{rows: 0}
				want = "rows"
			}
			picture, _, _, err := sequencePicture(sequencePlan{knee: .1, top: 100}, given, time.Second)
			if picture != nil || err == nil || !strings.Contains(err.Error(), want) {
				t.Fatalf("%s: image %v error %v", fault, picture, err)
			}
		})
	}
}
