package set

import (
	"time"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/generate"
	"github.com/janearc/slayer/scene/texture"
	yaml "gopkg.in/yaml.v3"
)

// planetSaid is a planet's numbers as a set says them, in the units a
// person says them in.
type planetSaid struct {
	mass, day, tilt, distance float64
	flux                      float64
	sunMass, luminosity       float64
	age, seed                 float64
	makeup                    generate.Makeup
	locked                    bool
	clouds                    *generate.CloudLook
	cloudsMadeUp              bool
	air                       *generate.AirLook
	airMadeUp                 bool
	giant                     *texture.Banded
	giantMadeUp               bool
	giantGas                  *generate.GiantGasLook
	giantGasMadeUp            bool
	ground                    *generate.GroundLook
	groundMadeUp              bool
}

// The most a set may say of a generated planet.
const (
	// heaviestPlanet is in Earth masses: 0.08 Suns, where Chen and
	// Kipping 2017 find the planets end and the stars begin
	heaviestPlanet = 26600
	// longestDay is in hours, past any planet's; Venus turns once in 5832
	longestDay = 1e6
	// heaviestSun is in the Sun's masses, past the heaviest stars known
	heaviestSun = 300
	// oldest is in years, about the age of the universe
	oldest = 1.4e10
)

// generated is a step that reads a generate block, a planet's numbers,
// into target, the planet named for its body: its mass in the Earth's,
// its day in hours, its tilt in degrees, its distance in astronomical
// units or in place of it the light it gets in the Earth's (flux), its
// sun's mass and light in the Sun's, and its age in years.
// Its day is left out where it is locked, which makes its day its year,
// and its sun where it goes round a lava drop, which is its sun; the body
// that reads it checks which (see wholly).
func (reading reader) generated(target **generate.Planet,
	name string) step {
	return func(_, value *yaml.Node) error {
		var said planetSaid
		err := reading.keys(value, reading.generateKeys(&said), nil)
		if err == nil && says(value, "flux") && says(value, "distance") {
			err = reading.refuse(value, "a generated planet says the "+
				"light it gets or its distance, not both")
		}
		if err == nil {
			err = reading.all(value, "a generated planet", "mass",
				"makeup", "tilt", reading.wherePut(value), "age", "seed")
		}
		if err == nil {
			err = reading.all(valueNode(value, "makeup"), "a makeup",
				"metal", "rock", "ice", "gas")
		}
		if err == nil && says(value, "sun") {
			err = reading.all(valueNode(value, "sun"), "a planet's sun",
				"mass", "luminosity")
		}
		if err == nil && says(value, "clouds") {
			err = reading.all(valueNode(value, "clouds"), "clouds",
				"base", "top", "cover", "loss", "made_up")
			if err == nil && !said.cloudsMadeUp {
				err = reading.refuse(valueNode(value, "clouds"),
					"an authored cloud shell says made_up: true")
			}
		}
		if err == nil && says(value, "air_look") {
			block := valueNode(value, "air_look")
			err = reading.all(block, "air look", "height_km", "loss_per_km", "colour", "made_up")
			if err == nil && !said.airMadeUp {
				err = reading.refuse(block, "an authored air look says made_up: true")
			}
		}
		if err == nil && says(value, "giant_look") {
			block := valueNode(value, "giant_look")
			err = reading.all(block, "a giant look", "paints", "roil", "shear", "grain", "fine", "mottle", "jets", "made_up")
			if err == nil && !said.giantMadeUp {
				err = reading.refuse(block, "an authored giant look says made_up: true")
			}
		}
		if err == nil && says(value, "giant_gas") {
			block := valueNode(value, "giant_gas")
			err = reading.all(block, "giant gas", "depth", "opacity", "made_up")
			if err == nil && !said.giantGasMadeUp {
				err = reading.refuse(block, "authored giant gas says made_up: true")
			}
		}
		if err == nil && says(value, "ground_look") {
			block := valueNode(value, "ground_look")
			err = reading.all(block, "a ground look", "ocean", "variation", "ridge_contrast", "made_up")
			if err == nil && !said.groundMadeUp {
				err = reading.refuse(block, "an authored ground look says made_up: true")
			}
		}
		*target = said.planet(name)
		return err
	}
}

// wherePut is the one key a generated planet says where it stands by:
// its flux, the light it gets, if it says that, and else its distance.
func (reading reader) wherePut(block *yaml.Node) string {
	if says(block, "flux") {
		return "flux"
	}
	return "distance"
}

