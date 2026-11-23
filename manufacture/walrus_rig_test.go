package manufacture

import (
	"slices"
	"testing"

	"github.com/janearc/slayer/maths"
)

func TestWalrusSkinRemainsConnectedAndDeforms(t *testing.T) {
	rig := NewWalrusRig()
	rest, err := rig.Skin(SealRigPose{})
	if err != nil {
		t.Fatal(err)
	}
	turn, err := rig.Skin(SealRigPose{Yaw: .85, Fore: [2]float64{.25, -.15}})
	if err != nil {
		t.Fatal(err)
	}
	if !rest.Closed() || !turn.Closed() || !sealSkinConnected(turn) || !slices.Equal(rest.Faces(), turn.Faces()) {
		t.Fatal("turn must retain the same single closed skin")
	}
	if slices.Equal(rest.Points(), turn.Points()) {
		t.Fatal("turn did not deform the skin")
	}
	parts, err := Walrus(rig, SealRigPose{}, maths.NewSubspace(maths.Vec{}, maths.Identity, 1))
	if err != nil || len(parts) < 10 {
		t.Fatalf("missing face and vibrissae: %d, %v", len(parts), err)
	}
}
