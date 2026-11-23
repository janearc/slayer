// Package smoke preserves rendering milestones as numbered scenes. Each smoke
// has a stable number, name, scene, and camera field of view, so a regression
// can be traced to the capability it exercises.
//
// Smokes accumulate. Do not renumber a passing scene or change it silently;
// changes to existing smokes belong in separate commits explaining why.
package smoke

import (
	"fmt"
	"image"
	"image/jpeg"
	"math"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"time"

	kerr "github.com/janearc/celestial/hole"
	"github.com/janearc/celestial/measure"
	"github.com/janearc/celestial/radiation"
	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/hexes"
	hash "github.com/janearc/slayer/maths/noise"
	"github.com/janearc/slayer/render/colour"
	"github.com/janearc/slayer/scene/bodies"
	"github.com/janearc/slayer/scene/bodies/template"
	"github.com/janearc/slayer/scene/medium"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/sky"
	"github.com/janearc/slayer/scene/swatches"
	"github.com/janearc/slayer/scene/texture"
	"github.com/janearc/slayer/scene/thing"
	"github.com/janearc/slayer/scene/world"
)

// Smoke is one milestone.
type Smoke struct {
	Number  int
	Name    string
	group   string // the one group the smoke is checked with, one of Groups
	heavy   bool   // true for a smoke that takes minutes to draw both ways
	noSmall string // why the smoke cannot be judged small, or empty when it can
	// Scene builds the world and where the eye is, fresh on every call.
	Scene func() (*thing.World, maths.Subspace)
	// Wide is how much the camera sees from its left edge to its right,
	// in radians.
	Wide float64
	// Samples is how many rays across and down go through each pixel;
	// zero or one is one ray, through the middle.
	Samples int
	// Bands are the bands outside the visible the smoke is also seen
	// in, each laid over the visible in false colour. Empty is visible
	// alone.
	Bands []maths.Band
	// Palette, when there is one, is the visible colour each of Bands
	// stands in, instead of libtheme's for its range: the hubble
	// palette, for a nebula's lines.
	Palette []maths.Light
	// Gauge is the film the smoke is shot on, when it is shot on one:
	// the picture is barred to the gauge's shape and grained as the
	// gauge grains. The zero gauge is none, the picture as rendered.
	Gauge film.Gauge
	// Tone is how the smoke's light becomes a picture: an exposure and
	// a film's shoulder. The zero tone is the clamp, as before.
	Tone colour.Tone
	// Inputs are what the smoke reads from disk, so a cache can hash what a
	// render read: paths; $NAME for every file in the folder the
	// environment variable NAME names; or $NAME/file for that one file in
	// it. A $NAME that is unset reads nothing.
	//
	// A smoke that reads nothing has none, which is most of them.
	Inputs []string
}

// Group is the one group the smoke is checked with, one InGroup knows.
func (made Smoke) Group() string { return made.group }

// Heavy is true for a smoke that takes minutes to draw both ways, so a
// check can leave it for a run of its own.
func (made Smoke) Heavy() bool { return made.heavy }

// NoSmall is why the smoke cannot be judged at small size, or empty when
// it can. A check at small size skips a smoke that has one, and says it
// did, with this reason.
func (made Smoke) NoSmall() string { return made.noSmall }

// matteLamp is how many times stronger a lamp is made than the strength a
// smoke names, for a matte surface to show its colour times that strength.
// trace reads a lamp's strength as irradiance, so a matte surface shows its
// colour over pi times it; the smokes' strengths are written for colour
// times strength.
//
// A smoke whose lamps light only volumes, bodies or other light that
// trace does not divide by pi names its lamps as they are.
const matteLamp = math.Pi

// matte is the strength a lamp needs for a matte surface to show its
// colour times named, under trace.
func matte(named maths.Light) maths.Light { return named.Scale(matteLamp) }

// matteSun is a thing.Sun whose light is matte of the strength named, so a
// matte surface it lights shows its colour times that strength.
func matteSun(toward maths.Vec, named maths.Light) thing.Lamp {
	return thing.Sun(toward, matte(named))
}

// placed is a template's refusal to be placed, which in a smoke is a
// bug in the smoke: the picture stops with the reason.
func placed(err error) {
	if err != nil {
		panic(err)
	}
}

// sure is a door's refusal surfaced the smoke way: the picture stops
// with the reason, and the made thing passes through.
func sure[T any](made T, err error) T {
	placed(err)
	return made
}

// with is the body wearing parts, or the picture stops with the
// reason it cannot.
func with(body bodies.Body, parts ...bodies.Part) bodies.Body {
	return sure(body.With(parts...))
}

// found is the tile Find reaches from a direction for a want, or the
// picture stops with the reason: a smoke that asks for land or sea near
// a direction and is told there is none has no scene, and the direction
// itself in the tile's place would draw a wrong one without saying.
func found(w *world.World, what string, toward maths.Vec,
	want func(world.Biome) bool) hexes.Tile {
	tile, ok := w.Find(toward, want)
	if !ok {
		placed(fmt.Errorf("smoke: no tile of %s near the direction %v",
			what, toward))
	}
	return tile
}

// Reads is every file a smoke reads, its Inputs with each $NAME opened
// out to the files in that folder and each $NAME/file to that one file
// in it, sorted; a NAME that is unset reads nothing.
//
// The files themselves are not opened here, so a named file that is not
// there is still listed.
func (s Smoke) Reads() ([]string, error) {
	var files []string
	for _, in := range s.Inputs {
		if !strings.HasPrefix(in, "$") {
			files = append(files, in)
			continue
		}
		name, file, _ := strings.Cut(in[1:], "/")
		dir := os.Getenv(name)
		if dir == "" {
			continue
		}
		if file != "" {
			files = append(files, filepath.Join(dir, file))
			continue
		}
		entries, err := os.ReadDir(dir)
		if err != nil {
			return nil, err
		}
		for _, e := range entries {
			if !e.IsDir() {
				files = append(files, filepath.Join(dir, e.Name()))
			}
		}
	}
	sort.Strings(files)
	return files, nil
}

// all is every smoke, in order.
var all = []Smoke{
	{Number: 1, group: "light", Name: "a ball on a floor", Scene: one,
		Wide: 50 * maths.Degree},
	{Number: 2, group: "light", Name: "two lamps", Scene: two, Wide: 50 * maths.Degree},
	// three rays a side: the checker near the horizon is finer than a
	// pixel, and one ray a pixel turns it into dashes
	{Number: 3, group: "light", Name: "a ball on a set", Scene: three,
		Wide: 50 * maths.Degree, Samples: 3},
	{Number: 4, group: "light", Name: "red, green and blue on the set", Scene: four,
		Wide: 50 * maths.Degree, Samples: 3},
	{Number: 5, group: "light", Name: "a spotlight on the set", Scene: five,
		Wide: 50 * maths.Degree, Samples: 3},
	{Number: 6, group: "worlds", noSmall: "the hex edges are thinner than a " +
		"small terminal pixel: at small size and one real pixel a sextant " +
		"third the terminal differs from the real picture beyond what an " +
		"eye can see (worst 0.0271), where at full size they agree " +
		"(worst 0.0120)", Name: "a world of hexes", Scene: six,
		Wide: 50 * maths.Degree, Samples: 3},
	{Number: 7, group: "worlds", Name: "a planet", Scene: seven,
		Wide: 50 * maths.Degree, Samples: 3},
	{Number: 8, group: "worlds", Name: "a floor and a ring in hexes", Scene: eight,
		Wide: 50 * maths.Degree, Samples: 3},
	{Number: 9, group: "gas", Name: "smoke on the set", Scene: nine,
		Wide: 50 * maths.Degree, Samples: 3},
	{Number: 10, group: "worlds", Name: "a gas giant", Scene: ten,
		Wide: 50 * maths.Degree, Samples: 3},
	{Number: 11, group: "worlds", Name: "a ringed giant", Scene: eleven,
		Wide: 50 * maths.Degree, Samples: 3},
	{Number: 12, group: "worlds", Name: "a planet with features", Scene: twelve,
		Wide: 50 * maths.Degree, Samples: 3},
	// the first smoke whose width is a lens's and not a number: a long
	// lens from far off, the shot that flattens a planet against its sky
	{Number: 13, group: "worlds", Name: "a ringed giant through a long lens",
		Scene: thirteen, Wide: thirteenLens.Wide(), Samples: 3},
	{Number: 14, group: "worlds", Name: "a gas giant in detail", Scene: fourteen,
		Wide: fourteenLens.Wide(), Samples: 3},
	{Number: 15, group: "worlds", Name: "the giants", Scene: fifteen,
		Wide: 50 * maths.Degree, Samples: 3},
	{Number: 16, group: "worlds", Name: "the rocks", Scene: sixteen,
		Wide: 50 * maths.Degree, Samples: 3},
	{Number: 17, group: "worlds", Name: "moons on orbits", Scene: seventeen,
		Wide: 50 * maths.Degree, Samples: 3},
	{Number: 18, group: "lens", Name: "hubble looks at jupiter", Scene: eighteen,
		Wide: film.Hubble().Wide(), Samples: 3},
	{Number: 19, group: "lens", Name: "a satellite looks down", Scene: nineteen,
		Wide: nineteenLens.Wide(), Samples: 3},
	{Number: 20, group: "energy", Name: "a magnetar behind a giant, in gamma and x-ray",
		Scene: twenty, Wide: 40 * maths.Degree, Samples: 3,
		Bands: []maths.Band{twentyGamma, twentyXRay}},
	{Number: 21, group: "worlds", Name: "magellan maps venus", Scene: twentyOne,
		Wide: 30 * maths.Degree, Samples: 3,
		Bands: []maths.Band{twentyOneBand}},
	{Number: 22, group: "lens", Name: "the ringed giant on super 8", Scene: twentyTwo,
		Wide: film.Lens{Focal: 50, Gauge: film.Super8}.Wide(), Samples: 3,
		Gauge: film.Super8},
	{Number: 23, group: "lens", Name: "the ringed giant on imax", Scene: twentyTwo,
		Wide: film.Lens{Focal: 50, Gauge: film.IMAX}.Wide(), Samples: 3,
		Gauge: film.IMAX},
	{Number: 24, group: "worlds", Name: "io looks up at jupiter", Scene: twentyFour,
		Wide: twentyFourWide, Samples: 3},
	{Number: 25, group: "energy", Name: "a quasar and its jets", Scene: twentyFive,
		Wide: 40 * maths.Degree, Samples: 3, Bands: []maths.Band{twentyXRay},
		Tone: quasarTone},
	{Number: 26, group: "energy", Name: "a blazar, down the jet", Scene: twentySix,
		Wide: 40 * maths.Degree, Samples: 3,
		Bands: []maths.Band{twentyXRay, twentyGamma},
		Tone:  colour.Tone{Knee: 0.5, Top: 300}},
	{Number: 27, group: "energy", Name: "a pulsar turns", Scene: twentySeven,
		Wide: 40 * maths.Degree, Samples: 3},
	{Number: 28, group: "energy", Name: "a magnetar flares", Scene: twentyEight,
		Wide: 40 * maths.Degree, Samples: 3, Bands: []maths.Band{twentyGamma}},
	{Number: 29, group: "gas", heavy: true, Name: "the pillars of creation", Scene: twentyNine,
		Wide: 40 * maths.Degree, Samples: 3, Bands: twentyNineLines,
		Palette: twentyNinePalette, Tone: twentyNineTone},
	{Number: 30, group: "earth", Name: "the earth by night", Scene: thirty,
		Wide: 40 * maths.Degree, Samples: 3, Tone: thirtyTone},
	{Number: 31, group: "gas", Name: "the bullet cluster, in its palette", Scene: thirtyOne,
		Wide: 40 * maths.Degree, Samples: 3, Bands: thirtyOneBands,
		Palette: thirtyOnePalette, Tone: thirtyOneTone},
	{Number: 32, group: "earth", Name: "the aurora from orbit", Scene: thirtyTwo,
		Wide: 60 * maths.Degree, Samples: 3, Tone: thirtyTone},
	{Number: 33, group: "energy", Name: "a world beside a magnetar", Scene: thirtyThree,
		Wide: 40 * maths.Degree, Samples: 3, Bands: []maths.Band{twentyXRay},
		Palette: []maths.Light{maths.RGB(0.3, 0.9, 1)}, Tone: thirtyThreeTone},
	{Number: 34, group: "gas", Name: "a hot jupiter, its air blown off", Scene: thirtyFour,
		Wide: 80 * maths.Degree, Samples: 3, Tone: cometTone},
	{Number: 35, group: "gas", heavy: true, Name: "hale-bopp", Scene: thirtyFive,
		Wide: 60 * maths.Degree, Samples: 3, Tone: cometTone},
	// the photon ring is thinner than a pixel, and a picture of it wants
	// many rays per pixel or it speckles
	{Number: 36, group: "holes", heavy: true, Name: "the photon ring", Scene: thirtySix,
		Wide: 40 * maths.Degree, Samples: 9, Tone: quasarTone},
	{Number: 37, group: "holes", heavy: true, Name: "falling in", Scene: thirtySeven,
		Wide: 90 * maths.Degree, Samples: 9, Tone: quasarTone},
	// the one smoke that reads from disk, and says so
	{Number: 38, group: "earth", Name: "the blue marble", Scene: thirtyEight,
		Wide: 40 * maths.Degree, Samples: 2,
		Inputs: []string{"$SLAYER_MARBLE/blue.jpg"}},
	{Number: 39, group: "earth", Name: "a windscreen catches the sky", Scene: thirtyNine,
		Wide: 60 * maths.Degree, Samples: 3},
	{Number: 40, group: "gas", Name: "four atmospheres", Scene: forty,
		Wide: 50 * maths.Degree, Samples: 3},
	{Number: 41, group: "gas", Name: "a laser in haze", Scene: fortyOne,
		Wide: 50 * maths.Degree, Samples: 3},
	{Number: 42, group: "light", Name: "a hill and a closed solid", Scene: fortyTwo,
		Wide: 50 * maths.Degree, Samples: 3},
	{Number: 43, group: "gas", Name: "a glass fizzing", Scene: fortyThree,
		Wide: 50 * maths.Degree, Samples: 3},
	{Number: 44, group: "earth", Name: "the earth by night from a picture of its lights",
		Scene: fortyFour, Wide: 40 * maths.Degree, Samples: 3, Tone: thirtyTone},
	{Number: 45, group: "worlds", Name: "three planets on a track of the caller's",
		Scene: fortyFive, Wide: 60 * maths.Degree, Samples: 3},
	{Number: 46, group: "gas", Name: "a baked puff beside the exact one",
		Scene: fortySix, Wide: 50 * maths.Degree, Samples: 3},
	{Number: 47, group: "light", Name: "a stage lit by ninjafilm's lamps",
		Scene: fortySeven, Wide: 50 * maths.Degree, Samples: 3},
	// white where the beam arrives whole, as a film held to the light is
	{Number: 48, group: "energy", Name: "lead, aluminium and water on an x-ray film",
		Scene: fortyEight, Wide: 50 * maths.Degree, Samples: 3,
		Bands:   []maths.Band{fortyEightBand},
		Palette: []maths.Light{swatches.Light(swatches.Chalk)}},
	{Number: fortyNineNo, group: "light",
		Name:  "a glass of water on a wooden table in an infinity cove",
		Scene: fortyNine, Wide: studioWide * maths.Degree, Samples: 3},
	{Number: fiftyNo, group: "light",
		Name:  "Webb at the edge of where it may point, lit by the Larson",
		Scene: fifty, Wide: webbWide * maths.Degree, Samples: 3,
		Tone:   colour.Tone{Exposure: unity, Knee: webbKnee, Top: webbTop},
		Inputs: []string{"$" + starsFolder + "/" + starsFile}},
	{Number: fiftyOneNo, group: "light",
		Name:  "a ball of water with bubbles in it",
		Scene: fiftyOne, Wide: bubblesWide * maths.Degree, Samples: 3},
	{Number: fiftyTwoNo, group: "light",
		Name:  "a glass of cola",
		Scene: fiftyTwo, Wide: colaWide * maths.Degree, Samples: 3},
}

