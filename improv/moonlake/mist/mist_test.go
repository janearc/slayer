package mist

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func sample(mode int) Image {
	image := Image{Width: 7, Height: 80, Scale: -1.25, Pixels: make([]float32, 7*80*3)}
	for i := range image.Pixels {
		switch mode {
		case 0:
			image.Pixels[i] = float32(i%23+1) / 17
		case 1:
			image.Pixels[i] = float32(i%13+2) * 3
		case 2:
			if i/3%4 == 0 {
				image.Pixels[i] = float32(i%7+1) / 9
			}
		case 3:
			if i/3%11 == 0 {
				image.Pixels[i] = float32(i%5+1) / 13
			}
		}
	}
	return image
}

func configuration() Config {
	return Config{Top: 6, Bottom: -6, Shore: 0, Strength: 0.43, Seed: 23, Colour: [3]float64{0.78, 0.88, 1.08}, Layers: 1}
}

func TestApplyLegacyBytes(t *testing.T) {
	for _, all := range []bool{false, true} {
		image, config := sample(0), configuration()
		want := "9ee432a0cbd5cb9544695a0c387d0c606f01eed25f86b0034683565f2a2c6f39"
		if all {
			image.Scale = 2.5
			ref, water, trees := sample(1), sample(2), sample(3)
			config.MoonRef, config.WaterMask, config.TreeMask = &ref, &water, &trees
			config.Colour, config.Layers, config.Seed, config.Seconds = [3]float64{0.31, 1.13, 0.87}, 3.5, 7920, -7.5
			want = "132662bacc98e1610ff92d97c83db5240c6172d31b0b3d4893bee171e272e75d"
		}
		out, err := Apply(image, config)
		if err != nil {
			t.Fatal(err)
		}
		path := filepath.Join(t.TempDir(), "out.pfm")
		if err := Write(path, out); err != nil {
			t.Fatal(err)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if got := fmt.Sprintf("%x", sha256.Sum256(data)); got != want {
			t.Fatalf("hash %s, want %s", got, want)
		}
	}
}

func TestApplyOwnershipAndExplicitConfig(t *testing.T) {
	image, ref, water, trees := sample(0), sample(1), sample(2), sample(3)
	originals := make([][]float32, 4)
	for i, im := range []Image{image, ref, water, trees} {
		originals[i] = append([]float32(nil), im.Pixels...)
	}
	config := configuration()
	config.MoonRef, config.WaterMask, config.TreeMask = &ref, &water, &trees
	first, err := Apply(image, config)
	if err != nil {
		t.Fatal(err)
	}
	for key, value := range map[string]string{"MIST_COLOUR": "bad", "MIST_THICK": "bad", "MOONREF": "missing", "WATERMASK": "missing", "TREEMASK": "missing"} {
		t.Setenv(key, value)
	}
	second, err := Apply(image, config)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatal("not deterministic or reads environment")
	}
	for i, im := range []Image{image, ref, water, trees} {
		if !reflect.DeepEqual(im.Pixels, originals[i]) {
			t.Fatalf("input %d mutated", i)
		}
	}
	first.Pixels[0]++
	if reflect.DeepEqual(first, second) || !reflect.DeepEqual(image.Pixels, originals[0]) {
		t.Fatal("output aliases input or another output")
	}
	second.Pixels[0]++
	saved := second.Pixels[1]
	image.Pixels[1]++
	ref.Pixels[1]++
	water.Pixels[1]++
	trees.Pixels[1]++
	if second.Pixels[1] != saved {
		t.Fatal("output not owned")
	}
}

func TestApplyConfigurationEffects(t *testing.T) {
	image, config := sample(0), configuration()
	base, err := Apply(image, config)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name   string
		change func(*Config)
	}{
		{"seed", func(c *Config) { c.Seed++ }},
		{"seconds", func(c *Config) { c.Seconds = 8.25 }},
		{"layers", func(c *Config) { c.Layers = 4 }},
		{"colour", func(c *Config) { c.Colour = [3]float64{1, 0.5, 0.25} }},
		{"strength", func(c *Config) { c.Strength *= 2 }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			changed := config
			tc.change(&changed)
			out, err := Apply(image, changed)
			if err != nil {
				t.Fatal(err)
			}
			if reflect.DeepEqual(base.Pixels, out.Pixels) {
				t.Fatal("configuration ignored")
			}
		})
	}
}

