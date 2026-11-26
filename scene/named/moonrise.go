package named

import (
	"fmt"
	"io"
	"math"
	"slices"

	"github.com/janearc/slayer/emit"
	"github.com/janearc/slayer/globe"
	"github.com/janearc/slayer/manufacture"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
)

// The moonrise: the full moon rising behind a tree on the earth, through
// a long lens. One unit is a metre. Its first product is a raw, the
// linear light of every pixel; its command develops the preview its own
// way, so the tone here is the plain clamp.

// the scene's numbers: measured ones name their source, picked ones say
// they are.
const (
	// the moon's mean radius and distance, in metres, and its geometric
	// albedo: nasa's moon fact sheet. its surface is drawn as matte,
	// which the moon is not; its brightening at full is not here.
	moonRadius, moonDistance, moonAlbedo = 1737.4e3, 384.4e6, 0.12
	// earthRadius is the earth's mean radius, the iugg's 6371 km
	earthRadius = 6371e3
	// eyeHeight is a person's eyes over the ground, picked
	eyeHeight = 1.7
	// sunlight is the light the sun lands, hillside's day, picked
	sunlight = 1.5
)

// soil is the earth's colour, picked by looking, as hillside's is.
var soil = maths.RGB(0.2, 0.17, 0.1)

// the moonrise as cmd/moonrise drew it: the tree's kind and seed, metres
// to it, the moon's and the sun's altitudes in degrees, and the bounds
// the command held them to; the lens's width in degrees, about a 2000 mm
// lens on 15/70; imax's frame, 1.43 to 1, with its rays a side; and the
// degrees in a half turn, defined
const (
	moonriseKind                  = "norway"
	moonriseSeed                  = 3
	moonriseAway                  = 1100
	moonAltitude, sunAltitude     = 0.3, 2
	moonLowest, highest           = -5, 90
	moonriseWideDegrees           = 2
	moonriseWidth, moonriseHeight = 1430, 1000
	moonriseRays                  = 2
	halfTurn                      = 180
)

// moonriseScene is the moon rising behind a tree.
func moonriseScene() Scene {
	return Scene{name: "moonrise",
		about: "the full moon rising behind a tree on the earth, " +
			"through a long lens",
		unit:       "a metre",
		renderers:  []Renderer{CPU},
		parameters: moonriseParameters(),
		defaults: Defaults{width: moonriseWidth,
			height: moonriseHeight, rays: moonriseRays,
			wide: moonriseWideDegrees * maths.Degree},
		build: buildMoonrise}
}

// moonriseParameters are what the moonrise takes, as cmd/moonrise's
// flags were, with the bounds it held them to.
func moonriseParameters() []Parameter {
	choices := manufacture.KindNames()
	for name := range moonrisePlants {
		choices = append(choices, name)
	}
	slices.Sort(choices)
	choices = slices.Compact(choices)
	return []Parameter{
		oneOf("kind", moonriseKind, choices,
			"the tree's kind"),
		whole("seed", moonriseSeed, atLeast(nothing), "the tree's seed"),
		number("away", moonriseAway, above(nothing),
			"metres to the tree"),
		number("moon", moonAltitude, between(moonLowest, highest),
			"the moon's altitude, in degrees"),
		number("sun", sunAltitude, between(nothing, highest),
			"the sun's altitude, in degrees"),
	}
}

type moonrisePlant struct {
	book    *manufacture.PlantBook
	options manufacture.PlantOptions
}

type moonriseBook struct {
	family string
	book   *manufacture.PlantBook
}

// The command's species choices come from the checked embedded books, not
// the kind registry. The two loose shoots were also offered by the command.
var moonrisePlants = func() map[string]moonrisePlant {
	var books []moonriseBook
	for _, family := range []string{"seasonal", "conifer", "fern", "cactus"} {
		name, raw, err := manufacture.EmbeddedPlantBook(family)
		if err != nil {
			panic(err)
		}
		book, err := manufacture.ReadPlantBook(family, name, raw)
		if err != nil {
			panic(err)
		}
		books = append(books, moonriseBook{family, book})
	}
	plants, err := collectMoonrisePlants(books, manufacture.KindNames())
	if err != nil {
		panic(err)
	}
	return plants
}()

