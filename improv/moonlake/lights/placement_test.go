package lights

import (
	"math"
	"reflect"
	"slices"
	"testing"
)

func TestOverheadExplicitInputsAndOriginalMath(t *testing.T) {
	t.Setenv("OVERHEAD_GAIN", "bad")
	t.Setenv("PASSES", "/missing")
	pass := Image{2, 1, []float64{0.1, 0.2, 0.3, 4, 5, 6}}
	original := slices.Clone(pass.Light)
	for _, gain := range []float64{0, -2, 0.13} {
		config := OverheadConfig{Gain: gain, Colour: [3]float64{0.62, 0.8, 1}}
		sky := Image{2, 1, []float64{1, 2, 3, 4, 5, 6}}
		want := slices.Clone(sky.Light)
		amount := gain * 0.37
		for i := 0; i < len(want); i += 3 {
			for c := range 3 {
				want[i+c] += amount * config.Colour[c] * pass.Light[i+c]
			}
		}
		if err := addOverhead(sky, pass, config, 0.37); err != nil {
			t.Fatal(err)
		}
		if !slices.Equal(sky.Light, want) {
			t.Fatalf("got %v want %v", sky.Light, want)
		}
	}
	if !slices.Equal(pass.Light, original) {
		t.Fatal("pass mutated")
	}
}

func TestOverheadRejectsInvalidImagesEvenAtZeroLevel(t *testing.T) {
	sky := Image{1, 1, []float64{1, 2, 3}}
	for _, pass := range []Image{{}, {2, 1, make([]float64, 6)}, {1, 1, []float64{1}}} {
		if err := addOverhead(sky, pass, OverheadConfig{}, 0); err == nil {
			t.Fatalf("accepted %v", pass)
		}
		if !slices.Equal(sky.Light, []float64{1, 2, 3}) {
			t.Fatal("changed sky on error")
		}
	}
	if err := addOverhead(Image{}, sky, OverheadConfig{}, 1); err == nil {
		t.Fatal("accepted invalid output")
	}
}

func TestMoonDiscOriginalInterpolationAndMasks(t *testing.T) {
	t.Setenv("MOON_DISC_ALT", "bad")
	t.Setenv("MOON_DISC", "/missing")
	disc := Image{2, 4, []float64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24}}
	original := slices.Clone(disc.Light)
	open := []bool{true, true, false, true, true, true, true, false}
	for _, shift := range []float64{0, 0.25, -0.75, 10} {
		sky := Image{2, 4, make([]float64, 24)}
		for i := range sky.Light {
			sky.Light[i] = 0.3
		}
		want := slices.Clone(sky.Light)
		for y := range 4 {
			source := float64(y) - shift
			low := int(math.Floor(source))
			if low < 0 || low+1 >= 4 {
				continue
			}
			f := source - float64(low)
			for x := range 2 {
				if !open[y*2+x] {
					continue
				}
				for c := range 3 {
					a, b := 0.0, 0.0
					if open[low*2+x] {
						a = disc.Light[(low*2+x)*3+c]
					}
					if open[(low+1)*2+x] {
						b = disc.Light[((low+1)*2+x)*3+c]
					}
					want[(y*2+x)*3+c] += 0.73 * ((1-f)*a + f*b)
				}
			}
		}
		if err := AddMoonDisc(sky, disc, open, MoonDiscConfig{Shift: shift, Level: 0.73}); err != nil {
			t.Fatal(err)
		}
		if !slices.Equal(sky.Light, want) {
			t.Fatalf("shift %v: got %v want %v", shift, sky.Light, want)
		}
	}
	if !slices.Equal(disc.Light, original) || !reflect.DeepEqual(open, []bool{true, true, false, true, true, true, true, false}) {
		t.Fatal("inputs mutated")
	}
}

func TestMoonDiscRejectsInvalidInputsWithoutMutation(t *testing.T) {
	valid := Image{1, 2, []float64{1, 2, 3, 4, 5, 6}}
	for _, test := range []struct {
		disc  Image
		open  []bool
		shift float64
	}{
		{Image{}, []bool{true, true}, 0},
		{Image{2, 1, make([]float64, 6)}, []bool{true, true}, 0},
		{valid, nil, 0}, {valid, []bool{true}, 0},
		{valid, []bool{true, true}, math.NaN()},
		{valid, []bool{true, true}, math.Inf(1)},
	} {
		sky := Image{1, 2, slices.Clone(valid.Light)}
		if err := AddMoonDisc(sky, test.disc, test.open, MoonDiscConfig{Shift: test.shift}); err == nil {
			t.Fatalf("accepted %+v", test)
		}
		if !slices.Equal(sky.Light, valid.Light) {
			t.Fatal("changed output on error")
		}
	}
}

func TestOverheadMoonConcurrentInstances(t *testing.T) {
	pass := Image{1, 3, []float64{1, 2, 3, 4, 5, 6, 7, 8, 9}}
	for _, level := range []float64{0.1, 0.9} {
		t.Run("independent", func(t *testing.T) {
			t.Parallel()
			for range 8 {
				sky := Image{1, 3, make([]float64, 9)}
				if err := addOverhead(sky, pass, OverheadConfig{Gain: 2, Colour: [3]float64{1, 1, 1}}, level); err != nil {
					t.Fatal(err)
				}
				if err := AddMoonDisc(sky, pass, []bool{true, true, true}, MoonDiscConfig{Level: level}); err != nil {
					t.Fatal(err)
				}
				for i, value := range sky.Light {
					want := 2 * level * pass.Light[i]
					if i < 6 {
						want += level * pass.Light[i]
					}
					if value != want {
						t.Fatalf("channel %d: %v want %v", i, value, want)
					}
				}
			}
		})
	}
}
