package main

import (
	"strings"
	"testing"
)

func TestLegacySceneTallyIncludesItsActualBaseLamps(t *testing.T) {
	for _, mode := range []string{"sun", "moon", "none"} {
		t.Run(mode, func(t *testing.T) {
			given := mustParse(t)
			text := strings.Replace(fixtureScene, "light: moon", "light: "+mode, 1)
			text = strings.Replace(text, "alight: true", "alight: false", 1)
			given.set = onFixture(t, fixtureScene, text)
			world, made, ready, err := staged(given)
			if err != nil {
				t.Fatal(err)
			}
			defer ready.live.closeBeds()
			want := 2
			if mode == "none" {
				want = 0
			}
			if len(world.Lamps) != want {
				t.Fatalf("real %s scene has %d lights, want %d", mode, len(world.Lamps), want)
			}
			if made.lamps != len(world.Lamps) {
				t.Fatalf("scene reported %d lamps for %d actual base lights", made.lamps, len(world.Lamps))
			}
		})
	}
}