// wholly checks what a generated body says beyond what its generate block
// does: a body that goes round a lava drop is locked, has no sun of its
// own and no middle, since the drop gives it all three; any other says
// its sun, and its day or that it is locked, and its middle.
func (reading reader) wholly(block *yaml.Node) error {
	made := valueNode(block, "generate")
	if says(block, "circles") {
		switch {
		case valueNode(made, "locked") == nil ||
			valueNode(made, "locked").Value != "true":
			return reading.refuse(made, "a body that goes round a lava "+
				"drop is locked, and says locked: true")
		case says(made, "sun") || says(block, "middle"):
			return reading.refuse(block, "a body that goes round a lava "+
				"drop has the drop for its sun and its place, and says "+
				"neither a sun nor a middle")
		}
		return reading.all(block, "a body that goes round a lava drop",
			"generate", "circles", "detail")
	}
	if !says(made, "sun") || !says(made, "day") && !says(made, "locked") {
		return reading.refuse(made, "a generated planet says its sun, "+
			"and its day or that it is locked")
	}
	return reading.all(block, "a generated body", "generate", "middle",
		"detail")
}

// generateKeys is the steps that read a generate block into said.
func (reading reader) generateKeys(said *planetSaid) map[string]step {
	return map[string]step{
		"mass": reading.number(&said.mass, 0, heaviestPlanet),
		"makeup": reading.block(map[string]step{
			"metal": reading.number(&said.makeup.Metal, 0, 1),
			"rock":  reading.number(&said.makeup.Rock, 0, 1),
			"ice":   reading.number(&said.makeup.Ice, 0, 1),
			"gas":   reading.number(&said.makeup.Gas, 0, 1)}),
		"day":      reading.number(&said.day, -longestDay, longestDay),
		"tilt":     reading.number(&said.tilt, -180, 180),
		"distance": reading.number(&said.distance, 0, farthest),
		"flux":     reading.number(&said.flux, 0, brightest),
		"sun": reading.block(map[string]step{
			"mass":       reading.number(&said.sunMass, 0, heaviestSun),
			"luminosity": reading.number(&said.luminosity, 0, brightest)}),
		"age":    reading.number(&said.age, 0, oldest),
		"seed":   reading.whole(&said.seed, 0, mostSeed),
		"locked": reading.yes(&said.locked),
		"air_look": func(key, value *yaml.Node) error {
			said.air = &generate.AirLook{}
			var colour [3]float64
			err := reading.block(map[string]step{
				"height_km":   reading.number(&said.air.Height, 0, 100000),
				"loss_per_km": reading.number(&said.air.Loss, 0, 100000),
				"colour":      reading.three(&colour, 0, 1),
				"made_up":     reading.yes(&said.airMadeUp),
			})(key, value)
			said.air.Colour = maths.RGB(colour[0], colour[1], colour[2])
			if err == nil && !(said.air.Height > 0) {
				err = reading.refuse(value, "air scale height must be above zero")
			}
			return err
		},
		"giant_look": reading.giantLook(said),
		"giant_gas": func(key, value *yaml.Node) error {
			said.giantGas = &generate.GiantGasLook{}
			err := reading.block(map[string]step{
				"depth":   reading.number(&said.giantGas.Depth, 0, 1),
				"opacity": reading.number(&said.giantGas.Opacity, 0, 100000),
				"made_up": reading.yes(&said.giantGasMadeUp),
			})(key, value)
			if err == nil && !(said.giantGas.Depth > 0) {
				err = reading.refuse(value, "giant gas depth must be above zero")
			}
			return err
		},
		"ground_look": func(key, value *yaml.Node) error {
			said.ground = &generate.GroundLook{}
			return reading.block(map[string]step{
				"ocean":          reading.number(&said.ground.Ocean, 0, 1),
				"variation":      reading.number(&said.ground.Variation, 0, 1),
				"ridge_contrast": reading.number(&said.ground.RidgeContrast, 0, 1),
				"made_up":        reading.yes(&said.groundMadeUp),
			})(key, value)
		},
		"clouds": func(key, value *yaml.Node) error {
			said.clouds = &generate.CloudLook{}
			return reading.block(map[string]step{
				"base":    reading.number(&said.clouds.Base, 0, 100000),
				"top":     reading.number(&said.clouds.Top, 0, 100000),
				"cover":   reading.number(&said.clouds.Cover, 0, 1),
				"loss":    reading.number(&said.clouds.Loss, 0, 100000),
				"weather": reading.word(&said.clouds.Weather, "bands", "soft-bands"),
				"storm_turn": func(key, value *yaml.Node) error {
					said.clouds.StormTurn = new(float64)
					return reading.number(said.clouds.StormTurn, 0, 10)(key, value)
				},
				"made_up": reading.yes(&said.cloudsMadeUp),
			})(key, value)
		},
	}
}

// planet is the Planet a set's numbers say, in the units it is made in.
func (said planetSaid) planet(name string) *generate.Planet {
	return &generate.Planet{
		Name:   name,
		Mass:   measure.Kilograms(said.mass * generate.EarthMass),
		Makeup: said.makeup,
		Day:    time.Duration(said.day * float64(time.Hour)),
		Tilt:   measure.Degrees(said.tilt),
		Sun: generate.Sun{Mass: measure.SolarMasses(said.sunMass),
			Luminosity: said.luminosity},
		Orbit: generate.Orbit{Distance: measure.Metres(said.distance *
			generate.AstronomicalUnit), Flux: said.flux},
		Age:        generate.Years(said.age),
		Seed:       uint64(said.seed),
		Locked:     said.locked,
		Clouds:     said.clouds,
		AirLook:    said.air,
		GiantLook:  said.giant,
		GiantGas:   said.giantGas,
		GroundLook: said.ground,
	}
}

