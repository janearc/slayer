// Command orbital draws a pair of culture orbitals round a magnetar: two bands
// side by side on one axis, spinning opposite ways so their turning cancels, a
// gap between them where lights run along the walls and spill out onto the
// hulls.
//
// Each is the size a culture orbital is for one g and a day of 24 hours, 1.84
// million km in radius, and a thread at that size: 32,000 km wide, with walls a
// thousand high. The eye is inside, above the gap, looking along the floors as
// they curve up into an arch.
//
//	orbital [-w 1600] [-h 900] [-n 2] [-out out/orbital.png]
//	orbital -cells COLSxROWS [-frames 0] [-fps 24] [-n 2] [-far] [-keys]
//
// With -cells it writes no file. It writes the scene to standard output as
// frames of ANSI text, sextant cells in 24-bit colour, one frame after
// another as the scene turns. -frames 0 runs until the reader goes away.
//
// It asks the terminal nothing and sets no mode, so a window can show it.
// What it says of itself goes to standard error.
//
// With -keys as well it does ask: it takes the terminal over, draws a
// frame, and draws the next at each key. q, escape and ctrl-c stop it, and
// after the last frame any key does.
//
// SIGTERM, SIGINT and SIGHUP give the terminal back and exit 128 plus the
// signal. The frames are the same cells, at the same times, as the
// stream's.
//
// One unit is a thousand kilometres.
package main

import (
	"flag"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/interfaces/file"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/hexes"
	"github.com/janearc/slayer/maths/noise"
	"github.com/janearc/slayer/mesh"
	"github.com/janearc/slayer/raw"
	"github.com/janearc/slayer/scene/sky"
	"github.com/janearc/slayer/scene/texture"
	"github.com/janearc/slayer/scene/thing"
	"github.com/janearc/slayer/trace"

	"github.com/janearc/celestial/bands"
	"github.com/janearc/libtheme-css/primitives/swatch"
	"github.com/janearc/libtheme-css/spaces/srgb"
	"github.com/janearc/slayer/biome"
	"github.com/janearc/slayer/emit"
	"github.com/janearc/slayer/lantern"
	"github.com/janearc/slayer/lines"
	"github.com/janearc/slayer/observer"
)

// the orbital, in thousands of kilometres:
//
// the radius for one g at a 24 hour turn, T = 2 pi root(R/g), a band 192 wide
// (six times the first, her calls of 2026-10-04, so the pair reads through the
// gas) with walls one high and a hull a tenth thick, and the gap between the
// pair, a quarter of a band, so they read as two
const (
	radius = 1840.0
	width  = 192.0
	wall   = 1.0
	thick  = 0.1
	gap    = 48.0
	around = 6144
	// the hexes are about a unit and a half across
	hexAround = 7700
)

