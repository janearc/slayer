// Command mistlake is improv only: moonlit mist swirling on a lake, laid
// over a raw. The mist takes the moon's own colour from the raw's
// brightest pixels, is densest toward the far shore where a look runs
// through most of it, loosens and grows toward the camera, laps a little
// above the shore, and is brightest along the moon's column, where light
// scatters forward.
//
//	mistlake in.pfm out.pfm top bottom shore strength seed
//
// top, bottom and shore are degrees of elevation as mirrorlake takes
// them; strength is the mist's brightest share of the moon's.
package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/janearc/slayer/improv/moonlake/mist"
)

// mistColour is the moonlit mist's colour, from MIST_COLOUR as r,g,b, a
// cool night white unless told
var mistColour = colourFrom("MIST_COLOUR", [3]float64{0.78, 0.88, 1.08})

// mistThick is how many times as much mist hangs, from MIST_THICK, one
// unless told: n layers of it, so a pixel's mist d becomes 1 - (1 - d)^n.
// Thin mist fills in and the thickest stays the thickest, not brighter
// than the moon: from "ten times more mist", 2026-11-10.
var mistThick = numberFrom("MIST_THICK", 1)

// numberFrom is a number from the environment, or the default.
func numberFrom(key string, otherwise float64) float64 {
	raw := os.Getenv(key)
	if raw == "" {
		return otherwise
	}
	v, err := strconv.ParseFloat(raw, 64)
	if err != nil || v < 1 {
		fmt.Fprintf(os.Stderr, "%s=%q is not a number of at least one\n", key, raw)
		os.Exit(2)
	}
	return v
}

// colourFrom is a colour from the environment as r,g,b, or the default.
func colourFrom(key string, otherwise [3]float64) [3]float64 {
	raw := os.Getenv(key)
	if raw == "" {
		return otherwise
	}
	parts := strings.Split(raw, ",")
	if len(parts) != 3 {
		fmt.Fprintf(os.Stderr, "%s=%q is not r,g,b\n", key, raw)
		os.Exit(2)
	}
	var got [3]float64
	for c, part := range parts {
		v, err := strconv.ParseFloat(strings.TrimSpace(part), 64)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s=%q is not r,g,b\n", key, raw)
			os.Exit(2)
		}
		got[c] = v
	}
	return got
}

// main lays the mist over the lake and writes the raw.
func main() {
	if len(os.Args) != 9 {
		fmt.Fprintln(os.Stderr, "mistlake in.pfm out.pfm top bottom shore "+
			"strength seed seconds")
		os.Exit(2)
	}
	numbers := make([]float64, 6)
	for i := range numbers {
		v, err := strconv.ParseFloat(os.Args[3+i], 64)
		if err != nil {
			fail(err)
		}
		numbers[i] = v
	}
	image, err := mist.Read(os.Args[1])
	if err != nil {
		fail(err)
	}
	config := mist.Config{
		Top: numbers[0], Bottom: numbers[1], Shore: numbers[2], Strength: numbers[3],
		Seed: uint64(numbers[4]), Seconds: numbers[5], Colour: mistColour, Layers: mistThick,
	}
	for _, optional := range []struct {
		key    string
		target **mist.Image
	}{
		{"MOONREF", &config.MoonRef}, {"WATERMASK", &config.WaterMask}, {"TREEMASK", &config.TreeMask},
	} {
		if path := os.Getenv(optional.key); path != "" {
			loaded, err := mist.Read(path)
			if err != nil {
				fail(err)
			}
			*optional.target = &loaded
		}
	}
	out, err := mist.Apply(image, config)
	if err != nil {
		fail(err)
	}
	if err := mist.Write(os.Args[2], out); err != nil {
		fail(err)
	}
}

// fail says what went wrong and stops with a failing status.
func fail(err error) {
	fmt.Fprintln(os.Stderr, "mistlake:", err)
	os.Exit(1)
}
