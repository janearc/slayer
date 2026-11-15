package finish

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/janearc/slayer/raw"
	"github.com/janearc/slayer/render/colour"
	"github.com/janearc/slayer/render/frame"
)

// pfmIn writes a pfm of a size and a light into a directory, by name.
func pfmIn(t *testing.T, directory, name string, w, h int, light float32) {
	t.Helper()
	made := frame.New(w, h)
	for i := range made.Pix {
		made.Pix[i].R, made.Pix[i].G, made.Pix[i].B = light, light/2,
			light/4
	}
	if err := raw.Write(filepath.Join(directory, name), made); err != nil {
		t.Fatal(err)
	}
}

// pngIn writes a png of a size and one grey code into a directory.
func pngIn(t *testing.T, directory, name string, w, h int, code uint8) {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for i := 0; i < len(img.Pix); i += 4 {
		img.Pix[i], img.Pix[i+1], img.Pix[i+2], img.Pix[i+3] = code,
			code/2, code/4, 255
	}
	file, err := os.Create(filepath.Join(directory, name))
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	if err := png.Encode(file, img); err != nil {
		t.Fatal(err)
	}
}

// the plates come in name order, each its own light, and say their size
// and count.
func TestPlatesComeInNameOrder(t *testing.T) {
	directory := t.TempDir()
	pfmIn(t, directory, "b.pfm", 4, 3, 2)
	pfmIn(t, directory, "a.pfm", 4, 3, 1)
	pfmIn(t, directory, "c.pfm", 4, 3, 3)
	if err := os.WriteFile(filepath.Join(directory, "notes.txt"), nil,
		private); err != nil {
		t.Fatal(err)
	}
	plates, err := OpenPlates(directory)
	if err != nil {
		t.Fatal(err)
	}
	if width, height := plates.Size(); plates.Len() != 3 || width != 4 ||
		height != 3 {
		t.Errorf("%d plates of %dx%d", plates.Len(), width, height)
	}
	for place, want := range []float32{1, 2, 3} {
		got, err := plates.Frame(place)
		if err != nil {
			t.Fatal(err)
		}
		if got.Pix[0].R != want {
			t.Errorf("plate %d has light %v, want %v", place,
				got.Pix[0].R, want)
		}
	}
	if names := plates.Names(); len(names) != 3 ||
		filepath.Base(names[0]) != "a.pfm" {
		t.Errorf("the names are %v", names)
	}
}

// the anti-test: a place with no plate is refused, by the place.
func TestAPlaceWithNoPlateIsRefused(t *testing.T) {
	directory := t.TempDir()
	pfmIn(t, directory, "a.pfm", 2, 2, 1)
	plates, err := OpenPlates(directory)
	if err != nil {
		t.Fatal(err)
	}
	for _, place := range []int{-1, 1, 9} {
		if _, err := plates.Frame(place); err == nil ||
			!strings.Contains(err.Error(), "there is no frame") {
			t.Errorf("place %d gave %v", place, err)
		}
	}
}

// a size mismatch is refused naming the odd file; so are mixed kinds
// and an empty directory, each by name.
func TestPlatesAreRefusedByName(t *testing.T) {
	odd := t.TempDir()
	pfmIn(t, odd, "a.pfm", 4, 3, 1)
	pfmIn(t, odd, "b.pfm", 5, 3, 1)
	mixed := t.TempDir()
	pfmIn(t, mixed, "a.pfm", 4, 3, 1)
	pngIn(t, mixed, "b.png", 4, 3, 100)
	oddPng := t.TempDir()
	pngIn(t, oddPng, "a.png", 4, 3, 100)
	pngIn(t, oddPng, "b.png", 4, 4, 100)
	empty := t.TempDir()
	if err := os.WriteFile(filepath.Join(empty, "readme.txt"), nil,
		private); err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name, directory string
		says            []string
	}{
		{"size", odd, []string{"b.pfm", "5x3", "4x3"}},
		{"png size", oddPng, []string{"b.png", "4x4", "4x3"}},
		{"kinds", mixed, []string{"b.png", ".png among .pfm"}},
		{"empty", empty, []string{"holds no .pfm or .png"}},
		{"missing", filepath.Join(empty, "nowhere"),
			[]string{"cannot be read"}},
	}
	for _, each := range cases {
		_, err := OpenPlates(each.directory)
		if err == nil {
			t.Errorf("%s: taken", each.name)
			continue
		}
		for _, word := range each.says {
			if !strings.Contains(err.Error(), word) {
				t.Errorf("%s: %q does not say %q", each.name, err, word)
			}
		}
	}
}

