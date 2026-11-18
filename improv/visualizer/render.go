package main

import (
	"bufio"
	"fmt"
	"image"
	"math"
	"strconv"

	"github.com/janearc/puzzlemouth/pzm"
)

// smallest is the least width or height a frame may be, in pixels: under
// it the lanes' names do not fit.
const smallest = 480

// tail is the seconds drawn past the score's end when no length is
// given, for the last notes' release.
const tail = 2.0

// bufferSize is the output buffer, a mebibyte.
const bufferSize = 1 << 20

// render reads the lanes and writes every frame of the film to stdout.
func render(call pzm.Call) error {
	lanes, score, err := readLanes(call.Args())
	if err != nil {
		return err
	}
	fps, err1 := strconv.Atoi(call.Value("fps"))
	width, err2 := strconv.Atoi(call.Value("width"))
	height, err3 := strconv.Atoi(call.Value("height"))
	if err1 != nil || err2 != nil || err3 != nil || fps < 1 ||
		width < smallest || height < smallest {
		return fmt.Errorf("positive fps and at least %d by %d required",
			smallest, smallest)
	}
	seconds, err := length(call.Value("seconds"), score.Seconds()+tail)
	if err != nil {
		return err
	}
	if call.DryRun() {
		return nil
	}
	f, err := newFilm(score, lanes, width, height)
	if err != nil {
		return err
	}
	return f.write(call, fps, int(math.Ceil(seconds*float64(fps))))
}

// length is the film's length: the flag's, or the fallback if empty.
func length(flag string, fallback float64) (float64, error) {
	if flag == "" {
		return fallback, nil
	}
	seconds, err := strconv.ParseFloat(flag, 64)
	if err != nil || seconds <= 0 {
		return 0, fmt.Errorf("seconds %q: want a positive number", flag)
	}
	return seconds, nil
}

// write draws the frames and writes them as rgb24.
func (f *film) write(call pzm.Call, fps, frames int) error {
	out := bufio.NewWriterSize(call.Out(), bufferSize)
	img := image.NewRGBA(f.plate.Bounds())
	pixels := make([]byte, len(img.Pix)/4*3)
	for n := range frames {
		f.frame(img, float64(n)/float64(fps))
		for i := range len(img.Pix) / 4 {
			copy(pixels[3*i:3*i+3], img.Pix[4*i:4*i+3])
		}
		if _, err := out.Write(pixels); err != nil {
			return err
		}
	}
	return out.Flush()
}
