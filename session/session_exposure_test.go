package session

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/janearc/slayer/ninjafilm/lens"
)

// The numbers the exposure's tests pick, each by the blur slice's
// author, 2026-11-12.
const (
	// exposureAperture and exposureFocus are a lens open two radii of
	// the swarm's star and focused twenty off, in the star's radii.
	exposureAperture = 2.0
	exposureFocus    = 20.0
	// exposureShare is a tenth of a second of open shutter in one draw.
	exposureShare = 100 * time.Millisecond
	// exposureAt is a second into the camera's path, while it moves.
	exposureAt = time.Second
)

// movingCamera is the camera path of the smoke sequence in the doc
// folder: three seconds along a line, then a second held still.
func movingCamera(t *testing.T) string {
	t.Helper()
	raw, err := os.ReadFile("../doc/camera-smoke-sequence.yaml")
	if err != nil {
		t.Fatal(err)
	}
	text := string(raw)
	return text[strings.Index(text, "camera:"):]
}

// an aperture over a set's rig is refused by name: a rig has its own lens
func TestAJobsApertureOverARigIsRefused(t *testing.T) {
	given := asked{set: "a set", glass: &lens.Thin{}, wide: 40, width: 4,
		height: 3, optics: optics{aperture: exposureAperture,
			focus: exposureFocus}}
	err := given.openLens()
	if err == nil || !strings.Contains(err.Error(), "rig") ||
		given.optics.opened != nil {
		t.Errorf("an aperture over a rig gave %v", err)
	}
	given.glass = nil
	if err := given.openLens(); err != nil || given.optics.opened == nil {
		t.Errorf("the same aperture with no rig gave %v", err)
	}
}

// an aperture no lens has is refused by name, with its numbers: one
// wider than its focus is far, which a job's own check refuses first
func TestALensNoLensCanBeIsRefused(t *testing.T) {
	given := asked{wide: 40, width: 4, height: 3,
		optics: optics{aperture: exposureFocus, focus: exposureFocus /
			shareHalves}}
	err := given.openLens()
	if err == nil || !strings.Contains(err.Error(), "the job's lens") {
		t.Errorf("an aperture twice its focus gave %v", err)
	}
}

// pathPlan is the camera plan of a smoke film along the moving camera.
func pathPlan(t *testing.T) sequencePlan {
	t.Helper()
	text := fmt.Sprintf("scene: {smoke: %d}\nfps: "+
		"%d\nduration: {value: 1, unit: s}\n", sessionPlainSmoke,
		shutterRate) + movingCamera(t)
	return cameraPlan(jobFrom(t, text))
}

// on the gpu a draw's eye is the path's at the start of its share of the
// shutter and the closing eye the path's at its end, the draw's time the
// middle
func TestOnTheGPUAnEyeMovesAcrossItsShareOfTheShutter(t *testing.T) {
	plan := pathPlan(t)
	given := asked{onGPU: true, wide: 40, width: 4, height: 3,
		optics: optics{share: exposureShare}}
	if err := plan.exposing(&given, exposureAt); err != nil {
		t.Fatal(err)
	}
	opens, err := plan.camera.At(exposureAt - exposureShare/shareHalves)
	if err != nil {
		t.Fatal(err)
	}
	closes, err := plan.camera.At(exposureAt + exposureShare/shareHalves)
	if err != nil {
		t.Fatal(err)
	}
	closing := given.optics.closing
	if closing == nil || *given.eye != *opens || *closing != *closes ||
		*opens == *closes {
		t.Errorf("the eye opens at %v and closes at %v; want %v and %v",
			given.eye, closing, opens, closes)
	}
}

// and nowhere else: on the cpu, with no share, or with no camera path,
// the eye is the one at the draw's own time and there is no closing eye
func TestAnEyeDoesNotMoveInADrawWithoutAShareOnTheGPU(t *testing.T) {
	plan := pathPlan(t)
	middle, err := plan.camera.At(exposureAt)
	if err != nil {
		t.Fatal(err)
	}
	for name, given := range map[string]asked{
		"on the cpu":    {optics: optics{share: exposureShare}},
		"with no share": {onGPU: true},
	} {
		given.wide, given.width, given.height = 40, 4, 3
		if err := plan.exposing(&given, exposureAt); err != nil {
			t.Fatal(err)
		}
		if given.optics.closing != nil || *given.eye != *middle {
			t.Errorf("%s the eye moves in a draw: %v to %v", name,
				given.eye, given.optics.closing)
		}
	}
	still := asked{onGPU: true, wide: 40, width: 4, height: 3,
		optics: optics{share: exposureShare}}
	if err := (sequencePlan{}).exposing(&still, exposureAt); err != nil ||
		still.optics.closing != nil || still.eye != nil {
		t.Errorf("with no camera path the eye moves: %v, %v",
			still.optics.closing, err)
	}
}
