//go:build darwin

package gpu

import (
	"encoding/binary"
	"math"
	"slices"
	"testing"
	"time"

	"github.com/janearc/slayer/maths"
)

func TestFlamePreparationDefersNoiseAndPreservesOtherGrids(t *testing.T) {
	encoded := newScene()
	encoded.boxes.floats([]float32{1.5, 2.5})
	grid := fireGridFixture(t, 13, 210*time.Millisecond)
	encoded.boxedVolume(maths.Subspace{}, grid, 0)
	if len(encoded.flames) != 1 {
		t.Fatal("flame was not deferred to GPU")
	}
	first := encoded.flames[0]
	if first.density != 2 || first.shine != 2+grid.count() {
		t.Fatal("wrong reserved offsets")
	}
	tail := encoded.boxes.floats([]float32{3.5, 4.5})
	if len(encoded.boxes.bytes) != (4+grid.count()*4)*4 {
		t.Fatal("wrong grid reservation")
	}
	for _, b := range encoded.boxes.bytes[8 : tail*4] {
		if b != 0 {
			t.Fatal("flame was sampled before the GPU bake")
		}
	}
	elapsed, err := encoded.bakeAll()
	if err != nil {
		t.Fatal(err)
	}
	if elapsed <= 0 {
		t.Fatal("flame bake did no GPU work")
	}
	number := func(i int) float32 {
		return math.Float32frombits(binary.LittleEndian.Uint32(encoded.boxes.bytes[4*i:]))
	}
	if number(0) != 1.5 || number(1) != 2.5 || number(tail) != 3.5 || number(tail+1) != 4.5 {
		t.Fatal("flame bake overwrote a neighboring grid")
	}
	density, glow, _ := grid.asked()
	total, apart, light, errorLight := 0.0, 0.0, 0.0, 0.0
	for i, d := range density {
		got := number(first.density + i)
		total += float64(d)
		apart += math.Abs(float64(d - got))
		for c := range 3 {
			expected := float64(d) * float64(glow[3*i+c])
			actual := float64(got) * float64(number(first.shine+3*i+c))
			light += expected
			errorLight += math.Abs(expected - actual)
		}
	}
	if total <= 0 || light <= 0 || apart/total > 1e-4 || errorLight/light > 1e-4 {
		t.Fatal("reserved flame grid differs from the CPU reference")
	}
}

func TestCustomFlameEmissionKeepsItsCPUPreparation(t *testing.T) {
	grid := fireGridFixture(t, 5, 0)
	grid.shine = banded{}
	encoded := newScene()
	encoded.boxedVolume(maths.Subspace{}, grid, 0)
	if len(encoded.flames) != 0 {
		t.Fatal("custom emission took the flame kernel")
	}
	density, glow, _ := grid.asked()
	for i, d := range density {
		actual := math.Float32frombits(binary.LittleEndian.Uint32(encoded.boxes.bytes[i*4:]))
		if actual != d {
			t.Fatal("custom field changed")
		}
	}
	start := len(density) * 4
	for i, light := range glow {
		actual := math.Float32frombits(binary.LittleEndian.Uint32(encoded.boxes.bytes[start+i*4:]))
		if actual != light {
			t.Fatal("custom emission changed")
		}
	}
}

func TestAnEmptyFlameBakeLeavesEncodedDataAlone(t *testing.T) {
	encoded := newScene()
	encoded.boxes.floats([]float32{1, 2, 3})
	before := slices.Clone(encoded.boxes.bytes)
	elapsed, err := encoded.bakeAll()
	if err != nil || elapsed != 0 || !slices.Equal(before, encoded.boxes.bytes) {
		t.Fatal("empty bake changed scene data")
	}
}
