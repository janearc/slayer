package bodies_test

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"strings"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/hexes"
	"github.com/janearc/slayer/scene/bodies"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/texture"
	"github.com/janearc/slayer/scene/thing"
	"github.com/janearc/slayer/scene/world"
)

// grounds for the tests that follow: a terrain of a few plates, and a
// look that paints each biome its own grey, so that a tile of one class
// is told from a tile of another.

func terrain() world.Terrain { return world.Terrain{Seed: 3, Ocean: 0.5, Plates: 3} }

func look() world.Look {
	var l world.Look
	for i := range l.Paints {
		l.Paints[i] = maths.Grey(float64(i+1) / 16)
	}
	return l
}

// spread is n directions about the sphere, a fibonacci spiral: y steps
// evenly from 1 to -1 and the turn about y is a golden angle each time.
func spread(n int) []maths.Vec {
	golden := math.Pi * (3 - math.Sqrt(5))
	out := make([]maths.Vec, n)
	for i := range out {
		y := 1 - 2*(float64(i)+0.5)/float64(n)
		r := math.Sqrt(1 - y*y)
		out[i] = maths.V(r*math.Cos(golden*float64(i)), y, r*math.Sin(golden*float64(i)))
	}
	return out
}

// placed is the material of a rock body with the ground, placed at a
// detail, as the one thing in its world.
func placed(t *testing.T, g *bodies.Ground, detail int) *thing.Material {
	t.Helper()
	b := bodies.Body{Name: "r", Kind: bodies.Rock, Radius: 8, Ground: g}
	w := thing.NewWorld()
	if err := b.Place(w, maths.Vec{}, 0.5, detail); err != nil {
		t.Fatal(err)
	}
	return solidNamed(t, w, "r").Material
}

// a ground is tiled on a sphere split detail times, and never fewer than once:
// a detail of zero, or under, is one.
func TestAGroundIsTiledOnASphereSplitDetailTimes(t *testing.T) {
	for detail, want := range map[int]int{-3: 1, 0: 1, 1: 1, 2: 2, 5: 5} {
		m := placed(t, &bodies.Ground{Terrain: terrain(), Look: look()}, detail)
		if got := m.Texture.(texture.Tiled).Tiling; got != hexes.Tiling(hexes.Sphere{N: want}) {
			t.Errorf("a detail of %d is tiled %#v, want a split of %d", detail, got, want)
		}
	}
}

// a moon's ground is tiled at half its parent's detail, and not under a
// split of twice: 8 is 4, 12 is 6, 13 is 6, and 5, 2 and 1 are 2.
func TestAMoonsGroundIsTiledAtHalfTheParentsDetail(t *testing.T) {
	for detail, want := range map[int]int{8: 4, 12: 6, 13: 6, 5: 2, 2: 2, 1: 2} {
		parent := bodies.Body{Name: "p", Kind: bodies.Star, Radius: 8, Mass: bodies.Sun,
			Moons: []bodies.Moon{{Orbit: 4e5, Body: bodies.Body{Name: "m", Kind: bodies.Rock,
				Radius: 2, Ground: &bodies.Ground{Terrain: terrain(), Look: look()}}}}}
		w := thing.NewWorld()
		if err := parent.Place(w, maths.Vec{}, 0.5, detail); err != nil {
			t.Fatal(err)
		}
		got := solidNamed(t, w, "m").Material.Texture.(texture.Tiled).Tiling
		if got != hexes.Tiling(hexes.Sphere{N: want}) {
			t.Errorf("a parent of detail %d has a moon tiled %#v, want a split of %d", detail, got, want)
		}
	}
}

// a moon whose ground cannot be made refuses the placing, by name, as
// its parent's would.
func TestAMoonWhoseGroundCannotBeMadeRefusesTheParent(t *testing.T) {
	parent := bodies.Body{Name: "p", Kind: bodies.Star, Radius: 8, Mass: bodies.Sun,
		Moons: []bodies.Moon{{Orbit: 4e5, Body: bodies.Body{Name: "m", Kind: bodies.Rock,
			Radius: 2, Ground: &bodies.Ground{Terrain: terrain(),
				Features: func(*world.World) ([]world.Feature, error) {
					return nil, fmt.Errorf("no land for maxwell")
				}}}}}}
	if err := parent.Place(thing.NewWorld(), maths.Vec{}, 0.5, 4); err == nil ||
		!strings.Contains(err.Error(), "m: no land for maxwell") {
		t.Errorf("a parent with such a moon is placed with %v", err)
	}
}

