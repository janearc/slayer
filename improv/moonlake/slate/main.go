// Command slate is improv/moonlake's: it writes a frame's timing over
// the frame, as a clapperboard does, so a still can be matched to its
// place in the film. It draws with slayer's hud, in the top left
// corner:
//
//	frame 5245          the frame's number in the film, from the song's start
//	3:38.54             minutes and seconds into the song
//	bar 97.1            the bar and beat, from the cues
//	fire lit +0.04 s    since the fire was lit, and since its newest strike
//
//	slate cue.env seconds fps frame.png
//
// cue.env is the frame's cues as cueshow wrote them. The png is drawn
// over in place.
package main

import (
	"bufio"
	"errors"
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/janearc/libtheme-css/css"
	"github.com/janearc/libtheme-css/spaces/srgb"
	"github.com/janearc/slayer/hud"
)

// rows is how many rows of cells the slate lays the picture out in: 36,
// so a 1152-pixel frame has cells 32 pixels high.
const rows = 36

// main draws the slate; see the package comment.
func main() {
	if len(os.Args) != 5 {
		fail(errors.New("usage: slate cue.env seconds fps frame.png"))
	}
	cues, err := readCues(os.Args[1])
	if err != nil {
		fail(err)
	}
	seconds, err := strconv.ParseFloat(os.Args[2], 64)
	if err != nil || !(seconds >= 0) || math.IsInf(seconds, 1) {
		fail(fmt.Errorf("seconds %q: want a time into the song, zero or more",
			os.Args[2]))
	}
	fps, err := strconv.ParseFloat(os.Args[3], 64)
	if err != nil || !(fps > 0) || math.IsInf(fps, 1) {
		fail(fmt.Errorf("fps %q: want frames a second, more than zero",
			os.Args[3]))
	}
	lines, err := wording(cues, seconds, fps)
	if err != nil {
		fail(err)
	}
	if err := stamp(os.Args[4], lines, seconds); err != nil {
		fail(err)
	}
}

// readCues is cueshow's key=value lines, by key.
func readCues(path string) (map[string]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	cues := map[string]string{}
	lines := bufio.NewScanner(file)
	for lines.Scan() {
		key, value, found := strings.Cut(lines.Text(), "=")
		if found {
			cues[key] = value
		}
	}
	return cues, lines.Err()
}

// wording is the slate's lines for a frame seconds into the song at fps
// frames a second, with its cues. The fire line needs fire_lit, which
// is negative before the fire is lit, and fire_since, the seconds since
// each strike, newest first.
func wording(cues map[string]string, seconds, fps float64) ([]string, error) {
	minutes := math.Floor(seconds / 60)
	lines := []string{
		fmt.Sprintf("frame %d", int(math.Round(seconds*fps))),
		fmt.Sprintf("%.0f:%05.2f", minutes, seconds-60*minutes),
		"bar " + cues["bar"],
	}
	lit, err := strconv.ParseFloat(cues["fire_lit"], 64)
	if err != nil || math.IsNaN(lit) {
		return nil, fmt.Errorf("fire_lit %q: want seconds since the fire "+
			"was lit, or a negative number before it is", cues["fire_lit"])
	}
	if lit < 0 {
		return append(lines, "fire not lit"), nil
	}
	newest, _, _ := strings.Cut(cues["fire_since"], ",")
	since, err := strconv.ParseFloat(newest, 64)
	if err != nil || !(since >= 0) {
		return nil, fmt.Errorf("fire_since %q: want seconds since each "+
			"strike, newest first", cues["fire_since"])
	}
	return append(lines, fmt.Sprintf("fire lit %+.2f s  struck %+.2f s",
		lit, since)), nil
}

// stamp draws the lines over the png at path, one a row from the top
// left, in light ink on a dark ground, and writes it back in place: to a
// file beside it first, which is renamed over it, so a failure leaves
// the frame as it was.
func stamp(path string, lines []string, seconds float64) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	decoded, err := png.Decode(file)
	file.Close()
	if err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	img := image.NewRGBA(decoded.Bounds())
	draw.Draw(img, img.Bounds(), decoded, decoded.Bounds().Min, draw.Src)
	slate := hud.Hud{}
	for row, line := range lines {
		slate.Elements = append(slate.Elements, hud.Text{
			Words: hud.Words(" " + line + " "),
			At:    hud.Point{Across: 0.01, Down: (float64(row) + 1.5) / rows},
			Align: hud.Left, Ink: "ink", Ground: "ground"})
	}
	at := time.Duration(seconds * float64(time.Second))
	if err := slate.Picture(img, rows, at, theme(), hud.View{}); err != nil {
		return err
	}
	out, err := os.CreateTemp(filepath.Dir(path), ".slate-*.png")
	if err != nil {
		return err
	}
	if err := png.Encode(out, img); err != nil {
		out.Close()
		os.Remove(out.Name())
		return err
	}
	if err := out.Close(); err != nil {
		os.Remove(out.Name())
		return err
	}
	return os.Rename(out.Name(), path)
}

// theme is the slate's two colours: a warm white ink on a near-black
// ground, so it reads over the night and over the fire alike.
func theme() hud.Theme {
	sheet := css.New()
	sheet.Set("ink", srgb.MustHex("#f4efe6").Swatch())
	sheet.Set("ground", srgb.MustHex("#141218").Swatch())
	return hud.NewTheme(sheet)
}

// fail says what went wrong and stops.
func fail(err error) {
	fmt.Fprintln(os.Stderr, "slate:", err)
	os.Exit(1)
}
