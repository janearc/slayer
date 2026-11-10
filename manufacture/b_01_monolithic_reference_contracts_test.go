//go:build darwin && arm64

package manufacture

import (
	"bufio"
	"crypto/sha256"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/shape"
)

// The main catalogue uses segmented mirrors. Hold the authored monolithic
// shape separately, with the same material and geometry record format.
func TestAuthoredMonolithicMirrorAtEveryDetail(t *testing.T) {
	file, err := os.Open("testdata/monolithic-reference-arm64.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	spec := opticalSpec()
	spec.Segmented = false
	row, err := Observatory(spec)
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]bool{}
	scan := bufio.NewScanner(file)
	for scan.Scan() {
		columns := strings.Fields(scan.Text())
		if len(columns) != 2 || seen[columns[0]] {
			t.Fatal("invalid monolithic record")
		}
		level, expected := columns[0], columns[1]
		seen[level] = true
		made, err := MakeWithComplexity(row, Sow(5), level)
		if err != nil {
			t.Fatal(err)
		}
		things, err := made.Things(maths.Subspace{})
		if err != nil {
			t.Fatal(err)
		}
		h := sha256.New()
		bounds := made.Bounds()
		referenceVector(h, bounds.Min)
		referenceVector(h, bounds.Max)
		referenceWrite(h, int64(len(things)))
		shared := map[shape.Shape]int{}
		for _, thing := range things {
			referenceSpace(h, thing.At(0))
			referenceGeometry(h, thing.Shape, shared)
			material := thing.Material
			if material == nil {
				referenceWrite(h, byte(0))
				continue
			}
			referenceWrite(h, byte(1))
			referenceWrite(h, int64(len(material.Name)))
			if _, err := h.Write([]byte(material.Name)); err != nil {
				t.Fatal(err)
			}
			referenceWrite(h, [8]float64{material.Colour.R, material.Colour.G, material.Colour.B,
				material.Glow.R, material.Glow.G, material.Glow.B, material.Gloss, material.Index})
		}
		if got := fmt.Sprintf("%x", h.Sum(nil)); got != expected {
			t.Fatalf("%s authored monolithic geometry changed: %s, recorded %s", level, got, expected)
		}
	}
	if err := scan.Err(); err != nil {
		t.Fatal(err)
	}
	if len(seen) != 4 {
		t.Fatal("incomplete monolithic detail record")
	}
	for _, level := range Complexities() {
		if !seen[level] {
			t.Fatalf("missing %s record", level)
		}
	}
}
