//go:build draw && darwin

package gpu

import (
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/medium"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
	"strings"
	"testing"
)

// Poison the lamp-query branch: an emissive-only volume must return its own
// light before reaching it. This checks the compiled path, not a stopwatch.
func TestGravityEmissiveVolumeDoesNotAskForLampShadows(t *testing.T) {
	anchor := "int count = asked(world, point, float3(0), asks);"
	if strings.Count(traceSource, anchor) != 1 {
		t.Fatal("emission probe anchor changed")
	}
	original := traceSource
	traceSource = strings.Replace(traceSource, anchor, "return float3(-123);\n"+anchor, 1)
	defer func() { traceSource = original }()
	world := thing.NewWorld()
	world.AddVolume(thing.Volume{Medium: medium.Puff{Radius: 1, Thick: 1}, Material: &thing.Material{Glow: maths.RGB(1, .5, .2)}})
	within(t, world, maths.Subspace{Origin: maths.V(0, 0, 4)}, 16, 12, 1, exact)
}

func TestGravitySegmentStopsAtItsEnd(t *testing.T) {
	anchor := "Seen seen = shade(world, from, way);"
	if strings.Count(traceSource, anchor) != 1 {
		t.Fatal("segment probe anchor changed")
	}
	original := traceSource
	traceSource = strings.Replace(traceSource, anchor, `
 Hit segment=first_limited(world,from,way,0,1);
 Seen seen={float3(segment.material<0 ? segment.distance : -123.0f),
 0,nowhere,float3(0),float3(0),unpainted,unpainted};`, 1)
	defer func() { traceSource = original }()
	world := thing.NewWorld()
	world.Add(thing.Thing{Shape: shape.Sphere{Radius: 1}})
	picture, err := Sampled(world, maths.Subspace{Origin: maths.V(0, 0, 4)}, nil, 0, 2, 2, 1)
	if err != nil {
		t.Fatal(err)
	}
	for y := range 2 {
		for x := range 2 {
			if p := picture.At(x, y); p.R != 1 || p.G != 1 || p.B != 1 {
				t.Fatal("segment queried past its end", p)
			}
		}
	}
}
