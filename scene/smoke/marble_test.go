package smoke

import (
	"bytes"
	"errors"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/interfaces/sheet"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/render/colour"
	"github.com/janearc/slayer/render/trace"
	"github.com/janearc/slayer/scene/bodies/template"
	"github.com/janearc/slayer/scene/sky"
	"github.com/janearc/slayer/scene/thing"
)

func load(t *testing.T, path string) image.Image {
	f, err := os.Open(path)
	if err != nil {
		t.Skip(err)
	}
	defer f.Close()
	img, _, err := image.Decode(f)
	if err != nil {
		t.Fatal(err)
	}
	return img
}

// The earth wearing nasa's marbles: the blue by day as its map, the
// black by night as its lights, two frames at two exposures, when
// SLAYER_MARBLE names a folder holding blue.jpg and black.jpg. The images
// are nasa's and stay out of the repository; without them this skips.
//
// The pictures are for looking at. The sheet goes in the folder
// SLAYER_SHEET names, as the other tests' do, and in a temporary folder
// when it is unset. What the test holds is that the day frame has an
// earth in it, whatever the marbles show.
func TestTheMarblesOnTheEarth(t *testing.T) {
	dir := os.Getenv("SLAYER_MARBLE")
	if dir == "" {
		t.Skip("SLAYER_MARBLE is not set")
	}
	blue, black := load(t, filepath.Join(dir, "blue.jpg")), load(t, filepath.Join(dir, "black.jpg"))
	build := func(sun maths.Vec, night, own bool) (*thing.World, maths.Subspace) {
		w := dark()
		if night {
			// at two hundred times the exposure the sky's floor would
			// be grey and the stars white blobs; the floor is black
			// and the stars are as faint as a long exposure leaves
			w.Sky = sky.Stars{Seed: 15, Share: 0.06, Size: 0.35, Bright: 0.004}
		}
		w.Lamp(thing.Sun(sun, maths.RGB(1.2, 1.2, 1.2)))
		e := template.Earth()
		e.Moons, e.Clouds = nil, nil
		// the pictures' left edge is the antimeridian; turned so
		// africa faces the eye on plus z
		e.Ground.Map, e.Ground.Turn = blue, -1.6
		if own {
			// the template's own cities, laid as the marble lays
			// them, on the real map
			e.Ground.Lights = nil
		} else {
			e.Ground.Lights, e.Cities = black, nil
		}
		if err := e.Place(w, maths.Vec{}, 1/e.Radius, 10); err != nil {
			t.Fatal(err)
		}
		from := maths.V(0, 0, 3.2)
		return w, maths.Subspace{Origin: from, Orientation: maths.LookAt(from, maths.Vec{})}
	}
	lens := film.Pinhole{Wide: 40 * maths.Degree, Pixel: 0.75}
	dayW, eye := build(maths.V(0.6, 0.3, 1), false, false)
	day := trace.Sampled(dayW, eye, lens, 0, 480, 200, 2)
	nightW, _ := build(maths.V(0, 0, -1), true, false)
	night := trace.Sampled(nightW, eye, lens, 0, 480, 200, 2)
	ownW, _ := build(maths.V(0, 0, -1), true, true)
	own := trace.Sampled(ownW, eye, lens, 0, 480, 200, 2)
	nightTone := colour.Tone{Exposure: 200, Knee: 0.6, Top: 20, Bloom: 1}
	s := sheet.New(3)
	s.Add("the blue marble, by day", colour.Tone{Exposure: 1}.Develop(day), 0.75)
	s.Add("the black marble, by night, exposed two hundred times", nightTone.Develop(night), 0.75)
	s.Add("the scene's own cities on the same map, laid as the marble lays them", nightTone.Develop(own), 0.75)
	out := os.Getenv("SLAYER_SHEET")
	if out == "" {
		out = t.TempDir()
	}
	if err := s.PNG(filepath.Join(out, t.Name()+".png")); err != nil {
		t.Fatal(err)
	}
	covered := 0
	for _, p := range day.Pix {
		if p.Cover > 0.5 {
			covered++
		}
	}
	if covered < len(day.Pix)/3 {
		t.Errorf("the earth covers %d of %d pixels by day", covered, len(day.Pix))
	}
}