// groups are the names a smoke's Group may take, in the order a check
// lists them: lamps and a ball on a floor, planets and moons and their
// surfaces, the lenses and films that look at them, objects that emit
// beyond the visible, gas and dust, the black hole, and the earth seen
// from orbit.
var groups = []string{"light", "worlds", "lens", "energy", "gas", "holes",
	"earth"}

// Groups is the name of every group of smokes, a copy.
func Groups() []string { return slices.Clone(groups) }

// InGroup is every smoke in the named group, oldest first, each a copy.
// A name that is no group's is refused, with the names there are.
func InGroup(name string) ([]Smoke, error) {
	if !slices.Contains(groups, name) {
		return nil, fmt.Errorf("smoke: no group %q; the groups are %s",
			name, strings.Join(groups, ", "))
	}
	var in []Smoke
	for _, s := range all {
		if s.group == name {
			in = append(in, own(s))
		}
	}
	return in, nil
}

// All is every smoke, oldest first, each a copy a caller may change.
func All() []Smoke {
	out := make([]Smoke, len(all))
	for i, s := range all {
		out[i] = own(s)
	}
	return out
}

// Newest is the latest smoke, a copy.
func Newest() Smoke { return own(all[len(all)-1]) }

// Get is the smoke with this number, if there is one, a copy.
func Get(n int) (Smoke, bool) {
	for _, s := range all {
		if s.Number == n {
			return own(s), true
		}
	}
	return Smoke{}, false
}

// own is a smoke whose slices are its own, so a caller that changes them
// changes nothing here. a nil slice stays nil.
func own(s Smoke) Smoke {
	s.Bands = slices.Clone(s.Bands)
	s.Palette = slices.Clone(s.Palette)
	s.Inputs = slices.Clone(s.Inputs)
	return s
}

// one is the first milestone, as doc/walkthrough.md wrote it before any of
// this existed: a clay ball on a chalk floor, one sun, one shadow.
func one() (*thing.World, maths.Subspace) {
	w := thing.NewWorld()
	w.Add(thing.Thing{
		Shape: shape.Plane{},
		Material: &thing.Material{Name: "chalk",
			Colour: swatches.Light(swatches.Chalk)},
	})
	w.Add(thing.Thing{
		Shape: shape.Sphere{Radius: 1},
		Material: &thing.Material{Name: "clay",
			Colour: swatches.Light(swatches.Clay)},
		Motion: thing.Still(maths.Subspace{Origin: maths.V(0, 1, 0)}),
	})
	w.Lamp(matteSun(maths.V(1, 2, 1),
		swatches.Light(swatches.Daylight).Scale(1.1)))
	from := maths.V(0, 2.2, 6)
	return w, maths.Subspace{Origin: from,
		Orientation: maths.LookAt(from, maths.V(0, 0.8, 0))}
}

// the two lamps of smoke two, a hard red from the left and a hard green
// from the right, at the strength smoke one's sun had
var (
	redLamp = matteSun(maths.V(-1.5, 2, 1),
		swatches.Light(swatches.Red).Scale(1.1))
	greenLamp = matteSun(maths.V(1.5, 2, 1),
		swatches.Light(swatches.Green).Scale(1.1))
)

// two is the second milestone: a white ball on the chalk floor, lit by a
// hard red lamp from the left and a hard green one from the right. where
// only red lands it is red, where only green lands it is green, and where
// both land it is yellow, because light adds.
//
// each lamp's shadow is lit by the other, and under the ball, where
// neither reaches, it is black.
func two() (*thing.World, maths.Subspace) {
	return lampsOnPaint(redLamp, greenLamp)
}

// lampsOnPaint is smoke two's floor, ball and camera under any lamps, so
// its checks can light it with one lamp at a time.
func lampsOnPaint(lamps ...thing.Lamp) (*thing.World, maths.Subspace) {
	w := thing.NewWorld()
	w.Add(thing.Thing{
		Shape: shape.Plane{},
		Material: &thing.Material{Name: "chalk",
			Colour: swatches.Light(swatches.Chalk)},
	})
	w.Add(thing.Thing{
		Shape: shape.Sphere{Radius: 1},
		Material: &thing.Material{Name: "white paint",
			Colour: swatches.Light(swatches.Paint)},
		Motion: thing.Still(maths.Subspace{Origin: maths.V(0, 1, 0)}),
	})
	for _, l := range lamps {
		w.Lamp(l)
	}
	from := maths.V(0, 2.2, 6)
	return w, maths.Subspace{Origin: from,
		Orientation: maths.LookAt(from, maths.V(0, 0.8, 0))}
}

// the air of smoke three's set: thin enough that the near floor is crisp,
// thick enough that the checker has faded into the sky before it gets too
// fine for two colours a cell
const setHaze = 0.05

// three is the third milestone: smoke one's clay ball, on a set you can see
// is a floor. the floor is checkered, chalk and slate a unit square, so its
// lines run away toward the horizon and its squares shrink with distance.
//
// the air is hazy, so the far floor fades into the sky; and the sky is pale
// at the horizon and blue straight up, so the ground ends somewhere.
func three() (*thing.World, maths.Subspace) {
	w := set(true)
	w.Add(thing.Thing{
		Shape: shape.Sphere{Radius: 1},
		Material: &thing.Material{Name: "clay",
			Colour: swatches.Light(swatches.Clay)},
		Motion: thing.Still(maths.Subspace{Origin: maths.V(0, 1, 0)}),
	})
	w.Lamp(matteSun(maths.V(1, 2, 1),
		swatches.Light(swatches.Daylight).Scale(1.1)))
	return w, setEye()
}

// set is the stage smoke three built: the checkered floor, and with air,
// the sky and the haze. without air it is the floor alone, so a check can
// look at the lamps' light and nothing else.
func set(air bool) *thing.World {
	w := thing.NewWorld()
	w.Add(thing.Thing{
		Shape: shape.Plane{},
		Material: &thing.Material{Name: "chalk and slate",
			Texture: texture.Checker(1,
				swatches.Light(swatches.Chalk),
				swatches.Light(swatches.Slate))},
	})
	if air {
		w.Sky = sky.Gradient{
			Horizon: swatches.Light(swatches.Horizon),
			Zenith:  swatches.Light(swatches.Zenith),
			Ground:  swatches.Light(swatches.Horizon),
		}
		w.Haze = thing.Haze{Density: setHaze, FromSky: true}
	}
	return w
}

// setEye is where the camera stands on the set, the same as every smoke so
// far: a little above the ball and six back, looking just below its middle.
func setEye() maths.Subspace {
	from := maths.V(0, 2.2, 6)
	return maths.Subspace{Origin: from,
		Orientation: maths.LookAt(from, maths.V(0, 0.8, 0))}
}

// the three lamps of smoke four, a hard red from the left, a hard green from
// the right and a hard blue from behind, each at the strength of smoke one's
// sun
var (
	fourRed = matteSun(maths.V(-1.5, 2, 1),
		swatches.Light(swatches.Red).Scale(1.1))
	fourGreen = matteSun(maths.V(1.5, 2, 1),
		swatches.Light(swatches.Green).Scale(1.1))
	fourBlue = matteSun(maths.V(0, 2, -1.5),
		swatches.Light(swatches.Blue).Scale(1.1))
)

// four is the fourth milestone: smoke three's set, with a white ball lit by
// a red lamp, a green one and a blue one. where all three land it is white;
// each lamp's shadow is lit by the other two, so the shadows are cyan,
// magenta and yellow; and under the ball, where none reaches, it is black.
func four() (*thing.World, maths.Subspace) {
	return fourUnder(true, fourRed, fourGreen, fourBlue)
}

// fourUnder is smoke four's set and ball under any lamps, with or without
// the air, so its checks can light it one lamp at a time.
func fourUnder(air bool, lamps ...thing.Lamp) (*thing.World,
	maths.Subspace) {
	w := set(air)
	w.Add(thing.Thing{
		Shape: shape.Sphere{Radius: 1},
		Material: &thing.Material{Name: "white paint",
			Colour: swatches.Light(swatches.Paint)},
		Motion: thing.Still(maths.Subspace{Origin: maths.V(0, 1, 0)}),
	})
	for _, l := range lamps {
		w.Lamp(l)
	}
	return w, setEye()
}

// the lamps of smoke five: a tungsten spot from above, in front and to the
// left, aimed at the ball, and a flood the colour of the sky overhead
var (
	fiveSpot = film.Spot{
		At:       maths.V(-2, 5, 2),
		Aim:      maths.V(2, -4, -2),
		Strength: matte(swatches.Light(swatches.Tungsten).Scale(26)),
		Cone:     25 * maths.Degree,
		Soft:     8 * maths.Degree,
	}
	fiveFlood = film.Flood{
		Strength: swatches.Light(swatches.Zenith).Scale(0.35),
	}
)

// five is the fifth milestone: smoke four's set and white ball, lit the way
// a stage is. a tungsten spot makes a warm pool of light around the ball
// with a hard shadow, and the floor past the pool falls away; a cool flood,
// the sky's colour, fills everything, so no shadow is black.
func five() (*thing.World, maths.Subspace) {
	return fourUnder(true, fiveSpot, fiveFlood)
}

