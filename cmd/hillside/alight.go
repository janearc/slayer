package main

import (
	"math"

	"github.com/janearc/celestial/atmosphere"
	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/emit"
	"github.com/janearc/slayer/flame"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/plume"
	"github.com/janearc/slayer/scene/thing"
)

// the smoke over a tree alight. the numbers, and where each is from:
const (
	// treeHeat is the fire's heat release rate at its peak, kw: nist
	// tn 2362 measured 26 to 42 mw at the peak of 5.6 to 7.4 m douglas
	// firs. the manufactory's spruce is 10 m; 40 mw is picked at the
	// top of what was measured, not worked out for a taller tree.
	treeHeat = 40000.0
	// treeConvective is the share of the heat the air carries up: one
	// less the radiative fraction nist tn 2314 measured for douglas
	// fir trees, 0.29 on average, trees 0.9 to 1.8 m tall.
	treeConvective = 0.71
	// treeSoot is soot's yield, g for each g burned, as nist tn 2314
	// measured it for the same small douglas firs, by laser
	// extinction; the report says the estimate needs more work.
	treeSoot = 0.0024
	// treeCombustion is what burning gives, mj a kg: nist tn 2362
	// measured 18 to 25.5 for its 3 to 7 m firs, 21.1 on average.
	treeCombustion = 21.0
	// sootExtinction is soot's mass extinction, m2 a g at 632 nm:
	// mulholland and croarkin (2000), as nist tn 2314 cites it. the
	// plume takes it at 550 nm, so it is carried there by extinction.
	sootExtinction = 8.7
	// sootMeasuredAt is the wavelength sootExtinction was measured at,
	// in nm
	sootMeasuredAt = 632.0
)

// sootAt550 is soot's mass extinction at 550 nm, m2 a g: the measured
// one, carried from its wavelength by the same tilt the soot's colour
// takes, hottel and broughton's exponent for soot, flame.Visible.
var sootAt550 = sootExtinction * math.Pow(sootMeasuredAt/550,
	flame.Visible)

const (
	// smokeTop is how high above the tree the column is drawn, m:
	// picked. how a plume stops in the sky's layers is not built.
	smokeTop = 150.0
	// smokeAlbedo is the share of what soot stops that it scatters:
	// picked, as the plume research picked it for flaming smoke, with
	// nothing measured under it.
	smokeAlbedo = 0.3
)

// smoke sets the column of smoke over a tree alight: its soot carried
// up its plume, in the standard atmosphere's air at the tree's height
// above the sea, the floor's and its own above the floor.
func smoke(world *thing.World, at maths.Subspace, radius, floor float64,
	seed uint64) error {
	air, err := atmosphere.Standard(measure.Metres(math.Max(0, floor+
		at.Origin.Y)))
	if err != nil {
		return err
	}
	column, err := plume.New(plume.Fire{Heat: treeHeat,
		Convective: treeConvective, Base: 2 * radius, Yield: treeSoot,
		Combustion: treeCombustion, Extinction: sootAt550,
		Air:     air.Temperature().InKelvin(),
		Density: air.Density().InKilogramsPerCubicMetre(),
		Top:     smokeTop, Seed: seed})
	if err != nil {
		return err
	}
	world.AddVolume(thing.Volume{Medium: column,
		Material: &thing.Material{Name: "smoke",
			Colour: maths.Grey(smokeAlbedo)},
		Motion: thing.Still(maths.Subspace{Origin: at.Origin})})
	return nil
}

// a spruce alight is in a crown fire: a flame's body over its whole
// crown. the numbers, and where each is from:
const (
	// tallerThanTrees is how many times the tree's height its flames
	// reach: 1.3 to 1.8 measured, flames 18 to 25 m on 14 m jack pine in
	// the international crown fire modeling experiment (butler, us
	// forest service, 2010). 1.5 is picked inside that.
	tallerThanTrees = 1.5
	// crownHottest is the flame at its hottest, in kelvin: butler and
	// others (2004), crown fires in the international crown fire
	// modeling experiment, peaks that "exceeded 1330 c", 1603 k, by
	// thermocouple; their radiometric temperatures read higher.
	//
	// from the abstract, the paper not yet read in full. it was 1300
	// k, cited to kremens and dickinson (2015), whose abstract gives no
	// flame temperature.
	crownHottest = 1603.0
	// crownEdge is its edge, in kelvin: picked, inside the 830 to 1440
	// k riggan and others (2004) found most fire light comes from.
	crownEdge = 900.0
	// crownSoot is its soot's absorption, a metre: hägglund and
	// persson measured 0.75 to 1.3 a metre over wood cribs, as an
	// effective grey number (cited by quintiere and mccaffrey, nbs,
	// 1980). 1.0 is picked inside that and taken as at 550 nm.
	crownSoot = 1.0
)

// alight sets a made thing on fire where it stands: a flame's body as
// wide as the thing and tallerThanTrees as tall, its turbulence seeded
// by the place, its glow at the picture's exposure; and its light as
// lamps, the flame at the picture's time. it says how many lamps.
func alight(world *thing.World, one kept, at maths.Subspace, seed uint64,
	given asked) (int, error) {
	radius, height := reach(one.things, at.Scale)
	fire := flame.Fire{Radius: radius, Height: tallerThanTrees * height,
		Absorption: crownSoot, Seed: seed}
	fire.Hottest, fire.Edge = measure.Kelvin(crownHottest),
		measure.Kelvin(crownEdge)
	body, err := flame.NewBody(fire, given.exposure)
	if err != nil {
		return 0, err
	}
	world.AddVolume(thing.Volume{Medium: body,
		Material: &thing.Material{Name: "flame", Shine: body.Shine()},
		Motion:   thing.Still(maths.Subspace{Origin: at.Origin})})
	if err := smoke(world, at, radius, given.floor, seed); err != nil {
		return 0, err
	}
	burning := body.Tick(given.at).(*flame.Body)
	sources := burning.Sources(flameCells)
	for _, source := range sources {
		world.Lamp(emit.Lit{At: at.Origin.Add(source.At),
			Strength: source.Strength, Beyond: source.Across / 2})
	}
	return len(sources), nil
}

// flameCells is how many cells a side a flame's light is cut into to
// light what is near it: picked for time. each cell's light starts half
// a cell from its middle, so no point of it flares on a branch it
// almost touches; the flame between dims it, as any lamp in a medium.
const flameCells = 6

// reach is how far out from its middle and how high a made thing's
// shapes go, at a scale, by their boxes in their own space.
func reach(things []thing.Thing, scale float64) (radius, height float64) {
	if scale == 0 {
		scale = 1
	}
	for _, each := range things {
		box := each.Shape.Bounds()
		out := math.Max(math.Max(-box.Min.X, box.Max.X),
			math.Max(-box.Min.Z, box.Max.Z))
		radius = math.Max(radius, out)
		height = math.Max(height, box.Max.Y)
	}
	return radius * scale, height * scale
}
