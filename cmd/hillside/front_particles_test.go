package main

import (
	"fmt"
	"math"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
)

func TestFrontFirebrandsKeepPerTargetSeedsAndBurnWindows(t *testing.T) {
	given := frontOnFixture(t)
	raw, err := os.ReadFile(given.set)
	if err != nil {
		t.Fatal(err)
	}
	study, err := os.ReadFile("../../doc/fire-night.yaml")
	if err != nil {
		t.Fatal(err)
	}
	start := strings.Index(string(study), "      sparks:\n")
	if start < 0 {
		t.Fatal("shipped night scene has no sparks")
	}
	end := strings.Index(string(study[start:]), "      light-cells:")
	if end < 0 {
		t.Fatal("shipped sparks block has no following sampling setting")
	}
	block := string(study[start : start+end])
	text := strings.Replace(string(raw), "      flame:", block+"      flame:", 1)
	if text == string(raw) {
		t.Fatal("front fixture has no flame block")
	}
	if err := os.WriteFile(given.set, []byte(text), 0600); err != nil {
		t.Fatal(err)
	}
	_, _, ready, err := staged(given)
	if err != nil {
		t.Fatal(err)
	}
	defer ready.live.closeBeds()
	if len(ready.live.fronts) != 1 || len(ready.live.fronts[0].sparks) != 4 {
		t.Fatal("one particle source per placed target was not retained")
	}
	front := ready.live.fronts[0]
	shared := front.sparks[0].sparks.shape
	trajectories := map[maths.Vec]bool{}
	for index, event := range front.sparks {
		// Keep the real plan and source; isolate each scheduled particle
		// stream so other burning targets cannot mask a broken window.
		only := front
		only.groups, only.smoke = nil, nil
		only.sparks = []scheduledSparks{event}
		for _, sample := range []struct {
			name string
			age  time.Duration
			want int
		}{
			{"before ignition", -time.Nanosecond, 0},
			{"half second of births", 500 * time.Millisecond, 80},
			{"airborne after burnout", 3 * time.Second, 320},
			{"last second airborne", 5 * time.Second, 160},
			{"all expired", 6 * time.Second, 0},
		} {
			t.Run(fmt.Sprintf("target%d/%s", index, sample.name), func(t *testing.T) {
				world, made := thing.NewWorld(), tally{}
				if err := only.addAt(world, &made, event.start+sample.age); err != nil {
					t.Fatal(err)
				}
				count := len(world.Things)
				// 160/s for two seconds, four-second life; a seeded birth
				// phase may move one particle across a sampling boundary.
				if (sample.want == 0 && count != 0) || count < sample.want-1 || count > sample.want+1 || made.sparks != count {
					t.Fatalf("birth/death count %d, tally %d, want %d", count, made.sparks, sample.want)
				}
				if len(world.Lamps) != 0 {
					t.Fatal("each particle acquired its own lamp")
				}
				for _, particle := range world.Things {
					glow := particle.Material.Glow
					place := particle.At(0)
					if particle.Shape != shared || place.Scale != .025 || !(glow.R+glow.G+glow.B > 0) {
						t.Fatal("particle lost shared geometry, physical radius or glow")
					}
					if distance := place.Origin.Len(); !(distance >= 0) || math.IsInf(distance, 0) {
						t.Fatal("particle has a nonfinite placement")
					}
				}
				if sample.age == 500*time.Millisecond {
					trajectories[world.Things[0].At(0).Origin.Sub(event.sparks.place.Origin)] = true
				}
			})
		}
	}
	if len(trajectories) < 2 {
		t.Fatal("different target seeds produced identical relative trajectories")
	}
}
