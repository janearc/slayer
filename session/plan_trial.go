package session

import (
	"os"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/strategies"
	"github.com/janearc/slayer/set"
)

// A trial of the plan, for measuring it and nothing else: where the
// environment says SLAYER_TRIAL_PLAN=1, staging leaves out every thing
// and every stand member that the plan finds out of the camera's view.
// It drops what a shadow or a reflection from out of view would show, so
// a picture drawn under it is compared with one drawn without it, from
// the same binary, and is not kept.

// planTrialVariable names the environment switch of the trial.
const planTrialVariable = "SLAYER_TRIAL_PLAN"

// planTrial is what the trial knows once the things are placed: the
// camera, the things it does not see, and each stand's member size.
type planTrial struct {
	eye     strategies.Eye
	unseen  map[string]bool
	members map[string]float64
}

// trialSighted places the set's things, works out the camera and which
// things it does not see, where the trial is asked for.
func (stage *staging) trialSighted(read set.Set) error {
	if os.Getenv(planTrialVariable) != "1" {
		return nil
	}
	things, err := stage.plannedThings(read)
	if err != nil {
		return err
	}
	eye, err := stage.seen()
	if err != nil {
		return err
	}
	trial := &planTrial{eye: eye, unseen: map[string]bool{},
		members: map[string]float64{}}
	for _, one := range things {
		if one.line.verdict == "" && !eye.Cover(one.box).Visible {
			trial.unseen[one.line.name] = true
		}
	}
	stage.trial = trial
	return nil
}

// trialUnseen is whether the trial leaves out a thing by its name.
func (stage *staging) trialUnseen(name string) bool {
	return stage.trial != nil && stage.trial.unseen[name]
}

// trialMemberUnseen is whether the trial leaves out a stand's member
// standing at a place: the ball that holds its largest design, stood
// there, is out of view.
func (stage *staging) trialMemberUnseen(each set.Stand, seed uint64,
	place maths.Vec) bool {
	if stage.trial == nil {
		return false
	}
	reach, known := stage.trial.members[each.Name()]
	if !known {
		box, _, err := stage.memberExtent(each, seed)
		if err != nil {
			reach = -1
		} else {
			reach = box.Max.Sub(box.Min).Len()
		}
		stage.trial.members[each.Name()] = reach
	}
	if reach < 0 {
		return false
	}
	return !stage.trial.eye.SeesBall(place.Add(maths.V(0, reach/2, 0)),
		reach/2)
}
