// Command ring draws a ring of triangles round a magnetar: a band, mirror
// on its inner face and dark on its outer, the kind of thing the culture
// would hang round a flaring star for the fun of it. Its first job is to
// put slayer's meshes (mesh) in a picture; there is no smoke of them yet.
//
//	ring [-at 1.2s] [-w 960] [-h 540] [-n 2] [-around 1024]
//	     [-out out/ring.png]
//
// The magnetar is scene/bodies/template's, sgr 1806-20's, placed at a
// thousandth of a unit a kilometre, so the ring is 10,000 km across in the
// picture's scale and the flare's fireball, leaving at a tenth of light
// speed a second in, reaches it about a third of a second later.
//
// The scene itself is scene/named's ring; this command parses its flags,
// asks for it, and draws it, with the false-colour picture beside it
// when -bands asks.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/interfaces/file"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/render/colour"
	"github.com/janearc/slayer/scene/named"
	"github.com/janearc/slayer/trace"
)

// main draws the ring from the flags and writes the picture.
func main() {
	at := flag.Duration("at", 1200*time.Millisecond, "the moment, from the magnetar's zero; the flare leaves at 1s")
	w := flag.Int("w", 960, "width in pixels")
	h := flag.Int("h", 540, "height in pixels")
	n := flag.Int("n", 2, "rays per pixel along each side")
	around := flag.Int("around", 1024, "steps round the ring; fewer shows the triangles in the mirror")
	out := flag.String("out", "out/ring.png", "where the picture goes")
	falseColour := flag.Bool("bands", false, "also a false-colour picture beside it: x-ray green, gamma violet, infrared deep red, over the visible")
	visible := flag.Float64("visible", 0.6, "how much of the visible goes under the bands")
	radiation := flag.Bool("radiation", false, "let libtheme's radiation colour every band, instead of the tints here")
	sun := flag.Bool("sun", true, "the small far sun that gives the hull a shape")
	dress := flag.Bool("dress", false, "dress the inner face in hexes, in the earth's paints, and light it")
	night := flag.Bool("night", false, "with -dress, put the ring's sun out: only what glows shows, the cities and the lava")
	flag.Parse()

	scene, err := named.Get("ring")
	if err != nil {
		fmt.Fprintln(os.Stderr, "ring:", err)
		os.Exit(1)
	}
	built, err := scene.Build(map[string]any{"around": *around,
		"sun": *sun, "dress": *dress, "night": *night}, os.Stdout)
	if err != nil {
		fmt.Fprintln(os.Stderr, "ring:", err)
		os.Exit(1)
	}
	world, eye := built.World(), built.Eye()
	lens := film.Pinhole{Wide: scene.Defaults().Wide()}
	start := time.Now()
	f, err := trace.Sampled(world, eye, lens, *at, *w, *h, *n)
	if err != nil {
		fmt.Fprintln(os.Stderr, "ring:", err)
		os.Exit(1)
	}
	if err := os.MkdirAll(filepath.Dir(*out), 0o755); err != nil {
		fmt.Fprintln(os.Stderr, "ring:", err)
		os.Exit(1)
	}
	developed := scene.Defaults().Tone().Develop(f)
	if err := file.PNG(*out, developed, 1); err != nil {
		fmt.Fprintln(os.Stderr, "ring:", err)
		os.Exit(1)
	}
	fmt.Printf("%s at %v: %dx%d, %d rays a pixel, %v\n", *out, *at, *w, *h,
		*n**n, time.Since(start).Round(time.Second))
	if !*falseColour {
		return
	}
	// each band as a sensor for it sees it: nothing lit by lamps, only what
	// gives it off, through what lets it pass. the ring has no x-ray or
	// gamma of its own, so in those it is a dark shape: a mirror facing
	// you returns neither, which only glance off at grazing angles
	start = time.Now()
	layers := []colour.Layer{{Frame: f, Gain: *visible}}
	for _, b := range bands {
		fr, err := trace.RenderIn(world, eye, lens, *at, *w, *h, b.band)
		if err != nil {
			fmt.Fprintln(os.Stderr, "ring:", err)
			os.Exit(1)
		}
		brightest, lit := float32(0), 0
		for _, p := range fr.Pix {
			if p.R > 0 {
				lit++
			}
			brightest = max(brightest, p.R)
		}
		// stretched to its own range, as every false-colour picture is:
		// the band's brightest pixel set most of the way to white
		gain := 1.0
		if brightest > 0 {
			gain = 0.8 / float64(brightest)
		}
		as := b.as
		if *radiation {
			as = "libtheme's own colour"
		}
		fmt.Printf("%s as %s: %d pixels give any, the brightest %.3g, stretched %.3g times\n",
			b.name, as, lit, brightest, gain)
		tint := b.tint
		if *radiation {
			// no tint: the composite asks libtheme's radiation for
			// the band's own stand-in colour, by its wavelength
			tint = maths.Light{}
		}
		layers = append(layers, colour.Layer{Band: b.band, Frame: fr, Gain: gain, Tint: tint})
	}
	fc := scene.Defaults().Tone().False(layers)
	name := strings.TrimSuffix(*out, filepath.Ext(*out)) + "-bands.png"
	if err := file.PNG(name, fc, 1); err != nil {
		fmt.Fprintln(os.Stderr, "ring:", err)
		os.Exit(1)
	}
	fmt.Printf("%s: the visible and %d bands, %v\n", name, len(bands), time.Since(start).Round(time.Second))
}

// bands are the false-colour picture's layers past the visible, each with
// the colour it is painted: the hot, high-energy light in vivid colours, and
// the warm, in a deep red.
var bands = []struct {
	name, as string
	band     maths.Band
	tint     maths.Light
}{
	{"x-ray", "green", maths.Band{Nanometres: 1}, maths.RGB(0.25, 1, 0.3)},
	{"gamma ray", "violet", maths.Band{Nanometres: 1e-4}, maths.RGB(0.65, 0.3, 1)},
	{"infrared", "deep red", maths.Band{Nanometres: 10000}, maths.RGB(0.75, 0.04, 0.03)},
}
