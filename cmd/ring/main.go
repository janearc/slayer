// Command ring draws a ring of triangles round a magnetar: a band, mirror
// on its inner face and dark on its outer, the kind of thing the culture
// would hang round a flaring star for the fun of it. Its first job is to
// put slayer's meshes (mesh) in a picture; there is no smoke of them yet.
//
//	ring [-at 1.2s] [-w 960] [-h 540] [-n 2] [-around 96]
//	     [-out out/ring.png]
//
// The magnetar is scene/bodies/template's, sgr 1806-20's, placed at a
// thousandth of a unit a kilometre, so the ring is 10,000 km across in the
// picture's scale and the flare's fireball, leaving at a tenth of light
// speed a second in, reaches it about a third of a second later.
package main

import (
	"flag"
	"fmt"
	"github.com/janearc/celestial/radiation"
	"math"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/interfaces/file"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/hexes"
	"github.com/janearc/slayer/mesh"
	"github.com/janearc/slayer/render/colour"
	"github.com/janearc/slayer/render/trace"
	"github.com/janearc/slayer/scene/bodies/template"
	"github.com/janearc/slayer/scene/sky"
	"github.com/janearc/slayer/scene/texture"
	"github.com/janearc/slayer/scene/thing"

	"github.com/janearc/slayer/biome"
	"github.com/janearc/slayer/cast"
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

	world, eye, err := scene(*around, *sun, *dress, *night)
	if err != nil {
		fmt.Fprintln(os.Stderr, "ring:", err)
		os.Exit(1)
	}
	start := time.Now()
	f := trace.Sampled(world, eye, film.Pinhole{Wide: 55 * maths.Degree}, *at, *w, *h, *n)
	if err := os.MkdirAll(filepath.Dir(*out), 0o755); err != nil {
		fmt.Fprintln(os.Stderr, "ring:", err)
		os.Exit(1)
	}
	if err := file.PNG(*out, colour.Tone{Knee: 0.7, Top: 8}.Develop(f), 1); err != nil {
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
		fr := trace.RenderIn(world, eye, film.Pinhole{Wide: 55 * maths.Degree}, *at, *w, *h, b.band)
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
	fc := colour.Tone{Knee: 0.7, Top: 8}.False(layers)
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

// the ring, in the picture's units: 10,000 km across at a thousandth of a
// unit a kilometre, four units wide and a fifth of a unit thick
const (
	radius = 10.0
	width  = 4.0
	thick  = 0.2
	rows   = 4
	scale  = 1e-3
)

// scene is the magnetar at the middle, the ring round it with its axis
// up, the stars, and the eye outside and above, looking in across the
// near rim at the mirror on the far side.
func scene(around int, sun, dress, night bool) (*thing.World, maths.Subspace, error) {
	w := thing.NewWorld()
	w.Sky = sky.Stars{Dark: maths.RGB(0.004, 0.004, 0.008), Seed: 1806}
	template.Magnetar().Place(w, maths.V(0, 0, 0), scale, 4)

	if dress {
		// the inner face alone, dressed: hexes round the inside in the
		// earth's paints, with a little of the mirror left on top so
		// the flare still shimmers across them
		//
		// cast from the hex lattice itself, centre to centre, so the
		// ground and its tiles agree; the land rises toward the axis by
		// its height above the sea, the sea lies flat, and the paint is
		// the same two fields read at each tile, so the biomes clump
		tiles := hexes.Band{Radius: radius - thick/2, Width: width,
			Around: 360}
		fields := biome.Fields{Tiling: tiles, Seed: 1806, Scale: 2.2, Sea: 0.02}
		world := biome.Make(fields)
		fmt.Printf("the world: %d rivers, %d lakes, %d volcanoes, %d cities\n",
			world.Rivers, world.Lakes, world.Volcanoes, world.Cities)
		inward := func(p maths.Vec) maths.Vec { return maths.V(-p.X, 0, -p.Z) }
		rise := func(_ hexes.Tile, c maths.Vec) float64 {
			return 0.35 * fields.Above(c)
		}
		floor, n, err := cast.Mesh(tiles, inward, rise, true)
		if err != nil {
			return nil, maths.Subspace{}, err
		}
		fmt.Printf("the floor: %d tiles cast into %d triangles\n",
			len(tiles.Tiles()), n)
		w.Add(thing.Thing{Shape: floor, Material: &thing.Material{
			Name:    "ring floor",
			Texture: texture.Tiled{Tiling: tiles, Paints: world, Lines: 0.2},
			Shine:   texture.Tiled{Tiling: tiles, Paints: world.Shine()},
			Gloss:   0.1, Index: 1.5}})
	}
	skin, err := band(around, func(s int) bool { return !dress || s != 0 })
	if err != nil {
		return nil, maths.Subspace{}, err
	}
	w.Add(thing.Thing{
		Shape: skin,
		// one material for the whole solid: a mirror, so the inner
		// face throws the flare round, and the outer one shows it too,
		// faintly
		Material: &thing.Material{Name: "ring",
			Colour: maths.RGB(0.06, 0.06, 0.08),
			Gloss:  1, Index: 8,
			// warm from the flare: a hull beside a giant flare
			// heats, and a warm thing glows in the infrared. set
			// by hand, not worked out from a temperature, since
			// nothing in slayer does that yet
			Radiant: &thing.Spectrum{Gives: map[radiation.Range]float64{radiation.Infrared(): 0.3}}},
	})
	// a far, small sun, so the hull has a shape where nothing mirrors;
	// dressed, the ring's own sun, low and behind the eye, so the far inner
	// face it is lit, and a dim dome of sky to fill the shadows
	switch {
	case dress && night:
		// night: a whisper of starlight, so the coasts read by their
		// lights
		w.Lamp(film.Flood{Strength: maths.RGB(0.01, 0.012, 0.02)})
	case dress:
		w.Lamp(thing.Sun(maths.V(0.25, 0.45, 1), maths.RGB(1.1, 1.05, 0.95)))
		w.Lamp(film.Flood{Strength: maths.RGB(0.08, 0.09, 0.12)})
	case sun:
		w.Lamp(thing.Sun(maths.V(-0.4, 1, 0.3), maths.RGB(0.35, 0.33, 0.3)))
	}

	from := maths.V(0, 9, 26)
	return w, maths.Subspace{Origin: from,
		Orientation: maths.LookAt(from, maths.V(0, -1, 0))}, nil
}

// band is the ring as a solid of triangles, in four strips round the axis: the
// inner face, the top rim, the outer face and the bottom rim, each cut into
// rows across and stepped round. A strip has its own vertices, so its normals
// are its own:
//
// smooth round the ring, so the mirror bends each reflection as a curved
// surface would, and hard at the corners, where one face meets the next. Every
// triangle is turned to face away from the middle of the cross-section, so the
// solid is wound outward throughout.
func band(around int, keep func(strip int) bool) (*mesh.Mesh, error) {
	if around < 8 {
		around = 8
	}
	inner, outer := radius-thick/2, radius+thick/2
	// a strip is a line across the cross-section, as radius and height,
	// and the way its face looks in the same terms
	type rz struct{ r, y float64 }
	strip := func(from, to rz, n int) []rz {
		var s []rz
		for j := 0; j <= n; j++ {
			f := float64(j) / float64(n)
			s = append(s, rz{from.r + (to.r-from.r)*f, from.y + (to.y-from.y)*f})
		}
		return s
	}
	lo, hi := -width/2, width/2
	strips := []struct {
		line   []rz
		facing rz
	}{
		{strip(rz{inner, lo}, rz{inner, hi}, rows), rz{-1, 0}},
		{strip(rz{inner, hi}, rz{outer, hi}, 1), rz{0, 1}},
		{strip(rz{outer, hi}, rz{outer, lo}, rows), rz{1, 0}},
		{strip(rz{outer, lo}, rz{inner, lo}, 1), rz{0, -1}},
	}
	var verts, normals []maths.Vec
	var faces [][3]int
	for n, s := range strips {
		if !keep(n) {
			continue
		}
		base, k := len(verts), len(s.line)
		for i := range around {
			a := 2 * math.Pi * float64(i) / float64(around)
			c, sn := math.Cos(a), math.Sin(a)
			for _, p := range s.line {
				verts = append(verts, maths.V(p.r*c, p.y, p.r*sn))
				normals = append(normals, maths.V(s.facing.r*c, s.facing.y, s.facing.r*sn))
			}
		}
		at := func(i, j int) int { return base + (i%around)*k + j }
		for i := range around {
			a := 2 * math.Pi * (float64(i) + 0.5) / float64(around)
			mid := maths.V(radius*math.Cos(a), 0, radius*math.Sin(a))
			for j := 0; j+1 < k; j++ {
				q := [4]int{at(i, j), at(i+1, j), at(i+1, j+1), at(i, j+1)}
				for _, t := range [][3]int{{q[0], q[1], q[2]}, {q[0], q[2], q[3]}} {
					p0, p1, p2 := verts[t[0]], verts[t[1]], verts[t[2]]
					normal := p1.Sub(p0).Cross(p2.Sub(p0))
					centre := p0.Add(p1).Add(p2).Scale(1.0 / 3)
					if normal.Dot(centre.Sub(mid)) < 0 {
						t[1], t[2] = t[2], t[1]
					}
					faces = append(faces, t)
				}
			}
		}
	}
	return mesh.New(verts, faces, normals, true)
}