// a file with the right name and the wrong insides is refused by name:
// a pfm that is not colour, one with no size, a png that is not one.
func TestAPlateWithTheWrongInsidesIsRefused(t *testing.T) {
	cases := map[string]string{
		"grey.pfm":     "Pf\n2 2\n-1\n",
		"flat.pfm":     "PF\n0 0\n-1\n",
		"headless.pfm": "PF\n",
		"pretend.png":  "not a png",
	}
	for name, insides := range cases {
		directory := t.TempDir()
		if err := os.WriteFile(filepath.Join(directory, name),
			[]byte(insides), private); err != nil {
			t.Fatal(err)
		}
		if _, err := OpenPlates(directory); err == nil ||
			!strings.Contains(err.Error(), name) {
			t.Errorf("%s gave %v", name, err)
		}
	}
}

// a plate that opened but cannot be read later, a pfm cut short or a
// png whose body is gone, is refused when asked for.
func TestAPlateThatBreaksAfterOpeningIsRefused(t *testing.T) {
	directory := t.TempDir()
	pfmIn(t, directory, "a.pfm", 4, 4, 1)
	plates, err := OpenPlates(directory)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Truncate(plates.Names()[0], 30); err != nil {
		t.Fatal(err)
	}
	if _, err := plates.Frame(0); err == nil {
		t.Error("a pfm cut short was read")
	}
	pngs := t.TempDir()
	pngIn(t, pngs, "a.png", 8, 8, 100)
	plates, err = OpenPlates(pngs)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Truncate(plates.Names()[0], 40); err != nil {
		t.Fatal(err)
	}
	if _, err := plates.Frame(0); err == nil {
		t.Error("a png cut short was read")
	}
	if err := os.Remove(plates.Names()[0]); err != nil {
		t.Fatal(err)
	}
	if _, err := plates.Frame(0); err == nil {
		t.Error("a png that is gone was read")
	}
}

// a png's round trip is within one code value: develop a frame, write
// it as a png, read it back and develop it again at one exposure.
func TestAPngRoundTripIsWithinOneCodeValue(t *testing.T) {
	directory := t.TempDir()
	first := colour.Develop(gradient(32, 32), 1)
	file, err := os.Create(filepath.Join(directory, "a.png"))
	if err != nil {
		t.Fatal(err)
	}
	if err := png.Encode(file, first); err != nil {
		t.Fatal(err)
	}
	file.Close()
	plates, err := OpenPlates(directory)
	if err != nil {
		t.Fatal(err)
	}
	light, err := plates.Frame(0)
	if err != nil {
		t.Fatal(err)
	}
	again := colour.Develop(light, 1)
	worst := 0
	for i := range first.Pix {
		worst = max(worst, abs(int(first.Pix[i])-int(again.Pix[i])))
	}
	if worst > 1 {
		t.Errorf("a code value moved by %d on the round trip", worst)
	}
}

// the anti-test: a png's light is bounded by white, since it was
// clipped when it was written; a pfm's is not.
func TestAPngsLightIsClippedAtWhite(t *testing.T) {
	directory := t.TempDir()
	pngIn(t, directory, "a.png", 2, 2, 255)
	plates, err := OpenPlates(directory)
	if err != nil {
		t.Fatal(err)
	}
	light, _ := plates.Frame(0)
	if light.Pix[0].R > 1 {
		t.Errorf("a png holds light %v, past white", light.Pix[0].R)
	}
	if light.Pix[0].R < 0.99 {
		t.Errorf("a white png holds light %v", light.Pix[0].R)
	}
	if got := colour.Decode(0); got != 0 {
		t.Errorf("black decodes to %v", got)
	}
}

// a png with transparency is read as the colour it holds, alpha not
// counting as light.
func TestAPngsAlphaIsNotLight(t *testing.T) {
	directory := t.TempDir()
	img := image.NewNRGBA(image.Rect(0, 0, 1, 1))
	img.SetNRGBA(0, 0, color.NRGBA{R: 200, G: 100, B: 50, A: 128})
	file, err := os.Create(filepath.Join(directory, "a.png"))
	if err != nil {
		t.Fatal(err)
	}
	if err := png.Encode(file, img); err != nil {
		t.Fatal(err)
	}
	file.Close()
	plates, err := OpenPlates(directory)
	if err != nil {
		t.Fatal(err)
	}
	light, _ := plates.Frame(0)
	if float32(colour.Decode(200)) != light.Pix[0].R {
		t.Errorf("a half clear pixel reads %v, not its colour",
			light.Pix[0].R)
	}
}

// abs is the size of a number.
func abs(number int) int { return max(number, -number) }
