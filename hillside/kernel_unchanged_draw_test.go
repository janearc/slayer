//go:build draw && darwin

package hillside

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/janearc/slayer/render/frame"
)

// A job's frames on the gpu with no shutter and no aperture are the ones
// drawn before the kernel could open either, byte for byte: the night
// fire, with its trees, sparks, flames and cached light, and the orbital
// from far, whose lantern has beams.
//
// The hashes were recorded from commit 15c2fe4c, the last before
// gpu/exposure.go, by running this test there with
// SLAYER_KERNEL_HASHES=print. A change meant to move these pictures
// records them again the same way and says so.
var jobsUnchanged = map[string]string{
	"the night fire":      "63324d501eb0401de3c451495d2c8187fdc7b55211c72cd74c81b550f64e74e0",
	"the orbital's beams": "0e2432c8d53fa3ceca4113efd05197aa7d0772d9f722903c6687891381412739",
}

// lightHash is the sha256 of a frame's pixels as bits.
func lightHash(picture *frame.Frame) string {
	sum := sha256.New()
	var word [4]byte
	for _, pixel := range picture.Pix {
		for _, part := range []float32{pixel.R, pixel.G, pixel.B,
			pixel.Cover, pixel.Depth} {
			binary.LittleEndian.PutUint32(word[:], math.Float32bits(part))
			sum.Write(word[:])
		}
	}
	return hex.EncodeToString(sum.Sum(nil))
}

// each jobs first frame is hashed and held to what the base commit drew
func TestClosedJobsDrawWhatTheyDrewBeforeTheExposure(t *testing.T) {
	fire, err := filepath.Abs("../doc/fire-night.yaml")
	if err != nil {
		t.Fatal(err)
	}
	film := "fps: 1\nduration: {value: 1, unit: s}\n"
	for name, document := range map[string]string{
		"the night fire": fmt.Sprintf("scene: {set: %s}\nfilm: {width: "+
			"96, height: 160, rays: 2}\nstart: {value: 26, unit: s}\n",
			fire) + film,
		"the orbital's beams": "scene: {named: orbital, with: {far: " +
			"true}}\nfilm: {width: 96, height: 54, rays: 2}\nrenderer: " +
			"gpu\n" + film,
	} {
		drawn := drawnOrSkipped(t, jobFrom(t, document))
		if drawn.Profile().Device == "" {
			t.Fatalf("%s was drawn on the cpu", name)
		}
		got := lightHash(drawn.Light())
		if os.Getenv("SLAYER_KERNEL_HASHES") == "print" {
			t.Logf("\t%q: %q,", name, got)
			continue
		}
		if want, known := jobsUnchanged[name]; !known || got != want {
			t.Errorf("%s: the frame's hash is %s, and before the "+
				"exposure it was %q", name, got, want)
		}
	}
}