// a rock with no ground is a mid-grey ball of its radius, 4 units.
func TestARockWithNoGroundIsAMidGreyBall(t *testing.T) {
	w := thing.NewWorld()
	if err := (bodies.Body{Name: "r", Kind: bodies.Rock, Radius: 8}).Place(w, maths.Vec{}, 0.5, 4); err != nil {
		t.Fatal(err)
	}
	ball := solidNamed(t, w, "r")
	if ball.Material.Colour != maths.Grey(0.5) {
		t.Errorf("it is %v", ball.Material.Colour)
	}
	if got := ball.Shape.(shape.Sphere).Radius; got != 4 {
		t.Errorf("it is %v across its radius", got)
	}
}

// a map wears on the sphere as the picture is drawn, which is mirrored,
// as texture.Picture says: longitude runs from +x toward +z, and the
// picture's left edge is minus pi.
//
// a picture 8 wide and 4 tall has its pixel (x, y) centred at longitude
// -pi + (x + 1/2) 2 pi / 8 and latitude pi/2 - (y + 1/2) pi / 4. Turn
// moves the left edge round, so a direction at longitude L reads the
// picture at L + Turn.
//
// the map has one white pixel, (5, 1), at longitude 3 pi / 8 and latitude
// pi / 8; the lights one white pixel, (1, 2), at longitude -5 pi / 8 and
// latitude -pi / 8. a white pixel is a linear light of one, in all three.
//
// the lights are a gain of the Glow, a city's light against the sun's,
// four thousandths when it is not given.
func TestAGroundWearsItsMapAndLightsAsDrawnMirrored(t *testing.T) {
	dot := func(x, y int) image.Image {
		img := image.NewRGBA(image.Rect(0, 0, 8, 4))
		for i := 0; i < len(img.Pix); i += 4 {
			img.Pix[i+3] = 255
		}
		img.Set(x, y, color.RGBA{255, 255, 255, 255})
		return img
	}
	toward := func(lon, lat float64) maths.Vec {
		return maths.V(math.Cos(lat)*math.Cos(lon), math.Sin(lat), math.Cos(lat)*math.Sin(lon))
	}
	for _, c := range []struct {
		name       string
		turn, glow float64
		gain       float64
	}{{"turned nought", 0, 0, 0.004}, {"turned an eighth of a turn", math.Pi / 8, 0.5, 0.5}} {
		g := &bodies.Ground{Terrain: terrain(), Look: look(), Map: dot(5, 1),
			Lights: dot(1, 2), Turn: c.turn, Glow: c.glow}
		m := placed(t, g, 4)
		onMap := toward(3*math.Pi/8-c.turn, math.Pi/8)
		onLights := toward(-5*math.Pi/8-c.turn, -math.Pi/8)
		mirrored := func(v maths.Vec) maths.Vec { return maths.V(v.X, v.Y, -v.Z) }
		nearLight(t, c.name+": the map's white pixel", m.Texture.At(onMap), maths.Grey(1), 1e-6)
		nearLight(t, c.name+": the map, the right way round", m.Texture.At(mirrored(onMap)),
			maths.Light{}, 1e-6)
		nearLight(t, c.name+": the lights' white pixel", m.Shine.At(onLights),
			maths.Grey(c.gain), 1e-6)
		nearLight(t, c.name+": the lights, the right way round", m.Shine.At(mirrored(onLights)),
			maths.Light{}, 1e-6)
		if got := m.Texture.(texture.Picture).Turn; got != c.turn {
			t.Errorf("%s: the map is turned %v", c.name, got)
		}
		if got := m.Shine.(texture.Picture); got.Turn != c.turn || got.Gain != c.gain {
			t.Errorf("%s: the lights are turned %v at a gain of %v", c.name, got.Turn, got.Gain)
		}
	}
}

// a ground with a Fine is painted finer than its tiles, as world.Fine
// does with the rungs, vary and mix it is given, and not as the tiles
// alone: the same world, painted by the same call, reads the same
// everywhere, and the tiles' own paint does not.
func TestAGroundWithAFinenessIsPaintedFiner(t *testing.T) {
	g := &bodies.Ground{Terrain: terrain(), Look: look(),
		Fine: &bodies.Fineness{Rungs: 2, Vary: 0.25, Mix: 0.5}}
	m := placed(t, g, 4)
	tr := terrain()
	tr.Tiling = hexes.Sphere{N: 4}
	ref := world.New(tr)
	fine, tiles := ref.Fine(look(), 2, 0.25, 0.5), ref.Painted(look())
	differs := false
	for i, p := range spread(400) {
		if got, want := m.Texture.At(p), fine.At(p); got != want {
			t.Fatalf("point %d reads %v, the fine world %v", i, got, want)
		}
		differs = differs || fine.At(p) != tiles.At(p)
	}
	if !differs {
		t.Error("the fine paint is the tiles' paint everywhere, so this holds nothing")
	}
}