// the tiles of smoke six: the ball's, coarse enough that a tile is many
// cells across, and the sky's, finer, so the dome reads as a dome
var (
	sixBall = hexes.Sphere{Splits: 5}
	sixSky  = hexes.Sphere{Splits: 6}
)

// sixWorld is the ball's surface: sea and moss, a tile each by hash, with
// dark lines between the tiles.
var sixWorld = texture.Hexed{
	Tiling: sixBall,
	Palette: []maths.Light{swatches.Light(swatches.Sea),
		swatches.Light(swatches.Moss)},
	Seed:  6,
	Lines: 0.6,
}

// six is the sixth milestone: a world of hexes. smoke three's set and sun,
// a ball whose surface is sea and moss in hexagons with dark lines between
// them, and the set's sky gently rendered into hexes overhead, flat inside
// each tile and melting into the true sky at the seams.
func six() (*thing.World, maths.Subspace) {
	return sixUnder(true)
}

// sixUnder is smoke six with or without the air, so a check can look at
// the ball's tiles and nothing else. Without air there is no sky to hex.
func sixUnder(air bool) (*thing.World, maths.Subspace) {
	w := set(air)
	if air {
		w.Sky = sky.Hexed{Of: w.Sky, Tiling: sixSky}
	}
	w.Add(thing.Thing{
		Shape: shape.Sphere{Radius: 1},
		Material: &thing.Material{Name: "sea and moss",
			Texture: sixWorld},
		Motion: thing.Still(maths.Subspace{Origin: maths.V(0, 1, 0)}),
	})
	w.Lamp(matteSun(maths.V(1, 2, 1),
		swatches.Light(swatches.Daylight).Scale(1.1)))
	return w, sixEye()
}

// sixEye stands where the set's eye stands and looks a little higher, so
// the top half of the picture is sky and the facets have room to show.
func sixEye() maths.Subspace {
	from := maths.V(0, 2.2, 6)
	return maths.Subspace{Origin: from,
		Orientation: maths.LookAt(from, maths.V(0, 1.7, 0))}
}

// sevenTerrain is smoke seven's world: plates, a sea over six tenths of
// it, a tilted axis, on a rung coarse enough that a tile is many cells
// across at the planet's size in the picture.
var sevenTerrain = world.Terrain{Tiling: hexes.Sphere{Splits: 8}, Seed: 5,
	Ocean: 0.6, Plates: 7, Axis: maths.V(0.2, 1, 0)}

// sevenLook paints the world's eleven biomes from the paint box's terran
// set, with lines between the tiles.
func sevenLook() world.Look {
	var look world.Look
	for b, s := range swatches.Terran {
		look.Paints[b] = swatches.Light(s)
	}
	look.Lines = 0.5
	return look
}

// seven is the seventh milestone: a planet. a terran world, sea and land
// in hexes with reasons, plates and coasts and snow at the poles, floating
// over the set with its shadow under it, the hexed sky overhead.
func seven() (*thing.World, maths.Subspace) {
	return sevenUnder(true, true)
}

// sevenUnder is smoke seven with or without the set and the air, so a
// check can look at the planet and nothing else.
func sevenUnder(set_, air bool) (*thing.World, maths.Subspace) {
	w := thing.NewWorld()
	if set_ {
		w = set(air)
		if air {
			w.Sky = sky.Hexed{Of: w.Sky, Tiling: sixSky}
		}
	}
	planet := world.New(sevenTerrain)
	w.Add(thing.Thing{
		Shape: shape.Sphere{Radius: 1.5},
		Material: &thing.Material{Name: "terran",
			Texture: planet.Painted(sevenLook())},
		Motion: thing.Still(maths.Subspace{Origin: maths.V(0, 2.2, 0)}),
	})
	w.Lamp(matteSun(maths.V(1, 2, 1),
		swatches.Light(swatches.Daylight).Scale(1.1)))
	from := maths.V(0, 2.6, 6.8)
	return w, maths.Subspace{Origin: from,
		Orientation: maths.LookAt(from, maths.V(0, 2.2, 0))}
}

// the two lattices of smoke eight: the floor, a plane of hexes two units
// apart middle to middle, and the ring, a band round a tube stood on its
// edge
var (
	eightFloor = hexes.Flat{Step: 2, Across: 200, Down: 200}
	eightRing  = hexes.Band{Radius: 1.4, Width: 1, Around: 24}
	eightTube  = shape.Tube{Radius: 1.4, Thick: 0.08, Height: 1}
)

// eight is the eighth milestone: a floor and a ring in hexes. the floor
// is a plane tiled in hexes, chalk and slate by hash with lines between,
// and on it stands a ring, a short tube on its edge, tiled round its
// inside and its outside in sea and moss.
//
// a flood the sky's colour fills the shadows so the inside of the ring
// shows, and the sky is the hexed one.
func eight() (*thing.World, maths.Subspace) {
	return eightUnder(true)
}

// eightUnder is smoke eight with or without the air, so a check can look
// at the tiles and nothing else.
func eightUnder(air bool) (*thing.World, maths.Subspace) {
	w := thing.NewWorld()
	// the floor is a plane, placed so the lattice's extent is centred
	// under the eye; a plane's tiles beyond the extent are still tiles
	w.Add(thing.Thing{
		Shape: shape.Plane{},
		Material: &thing.Material{Name: "hex floor",
			Texture: texture.Hexed{Tiling: eightFloor,
				Palette: []maths.Light{swatches.Light(swatches.Chalk),
					swatches.Light(swatches.Slate)},
				Seed: 8, Lines: 0.5}},
		Motion: thing.Still(maths.Subspace{Origin: maths.V(-200, 0, -200)}),
	})
	// the ring stands on its edge: turned a quarter about x so its axis
	// lies along z, its lowest point on the floor
	w.Add(thing.Thing{
		Shape: eightTube,
		Material: &thing.Material{Name: "hex ring",
			Texture: texture.Hexed{Tiling: eightRing,
				Palette: []maths.Light{swatches.Light(swatches.Sea),
					swatches.Light(swatches.Moss)},
				Seed: 9, Lines: 0.6}},
		Motion: thing.Still(maths.Subspace{
			Origin:      maths.V(0, eightTube.Radius+eightTube.Thick/2, 0),
			Orientation: maths.Rotation(maths.XAxis, 90*maths.Degree)}),
	})
	w.Lamp(matteSun(maths.V(1, 2, 1),
		swatches.Light(swatches.Daylight).Scale(1.1)))
	if air {
		w.Sky = sky.Hexed{Of: sky.Gradient{
			Horizon: swatches.Light(swatches.Horizon),
			Zenith:  swatches.Light(swatches.Zenith),
			Ground:  swatches.Light(swatches.Horizon),
		}, Tiling: sixSky}
		w.Haze = thing.Haze{Density: setHaze, FromSky: true}
		w.Lamp(film.Flood{Strength: swatches.Light(swatches.Zenith).Scale(0.35)})
	}
	from := maths.V(2.5, 2.4, 6)
	return w, maths.Subspace{Origin: from,
		Orientation: maths.LookAt(from, maths.V(0, 1.2, 0))}
}

// ninePuff is smoke nine's smoke: a puff of radius one, two units across,
// made ragged by noise, thick enough that the floor shows through its
// edges and not through its middle.
var ninePuff = medium.Wispy{
	Of:     medium.Puff{Radius: 1, Thick: 6},
	Seed:   9,
	Scale:  0.3,
	Amount: 0.7,
}

// nine is the ninth milestone: smoke on the set. the clay ball of smoke
// three with a puff of white smoke drifting beside it, lit by the sun
// and the sky's flood, the floor showing through its thin edges. the
// first thing a ray passes through rather than stops at.
func nine() (*thing.World, maths.Subspace) {
	return nineUnder(true)
}

// nineUnder is smoke nine with or without the air, so a check can look
// at the smoke and nothing else.
func nineUnder(air bool) (*thing.World, maths.Subspace) {
	w := set(air)
	w.Add(thing.Thing{
		Shape: shape.Sphere{Radius: 1},
		Material: &thing.Material{Name: "clay",
			Colour: swatches.Light(swatches.Clay)},
		Motion: thing.Still(maths.Subspace{Origin: maths.V(-1.2, 1, 0)}),
	})
	w.AddVolume(thing.Volume{
		Medium: ninePuff,
		Material: &thing.Material{Name: "smoke",
			Colour: swatches.Light(swatches.Paint)},
		Motion: thing.Still(maths.Subspace{Origin: maths.V(1.4, 1.5, 0.3)}),
	})
	w.Lamp(matteSun(maths.V(1, 2, 1),
		swatches.Light(swatches.Daylight).Scale(1.1)))
	if air {
		w.Lamp(film.Flood{Strength: swatches.Light(swatches.Zenith).Scale(0.35)})
	}
	return w, setEye()
}

// tenGiant is smoke ten's body: a gas giant of radius one and a half,
// three units across, clear above its radius and thickening to a density
// of forty over the 0.12 of a unit below it.
var tenGiant = medium.Giant{Radius: 1.5, Depth: 0.12, Thick: 40}

// tenBands is what the giant's gas looks like: cream and rust bands
// roiled along their edges, a red storm north of the equator on the lit
// side, the axis tipped a little toward the eye.
var tenBands = texture.Banded{
	Paints: []maths.Light{
		swatches.Light(swatches.Cream), swatches.Light(swatches.Rust),
		swatches.Light(swatches.Cream), swatches.Light(swatches.Ochre),
		swatches.Light(swatches.Cream), swatches.Light(swatches.Rust),
		swatches.Light(swatches.Cream), swatches.Light(swatches.Ochre),
		swatches.Light(swatches.Cream), swatches.Light(swatches.Rust),
		swatches.Light(swatches.Cream),
	},
	Axis:  maths.V(0.15, 1, 0.3),
	Roil:  0.06,
	Shear: 6,
	Grain: 0.08,
	Spots: []texture.Spot{{Lat: 0.3, Lon: -1.1, Size: 0.18,
		Stretch: 1.6, Turn: 2.5, Paint: swatches.Light(swatches.Storm)}},
	Seed: 10,
}

// ten is the tenth milestone: a gas giant. not a ball painted like one: a
// body of gas with no surface, thickening below its cloud tops until the
// light stops, banded and roiled, with a storm, over the set with the
// hexed sky behind. a ray grazing its limb sees through it.
func ten() (*thing.World, maths.Subspace) {
	return tenUnder(true)
}

// tenUnder is smoke ten with or without the air, so a check can look at
// the giant and nothing else.
func tenUnder(air bool) (*thing.World, maths.Subspace) {
	w := thing.NewWorld()
	if air {
		w = set(true)
		w.Sky = sky.Hexed{Of: w.Sky, Tiling: sixSky}
	}
	w.AddVolume(thing.Volume{
		Medium:   tenGiant,
		Material: &thing.Material{Name: "jovian", Texture: tenBands},
		Motion:   thing.Still(maths.Subspace{Origin: maths.V(0, 2.2, 0)}),
	})
	w.Lamp(thing.Sun(maths.V(1, 2, 1),
		swatches.Light(swatches.Daylight).Scale(1.1)))
	if air {
		w.Lamp(film.Flood{Strength: swatches.Light(swatches.Zenith).Scale(0.2)})
	}
	from := maths.V(0, 2.6, 6.8)
	return w, maths.Subspace{Origin: from,
		Orientation: maths.LookAt(from, maths.V(0, 2.2, 0))}
}

// elevenTilt is how far the ringed giant's axis leans away from the eye,
// so the rings open into an ellipse and the eye is south of their plane,
// where the sun from the north throws their shadow on the giant.
const elevenTilt = -24 * maths.Degree

// elevenRings is the ringed giant's rings: ice from just above the cloud
// tops to twice the radius, thin, with a wide bright ring, a dark gap,
// a dimmer outer ring and ringlets through all of it.
var elevenRings = medium.Disc{
	Inner: 1.75, Outer: 3.1, Thick: 0.03, Dense: 40,
	Profile:  []float64{0.25, 0.5, 0.95, 1, 1, 0.9, 0.08, 0.65, 0.7, 0.6, 0.1, 0.5},
	Grain:    0.6,
	Ringlets: 50,
	Seed:     11,
}

// eleven is the eleventh milestone: a ringed giant. smoke ten's giant,
// leaning toward the eye, with rings of ice round it: the giant's shadow
// across the rings behind it, and the rings' shadow across the giant.
func eleven() (*thing.World, maths.Subspace) {
	return elevenUnder(true)
}

