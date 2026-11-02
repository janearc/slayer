// Command develop prints raws: it reads one or more .pfm raws, adds them light
// to light when there are several, which is a long exposure, and develops the
// total through the film curve at an exposure in stops.
//
//	develop [-exposure 0] [-white 0] [-knee 0.7] [-top 8] [-bloom 0]
//		-out x.png a.pfm ...
//
// The knee and top default to the orbital's own development, so a raw developed
// at zero stops prints as the orbital printed it.
package main

import (
	"flag"
	"fmt"
	"math"
	"os"

	"github.com/janearc/celestial/bands"
	"github.com/janearc/libtheme-css/primitives/swatch"
	"github.com/janearc/libtheme-css/spaces/srgb"
	"github.com/janearc/slayer/interfaces/file"
	"github.com/janearc/slayer/observer"
	"github.com/janearc/slayer/raw"
	"github.com/janearc/slayer/render/colour"
	"github.com/janearc/slayer/render/frame"
)

// main develops a picture from the flags; see the package comment.
func main() {
	stops := flag.Float64("exposure", 0, "stops of exposure; each one "+
		"doubles the light")
	knee := flag.Float64("knee", 0.7, "where the film's shoulder starts")
	top := flag.Float64("top", 8, "the brightest light the print holds, "+
		"in whites")
	bloom := flag.Float64("bloom", 0, "how many pixels light past white "+
		"spills into its neighbours")
	white := flag.Float64("white", 0, "white balance: the temperature, "+
		"in kelvin, of light that prints neutral; nought leaves it")
	out := flag.String("out", "", "where the print goes")
	flag.Parse()
	if *out == "" || flag.NArg() == 0 {
		fail(fmt.Errorf("develop wants -out and at least one raw"))
	}
	frames := make([]*frame.Frame, 0, flag.NArg())
	for _, path := range flag.Args() {
		read, err := raw.Read(path)
		if err != nil {
			fail(err)
		}
		frames = append(frames, read)
	}
	total, err := raw.Sum(frames)
	if err != nil {
		fail(err)
	}
	if *white != 0 {
		if err := balance(total, *white); err != nil {
			fail(err)
		}
	}
	tone := colour.Tone{Exposure: math.Exp2(*stops), Knee: *knee,
		Top: *top, Bloom: *bloom}
	if err := file.PNG(*out, tone.Develop(total), 1); err != nil {
		fail(err)
	}
	fmt.Printf("%s: %d raw%s at %+.1f stops\n", *out, len(frames),
		plural(len(frames)), *stops)
}

// balance scales a frame's red, green and blue so that a black body at
// the given temperature comes out neutral, its green kept as it is:
// what a camera's white balance does
func balance(total *frame.Frame, kelvin float64) error {
	if !(kelvin > 0) || math.IsInf(kelvin, 0) {
		return fmt.Errorf("a white balance of %v kelvin", kelvin)
	}
	spectrum := bands.Split(swatch.Blackbody(kelvin))
	red, green, blue := srgb.Light(observer.Swatch(spectrum))
	if !(red > 0 && blue > 0) {
		return fmt.Errorf("no white at %v kelvin", kelvin)
	}
	gains := [3]float32{float32(green / red), 1, float32(green / blue)}
	for i := range total.Pix {
		total.Pix[i].R *= gains[0]
		total.Pix[i].B *= gains[2]
	}
	return nil
}

// plural is the s a count wants
func plural(count int) string {
	if count == 1 {
		return ""
	}
	return "s"
}

// fail says what went wrong and stops
func fail(err error) {
	fmt.Fprintln(os.Stderr, "develop:", err)
	os.Exit(1)
}
