//go:build draw

package pan

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The tests' film and stills. Picked by the pan slice's author,
// 2026-11-12: small, and a still a fifth larger each way than the film.
const (
	filmWidth   = 64
	filmHeight  = 32
	stillWidth  = 80
	stillHeight = 40
	testRate    = 4
)

// stillIn writes a png still of the tests' size to directory by name,
// each pixel's red its column and green its row, blue the shade given,
// so any crop of it says where it was cut from.
func stillIn(t *testing.T, directory, name string, shade uint8) {
	t.Helper()
	picture := image.NewRGBA(image.Rect(0, 0, stillWidth, stillHeight))
	for y := range stillHeight {
		for x := range stillWidth {
			picture.Set(x, y, color.RGBA{R: uint8(x), G: uint8(y),
				B: shade, A: 255})
		}
	}
	var encoded bytes.Buffer
	if err := png.Encode(&encoded, picture); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(directory, name)
	if err := os.WriteFile(path, encoded.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}
}

// scripted is a script of two shots of one bar each over crypt's first
// two bars, with more lines after the shots' list, written and read.
func scripted(t *testing.T, shots, after string) (Cut, error) {
	t.Helper()
	directory := t.TempDir()
	stillIn(t, directory, "a.png", 10)
	stillIn(t, directory, "b.png", 200)
	text := fmt.Sprintf("size: {width: %d, height: %d}\nfps: %d\n"+
		"sound: {score: \"@crypt\", voicing: \"@crypt-dmg\", "+
		"bars: [1, 2]}\nshots:\n%s%s", filmWidth, filmHeight, testRate,
		shots, after)
	path := filepath.Join(directory, "cut.yaml")
	if err := os.WriteFile(path, []byte(text), 0o600); err != nil {
		t.Fatal(err)
	}
	return Read(path)
}

// twoShots is the tests' two shots: a moving across, b still.
const twoShots = "  - {still: a.png, bars: 1, from: [0, 0], to: [1, 1]}\n" +
	"  - {still: b.png, bars: 1, from: [0.5, 0.5], to: [0.5, 0.5]}\n"

// a frame is a real crop of its shot's still, the film's size, its
// window at from at the shot's start; the next shot starts on its bar.
func TestPictureIsACropOfTheStill(t *testing.T) {
	cut, err := scripted(t, twoShots, "")
	if err != nil {
		t.Fatal(err)
	}
	first, err := cut.Picture(0)
	if err != nil {
		t.Fatal(err)
	}
	if first.Bounds().Size() != image.Pt(filmWidth, filmHeight) {
		t.Fatalf("a frame is %v", first.Bounds())
	}
	if got := first.RGBAAt(5, 7); got.R != 5 || got.G != 7 || got.B != 10 {
		t.Errorf("frame 0's pixel 5,7 is %v, not the still's own", got)
	}
	laid := cut.spans()
	second := int(laid[1].start.Seconds() * testRate)
	cutTo, err := cut.Picture(second + 1)
	if err != nil {
		t.Fatal(err)
	}
	middle := cutTo.RGBAAt(0, 0)
	if middle.B != 200 || middle.R != (stillWidth-filmWidth)/2 {
		t.Errorf("the second shot's frame begins at %v", middle)
	}
}

// the anti-test: the window moves, eased: the last frame of the moving
// shot is near its end and the middle frame between, never past.
func TestWindowMovesFromItsStartToItsEnd(t *testing.T) {
	cut, err := scripted(t, twoShots, "")
	if err != nil {
		t.Fatal(err)
	}
	laid := cut.spans()
	last := int(laid[0].end.Seconds()*testRate) - 1
	picture, err := cut.Picture(last)
	if err != nil {
		t.Fatal(err)
	}
	slack := stillWidth - filmWidth
	if got := int(picture.RGBAAt(0, 0).R); got <= slack/2 || got > slack {
		t.Errorf("the last frame's window starts at %d of %d", got, slack)
	}
}

// a fade in starts the shot black; the anti-test: a shot with no fade
// starts on its picture.
func TestFadeFromBlack(t *testing.T) {
	cut, err := scripted(t, "  - {still: a.png, bars: 1, fade: {in: 1}}\n"+
		"  - {still: b.png, bars: 1}\n", "")
	if err != nil {
		t.Fatal(err)
	}
	black, err := cut.Picture(0)
	if err != nil {
		t.Fatal(err)
	}
	if got := black.RGBAAt(20, 20); got.R != 0 || got.G != 0 || got.A != 255 {
		t.Errorf("a faded shot's first frame is %v", got)
	}
	plain, _ := scripted(t, twoShots, "")
	lit, _ := plain.Picture(0)
	if lit.RGBAAt(20, 20).G == 0 {
		t.Error("a shot with no fade started black")
	}
}

