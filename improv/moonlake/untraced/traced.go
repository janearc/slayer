package main

import (
	"fmt"
	"image"
	"math"
	"path/filepath"

	"github.com/janearc/slayer/improv/moonlake/rig"
	"github.com/janearc/slayer/raw"
	"github.com/janearc/slayer/render/colour"
	"github.com/janearc/slayer/render/frame"
)

// traced is a camera's traced passes at their own size, linear light,
// three channels a pixel: the plate (the set moonlit, its disc off),
// and each fixture's pass, by name. A moment's picture from that camera
// is the plate dimmed by the master plus each pass at its cue's level,
// as lightshow sums them (package rig): rendered, not drawn. Any part
// of it may be shown, at any size, so a pane's window can move.
type traced struct {
	size   image.Point
	plate  []float64
	passes map[string][]float64
}

// The passes a traced pane is summed from, the 4k recipe's names. The
// plate is "nodisc". From overnight.sh's pass list.
var passNames = []string{"nodisc", "back-left", "back-right", "overhead"}

// The film's gains and print, from the 4k recipe's environment: the
// floods' FLOOD_GAIN (lightshow's default, unset there), OVERHEAD_GAIN,
// the moon's 23 stops of exposure the cue's exposure moves, the print's
// knee and top (develop's defaults), and BLOOM, 0.8 pixels at 2048
// across, grown with the width.
const (
	floodGain     = 400.0
	overheadGain  = 0.002
	moonStops     = 23.0
	printKnee     = 0.7
	printTop      = 8.0
	bloomAt2048   = 0.8
	bloomWidth    = 2048.0
	tracedChannel = 3
)

// floodPasses are the passes the floods fire, left then right: the
// film's FLOOD_PASSES.
var floodPasses = [2]string{"back-left", "back-right"}

// tracedOf reads a camera's passes, all of one size.
func tracedOf(dir, camera string) (traced, error) {
	made := traced{passes: map[string][]float64{}}
	for _, name := range passNames {
		read, err := raw.Read(filepath.Join(dir, camera+"-"+name+".pfm"))
		if err != nil {
			return traced{}, err
		}
		size := image.Pt(read.W, read.H)
		if made.size != (image.Point{}) && size != made.size {
			return traced{}, fmt.Errorf("the pass %s-%s is %v, the plate "+
				"%v; trace them at one size", camera, name, size, made.size)
		}
		made.size = size
		light := make([]float64, read.W*read.H*tracedChannel)
		for i, p := range read.Pix {
			light[i*tracedChannel] = float64(p.R)
			light[i*tracedChannel+1] = float64(p.G)
			light[i*tracedChannel+2] = float64(p.B)
		}
		if name == passNames[0] {
			made.plate = light
			continue
		}
		made.passes[name] = light
	}
	return made, nil
}

// averaged is a part of light of a size averaged down to another size:
// each pixel the mean of the pixels it covers.
func averaged(light []float64, from image.Point, part image.Rectangle,
	size image.Point) []float64 {
	out := make([]float64, size.X*size.Y*tracedChannel)
	scale := float64(part.Dx()) / float64(size.X)
	for y := range size.Y {
		y0 := part.Min.Y + int(float64(y)*scale)
		y1 := max(y0+1, part.Min.Y+int(float64(y+1)*scale))
		for x := range size.X {
			x0 := part.Min.X + int(float64(x)*scale)
			x1 := max(x0+1, part.Min.X+int(float64(x+1)*scale))
			var sum [tracedChannel]float64
			for sy := max(y0, 0); sy < min(y1, from.Y); sy++ {
				for sx := max(x0, 0); sx < min(x1, from.X); sx++ {
					at := (sy*from.X + sx) * tracedChannel
					sum[0], sum[1], sum[2] = sum[0]+light[at],
						sum[1]+light[at+1], sum[2]+light[at+2]
				}
			}
			count := float64((x1 - x0) * (y1 - y0))
			for c := range sum {
				out[(y*size.X+x)*tracedChannel+c] = sum[c] / count
			}
		}
	}
	return out
}

// at is the camera's rendered picture of a part of it at a size, at a
// moment's cues: the plate at the master, the floods' and the
// overhead's passes at their levels, developed as the film was, at the
// moon's stops moved by the cue's.
func (made traced) at(state cues, part image.Rectangle,
	size image.Point) (*image.RGBA, error) {
	if !part.In(image.Rectangle{Max: made.size}) {
		return nil, fmt.Errorf("a window %v on passes %v; keep it on them",
			part, made.size)
	}
	light := averaged(made.plate, made.size, part, size)
	for i := range light {
		light[i] *= state.master
	}
	layers := append(rig.Floods(floodPasses, state.floodLeft,
		state.floodRight, floodGain), rig.Overhead(state.overhead,
		overheadGain))
	for _, layer := range layers {
		pass, found := made.passes[layer.Name]
		if layer.Amount == 0 {
			continue
		}
		if !found || len(pass) != len(made.plate) {
			return nil, fmt.Errorf("the pass %s is missing or not the "+
				"plate's size; trace it with the plate", layer.Name)
		}
		if err := layer.Add(light, averaged(pass, made.size, part,
			size)); err != nil {
			return nil, err
		}
	}
	print := frame.New(size.X, size.Y)
	for i := range print.Pix {
		print.Pix[i].R = float32(light[i*tracedChannel])
		print.Pix[i].G = float32(light[i*tracedChannel+1])
		print.Pix[i].B = float32(light[i*tracedChannel+2])
	}
	tone := colour.Tone{Exposure: math.Exp2(moonStops + state.exposure),
		Knee: printKnee, Top: printTop,
		Bloom: bloomAt2048 * float64(size.X) / bloomWidth}
	return tone.Develop(print), nil
}