// elevenUnder is smoke eleven with or without the air, so a check can
// look at the giant and its rings and nothing else.
func elevenUnder(air bool) (*thing.World, maths.Subspace) {
	w := thing.NewWorld()
	if air {
		w = set(true)
		w.Sky = sky.Hexed{Of: w.Sky, Tiling: sixSky}
	}
	lean := maths.Rotation(maths.XAxis, elevenTilt)
	bands := tenBands
	bands.Axis = maths.Up // the lean is the volume's, not the paint's
	at := maths.V(0, 2.6, 0)
	w.AddVolume(thing.Volume{
		Medium:   tenGiant,
		Material: &thing.Material{Name: "jovian", Texture: bands},
		Motion:   thing.Still(maths.Subspace{Origin: at, Orientation: lean}),
	})
	w.AddVolume(thing.Volume{
		Medium: elevenRings,
		Material: &thing.Material{Name: "ring ice",
			Colour: swatches.Light(swatches.Ice)},
		Motion: thing.Still(maths.Subspace{Origin: at, Orientation: lean}),
	})
	// the sun low over the rings, so the giant's shadow reaches across
	// them, and north of their plane, so their shadow falls on the south
	w.Lamp(thing.Sun(maths.V(1, 0.55, 0.3),
		swatches.Light(swatches.Daylight).Scale(1.1)))
	if air {
		w.Lamp(film.Flood{Strength: swatches.Light(swatches.Zenith).Scale(0.15)})
	}
	from := maths.V(0, 3.4, 9.5)
	return w, maths.Subspace{Origin: from,
		Orientation: maths.LookAt(from, at)}
}

// twelveTerrain is smoke seven's world on a finer rung, so a feature is
// a group of tiles and not one.
var twelveTerrain = world.Terrain{Tiling: hexes.Sphere{Splits: 12}, Seed: 5,
	Ocean: 0.6, Plates: 7, Axis: maths.V(0.2, 1, 0)}

// twelveWorld is the terrain with its features placed by what the plain
// world has where: a volcano on the land facing the eye, a canyon cut
// across the same land, an ice shelf off the northern coast, and a
// plateau on the land to the right.
func twelveWorld() *world.World {
	plain := world.New(twelveTerrain)
	tl, step := plain.Tiling, plain.Tiling.Spacing()
	centre := func(toward maths.Vec, want func(world.Biome) bool) maths.Vec {
		tile, ok := plain.Find(toward, want)
		if !ok {
			return toward.Unit()
		}
		return tl.Centre(tile)
	}
	// centreOf is centre after found has refused a Find that finds
	// nothing, so centre's fallback to the raw direction is never
	// reached. centre is left as it was written, since its inlining and
	// its cost are held where they are; the Find is made twice.
	centreOf := func(what string, toward maths.Vec, want func(world.Biome) bool) maths.Vec {
		found(plain, what, toward, want)
		return centre(toward, want)
	}
	heart := centreOf("land facing the eye", maths.V(0.35, 0.1, 1), world.Land)
	from := centreOf("land for the canyon's start", maths.V(-0.5, -0.1, 0.9), world.Land)
	to := from.Add(heart.Sub(from).Scale(0.55)).Unit()
	shelf := centreOf("water for the shelf", maths.V(0.1, 0.95, 0.3), world.Water)
	flat := centreOf("land for the plateau", maths.V(0.85, -0.2, 0.5), world.Land)
	t := twelveTerrain
	t.Features = []world.Feature{
		world.Volcano{At: heart, Reach: 3.5 * step, Rise: 0.5},
		world.Canyon{From: from, To: to, Width: 1.1 * step, Depth: 0.5},
		world.Shelf{At: shelf, Reach: 3 * step},
		world.Plateau{At: flat, Reach: 2.5 * step, Top: 0.7},
	}
	return world.New(t)
}

// twelve is the twelfth milestone: a planet with features. smoke seven's
// world a rung finer, with a volcano, a canyon, an ice shelf and a
// plateau on it, where the world had land and sea for them.
func twelve() (*thing.World, maths.Subspace) {
	return twelveUnder(true)
}

// twelveUnder is smoke twelve with or without the set and the air.
func twelveUnder(air bool) (*thing.World, maths.Subspace) {
	w := thing.NewWorld()
	if air {
		w = set(true)
		w.Sky = sky.Hexed{Of: w.Sky, Tiling: sixSky}
	}
	planet := twelveWorld()
	w.Add(thing.Thing{
		Shape: shape.Sphere{Radius: 1.5},
		Material: &thing.Material{Name: "terran with features",
			Texture: planet.Painted(sevenLook())},
		Motion: thing.Still(maths.Subspace{Origin: maths.V(0, 2.2, 0)}),
	})
	w.Lamp(matteSun(maths.V(1, 2, 1),
		swatches.Light(swatches.Daylight).Scale(1.1)))
	from := maths.V(0, 2.6, 6.8)
	return w, maths.Subspace{Origin: from,
		Orientation: maths.LookAt(from, maths.V(0, 2.2, 0))}
}

// thirteenLens is a 200 on Super 35: seven degrees across, the lens a
// cinematographer reaches for to make a far planet fill the frame and
// sit flat against its sky.
var thirteenLens = film.Lens{Focal: 200, Gauge: film.Super35}

// thirteenAway is how far the eye stands back from the giant: far enough
// that the rings fill most of the frame through the long lens.
const thirteenAway = 58.0

// thirteen is the thirteenth milestone: a ringed giant through a long
// lens. smoke eleven's giant and rings, the eye nearly sixty units back
// and a 200 on it, so the picture is the same size as eleven's and the
// perspective is gone, which is how a planet looks in a telescope.
func thirteen() (*thing.World, maths.Subspace) {
	w, _ := elevenUnder(true)
	at := maths.V(0, 2.6, 0)
	from := at.Add(maths.V(0, 0.8, 9.5).Unit().Scale(thirteenAway))
	return w, maths.Subspace{Origin: from, Orientation: maths.LookAt(from, at)}
}

// fourteenLens is a 135 on super 35, and fourteenAway how far back the
// eye stands: together they put the giant across most of the frame.
var (
	fourteenLens = film.Lens{Focal: 135, Gauge: film.Super35}
	// var, not const, on purpose: see nineteenRungs
	fourteenAway = 30.0
)

// fourteenBands is smoke ten's giant painted in detail: the same bands,
// with fine turbulence inside each, jets shearing the roil the opposite
// way in neighbouring bands, the paint mottled along a band, the great
// storm, and a street of small white storms.
//
// it was asked for because ten bands were not detail enough for the gas
// giants.
var fourteenBands = texture.Banded{
	Paints: tenBands.Paints,
	Axis:   maths.V(0.1, 1, 0.25),
	Roil:   0.06,
	Shear:  7,
	Grain:  0.08,
	Fine:   0.05,
	Jets:   10,
	Mottle: 0.5,
	Spots: []texture.Spot{{Lat: -0.22, Lon: -1.3, Size: 0.16,
		Stretch: 1.7, Turn: 2.8, Paint: swatches.Light(swatches.Storm)}},
	Streets: []texture.Street{{Lat: 0.36, Count: 7, Size: 0.05,
		Turn: 1.4, Paint: swatches.Light(swatches.Paint), Seed: 14}},
	Seed: 14,
}

// fourteen is the fourteenth milestone: a gas giant in detail. smoke
// ten's giant, painted with fine roil, jets, mottle, the great storm and
// a street of white ovals, through a long lens so it fills the frame.
func fourteen() (*thing.World, maths.Subspace) {
	return fourteenUnder(true)
}

// fourteenUnder is smoke fourteen with or without the set and the air.
func fourteenUnder(air bool) (*thing.World, maths.Subspace) {
	w := thing.NewWorld()
	if air {
		w = set(true)
		w.Sky = sky.Hexed{Of: w.Sky, Tiling: sixSky}
	}
	at := maths.V(0, 2.2, 0)
	w.AddVolume(thing.Volume{
		Medium:   tenGiant,
		Material: &thing.Material{Name: "jovian, in detail", Texture: fourteenBands},
		Motion:   thing.Still(maths.Subspace{Origin: at}),
	})
	// the sun from the eye's side, so the face the eye sees is lit
	w.Lamp(thing.Sun(maths.V(0.6, 0.8, 1.5),
		swatches.Light(swatches.Daylight).Scale(1.1)))
	if air {
		w.Lamp(film.Flood{Strength: swatches.Light(swatches.Zenith).Scale(0.15)})
	}
	from := at.Add(maths.V(0, 0.4, 6.8).Unit().Scale(fourteenAway))
	return w, maths.Subspace{Origin: from, Orientation: maths.LookAt(from, at)}
}

// space is a world with no set: a sky of stars and the sun from the
// right, for bodies to sit in. the stars are about a pixel across at
// the smokes' picture size; smaller, the terminal's nine rays and smokejudge's
// twelve land on one differently and smokejudge calls the picture a liar.
func space() *thing.World {
	w := thing.NewWorld()
	w.Sky = sky.Stars{Seed: 15, Size: 0.35, Dark: maths.Grey(0.002)}
	w.Lamp(thing.Sun(maths.V(1, 0.5, 0.8),
		swatches.Light(swatches.Daylight).Scale(1.2)))
	return w
}

// row places bodies in a row along x from zero, at scale, each sitting gap
// units from the last's edge, rings included, and hands back where the row
// ends: the far edge of the last body.
func row(w *thing.World, scale float64, gap float64, detail int,
	bodies ...bodies.Body) float64 {
	x := 0.0
	for i, b := range bodies {
		r := b.Radius() * scale
		if rings, worn := b.Rings(); worn {
			r *= rings.Outer()
		}
		if i > 0 {
			x += r + gap
		}
		placed(b.Place(w, maths.V(x, 0, 0), scale, detail))
		x += r
	}
	return x
}

// framing is the eye that holds a row from left to right in a picture
// wide radians across, a little above it and looking at its middle, with
// a tenth of the row to spare at each end.
func framing(left, right, wide float64) maths.Subspace {
	mid := (left + right) / 2
	half := (right-left)/2 + (right-left)*0.1
	dist := half / math.Tan(wide/2)
	from := maths.V(mid, 0.12*dist, dist)
	return maths.Subspace{Origin: from,
		Orientation: maths.LookAt(from, maths.V(mid, 0, 0))}
}

// fifteen is the fifteenth milestone: the giants. jupiter, saturn with
// its rings, uranus on its side with its thin dark rings, and neptune,
// in a row at their true relative sizes, from the templates in bodies,
// under a sun from the right with the stars behind.
func fifteen() (*thing.World, maths.Subspace) {
	w := space()
	jupiter := sure(template.Jupiter())
	right := row(w, 1/jupiter.Radius(), 0.8, 6,
		jupiter, sure(template.Saturn()), sure(template.Uranus()), sure(template.Neptune()))
	return w, framing(-1, right, 50*maths.Degree)
}

// sixteen is the sixteenth milestone: the rocks. mercury, venus, the
// earth, mars, and pluto with charon, in a row at their true relative
// sizes, from the templates, under the same sun with the same stars.
// pluto and charon go round their common centre, placed by Binary.
func sixteen() (*thing.World, maths.Subspace) {
	w := space()
	earth, mercury := sure(template.Earth()), sure(template.Mercury())
	scale := 1 / earth.Radius()
	right := row(w, scale, 0.6, 8,
		mercury, sure(template.Venus()), earth, sure(template.Mars()))
	// pluto and charon at the end of the row at their true distance,
	// 19,591 km: their common centre is then about 2,130 km from
	// pluto's, outside it, which is what makes them a binary and the
	// one thing that matters about the pair.
	//
	// Binary puts pluto past the centre and charon short of it, each
	// by the other's share of the mass, so the centre stands where
	// charon clears mars, and the frame reaches pluto
	pluto, charon := sure(template.Pluto()), sure(template.Charon())
	apart := 19591.0
	total := pluto.Mass() + charon.Mass()
	ra, rb := apart*charon.Mass()/total*scale, apart*pluto.Mass()/total*scale
	at := maths.V(right+0.6+rb+charon.Radius()*scale, 0, 0)
	placed(bodies.Binary(w, pluto, charon, at, bodies.Kilometres(apart), scale, 4))
	left := -mercury.Radius() * scale
	return w, framing(left, at.X+ra+pluto.Radius()*scale, 50*maths.Degree)
}

