package conifers

import "testing"

func TestSelectedProfilesAndSourcesAreIndependent(t *testing.T) {
	book, err := Read("conifers.yaml", Embedded(), func(string) float64 { return .4 }, 1e6)
	if err != nil {
		t.Fatal(err)
	}
	first, _, err := book.Select("coast-redwood")
	if err != nil {
		t.Fatal(err)
	}
	joint := first.Trunk.Joints[0]
	if first.Shoots == nil {
		t.Fatal("missing branchlets")
	}
	shootJoint := first.Shoots.Wear.Joints[0]
	measured, _ := first.Notes()
	if len(measured) == 0 || measured[0].From == nil {
		t.Fatal("missing measured source")
	}
	source := measured[0].From.Key
	first.Trunk.Joints[0][0]++
	first.Shoots.Wear.Joints[0][1]++
	measured[0].From.Key = "changed"
	again, _, err := book.Select("coast-redwood")
	if err != nil {
		t.Fatal(err)
	}
	more, _ := again.Notes()
	if again.Trunk.Joints[0] != joint || again.Shoots.Wear.Joints[0] != shootJoint || more[0].From.Key != source {
		t.Fatal("caller changed the checked family book")
	}
}