func TestApplyValidation(t *testing.T) {
	for _, tc := range []struct {
		name   string
		change func(*Image, *Config)
	}{
		{"zero-width", func(i *Image, c *Config) { i.Width = 0 }},
		{"negative-height", func(i *Image, c *Config) { i.Height = -1 }},
		{"overflow", func(i *Image, c *Config) { i.Width = int(^uint(0) >> 1) }},
		{"too-large", func(i *Image, c *Config) { i.Width, i.Height = MaxPixels+1, 1 }},
		{"short", func(i *Image, c *Config) { i.Pixels = i.Pixels[:3] }},
		{"long", func(i *Image, c *Config) { i.Pixels = append(i.Pixels, 0) }},
		{"scale", func(i *Image, c *Config) { i.Scale = math.Inf(1) }},
		{"span", func(i *Image, c *Config) { c.Bottom = c.Top }},
		{"reverse", func(i *Image, c *Config) { c.Bottom = c.Top + 1 }},
		{"shore-bottom", func(i *Image, c *Config) { c.Shore = c.Bottom }},
		{"shore-top", func(i *Image, c *Config) { c.Shore = c.Top }},
		{"empty-sky", func(i *Image, c *Config) { c.Shore = c.Top - 0.001 }},
		{"layers", func(i *Image, c *Config) { c.Layers = 0.5 }},
		{"nonfinite", func(i *Image, c *Config) { c.Seconds = math.NaN() }},
		{"colour", func(i *Image, c *Config) { c.Colour[1] = math.Inf(-1) }},
		{"reference", func(i *Image, c *Config) { r := sample(1); r.Width--; c.MoonRef = &r }},
		{"water", func(i *Image, c *Config) { r := sample(2); r.Height--; c.WaterMask = &r }},
		{"trees", func(i *Image, c *Config) { r := sample(3); r.Pixels = nil; c.TreeMask = &r }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			image, config := sample(0), configuration()
			tc.change(&image, &config)
			out, err := Apply(image, config)
			if err == nil || len(out.Pixels) != 0 {
				t.Fatalf("accepted invalid input: %v", err)
			}
		})
	}
}

func TestPFMRoundTripOwnedBottomUp(t *testing.T) {
	for _, scale := range []float64{-1.25, 2.5, 0} {
		path := filepath.Join(t.TempDir(), "test.pfm")
		image := Image{Width: 1, Height: 2, Scale: scale, Pixels: []float32{1, 2, 3, 4, 5, 6}}
		if err := Write(path, image); err != nil {
			t.Fatal(err)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		var expected bytes.Buffer
		fmt.Fprintf(&expected, "PF\n1 2\n%g\n", scale)
		var endian binary.ByteOrder = binary.LittleEndian
		if scale > 0 {
			endian = binary.BigEndian
		}
		if err := binary.Write(&expected, endian, image.Pixels); err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(data, expected.Bytes()) {
			t.Fatal("changed row order, scale, or float32 encoding")
		}
		got, err := Read(path)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(image, got) {
			t.Fatalf("round trip: %#v", got)
		}
		again, err := Read(path)
		if err != nil {
			t.Fatal(err)
		}
		got.Pixels[0]++
		if again.Pixels[0] != 1 {
			t.Fatal("reads share pixel storage")
		}
	}
}

func TestPFMValidation(t *testing.T) {
	for _, text := range []string{
		"", "PF\n", "Pf\n1 1\n-1\n", "PF\n-1 1\n-1\n", "PF\n0 1\n-1\n", "PF\n16777217 1\n-1\n", "PF\n9223372036854775807 2\n-1\n", "PF\n1 1\nNaN\n", "PF\n1 1\n+Inf\n", "PF\n1 1\n-1\n", "PF\n1 1\n-1\nx", "PF\n4096 4096\n-1\nx",
	} {
		path := filepath.Join(t.TempDir(), "bad.pfm")
		if err := os.WriteFile(path, []byte(text), 0600); err != nil {
			t.Fatal(err)
		}
		if _, err := Read(path); err == nil {
			t.Fatalf("accepted %q", text)
		}
	}
	path := filepath.Join(t.TempDir(), "invalid.pfm")
	if err := Write(path, Image{}); err == nil {
		t.Fatal("accepted invalid image")
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("invalid write created file")
	}
	if _, err := Read(path); err == nil {
		t.Fatal("missing input accepted")
	}
	if err := Write(t.TempDir(), sample(0)); err == nil {
		t.Fatal("directory output accepted")
	}
}

func TestLegacyZeroMasks(t *testing.T) {
	image, config := sample(0), configuration()
	zero := sample(4)
	config.TreeMask = &zero
	without, err := Apply(image, configuration())
	if err != nil {
		t.Fatal(err)
	}
	with, err := Apply(image, config)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(without, with) {
		t.Fatal("zero trees changed mist")
	}
	config.WaterMask = &zero
	out, err := Apply(image, config)
	if err != nil {
		t.Fatal(err)
	}
	if !math.IsNaN(float64(out.Pixels[0])) {
		t.Fatal("zero-water legacy float32 division changed")
	}
}
