//go:build draw && darwin

package gpu

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"math"
	"testing"

	"github.com/janearc/slayer/emit"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/render/frame"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
)

// crowd is a ground, two balls and lamps small lights over it, placed by
// a fixed generator so every run draws the same world. The same builder
// serves the exact-path digest and the restir tests.
func crowd(t *testing.T, lamps int) (*thing.World, maths.Subspace) {
	t.Helper()
	return crowdGain(t, lamps, 1)
}

// crowdGain is crowd with every lamp's strength times gain.
func crowdGain(t *testing.T, lamps int, gain float64) (*thing.World, maths.Subspace) {
	t.Helper()
	world := thing.NewWorld()
	grey := &thing.Material{Name: "floor", Colour: maths.Grey(0.6)}
	world.Add(thing.Thing{Shape: floor(t, 30), Material: grey})
	red := &thing.Material{Name: "ball", Colour: maths.RGB(0.7, 0.3, 0.2)}
	world.Add(thing.Thing{Shape: shape.Sphere{Radius: 0.7}, Material: red,
		Motion: at(0.8, 0.7, 0.2)})
	world.Add(thing.Thing{Shape: shape.Sphere{Radius: 1.2}, Material: red,
		Motion: at(1.2, 7, -2.4)})
	state := uint64(0x9e3779b97f4a7c15)
	unit := func() float64 {
		state = state*6364136223846793005 + 1442695040888963407
		return float64(state>>11) / float64(1<<53)
	}
	for range lamps {
		world.Lamp(emit.Lit{
			At: maths.V((unit()-0.5)*24, 0.3+unit()*5, (unit()-0.5)*24),
			Strength: maths.RGB(gain*(0.05+unit()*0.1), gain*(0.05+unit()*0.1),
				gain*(0.05+unit()*0.1))})
	}
	from := maths.V(0, 6, 22)
	return world, maths.Subspace{Origin: from,
		Orientation: maths.LookAt(from, maths.V(0, 3.5, 0))}
}

// digest is the sha256 of every pixel's three channels, as bits.
func digest(picture *frame.Frame, width, height int) string {
	sum := sha256.New()
	var word [4]byte
	for y := range height {
		for x := range width {
			p := picture.At(x, y)
			for _, v := range []float32{p.R, p.G, p.B} {
				binary.LittleEndian.PutUint32(word[:],
					math.Float32bits(v))
				sum.Write(word[:])
			}
		}
	}
	return hex.EncodeToString(sum.Sum(nil))
}
