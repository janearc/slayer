package trace

import (
	"math"
	"strings"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
)

func TestInvalidHoleParametersCannotLeaveAPartialPlacedField(t *testing.T) {
	world := thing.NewWorld()
	world.Fields = []*thing.Field{{Mass: 1, Spin: 2, Reach: 10}}
	if got := placeFields(world, 0); len(got) != 0 {
		t.Fatalf("invalid Kerr spin was staged: %+v", got)
	}
}

func TestReplayHazeReportsTheSameBeerLawLightItReturns(t *testing.T) {
	walked := &walk{ray: 3}
	scene := staged{walked: walked, haze: thing.Haze{Density: .1, Colour: maths.RGB(.2, .3, .4)}}
	input := maths.RGB(1, 2, 3)
	got := scene.air(maths.ZAxis, 10, input)
	want := input.Scale(math.Exp(-1)).Add(scene.haze.Colour.Scale(1 - math.Exp(-1)))
	if got != want || len(walked.steps) != 1 || walked.steps[0].What != "the haze" || !strings.Contains(walked.steps[0].Said, "through=10") {
		t.Fatalf("haze replay: %v %+v", got, walked.steps)
	}
	if exactly(nil) != "none" || exactly((*maths.Vec)(nil)) != "maths.Vec" {
		t.Fatal("diagnostics expose an address or lose an absent value")
	}
}
