package trace

import (
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/medium"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
)

func TestClearCloudOverlayRequiresVisibleWitness(t *testing.T) {
	cloud, err := medium.NewClearCloud(medium.Puff{Radius: 1})
	if err != nil {
		t.Fatal(err)
	}
	ordinary := thing.Volume{Medium: medium.Puff{Radius: 1}}
	overlay := thing.Volume{Medium: cloud}
	hiddenOrdinary, hiddenOverlay := ordinary, overlay
	hiddenOrdinary.Attrs.Hidden = true
	hiddenOverlay.Attrs.Hidden = true
	for _, row := range []struct {
		name    string
		volumes []thing.Volume
		want    bool
	}{
		{"none", nil, false},
		{"hidden ordinary", []thing.Volume{hiddenOrdinary}, false},
		{"hidden overlay", []thing.Volume{hiddenOverlay}, false},
		{"visible ordinary", []thing.Volume{ordinary}, false},
		{"ordinary and hidden overlay", []thing.Volume{ordinary, hiddenOverlay}, false},
		{"visible overlay", []thing.Volume{overlay}, true},
		{"overlay and hidden ordinary", []thing.Volume{overlay, hiddenOrdinary}, true},
		{"mixed visible", []thing.Volume{ordinary, overlay}, false},
	} {
		t.Run(row.name, func(t *testing.T) {
			world := thing.NewWorld()
			for _, volume := range row.volumes {
				world.AddVolume(volume)
			}
			if got := HasClearCloudOverlay(world); got != row.want {
				t.Fatalf("overlay = %v, want %v", got, row.want)
			}
		})
	}
}

func TestHiddenVolumesRetainNamedClearCPUCompatibility(t *testing.T) {
	water := opticalWater(t, "hidden-only water", maths.Grey(.1), 1)
	world := thing.NewWorld()
	world.Add(thing.Thing{Shape: shape.Plane{}, Material: &thing.Material{
		Through: 1, Index: 1.333, ClearMedium: water,
	}})
	world.AddVolume(thing.Volume{Medium: medium.Puff{Radius: 1},
		Attrs: thing.Attrs{Hidden: true}})
	initial := []InitialMedium{{Index: 1.333, ClearMedium: water}}
	if _, err := PrepareClearState(world, initial); err != nil {
		t.Fatalf("hidden volume changed named clear eligibility: %v", err)
	}
}
