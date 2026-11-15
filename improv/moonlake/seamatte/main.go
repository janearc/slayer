// Command seamatte is improv only: below the far shore, a raw becomes
// oceantex's water, brought back into the raw's units by the moon: the
// water's units are the moon's irradiance, and its sky was the raw over
// the moon disc's own light, so each channel is multiplied back by the
// moon disc's light and its solid angle.
//
//	seamatte raw.pfm sea.pfm out.pfm top bottom shore moonradius
//
// top, bottom and shore are degrees of elevation; moonradius is the
// moon disc's in degrees.
package main

import (
	"fmt"
	"os"
	"strconv"

	"github.com/janearc/slayer/improv/moonlake/matte"
)

// main lays the water in below the far shore and what stands in front of
// it over that, and writes the raw.
func main() {
	if len(os.Args) != 8 {
		fmt.Fprintln(os.Stderr, "seamatte raw.pfm sea.pfm out.pfm top bottom shore moonradius")
		os.Exit(2)
	}
	numbers := make([]float64, 4)
	for i := range numbers {
		v, err := strconv.ParseFloat(os.Args[4+i], 64)
		if err != nil {
			fail(err)
		}
		numbers[i] = v
	}
	raw, err := matte.Read(os.Args[1])
	if err != nil {
		fail(err)
	}
	sea, err := matte.Read(os.Args[2])
	if err != nil {
		fail(err)
	}
	if sea.Width != raw.Width || sea.Height != raw.Height {
		fail(fmt.Errorf("the raw is %dx%d and the sea %dx%d", raw.Width, raw.Height, sea.Width, sea.Height))
	}
	config := matte.Config{Top: numbers[0], Bottom: numbers[1], Shore: numbers[2], MoonRadius: numbers[3]}
	for _, input := range []struct {
		name   string
		target **matte.Image
	}{
		{"MOONREF", &config.MoonRef},
		{"WATERMASK", &config.WaterMask},
		{"FRONT", &config.Front},
	} {
		if path := os.Getenv(input.name); path != "" {
			image, err := matte.Read(path)
			if err != nil {
				fail(err)
			}
			*input.target = &image
		}
	}
	out, err := matte.Apply(raw, sea, config)
	if err != nil {
		fail(err)
	}
	if err := matte.Write(os.Args[3], out); err != nil {
		fail(err)
	}
}

// fail says what went wrong and stops with a failing status, so a
// script running the steps stops where one breaks.
func fail(err error) {
	fmt.Fprintln(os.Stderr, "seamatte:", err)
	os.Exit(1)
}
