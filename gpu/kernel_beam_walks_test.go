package gpu

import (
	"strings"
	"testing"
)

// beamLoops are the two kernel functions that walk the beam list, each by
// itself.
var beamLoops = []string{"beam_march_boxes", "beam_march_apart"}

// doctoredIn is the text with the one place of old made new inside the
// named function only, as the two beam loops share their lines.
func doctoredIn(t *testing.T, text, name, old, now string) string {
	t.Helper()
	body, found := kernelBody(text, name)
	if !found {
		t.Fatalf("the kernel's text has no function %s to doctor", name)
	}
	return strings.Replace(text, body, doctored(t, body, old, now), 1)
}

// Anti: each beam loop, doctored alone to read the world's count of every
// lamp, is reported by name and only it; the check on the flood loop does
// not stand in for them.
func TestABeamLoopThatWalksEveryLampIsReported(t *testing.T) {
	for _, name := range beamLoops {
		again := doctoredIn(t, kernelText(t), name,
			"int beams = world.header->lamp_lists.z;",
			"int beams = world.header->counts.w;")
		faults := loopFaults(again)
		if len(faults) != 1 || !strings.HasPrefix(faults[0], name+" ") {
			t.Errorf("%s over every lamp gave %v", name, faults)
		}
	}
}

// Anti: each beam loop, doctored alone to walk another list, is reported
// by name and only it.
func TestABeamLoopThatWalksAnotherListIsReported(t *testing.T) {
	for _, name := range beamLoops {
		other := doctoredIn(t, kernelText(t), name,
			"lamp_list(world, beam_list)", "lamp_list(world, flood_list)")
		faults := loopFaults(other)
		if len(faults) != 1 || !strings.HasPrefix(faults[0], name+" ") {
			t.Errorf("%s over the flood list gave %v", name, faults)
		}
	}
}