// seventeenGiant is the jupiter template changed, as a template is for:
// its mass and so its gravity kept, its first three moons six times
// their size and their orbits drawn in close, so that in a picture a
// giant's width across they are discs and not points.
//
// kepler still times them from the mass, and they go round in hours.
func seventeenGiant() bodies.Body {
	j := sure(template.Jupiter())
	grown := j.WithoutMoons()
	for _, moon := range j.Moons()[:3] {
		body := moon.Body()
		var parts []bodies.Part
		if ground, worn := body.Ground(); worn {
			terrain := ground.Terrain()
			terrain.Seed += 17
			fresh := sure(bodies.NewGround(terrain, ground.Look())).
				Featured(ground.Features())
			parts = append(parts, fresh)
		}
		for _, plume := range body.Plumes() {
			parts = append(parts, plume)
		}
		bigger := sure(bodies.NewBody(body.Name(), body.Kind(),
			bodies.Kilometres(body.Radius()*6),
			measure.Kilograms(body.Mass()), parts...))
		grown = with(grown, sure(bodies.NewMoon(bigger,
			bodies.Kilometres(moon.Orbit()/2.4),
			measure.Radians(moon.Phase()))))
	}
	return grown
}

// seventeen is the seventeenth milestone: moons on orbits. the giant with three
// big moons going round it, timed by its mass through kepler's law, seen from
// above the plane of their orbits so the orbits read as ellipses.
//
// the smoke is drawn at zero; its test puts ten hours on one sheet.
func seventeen() (*thing.World, maths.Subspace) {
	w := space()
	j := seventeenGiant()
	scale := 1 / j.Radius()
	placed(j.Place(w, maths.Vec{}, scale, 6))
	// the widest orbit, with room round it
	far := j.Moons()[2].Orbit() * scale
	from := maths.V(0, 0.55*far, 1.0*far).Scale(1.35)
	return w, maths.Subspace{Origin: from, Orientation: maths.LookAt(from, maths.Vec{})}
}

// eighteenAU is an astronomical unit in kilometres, and eighteenAway how
// far jupiter is from the earth at opposition, in astronomical units.
const (
	eighteenAU   = 149597870.7
	eighteenAway = 4.2
)

// eighteen is the eighteenth milestone: hubble looks at jupiter. the
// jupiter template at its true size and its true distance from the
// earth at opposition, through film.Hubble(), the bare telescope on
// WFC3's detector: 46 arcseconds in a field of 220.
//
// that field is film's, not hubble's. WFC3's real field is 162
// arcseconds on a side, as film/lens.go says where it defines Hubble,
// so jupiter here fills 0.21 of the field where the real camera gives
// it 0.28: about a quarter small.
//
// the field stays as it is; it is not changed here.
//
// the four moons are the points they are at that size, one of them in
// front of the disc with its shadow on it, and the stars behind are on
// a rung fine enough that a few fall in so narrow a field.
func eighteen() (*thing.World, maths.Subspace) {
	w := thing.NewWorld()
	// a field this narrow needs stars eleven thousand splits fine, a
	// billion tiles, which cost nothing to have
	w.Sky = sky.Stars{Tiling: hexes.Sphere{Splits: 11000}, Share: 0.15, Size: 0.05,
		Bright: 0.8, Seed: 18, Dark: maths.Grey(0.002)}
	j := sure(template.Jupiter())
	// the moons where they are seen from the earth: io in front of the
	// disc, which on an orbit whose plane runs toward minus z is three
	// quarters round, the others behind and off to its sides
	moons := j.Moons()
	j = j.WithoutMoons()
	for i, phase := range []float64{270 * maths.Degree, 80 * maths.Degree,
		85 * maths.Degree, 92 * maths.Degree} {
		j = with(j, sure(bodies.NewMoon(moons[i].Body(),
			bodies.Kilometres(moons[i].Orbit()),
			measure.Radians(phase))))
	}
	scale := 1 / j.Radius()
	away := eighteenAway * eighteenAU * scale
	placed(j.Place(w, maths.V(0, 0, -away), scale, 10))
	// at opposition the sun is behind the earth: a little off the line
	// of sight, so a moon's shadow lands a third of a radius beside the
	// moon on the disc; further off and it is thrown past the limb
	w.Lamp(thing.Sun(maths.V(0.05, 0.03, 1),
		swatches.Light(swatches.Daylight).Scale(1.2)))
	return w, maths.Local()
}

// nineteenLens sees forty kilometres of ground from four hundred up, and
// nineteenRungs is how many rungs below the earth template's own tiles
// the ground is painted: seven, so a tile is about five kilometres and
// the strip holds eight of them.
//
// a texture can pick its rung by the pixel asking (world.Fine with no
// rungs), and that is the ladder in use; a smoke cannot show it to
// smokejudge, since its finer picture would pick a finer rung and every
// cell would hold an edge. the unit test holds it; the smoke is fixed.
var (
	nineteenLens = film.Swath(40, 400)
	// var, not const, on purpose: a const is folded into the arithmetic
	// round it, and the pictures are held bit for bit; that is for after
	// the switch
	nineteenRungs = 7
	nineteenUp    = 400.0
)

// nineteen is the nineteenth milestone: a satellite looks down. the
// earth template, its ground painted seven rungs below its own tiles so
// the hexes are kilometres across, seen straight down from four hundred
// kilometres over a coast, through a lens that sees forty kilometres.
//
// a world one wants to stand on has to be generated at the eye, not
// stored: the coarse world decides the coast, the fine rung elaborates
// it, and nothing between is stored. the earth's air is in the way, a little.
func nineteen() (*thing.World, maths.Subspace) {
	w := thing.NewWorld()
	// a clear day over the coast: the clouds are a part of the earth
	// now, and this shot is about the ground
	e := sure(template.Earth()).WithoutMoons().WithoutClouds()
	ground, worn := e.Ground()
	if !worn {
		panic("the earth template wears no ground")
	}
	scale := 1 / e.Radius()
	// the ground, made as the template makes it and then painted fine
	t := ground.Terrain()
	t.Tiling = hexes.Sphere{Splits: 12}
	plain := world.New(t)
	placed(e.Place(w, maths.Vec{}, scale, 12))
	w.Things[0].Material = &thing.Material{Name: "earth, fine",
		Texture: plain.Fine(ground.Look(), nineteenRungs, 0.2, 0.35)}
	// over a coast: the nearest shore to a direction, and the line
	// between it and the sea beside it, turned with the planet, which
	// leans on its axis as the template says
	shore := found(plain, "beach", maths.V(0.5, 0.4, 0.75),
		func(b world.Biome) bool {
			return b == world.Beach
		})
	coast := plain.Tiling.Centre(shore)
	for _, nb := range plain.Tiling.Neighbours(shore) {
		if world.Water(plain.Biome(nb)) {
			coast = coast.Add(plain.Tiling.Centre(nb)).Unit()
			break
		}
	}
	lean := maths.Rotation(maths.XAxis, e.Tilt())
	over := lean.Out(coast)
	w.Lamp(matteSun(over.Add(maths.V(0.2, 0.3, 0.1)),
		swatches.Light(swatches.Daylight).Scale(1.1)))
	from := over.Scale(1 + nineteenUp*scale)
	return w, maths.Subspace{Origin: from, Orientation: maths.LookAt(from, over)}
}

// the two bands smoke twenty is seen in beside the visible: a gamma ray
// of a ten-thousandth of a nanometre and an x-ray of a nanometre
var (
	twentyGamma = maths.Band{Nanometres: 1e-4}
	twentyXRay  = maths.Band{Nanometres: 1}
)

// twenty is the twentieth milestone: a magnetar behind a giant, in
// gamma and x-ray. a gas giant with a moon in front of it, the sun from
// the right; behind the giant a magnetar, faint in visible light and
// hidden by the giant, and off to the side a pulsar.
//
// the magnetar gives off gamma rays and the pulsar x-rays, straight at
// the eye. in gamma the giant's gas passes most of what reaches it, so
// the magnetar shines through the giant as through a fog, and the moon,
// which passes nothing, is a black disc on it.
//
// in x-ray the gas passes less, so the pulsar shows only through the
// giant's thin limb. the three are laid over each other in false colour,
// gamma standing for one visible colour and x-ray another, as libtheme
// maps the ranges.
//
// a scene lit by several emitters, where a false colour picture shows
// how gamma radiation gets through several different things.
func twenty() (*thing.World, maths.Subspace) {
	w := space()
	w.Lamps = nil
	w.Lamp(thing.Sun(maths.V(1, 0.6, 0.5),
		swatches.Light(swatches.Daylight).Scale(1.1)))
	// what passes a unit of the gas at unit density: the giant is a
	// hundred and fifteen units of density across its middle, so the
	// whole of it passes about a sixth of the gamma and an eighth of
	// the x-ray, and the limb nearly all
	gas := &thing.Material{Name: "jovian", Texture: tenBands,
		Radiant: thing.Spectrum{Passes: map[radiation.Range]float64{
			radiation.Gamma(): 0.985, radiation.XRay(): 0.982}}}
	w.AddVolume(thing.Volume{Medium: tenGiant, Material: gas,
		Motion: thing.Still(maths.Local())})
	w.Add(thing.Thing{
		Shape: shape.Sphere{Radius: 0.35},
		Material: &thing.Material{Name: "moon",
			Colour: swatches.Light(swatches.Slate)},
		Motion: thing.Still(maths.Subspace{Origin: maths.V(0.5, 0.3, 3)}),
	})
	w.Add(thing.Thing{
		Shape: shape.Sphere{Radius: 0.5},
		Material: &thing.Material{Name: "magnetar",
			Glow: maths.RGB(0.25, 0.3, 0.45),
			Radiant: thing.Spectrum{
				Gives: map[radiation.Range]float64{radiation.Gamma(): 1.3, radiation.XRay(): 0.3}}},
		Motion: thing.Still(maths.Subspace{Origin: maths.V(0.9, 0.6, -14)}),
	})
	w.Add(thing.Thing{
		Shape: shape.Sphere{Radius: 0.3},
		Material: &thing.Material{Name: "pulsar",
			Glow: maths.RGB(0.3, 0.3, 0.35),
			Radiant: thing.Spectrum{
				Gives: map[radiation.Range]float64{radiation.XRay(): 1.2}}},
		Motion: thing.Still(maths.Subspace{Origin: maths.V(-2.4, -0.8, -9)}),
	})
	from := maths.V(0, 0, 8)
	return w, maths.Subspace{Origin: from, Orientation: maths.LookAt(from, maths.Vec{})}
}

// twentyOneBand is magellan's radar, 1.26e8 nanometres, twelve and six
// tenths centimetres, which libtheme calls radio: its microwave stops at
// fifteen millimetres.
var twentyOneBand = maths.Band{Nanometres: 1.26e8}

// twentyOne is the twenty-first milestone: magellan maps venus. the
// venus template from orbit, a pale yellow ball of cloud in visible
// light, and in radio, with the radar at the eye, the ground under the
// clouds: dark where it is smooth plain, bright where it is rough rock.
//
// maxwell and maat are brightest, and the limb falls off as the beam
// lands aslant: venus's surface, seen by synthetic aperture radar.
//
// what a synthetic aperture makes is a picture of what scatters back
// from the radar's own direction, and this is that picture; the
// aperture's trick is not modelled.
func twentyOne() (*thing.World, maths.Subspace) {
	w := space()
	v := sure(template.Venus())
	scale := 1 / v.Radius()
	placed(v.Place(w, maths.Vec{}, scale, 10))
	from := maths.V(0.4, 0.3, 1).Unit().Scale(6.5)
	w.Lamp(film.Radar{At: from, Range: radiation.Radio(), Strength: 1})
	return w, maths.Subspace{Origin: from, Orientation: maths.LookAt(from, maths.Vec{})}
}

// twentyTwoAway is how far the eye stands from the ringed giant for
// smokes twenty-two and twenty-three: far enough that a 50 on super 8,
// which sees under seven degrees, holds the rings.
const twentyTwoAway = 54.0

// twentyTwo is the twenty-second and the twenty-third milestone, the
// same scene shot twice: smoke eleven's ringed giant on the set, the
// eye fifty-four units back, through a 50 on super 8 and then through
// the same 50 on imax.
//
// on super 8 the giant fills the frame and the picture is grainy; on
// imax the 50 is a wide lens, the set stretches away round a small
// giant, and the picture is clean: the films told apart, imax from
// super 8.
func twentyTwo() (*thing.World, maths.Subspace) {
	w, _ := elevenUnder(true)
	at := maths.V(0, 2.6, 0)
	from := at.Add(maths.V(0, 0.8, 9.5).Unit().Scale(twentyTwoAway))
	return w, maths.Subspace{Origin: from, Orientation: maths.LookAt(from, at)}
}