// a title is drawn until its time and not after.
func TestTitleUntilItsTime(t *testing.T) {
	titled, err := scripted(t, twoShots,
		"title: {words: glass hour, until: 0.5, rows: 4}\n")
	if err != nil {
		t.Fatal(err)
	}
	plain, _ := scripted(t, twoShots, "")
	early, _ := titled.Picture(0)
	bare, _ := plain.Picture(0)
	if bytes.Equal(early.Pix, bare.Pix) {
		t.Error("the title is not on the first frame")
	}
	if !samePicture(t, titled, plain, testRate) {
		t.Error("the title is on a frame after its time")
	}
}

// samePicture is whether two cuts' frame index are the same picture.
func samePicture(t *testing.T, cut, other Cut, index int) bool {
	t.Helper()
	first, err := cut.Picture(index)
	if err != nil {
		t.Fatal(err)
	}
	second, err := other.Picture(index)
	if err != nil {
		t.Fatal(err)
	}
	return bytes.Equal(first.Pix, second.Pix)
}

// the sound is the bars played, as long as the film to within a
// sample, faded to nothing at its end.
func TestSoundIsTheBarsPlayed(t *testing.T) {
	cut, err := scripted(t, twoShots, "")
	if err != nil {
		t.Fatal(err)
	}
	sound, err := cut.Sound()
	if err != nil {
		t.Fatal(err)
	}
	want := cut.Length().Seconds() * float64(sound.Rate())
	if got := float64(sound.Length()); got < want-1 || got > want+1 {
		t.Errorf("the sound has %v samples, not %v", got, want)
	}
	samples := sound.Samples()
	if last := samples[len(samples)-1]; last != 0 {
		t.Errorf("the sound ends at %v, not faded", last)
	}
}

// the anti-tests: a script is refused, by what was wrong, where its
// shots' bars and the sound's disagree, a still is smaller than the
// film, a window's share is outside 0 to 1, a key is unknown, or a
// preset is not one.
func TestReadRefuses(t *testing.T) {
	for _, each := range []struct{ shots, after, says string }{
		{"  - {still: a.png, bars: 1}\n", "", "take 1 bars"},
		{"  - {still: a.png, bars: 2, from: [2, 0]}\n", "", "0 to 1"},
		{twoShots, "colour: red\n", "colour"},
		{"  - {still: a.png, bars: 2}\n", "", ""},
	} {
		_, err := scripted(t, each.shots, each.after)
		if each.says == "" {
			if err != nil {
				t.Errorf("a good script was refused: %v", err)
			}
			continue
		}
		if err == nil || !strings.Contains(err.Error(), each.says) {
			t.Errorf("%q gave %v", each.shots+each.after, err)
		}
	}
	directory := t.TempDir()
	stillIn(t, directory, "a.png", 10)
	for _, each := range []struct{ size, says string }{
		{"{width: 640, height: 360}", "at least the film's size"},
		{"nonesuch", "no job preset"},
	} {
		path := filepath.Join(directory, "cut.yaml")
		text := "size: " + each.size + "\nfps: 4\nsound: {score: " +
			"\"@crypt\", voicing: \"@crypt-dmg\", bars: [1, 1]}\n" +
			"shots:\n  - {still: a.png, bars: 1}\n"
		if err := os.WriteFile(path, []byte(text), 0o600); err != nil {
			t.Fatal(err)
		}
		if _, err := Read(path); err == nil ||
			!strings.Contains(err.Error(), each.says) {
			t.Errorf("size %s gave %v", each.size, err)
		}
	}
}

// a still named by an absolute path is read from there, not under the
// script's folder; the anti-test: a relative one that is not there is
// refused by its name.
func TestStillsByAbsolutePath(t *testing.T) {
	elsewhere := t.TempDir()
	stillIn(t, elsewhere, "far.png", 10)
	shots := "  - {still: " + filepath.Join(elsewhere, "far.png") +
		", bars: 1}\n  - {still: b.png, bars: 1}\n"
	if _, err := scripted(t, shots, ""); err != nil {
		t.Errorf("an absolute still was refused: %v", err)
	}
	_, err := scripted(t, "  - {still: far.png, bars: 2}\n", "")
	if err == nil || !strings.Contains(err.Error(), "far.png") {
		t.Errorf("a still not there gave %v", err)
	}
}
