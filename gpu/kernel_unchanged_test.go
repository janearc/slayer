//go:build draw && darwin

package gpu

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"math"
	"os"
	"testing"

	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/render/frame"
)

// The kernel's pictures with the shutter closed and no aperture are the
// ones it drew before it could open either, byte for byte.
//
// The hashes below were recorded from the kernel at commit 15c2fe4c, the
// last before gpu/exposure.go, by running this test there with
// SLAYER_KERNEL_HASHES=print. There is one kernel in the tree; the old
// one is held here only as what it drew. A change that is meant to move
// these pictures records them again the same way and says so.
//
// Each is the sha256 of every pixel's light, cover and depth as bits, in
// row order, of a picture 96 by 64 at two rays a side.
var closedHashes = map[string]string{
	"a lens given as rays":   "5fdfc72066f41879ff314f904fb87f0eab7b10f1fa2514a78c1000ef922964e1",
	"native meshes":          "8c80b3da3514ccebf46901e002be4c864bbad2a6be27dc2db4836413781696e9",
	"native shadows":         "664557233fd2075f47e4208d391fe423d10f74b0708e4518ac7ac3e39fc225c4",
	"software":               "e7a8f11e92ee8075ef01232c9138f0beed692096690850b0ae6a5f4d280755ca",
	"spheres alone":          "0c7cb96a383a2ab53cd97812c4dad94eab50e87028ffd6452583e6bcf11770b0",
	"the fire, cached light": "5ced8345f311291bcdbaefc3dc51a322fdd046fe5075bc51febf5a152c423abf",
}

// printHashes is the variable that asks the test to say the hashes it
// finds and compare nothing.
const printHashes = "SLAYER_KERNEL_HASHES"

// givenAsRays is a lens whose rays the cpu makes and sends, whatever it
// wraps: the kernel knows no lens by this type.
type givenAsRays struct{ film.Sensor }

// frameHash is the sha256 of a frame's pixels as bits.
func frameHash(picture *frame.Frame) string {
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

// heldTo fails unless a picture's hash is the one recorded for its name,
// or says the hash where the test is asked to print.
func heldTo(t *testing.T, recorded map[string]string, name string,
	picture *frame.Frame) {
	t.Helper()
	got := frameHash(picture)
	if os.Getenv(printHashes) == "print" {
		t.Logf("\t%q: %q,", name, got)
		return
	}
	if want, known := recorded[name]; !known || got != want {
		t.Errorf("%s: the picture's hash is %s, and the kernel before "+
			"the exposure drew %q", name, got, want)
	}
}

// the hall, with its mesh, its mirror, its clear things and its two
// volumes, in each traversal; the fire with its smoke's cached light;
// spheres alone; and the hall through a lens sent as rays
func TestTheClosedKernelDrawsWhatItDrewBeforeTheExposure(t *testing.T) {
	const width, height, side = 96, 64, 2
	names := map[traversal]string{softwareTraversal: "software",
		nativeShadowTraversal: "native shadows",
		nativeTraversal:       "native meshes",
		cachedTraversal:       "the fire, cached light"}
	for _, mode := range traversalsChecked {
		if _, pinned := names[mode]; !pinned {
			continue // a mode added after 15c2fe4c has no picture from before
		}
		world, eye := sceneFor(t, mode)
		picture, _, err := perCalls[mode](world, eye, shotLens, frameStep,
			width, height, side)
		mustDevice(t, err)
		heldTo(t, closedHashes, names[mode], picture)
	}
	world, eye := orrery()
	picture, err := Sampled(world, eye, shotLens, 0, width, height, side)
	mustDevice(t, err)
	heldTo(t, closedHashes, "spheres alone", picture)
	world, eye = hall(t)
	picture, err = Sampled(world, eye, givenAsRays{thinLens(t)}, 0, width,
		height, side)
	mustDevice(t, err)
	heldTo(t, closedHashes, "a lens given as rays", picture)
}

// movedHashes are pictures the exposure moved on purpose: a thin lens
// with the shutter closed, whose rays the kernel now makes where the cpu
// made and sent them, within the exact tolerance of the old and not byte
// for byte (2026-11-12). They
// are held here as drawn since, so a later change to them is seen. The
// cpu tracer, which the proof path replays, is unchanged.
var movedHashes = map[string]string{
	"a thin lens in the kernel": "8ca715b63e7c49717093d87a19ab47e2e8a69e98b12a1d3f46a2500a22c84308",
}

func TestAClosedThinLensDrawsWhatItDrewWhenItMoved(t *testing.T) {
	const width, height, side = 96, 64, 2
	world, eye := hall(t)
	picture, err := Sampled(world, eye, thinLens(t), 0, width, height,
		side)
	mustDevice(t, err)
	heldTo(t, movedHashes, "a thin lens in the kernel", picture)
}