// twentyFourWide is a 20 on super 35: sixty-four degrees.
var twentyFourWide = film.Lens{Focal: 20, Gauge: film.Super35}.Wide()

// twentyFourUp is how high the eye stands over io's ground, in io
// radii: nine hundred metres, a highland.
const twentyFourUp = 0.0005

// twentyFourEye is where on io the eye stands, in io's own space:
// seventy-two degrees from the point under jupiter, so jupiter stands
// eighteen degrees up, and round toward the sun so that is thirty up.
var twentyFourEye = maths.V(-0.31, 0.73, 0.61).Unit()

// twentyFourPlume is where the shot's plume stands: twelve degrees
// from the eye toward jupiter, so it rises over the horizon under the
// giant and under it.
var twentyFourPlume = swung(twentyFourEye, maths.XAxis.Neg(), 12*maths.Degree)

// swung is a direction turned from one toward another by an angle,
// along the great circle between them.
func swung(from, toward maths.Vec, angle float64) maths.Vec {
	side := toward.Sub(from.Scale(from.Dot(toward))).Unit()
	return from.Scale(math.Cos(angle)).Add(side.Scale(math.Sin(angle)))
}

// twentyFour is the twenty-fourth milestone: io looks up at jupiter.
// the jupiter template with io alone, the eye standing on io's ground
// nine hundred metres up, jupiter overhead gibbous and banded, the sun
// off to the side.
//
// a plume stands on the horizon under the giant, and io's ground is in
// hexes down to the eye's feet.
//
// standing on the surface of io, or any other moon, which has to be
// generated where one stands, since one is there to move about it.
//
// the ground is the template's own, painted on the rung the eye asks
// for; the plume is placed in io's space and goes round with it; the
// eye stands in that space too. nothing is made for the shot but where
// to stand.
func twentyFour() (*thing.World, maths.Subspace) { return twentyFourAt(0) }

// twentyFourAt is the shot at a time: io goes round, locked, and the
// eye with it; half a circuit on, io is in jupiter's shadow.
func twentyFourAt(t time.Duration) (*thing.World, maths.Subspace) {
	w := thing.NewWorld()
	// a sunlit ground: only the brighter stars show beside it
	w.Sky = sky.Stars{Seed: 15, Share: 0.06, Size: 0.35, Dark: maths.Grey(0.002)}
	w.Lamp(matteSun(maths.V(0.3, 0.05, 1),
		swatches.Light(swatches.Daylight).Scale(1.2)))
	j := sure(template.Jupiter())
	first := j.Moons()[0]
	io := first.Body()
	ground, worn := io.Ground()
	if !worn {
		placed(fmt.Errorf("smoke: the jupiter template's io wears no ground"))
	}
	volcanoes := ground.Features()
	if volcanoes == nil {
		placed(fmt.Errorf("smoke: the jupiter template's io has no features, " +
			"which the plume's volcano is added to"))
	}
	ground = ground.
		Fined(sure(bodies.NewFineness(0, maths.Share(0.25),
			maths.Share(0.35)))).
		Featured(func(pw *world.World) ([]world.Feature, error) {
			fs, err := volcanoes(pw)
			return append(fs, world.Volcano{At: twentyFourPlume,
				Reach: 2.5 * pw.Tiling.Spacing(), Rise: 0.3}), err
		})
	// a small plume where the shot wants one, on a live volcano; pele's
	// is on the far side
	io = with(io, ground, sure(bodies.NewPlume(twentyFourPlume,
		bodies.Kilometres(60), bodies.Kilometres(200),
		bodies.PerKilometre(0.03), maths.RGB(0.82, 0.78, 0.7))))
	j = with(j.WithoutMoons(), sure(bodies.NewMoon(io,
		bodies.Kilometres(first.Orbit()),
		measure.Radians(first.Phase()))))
	scale := 1 / io.Radius()
	placed(j.Place(w, maths.Vec{}, scale, 10))
	// the detail is the shot's: a tile is six pixels of a picture three
	// hundred and sixty across, whatever the screen
	w.Pixel = twentyFourWide / 360
	orbit := sure(j.Orbit(0, maths.Vec{}, scale))
	from, up := bodies.Stand(orbit.At(t), io.Radius()*scale, twentyFourEye,
		twentyFourUp)
	// at jupiter's centre, then nine degrees down to bring the horizon in
	look := maths.LookAtUp(from, maths.Vec{}, up).
		Then(maths.Rotation(maths.XAxis, -9*maths.Degree))
	return w, maths.Subspace{Origin: from, Orientation: look}
}

// dark is a world of stars with no sun, for bodies that are their own
// light.
func dark() *thing.World {
	w := space()
	w.Lamps = nil
	return w
}

// quasarTone is the quasar smokes' tone: a shoulder from half, with the
// disc's hottest ring, sixty times white, near the top.
var quasarTone = colour.Tone{Knee: 0.5, Top: 80}

// twentyFive is the twenty-fifth milestone: a quasar and its jets. the
// quasar template, the hole's radius two units, leaning thirty degrees
// so the disc is a flat ellipse and the jets stand clear of it.
//
// the upper jet leans toward the eye and so is the brighter, since the
// jets are beamed; the jets are cut to what the frame holds.
//
// the disc is a black body by radius, blue-white inside and orange at
// the rim, brighter on the side coming toward the eye, and the hole
// is a black ball. in x-ray the jets alone, the disc dark.
//
// the template places the hole's field, so in visible light the hole
// bends the light that passes it; a picture in a band is drawn without
// the field, so nothing bends there.
func twentyFive() (*thing.World, maths.Subspace) {
	w := dark()
	q := sure(template.Quasar()).Tilting(measure.Degrees(30))
	jets, worn := q.Jets()
	if !worn {
		panic("the quasar template wears no jets")
	}
	q = with(q, rejets(jets, 25*q.Radius(), jets.Speed()))
	placed(q.Place(w, maths.Vec{}, 2/q.Radius(), 4))
	from := maths.V(0, 0, 300)
	return w, maths.Subspace{Origin: from, Orientation: maths.LookAt(from, maths.Vec{})}
}

// twentySix is the twenty-sixth milestone: a blazar, down the jet. the
// same body leaning eighty-five degrees, so a jet points five degrees
// off the eye, which stands past its end.
//
// beamed two hundredfold, the jet's end is a blaze that hides the disc
// behind it, which is what a blazar is; in x-ray and gamma the same.
func twentySix() (*thing.World, maths.Subspace) {
	w := dark()
	// a blazar is a quasar seen down its jet, so the body is the
	// quasar's and the shot aims it; the jets are run at nineteen
	// twentieths of the speed of light here so the beaming shows, a
	// look the shot chooses and the body does not know about
	b := sure(template.Quasar()).Tilting(measure.Degrees(85))
	beams, worn := b.Jets()
	if !worn {
		panic("the quasar template wears no jets")
	}
	b = with(b, rejets(beams, beams.Length(), 0.95))
	placed(b.Place(w, maths.Vec{}, 1/b.Radius(), 4))
	from := maths.V(0, 0, 420)
	return w, maths.Subspace{Origin: from, Orientation: maths.LookAt(from, maths.Vec{})}
}

// twentySeven is the twenty-seventh milestone: a pulsar turns. the
// crab's, two hundred kilometres to the unit so the star is a dot, its
// spin up, its beams forty-five degrees off and sweeping round thirty
// times a second; the test lays a turn out on a sheet.
func twentySeven() (*thing.World, maths.Subspace) {
	w := dark()
	p := sure(template.Pulsar())
	placed(p.Place(w, maths.Vec{}, 1/200.0, 4))
	from := maths.V(0, 0, 10)
	return w, maths.Subspace{Origin: from, Orientation: maths.LookAt(from, maths.Vec{})}
}

// twentyEight is the twenty-eighth milestone: a magnetar flares. the template
// with its giant flare moved to two milliseconds before the shot, so the
// fireball at zero has a radius of five of the star's: sixty kilometres to its
// twelve, ten radii across.
//
// it grows at 30,000 kilometres a second, a tenth of the speed of light.
//
// dim blue in visible light, and in gamma the fireball bright and nearly clear.
// the test lays it out over milliseconds.
func twentyEight() (*thing.World, maths.Subspace) {
	w := dark()
	m := sure(template.Magnetar())
	fireball, worn := m.Flare()
	if !worn {
		panic("the magnetar template wears no flare")
	}
	m = with(m, sure(bodies.NewFlare(-2*time.Millisecond, fireball.Speed(),
		fireball.Opacity(), fireball.Glow())).
		Radiating(fireball.Radiant()))
	placed(m.Place(w, maths.Vec{}, 1/m.Radius(), 4))
	from := maths.V(0, 0, 40)
	return w, maths.Subspace{Origin: from, Orientation: maths.LookAt(from, maths.Vec{})}
}

// the eagle's lines and the hubble palette: sulphur ii red, hydrogen
// alpha green, oxygen iii blue
var (
	twentyNineLines = []maths.Band{{Nanometres: 672}, {Nanometres: 656},
		{Nanometres: 501}}
	twentyNinePalette = []maths.Light{maths.RGB(1, 0.12, 0.08),
		maths.RGB(0.1, 1, 0.12), maths.RGB(0.15, 0.35, 1)}
	// a shoulder from a third, so the lit rims keep their colour, and
	// a bloom so the young stars glow
	twentyNineTone = colour.Tone{Knee: 0.35, Top: 6, Bloom: 1.2}
)

// twentyNine is the twenty-ninth milestone: the pillars of creation,
// by procedure. three columns of gas on a wall, placed, sized and
// eroded by a seed, dressed with filaments, and with young stars at
// their tips.
//
// they are lit from above, to the right and a little behind by the
// young cluster out of frame, so the tips and the edges glow and the
// faces are dark.
//
// in visible light scattered starlight, and in the eagle's three
// lines, painted in the hubble palette, the front where the star's
// light reaches the gas.
//
// a nebula, the hard one: the pillars of creation, generated.
func twentyNine() (*thing.World, maths.Subspace) {
	w := thing.NewWorld()
	w.Sky = sky.Stars{Seed: 29, Share: 0.1, Size: 0.35, Dark: maths.Grey(0.002)}
	// the cluster is above, to the right, and a little behind the
	// pillars, so their tops and edges are lit and their faces dark
	w.Lamp(thing.Sun(maths.V(0.35, 1, -0.3), maths.RGB(1.3, 1.4, 1.6)))
	// dressed: filaments up the columns, a toggle on the part
	pillars := medium.Pillars{Count: 3, Height: 7, Width: 10, Thick: 1.6,
		Seed: 29, Filaments: 0.6}.Prepared()
	at := maths.Subspace{Origin: maths.V(0, -3.5, 0)}
	w.AddVolume(thing.Volume{
		Medium: pillars,
		Material: &thing.Material{Name: "the pillars",
			Colour: maths.RGB(0.55, 0.45, 0.38),
			Radiant: thing.Excited{Lines: map[float64]float64{
				672: 0.5, 656: 0.6, 501: 0.4},
				Excitations: map[float64]float64{672: 0.4, 656: 1, 501: 2},
				Width:       3, Passes: 0.5}.Prepared()},
		Motion: thing.Still(at),
	})
	// the young stars, a few at each tip where the gas is being eaten
	// away, bright in every band
	for i, tip := range pillars.Tips() {
		for j := range 3 {
			k := int64(i*10 + j)
			off := maths.V(hash.Hash(290, k)-0.5, hash.Hash(291, k)*0.6,
				hash.Hash(292, k)-0.5).Scale(0.7)
			w.Add(thing.Thing{
				Shape: shape.Sphere{Radius: 0.07},
				Material: &thing.Material{Name: "a young star",
					Glow:    maths.RGB(1.6, 1.7, 2),
					Radiant: thing.Spectrum{Gives: map[radiation.Range]float64{thing.Visible(): 1}}},
				Motion: thing.Still(maths.Subspace{Origin: at.Out(tip.Add(off))}),
				// a ball of light casts no shadow
				Attrs: thing.Attrs{Clear: true},
			})
		}
	}
	from := maths.V(0, 0, 14)
	return w, maths.Subspace{Origin: from, Orientation: maths.LookAt(from, maths.Vec{})}
}

// thirtyTone is a night exposure: two hundred times the day's, with a
// shoulder, and a little bloom so the cities glow.
var thirtyTone = colour.Tone{Exposure: 200, Knee: 0.6, Top: 20, Bloom: 1}

