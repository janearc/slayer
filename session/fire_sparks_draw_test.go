//go:build draw

package session

import (
	"os"
	"strings"
	"testing"

	"github.com/janearc/slayer/maths"
)

func TestFireSparksBecomeSharedGPUReadyInstances(t *testing.T) {
	given := fireOnFixture(t)
	raw, err := os.ReadFile(given.set)
	if err != nil {
		t.Fatal(err)
	}
	block := "      sparks:\n" +
		"        rate: {value: 100, unit: s^-1}\n" +
		"        lifetime: {value: 3, unit: s}\n" +
		"        radius: {value: 0.025, unit: m}\n" +
		"        release-height: {value: 3, unit: m}\n" +
		"        launch-speed: {value: 9, unit: m/s}\n" +
		"        spread: {value: 2, unit: m/s}\n" +
		"        updraft: {value: 8, unit: m/s}\n" +
		"        drag-time: {value: 0.8, unit: s}\n" +
		"        gravity: {value: 9.80665, unit: m/s^2}\n" +
		"        temperature: {value: 1650, unit: K}\n" +
		"        end-temperature: {value: 900, unit: K}\n"
	text := strings.Replace(string(raw), "      flame:", block+"      flame:", 1)
	if err = os.WriteFile(given.set, []byte(text), 0644); err != nil {
		t.Fatal(err)
	}
	given.at = 5 * 1e9
	world, _, _, err := staged(given)
	if err != nil {
		t.Fatal(err)
	}
	var count int
	var first any
	for _, placed := range world.Things {
		if placed.Material == nil || placed.Material.Name != "firebrand" {
			continue
		}
		count++
		if placed.Material.Glow == (maths.Light{}) || placed.At(0).Scale != .025 {
			t.Fatal("spark lost size or thermal glow")
		}
		if first == nil {
			first = placed.Shape
		} else if placed.Shape != first {
			t.Fatal("particles duplicated mesh geometry")
		}
	}
	if count < 299 || count > 301 {
		t.Fatal("wrong live spark count")
	}
	if len(world.Lamps) > 10 {
		t.Fatal("sparks each became a lamp")
	}
}