// giantLook reads artistic band and storm controls in the texture's units.
func (reading reader) giantLook(said *planetSaid) step {
	return func(_, value *yaml.Node) error {
		look := &texture.Banded{}
		said.giant = look
		var jets float64
		err := reading.keys(value, map[string]step{
			"paints": func(name, list *yaml.Node) error {
				if list.Kind != yaml.SequenceNode || len(list.Content) == 0 || len(list.Content) > 32 {
					return reading.refuse(list, "paints is a list of one to 32 linear RGB triples")
				}
				for _, item := range list.Content {
					var rgb [3]float64
					if err := reading.three(&rgb, 0, 1)(name, item); err != nil {
						return err
					}
					look.Paints = append(look.Paints, maths.RGB(rgb[0], rgb[1], rgb[2]))
				}
				return nil
			},
			"roil":    reading.number(&look.Roil, 0, 1),
			"shear":   reading.number(&look.Shear, 1, 100),
			"grain":   reading.number(&look.Grain, 0.001, 1),
			"fine":    reading.number(&look.Fine, 0, 1),
			"mottle":  reading.number(&look.Mottle, 0, 1),
			"jets":    reading.whole(&jets, 0, 64),
			"spots":   reading.giantSpots(look),
			"streets": reading.giantStreets(look),
			"made_up": reading.yes(&said.giantMadeUp),
		}, nil)
		look.Jets = int(jets)
		return err
	}
}

func (reading reader) giantStreets(look *texture.Banded) step {
	return func(_, list *yaml.Node) error {
		if list.Kind != yaml.SequenceNode || len(list.Content) > 16 {
			return reading.refuse(list, "streets is a list of at most 16 storm rows")
		}
		for _, item := range list.Content {
			var street texture.Street
			var count, seed float64
			var rgb [3]float64
			err := reading.keys(item, map[string]step{
				"lat":   reading.number(&street.Lat, -0.9, 0.9),
				"count": reading.whole(&count, 1, 32),
				"size":  reading.number(&street.Size, 0.001, 0.1),
				"turn":  reading.number(&street.Turn, -10, 10),
				"seed":  reading.whole(&seed, 0, mostSeed),
				"paint": reading.three(&rgb, 0, 1),
			}, nil)
			if err == nil {
				err = reading.all(item, "a giant storm street", "lat", "count", "size", "turn", "seed", "paint")
			}
			if err != nil {
				return err
			}
			street.Count, street.Seed = int(count), uint64(seed)
			street.Paint = maths.RGB(rgb[0], rgb[1], rgb[2])
			look.Streets = append(look.Streets, street)
		}
		return nil
	}
}

func (reading reader) giantSpots(look *texture.Banded) step {
	return func(_, list *yaml.Node) error {
		if list.Kind != yaml.SequenceNode || len(list.Content) > 64 {
			return reading.refuse(list, "spots is a list of at most 64 storms")
		}
		for _, item := range list.Content {
			var spot texture.Spot
			var seed float64
			var rgb [3]float64
			err := reading.keys(item, map[string]step{
				"lat":     reading.number(&spot.Lat, -0.99, 0.99),
				"lon":     reading.number(&spot.Lon, -3.141592653589793, 3.141592653589793),
				"size":    reading.number(&spot.Size, 0.001, 1),
				"stretch": reading.number(&spot.Stretch, 0.1, 10),
				"turn":    reading.number(&spot.Turn, -10, 10),
				"detail":  reading.number(&spot.Detail, 0, 1),
				"seed":    reading.whole(&seed, 0, mostSeed),
				"paint":   reading.three(&rgb, 0, 1),
			}, nil)
			if err == nil {
				err = reading.all(item, "a giant storm", "lat", "lon", "size", "stretch", "turn", "paint")
			}
			if err != nil {
				return err
			}
			spot.Paint = maths.RGB(rgb[0], rgb[1], rgb[2])
			spot.Seed = uint64(seed)
			look.Spots = append(look.Spots, spot)
		}
		return nil
	}
}

// madeOnce refuses a body that is not made exactly one way: from a
// template, or generated, each with its middle and detail.
func (reading reader) madeOnce(block *yaml.Node) error {
	if !says(block, "generate") {
		if says(block, "circles") {
			return reading.refuse(block, "a body that goes round a lava "+
				"drop is generated")
		}
		return reading.all(block, "a body", "template", "middle", "detail")
	}
	if says(block, "template") {
		return reading.refuse(block, "a body is from a template or "+
			"generated, not both")
	}
	return reading.wholly(block)
}