// thirty is the thirtieth milestone: the earth by night. the template
// as it is with the sun behind it, so the eye sees the night side lit
// only by its cities, a few thousandths of the sun's light, and the
// shot exposes two hundred times for them.
//
// a city is a part of the body, placed by hash, and a night is an
// exposure, not a paint.
//
// the test also shoots the day, the sun from the right, where the same
// cities are nothing and the clouds shadow the ground.
func thirty() (*thing.World, maths.Subspace) {
	return thirtyLit(maths.V(0, 0.1, -1), true)
}

// thirtyLit is the earth under a sun from a direction; at night the
// sky's floor is black and its stars faint, as a long exposure leaves.
func thirtyLit(sun maths.Vec, night bool) (*thing.World, maths.Subspace) {
	w := dark()
	if night {
		w.Sky = sky.Stars{Seed: 15, Share: 0.06, Size: 0.35, Bright: 0.004}
	}
	w.Lamp(thing.Sun(sun, swatches.Light(swatches.Daylight).Scale(1.2)))
	e := sure(template.Earth()).WithoutMoons()
	placed(e.Place(w, maths.Vec{}, 1/e.Radius(), 10))
	from := maths.V(0, 0, 4)
	return w, maths.Subspace{Origin: from, Orientation: maths.LookAt(from, maths.Vec{})}
}

// the bullet cluster's bands: x-ray for the hot gas, painted pink as
// chandra's is, and radio standing in for the mass, painted blue as the
// lensing map is. the mass is drawn until fields can lens.
var (
	thirtyOneBands   = []maths.Band{twentyXRay, {Nanometres: 1e9}}
	thirtyOnePalette = []maths.Light{maths.RGB(1, 0.35, 0.6),
		maths.RGB(0.35, 0.5, 1)}
	// a bloom, so the galaxies glow instead of sitting as squares
	thirtyOneTone = colour.Tone{Knee: 0.7, Top: 4, Bloom: 1.2}
)

// thirtyOne is the thirty-first milestone: the bullet cluster, 1e
// 0657-56, in its palette. two clusters of galaxies that have passed
// through each other, the smaller to the right and moving right.
//
// the hot gas of each, which met and slowed, lags behind its galaxies
// toward the middle, the bullet's a compact knot; the mass, which
// passed through with the galaxies, sits with them.
//
// the pillars, the eagle nebula and 1e 0657-56, the bullet cluster, are
// dull imaged as the eye sees them, and are shown in palettes that are
// beautiful because what is sought in the cosmos is awe.
//
// in visible light, galaxies against stars; in x-ray, the gas; in a
// band of its own, the mass. the mass is drawn as a thin medium that
// gives off a band nothing else does, since slayer's tracer cannot yet
// lens; that is said here so no one takes it for physics.
func thirtyOne() (*thing.World, maths.Subspace) {
	w := dark()
	galaxies := func(at maths.Vec, reach float64, n int, seed int64) {
		for i := range n {
			k := seed + int64(i)
			off := maths.V(hash.Hash(310, k)-0.5, hash.Hash(311, k)-0.5,
				hash.Hash(312, k)-0.5).Scale(2 * reach)
			size := 0.05 + 0.08*hash.Hash(313, k)
			w.Add(thing.Thing{
				Shape: shape.Sphere{Radius: size},
				Material: &thing.Material{Name: "a galaxy",
					Glow: maths.RGB(0.9, 0.85, 0.75).Scale(0.6 + hash.Hash(314, k)),
					// stars and space: the bands pass
					Radiant: thing.Spectrum{Passes: map[radiation.Range]float64{
						radiation.XRay(): 1, radiation.Radio(): 1}}},
				Motion: thing.Still(maths.Subspace{Origin: at.Add(off)}),
			})
		}
	}
	galaxies(maths.V(-3, 0, 0), 2.4, 40, 0)
	galaxies(maths.V(3.5, 0, 0), 1.1, 14, 100)
	gas := func(at maths.Vec, radius, thick, xray float64, name string) {
		w.AddVolume(thing.Volume{
			Medium: medium.Wispy{Of: medium.Puff{Radius: radius, Thick: thick},
				Seed: 31, Scale: radius / 3, Amount: 0.8},
			Material: &thing.Material{Name: name, Colour: maths.Grey(0.3),
				Radiant: thing.Spectrum{
					Gives:  map[radiation.Range]float64{radiation.XRay(): xray},
					Passes: map[radiation.Range]float64{radiation.XRay(): 0.9, radiation.Radio(): 1}}},
			Motion: thing.Still(maths.Subspace{Origin: at}),
		})
	}
	gas(maths.V(-1.8, 0, 0), 3, 0.05, 8, "the cluster's gas")
	// the bullet: a head and a tail, so it is a bullet and not a ball
	gas(maths.V(2.1, 0, 0), 0.8, 0.14, 12, "the bullet's head")
	gas(maths.V(1.3, 0, 0), 0.55, 0.1, 10, "the bullet's tail")
	mass := func(at maths.Vec, radius float64, name string) {
		w.AddVolume(thing.Volume{
			Medium: medium.Puff{Radius: radius, Thick: 0.01},
			Material: &thing.Material{Name: name, Colour: maths.Grey(0.3),
				Radiant: thing.Spectrum{
					Gives:  map[radiation.Range]float64{radiation.Radio(): 40},
					Passes: map[radiation.Range]float64{radiation.Radio(): 1, radiation.XRay(): 1}}},
			Motion: thing.Still(maths.Subspace{Origin: at}),
		})
	}
	mass(maths.V(-3, 0, 0), 2.8, "the cluster's mass")
	mass(maths.V(3.5, 0, 0), 1.4, "the bullet's mass")
	from := maths.V(0, 0, 16)
	return w, maths.Subspace{Origin: from, Orientation: maths.LookAt(from, maths.Vec{})}
}

// thirtyTwo is the thirty-second milestone: the aurora from orbit. the
// earth by night from four hundred kilometres up, the eye turned along
// the limb toward the world's up, so the auroral oval lies ahead as
// green curtains with a red top, the cities below; exposed as a night.
//
// aurorae, since planets seen from space, as from the iss, show them.
//
// the aurora is a part of the body: a shell on an oval round each
// magnetic pole, which the earth's template carries at sixty-seven
// degrees magnetic; the curtains are a dressing on it.
func thirtyTwo() (*thing.World, maths.Subspace) {
	w := dark()
	w.Sky = sky.Stars{Seed: 15, Share: 0.06, Size: 0.35, Bright: 0.004}
	e := sure(template.Earth()).WithoutMoons()
	placed(e.Place(w, maths.Vec{}, 1/e.Radius(), 10))
	// standing four hundred kilometres over the direction 62 degrees
	// from the world's z axis toward its y axis, which is 28 from y.
	// the 62 is that angle and not a latitude.
	//
	// the earth leans 23.44 degrees about x and its magnetic axis 11
	// more, so from there the eye is at 85.4 degrees north
	// geographically and 83.6 magnetically, 4.6 and 6.4 from the poles.
	//
	// it looks along the world's up, which from there is away from both
	// poles, with the sun on the far side: night below.
	//
	// the oval's heart, at 67 magnetic, is 16.6 degrees of arc ahead,
	// 1,846 km, and its 200 km level is 14.2 degrees below the eye's
	// level, between it and the limb, 19.8 below. measured, the green
	// in a 240 by 100 frame lies 13.5 to 18.3 below level.
	lat := 62 * maths.Degree
	under := maths.V(0, math.Sin(lat), math.Cos(lat))
	w.Lamp(thing.Sun(under.Scale(-1).Add(maths.V(0.3, 0, 0)),
		swatches.Light(swatches.Daylight).Scale(1.2)))
	from := under.Scale(1 + 400/e.Radius())
	north := maths.Up.Add(under.Scale(-under.Dot(maths.Up))).Unit()
	look := maths.LookAtUp(from, from.Add(north).Add(under.Scale(-0.3)), under)
	return w, maths.Subspace{Origin: from, Orientation: look}
}

// thirtyThreeTone has a shoulder for an aurora brighter than day, and
// a bloom for the magnetar, a point too bright to hold.
var thirtyThreeTone = colour.Tone{Knee: 0.5, Top: 12, Bloom: 1.5}

// thirtyThree is the thirty-third milestone: a world beside a magnetar.
// a planet near a magnetar would have aurorae comically huge, and this
// draws what that could be.
//
// the earth's template with its aurora pushed as a magnetar's wind
// would push it: the oval down to twenty degrees from the equator and
// thirty wide, the shell out to four radii, the glow brighter than
// day, the curtains wild, and x-ray bright.
//
// the magnetar is a point off to the right, blooming.
//
// nothing new is built for it: every number is a knob the earth's
// aurora already had, turned up.
func thirtyThree() (*thing.World, maths.Subspace) {
	w := dark()
	w.Lamp(thing.Sun(maths.V(1, 0.3, 0.6), swatches.Light(swatches.Daylight)))
	e := sure(template.Earth()).WithoutMoons().WithoutCities()
	storm := sure(bodies.NewAurora(
		bodies.NewShell(bodies.Kilometres(300), bodies.Kilometres(25000)),
		measure.Degrees(20), measure.Degrees(30), measure.Degrees(30),
		maths.Share(0.8), maths.Share(1), 33,
		bodies.NewAuroraColours(maths.RGB(1, 0.3, 0.9),
			maths.RGB(0.3, 1, 0.4), maths.RGB(1, 0.35, 0.15)))).
		Radiating(thing.Spectrum{
			Gives:  map[radiation.Range]float64{radiation.XRay(): 1},
			Passes: map[radiation.Range]float64{radiation.XRay(): 0.9}})
	e = with(e, storm)
	placed(e.Place(w, maths.Vec{}, 1/e.Radius(), 8))
	m := sure(template.Magnetar()).WithoutFlare()
	// a thousand times the template's glow, a number chosen for the
	// shot (no distance or inverse square is worked out for it).
	//
	// a radius of two hundred kilometres where the template's is
	// twelve: 278 times the area, since 200 over 12 squared is 278,
	// so a ray lands on it at all. the bloom makes the point
	m = m.Resized(bodies.Kilometres(200)).
		Glowing(m.Glow().Scale(1000), m.Radiant())
	placed(m.Place(w, maths.V(5, 1.5, -4), 1/e.Radius(), 2))
	from := maths.V(0, 0, 14)
	return w, maths.Subspace{Origin: from, Orientation: maths.LookAt(from, maths.Vec{})}
}

// cometTone has a shoulder for a sun or a coma in the frame and a
// bloom for the head.
var cometTone = colour.Tone{Knee: 0.5, Top: 10, Bloom: 1.5}

// thirtyFour is the thirty-fourth milestone: a hot jupiter, its air
// blown off: a planet very close to its sun has a tail like a comet's,
// its air ablated, and fast.
//
// the hot jupiter template forty radii from a sun, its tail streaming
// away from the star and bending back along the orbit, the sun's disc
// at the left of the frame, in planet radii.
func thirtyFour() (*thing.World, maths.Subspace) {
	w := dark()
	// a sun ten jupiter radii across, for the shot
	star := sure(template.Sol()).Resized(bodies.Kilometres(10 * 69911))
	p := sure(template.HotJupiter())
	scale := 1 / p.Radius()
	sun := maths.V(-40, 0, 0)
	w.Lamp(thing.Sun(sun, swatches.Light(swatches.Daylight).Scale(1.3)))
	placed(star.Place(w, sun, scale, 2))
	// the planet goes round the star clockwise seen from above, from
	// plus y: at plus x of it, it moves toward plus z, and r cross v,
	// from the star to the planet and the way it moves, is along minus y
	p = p.Aimed(sun, maths.V(0, 0, 1))
	placed(p.Place(w, maths.Vec{}, scale, 6))
	from := maths.V(8, 4, 25)
	return w, maths.Subspace{Origin: from, Orientation: maths.LookAt(from, maths.V(-3, 0, 0))}
}

// thirtyFive is the thirty-fifth milestone: hale-bopp. an actual comet,
// and one of the two best loved, with hyakutake.
//
// the comet near perihelion, the sun out of frame to the lower left,
// the coma bright and a little green, the dust tail broad and curved
// back along the orbit, the ion tail straight and blue; a million
// kilometres to the unit.
func thirtyFive() (*thing.World, maths.Subspace) {
	w := dark()
	// a long exposure's stars: soft, several pixels across and dim,
	// as a wide comet photograph has them; a star a pixel across has
	// an edge the terminal's rays and smokejudge's sample apart by more
	// than an eye in the dark
	w.Sky = sky.Stars{Seed: 15, Share: 0.1, Size: 1.2, Bright: 0.3,
		Dark: maths.Grey(0.002)}
	c := sure(template.HaleBopp())
	sun := maths.V(-130, -40, 0)
	w.Lamp(thing.Sun(sun, swatches.Light(swatches.Daylight).Scale(1.2)))
	c = c.Aimed(sun, maths.V(0, 0, 1))
	placed(c.Place(w, maths.Vec{}, 1e-6, 2))
	from := maths.V(10, 12, 50)
	return w, maths.Subspace{Origin: from, Orientation: maths.LookAt(from, maths.V(14, 5, 0))}
}

