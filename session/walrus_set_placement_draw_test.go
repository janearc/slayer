//go:build draw

package session

import (
	"testing"

	"github.com/janearc/slayer/manufacture"
	"github.com/janearc/slayer/maths"
)

// Exercise the actual authored-set stage, including retained native attachments.
func TestWalrusSetStageKeepsNativeAttachments(t *testing.T) {
	given := mustParse(t)
	given.set = "../sets/bioforms/walrus-fresh-5.yaml"
	world, _, _, err := staged(given)
	if err != nil {
		t.Fatal(err)
	}
	source, err := manufacture.WalrusFresh5(0, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(world.Things) != len(source) {
		t.Fatalf("stage has %d parts, expected %d", len(world.Things), len(source))
	}
	place := world.Things[0].At(0)
	if place.Scale == 0 {
		t.Fatal("stage body lacks placement")
	}
	var offset bool
	for i, part := range source {
		want := part.At(0).Within(place)
		got := world.Things[i].At(0)
		if got != want {
			t.Fatalf("part %d (%s) lost placement: got %+v, want %+v", i, part.Material.Name, got, want)
		}
		if part.At(0).Origin != (maths.Vec{}) {
			offset = true
		}
	}
	if !offset {
		t.Fatal("test specimen has no displaced attachments")
	}
}
