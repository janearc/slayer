// Package raw stores rendered linear RGB before development, preserving values
// above display white. Raw frames can be developed at a different exposure or
// summed to form a long exposure.
//
// Files use PFM: a three-line header followed by 32-bit RGB samples, bottom row
// first. Coverage and depth are not stored.
package raw

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"os"

	"github.com/janearc/slayer/render/frame"
)

const (
	// colour is PFM's mark for three channels a pixel
	colour = "PF"
	// littleScale is PFM's scale line for little-endian floats kept as
	// they are: its sign says the byte order, its size the scale
	littleScale = -1.0
	// channels is how many floats a pixel holds: red, green and blue
	channels = 3
)

// Write saves a frame’s RGB light to a PFM at path. NaN and infinite channel
// values are rejected.
func Write(path string, kept *frame.Frame) error {
	floats := make([]float32, 0, len(kept.Pix)*channels)
	for y := kept.H - 1; y >= 0; y-- {
		for x := 0; x < kept.W; x++ {
			pixel := kept.At(x, y)
			lights := []float32{pixel.R, pixel.G, pixel.B}
			for _, light := range lights {
				if !finite(light) {
					return refuse("light of %v at %d, %d",
						light, x, y)
				}
				floats = append(floats, light)
			}
		}
	}
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	written := bufio.NewWriter(file)
	fmt.Fprintf(written, "%s\n%d %d\n%v\n", colour, kept.W, kept.H,
		littleScale)
	err = binary.Write(written, binary.LittleEndian, floats)
	if err != nil {
		file.Close()
		return err
	}
	if err := written.Flush(); err != nil {
		file.Close()
		return err
	}
	return file.Close()
}

// Read loads a three-channel PFM from path. It rejects other PFM types and
// truncated pixel data.
func Read(path string) (*frame.Frame, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	reading := bufio.NewReader(file)
	var mark string
	var width, height int
	var scale float64
	if _, err := fmt.Fscanln(reading, &mark); err != nil || mark != colour {
		return nil, refuse("%s is not a colour PFM", path)
	}
	if _, err := fmt.Fscanln(reading, &width, &height); err != nil ||
		width < 1 || height < 1 {
		return nil, refuse("%s has no size", path)
	}
	if _, err := fmt.Fscanln(reading, &scale); err != nil || scale == 0 {
		return nil, refuse("%s has no scale", path)
	}
	return pixels(reading, path, width, height, scale)
}

// pixels reads bottom-up RGB samples after the header. The sign of the PFM
// scale selects byte order.
func pixels(reading io.Reader, path string, width, height int,
	scale float64) (*frame.Frame, error) {
	var order binary.ByteOrder = binary.BigEndian
	if scale < 0 {
		order = binary.LittleEndian
	}
	floats := make([]float32, width*height*channels)
	if err := binary.Read(reading, order, floats); err != nil {
		return nil, refuse("%s ends before its pixels do", path)
	}
	size := float32(math.Abs(scale))
	made := frame.New(width, height)
	for i := 0; i < width*height; i++ {
		x, y := i%width, height-1-i/width
		pixel := made.At(x, y)
		pixel.R = floats[i*channels] * size
		pixel.G = floats[i*channels+1] * size
		pixel.B = floats[i*channels+2] * size
		made.Set(x, y, pixel)
	}
	return made, nil
}

// Sum adds the linear RGB of equally sized frames, as a shutter integrates
// light over time. It rejects an empty input or mismatched dimensions.
func Sum(frames []*frame.Frame) (*frame.Frame, error) {
	if len(frames) == 0 {
		return nil, refuse("nothing to add up")
	}
	first := frames[0]
	total := frame.New(first.W, first.H)
	for _, each := range frames {
		if each.W != first.W || each.H != first.H {
			return nil, refuse("a %dx%d frame among %dx%d", each.W,
				each.H, first.W, first.H)
		}
		for i, pixel := range each.Pix {
			total.Pix[i].R += pixel.R
			total.Pix[i].G += pixel.G
			total.Pix[i].B += pixel.B
		}
	}
	return total, nil
}

// finite reports whether all RGB channels are finite.
func finite(light float32) bool {
	return !math.IsNaN(float64(light)) && !math.IsInf(float64(light), 0)
}

// refuse formats an error with the raw package prefix.
func refuse(format string, about ...any) error {
	return fmt.Errorf("raw: "+format, about...)
}