// thirtyFiveHyakutake is hyakutake for the test's second frame: the
// head at the left and the longest ion tail ever seen, five hundred
// million kilometres, streaming out of the frame, two million
// kilometres to the unit.
func thirtyFiveHyakutake() (*thing.World, maths.Subspace) {
	w := dark()
	c := sure(template.Hyakutake())
	sun := maths.V(-65, -20, 0)
	w.Lamp(thing.Sun(sun, swatches.Light(swatches.Daylight).Scale(1.2)))
	c = c.Aimed(sun, maths.V(0, 0, 1))
	placed(c.Place(w, maths.Vec{}, 5e-7, 2))
	from := maths.V(20, 8, 50)
	return w, maths.Subspace{Origin: from, Orientation: maths.LookAt(from, maths.V(18, 5, 0))}
}

// thirtySix is the thirty-sixth milestone: the photon ring. the quasar
// template's hole and disc, no jets, the hole's mass one unit so the
// horizon is two and the shadow five and a bit, seen from fifty units
// ten degrees off edge-on, for the photon rings.
//
// the disc's far side shows over the top of the shadow and under it,
// bent round the hole, and a thin ring of light hugs the shadow's
// edge, where rays have gone round. the bending is celestial's hole's,
// called inside the field's reach; the disc's light arrives shifted.
func thirtySix() (*thing.World, maths.Subspace) {
	return thirtySixBent(true)
}

// thirtySixBent is the photon ring's scene with the hole's field, or
// without it for the test's comparison.
func thirtySixBent(bend bool) (*thing.World, maths.Subspace) {
	return thirtySixSpun(bend, 0)
}

// thirtySixSpun is the photon ring's scene with the hole spun, its spin
// a share of its mass: the shadow leans and flattens on the side coming
// toward the eye, and the horizon wears its hexes, a dressing.
func thirtySixSpun(bend bool, spin float64) (*thing.World, maths.Subspace) {
	w := dark()
	q := sure(template.Quasar()).WithoutJets().
		Tilting(measure.Degrees(10)).Spinning(spin)
	if spin > 0 {
		q = with(q, sure(bodies.NewHorizon(0.3, maths.Light{})))
	}
	// a tenth of the quasar's light, so the disc's structure shows
	q = with(q, brightened(q, 6))
	placed(q.Place(w, maths.Vec{}, 2/q.Radius(), 4))
	if !bend {
		w.Fields = nil
	}
	from := maths.V(0, 0, 50)
	return w, maths.Subspace{Origin: from, Orientation: maths.LookAt(from, maths.Vec{})}
}

// thirtySeven is the thirty-seventh milestone: falling in. the hole and
// the disc of thirty-six, the camera let go from rest at twenty masses,
// thirty degrees above the disc's plane, looking at the hole, and
// falling in.
//
// andrew hamilton's pictures were the look aimed at; no test compares
// against them.
//
// the smoke is the first frame; the sheet in its test is the fall,
// frame by frame as the camera passes each radius, through the
// horizon, which the chart lets, to a mass from the middle.
//
// one row looks at the hole, whose black grows until it fills the
// view, and one looks back, where the light that fell in with the
// camera still arrives.
//
// the picture is taken from the falling camera's own frame, so the sky
// crowds forward as it speeds up and the disc's light shifts as its
// speed and the hole say.
func thirtySeven() (*thing.World, maths.Subspace) {
	return thirtySevenAt(20)
}

// thirtySevenAt is the fall's frame as the camera passes a radius, in
// masses: twenty is at rest, two the horizon, under that inside.
func thirtySevenAt(radius float64) (*thing.World, maths.Subspace) {
	return thirtySevenFrame(radius, false, false)
}

// thirtySevenFrame is thirtySevenAt, looking back from the hole if asked, or
// from a camera held still at the same place, for the test's comparison, which
// no camera can be inside the horizon.
//
// a radius that is not greater than zero, NaN included, or is infinite, is
// refused by name. so is one the chart cannot carry the camera down to: the
// hole's Advance stops short of the singularity, and a step that moves the
// camera nowhere would be taken forever.
func thirtySevenFrame(radius float64, back, still bool) (*thing.World, maths.Subspace) {
	if !(radius > 0) || math.IsInf(radius, 1) {
		placed(fmt.Errorf("smoke: thirtySevenFrame wants a radius above "+
			"zero and finite, in masses, and was given %v", radius))
	}
	w := dark()
	q := sure(template.Quasar()).WithoutJets().Turning(0, measure.Radians(0))
	q = with(q, brightened(q, 6))
	placed(q.Place(w, maths.Vec{}, 2/q.Radius(), 4))
	f := w.Fields[0]
	hole, err := kerr.NewHole(f.Mass, f.Spin)
	if err != nil {
		placed(fmt.Errorf("smoke: thirtySevenFrame: %w", err))
	}
	start := kerr.NewVec3(20*f.Mass*math.Cos(30*maths.Degree), 0,
		20*f.Mass*math.Sin(30*maths.Degree))
	cam := hole.Dropped(start)
	for hole.Radius(cam.Space()) > radius*f.Mass {
		// Advance's second result is true for a camera inside the
		// horizon as well as for one that was stopped, so it cannot
		// say a step failed; a step that moves the camera nowhere does
		next, _ := hole.Advance(cam, 0.05*f.Mass)
		if next == cam {
			placed(fmt.Errorf("smoke: thirtySevenFrame: the camera cannot "+
				"fall to %v masses: it stops at %.4g", radius,
				hole.Radius(cam.Space())/f.Mass))
		}
		cam = next
	}
	x := cam.Space()
	at := f.FromHole([3]float64{x.AlongX(), x.AlongY(), x.AlongZ()})
	if !still {
		v := hole.Velocity(cam)
		f.EyeTime = v.Time()
		f.Eye = f.FromHole([3]float64{v.AlongX(), v.AlongY(), v.AlongZ()})
	}
	toward := maths.Vec{}
	if back {
		toward = at.Scale(2)
	}
	return w, maths.Subspace{Origin: at, Orientation: maths.LookAt(at, toward)}
}

// thirtyEight is the thirty-eighth milestone: the blue marble. the earth
// wearing nasa's blue marble as its map when SLAYER_MARBLE names a folder
// holding blue.jpg, and its own painted world when SLAYER_MARBLE is unset,
// since the image is nasa's and stays out of the repository.
//
// SLAYER_MARBLE set to a folder with no blue.jpg that decodes as a jpeg
// is refused, by name, and not drawn as the painted world.
//
// africa faces the eye, the sun from the upper right. it is the one
// smoke that reads from disk, and its Inputs say so, for the smoke cache:
// smokes declare their input files.
func thirtyEight() (*thing.World, maths.Subspace) {
	w := dark()
	w.Lamp(thing.Sun(maths.V(0.6, 0.3, 1), maths.RGB(1.2, 1.2, 1.2)))
	e := sure(template.Earth())
	e = e.WithoutMoons().WithoutClouds()
	if blue := marble("blue.jpg"); blue != nil {
		// the picture's left edge is the antimeridian; turned so
		// africa faces the eye on plus z
		ground, worn := e.Ground()
		if !worn {
			placed(fmt.Errorf("smoke: the earth template wears no ground"))
		}
		e = with(e, ground.Mapped(blue, measure.Radians(-1.6)))
	}
	placed(e.Place(w, maths.Vec{}, 1/e.Radius(), 10))
	from := maths.V(0, 0, 3.2)
	return w, maths.Subspace{Origin: from, Orientation: maths.LookAt(from, maths.Vec{})}
}

// marble is one of nasa's marbles from the folder SLAYER_MARBLE names, or
// nothing when SLAYER_MARBLE is unset. When it is set, a file that is not
// there, cannot be read or is not a jpeg that decodes (a png is not) is
// refused by name, with its path and the reason.
func marble(name string) image.Image {
	dir := os.Getenv("SLAYER_MARBLE")
	if dir == "" {
		return nil
	}
	path := filepath.Join(dir, name)
	f, err := os.Open(path)
	if err != nil {
		placed(fmt.Errorf("smoke: SLAYER_MARBLE is set, and %s "+
			"cannot be opened: %w", path, err))
	}
	defer f.Close()
	img, err := jpeg.Decode(f)
	if err != nil {
		placed(fmt.Errorf("smoke: SLAYER_MARBLE is set, and %s "+
			"is not a jpeg that decodes: %w", path, err))
	}
	return img
}

// thirtyNine is the thirty-ninth milestone: a windscreen catches the
// sky. a sheet of dark glass leaning back sixty degrees from upright,
// the eye close behind it as a driver's is, looking a little up along
// it, a day sky over: darker low, and brighter toward the top.
//
// for a camera 60 degrees across, in a frame 240 by 100, the rays that
// meet the glass first meet it from 52.4 degrees off its normal, at the
// bottom of the middle column, to 87.9 at the top of the frame.
//
// the glass mirrors by schlick's fresnel: at n = 1.52, 0.043 face on,
// 0.051 at 52.4 degrees, 0.82 at 87.6 (the middle column's top row)
// and 0.84 at 87.9. so low on the glass a twentieth of the sky is
// mirrored and it is dark; at the top, four fifths, and it is bright.
func thirtyNine() (*thing.World, maths.Subspace) {
	return thirtyNineGloss(1)
}

// thirtyNineGloss is the windscreen's scene with the glass's gloss as asked:
// zero is the same sheet, dull, for the test's comparison.
func thirtyNineGloss(gloss float64) (*thing.World, maths.Subspace) {
	w := thing.NewWorld()
	w.Add(thing.Thing{
		Shape: shape.Plane{},
		Material: &thing.Material{Name: "chalk",
			Colour: swatches.Light(swatches.Chalk)},
	})
	// the glass passes half a unit up the axis and leans back: its
	// up, the normal, is turned thirty degrees from the world's toward
	// the eye, so the sheet meets the floor in front of the eye and
	// climbs away from it
	w.Add(thing.Thing{
		Shape: shape.Plane{},
		Material: &thing.Material{Name: "glass", Colour: maths.Grey(0.02),
			Gloss: gloss, Index: 1.52},
		Motion: thing.Still(maths.Subspace{Origin: maths.V(0, 0.5, 0),
			Orientation: maths.Rotation(maths.XAxis, 30*maths.Degree)}),
	})
	w.Sky = sky.Gradient{Horizon: maths.RGB(0.75, 0.8, 0.9),
		Zenith: maths.RGB(0.25, 0.4, 0.8), Ground: maths.Grey(0.2)}
	w.Lamp(matteSun(maths.V(0.5, 2, 2),
		swatches.Light(swatches.Daylight).Scale(1.1)))
	from := maths.V(0, 1, 1.5)
	return w, maths.Subspace{Origin: from,
		Orientation: maths.LookAt(from, maths.V(0, 1.6, -1.9))}
}

// rejets is a template's jets rebuilt with the shot's own length and
// speed, the rest as the template made them: a look the shot chooses
// and the body does not know about.
func rejets(jets bodies.Jets, lengthKm, speed float64) bodies.Jets {
	return sure(bodies.NewJets(bodies.Kilometres(lengthKm),
		bodies.Kilometres(jets.Base()), jets.Spread(),
		bodies.Kilometres(jets.From()),
		bodies.PerKilometre(jets.Dense()),
		measure.Radians(jets.Tilt()), maths.Share(speed),
		jets.Glow())).Knotted(jets.Knots()).Radiating(jets.Radiant())
}

// brightened is the body's disc rebuilt at another brightness, the
// rest as the template made it: a look the shot chooses.
func brightened(body bodies.Body, bright float64) bodies.Disc {
	disc, worn := body.Disc()
	if !worn {
		placed(fmt.Errorf("smoke: %s wears no disc to brighten",
			body.Name()))
	}
	fresh := sure(bodies.NewDisc(bodies.Kilometres(disc.Inner()),
		bodies.Kilometres(disc.Outer()), bodies.Kilometres(disc.Thick()),
		bodies.PerKilometre(disc.Dense()), measure.Kelvin(disc.Hot()),
		bright))
	fresh = sure(fresh.Beamed(maths.Share(disc.Speed())))
	return fresh.Dressed(disc.Streaks(), disc.Shear())
}
