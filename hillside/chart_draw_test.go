//go:build draw

package hillside

import (
	"os"
	"strings"
	"testing"

	"github.com/janearc/slayer/planet"
	"github.com/janearc/slayer/scene/texture"
)

// onePlace is how many places of the planet's block take the added line.
// It is defined as one: the block says made_up once.
const onePlace = 1

// giantShine is the shine of the giant in Melancholia's planet staged
// alone, on the gpu or not, with a line added to the planet's block.
func giantShine(t *testing.T, onGPU bool, line string) texture.Texture {
	t.Helper()
	given := mustParse(t)
	given.set = melancholiaAlone(t)
	if line != "" {
		written, err := os.ReadFile(given.set)
		if err != nil {
			t.Fatal(err)
		}
		changed := strings.Replace(string(written), "    made_up: true\n",
			"    made_up: true\n"+line, onePlace)
		if changed == string(written) {
			t.Fatal("the planet's block does not say made_up as expected")
		}
		if err := os.WriteFile(given.set, []byte(changed), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	given.onGPU = onGPU
	world, _, _, err := staged(given)
	if err != nil {
		t.Skipf("the set cannot be staged here: %v", err)
	}
	for _, each := range world.Things {
		if each.Material != nil && each.Material.Shine != nil {
			return each.Material.Shine
		}
	}
	t.Fatal("the staged set has no thing that shines")
	return nil
}

// on the gpu the giant's face is a chart, painted once; on the cpu, or
// where the planet says a chart of zero, it is the face itself, worked
// out for every ray, so the cpu stays the picture to check against.
func TestTheGiantIsChartedOnTheGPUAlone(t *testing.T) {
	if _, ok := giantShine(t, true, "").(*planet.Chart); !ok {
		t.Error("on the gpu the giant's face is not a chart")
	}
	if _, ok := giantShine(t, false, "").(*planet.Face); !ok {
		t.Error("on the cpu the giant's face is not the face itself")
	}
	if _, ok := giantShine(t, true, "    chart: 0\n").(*planet.Face); !ok {
		t.Error("with a chart of zero the giant's face is not the face itself")
	}
}
