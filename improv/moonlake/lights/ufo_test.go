package lights

import (
	"math"
	"slices"
	"testing"
)

func ufoFixture(name string) (Image, Image, UFOConfig, UFOInput) {
	sky, front, _, laser := laserFixture()
	config := UFOConfig{Out: 900, Radius: 90, Apex: 450, Gain: 40, Halo: 0.1, Azimuth: 109.65, WaterZ: -(1129.6 - 1076)}
	input := UFOInput{Camera: laser.Camera, Moon: laser.Moon, Seconds: 1.25, Cues: UFOCues{Overhead: 0.7}}
	if name == "golden" {
		return sky, front, config, input
	}
	const width, height = 128, 96
	sky = Image{width, height, make([]float64, width*height*3)}
	front = Image{width, height, make([]float64, width*height*3)}
	input.Camera = NewCamera(width, height, CameraSettings{109.65, -0.5, 70.41, 150})
	input.Water = Image{width, height, make([]float64, width*height*3)}
	for i := range sky.Light {
		sky.Light[i] = float64(i%17+1) / 1024
		front.Light[i] = float64(i%7+1) / 2048
		input.Water.Light[i] = 1
	}
	if name == "halo-foot" {
		config.Halo = 1
	}
	if name == "water" {
		for y := 0; y < height; y++ {
			for x := 0; x < width; x++ {
				v := 0.0
				if x%3 == 0 {
					v = 0.3
				} else if x%3 == 1 {
					v = 0.31
				}
				i := input.Water.At(x, y)
				for c := range 3 {
					input.Water.Light[i+c] = v
				}
			}
		}
		input.Water.Light[0] = 1
	}
	if name == "trees" {
		input.Trees = Image{width, height, make([]float64, width*height*3)}
		for y := 0; y < height; y++ {
			for x := 0; x < width; x++ {
				v := 0.3
				if x%2 == 0 {
					v = 0.31
				}
				i := input.Trees.At(x, y)
				for c := range 3 {
					input.Trees.Light[i+c] = v
				}
			}
		}
		input.Trees.Light[0] = 1
	}
	if name == "off" {
		input.Cues.Overhead = 0
	}
	return sky, front, config, input
}

func TestUFOCapturedOutputs(t *testing.T) {
	wants := map[string][3]string{
		"golden":    {"f8ec0672b6a0d5b950dbdaed13da9dfee8ab6662e61a44d4c7af3e42564e8578", "78edde8af5c6a6a8aaaa42531fa9158f0593071c343f248b7c687f62b2994af4", "3b8b40b17f089bb594741d74ef45006e9f7bdbf1e4c59306cbaceeb5a29285c3"},
		"halo-foot": {"94b58196aa14eba4f647a49102f4ca63b20b1fdd273533631cddacf9f2c4edc1", "ea59b27bd4b0fc2e7c39d01a7f0c9a549913921a4999af4453aa4dd91da852c8", "b0d0bc3ea31abb6a94cf5347e2f206a1db402c5311c5ebcb5a11b9fa66da4f63"},
		"sky-front": {"34f9bb5515f16af73869260ba2d78327b60f0985ebf06ccc7aad5e4156f809ab", "6655d8ff69ecb6f3cb662b5a192ee00ab4ffec387b37abd4647f86fed34fd422", "6c05ac6b57b7b176e9de0aa1d640986a2808637dcc11711fb3b077e0696e5f09"},
		"water":     {"b071374cd0dcbba75f2300550c858ce219f302dbba545ff6d99d5f40d3738bd5", "6655d8ff69ecb6f3cb662b5a192ee00ab4ffec387b37abd4647f86fed34fd422", "41484d951355dca7d6005c66f7fb9d5638865a092641ecd3a26336493df9b247"},
		"trees":     {"526fee6efe7e311ec162fdecb256ce094898b981ed018fb50b6cb1f3d1c71f38", "26c592c777d03f5ff6a58db203eaafc4394f6438f414b130697db0325f393908", "7a50a22a54c4f57ac7e9a8b9214171fe8bd45a67988644ab5e0d23f9c2b86188"},
		"off":       {"2b768b6ed9f0bf5546fbdee9b0553e2a763a06bc57e3dc3d206bc08e1fd17a23", "84e79f458b007b1f713f60916d2035bedc57643a7b2acfd7f9bfa7968256cde9", "a77a3619ade647808cd975115531407150968a48317b878cd89d83c186e063f7"},
	}
	for name, want := range wants {
		t.Run(name, func(t *testing.T) {
			sky, front, config, input := ufoFixture(name)
			beforeSky, beforeFront := slices.Clone(sky.Light), slices.Clone(front.Light)
			trees, water := slices.Clone(input.Trees.Light), slices.Clone(input.Water.Light)
			AddUFOBeam(sky, front, config, input)
			both := append(slices.Clone(sky.Light), front.Light...)
			got := [3]string{effectDigest(both), effectDigest(sky.Light), effectDigest(front.Light)}
			if got != want {
				t.Fatalf("hashes=%q want %q", got, want)
			}
			if !slices.Equal(trees, input.Trees.Light) || !slices.Equal(water, input.Water.Light) {
				t.Fatal("effect changed masks")
			}
			if name == "off" && (!slices.Equal(beforeSky, sky.Light) || !slices.Equal(beforeFront, front.Light)) {
				t.Fatal("off cue changed images")
			}
		})
	}
}

func TestUFOBelowFootHalo(t *testing.T) {
	sky, front, config, input := ufoFixture("halo-foot")
	config.Halo = 0
	AddUFOBeam(sky, front, config, input)
	base := slices.Clone(sky.Light)
	sky, front, config, input = ufoFixture("halo-foot")
	AddUFOBeam(sky, front, config, input)
	axis := Toward(config.Azimuth, 0).Scale(config.Out)
	fx, fy, _ := input.Camera.Project(axis.Add(Vec{0, 0, config.WaterZ}))
	side := Toward(config.Azimuth+90, 0).Scale(config.Radius)
	edge, _, _ := input.Camera.Project(axis.Add(Vec{0, 0, config.WaterZ}).Add(side))
	x, y := int(fx), int(fy)+1
	i := sky.At(x, y)
	half := math.Max(1, math.Abs(edge-fx))
	curl := Curled(float64(x)/(60*float64(sky.Width)/1280)+0.2*input.Seconds, float64(y)/(25*float64(sky.Width)/1280)-0.1*input.Seconds, input.Seconds, 29)
	wisps := 0.4 + 0.6*Smooth(0.3, 0.75, curl)
	lum := (input.Moon[0] + input.Moon[1] + input.Moon[2]) / 3
	want := config.Halo * config.Gain * input.Cues.Overhead * lum * 2 * config.Radius * wisps * math.Exp(-(float64(y)-fy)/(0.5*half)) * 0.7
	if got := sky.Light[i] - base[i]; math.Abs(got-want) > 1e-9*want {
		t.Fatalf("below-foot halo=%v want %v", got, want)
	}
}
