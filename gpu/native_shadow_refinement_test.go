//go:build darwin

//ninja:vow dojo fault injection needs the private Metal source and encoder
package gpu

import (
	_ "embed"
	"strings"
	"testing"
	"time"

	"github.com/janearc/puzzlemouth/yamlfile"
	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/metal"
	"github.com/janearc/slayer/render/frame"
	"github.com/janearc/slayer/scene/thing"
)

//go:embed testdata/native-shadow-refinement.yaml
var shadowRefusalRaw []byte

// shadowFirst and shadowNext are defined byte positions in a function declaration.
const shadowFirst, shadowNext = 0, 1

// shadowEpoch is the defined stationary scene time.
const shadowEpoch time.Duration = 0

type shadowRefusalPlan struct {
	Width, Height, Samples int
	Wide                   float64
}

// Reject actual native candidates, then require the other blockers to remain.
// The hall includes clear and opaque surfaces and a baked volume.
func TestNativeShadowCandidateRejectionStillChecksOtherOccluders(t *testing.T) {
	var data shadowRefusalPlan
	if err := yamlfile.Read("native-shadow-refinement.yaml",
		shadowRefusalRaw, &data); err != nil {
		t.Fatal(err)
	}
	world, eye := hall(t)
	lens, err := film.PinholeOf(data.Wide)
	if err != nil {
		t.Fatal(err)
	}
	picture := data.draw(t, world, eye, lens)
	truth, err := Sampled(world, eye, lens, shadowEpoch,
		data.Width, data.Height, data.Samples)
	if err != nil {
		t.Fatal(err)
	}
	for index, pixel := range truth.Pix {
		if pixel != picture.Pix[index] {
			t.Fatalf("rejected shadow candidate lost another blocker at %d", index)
		}
	}
}

// draw sends the fault-injected source through actual Metal traversal.
func (data shadowRefusalPlan) draw(t *testing.T, world *thing.World,
	eye maths.Subspace, lens *film.Pinhole,
) *frame.Frame {
	t.Helper()
	encoded, err := encodeGrown(world, nil, eye, lens, shadowEpoch,
		data.Width, data.Height)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := encoded.bakeAll(); err != nil {
		t.Fatal(err)
	}
	kernel := refusedShadowKernel(t, encoded)
	defer kernel.Close()
	samples, err := kernel.Grid(data.Width, data.Height, rows,
		encoded.perSample(), encoded.inputs())
	if err != nil {
		t.Fatal(err)
	}
	return averaged(finishing{world: world, records: encoded.records(),
		paints: encoded.paints,
		wide:   wideOf(world, eye, lens, data.Width, data.Height)},
		samples, data.Width, data.Height, data.Samples)
}

// refusedShadowKernel keeps actual hardware traversal but refuses its candidates.
func refusedShadowKernel(t *testing.T, encoded *scene) *metal.Kernel {
	t.Helper()
	text := rejectedShadowSource(t, encoded)
	kernel, err := metal.Open("#define SLAYER_NATIVE_SHADOWS 1\n"+text, "trace")
	if err != nil {
		t.Fatal(err)
	}
	plan, root, err := encoded.accelerationPlan()
	if err != nil {
		kernel.Close()
		t.Fatal(err)
	}
	if err = kernel.Accelerate(plan, root); err != nil {
		kernel.Close()
		t.Fatal(err)
	}
	return kernel
}

// rejectedShadowSource injects only a candidate refusal, not an empty scene.
func rejectedShadowSource(t *testing.T, encoded *scene) string {
	t.Helper()
	text := sourceWith(encoded.records())
	function := strings.Index(text, "static Hit native_shadow_standing(")
	if function < shadowFirst {
		t.Fatal("native shadow refinement missing")
	}
	brace := strings.Index(text[function:], "{") + function
	if brace < function {
		t.Fatal("native shadow refinement body missing")
	}
	return text[:brace+shadowNext] + "\nreturn {far, -1, float3(0)};\n" +
		text[brace+shadowNext:]
}
