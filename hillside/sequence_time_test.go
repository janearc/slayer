package hillside

import (
	"testing"

	"github.com/janearc/slayer/job"
	"pgregory.net/rapid"
)

// The schema's bounds on a plan's clock, and a print it takes; the
// print is picked, by the job slice's author on 2026-11-12, as any
// print the two schemas take.
const (
	timedLeastRate  = 1
	timedMostRate   = 120
	timedMostStart  = 86400
	timedLeastSpan  = 0.001
	timedMostSpan   = 3600
	timedMostScene  = 86400
	timedKnee       = 0.5
	timedTop        = 8
	timedSampled    = 64
	timedStretchOne = 1
)

// timedInput is a smoke plan with the clock the trial draws.
func timedInput(trial *rapid.T) sequenceInput {
	in := sequenceInput{Smoke: sequenceOne, Output: "/frames",
		FPS: rapid.IntRange(timedLeastRate, timedMostRate).Draw(trial,
			"fps")}
	in.Start = sequenceQuantity{Value: rapid.Float64Range(sequenceZero,
		timedMostStart).Draw(trial, "start"), Unit: "s"}
	in.Duration = sequenceQuantity{Value: rapid.Float64Range(
		timedLeastSpan, timedMostSpan).Draw(trial, "duration"), Unit: "s"}
	if rapid.IntRange(sequenceZero, timedStretchOne).Draw(trial,
		"stretched") == timedStretchOne {
		in.SceneDuration = &sequenceQuantity{Value: rapid.Float64Range(
			timedLeastSpan, timedMostScene).Draw(trial, "scene"), Unit: "s"}
	}
	in.Film.Width, in.Film.Height, in.Film.Rays = sequenceOne,
		sequenceOne, sequenceOne
	in.Print.Knee, in.Print.Top = timedKnee, timedTop
	return in
}

// TestAJobsTimesAreTheSequencePlansTimes is the property that holds the
// two clocks together while both exist: a plan lifted to a job puts
// every frame where the plan's own clock does.
func TestAJobsTimesAreTheSequencePlansTimes(t *testing.T) {
	rapid.Check(t, func(trial *rapid.T) {
		in := timedInput(trial)
		plan := in.plan("/plan.yaml", nil, nil)
		lifted, err := job.Lift(in.lifted("plan.yaml", "/plan.yaml"))
		if err != nil {
			trial.Fatal(err)
		}
		times := lifted.Times()
		if len(times) != plan.frames {
			trial.Fatalf("the job has %d frames, the plan %d",
				len(times), plan.frames)
		}
		for range timedSampled {
			index := rapid.IntRange(sequenceZero, plan.frames-sequenceOne).
				Draw(trial, "index")
			if times[index] != plan.frameTime(index) {
				trial.Fatalf("frame %d: the job at %s, the plan at %s",
					index, times[index], plan.frameTime(index))
			}
		}
	})
}
