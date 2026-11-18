package hillside

// A set that names an air is lit by it: the sky, the sun and the
// exposure are the air's, worked out by package airlight from what the
// air is made of, with no painted gradient and no fill lamp. The design
// is doc/sky-from-the-air.md.

import (
	"fmt"
	"os"

	"github.com/janearc/slayer/airlight"
	"github.com/janearc/slayer/airs"
	"github.com/janearc/slayer/emit"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/set"
)

// stagedDistantSource is the exact emitter inserted by aired or moonlit,
// ocean-path validator without identifying it by its floating-point values.
type stagedDistantSource struct {
	index int
	lamp  emit.Distant
}

// aired is the set's light from its air: the computed sky, the direct
// sun as one distant lamp, and the exposure, the job's or the film's
// where either says one and the meter's where neither does. A set with
// no sun, a sun not up, or a sun it tints is refused, and so is -sky
// black.
func (stage *staging) aired(read set.Set) error {
	if err := refusedBlack(stage.given); err != nil {
		return err
	}
	way, err := stage.airSun(read)
	if err != nil {
		return err
	}
	fields, plan, note, err := stage.airFields(read.Air())
	if err != nil {
		return err
	}
	exposure, metered, err := exposureFor(stage.given,
		func() (float64, error) { return airlight.Meter(fields, way) })
	if err != nil {
		return fmt.Errorf("%s: %w", stage.given.set, err)
	}
	stage.given.exposure = exposure
	stage.airedSky = airlight.NewSky(fields, way, exposure)
	stage.world.Sky = stage.airedSky
	lamp, err := emit.NewDistant(way, airlight.Sun(fields, way, exposure))
	if err != nil {
		return err
	}
	// Record this exact insertion. Groundless-air stages have no synthetic
	// fill, and later scene validation must never guess which Distant lamp is
	// the air model's sun by matching its values.
	lampIndex := len(stage.world.Lamps)
	stage.world.Lamp(lamp)
	stage.airedSun = &stagedDistantSource{index: lampIndex, lamp: lamp}
	stage.made.lamps++
	stage.made.sky = skySaid(plan, note, exposure, metered)
	stage.given.cached = append(stage.given.cached,
		cacheNote{what: stage.made.sky, told: true})
	return nil
}

// airSun is the way to the sun of a set under an air: from its azimuth
// and altitude, as daylight's is. A set that says no sun, a sun not
// above the horizon, and a tint are refused.
func (stage *staging) airSun(read set.Set) (maths.Vec, error) {
	sun, has := read.Sunlit()
	if !has {
		return maths.Vec{}, fmt.Errorf("%s: an air is lit by the sun "+
			"where the set puts it, and the set says no sun: give it "+
			"sun: with an azimuth and an altitude", stage.given.set)
	}
	if err := refusedTint(sun, stage.given.set); err != nil {
		return maths.Vec{}, err
	}
	azimuth, altitude := sun.Where()
	if !(altitude > nothing) {
		return maths.Vec{}, fmt.Errorf("%s: daylight sun must be above "+
			"the horizon", stage.given.set)
	}
	return skyward(azimuth, altitude), nil
}

// airFields is the fields of the named air seen from the ground's lowest
// height, lit by the sun's spectrum on the shelf, from the cache where
// they were worked out before, with the plan that decides them and the
// note of where they came from.
func (stage *staging) airFields(name string) (*airlight.Fields,
	airlight.Plan, cacheNote, error) {
	if stage.airFieldsProvider != nil {
		return stage.airFieldsProvider(name)
	}
	sun, err := stage.sunSpectrum()
	if err != nil {
		return nil, airlight.Plan{}, cacheNote{}, fmt.Errorf("%s: %w",
			stage.given.set, err)
	}
	known, err := airlight.Named(name, sun)
	if err != nil {
		return nil, airlight.Plan{}, cacheNote{}, fmt.Errorf("%s: %w",
			stage.given.set, err)
	}
	ground := stage.lowest
	if stage.groundlessAir {
		ground = stage.airAltitude
	}
	plan, err := airlight.NewPlan(known, ground)
	if err != nil {
		return nil, plan, cacheNote{}, fmt.Errorf("%s: %w", stage.given.set,
			err)
	}
	fields, note, err := keptFields(plan)
	stage.given.cached = append(stage.given.cached, note)
	return fields, plan, note, err
}

// sunSpectrum is the sun's spectrum above any air, read from the shelf
// through the pin of sources.yaml nearest the set, as a planet's
// methane table is. It downloads nothing: a spectrum not on the shelf
// is refused, saying how to fetch it.
func (stage *staging) sunSpectrum() (airs.Sun, error) {
	path, err := stage.shelved(airs.SunSource)
	if err != nil {
		return airs.Sun{}, err
	}
	file, err := os.Open(path)
	if err != nil {
		return airs.Sun{}, err
	}
	defer file.Close()
	return airs.ReadSun(file)
}

// refusedBlack is the refusal of -sky black under an air, or nil for the
// sky left as it is. An air's sky is its air's, worked out from what the
// air is made of, and not a flag's to put out.
func refusedBlack(given asked) error {
	if given.sky != "black" {
		return nil
	}
	return fmt.Errorf("%s: -sky black under an air: an air's sky is its "+
		"air's, not a flag's, and is not put out; leave out -sky black, "+
		"or the air", given.set)
}

// refusedTint is the refusal of a sun the set tints under an air, or nil
// for one it does not. The air gives the sun its colour; a tint would
// paint over it.
func refusedTint(sun set.Sun, file string) error {
	if !sun.Tinted() {
		return nil
	}
	return fmt.Errorf("%s: the sun under an air takes its colour from the "+
		"air, and sun: tint: would paint over it; leave the tint out", file)
}

// exposureFor is the exposure a set under an air is drawn at, and
// whether the meter gave it. The job's or the film's stands where either
// said one: by staging it is a number above nothing, since filmed puts
// the film's in where the job said none, and a rig's is one.
//
// Where neither said one, it is the meter's.
func exposureFor(given asked, meter func() (float64, error)) (float64,
	bool, error) {
	if given.exposure > nothing {
		return given.exposure, false, nil
	}
	metered, err := meter()
	return metered, true, err
}

// skySaid is what a run says of a sky worked from an air: the air and
// its plan, whether it was worked out or from the cache and in what
// time, and the exposure, metered or said.
func skySaid(plan airlight.Plan, note cacheNote, exposure float64,
	metered bool) string {
	made := fmt.Sprintf("the sky is computed from %s, worked out in %.1fs",
		plan.Said(), note.spent.Seconds())
	if note.hit {
		made = fmt.Sprintf("the sky is computed from %s, from the cache, "+
			"read in %.3fs", plan.Said(), note.spent.Seconds())
	}
	if metered {
		return made + fmt.Sprintf("; exposure %.4g, metered: a level grey "+
			"card under the sun and the sky at its own 0.18", exposure)
	}
	return made + fmt.Sprintf("; exposure %.4g, as the job or the film "+
		"said", exposure)
}
