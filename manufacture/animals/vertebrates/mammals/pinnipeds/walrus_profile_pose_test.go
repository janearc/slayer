package pinnipeds

import (
	"reflect"
	"testing"

	"github.com/janearc/slayer/bioforms/skeleton"
)

func TestWalrusProfilePoseTurnsItsOwnNeckRoot(t *testing.T) {
	profile := defaultPinniped("walrus")
	profile.Cervical = 9
	body := newWalrusBones(profile)
	if body.fault != nil {
		t.Fatal(body.fault)
	}
	got := walrusPoseOf(body, walrusHauledOut)
	want := skeleton.Pose{}.
		With("cervical 9", skeleton.NewTurn(walrusSideAxis, .28)).
		With("cervical 5", skeleton.NewTurn(walrusSideAxis, .16)).
		With("skull", skeleton.NewTurn(walrusSideAxis, -.26))
	if !reflect.DeepEqual(got, want) {
		t.Fatal("hauled pose does not use the authored neck root and midpoint")
	}
	for _, pose := range []walrusPose{walrusHauledOut, walrusSwimming} {
		frames := body.tree.Frames(walrusPoseOf(body, pose))
		bone, index := body.bone("cervical 9")
		if frames[index].Apply(bone.Tail()).Sub(bone.Tail()).Len() < 1e-6 {
			t.Fatalf("pose %v leaves its neck root unturned", pose)
		}
	}
}

func TestWalrusProfileRefusesFieldGridClipping(t *testing.T) {
	for _, tc := range []struct {
		lumbar int
		valid  bool
	}{{6, true}, {7, true}, {32, false}} {
		profile := defaultPinniped("walrus")
		profile.Lumbar = tc.lumbar
		rig, err := newWalrusRig(profile)
		if err != nil {
			t.Fatalf("lumbar %d: %v", tc.lumbar, err)
		}
		outline, err := rig.outline()
		if err != nil {
			t.Fatalf("lumbar %d: %v", tc.lumbar, err)
		}
		err = rig.checkFieldGrid(outline, walrusLying)
		if (err == nil) != tc.valid {
			t.Errorf("lumbar %d valid %v: %v", tc.lumbar, tc.valid, err)
		}
	}
}

func TestWalrusProfileCarriesLimbGirdlesWithTheirSpine(t *testing.T) {
	profile := defaultPinniped("walrus")
	baseline := newWalrusBones(profile)
	profile.Thoracic++
	profile.Lumbar++
	changed := newWalrusBones(profile)
	if baseline.fault != nil || changed.fault != nil {
		t.Fatalf("rigs: %v, %v", baseline.fault, changed.fault)
	}
	for _, pair := range [][2]string{{"thoracic 4", "scapula left"}, {"sacral 1", "innominate left"}} {
		oldParent, _ := baseline.bone(pair[0])
		oldLimb, _ := baseline.bone(pair[1])
		parent, _ := changed.bone(pair[0])
		limb, _ := changed.bone(pair[1])
		oldOffset := oldLimb.Head().Sub(oldParent.Head())
		newOffset := limb.Head().Sub(parent.Head())
		if newOffset.Sub(oldOffset).Len() > 1e-12 || parent.Head().Sub(oldParent.Head()).Len() < 1e-6 {
			t.Errorf("%s did not follow %s", pair[1], pair[0])
		}
	}
}