// main draws the orbital scene from the flags and writes the picture.
func main() {
	w := flag.Int("w", 1600, "width in pixels")
	h := flag.Int("h", 900, "height in pixels")
	n := flag.Int("n", 2, "rays per pixel along each side")
	out := flag.String("out", "out/orbital.png", "where the picture goes")
	far := flag.Bool("far", false, "from far outside, 25 degrees above "+
		"their plane, the hoops standing upright, lit by the lantern")
	away := flag.Float64("from", 3600,
		"with -far, how far out the eye is, in thousands of km")
	frames := flag.Int("frames", 1, "how many frames; more than one "+
		"writes out-NNNN.png")
	rate := flag.Int("fps", 24, "frames a second")
	first := flag.Int("first", 0, "the frame to start at, to resume a film")
	still := flag.Bool("still", false, "hold the lantern's rings still")
	thicker := flag.Float64("gas", 1, "how many times thicker the nebula")
	neon := flag.Float64("neon", 0, "the share of the nebula that is "+
		"neon, nought to one")
	argon := flag.Float64("argon", 0, "the share of the nebula that is "+
		"argon, ionized; what neon and argon leave is hydrogen")
	glow := flag.Float64("glow", 0.2, "how bright the nebula's glow is, "+
		"as a luminance")
	keep := flag.Bool("raw", false, "keep the raw beside each png, as a "+
		".pfm: its light before it was developed")
	stops := flag.Float64("exposure", 0, "stops of exposure to develop "+
		"at; each one doubles the light")
	eyeAt := flag.String("eye", "", "where the eye is, as x,y,z in "+
		"thousands of km from the lantern; empty keeps the view's own")
	lookAt := flag.String("look", "0,0,0", "with -eye, what it looks at")
	upward := flag.String("up", "1,0,0", "with -eye, which way is up")
	wide := flag.Float64("wide", 90, "how wide the pinhole sees, "+
		"in degrees across")
	day := flag.Duration("day", 24*time.Hour, "how long one band takes "+
		"to turn; the pair turn opposite ways; nought holds them still")
	toCells := flag.String("cells", "", "COLSxROWS: write the scene to "+
		"standard output as frames of terminal cells, and no file; "+
		"-frames 0 has no end")
	keys := flag.Bool("keys", false, "with -cells, draw a frame "+
		"and wait for a key: q, escape or ctrl-c stop, any other "+
		"key draws the next; after the last frame any key ends it; "+
		"needs a terminal; -frames 0 has no end")
	flag.Parse()

	asked, err := askCells(cellsFlags{size: *toCells, set: givenFlags(),
		first: *first, frames: *frames, rate: *rate, rays: *n,
		printed: printing{stops: *stops, raw: *keep,
			wide: *wide * maths.Degree},
		keys: *keys, in: os.Stdin, out: os.Stdout})
	if err != nil {
		fail(err)
	}
	gas := nebula{thick: 4.5e-5 * *thicker, neon: *neon, argon: *argon,
		glow: *glow}
	world, eye, err := scene(setting{far: *far, away: *away, still: *still,
		gas: gas, day: *day, chat: asked.chat()})
	if err != nil {
		fail(err)
	}
	if *eyeAt != "" {
		if eye, err = placed(*eyeAt, *lookAt, *upward); err != nil {
			fail(err)
		}
	}
	if asked.on {
		if err := asked.run(os.Stdout, world, eye); err != nil {
			fail(err)
		}
		return
	}
	for frame := *first; frame < *first+*frames; frame++ {
		at := 500*time.Millisecond +
			time.Duration(frame)*time.Second/time.Duration(*rate)
		path := *out
		if *frames > 1 || *first > 0 {
			path = strings.TrimSuffix(*out, ".png") +
				fmt.Sprintf("-%04d.png", frame)
		}
		printed := printing{stops: *stops, raw: *keep,
			wide: *wide * maths.Degree}
		if err := draw(world, eye, at, *w, *h, *n, path,
			printed); err != nil {
			fail(err)
		}
	}
}

// placed is an eye at one point looking at another, its up as given,
// each written x,y,z in thousands of km
func placed(at, toward, up string) (maths.Subspace, error) {
	points := make([]maths.Vec, 0, 3)
	for _, written := range []string{at, toward, up} {
		point, err := point(written)
		if err != nil {
			return maths.Subspace{}, err
		}
		points = append(points, point)
	}
	return maths.Subspace{Origin: points[0], Orientation: maths.LookAtUp(
		points[0], points[1], points[2])}, nil
}

// point is a point written x,y,z
func point(written string) (maths.Vec, error) {
	parts := strings.Split(written, ",")
	if len(parts) != 3 {
		return maths.Vec{}, fmt.Errorf("%q is not x,y,z", written)
	}
	var axes [3]float64
	for i, part := range parts {
		number, err := strconv.ParseFloat(strings.TrimSpace(part), 64)
		if err != nil {
			return maths.Vec{}, fmt.Errorf("%q is not x,y,z",
				written)
		}
		axes[i] = number
	}
	return maths.V(axes[0], axes[1], axes[2]), nil
}

// fail says what went wrong and stops
func fail(err error) {
	fmt.Fprintln(os.Stderr, "orbital:", err)
	os.Exit(1)
}

// beamClock is the time the lantern's beams are drawn at: a thing.Lamp
// has no time, so draw sets it before every frame
var beamClock time.Duration

// printing is how a frame is printed: how many stops of exposure it is
// developed at, and whether its raw is kept beside it
type printing struct {
	stops float64
	raw   bool
	wide  float64
}

// draw is one frame at a time, written as a png, and its raw too when
// printed says to keep it
func draw(world *thing.World, eye maths.Subspace, at time.Duration,
	width, height, side int, path string, printed printing) error {
	start := time.Now()
	beamClock = at
	lens := lensFor(printed, square)
	drawn, err := trace.Sampled(world, eye, lens, at, width, height, side)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	if printed.raw {
		kept := strings.TrimSuffix(path, ".png") + ".pfm"
		if err := raw.Write(kept, drawn); err != nil {
			return err
		}
	}
	developed := developer(printed).Develop(drawn)
	if err := file.PNG(path, developed, 1); err != nil {
		return err
	}
	fmt.Printf("%s: %dx%d, %d rays a pixel, %v\n", path, width, height,
		side*side, time.Since(start).Round(time.Second))
	return nil
}

