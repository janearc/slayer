package matte

import (
	"errors"
	"io"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestOwnedInputsAndOutput(t *testing.T) {
	raw := Image{Width: 1, Height: 1, Scale: -2, Pixels: []float32{3, 6, 9}}
	sea := Image{Width: 1, Height: 1, Scale: 1, Pixels: []float32{1, 2, 3}}
	ref := Image{Width: 1, Height: 1, Pixels: []float32{9, 12, 15}}
	mask := Image{Width: 1, Height: 1, Pixels: []float32{1, 1, 1}}
	front := Image{Width: 1, Height: 1, Pixels: []float32{0.25, 0.5, 0.75}}
	cfg := Config{Top: 6, Bottom: -6, Shore: 0, MoonRadius: 0.27, MoonRef: &ref, WaterMask: &mask, Front: &front}
	before := [][]float32{append([]float32(nil), raw.Pixels...), append([]float32(nil), sea.Pixels...), append([]float32(nil), ref.Pixels...), append([]float32(nil), mask.Pixels...), append([]float32(nil), front.Pixels...)}
	out, err := Apply(raw, sea, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if out.Width != 1 || out.Height != 1 || out.Scale != -2 {
		t.Fatalf("metadata: %+v", out)
	}
	disc := math.Pi * math.Pow(cfg.MoonRadius*math.Pi/180, 2)
	for c := range 3 {
		want := float32(float64(sea.Pixels[c])*12*disc) + front.Pixels[c]
		if out.Pixels[c] != want {
			t.Fatalf("channel %d: %g, want %g", c, out.Pixels[c], want)
		}
	}
	inputs := []Image{raw, sea, ref, mask, front}
	for i, input := range inputs {
		if !reflect.DeepEqual(input.Pixels, before[i]) {
			t.Fatalf("input %d mutated", i)
		}
	}
	out.Pixels[0] = 100
	for i, input := range inputs {
		if !reflect.DeepEqual(input.Pixels, before[i]) {
			t.Fatalf("output aliases input %d", i)
		}
	}
	second, err := Apply(raw, sea, cfg)
	if err != nil || second.Pixels[0] == 100 {
		t.Fatalf("repeated apply: %+v, %v", second, err)
	}
}

func TestReadBounds(t *testing.T) {
	for _, tc := range []struct {
		name, data string
		want       error
	}{
		{"huge", "PF\n2147483647 2147483647\n-1\n", nil},
		{"ceiling", "PF\n16777217 1\n-1\n", nil},
		{"negative", "PF\n-1 1\n-1\n", nil},
		{"empty", "PF\n1 1\n-1\n", io.EOF},
		{"short", "PF\n1 1\n-1\nx", io.ErrUnexpectedEOF},
		{"large-short", "PF\n4096 4096\n-1\nx", io.ErrUnexpectedEOF},
		{"nan-scale", "PF\n1 1\nNaN\nabcdefghijkl", nil},
		{"inf-scale", "PF\n1 1\n+Inf\nabcdefghijkl", nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "input.pfm")
			if err := os.WriteFile(path, []byte(tc.data), 0600); err != nil {
				t.Fatal(err)
			}
			if _, err := Read(path); err == nil || tc.want != nil && !errors.Is(err, tc.want) {
				t.Fatalf("Read error %v, want %v", err, tc.want)
			}
		})
	}
}

func TestNonfiniteInputs(t *testing.T) {
	good := Image{Width: 1, Height: 1, Scale: -1, Pixels: []float32{1, 2, 3}}
	for _, value := range []float64{math.NaN(), math.Inf(1), math.Inf(-1)} {
		for _, field := range []string{"top", "bottom", "shore", "radius", "scale", "raw", "sea", "reference", "mask", "front"} {
			raw, sea := good, good
			bad := Image{Width: 1, Height: 1, Scale: -1, Pixels: []float32{float32(value), 2, 3}}
			cfg := Config{Top: 6, Bottom: -6, MoonRadius: 0.27}
			switch field {
			case "top":
				cfg.Top = value
			case "bottom":
				cfg.Bottom = value
			case "shore":
				cfg.Shore = value
			case "radius":
				cfg.MoonRadius = value
			case "scale":
				raw.Scale = value
			case "raw":
				raw = bad
			case "sea":
				sea = bad
			case "reference":
				cfg.MoonRef = &bad
			case "mask":
				cfg.WaterMask = &bad
			case "front":
				cfg.Front = &bad
			}
			if _, err := Apply(raw, sea, cfg); err == nil {
				t.Fatalf("accepted nonfinite %s", field)
			}
		}
	}
}

func TestWriteValidation(t *testing.T) {
	path := filepath.Join(t.TempDir(), "out.pfm")
	if err := Write(path, Image{}); err == nil {
		t.Fatal("accepted invalid image")
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("invalid write created output: %v", err)
	}
}

func TestInvalidInputs(t *testing.T) {
	good := Image{Width: 1, Height: 1, Pixels: []float32{1, 2, 3}}
	bad := Image{Width: 2, Height: 1, Pixels: []float32{1, 2, 3}}
	wide := Image{Width: 2, Height: 1, Pixels: make([]float32, 6)}
	for _, tc := range []struct {
		raw, sea Image
		config   Config
	}{
		{bad, good, Config{}},
		{good, bad, Config{}},
		{good, wide, Config{}},
		{good, good, Config{MoonRef: &bad}},
		{good, good, Config{WaterMask: &wide}},
		{good, good, Config{Front: &wide}},
	} {
		if _, err := Apply(tc.raw, tc.sea, tc.config); err == nil {
			t.Fatal("invalid input accepted")
		}
	}
}
