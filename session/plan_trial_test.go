package session

import (
	"fmt"
	"strings"
	"testing"

	"github.com/janearc/slayer/maths/strategies/scrooge"
)

// trialSetText is a set of two cans, one far ahead of the camera that looks
// north from the origin and one as far behind it, each at an absolute
// place so that no ground is wanted to stand it.
func trialSetText() string {
	var text strings.Builder
	text.WriteString(`scene:
  name: trial
  film: {width: 1, height: 1, rays: 1}
  cameras:
    between:
      title: between
      location:
        loc_relative: {object: ahead can}
      orientation:
        point_to: behind can
  set:
    things:
`)
	for _, each := range []struct {
		name  string
		north float64
	}{{"ahead can", trialDistance}, {"behind can", -trialDistance}} {
		fmt.Fprintf(&text, `      %s:
        kind: can
        seed: 1
        complexity: quick
        position:
          east: {value: 0, unit: m}
          north: {value: %g, unit: m}
          up: {value: 0, unit: m}
`, each.name, each.north)
	}
	return text.String()
}

// trialDistance is how far, in metres, each can of the trial's set stands
// from the camera, one ahead and one behind.
const trialDistance = 150.0

// stoodUnder is the stage that stood the trial's set with the trial's
// switch set to a value.
func stoodUnder(t *testing.T, switched string) *staging {
	t.Helper()
	t.Setenv(scrooge.Variable, "")
	t.Setenv(planTrialVariable, switched)
	read := readPositionSet(t, trialSetText())
	stage := warmStaging(t, read)
	if err := stage.stand(read); err != nil {
		t.Fatal(err)
	}
	return stage
}

// TestATrialStagesOnlyWhatThePlanFindsInView checks that where the trial
// is asked for the thing behind the camera is not stood, and the thing
// ahead of it is, and fewer parts are in the world than without it.
func TestATrialStagesOnlyWhatThePlanFindsInView(t *testing.T) {
	trial := stoodUnder(t, "1")
	if !trial.trialUnseen("behind can") || trial.trialUnseen("ahead can") {
		t.Error("the trial did not leave out the can behind the camera " +
			"and only it")
	}
	if len(trial.world.Things) == 0 {
		t.Error("the trial stood nothing, not even the can ahead")
	}
	whole := stoodUnder(t, "")
	if len(trial.world.Things) >= len(whole.world.Things) {
		t.Errorf("%d parts under the trial, %d without it",
			len(trial.world.Things), len(whole.world.Things))
	}
}

// TestWithTheTrialNotAskedEveryThingIsStood is the anti-test of the
// switch: unset, the stage holds no trial and leaves out neither can.
func TestWithTheTrialNotAskedEveryThingIsStood(t *testing.T) {
	stage := stoodUnder(t, "")
	for _, name := range []string{"ahead can", "behind can"} {
		if stage.trialUnseen(name) {
			t.Errorf("%s is unseen with no trial asked", name)
		}
	}
	if stage.trial != nil {
		t.Error("a stage with no trial asked holds one")
	}
}