// setting is what the flags choose about the scene
type setting struct {
	far   bool
	away  float64
	still bool
	gas   nebula
	// day is how long one band takes to turn once; the two turn opposite
	// ways, and zero holds them still
	day time.Duration
	// chat is where the scene says what it has built
	chat io.Writer
}

// scene builds the world for a setting, and the eye that looks at it.
func scene(set setting) (*thing.World, maths.Subspace, error) {
	w := thing.NewWorld()
	w.Sky = sky.Stars{Dark: maths.RGB(0.003, 0.003, 0.006), Seed: 1806}
	// the middle is the far view's lantern; the magnetar that stood
	// there, its flare a second in, comes back when it is the lantern's
	// own light (doc/emitters.md)

	// the two walls of each band: a black marble, dark, its cities
	// gathered in clusters and lit
	gapWall := &thing.Material{Name: "gap wall",
		Colour: maths.RGB(0.03, 0.03, 0.035), Shine: marble{seed: 1806}}
	pair := nested()
	for k, one := range pair {
		if err := one.place(w, k, gapWall, set.day,
			set.chat); err != nil {
			return nil, maths.Subspace{}, err
		}
	}
	// the nebula round it all, glowing hydrogen red and neon orange and
	// scattering the sun, with the channel the pair has swept through it
	gasMaterial, err := set.gas.material()
	if err != nil {
		return nil, maths.Subspace{}, err
	}
	// the channel the pair swept is twice a band's width about each
	// band's path
	w.AddVolume(thing.Volume{Medium: wake{reach: 5200, clear: 2 * width,
		thick: set.gas.thick, paths: pathsOf(pair)},
		Material: gasMaterial})
	// a far sun, so the floor near the eye is in day and, further round,
	// the arch turns to night and its cities light
	if set.far {
		return farView(w, set.away, set.still)
	}
	w.Lamp(thing.Sun(maths.V(0.45, 0.3, -0.85), maths.RGB(1.2, 1.15, 1.05)))
	w.Lamp(film.Flood{Strength: maths.RGB(0.03, 0.035, 0.05)})

	// the eye: inside, 500 km above one floor near the gap, its up
	// toward the axis as a person's on the floor would be, looking along
	// the floors and a little up, so they run away ahead and rise into
	// the arch
	from := maths.V(0, -3.0, radius-0.5)
	look := maths.V(math.Cos(0.2), 0.12, -math.Sin(0.2))
	to := from.Add(look.Scale(1000))
	return w, maths.Subspace{Origin: from,
		Orientation: maths.LookAtUp(from, to, maths.V(0, 0, -1))}, nil
}

// urbanLights are a city's from its edges in to its heart: sodium amber
// at the ends of the arms, warm white, and a dense blue-white in the
// middle, as black bodies through libtheme at one brightness
var urbanLights = warmLights(1900, 2600, 3400, 9000)

// cityLights are the cities' and the hull's lights: sodium amber to
// warm white, black bodies from 1900 to 3200 kelvin through libtheme, at
// one brightness, so they read as cities and not as glitter
var cityLights = warmLights(1900, 2300, 2700, 3200)

// warmLights is a black body at each temperature as three channels, all
// scaled to one luminance
func warmLights(kelvins ...float64) []maths.Light {
	lit := make([]maths.Light, 0, len(kelvins))
	for _, kelvin := range kelvins {
		spectrum := bands.Split(swatch.Blackbody(kelvin))
		red, green, blue := srgb.Light(observer.Swatch(spectrum))
		colour := maths.RGB(red, green, blue)
		lit = append(lit, colour.Scale(0.9/colour.Luminance()))
	}
	return lit
}

// ring is one band of the pair: how far out it runs, how far it leans
// from the lantern's axis, and which way round it turns
type ring struct {
	radius, lean, sense float64
}