// aJPEG is a small jpeg picture, as nasa's marble is a big one.
func aJPEG(t *testing.T) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 8, 4))
	for x := 0; x < 8; x++ {
		for y := 0; y < 4; y++ {
			img.Set(x, y, color.RGBA{uint8(30 * x), uint8(60 * y), 128, 255})
		}
	}
	var b bytes.Buffer
	if err := jpeg.Encode(&b, img, nil); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

// With SLAYER_MARBLE unset, or set to nothing, marble is no picture and the
// smoke draws the painted earth, as it documents.
func TestMarbleUnsetIsNoPicture(t *testing.T) {
	t.Setenv("SLAYER_MARBLE", "")
	var got image.Image
	if err := refusal(func() { got = marble("blue.jpg") }); err != nil || got != nil {
		t.Errorf("marble with SLAYER_MARBLE unset is %v, refused with %v", got, err)
	}
}

// With SLAYER_MARBLE set, the jpeg in it is the marble.
func TestMarbleSetGivesTheJPEG(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "blue.jpg"), aJPEG(t), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("SLAYER_MARBLE", dir)
	var got image.Image
	if err := refusal(func() { got = marble("blue.jpg") }); err != nil {
		t.Fatal(err)
	}
	if got == nil || got.Bounds() != image.Rect(0, 0, 8, 4) {
		t.Errorf("marble is %v", got)
	}
	if err := refusal(func() { thirtyEight() }); err != nil {
		t.Errorf("smoke 38 with a marble: %v", err)
	}
}

// SLAYER_MARBLE set and the marble not usable is refused by name, with the
// path and the reason, and is not drawn as the painted earth: a missing
// blue.jpg, one that cannot be read, one that is not a picture, one cut
// short, an empty one, and a png.
func TestMarbleSetButFailingIsRefused(t *testing.T) {
	whole := aJPEG(t)
	var aPNG bytes.Buffer
	if err := png.Encode(&aPNG, image.NewRGBA(image.Rect(0, 0, 2, 2))); err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name  string
		setup func(path string) error
		// is, when there is one, an error the refusal wraps; text is
		// what the refusal's message holds
		is   error
		text string
	}{
		{"missing", func(string) error { return nil }, fs.ErrNotExist, "cannot be opened"},
		{"unreadable: a folder where the file should be",
			func(p string) error { return os.Mkdir(p, 0o755) }, nil, "is a directory"},
		{"not a picture", func(p string) error {
			return os.WriteFile(p, []byte("this is not a jpeg"), 0o644)
		}, nil, "not a jpeg"},
		{"cut short", func(p string) error {
			return os.WriteFile(p, whole[:len(whole)/2], 0o644)
		}, nil, "not a jpeg"},
		{"empty", func(p string) error { return os.WriteFile(p, nil, 0o644) }, nil, "not a jpeg"},
		{"a png", func(p string) error { return os.WriteFile(p, aPNG.Bytes(), 0o644) }, nil, "not a jpeg"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "blue.jpg")
			if err := c.setup(path); err != nil {
				t.Fatal(err)
			}
			if _, err := os.Stat(path); c.name != "missing" && err != nil {
				t.Fatal(err)
			}
			t.Setenv("SLAYER_MARBLE", dir)
			var got image.Image
			err := refusal(func() { got = marble("blue.jpg") })
			if err == nil {
				t.Fatalf("marble is %v, and was not refused", got)
			}
			if !strings.Contains(err.Error(), path) || !strings.Contains(err.Error(), c.text) {
				t.Errorf("refused with %q, which should name %s and say %q", err, path, c.text)
			}
			if c.is != nil && !errors.Is(err, c.is) {
				t.Errorf("refused with %q, which should wrap %v", err, c.is)
			}
			if err := refusal(func() { thirtyEight() }); err == nil {
				t.Error("smoke 38 drew the painted earth with SLAYER_MARBLE set to a bad marble")
			}
		})
	}
}