// collectMoonrisePlants refuses ambiguous selectors before the scene offers
// its choices; neither a species nor a shoot can shadow a factory kind.
func collectMoonrisePlants(books []moonriseBook, public []string) (map[string]moonrisePlant, error) {
	plants := make(map[string]moonrisePlant)
	add := func(name string, plant moonrisePlant) error {
		if _, exists := plants[name]; exists || slices.Contains(public, name) {
			return fmt.Errorf("moonrise: ambiguous plant selector %q", name)
		}
		plants[name] = plant
		return nil
	}
	for _, family := range books {
		for _, species := range family.book.Species() {
			if err := add(species, moonrisePlant{family.book, manufacture.PlantOptions{Species: species}}); err != nil {
				return nil, err
			}
		}
		if family.family == "conifer" {
			shoot := "shoot"
			for _, species := range []string{"coast-redwood", "giant-sequoia"} {
				if err := add(species+"-shoot", moonrisePlant{family.book, manufacture.PlantOptions{Species: species, Representation: &shoot}}); err != nil {
					return nil, err
				}
			}
		}
	}
	return plants, nil
}

// buildMoonrise is the scene as asked: the earth, a tree on it at the
// origin, the eye back from it looking at the moon, and the sun and moon
// either side.
func buildMoonrise(held values, _ io.Writer) (*Built, error) {
	world := thing.NewWorld()
	earth, err := globe.New(earthRadius)
	if err != nil {
		return nil, err
	}
	world.Add(thing.Thing{Shape: earth,
		Material: &thing.Material{Name: "earth", Colour: soil},
		Motion: thing.Still(maths.Subspace{
			Origin: maths.V(0, -earthRadius, 0)})})
	if err := planted(world, held.choice("kind"),
		uint64(held.whole("seed"))); err != nil {
		return nil, err
	}
	eye := maths.V(0, eyeHeight, held.number("away"))
	moon, err := risen(world, eye, held.number("moon"))
	if err != nil {
		return nil, err
	}
	if err := lit(world, held.number("sun")); err != nil {
		return nil, err
	}
	return &Built{world: world, eye: maths.Subspace{Origin: eye,
		Orientation: maths.LookAt(eye, moon)}}, nil
}

// planted stands one tree of the kind and seed asked at the origin.
func planted(world *thing.World, kind string, seed uint64) error {
	var row manufacture.Kind
	var err error
	if plant, ok := moonrisePlants[kind]; ok {
		row, err = plant.book.New(plant.options)
	} else {
		row, err = manufacture.KindNamed(kind)
	}
	if err != nil {
		return err
	}
	tree, err := manufacture.Make(row, manufacture.Sow(seed))
	if err != nil {
		return err
	}
	things, err := tree.Things(maths.Subspace{})
	if err != nil {
		return err
	}
	for _, each := range things {
		world.Add(each)
	}
	return nil
}

// risen puts the moon straight ahead of the eye at an altitude, at its
// own distance, and is where its middle is.
func risen(world *thing.World, eye maths.Vec, degrees float64) (maths.Vec,
	error) {
	up := degrees * math.Pi / halfTurn
	centre := eye.Add(maths.V(0, math.Sin(up), -math.Cos(up)).
		Scale(moonDistance))
	moon, err := globe.New(moonRadius)
	if err != nil {
		return maths.Vec{}, err
	}
	world.Add(thing.Thing{Shape: moon,
		Material: &thing.Material{Name: "moon",
			Colour: maths.Grey(moonAlbedo)},
		Motion: thing.Still(maths.Subspace{Origin: centre})})
	return centre, nil
}

// lit puts the sun behind the eye at an altitude, at an endless
// distance.
func lit(world *thing.World, degrees float64) error {
	up := degrees * math.Pi / halfTurn
	sun, err := emit.NewDistant(maths.V(0, math.Sin(up), math.Cos(up)),
		maths.Grey(sunlight))
	if err != nil {
		return err
	}
	world.Lamp(sun)
	return nil
}