// nested is the pair, one inside the other: the outer at the radius for one g
// in a day, the inner four fifths of it, each leaning 45 degrees from the
// lantern's axis on either side, so the two stand at right angles and the
// lantern's axis splits the angle between them; they turn opposite ways.
//
// her design, 2026-10-04.
func nested() []ring {
	return []ring{
		{radius: radius, lean: 45 * maths.Degree, sense: 1},
		{radius: 0.8 * radius, lean: -45 * maths.Degree, sense: -1},
	}
}

// lanternAxis is the axis the magnetar's beams spin about, on average,
// since the spin axis precesses round it; the rings are set from it
func lanternAxis() maths.Vec {
	return maths.V(0.3, 1, 0.2).Unit()
}

// space is where the band sits before any turn: its axis is the
// lantern's, tipped by its lean toward the eye's side
func (one ring) space() maths.Subspace {
	axis := lanternAxis()
	toEye := maths.V(0, 0, 1)
	across := toEye.Sub(axis.Scale(toEye.Dot(axis))).Unit()
	tip := maths.Subspace{Orientation: maths.Rotation(across, one.lean)}
	return maths.Subspace{Orientation: maths.Upright(tip.Out(axis))}
}

// turning is the band turning about its own axis: once a day at one g's
// radius, longer or shorter with the square root of its own, and the
// other way round for the inner
func (one ring) turning(day time.Duration) thing.Motion {
	space := one.space()
	period := float64(day) * math.Sqrt(one.radius/radius) * one.sense
	return thing.Spin{In: space, Axis: space.Out(maths.V(0, 1, 0)),
		Period: time.Duration(period)}
}

// place puts the band in the world: its floor painted by biome with the
// seed for its number, its gap wall, and its hull, all turning together
func (one ring) place(w *thing.World, number int, gapWall *thing.Material,
	day time.Duration, chat io.Writer) error {
	tiles := hexes.Band{Radius: one.radius, Width: width, Around: hexAround}
	world := biome.Make(biome.Fields{Tiling: tiles,
		Seed: uint64(1806 + number), Scale: 45, Sea: 0.02, Sprawl: 4})
	world.Lights = urbanLights
	fmt.Fprintf(chat, "ring %d: %d rivers, %d lakes, %d volcanoes, "+
		"%d cities\n", number, world.Rivers, world.Lakes,
		world.Volcanoes, world.Cities)
	floorMat := &thing.Material{Name: "floor",
		Texture: texture.Tiled{Tiling: tiles, Paints: world,
			Lines: 0.12},
		Shine: texture.Tiled{Tiling: tiles, Paints: world.Shine()}}
	// the outside is hull: the people live on the inside, so it is dark
	hull := &thing.Material{Name: "hull",
		Colour: maths.RGB(0.12, 0.12, 0.14), Gloss: 0.3, Index: 4}
	floor, facing, rest, err := band(0, one.radius, number == 0)
	if err != nil {
		return err
	}
	turning := one.turning(day)
	w.Add(thing.Thing{Shape: floor, Material: floorMat, Motion: turning})
	w.Add(thing.Thing{Shape: facing, Material: gapWall, Motion: turning})
	w.Add(thing.Thing{Shape: rest, Material: hull, Motion: turning})
	return nil
}

// path is a band's path through the gas: where it sits and how far out
type path struct {
	space  maths.Subspace
	radius float64
}

// pathsOf is the bands' paths, each placed once
func pathsOf(pair []ring) []path {
	paths := make([]path, 0, len(pair))
	for _, one := range pair {
		paths = append(paths, path{space: one.space(),
			radius: one.radius})
	}
	return paths
}

// fromPaths is how far p is from the nearest band's path, the circle at
// its radius in its own plane
func (n wake) fromPaths(p maths.Vec) float64 {
	nearest := math.Inf(1)
	for _, one := range n.paths {
		q := one.space.In(p)
		away := math.Hypot(math.Hypot(q.X, q.Z)-one.radius, q.Y)
		nearest = math.Min(nearest, away)
	}
	return nearest
}

