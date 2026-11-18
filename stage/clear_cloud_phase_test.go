package stage_test

import (
	"testing"

	"github.com/janearc/slayer/scene/medium"
)

func TestFizzStagePreservesHGOverrideAndOmittedInheritance(t *testing.T) {
	changed := fizzOf(t, "        foot: [0, 0, 0]", "        foot: [0, 0, 0]\n        clear-water: true\n        phase-g: 0")
	cloud := changed.Medium.(medium.ClearCloud)
	g, explicit := cloud.PhaseAsymmetry()
	if !explicit || g != 0 {
		t.Fatal("explicit isotropic cloud lost")
	}
	inherited := fizzOf(t, "        foot: [0, 0, 0]", "        foot: [0, 0, 0]\n        clear-water: true").Medium.(medium.ClearCloud)
	if _, explicit := inherited.PhaseAsymmetry(); explicit {
		t.Fatal("omitted phase changed legacy inheritance")
	}
}
