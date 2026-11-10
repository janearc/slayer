//go:build darwin

package gpu

import (
	"bytes"
	"testing"

	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/plume"
	"github.com/janearc/slayer/scene/thing"
)

// The documented production choice is 64 cells along the longest side.
// This holds its authored work/storage budget, not a physical constant.
func TestDefaultLightingGridKeepsItsDocumentedWorkBudget(t *testing.T) {
	for _, spec := range []struct {
		base, top  float64
		tall, thin bool
	}{
		{2.4, 12, false, false},
		{2.4, 100, true, false},
		{10000, 12, false, true},
	} {
		column, err := plume.New(plume.Fire{Heat: 4000, Convective: 0.71,
			Base: spec.base, Yield: 0.0024, Combustion: 21, Extinction: 8,
			Air: 288, Density: 1.2, Top: spec.top, Seed: 5})
		if err != nil {
			t.Fatal(err)
		}
		bounds := column.Bounds()
		size := bounds.Max.Sub(bounds.Min)
		if spec.tall && !(size.Y > size.X && size.Y > size.Z) {
			t.Fatal("tall plume fixture lost its distinct longest axis")
		}
		if spec.thin && !(size.Y < size.X/64) {
			t.Fatal("flat plume fixture lost its thin axis")
		}
		world := thing.NewWorld()
		world.AddVolume(thing.Volume{Medium: column,
			Material: &thing.Material{Colour: maths.Grey(0.3)}})
		encoded, err := encode(world, maths.Subspace{}, film.Pinhole{}, 0, 4, 3)
		if err != nil {
			t.Fatal(err)
		}
		if len(encoded.smokes) != 1 {
			t.Fatal("fixture lost its cacheable smoke")
		}
		prefix := append([]byte(nil), encoded.boxes.bytes...)
		encoded.reserveLighting()
		for _, job := range encoded.smokes {
			longest := 0
			for axis, cells := range job.grid.cells {
				if cells < 2 || cells > 64 {
					t.Fatalf("default lighting axis %d exceeds its documented budget: %d", axis, cells)
				}
				longest = max(longest, cells)
			}
			if longest != 64 {
				t.Fatalf("default longest axis lost its documented resolution: %d", longest)
			}
			if spec.thin && job.grid.cells[1] != 2 {
				t.Fatalf("thin axis needs only the trilinear minimum: %d planes", job.grid.cells[1])
			}
		}
		// Three float32 source channels over at most 64 cubed cells are 3 MiB.
		if added := len(encoded.boxes.bytes) - len(prefix); added <= 0 || added > 3*1024*1024 {
			t.Fatalf("one default lighting grid reserved %d bytes", added)
		}
		if !bytes.Equal(encoded.boxes.bytes[:len(prefix)], prefix) {
			t.Fatal("lighting reservation changed the retained density grid")
		}
	}
}
