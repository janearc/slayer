package door

// The plan: assess FILE --plan reads the set and its terrain and says,
// for each thing and stand, how big it is, how many pixels it covers at
// the job's size and camera, and whether a draw would build it, average
// it or drop it. Nothing else is made. See doc/plan-before-staging.md.

import (
	"fmt"

	"github.com/janearc/puzzlemouth/pzm"
	"github.com/janearc/slayer/session"
)

// planFlag asks assess for the plan of the job's first frame.
const planFlag = "plan"

// planTakes are the flags assess FILE --plan takes: the picture's size
// and camera, and the time.
var planTakes = []string{"width", "height", "at", "camera", planFlag}

// plan prints the plan of the file's first frame.
func (held assessing) plan(call pzm.Call, given string) error {
	drawer := drawing(held)
	run, err := drawer.planned(call, given, false)
	if err != nil {
		return err
	}
	text, err := session.Planned(run.wanted)
	if err != nil {
		return err
	}
	_, err = fmt.Fprint(call.Out(), text)
	return err
}