// band is one ring as three meshes: its floor, the inner face between the
// walls; the wall face that looks across the gap at its twin; and the rest,
// the walls and the hull, each strip with its own
// normals, smooth round and hard at the corners.
func band(y, radius float64, gapUp bool) (*mesh.Mesh, *mesh.Mesh,
	*mesh.Mesh, error) {
	type rz struct{ r, y float64 }
	lo, hi := y-width/2, y+width/2
	in, out := radius-wall, radius+thick
	floor := []rz{{radius, lo}, {radius, hi}}
	// each wall's outer face: the one toward the gap looks at the twin
	upper := []rz{{in, hi + thick}, {out, hi + thick}}
	lower := []rz{{out, lo - thick}, {in, lo - thick}}
	facing, away := lower, upper
	if gapUp {
		facing, away = upper, lower
	}
	rest := [][]rz{
		{{radius, hi}, {in, hi}},     // the upper wall's inner face
		{{in, hi}, {in, hi + thick}}, // its top
		away,
		{{out, hi + thick}, {out, lo - thick}}, // the hull's outside
		{{in, lo - thick}, {in, lo}},
		{{in, lo}, {radius, lo}},
	}
	build := func(strips [][]rz) (*mesh.Mesh, error) {
		var verts, normals []maths.Vec
		var faces [][3]int
		for _, s := range strips {
			base := len(verts)
			d := rz{s[1].r - s[0].r, s[1].y - s[0].y}
			// the strip's face points to its left as the
			// profile runs, which, as the profile goes round
			// the section, is away from the solid: the floor
			// toward the axis, the hull out
			face := rz{-d.y, d.r}
			l := math.Hypot(face.r, face.y)
			face = rz{face.r / l, face.y / l}
			for i := range around {
				t := 2 * math.Pi * float64(i) / around
				c, sn := math.Cos(t), math.Sin(t)
				for _, p := range s {
					verts = append(verts,
						maths.V(p.r*sn, p.y, p.r*c))
					normal := maths.V(face.r*sn, face.y,
						face.r*c)
					normals = append(normals, normal)
				}
			}
			at := func(i, j int) int {
				return base + (i%around)*2 + j
			}
			for i := range around {
				q := [4]int{at(i, 0), at(i+1, 0), at(i+1, 1),
					at(i, 1)}
				triangles := [][3]int{{q[0], q[1], q[2]},
					{q[0], q[2], q[3]}}
				for _, t := range triangles {
					p0, p1 := verts[t[0]], verts[t[1]]
					p2 := verts[t[2]]
					across := p1.Sub(p0).Cross(p2.Sub(p0))
					if across.Dot(normals[t[0]]) < 0 {
						t[1], t[2] = t[2], t[1]
					}
					faces = append(faces, t)
				}
			}
		}
		return mesh.New(verts, faces, normals, true)
	}
	f, err := build([][]rz{floor})
	if err != nil {
		return nil, nil, nil, err
	}
	g, err := build([][]rz{facing})
	if err != nil {
		return nil, nil, nil, err
	}
	r, err := build(rest)
	return f, g, r, err
}

// nebula is the gas's settings: how thick, and what share of it is neon
// and argon, the rest hydrogen
type nebula struct {
	thick, neon, argon float64
	// glow is how bright its glow is, whatever it is made of, as a
	// luminance
	glow float64
}

// share is one element and how much of the gas it is
type share struct {
	element string
	part    float64
}

// material is the gas's look: its scatter, and its glow mixed from its
// elements' lines, in a fixed order so the sum is the same every time
func (gas nebula) material() (*thing.Material, error) {
	shares := []share{{"hydrogen", max(0, 1-gas.neon-gas.argon)},
		{"neon", gas.neon}, {"argon", gas.argon}}
	var glow maths.Light
	for _, one := range shares {
		element, err := lines.Of(one.element)
		if err != nil {
			return nil, err
		}
		glow = glow.Add(element.Glow(gas.glow).Scale(one.part))
	}
	return &thing.Material{Name: "nebula",
		Colour: maths.RGB(0.22, 0.14, 0.17), Glow: glow}, nil
}

// wake is the nebula the pair is parked in, with the channel their spin
// has swept through it: thin, ragged gas everywhere, almost none within
// clear of the rings' path, and what was swept piled along the channel's
// edges, as a moon clears a gap in saturn's rings and banks it up.
type wake struct {
	// reach is the nebula's radius, clear the channel's, and thick the
	// gas's density away from both
	reach, clear, thick float64
	// paths are the bands' paths through it
	paths []path
}

// Bounds is the cube that holds the nebula, its reach each way from the middle.
func (n wake) Bounds() maths.Box {
	r := n.reach
	return maths.Box{Min: maths.V(-r, -r, -r), Max: maths.V(r, r, r)}
}

