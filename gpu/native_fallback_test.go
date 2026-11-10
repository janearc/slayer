//go:build darwin

package gpu

import (
	"strings"
	"testing"

	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/metal"
	"github.com/janearc/slayer/scene/thing"
)

// This replayable fault forces real native candidates to use software. It
// must change the reported outcomes even when the resulting image is right.
func TestNativeCountsExposeARefinementFaultMaskedBySoftware(t *testing.T) {
	group, _ := groupFixture(t)
	world := thing.NewWorld()
	world.Add(thing.Thing{Shape: group,
		Material: &thing.Material{Glow: maths.Grey(1)}})
	from := maths.V(0, 4, 16)
	eye := maths.Subspace{Origin: from,
		Orientation: maths.LookAt(from, maths.Vec{})}
	lens := film.Pinhole{}
	const width, height = 32, 24
	encoded, err := encodeGrown(world, nil, eye, lens, 0, width, height)
	if err != nil {
		t.Fatal(err)
	}
	text := sourceWith(encoded.records())
	function := strings.Index(text, "static Hit native_face(")
	if function < 0 {
		t.Fatal("native refinement function missing")
	}
	brace := strings.Index(text[function:], "{") + function
	if brace < function {
		t.Fatal("native refinement body missing")
	}
	text = text[:brace+1] + "\nreturn {far, -1, float3(0)};\n" + text[brace+1:]
	kernel, err := metal.Open("#define SLAYER_NATIVE_SHADOWS 1\n"+
		"#define SLAYER_NATIVE_MESHES 1\n"+text, "trace")
	if err != nil {
		t.Fatal(err)
	}
	defer kernel.Close()
	plan, root, err := encoded.accelerationPlan()
	if err != nil {
		t.Fatal(err)
	}
	if err = kernel.Accelerate(plan, root); err != nil {
		t.Fatal(err)
	}
	inputs := append(encoded.inputs(), encoded.nativeFirsts())
	samples, err := kernel.Grid(width, height, rows, encoded.perSample(), inputs)
	if err != nil {
		t.Fatal(err)
	}
	counts := nativeCounts(samples, encoded.perSample())
	if counts.Refined != 0 || counts.Fallbacks == 0 ||
		counts.Fallbacks != counts.Candidates {
		t.Fatalf("refinement fault hidden: %+v", counts)
	}
	truth, err := Sampled(world, eye, lens, 0, width, height, 1)
	if err != nil {
		t.Fatal(err)
	}
	picture := averaged(finishing{world: world, records: encoded.records(),
		paints: encoded.paints, wide: wideOf(world, eye, lens, width, height)},
		samples, width, height, 1)
	for i, p := range truth.Pix {
		if p != picture.Pix[i] {
			t.Fatalf("fallback image changed at %d", i)
		}
	}
}