// a map is worn in place of a fineness, as it is in place of the paint.
func TestAMapIsWornInPlaceOfAFineness(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 8, 4))
	g := &bodies.Ground{Terrain: terrain(), Look: look(), Map: img,
		Fine: &bodies.Fineness{Rungs: 2, Vary: 0.25, Mix: 0.5}}
	if _, ok := placed(t, g, 4).Texture.(texture.Picture); !ok {
		t.Error("a ground with a map and a fineness is not the map")
	}
}

// a radar's roughness is the ground's rough look painted three rungs
// below the world's own, varied by 0.3 and mixed by 0.35, which the
// body chooses and nothing gives. two things are held of it.
//
// first, that the vary is a share of the paint: on a rough look of one
// grey, a half, each fine tile is a half scaled by 1 + 0.3 h for a hash
// h from -1/2 to 1/2, so it reads within 0.425 and 0.575 and reaches
// near both ends, and a vary of two thirds more or none shows.
//
// second, that the rungs and the mix are these: it reads as the same
// world's Fine called with 3, 0.3 and 0.35, on a look whose greys differ
// by class so that a mix takes effect, and not as 4, 1.6 or 1.7 would.
func TestARoughnessIsFinePaintedThreeRungsDown(t *testing.T) {
	flat := look()
	for i := range flat.Paints {
		flat.Paints[i] = maths.Grey(0.5)
	}
	g := &bodies.Ground{Terrain: terrain(), Look: look(), Rough: &flat}
	rough := placed(t, g, 4).Rough
	lo, hi := 1.0, 0.0
	for _, p := range spread(3000) {
		v := rough.At(p).R
		if v < 0.425-1e-9 || v > 0.575+1e-9 {
			t.Fatalf("a rough tile of a half reads %v, outside 0.425 to 0.575", v)
		}
		lo, hi = math.Min(lo, v), math.Max(hi, v)
	}
	if lo > 0.43 || hi < 0.57 {
		t.Errorf("the roughness runs only %v to %v of 0.425 to 0.575", lo, hi)
	}

	classes := look()
	g = &bodies.Ground{Terrain: terrain(), Look: look(), Rough: &classes}
	rough = placed(t, g, 4).Rough
	tr := terrain()
	tr.Tiling = hexes.Sphere{N: 4}
	ref := world.New(tr)
	want := ref.Fine(classes, 3, 0.3, 0.35)
	others := map[string]texture.Texture{
		"4 rungs":   ref.Fine(classes, 4, 0.3, 0.35),
		"a vary of": ref.Fine(classes, 3, 1.6, 0.35),
		"a mix of":  ref.Fine(classes, 3, 0.3, 1.7),
	}
	differs := map[string]bool{}
	for i, p := range spread(1500) {
		if got := rough.At(p); got != want.At(p) {
			t.Fatalf("point %d reads %v, the fine world %v", i, got, want.At(p))
		}
		for name, o := range others {
			differs[name] = differs[name] || o.At(p) != want.At(p)
		}
	}
	for name := range others {
		if !differs[name] {
			t.Errorf("%s reads the same everywhere, so this does not tell it from 3, 0.3 and 0.35", name)
		}
	}
}

