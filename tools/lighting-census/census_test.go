package census

import (
	"math"
	"path/filepath"
	"testing"

	"github.com/janearc/slayer/render/frame"
)

func TestMatrixNamesEverySceneModeAndLeavesItPending(t *testing.T) {
	m := Matrix()
	if len(m) != 52*7 {
		t.Fatalf("matrix has %d cases, want 52 scenes times seven modes", len(m))
	}
	seen := map[string]bool{}
	for _, c := range m {
		key := c.Key()
		if seen[key] || c.Status != "not-run" || c.Samples < 1 {
			t.Fatalf("bad or duplicate case: %+v", c)
		}
		seen[key] = true
	}
}

func TestDifferenceDistinguishesErrorsAndDarkReferences(t *testing.T) {
	a, b := frame.New(2, 1), frame.New(2, 1)
	a.Pix[0] = frame.Pixel{R: 2, G: 1, B: 1, Cover: 1, Depth: 4}
	b.Pix[0] = frame.Pixel{R: 1, G: 1, B: 1, Cover: 0, Depth: 5}
	d, err := Compare(a, b)
	if err != nil || d.Absolute != 1 || d.Relative == nil || *d.Relative != .25 || d.CoverDifferent != 1 || d.DepthDifferent != 1 {
		t.Fatalf("difference = %+v, %v", d, err)
	}
	dark, err := Compare(frame.New(1, 1), frame.New(1, 1))
	if err != nil || dark.Relative != nil {
		t.Fatalf("dark reference must have no relative ratio: %+v, %v", dark, err)
	}
	b.Pix[0].R = float32(math.NaN())
	if _, err := Compare(a, b); err == nil {
		t.Fatal("a nonfinite image was accepted")
	}
	if _, err := Compare(a, frame.New(1, 1)); err == nil {
		t.Fatal("different dimensions were accepted")
	}
}

func TestOutputDirectoryNeverReusesExistingData(t *testing.T) {
	base := t.TempDir()
	path := filepath.Join(base, "new")
	if err := CreateOutput(path); err != nil {
		t.Fatal(err)
	}
	if err := CreateOutput(path); err == nil {
		t.Fatal("existing output directory accepted")
	}
	if err := CreateOutput("relative"); err == nil {
		t.Fatal("relative output directory accepted")
	}
}

func TestFrameReservationIncludesTheSharedDiskFloor(t *testing.T) {
	need, err := RequiredFree(1074, 600)
	if err != nil || need <= MinFreeBytes+1074*600*20 {
		t.Fatalf("reservation %d: %v", need, err)
	}
	if _, err := RequiredFree(0, 1); err == nil {
		t.Fatal("zero width accepted")
	}
	if _, err := RequiredFree(math.MaxInt, math.MaxInt); err == nil {
		t.Fatal("overflow accepted")
	}
}
