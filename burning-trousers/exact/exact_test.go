// Package exact holds the tracer to frames it drew before a change, to
// the bit. An exact shortcut, such as a tree over the things, gives the
// same bytes as none; anything else is a bug. The record is drawn once
// with -record, by the tracer as it was, and kept in testdata.
package exact_test

import (
	"bytes"
	"encoding/binary"
	"flag"
	"fmt"
	"math"
	"os"
	"testing"

	"github.com/janearc/slayer/burning-trousers/divergence"
	"github.com/janearc/slayer/burning-trousers/fake"
	"github.com/janearc/slayer/burning-trousers/scenes"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/render/frame"
	"github.com/janearc/slayer/trace"
)

// record is whether to draw the frames again and keep them as the
// record, in place of holding the tracer to it.
var record = flag.Bool("record", false,
	"draw the frames again and keep them as the record")

// recorded is where the record is kept.
const recorded = "testdata/frames.bin"

// held is every scene the record holds: torture's two hundred seeds lit
// and unlit, and crowds of 16 to 1024 balls under a grid of looks.
func held() []scenes.Scene {
	var all []scenes.Scene
	for seed := range uint64(200) {
		all = append(all, scenes.Arranged(seed, false),
			scenes.Arranged(seed, true))
	}
	for _, many := range []int{16, 64, 256, 1024} {
		all = append(all, scenes.Crowd(uint64(many), many, 32))
	}
	return all
}

// drawn is a scene drawn by the tracer, a pixel for each look.
func drawn(t *testing.T, made scenes.Scene) []frame.Pixel {
	t.Helper()
	rays := made.Rays(1)
	picture, err := trace.Render(made.World(1), maths.Subspace{},
		fake.NewAimed(rays...), 0, len(rays), 1)
	if err != nil {
		t.Fatal(err)
	}
	return picture.Pix
}

// encoded is a frame's pixels as bytes: every value's bits, so two
// frames are the same exactly when their bytes are.
func encoded(t *testing.T, pixels []frame.Pixel) []byte {
	t.Helper()
	var kept bytes.Buffer
	if err := binary.Write(&kept, binary.LittleEndian, pixels); err != nil {
		t.Fatal(err)
	}
	return kept.Bytes()
}

// The tracer draws every held scene as the record has it, bit for bit.
func TestTheTracerDrawsWhatWasRecorded(t *testing.T) {
	var now []byte
	var frames [][]frame.Pixel
	for _, made := range held() {
		pixels := drawn(t, made)
		frames = append(frames, pixels)
		now = append(now, encoded(t, pixels)...)
	}
	if *record {
		if err := os.WriteFile(recorded, now, 0o644); err != nil {
			t.Fatal(err)
		}
		t.Logf("recorded %d frames, %d bytes", len(frames), len(now))
		return
	}
	kept, err := os.ReadFile(recorded)
	if err != nil {
		t.Fatalf("no record: %v; draw one with -record", err)
	}
	if !bytes.Equal(now, kept) {
		t.Error(unlike(t, frames, kept))
	}
}

// unlike says how the frames drawn now differ from the record: how many
// pixels, in how many frames, and the farthest apart.
func unlike(t *testing.T, frames [][]frame.Pixel, kept []byte) string {
	t.Helper()
	var was []frame.Pixel
	count := 0
	for _, pixels := range frames {
		count += len(pixels)
	}
	was = make([]frame.Pixel, count)
	reader := bytes.NewReader(kept)
	if err := binary.Read(reader, binary.LittleEndian, was); err != nil {
		return "the record is not the frames' size: " + err.Error()
	}
	moved, framesMoved := 0, 0
	var worst divergence.JNDs
	at := 0
	for _, pixels := range frames {
		before := moved
		for _, pixel := range pixels {
			if !same(pixel, was[at]) {
				moved++
				worst = max(worst, divergence.Apart(pixel.Light(),
					was[at].Light()))
			}
			at++
		}
		if moved > before {
			framesMoved++
		}
	}
	return describe(moved, framesMoved, len(frames), worst)
}

// describe is a difference from the record in a sentence.
func describe(moved, framesMoved, frames int, worst divergence.JNDs) string {
	return fmt.Sprintf("%d pixels in %d of %d frames are not as recorded; "+
		"the farthest is %.3g JND", moved, framesMoved, frames, worst)
}

// same is whether two pixels are the same bit for bit, a NaN alike.
func same(one, other frame.Pixel) bool {
	return math.Float32bits(one.R) == math.Float32bits(other.R) &&
		math.Float32bits(one.G) == math.Float32bits(other.G) &&
		math.Float32bits(one.B) == math.Float32bits(other.B) &&
		math.Float32bits(one.Cover) == math.Float32bits(other.Cover) &&
		math.Float32bits(one.Depth) == math.Float32bits(other.Depth)
}