// the cities of a world with no lights are laid by the terrain: Land is
// where the land is, and Coast is a tile with water beside it. here the
// ground is grass all over but for one tile of sea, on the +y pole, so a
// tile is land but that one, and a coast only if it neighbours that one.
//
// the town and metro shares, rungs, glow and seed are the body's own.
func TestCitiesAreLaidOnTheLandAndItsCoasts(t *testing.T) {
	g := &bodies.Ground{Terrain: terrain(), Look: look(),
		Features: func(w *world.World) ([]world.Feature, error) {
			seaAt := w.Tiling.Centre(w.Tiling.Nearest(maths.Up))
			return []world.Feature{
				world.Patch{At: maths.Up, Reach: 10, Of: world.Grass},
				world.Patch{At: seaAt, Reach: 0.01, Of: world.Sea},
			}, nil
		}}
	glow := maths.Light{R: 0.5, G: 0.25, B: 0.125}
	b := bodies.Body{Name: "r", Kind: bodies.Rock, Radius: 8, Ground: g,
		Cities: &bodies.Cities{Rungs: 3, Share: 0.5, Metro: 0.25, Glow: glow, Seed: 11}}
	w := thing.NewWorld()
	if err := b.Place(w, maths.Vec{}, 0.5, 4); err != nil {
		t.Fatal(err)
	}
	shine := solidNamed(t, w, "r").Material.Shine.(texture.Cities)
	tiling := hexes.Sphere{N: 4}
	if shine.Tiling != hexes.Tiling(tiling) || shine.Rungs != 3 || shine.Share != 0.5 ||
		shine.Metro != 0.25 || shine.Glow != glow || shine.Seed != 11 {
		t.Errorf("the cities are %+v", shine)
	}
	sea := tiling.Nearest(maths.Up)
	beside := map[int64]bool{}
	for _, n := range tiling.Neighbours(sea) {
		beside[tiling.ID(n)] = true
	}
	if len(beside) < 5 {
		t.Fatalf("the sea tile has %d neighbours", len(beside))
	}
	for _, tile := range tiling.Tiles() {
		p := tiling.Centre(tile)
		isSea := tile == sea
		if got := shine.Land(p); got == isSea {
			t.Errorf("tile %v: land is %v", tile, got)
		}
		if got, want := shine.Coast(p), beside[tiling.ID(tile)]; got != want {
			t.Errorf("tile %v: coast is %v, want %v", tile, got, want)
		}
	}
}

// the cities of a world with a map are laid by the map: land is where
// the map is not water (blue over red and green, by texture.Picture).
//
// a coast is land with water within a step of 0.04 of the radius to the
// east, west, north or south: 0.0509 of a pixel of an eight wide map at
// the equator.
//
// the map is green all over but for the pixel column 5, blue. between
// the centres of columns 4 and 5 the light blends linearly, and it is
// water when blue is above 1.05 of the green, a share of the way across
// of 1.05 / 2.05 = 0.512.
//
// so a point at 4.48 along the columns is land, and a step of 0.0509 east
// of it, 4.531, is water: a coast. a point at 3.7 is land and a step
// either way is land too, and a step of 1.08 would reach the water at
// 5.07, so it is not a coast.
//
// the same from the east: 5.52 is land, and 5.469 is water.
func TestCitiesOnAMapAreLaidByTheMap(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 8, 4))
	for y := 0; y < 4; y++ {
		for x := 0; x < 8; x++ {
			c := color.RGBA{0, 255, 0, 255}
			if x == 5 {
				c = color.RGBA{0, 0, 255, 255}
			}
			img.Set(x, y, c)
		}
	}
	g := &bodies.Ground{Terrain: terrain(), Look: look(), Map: img}
	b := bodies.Body{Name: "r", Kind: bodies.Rock, Radius: 8, Ground: g,
		Cities: &bodies.Cities{Rungs: 3, Share: 0.5, Metro: 0.25, Seed: 11}}
	w := thing.NewWorld()
	if err := b.Place(w, maths.Vec{}, 0.5, 4); err != nil {
		t.Fatal(err)
	}
	shine := w.Things[0].Material.Shine.(texture.Cities)

	// longitude of a point x of the way along the columns, in pixels,
	// on the equator; the pixel centres are at whole numbers
	at := func(x float64) maths.Vec {
		lon := -math.Pi + (x+0.5)*2*math.Pi/8
		return maths.V(math.Cos(lon), 0, math.Sin(lon))
	}
	for x := 0; x < 8; x++ {
		for _, lat := range []float64{math.Pi / 8, -math.Pi / 8} {
			lon := -math.Pi + (float64(x)+0.5)*2*math.Pi/8
			p := maths.V(math.Cos(lat)*math.Cos(lon), math.Sin(lat), math.Cos(lat)*math.Sin(lon))
			if got := shine.Land(p); got != (x != 5) {
				t.Errorf("column %d, latitude %.2f: land is %v", x, lat, got)
			}
		}
	}
	for _, c := range []struct {
		x           float64
		land, coast bool
	}{
		{4.48, true, true}, {5.52, true, true}, {3.7, true, false}, {6.3, true, false},
		{5.0, false, false},
	} {
		if got := shine.Land(at(c.x)); got != c.land {
			t.Errorf("%v along the columns: land is %v", c.x, got)
		}
		if c.land {
			if got := shine.Coast(at(c.x)); got != c.coast {
				t.Errorf("%v along the columns: coast is %v", c.x, got)
			}
		}
	}
}
