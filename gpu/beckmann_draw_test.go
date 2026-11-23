//go:build draw && darwin

package gpu

import (
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
)

// seaGlintSlopes is a mean-square slope of the order Cox and Munk give a
// sea under a light wind, for Beckmann's lobe. Chosen.
const seaGlintSlopes = 0.02

// beckmannHall is the hall with its floor a glossy water whose highlight
// is Beckmann's, by its slope variance, as a set's cox-munk glint makes it.
func beckmannHall(t *testing.T, slopes float64) (*thing.World,
	maths.Subspace) {
	world, eye := hall(t)
	for _, one := range world.Things {
		if one.Material != nil && one.Material.Name == "floor" {
			one.Material = &thing.Material{Name: "glinting water",
				Colour: maths.Grey(0.05), Gloss: 1, Index: 1.333,
				SlopeVariance: slopes}
		}
	}
	return world, eye
}

// The kernel lights a highlight by Beckmann's lobe where a material has a
// slope variance, as slayer's tracer does.
func TestTheGPUDrawsBeckmannsHighlightAsSlayerDoes(t *testing.T) {
	world, eye := beckmannHall(t, seaGlintSlopes)
	against(t, world, eye)
}

// Anti-test: the slope variance reaches the kernel's picture. The hall with
// Beckmann's highlight and the hall with none are further apart on the gpu
// than the exact tolerance, so a kernel that dropped the slope variance
// fails the test above.
func TestASlopeVarianceChangesTheGPUsPicture(t *testing.T) {
	glinting, eye := beckmannHall(t, seaGlintSlopes)
	plain, _ := beckmannHall(t, 0)
	withLobe, _ := bothWaysAt(t, glinting, eye, swellColumns, swellRows,
		swellRays)
	without, _ := bothWaysAt(t, plain, eye, swellColumns, swellRows,
		swellRays)
	found := compare(without, withLobe, swellColumns, swellRows)
	if len(findings(found, exact)) == 0 {
		t.Errorf("the gpu drew the same with a slope variance as without: "+
			"%+v", found)
	}
}