// Span is the nebula's ball, so a march starts on the gas.
func (n wake) Span(r maths.Ray, near, far float64) (float64, float64, bool) {
	b := r.At.Dot(r.Dir)
	c := r.At.Dot(r.At) - n.reach*n.reach
	disc := b*b - c
	if disc < 0 {
		return 0, 0, false
	}
	root := math.Sqrt(disc)
	enter, leave := math.Max(-b-root, near), math.Min(-b+root, far)
	return enter, leave, leave > enter
}

// Density is the nebula's gas at p: ragged, thin to its edge, and
// cleared round the bands' paths.
func (n wake) Density(p maths.Vec) float64 {
	if p.Len() > n.reach {
		return 0
	}
	// ragged at every size, as a nebula is
	rag := 0.5 + 0.5*noise.Fractal(1806, p.X/260, p.Y/260, p.Z/260, 5)
	// cubed, so most of it is thin and the wisps stand out against the dark
	base := n.thick * rag * rag * rag * 3
	// thinning to nothing toward its edge, so it has no rim to show
	base *= 1 - maths.Smooth(0.45*n.reach, n.reach, p.Len())
	// how far from the nearest band's path
	d := n.fromPaths(p)
	swept := maths.Smooth(0.55*n.clear, n.clear, d)
	ridge := math.Exp(-math.Pow((d-n.clear)/(0.3*n.clear), 2))
	return base * (swept + 2.2*ridge)
}

// farView lights the pair from the middle only, by a lantern: a star of
// 9,000 km in its dyson swarm, its light coming out between the panels
// in shafts, and the eye inside the nebula, 3.6 million km out and 25
// degrees above their plane, its up along the plane so the hoops stand.
//
// the lantern's light at the rings, 1,840 out, is about what the four
// spotlights it replaced gave: only just.
func farView(w *thing.World, away float64, still bool) (*thing.World,
	maths.Subspace, error) {
	glass := lantern.BlackGlass(2.2)
	beams := &emit.Polar{Width: 6 * maths.Degree, Soft: 4 * maths.Degree,
		Offset: 15 * maths.Degree, Tilt: 50 * maths.Degree,
		Spin: 1500 * time.Millisecond, Axis: lanternAxis(),
		Cone: 35 * maths.Degree, Precess: 5 * time.Second}
	middle := lantern.Lantern{Radius: 9, Orbits: 12, Panels: 4,
		Kelvin: 9000, Green: 6000, Turn: time.Minute, Wide: 9,
		Wear: 1, Skin: glass, Lining: glass,
		Shells: lantern.Shells{Spread: 1.2, Fastest: 15 * time.Second,
			Narrowest: 2, Tumble: 40 * time.Second},
		Beams: beams, Clock: &beamClock, Glare: 30}
	if still {
		middle.Turn = 0
	}
	if _, err := middle.Place(w); err != nil {
		return nil, maths.Subspace{}, err
	}
	el := 25 * maths.Degree
	from := maths.V(0, away*math.Sin(el), away*math.Cos(el))
	return w, maths.Subspace{Origin: from,
		Orientation: maths.LookAtUp(from, maths.V(0, 0, 0),
			maths.V(1, 0, 0))}, nil
}

// marble is a black marble for a wall: cities gathered where a slow noise
// says people are, each a speck lit warm, a few in neon, and dark between.
type marble struct{ seed uint64 }

// At is the light of the marble at p: a speck where a city stands, else dark.
func (m marble) At(p maths.Vec) maths.Light {
	// where the people are: clusters about thirty thousand km across
	crowd := maths.Smooth(0.05, 0.45,
		noise.Fractal(m.seed, p.X/30, p.Y/30, p.Z/30, 4))
	if crowd <= 0 {
		return maths.Light{}
	}
	// a city to every cell of about a hundred km, lit if the crowd is
	// enough
	const cell = 0.1
	i := int64(math.Floor(p.X / cell))
	j := int64(math.Floor(p.Y / cell))
	k := int64(math.Floor(p.Z / cell))
	if noise.Hash3(m.seed, i, j, k) > 0.6*crowd {
		return maths.Light{}
	}
	if noise.Hash3(m.seed+1, i, j, k) < 0.12 {
		pick := noise.Hash3(m.seed+2, i, j, k) *
			float64(len(cityLights))
		return cityLights[int(pick)%len(cityLights)].Scale(0.8)
	}
	return maths.RGB(1.0, 0.68, 0.32).Scale(0.6 + 0.6*crowd)
}
