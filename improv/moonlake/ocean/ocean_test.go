package ocean

import (
	"bytes"
	"encoding/binary"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/janearc/libtheme-css/spaces/srgb"
	"github.com/janearc/slayer/oceantex"
)

func TestReductions(t *testing.T) {
	sky := &Sky{width: 2, height: 40, light: make([]float32, 2*40*3)}
	copy(sky.light, []float32{12, 6, 3, 6, 3, 1.5})
	copy(sky.light[38*6:], []float32{1, 2, 3, 3, 4, 5, 5, 6, 7, 7, 8, 9})
	if got, want := sky.aboveOf(), ([3]float64{4, 5, 6}); got != want {
		t.Fatalf("top rows: %v, want %v", got, want)
	}
	if got, want := sky.brightest(), ([3]float64{6.75, 6, 6}); got != want {
		t.Fatalf("brightest: %v, want %v", got, want)
	}
	rounded := &Sky{light: []float32{16777216, 1, 1, 8388608, 0, 0}}
	if got, want := rounded.brightest(), ([3]float64{12582912, .5, .5}); got != want {
		t.Fatalf("float32 threshold: %v, want %v", got, want)
	}
}

func TestSkyProjection(t *testing.T) {
	sky := &Sky{width: 2, height: 2, light: []float32{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12}, moon: [3]float64{1, 2, 3}, forward: oceantex.Vec{Y: 1}, right: oceantex.Vec{X: 1}, up: oceantex.Vec{Z: 1}, focal: 1, disc: .5, above: [3]float64{20, 30, 40}}
	for _, test := range []struct {
		name      string
		direction oceantex.Vec
		want      [3]float64
	}{
		{"bottom", oceantex.Vec{X: -.5, Y: 1, Z: -.5}, [3]float64{1, 2, 3}},
		{"top", oceantex.Vec{X: .5, Y: 1, Z: .5}, [3]float64{10, 11, 12}},
		{"center", oceantex.Vec{Y: 1}, [3]float64{4, 5, 6}},
		{"behind above", oceantex.Vec{Y: -1, Z: 1}, [3]float64{20, 30, 40}},
		{"behind below", oceantex.Vec{Y: -1, Z: -1}, [3]float64{}},
		{"past right", oceantex.Vec{X: 1, Y: 1, Z: .5}, [3]float64{20, 30, 40}},
		{"past bottom", oceantex.Vec{Y: 1, Z: -1}, [3]float64{}},
	} {
		t.Run(test.name, func(t *testing.T) {
			r, g, b := srgb.Light(sky.At(test.direction))
			wr, wg, wb := srgb.Light(srgb.FromLight(test.want[0], test.want[1], test.want[2]))
			if got, want := ([3]float64{r, g, b}), ([3]float64{wr, wg, wb}); got != want {
				t.Fatalf("light: %v, want %v", got, want)
			}
		})
	}
}

func TestPFM(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sky.pfm")
	for _, test := range []struct {
		name, header string
		order        binary.ByteOrder
	}{
		{"little", "PF\n1 2\n-2\n", binary.LittleEndian},
		{"big", "PF\n1 2\n2\n", binary.BigEndian},
	} {
		t.Run(test.name, func(t *testing.T) {
			var data bytes.Buffer
			data.WriteString(test.header)
			want := []float32{1, -2, .25, 4, 5, 6}
			if err := binary.Write(&data, test.order, want); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, data.Bytes(), 0600); err != nil {
				t.Fatal(err)
			}
			got, err := ReadPFM(path)
			if err != nil || got.Width != 1 || got.Height != 2 || !reflect.DeepEqual(got.Light, want) {
				t.Fatalf("read: %v %v", got, err)
			}
		})
	}
	for _, raw := range []string{"Pf\n1 1\n-1\n", "PF\n1 1\n-1\n", "PF\nx 1\n-1\n"} {
		if err := os.WriteFile(path, []byte(raw), 0600); err != nil {
			t.Fatal(err)
		}
		if _, err := ReadPFM(path); err == nil {
			t.Fatalf("accepted %q", raw)
		}
	}
	picture := &oceantex.Texture{Width: 1, Height: 2, Light: [][3]float64{{1.0 / 3, -2, .25}, {4, 5, 6}}}
	if err := WritePFM(path, picture); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var want bytes.Buffer
	want.WriteString("PF\n1 2\n-1\n")
	for _, pixel := range picture.Light {
		for _, channel := range pixel {
			if err := binary.Write(&want, binary.LittleEndian, math.Float32bits(float32(channel))); err != nil {
				t.Fatal(err)
			}
		}
	}
	if !bytes.Equal(got, want.Bytes()) {
		t.Fatalf("PFM bytes: %x, want %x", got, want.Bytes())
	}
}

func TestExplicitConfiguration(t *testing.T) {
	config := DefaultConfig()
	image := RGB{Width: 2, Height: 2, Light: []float32{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12}}
	reference := RGB{Width: 1, Height: 1, Light: []float32{30, 60, 90}}
	sky, err := NewSky(config, image, &reference)
	if err != nil {
		t.Fatal(err)
	}
	if sky.moon != ([3]float64{30, 60, 90}) || sky.above != ([3]float64{8.5, 9.5, 10.5}) {
		t.Fatalf("reference replaced sky: %+v", sky)
	}
	if sky.focal != float64(2)/2/(70.41/2/150) || sky.disc != math.Pi*math.Pow(6*.259*math.Pi/180, 2) {
		t.Fatalf("projection: %+v", sky)
	}
	image.Light[0] = 1000
	reference.Light[0] = 1000
	if sky.light[0] != 1 || sky.moon[0] != 30 {
		t.Fatal("sky retained mutable input")
	}
	config.Width, config.Height, config.Seconds, config.Wind = 7, 9, 2.5, 4
	scene, camera, err := Setup(config, image, nil)
	if err != nil {
		t.Fatal(err)
	}
	if scene.Sea.Wind != 4 || scene.Sea.Fetch != 1500 || scene.Sun != toward(109.65, 3) || scene.Sea.Toward != (90-(109.65+180))*math.Pi/180 || scene.Salinity != 0 || scene.Celsius != 15 || scene.Chlorophyll != 2 {
		t.Fatalf("scene: %+v", scene)
	}
	if camera.Width != 7 || camera.Height != 9 || camera.Seconds != 2.5 || camera.Seed != 3 || camera.Eye.Origin.Z != 1129.6-1076 {
		t.Fatalf("camera: %+v", camera)
	}
	changed := config
	changed.LookDown, changed.Focal, changed.MoonScale = 10, 75, 3
	other, err := NewSky(changed, image, nil)
	if err != nil {
		t.Fatal(err)
	}
	if other.forward == sky.forward || other.focal != sky.focal/2 || other.disc != sky.disc/4 {
		t.Fatal("per-call configuration ignored")
	}
	if sky.forward != toward(109.65, -.5) || sky.moon != ([3]float64{30, 60, 90}) {
		t.Fatal("later call changed original sky")
	}
	for _, invalid := range []RGB{{}, {Width: -1, Height: 1}, {Width: 1, Height: 1, Light: []float32{1}}} {
		if _, err := NewSky(config, invalid, nil); err == nil {
			t.Fatalf("accepted invalid sky: %+v", invalid)
		}
		if _, err := NewSky(config, image, &invalid); err == nil {
			t.Fatalf("accepted invalid reference: %+v", invalid)
		}
	}
}
