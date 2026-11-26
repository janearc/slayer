package main

import (
	"bytes"
	"crypto/sha256"
	"testing"

	"github.com/janearc/slayer/scene/shape"
)

func TestSphereRecordKeepsRadius(t *testing.T) {
	// Fresh shape marker -1, sphere tag 4, IEEE-754 radius 1.0, little endian.
	want := sha256.Sum256([]byte{
		0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
		0x04, 0, 0, 0, 0, 0, 0, 0xf0, 0x3f,
	})
	record := func(radius float64) []byte {
		h := sha256.New()
		geometry(h, shape.Sphere{Radius: radius}, map[shape.Shape]int{})
		return h.Sum(nil)
	}
	if got := record(1); !bytes.Equal(got, want[:]) {
		t.Fatalf("radius 1 record %x, want %x", got, want)
	}
	if bytes.Equal(record(1), record(2)) {
		t.Fatal("different sphere radii produced the same record")
	}
}

func TestTubeRecordKeepsAllDimensions(t *testing.T) {
	// Fresh shape marker, tube tag, then radius 1, thickness 2, height 3.
	want := sha256.Sum256([]byte{
		0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0x05,
		0, 0, 0, 0, 0, 0, 0xf0, 0x3f,
		0, 0, 0, 0, 0, 0, 0, 0x40,
		0, 0, 0, 0, 0, 0, 0x08, 0x40,
	})
	record := func(tube shape.Tube) []byte {
		h := sha256.New()
		geometry(h, tube, map[shape.Shape]int{})
		return h.Sum(nil)
	}
	base := shape.Tube{Radius: 1, Thick: 2, Height: 3}
	if got := record(base); !bytes.Equal(got, want[:]) {
		t.Fatalf("tube record %x, want %x", got, want)
	}
	for _, changed := range []shape.Tube{
		{Radius: 4, Thick: 2, Height: 3},
		{Radius: 1, Thick: 4, Height: 3},
		{Radius: 1, Thick: 2, Height: 4},
	} {
		if bytes.Equal(record(base), record(changed)) {
			t.Fatalf("different tube dimensions produced the same record: %+v", changed)
		}
	}
}
